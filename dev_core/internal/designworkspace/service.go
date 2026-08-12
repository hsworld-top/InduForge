package designworkspace

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/project"
)

var (
	ErrInvalidAction   = errors.New("设计文件操作参数无效")
	ErrInvalidPath     = errors.New("设计文件路径无效")
	ErrProtectedPath   = errors.New("系统预设目录不允许修改")
	ErrFileTooLarge    = errors.New("设计文件超过大小限制")
	ErrUnsupported     = errors.New("暂不支持该设计文件操作")
	ErrSymbolicLink    = errors.New("设计文件路径包含符号链接")
	ErrDestinationBusy = errors.New("目标路径已存在")
)

const maxDesignFileSize int64 = 32 << 20

var protectedRootDirectories = map[string]struct{}{
	"assets": {}, "components": {}, "displays": {}, "materials": {},
	"models": {}, "scenes": {}, "symbols": {},
}

type ProjectRepository interface {
	Get(ctx context.Context, tenantID, projectID string) (project.Project, error)
}

type Action struct {
	Command string          `json:"command"`
	Data    json.RawMessage `json:"data"`
}

type Service struct {
	projects ProjectRepository
}

func NewService(projects ProjectRepository) *Service {
	return &Service{projects: projects}
}

type ContentFile struct {
	File *os.File
	Info fs.FileInfo
}

// Execute 将 HT 原有命令协议映射到工程工作空间文件操作；所有命令都要求工程写权限。
func (s *Service) Execute(ctx context.Context, actor auth.User, projectID string, action Action) (any, error) {
	workspacePath, err := s.writableWorkspace(ctx, actor, projectID)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(strings.TrimSpace(action.Command)) {
	case "explore":
		return s.explore(workspacePath, action.Data)
	case "upload":
		return s.upload(workspacePath, action.Data)
	case "import":
		return s.importArchive(workspacePath, action.Data)
	case "source":
		return s.source(workspacePath, action.Data)
	case "remove":
		return s.remove(workspacePath, action.Data)
	case "rename":
		return s.rename(workspacePath, action.Data)
	case "mkdir":
		return s.mkdir(workspacePath, action.Data)
	case "locate":
		return s.locate(workspacePath, action.Data)
	case "paste":
		return s.paste(workspacePath, action.Data)
	case "export":
		return nil, ErrUnsupported
	default:
		return nil, ErrInvalidAction
	}
}

// OpenContent 为 HT 内部的 XHR、图片和模型加载提供受控文件流，不暴露工程真实目录。
func (s *Service) OpenContent(ctx context.Context, actor auth.User, projectID, requested string) (ContentFile, error) {
	workspacePath, err := s.writableWorkspace(ctx, actor, projectID)
	if err != nil {
		return ContentFile{}, err
	}
	target, err := resolvePath(workspacePath, requested, false)
	if err != nil {
		return ContentFile{}, err
	}
	file, err := os.Open(target)
	if err != nil {
		return ContentFile{}, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return ContentFile{}, err
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return ContentFile{}, ErrInvalidPath
	}
	return ContentFile{File: file, Info: info}, nil
}

func (s *Service) writableWorkspace(ctx context.Context, actor auth.User, projectID string) (string, error) {
	item, err := s.projects.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return "", err
	}
	if err := project.RequireCapability(actor, item, auth.CapabilityProjectWrite); err != nil {
		return "", err
	}
	return validateWorkspaceRoot(item.WorkspacePath)
}

func (s *Service) explore(workspacePath string, raw json.RawMessage) (map[string]any, error) {
	requested, err := decodePath(raw)
	if err != nil {
		return nil, err
	}
	target, err := resolvePath(workspacePath, requested, true)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, nil
	} else if err != nil {
		return nil, err
	}
	return exploreDirectory(target)
}

