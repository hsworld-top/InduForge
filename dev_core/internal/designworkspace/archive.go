package designworkspace

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
)

const (
	maxArchiveFiles     = 10000
	maxArchiveFileSize  = 32 << 20
	maxArchiveTotalSize = 256 << 20
	archiveTempDir      = ".induforge/design-imports"
)

var archiveRoots = map[string]struct{}{
	"assets": {}, "components": {}, "displays": {}, "materials": {},
	"models": {}, "scenes": {}, "symbols": {},
}

type ArchiveFile struct {
	Content  []byte
	Filename string
}

type archiveImportResult struct {
	ImportPending bool     `json:"importPending"`
	Path          string   `json:"path"`
	Conflicts     []string `json:"conflicts"`
	Roots         []string `json:"roots"`
}

// ExportArchive 按 HT 资源引用关系生成 ZIP；调用方可直接将 Content 写入流式 HTTP 响应。
func (s *Service) ExportArchive(ctx context.Context, actor auth.User, projectID string, paths []string) (ArchiveFile, error) {
	workspacePath, err := s.writableWorkspace(ctx, actor, projectID)
	if err != nil {
		return ArchiveFile{}, err
	}
	if len(paths) == 0 {
		return ArchiveFile{}, ErrInvalidAction
	}

	files, err := collectArchiveFiles(workspacePath, paths)
	if err != nil {
		return ArchiveFile{}, err
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, relative := range files {
		if err := addFileToArchive(writer, workspacePath, relative); err != nil {
			_ = writer.Close()
			return ArchiveFile{}, err
		}
	}
	if err := writer.Close(); err != nil {
		return ArchiveFile{}, err
	}
	return ArchiveFile{
		Content:  output.Bytes(),
		Filename: "ht-design-" + time.Now().Format("20060102-150405") + ".zip",
	}, nil
}

func (s *Service) uploadArchive(workspacePath string, content []byte) (any, error) {
	if int64(len(content)) > maxArchiveTotalSize {
		return false, ErrFileTooLarge
	}
	token, err := newArchiveToken()
	if err != nil {
		return false, err
	}
	tempRoot := filepath.Join(workspacePath, filepath.FromSlash(archiveTempDir), token)
	filesRoot := filepath.Join(tempRoot, "files")
	if err := os.MkdirAll(filesRoot, 0o700); err != nil {
		return false, err
	}
	keepTemp := false
	defer func() {
		if !keepTemp {
			_ = os.RemoveAll(tempRoot)
		}
	}()

	paths, roots, err := extractArchive(content, filesRoot)
	if err != nil {
		return false, err
	}
	conflicts, err := findArchiveConflicts(workspacePath, paths)
	if err != nil {
		return false, err
	}
	if len(conflicts) == 0 {
		if err := commitArchive(workspacePath, filesRoot, paths, false); err != nil {
			return false, err
		}
		return archiveImportResult{Conflicts: []string{}, Roots: roots}, nil
	}

	keepTemp = true
	return archiveImportResult{
		ImportPending: true,
		Path:          token,
		Conflicts:     conflicts,
		Roots:         roots,
	}, nil
}

func (s *Service) importArchive(workspacePath string, raw json.RawMessage) (any, error) {
	var input struct {
		Path string `json:"path"`
		Move bool   `json:"move"`
	}
	if err := json.Unmarshal(raw, &input); err != nil || !validArchiveToken(input.Path) {
		return false, ErrInvalidAction
	}
	tempRoot := filepath.Join(workspacePath, filepath.FromSlash(archiveTempDir), input.Path)
	filesRoot := filepath.Join(tempRoot, "files")
	paths, err := listArchiveFiles(filesRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, ErrInvalidAction
		}
		return false, err
	}
	roots := archiveRootsForPaths(paths)
	if !input.Move {
		if err := os.RemoveAll(tempRoot); err != nil {
			return false, err
		}
		return archiveImportResult{Conflicts: []string{}, Roots: roots}, nil
	}

	if err := commitArchive(workspacePath, filesRoot, paths, true); err != nil {
		return false, err
	}
	if err := os.RemoveAll(tempRoot); err != nil {
		return false, err
	}
	return archiveImportResult{Conflicts: []string{}, Roots: roots}, nil
}

