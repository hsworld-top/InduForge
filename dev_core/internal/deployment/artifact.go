package deployment

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/objectstore"
)

type Workspace interface {
	Export(path string) (map[string]string, error)
}

type ArtifactStore interface {
	Put(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) (objectstore.ObjectRef, error)
	PresignGet(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
}

type Artifact struct {
	Content      []byte
	SourceHash   string
	ArtifactHash string
	Size         int64
}

// BuildArtifact 将源码快照和清单写入确定顺序的 ZIP，输出可由 Node Agent 直接解压的 .ifp 内容。
func BuildArtifact(files map[string]string, manifest map[string]any) (Artifact, error) {
	names := make([]string, 0, len(files))
	for name := range files {
		clean := filepath.ToSlash(filepath.Clean(name))
		if clean == "." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
			return Artifact{}, fmt.Errorf("工作空间文件路径非法: %s", name)
		}
		names = append(names, clean)
	}
	sort.Strings(names)

	sourceHasher := sha256.New()
	decoded := make(map[string][]byte, len(names))
	for _, name := range names {
		content, err := base64.StdEncoding.DecodeString(files[name])
		if err != nil {
			return Artifact{}, fmt.Errorf("解码工作空间文件失败 %s: %w", name, err)
		}
		decoded[name] = content
		_, _ = sourceHasher.Write([]byte(name))
		_, _ = sourceHasher.Write([]byte{0})
		_, _ = sourceHasher.Write(content)
		_, _ = sourceHasher.Write([]byte{0})
	}

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Artifact{}, fmt.Errorf("生成发布清单失败: %w", err)
	}
	if err := writeZipEntry(writer, "manifest.json", manifestJSON); err != nil {
		return Artifact{}, err
	}
	for _, name := range names {
		if err := writeZipEntry(writer, name, decoded[name]); err != nil {
			return Artifact{}, err
		}
	}
	if err := writer.Close(); err != nil {
		return Artifact{}, fmt.Errorf("完成 IFP 压缩失败: %w", err)
	}
	hash := sha256.Sum256(buffer.Bytes())
	return Artifact{
		Content: buffer.Bytes(), SourceHash: hex.EncodeToString(sourceHasher.Sum(nil)),
		ArtifactHash: hex.EncodeToString(hash[:]), Size: int64(buffer.Len()),
	}, nil
}

func writeZipEntry(writer *zip.Writer, name string, content []byte) error {
	entry, err := writer.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
	if err != nil {
		return fmt.Errorf("创建 IFP 文件项失败: %w", err)
	}
	if _, err := entry.Write(content); err != nil {
		return fmt.Errorf("写入 IFP 文件项失败: %w", err)
	}
	return nil
}