func (s *Service) upload(workspacePath string, raw json.RawMessage) (any, error) {
	var input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &input); err != nil || strings.TrimSpace(input.Path) == "" {
		return false, ErrInvalidAction
	}
	target, err := resolvePath(workspacePath, input.Path, true)
	if err != nil {
		return false, err
	}
	content, err := decodeUploadContent(input.Content)
	if err != nil {
		return false, err
	}
	if strings.EqualFold(filepath.Ext(input.Path), ".zip") {
		return s.uploadArchive(workspacePath, content)
	}
	if int64(len(content)) > maxDesignFileSize {
		return false, ErrFileTooLarge
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) source(workspacePath string, raw json.RawMessage) (string, error) {
	input := struct {
		URL      string `json:"url"`
		Encoding string `json:"encoding"`
		Prefix   string `json:"prefix"`
	}{}
	if err := json.Unmarshal(raw, &input); err != nil {
		if err := json.Unmarshal(raw, &input.URL); err != nil {
			return "", ErrInvalidAction
		}
	}
	target, err := resolvePath(workspacePath, input.URL, false)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	info, err := os.Stat(target)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > maxDesignFileSize {
		return "", ErrFileTooLarge
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(input.Encoding, "base64") || isImagePath(target) {
		return input.Prefix + base64.StdEncoding.EncodeToString(content), nil
	}
	return input.Prefix + string(content), nil
}

func (s *Service) remove(workspacePath string, raw json.RawMessage) (bool, error) {
	requested, err := decodePath(raw)
	if err != nil {
		return false, err
	}
	if isProtectedRoot(requested) {
		return false, ErrProtectedPath
	}
	target, err := resolvePath(workspacePath, requested, true)
	if err != nil {
		return false, err
	}
	if samePath(target, workspacePath) {
		return false, ErrProtectedPath
	}
	if err := os.RemoveAll(target); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) rename(workspacePath string, raw json.RawMessage) (bool, error) {
	var input struct {
		Old string `json:"old"`
		New string `json:"new"`
	}
	if err := json.Unmarshal(raw, &input); err != nil || input.Old == "" || input.New == "" {
		return false, ErrInvalidAction
	}
	if isProtectedRoot(input.Old) || isProtectedRoot(input.New) {
		return false, ErrProtectedPath
	}
	oldPath, err := resolvePath(workspacePath, input.Old, false)
	if err != nil {
		return false, err
	}
	newPath, err := resolvePath(workspacePath, input.New, true)
	if err != nil {
		return false, err
	}
	if samePath(oldPath, workspacePath) || samePath(newPath, workspacePath) {
		return false, ErrProtectedPath
	}
	if _, err := os.Stat(newPath); err == nil {
		return false, ErrDestinationBusy
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return false, err
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) mkdir(workspacePath string, raw json.RawMessage) (bool, error) {
	requested, err := decodePath(raw)
	if err != nil {
		return false, err
	}
	target, err := resolvePath(workspacePath, requested, true)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) locate(workspacePath string, raw json.RawMessage) (bool, error) {
	requested, err := decodePath(raw)
	if err != nil {
		return false, err
	}
	target, err := resolvePath(workspacePath, requested, true)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(target)
	return err == nil, nil
}

func (s *Service) paste(workspacePath string, raw json.RawMessage) (bool, error) {
	var input struct {
		Destination string   `json:"destDir"`
		Files       []string `json:"fileList"`
	}
	if err := json.Unmarshal(raw, &input); err != nil || len(input.Files) == 0 {
		return false, ErrInvalidAction
	}
	destination, err := resolvePath(workspacePath, input.Destination, true)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return false, err
	}
	for _, requested := range input.Files {
		if isProtectedRoot(requested) {
			return false, ErrProtectedPath
		}
		source, err := resolvePath(workspacePath, requested, false)
		if err != nil {
			return false, err
		}
		if samePath(source, workspacePath) {
			return false, ErrProtectedPath
		}
		target, err := nextCopyPath(destination, filepath.Base(source))
		if err != nil {
			return false, err
		}
		if err := copyTree(source, target); err != nil {
			return false, err
		}
	}
	return true, nil
}

func validateWorkspaceRoot(path string) (string, error) {
	resolved, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil || strings.TrimSpace(path) == "" {
		return "", ErrInvalidPath
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", ErrInvalidPath
	}
	return resolved, nil
}

// resolvePath 同时校验路径边界和现有父级符号链接，防止通过 ../ 或链接逃逸到其他工程。
func resolvePath(root, requested string, allowMissing bool) (string, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(strings.TrimLeft(requested, "/\\")), "\\", "/")
	cleaned := filepath.Clean(filepath.FromSlash(normalized))
	if cleaned == "." {
		cleaned = ""
	}
	if filepath.IsAbs(cleaned) || filepath.VolumeName(cleaned) != "" || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", ErrInvalidPath
	}
	// .induforge/context 是平台生成的只读镜像；场景契约只能通过专用 API 写入 .induforge/scenes。
	if isPlatformManagedPath(cleaned) {
		return "", ErrProtectedPath
	}
	target, err := filepath.Abs(filepath.Join(root, cleaned))
	if err != nil || !isWithin(root, target) {
		return "", ErrInvalidPath
	}
	current := root
	if cleaned != "" {
		for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
			current = filepath.Join(current, part)
			info, statErr := os.Lstat(current)
			if errors.Is(statErr, os.ErrNotExist) && allowMissing {
				break
			}
			if statErr != nil {
				return "", statErr
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return "", ErrSymbolicLink
			}
		}
	}
	return target, nil
}