// extractArchive 只接受七类 HT 资源目录，并在写盘前完成路径、链接和解压规模校验。
func extractArchive(content []byte, destination string) ([]string, []string, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, nil, ErrInvalidAction
	}
	if len(reader.File) == 0 || len(reader.File) > maxArchiveFiles {
		return nil, nil, ErrFileTooLarge
	}

	seen := make(map[string]struct{}, len(reader.File))
	roots := make(map[string]struct{})
	paths := make([]string, 0, len(reader.File))
	var totalSize int64
	for _, item := range reader.File {
		relative, root, err := normalizeArchivePath(item.Name)
		if err != nil {
			return nil, nil, err
		}
		if item.Mode()&os.ModeSymlink != 0 {
			return nil, nil, ErrSymbolicLink
		}
		if item.FileInfo().IsDir() {
			continue
		}
		if !strings.Contains(relative, "/") {
			return nil, nil, ErrInvalidPath
		}
		if !item.Mode().IsRegular() {
			return nil, nil, ErrInvalidAction
		}
		if item.UncompressedSize64 > maxArchiveFileSize {
			return nil, nil, ErrFileTooLarge
		}
		totalSize += int64(item.UncompressedSize64)
		if totalSize > maxArchiveTotalSize {
			return nil, nil, ErrFileTooLarge
		}
		key := strings.ToLower(relative)
		if _, exists := seen[key]; exists {
			return nil, nil, ErrInvalidAction
		}
		seen[key] = struct{}{}
		roots[root] = struct{}{}

		target := filepath.Join(destination, filepath.FromSlash(relative))
		if !isWithin(destination, target) {
			return nil, nil, ErrInvalidPath
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, nil, err
		}
		written, err := extractArchiveFile(item, target)
		if err != nil {
			return nil, nil, err
		}
		if written != int64(item.UncompressedSize64) {
			return nil, nil, ErrInvalidAction
		}
		paths = append(paths, relative)
	}
	if len(paths) == 0 {
		return nil, nil, ErrInvalidAction
	}
	sort.Strings(paths)
	return paths, sortedKeys(roots), nil
}

func extractArchiveFile(item *zip.File, target string) (int64, error) {
	source, err := item.Open()
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	written, copyErr := io.Copy(destination, io.LimitReader(source, maxArchiveFileSize+1))
	closeErr := destination.Close()
	if copyErr != nil {
		return 0, copyErr
	}
	if written > maxArchiveFileSize {
		return 0, ErrFileTooLarge
	}
	return written, closeErr
}

func normalizeArchivePath(name string) (string, string, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
	if normalized == "" || strings.HasPrefix(normalized, "/") || filepath.VolumeName(filepath.FromSlash(normalized)) != "" {
		return "", "", ErrInvalidPath
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(normalized)))
	if cleaned == "." || cleaned == "" || cleaned == ".." || strings.HasPrefix(cleaned, "../") || filepath.IsAbs(filepath.FromSlash(cleaned)) {
		return "", "", ErrInvalidPath
	}
	parts := strings.Split(cleaned, "/")
	root := strings.ToLower(parts[0])
	if _, allowed := archiveRoots[root]; !allowed {
		return "", "", ErrInvalidPath
	}
	parts[0] = root
	return strings.Join(parts, "/"), root, nil
}

