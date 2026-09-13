package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RuntimeAsset 是运行态对外公开的对象库元数据。Path 等内部位置永远不会写入响应。
type RuntimeAsset struct {
	ID          string    `json:"assetId"`
	Name        string    `json:"name"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	UpdatedAt   time.Time `json:"updatedAt"`
	ETag        string    `json:"etag"`
	Usage       string    `json:"usage,omitempty"`
	Entry       string    `json:"entry,omitempty"`
}

type AssetStore interface {
	List(context.Context) ([]RuntimeAsset, error)
	Get(context.Context, string) (RuntimeAsset, error)
	Open(context.Context, string) (RuntimeAsset, AssetReadSeekCloser, error)
}

type AssetReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

var ErrAssetNotFound = errors.New("运行资源不存在")

type fileAssetRecord struct {
	RuntimeAsset
	Path string `json:"path"`
}

// FileAssetStore 从受控目录的 manifest.json 加载资源。manifest 中 path 只在服务端使用。
type FileAssetStore struct {
	root    string
	entries map[string]fileAssetRecord
}

func NewFileAssetStore(root string) (*FileAssetStore, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" {
		return nil, errors.New("对象库运行态目录不能为空")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("对象库运行态目录无效")
	}
	payload, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("读取对象库运行态索引失败: %w", err)
	}
	var raw struct {
		SchemaVersion string            `json:"schemaVersion"`
		Assets        []fileAssetRecord `json:"assets"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("解析对象库运行态索引失败: %w", err)
	}
	if raw.SchemaVersion != "runtime-assets.v1" {
		return nil, errors.New("对象库运行态索引版本无效")
	}
	entries := make(map[string]fileAssetRecord, len(raw.Assets))
	for _, item := range raw.Assets {
		item.ID = strings.TrimSpace(item.ID)
		item.Path = filepath.Clean(item.Path)
		if item.ID == "" || item.Path == "." || filepath.IsAbs(item.Path) {
			return nil, errors.New("对象库运行态索引包含非法资源路径")
		}
		resolved := filepath.Join(root, item.Path)
		if relative, err := filepath.Rel(root, resolved); err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, errors.New("对象库运行态索引越过资源根目录")
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("对象库运行态资源文件不存在: %s", item.ID)
		}
		if item.Size < 0 || (item.Size > 0 && item.Size != info.Size()) {
			return nil, fmt.Errorf("对象库运行态资源大小不匹配: %s", item.ID)
		}
		if item.Size == 0 {
			item.Size = info.Size()
		}
		if item.UpdatedAt.IsZero() {
			item.UpdatedAt = info.ModTime().UTC()
		}
		if item.ETag == "" {
			item.ETag = `"` + item.ID + `"`
		}
		if _, exists := entries[item.ID]; exists {
			return nil, fmt.Errorf("对象库运行态索引包含重复资源 ID: %s", item.ID)
		}
		entries[item.ID] = item
	}
	return &FileAssetStore{root: root, entries: entries}, nil
}