func decodePath(raw json.RawMessage) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
		return "", ErrInvalidAction
	}
	return value, nil
}

func decodeUploadContent(content string) ([]byte, error) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(strings.ToLower(trimmed), "data:") {
		return []byte(content), nil
	}
	separator := strings.IndexByte(trimmed, ',')
	if separator < 0 {
		return nil, ErrInvalidAction
	}
	header, payload := trimmed[:separator], trimmed[separator+1:]
	if !strings.Contains(strings.ToLower(header), ";base64") {
		return []byte(payload), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, ErrInvalidAction
	}
	return decoded, nil
}

func exploreDirectory(path string) (map[string]any, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any, len(entries))
	for _, entry := range entries {
		if hiddenName(entry.Name()) {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, ErrSymbolicLink
		}
		if entry.IsDir() {
			children, err := exploreDirectory(filepath.Join(path, entry.Name()))
			if err != nil {
				return nil, err
			}
			result[entry.Name()] = children
		} else {
			result[entry.Name()] = true
		}
	}
	return result, nil
}

func nextCopyPath(directory, name string) (string, error) {
	extension := filepath.Ext(name)
	base := strings.TrimSuffix(name, extension)
	for index := 0; ; index++ {
		candidateName := name
		if index > 0 {
			candidateName = fmt.Sprintf("%s-%d%s", base, index, extension)
		}
		candidate := filepath.Join(directory, candidateName)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
}

func copyTree(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return ErrSymbolicLink
	}
	if info.Mode().IsRegular() {
		if info.Size() > maxDesignFileSize {
			return ErrFileTooLarge
		}
		content, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, content, 0o644)
	}
	if !info.IsDir() {
		return fs.ErrInvalid
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if hiddenName(entry.Name()) {
			continue
		}
		if err := copyTree(filepath.Join(source, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func isProtectedRoot(path string) bool {
	cleaned := filepath.ToSlash(filepath.Clean(strings.TrimSpace(strings.TrimLeft(path, "/\\"))))
	_, protected := protectedRootDirectories[cleaned]
	return protected
}

func isPlatformManagedPath(path string) bool {
	cleaned := filepath.ToSlash(filepath.Clean(path))
	return cleaned == ".induforge" || strings.HasPrefix(cleaned, ".induforge/")
}

func hiddenName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, ".") || strings.HasSuffix(lower, "__") || strings.HasSuffix(lower, ".swap") || strings.HasSuffix(lower, ".out") || strings.HasSuffix(lower, ".svn") || strings.HasPrefix(lower, "~$") || strings.HasSuffix(lower, "~")
}

func isImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp":
		return true
	default:
		return false
	}
}

func isWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func samePath(left, right string) bool {
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}