func findArchiveConflicts(workspacePath string, paths []string) ([]string, error) {
	conflicts := make([]string, 0)
	for _, relative := range paths {
		target, err := resolvePath(workspacePath, relative, true)
		if err != nil {
			return nil, err
		}
		if _, err := os.Lstat(target); err == nil {
			conflicts = append(conflicts, relative)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	return conflicts, nil
}

// commitArchive 先验证全部源文件和目标路径，再统一写入，避免校验失败时留下部分结果。
func commitArchive(workspacePath, filesRoot string, paths []string, overwrite bool) error {
	type copyItem struct {
		source       string
		target       string
		backup       string
		sourceMoved  bool
		targetBacked bool
	}
	items := make([]copyItem, 0, len(paths))
	backupRoot := filepath.Join(filepath.Dir(filesRoot), "backup")
	for _, relative := range paths {
		source, err := resolvePath(filesRoot, relative, false)
		if err != nil {
			return err
		}
		info, err := os.Lstat(source)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return ErrSymbolicLink
		}
		if !info.Mode().IsRegular() || info.Size() > maxArchiveFileSize {
			return ErrFileTooLarge
		}
		target, err := resolvePath(workspacePath, relative, true)
		if err != nil {
			return err
		}
		backup := ""
		if targetInfo, statErr := os.Lstat(target); statErr == nil {
			if !overwrite || !targetInfo.Mode().IsRegular() || targetInfo.Mode()&os.ModeSymlink != 0 {
				return ErrDestinationBusy
			}
			backup = filepath.Join(backupRoot, filepath.FromSlash(relative))
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		items = append(items, copyItem{source: source, target: target, backup: backup})
	}

	rollback := func(cause error) error {
		var rollbackErrors []error
		for index := len(items) - 1; index >= 0; index-- {
			item := &items[index]
			if item.sourceMoved {
				rollbackErr := os.MkdirAll(filepath.Dir(item.source), 0o755)
				if rollbackErr == nil {
					rollbackErr = os.Rename(item.target, item.source)
				}
				if rollbackErr != nil {
					rollbackErrors = append(rollbackErrors, rollbackErr)
				}
			}
			if item.targetBacked {
				rollbackErr := os.MkdirAll(filepath.Dir(item.target), 0o755)
				if rollbackErr == nil {
					rollbackErr = os.Rename(item.backup, item.target)
				}
				if rollbackErr != nil {
					rollbackErrors = append(rollbackErrors, rollbackErr)
				}
			}
		}
		return errors.Join(append([]error{cause}, rollbackErrors...)...)
	}

	for index := range items {
		item := &items[index]
		if err := os.MkdirAll(filepath.Dir(item.target), 0o755); err != nil {
			return rollback(err)
		}
		if item.backup != "" {
			if err := os.MkdirAll(filepath.Dir(item.backup), 0o700); err != nil {
				return rollback(err)
			}
			if err := os.Rename(item.target, item.backup); err != nil {
				return rollback(err)
			}
			item.targetBacked = true
		}
		if err := os.Rename(item.source, item.target); err != nil {
			return rollback(err)
		}
		item.sourceMoved = true
	}
	return os.RemoveAll(backupRoot)
}

func listArchiveFiles(root string) ([]string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return ErrSymbolicLink
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return ErrInvalidAction
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		normalized, _, err := normalizeArchivePath(filepath.ToSlash(relative))
		if err != nil {
			return err
		}
		paths = append(paths, normalized)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, ErrInvalidAction
	}
	sort.Strings(paths)
	return paths, nil
}

func collectArchiveFiles(workspacePath string, requested []string) ([]string, error) {
	return collectArchiveFilesWithLimit(workspacePath, requested, maxArchiveTotalSize)
}

func collectArchiveFilesWithLimit(workspacePath string, requested []string, maxTotalSize int64) ([]string, error) {
	queued := make([]string, 0, len(requested))
	for _, path := range requested {
		normalized, _, err := normalizeArchivePath(path)
		if err != nil {
			return nil, err
		}
		queued = append(queued, normalized)
	}

	selected := make(map[string]string)
	var totalSize int64
	for len(queued) > 0 {
		relative := queued[0]
		queued = queued[1:]
		key := strings.ToLower(relative)
		if _, exists := selected[key]; exists {
			continue
		}
		target, err := resolvePath(workspacePath, relative, false)
		if err != nil {
			return nil, err
		}
		info, err := os.Lstat(target)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, ErrSymbolicLink
		}
		if info.IsDir() {
			children, err := collectDirectoryFiles(workspacePath, target)
			if err != nil {
				return nil, err
			}
			queued = append(queued, children...)
			continue
		}
		if !info.Mode().IsRegular() || info.Size() > maxArchiveFileSize {
			return nil, ErrFileTooLarge
		}
		totalSize += info.Size()
		if totalSize > maxTotalSize {
			return nil, ErrFileTooLarge
		}
		selected[key] = relative

		dependencies, err := collectFileDependencies(workspacePath, relative, target)
		if err != nil {
			return nil, err
		}
		queued = append(queued, dependencies...)
		if strings.EqualFold(filepath.Ext(relative), ".json") {
			preview := strings.TrimSuffix(relative, filepath.Ext(relative)) + ".png"
			if existsRegularFile(workspacePath, preview) {
				queued = append(queued, preview)
			}
		}
		if len(selected) > maxArchiveFiles {
			return nil, ErrFileTooLarge
		}
	}

	files := make([]string, 0, len(selected))
	for _, relative := range selected {
		files = append(files, relative)
	}
	sort.Strings(files)
	return files, nil
}