func (s *FileAssetStore) List(context.Context) ([]RuntimeAsset, error) {
	if s == nil {
		return nil, errors.New("对象库运行态存储未配置")
	}
	items := make([]RuntimeAsset, 0, len(s.entries))
	for _, item := range s.entries {
		items = append(items, item.RuntimeAsset)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (s *FileAssetStore) Get(ctx context.Context, id string) (RuntimeAsset, error) {
	item, _, err := s.open(ctx, id)
	return item, err
}

func (s *FileAssetStore) Open(ctx context.Context, id string) (RuntimeAsset, AssetReadSeekCloser, error) {
	return s.open(ctx, id)
}

func (s *FileAssetStore) open(ctx context.Context, id string) (RuntimeAsset, AssetReadSeekCloser, error) {
	if err := ctx.Err(); err != nil {
		return RuntimeAsset{}, nil, err
	}
	item, ok := s.entries[strings.TrimSpace(id)]
	if !ok {
		return RuntimeAsset{}, nil, ErrAssetNotFound
	}
	file, err := os.Open(filepath.Join(s.root, item.Path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RuntimeAsset{}, nil, ErrAssetNotFound
		}
		return RuntimeAsset{}, nil, err
	}
	if item.Size <= 0 {
		if info, statErr := file.Stat(); statErr == nil {
			item.Size = info.Size()
			item.UpdatedAt = info.ModTime().UTC()
			if item.ETag == `"`+item.ID+`"` {
				sum := sha256.New()
				if _, copyErr := io.Copy(sum, file); copyErr == nil {
					item.ETag = `"` + hex.EncodeToString(sum.Sum(nil)) + `"`
				}
				_, _ = file.Seek(0, io.SeekStart)
			}
		}
	}
	return item.RuntimeAsset, file, nil
}

func (s *Server) listAssets(writer http.ResponseWriter, request *http.Request) {
	if s.config.Assets == nil {
		writeError(writer, request, http.StatusServiceUnavailable, 50320, "运行态对象库未配置")
		return
	}
	items, err := s.config.Assets.List(request.Context())
	if err != nil {
		writeError(writer, request, http.StatusServiceUnavailable, 50321, "读取运行态对象库失败")
		return
	}
	keyword := strings.ToLower(strings.TrimSpace(request.URL.Query().Get("query")))
	filtered := items[:0]
	for _, item := range items {
		if keyword == "" || strings.Contains(strings.ToLower(item.ID), keyword) || strings.Contains(strings.ToLower(item.Name), keyword) {
			filtered = append(filtered, item)
		}
	}
	page, pageSize := parsePage(request.URL.Query().Get("page"), request.URL.Query().Get("pageSize"))
	start := (page - 1) * pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	writeOK(writer, request, map[string]any{"items": filtered[start:end], "total": len(filtered), "page": page, "pageSize": pageSize})
}

func (s *Server) getAsset(writer http.ResponseWriter, request *http.Request) {
	if s.config.Assets == nil {
		writeError(writer, request, http.StatusServiceUnavailable, 50320, "运行态对象库未配置")
		return
	}
	item, err := s.config.Assets.Get(request.Context(), request.PathValue("assetId"))
	if err != nil {
		status := http.StatusServiceUnavailable
		code := 50321
		if errors.Is(err, ErrAssetNotFound) {
			status, code = http.StatusNotFound, 40420
		}
		writeError(writer, request, status, code, "运行资源不存在或暂不可用")
		return
	}
	writeOK(writer, request, item)
}

func (s *Server) openAsset(writer http.ResponseWriter, request *http.Request) {
	if s.config.Assets == nil {
		writeError(writer, request, http.StatusServiceUnavailable, 50320, "运行态对象库未配置")
		return
	}
	item, reader, err := s.config.Assets.Open(request.Context(), request.PathValue("assetId"))
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, ErrAssetNotFound) {
			status = http.StatusNotFound
		}
		writeError(writer, request, status, 40420, "运行资源不存在或暂不可用")
		return
	}
	defer reader.Close()
	if match := request.Header.Get("If-None-Match"); match != "" && match == item.ETag {
		writer.WriteHeader(http.StatusNotModified)
		return
	}
	writer.Header().Set("ETag", item.ETag)
	writer.Header().Set("Content-Type", item.ContentType)
	writer.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(writer, request, item.Name, item.UpdatedAt, reader)
}

func parsePage(rawPage, rawSize string) (int, int) {
	page, pageSize := 1, 50
	if value, err := strconv.Atoi(rawPage); err == nil && value > 0 {
		page = value
	}
	if value, err := strconv.Atoi(rawSize); err == nil && value > 0 {
		if value > 200 {
			value = 200
		}
		pageSize = value
	}
	return page, pageSize
}
