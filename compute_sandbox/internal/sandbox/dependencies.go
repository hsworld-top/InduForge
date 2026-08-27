package sandbox

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"
)

const maxDependencyArchiveBytes = 64 << 20

var (
	dependencyProjectPattern = regexp.MustCompile(`^[0-9a-fA-F-]{36}$`)
	nodePackagePattern       = regexp.MustCompile(`^(?:@[a-z0-9._-]+/)?[a-z0-9._-]+$`)
	pythonPackagePattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	dependencyVersionPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)+(?:[0-9A-Za-z._+~-]*)$`)
	nodeImportPattern        = regexp.MustCompile(`^(?:@[a-z0-9._-]+/)?[a-z0-9._-]+(?:/[a-z0-9._-]+)*$`)
	pythonImportPattern      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)
)

type dependencySpec struct {
	Language    string `json:"language"`
	PackageName string `json:"packageName"`
	ImportName  string `json:"importName"`
	Version     string `json:"version"`
}

type dependencyMutationRequest struct {
	ProjectID string `json:"projectId"`
	dependencySpec
}

func installDependencyHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request dependencyMutationRequest
		if err := decodeRequest(w, r, &request); err != nil {
			return
		}
		if err := validateDependencyMutation(&request, false); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		result, err := installDependency(r.Context(), config, request, "", false)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func importDependencyHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxDependencyArchiveBytes+(1<<20))
		if err := r.ParseMultipartForm(maxDependencyArchiveBytes); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("离线依赖文件无效或超过 64 MB"))
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("请选择离线依赖文件"))
			return
		}
		defer file.Close()
		request := dependencyMutationRequest{ProjectID: r.FormValue("projectId")}
		request.Language = strings.ToLower(strings.TrimSpace(r.FormValue("language")))
		if request.Language != "js" && request.Language != "python" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("language 仅支持 js/python"))
			return
		}
		if !validDependencyProjectID(request.ProjectID) {
			writeError(w, http.StatusBadRequest, fmt.Errorf("projectId 无效"))
			return
		}
		archivePath, cleanup, err := persistDependencyArchive(file, header, request.Language)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		defer cleanup()
		metadata, err := inspectDependencyArchive(archivePath, request.Language)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		request.dependencySpec = metadata
		if err := validateDependencyMutation(&request, false); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		result, err := installDependency(r.Context(), config, request, archivePath, true)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func uninstallDependencyHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request dependencyMutationRequest
		if err := decodeRequest(w, r, &request); err != nil {
			return
		}
		if err := validateDependencyMutation(&request, true); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		installDir := dependencyInstallDir(config.DependenciesDir, request.ProjectID, request.Language, request.PackageName)
		if err := os.RemoveAll(installDir); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("删除依赖失败"))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "removed"})
	}
}

func installDependency(ctx context.Context, config Config, request dependencyMutationRequest, archivePath string, offline bool) (dependencySpec, error) {
	installCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	baseDir := filepath.Join(config.DependenciesDir, request.ProjectID, request.Language)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return dependencySpec{}, fmt.Errorf("创建依赖目录失败")
	}
	tempDir, err := os.MkdirTemp(baseDir, ".install-")
	if err != nil {
		return dependencySpec{}, fmt.Errorf("创建依赖安装目录失败")
	}
	if err := os.Chmod(tempDir, 0o755); err != nil {
		return dependencySpec{}, fmt.Errorf("设置依赖目录权限失败")
	}
	defer os.RemoveAll(tempDir)

	command := dependencyInstallCommand(installCtx, config, request, tempDir, archivePath, offline)
	command.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "HOME=/tmp", "LANG=C.UTF-8", "PIP_NO_CACHE_DIR=1", "npm_config_cache=/tmp/npm-cache"}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if len(message) > 1000 {
			message = message[len(message)-1000:]
		}
		return dependencySpec{}, fmt.Errorf("依赖安装失败: %s", message)
	}

	resolved, err := resolveInstalledDependency(tempDir, request.Language, request.PackageName)
	if err != nil {
		return dependencySpec{}, err
	}
	resolved.Language = request.Language
	validated := dependencyMutationRequest{ProjectID: request.ProjectID, dependencySpec: resolved}
	if err := validateDependencyMutation(&validated, true); err != nil {
		return dependencySpec{}, fmt.Errorf("依赖元数据无效: %w", err)
	}
	finalDir := dependencyInstallDir(config.DependenciesDir, request.ProjectID, request.Language, resolved.PackageName)
	if err := os.RemoveAll(finalDir); err != nil {
		return dependencySpec{}, fmt.Errorf("清理旧依赖失败")
	}
	if err := os.Chmod(tempDir, 0o755); err != nil {
		return dependencySpec{}, fmt.Errorf("设置依赖目录权限失败")
	}
	if err := os.Rename(tempDir, finalDir); err != nil {
		return dependencySpec{}, fmt.Errorf("保存依赖失败")
	}
	return resolved, nil
}

func dependencyInstallCommand(ctx context.Context, config Config, request dependencyMutationRequest, installDir, archivePath string, offline bool) *exec.Cmd {
	if request.Language == "js" {
		source := archivePath
		if source == "" {
			source = request.PackageName
			if request.Version != "" {
				source += "@" + request.Version
			}
		}
		args := []string{"install", "--prefix", installDir, "--ignore-scripts", "--no-audit", "--no-fund", "--save-exact", "--omit=dev"}
		if offline {
			args = append(args, "--offline")
		}
		args = append(args, source)
		return exec.CommandContext(ctx, "/usr/local/bin/npm", args...)
	}
	source := archivePath
	if source == "" {
		source = request.PackageName
		if request.Version != "" {
			source += "==" + request.Version
		}
	}
	args := []string{"-m", "pip", "install", "--disable-pip-version-check", "--only-binary=:all:", "--target", installDir}
	if offline {
		args = append(args, "--no-index", "--no-deps")
	}
	args = append(args, source)
	return exec.CommandContext(ctx, config.PythonBinary, args...)
}

func resolveInstalledDependency(installDir, language, requestedPackage string) (dependencySpec, error) {
	if language == "js" {
		parts := append([]string{installDir, "node_modules"}, strings.Split(requestedPackage, "/")...)
		encoded, err := os.ReadFile(filepath.Join(filepath.Join(parts...), "package.json"))
		if err != nil {
			return dependencySpec{}, fmt.Errorf("无法读取 npm 包元数据")
		}
		var metadata struct{ Name, Version string }
		if err := json.Unmarshal(encoded, &metadata); err != nil {
			return dependencySpec{}, fmt.Errorf("npm 包元数据无效")
		}
		return dependencySpec{PackageName: metadata.Name, ImportName: metadata.Name, Version: metadata.Version}, nil
	}

	entries, err := os.ReadDir(installDir)
	if err != nil {
		return dependencySpec{}, fmt.Errorf("无法读取 Python 包元数据")
	}
	wanted := normalizePythonPackageName(requestedPackage)
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".dist-info") {
			continue
		}
		distDir := filepath.Join(installDir, entry.Name())
		name, version, err := readPythonMetadata(filepath.Join(distDir, "METADATA"))
		if err != nil || normalizePythonPackageName(name) != wanted {
			continue
		}
		importName := readPythonTopLevel(distDir, installDir)
		if importName == "" {
			return dependencySpec{}, fmt.Errorf("无法识别 Python 包导入名称")
		}
		return dependencySpec{PackageName: name, ImportName: importName, Version: version}, nil
	}
	return dependencySpec{}, fmt.Errorf("无法识别已安装的 Python 包")
}

func persistDependencyArchive(file multipart.File, header *multipart.FileHeader, language string) (string, func(), error) {
	extension := strings.ToLower(filepath.Ext(header.Filename))
	if (language == "js" && !strings.HasSuffix(strings.ToLower(header.Filename), ".tgz")) || (language == "python" && extension != ".whl") {
		return "", func() {}, fmt.Errorf("JavaScript 仅支持 .tgz，Python 仅支持 .whl")
	}
	temp, err := os.CreateTemp("", "induforge-dependency-*"+extension)
	if err != nil {
		return "", func() {}, fmt.Errorf("保存离线依赖失败")
	}
	cleanup := func() { _ = os.Remove(temp.Name()) }
	written, err := io.Copy(temp, io.LimitReader(file, maxDependencyArchiveBytes+1))
	closeErr := temp.Close()
	if err != nil || closeErr != nil || written > maxDependencyArchiveBytes {
		cleanup()
		return "", func() {}, fmt.Errorf("离线依赖文件无效或超过 64 MB")
	}
	return temp.Name(), cleanup, nil
}

func inspectDependencyArchive(path, language string) (dependencySpec, error) {
	if language == "js" {
		file, err := os.Open(path)
		if err != nil {
			return dependencySpec{}, fmt.Errorf("无法读取 npm 离线包")
		}
		defer file.Close()
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return dependencySpec{}, fmt.Errorf("npm 离线包格式无效")
		}
		defer gzipReader.Close()
		tarReader := tar.NewReader(gzipReader)
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return dependencySpec{}, fmt.Errorf("npm 离线包格式无效")
			}
			if filepath.ToSlash(header.Name) != "package/package.json" || header.Size > 1<<20 {
				continue
			}
			encoded, err := io.ReadAll(io.LimitReader(tarReader, 1<<20))
			if err != nil {
				return dependencySpec{}, fmt.Errorf("npm 包元数据无效")
			}
			var metadata struct{ Name, Version string }
			if err := json.Unmarshal(encoded, &metadata); err != nil {
				return dependencySpec{}, fmt.Errorf("npm 包元数据无效")
			}
			return dependencySpec{Language: "js", PackageName: metadata.Name, ImportName: metadata.Name, Version: metadata.Version}, nil
		}
		return dependencySpec{}, fmt.Errorf("npm 离线包缺少 package.json")
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		return dependencySpec{}, fmt.Errorf("Python wheel 格式无效")
	}
	defer reader.Close()
	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, ".dist-info/METADATA") || file.UncompressedSize64 > 1<<20 {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			return dependencySpec{}, fmt.Errorf("Python wheel 元数据无效")
		}
		name, version, readErr := readPythonMetadataReader(opened)
		_ = opened.Close()
		if readErr != nil {
			return dependencySpec{}, readErr
		}
		return dependencySpec{Language: "python", PackageName: name, Version: version}, nil
	}
	return dependencySpec{}, fmt.Errorf("Python wheel 缺少有效元数据")
}

func readPythonMetadata(path string) (string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer file.Close()
	return readPythonMetadataReader(file)
}

func readPythonMetadataReader(reader io.Reader) (string, string, error) {
	encoded, err := io.ReadAll(io.LimitReader(reader, 1<<20))
	if err != nil {
		return "", "", fmt.Errorf("Python 包元数据无效")
	}
	var name, version string
	for _, line := range strings.Split(string(encoded), "\n") {
		if strings.HasPrefix(line, "Name: ") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "Name: "))
		}
		if strings.HasPrefix(line, "Version: ") {
			version = strings.TrimSpace(strings.TrimPrefix(line, "Version: "))
		}
	}
	if name == "" || version == "" {
		return "", "", fmt.Errorf("Python 包元数据缺少名称或版本")
	}
	return name, version, nil
}

func readPythonTopLevel(distDir, installDir string) string {
	if encoded, err := os.ReadFile(filepath.Join(distDir, "top_level.txt")); err == nil {
		for _, line := range strings.Split(string(encoded), "\n") {
			name := strings.TrimSpace(line)
			if pythonImportPattern.MatchString(name) {
				return name
			}
		}
	}
	entries, _ := os.ReadDir(installDir)
	candidates := make([]string, 0)
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".py")
		if strings.HasSuffix(entry.Name(), ".dist-info") || strings.HasSuffix(entry.Name(), ".data") || strings.HasPrefix(name, "_") {
			continue
		}
		if pythonImportPattern.MatchString(name) {
			candidates = append(candidates, name)
		}
	}
	sort.Strings(candidates)
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

func normalizePythonPackageName(value string) string {
	return strings.Trim(strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(value)), "-")
}

func validateDependencyMutation(request *dependencyMutationRequest, requireResolved bool) error {
	if request == nil {
		return fmt.Errorf("请求不能为空")
	}
	if !validDependencyProjectID(request.ProjectID) {
		return fmt.Errorf("projectId 无效")
	}
	request.Language = strings.ToLower(strings.TrimSpace(request.Language))
	request.PackageName = strings.TrimSpace(request.PackageName)
	request.ImportName = strings.TrimSpace(request.ImportName)
	request.Version = strings.TrimSpace(request.Version)
	if request.Language != "js" && request.Language != "python" {
		return fmt.Errorf("language 仅支持 js/python")
	}
	packagePattern := nodePackagePattern
	if request.Language == "python" {
		packagePattern = pythonPackagePattern
	}
	if !packagePattern.MatchString(request.PackageName) {
		return fmt.Errorf("依赖包名格式无效")
	}
	if !requireResolved {
		if request.Version != "" && !dependencyVersionPattern.MatchString(request.Version) {
			return fmt.Errorf("依赖版本格式无效")
		}
		return nil
	}
	importPattern := nodeImportPattern
	if request.Language == "python" {
		importPattern = pythonImportPattern
	}
	if !importPattern.MatchString(request.ImportName) {
		return fmt.Errorf("导入名称格式无效")
	}
	if !dependencyVersionPattern.MatchString(request.Version) {
		return fmt.Errorf("依赖版本格式无效")
	}
	return nil
}

func validDependencyProjectID(projectID string) bool {
	return dependencyProjectPattern.MatchString(strings.TrimSpace(projectID))
}

func dependencyInstallDir(root, projectID, language, packageName string) string {
	digest := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(packageName))))
	return filepath.Join(root, projectID, language, fmt.Sprintf("%x", digest[:12]))
}