func collectDirectoryFiles(workspacePath, directory string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return ErrSymbolicLink
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(workspacePath, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	return files, err
}

func collectFileDependencies(workspacePath, relative, target string) ([]string, error) {
	content, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(filepath.Ext(relative)) {
	case ".json":
		return collectJSONDependencies(workspacePath, relative, content), nil
	case ".obj":
		return collectOBJDependencies(workspacePath, relative, content), nil
	case ".mtl":
		return collectMTLDependencies(workspacePath, relative, content), nil
	default:
		return nil, nil
	}
}

func collectJSONDependencies(workspacePath, relative string, content []byte) []string {
	var document any
	if json.Unmarshal(content, &document) != nil {
		return nil
	}
	dependencies := make(map[string]struct{})
	var visit func(any)
	visit = func(value any) {
		switch typed := value.(type) {
		case string:
			if path, ok := resolveReferencedResource(workspacePath, relative, typed); ok {
				dependencies[path] = struct{}{}
			}
		case []any:
			for _, item := range typed {
				visit(item)
			}
		case map[string]any:
			for _, item := range typed {
				visit(item)
			}
		}
	}
	visit(document)
	return sortedKeys(dependencies)
}

func collectOBJDependencies(workspacePath, relative string, content []byte) []string {
	dependencies := make(map[string]struct{})
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || !strings.EqualFold(fields[0], "mtllib") {
			continue
		}
		for _, reference := range fields[1:] {
			if path, ok := resolveReferencedResource(workspacePath, relative, reference); ok {
				dependencies[path] = struct{}{}
			}
		}
	}
	return sortedKeys(dependencies)
}

func collectMTLDependencies(workspacePath, relative string, content []byte) []string {
	dependencies := make(map[string]struct{})
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || !isMTLTextureCommand(fields[0]) {
			continue
		}
		// MTL 贴图命令可带选项，文件路径位于最后一个参数。
		if path, ok := resolveReferencedResource(workspacePath, relative, fields[len(fields)-1]); ok {
			dependencies[path] = struct{}{}
		}
	}
	return sortedKeys(dependencies)
}

func resolveReferencedResource(workspacePath, owner, reference string) (string, bool) {
	value := strings.TrimSpace(strings.Trim(reference, "\"'"))
	if value == "" || strings.HasPrefix(strings.ToLower(value), "data:") || strings.Contains(value, "://") {
		return "", false
	}
	if index := strings.IndexAny(value, "?#"); index >= 0 {
		value = value[:index]
	}
	value = strings.ReplaceAll(value, "\\", "/")
	candidates := []string{strings.TrimLeft(value, "/")}
	if !strings.HasPrefix(value, "/") {
		candidates = append(candidates, filepath.ToSlash(filepath.Join(filepath.Dir(owner), filepath.FromSlash(value))))
	}
	for _, candidate := range candidates {
		normalized, _, err := normalizeArchivePath(candidate)
		if err != nil || !existsRegularFile(workspacePath, normalized) {
			continue
		}
		return normalized, true
	}
	return "", false
}

func addFileToArchive(writer *zip.Writer, workspacePath, relative string) error {
	target, err := resolvePath(workspacePath, relative, false)
	if err != nil {
		return err
	}
	info, err := os.Lstat(target)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return ErrSymbolicLink
	}
	if !info.Mode().IsRegular() || info.Size() > maxArchiveFileSize {
		return ErrFileTooLarge
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(relative)
	header.Method = zip.Deflate
	destination, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	source, err := os.Open(target)
	if err != nil {
		return err
	}
	defer source.Close()
	_, err = io.Copy(destination, source)
	return err
}

func existsRegularFile(workspacePath, relative string) bool {
	target, err := resolvePath(workspacePath, relative, false)
	if err != nil {
		return false
	}
	info, err := os.Lstat(target)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func isMTLTextureCommand(command string) bool {
	lower := strings.ToLower(command)
	return strings.HasPrefix(lower, "map_") || lower == "bump" || lower == "disp" || lower == "decal" || lower == "refl"
}

func newArchiveToken() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func validArchiveToken(token string) bool {
	if len(token) != 32 {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func archiveRootsForPaths(paths []string) []string {
	roots := make(map[string]struct{})
	for _, path := range paths {
		root := strings.SplitN(filepath.ToSlash(path), "/", 2)[0]
		if _, allowed := archiveRoots[root]; allowed {
			roots[root] = struct{}{}
		}
	}
	return sortedKeys(roots)
}
