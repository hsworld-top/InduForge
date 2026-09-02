// Package loader 严格加载 k3s release-pvc 或本机受控 native-release 中的 RuntimeEngine 配置和项目 Artifact。
package loader

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/indu-forge/runtime-engine/internal/model"
)

const (
	maxConfigBytes   = 1 << 20
	maxArtifactBytes = 16 << 20
)

type Options struct {
	ConfigPath string
	// ConfigRoot 是命令行提供的可信目录；留空时仅允许 ConfigPath 所在目录。
	// 它在校验与打开之间仍可能被有权限者替换，部署需由宿主控制该根目录写权限。
	ConfigRoot string
	// RequireReadOnlyMount 默认 true；仅受控的非 Linux 测试可显式关闭。
	RequireReadOnlyMount *bool
	MountInfoPath        string
}

type Loaded struct {
	Config             model.EngineConfig
	Artifact           model.ProjectArtifact
	ConfigBytes        []byte
	ArtifactBytes      []byte
	ConfigPath         string
	ArtifactPath       string
	ConfigSHA256       string
	ArtifactSHA256     string
	CollectorArtifacts map[string]model.CollectorArtifact
}

// DiagnosticCode 将启动期的制品错误归类为固定阶段码。日志只使用该值，不能回显
// 文件路径、摘要或配置内容；对外健康原因仍由 RuntimeEngine 保持稳定错误码。
func DiagnosticCode(err error) string {
	if err == nil {
		return "ok"
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "配置文件"):
		return "config-file"
	case strings.Contains(message, "配置根目录"):
		return "config-root"
	case strings.Contains(message, "runtime-engine-config"):
		return "config-schema"
	case strings.Contains(message, "artifactMount"):
		return "artifact-mount"
	case strings.Contains(message, "只读挂载"):
		return "artifact-mount-readonly"
	case strings.Contains(message, "项目 Artifact"):
		return "project-artifact"
	case strings.Contains(message, "collector"):
		return "collector-artifact"
	default:
		return "artifact-contract"
	}
}

func Load(options Options) (*Loaded, error) {
	if options.ConfigPath == "" {
		return nil, fmt.Errorf("未提供 RuntimeEngine 配置路径")
	}
	configPath, err := requireAbsoluteExistingFile(options.ConfigPath, "配置文件")
	if err != nil {
		return nil, err
	}
	root := options.ConfigRoot
	if root == "" {
		root = filepath.Dir(configPath)
	}
	configRoot, err := requireAbsoluteExistingDirectory(root, "配置根目录")
	if err != nil {
		return nil, err
	}
	if !within(configRoot, configPath) {
		return nil, fmt.Errorf("配置文件 %q 经 clean/eval 后逃逸配置根 %q", configPath, configRoot)
	}
	configBytes, err := readRegularFile(configPath, maxConfigBytes, "配置文件")
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	schemas, err := compileSchemas()
	if err != nil {
		return nil, err
	}
	configSchema, err := schemas.engineConfigSchema(configBytes)
	if err != nil {
		return nil, err
	}
	if err := validateJSON(configSchema, configBytes, "runtime-engine-config"); err != nil {
		return nil, err
	}
	var config model.EngineConfig
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("反序列化 runtime-engine-config 失败: %w", err)
	}
	if err := model.ValidateEngineConfig(config); err != nil {
		return nil, err
	}

	// Schema 已限制 artifactFile 为相对路径；这里仍以文件系统最终解析结果复核 symlink 边界。
	if err := validateMountPath(config.ArtifactMount.MountPath); err != nil {
		return nil, err
	}
	mountPath, err := requireAbsoluteExistingDirectory(config.ArtifactMount.MountPath, "artifactMount.mountPath")
	if err != nil {
		return nil, err
	}
	if err := validateArtifactFile(config.ArtifactMount.ArtifactFile); err != nil {
		return nil, err
	}
	// native-release 的根就是节点 Agent 选定的不可变 release 根；先检查根本身，
	// 再检查每个文件，防止只读根下叠加可写子挂载绕过校验。
	if config.SchemaVersion == "runtime-engine.config.v2" {
		if err := requireArtifactReadOnly(options, mountPath); err != nil {
			return nil, fmt.Errorf("native-release 根必须为只读挂载: %w", err)
		}
	}
	artifactPath, artifactBytes, err := readMountedArtifact(mountPath, config.ArtifactMount.ArtifactFile, "项目 Artifact")
	if err != nil {
		return nil, fmt.Errorf("读取项目 Artifact 失败: %w", err)
	}
	if err := requireArtifactReadOnly(options, artifactPath); err != nil {
		return nil, err
	}
	digest := sha256.Sum256(artifactBytes)
	actualDigest := "sha256:" + hex.EncodeToString(digest[:])
	configDigest := sha256.Sum256(configBytes)
	if actualDigest != config.ProjectArtifact.ArtifactDigest {
		return nil, fmt.Errorf("项目 Artifact 原始字节 SHA-256 不匹配: 期望 %s，实际 %s", config.ProjectArtifact.ArtifactDigest, actualDigest)
	}
	if err := validateJSON(schemas.projectArtifact, artifactBytes, "runtime-project-artifact"); err != nil {
		return nil, err
	}
	var artifact model.ProjectArtifact
	if err := json.Unmarshal(artifactBytes, &artifact); err != nil {
		return nil, fmt.Errorf("反序列化 runtime-project-artifact 失败: %w", err)
	}
	if config.ProjectID != artifact.ProjectID {
		return nil, fmt.Errorf("config projectId %q 与项目 Artifact projectId %q 不一致", config.ProjectID, artifact.ProjectID)
	}
	if err := model.ValidateProjectArtifact(artifact, config); err != nil {
		return nil, err
	}
	collectorArtifacts := make(map[string]model.CollectorArtifact)
	for _, assignment := range config.ProducerAssignments {
		if assignment.ProducerType != "collector" {
			continue
		}
		binding := assignment.CollectorArtifact
		if binding == nil {
			return nil, fmt.Errorf("collector %q 缺少 collectorArtifact", assignment.CollectorID)
		}
		if binding.ArtifactFile == config.ArtifactMount.ArtifactFile {
			return nil, fmt.Errorf("collector %q artifactFile 不得与项目 Artifact 相同", assignment.CollectorID)
		}
		path, bytes, readErr := readMountedArtifact(mountPath, binding.ArtifactFile, "Collector Artifact")
		if readErr != nil {
			return nil, readErr
		}
		if err := requireArtifactReadOnly(options, path); err != nil {
			return nil, err
		}
		collectorDigest := digestArtifact(bytes)
		if collectorDigest != binding.Artifact.ArtifactDigest {
			return nil, fmt.Errorf("collector %q Artifact 原始字节 SHA-256 不匹配: 期望 %s，实际 %s", assignment.CollectorID, binding.Artifact.ArtifactDigest, collectorDigest)
		}
		if err := validateJSON(schemas.collectorArtifact, bytes, "collector-runtime-artifact"); err != nil {
			return nil, err
		}
		var collectorArtifact model.CollectorArtifact
		if err := json.Unmarshal(bytes, &collectorArtifact); err != nil {
			return nil, fmt.Errorf("反序列化 collector-runtime-artifact 失败: %w", err)
		}
		if collectorArtifact.ArtifactID != binding.Artifact.ArtifactID || collectorArtifact.ArtifactRevision != binding.Artifact.ArtifactRevision {
			return nil, fmt.Errorf("collector %q Artifact identity 与配置引用不一致", assignment.CollectorID)
		}
		if _, exists := collectorArtifacts[assignment.CollectorID]; exists {
			return nil, fmt.Errorf("collector %q 存在重复 Artifact", assignment.CollectorID)
		}
		_ = path // 已通过原始路径、no-follow 和 TOCTOU 校验；返回 map 无需暴露载体路径。
		collectorArtifacts[assignment.CollectorID] = collectorArtifact
	}
	// compute/alarm 的分布式工作负载不挂载 collector Artifact；采集映射由
	// collector 自己的受控 binding 校验。只有声明 collector producer 时才在此复核映射。
	if len(collectorArtifacts) > 0 {
		if err := model.ValidateCollectorIngressMappings(config, artifact, collectorArtifacts); err != nil {
			return nil, err
		}
	}
	return &Loaded{Config: config, Artifact: artifact, CollectorArtifacts: collectorArtifacts, ConfigBytes: configBytes, ArtifactBytes: artifactBytes, ConfigPath: configPath, ArtifactPath: artifactPath, ConfigSHA256: "sha256:" + hex.EncodeToString(configDigest[:]), ArtifactSHA256: actualDigest}, nil
}

func readMountedArtifact(mountPath, artifactFile, label string) (string, []byte, error) {
	if err := validateArtifactFile(artifactFile); err != nil {
		return "", nil, err
	}
	candidate := filepath.Join(mountPath, artifactFile)
	path, err := requireAbsoluteExistingFile(candidate, label+" 文件")
	if err != nil {
		return "", nil, err
	}
	if !within(mountPath, path) {
		return "", nil, fmt.Errorf("%s %q 经 clean/eval 后逃逸挂载根 %q", label, path, mountPath)
	}
	bytes, err := readArtifactFile(path, maxArtifactBytes)
	if err != nil {
		return "", nil, fmt.Errorf("读取%s失败: %w", label, err)
	}
	return path, bytes, nil
}

func digestArtifact(bytes []byte) string {
	digest := sha256.Sum256(bytes)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func requireArtifactReadOnly(options Options, path string) error {
	if !requireReadOnlyMount(options) {
		return nil
	}
	mountInfoPath := options.MountInfoPath
	if mountInfoPath == "" {
		mountInfoPath = "/proc/self/mountinfo"
	}
	mountInfo, err := os.ReadFile(mountInfoPath)
	if err != nil {
		return fmt.Errorf("读取 Linux mountinfo 失败: %w", err)
	}
	return mountReadOnly(path, string(mountInfo))
}

func requireReadOnlyMount(options Options) bool {
	return options.RequireReadOnlyMount == nil || *options.RequireReadOnlyMount
}

func requireAbsoluteExistingFile(path, label string) (string, error) {
	evaluated, info, err := resolveAbsolute(path, label)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s 必须是普通文件: %q", label, path)
	}
	return evaluated, nil
}

func requireAbsoluteExistingDirectory(path, label string) (string, error) {
	evaluated, info, err := resolveAbsolute(path, label)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s 必须是目录: %q", label, path)
	}
	return evaluated, nil
}

func resolveAbsolute(path, label string) (string, os.FileInfo, error) {
	if strings.ContainsRune(path, '\x00') {
		return "", nil, fmt.Errorf("%s 路径不能包含 NUL", label)
	}
	if !filepath.IsAbs(path) {
		return "", nil, fmt.Errorf("%s 路径必须为绝对路径: %q", label, path)
	}
	clean := filepath.Clean(path)
	if clean != path {
		return "", nil, fmt.Errorf("%s 路径必须已规范化: %q", label, path)
	}
	if err := rejectPathSymlinks(clean, label); err != nil {
		return "", nil, err
	}
	evaluated, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return "", nil, fmt.Errorf("解析%s路径 %q 失败: %w", label, clean, err)
	}
	info, err := os.Stat(evaluated)
	if err != nil {
		return "", nil, fmt.Errorf("读取%s属性失败: %w", label, err)
	}
	return evaluated, info, nil
}

func rejectPathSymlinks(path, label string) error {
	current := string(filepath.Separator)
	for _, segment := range strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator)) {
		if segment == "" {
			continue
		}
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("检查%s路径段 %q 失败: %w", label, current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s 路径链不得包含符号链接: %q", label, current)
		}
	}
	return nil
}

func readRegularFile(path string, limit int64, label string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开%s失败: %w", label, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s 不是稳定的普通文件", label)
	}
	bytes, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("读取%s失败: %w", label, err)
	}
	if int64(len(bytes)) > limit {
		return nil, fmt.Errorf("%s 超过 %d 字节上限", label, limit)
	}
	if err := rejectDuplicateJSONKeys(bytes); err != nil {
		return nil, fmt.Errorf("%s JSON 键无效: %w", label, err)
	}
	return bytes, nil
}

func readArtifactFile(path string, limit int64) ([]byte, error) {
	var before unix.Stat_t
	if err := unix.Lstat(path, &before); err != nil {
		return nil, fmt.Errorf("检查项目 Artifact 文件失败: %w", err)
	}
	if before.Mode&unix.S_IFMT != unix.S_IFREG {
		return nil, fmt.Errorf("项目 Artifact 文件必须是普通文件")
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("以 no-follow 打开项目 Artifact 失败: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	var after unix.Stat_t
	if err := unix.Fstat(fd, &after); err != nil {
		return nil, fmt.Errorf("读取项目 Artifact 文件属性失败: %w", err)
	}
	if after.Mode&unix.S_IFMT != unix.S_IFREG || before.Dev != after.Dev || before.Ino != after.Ino {
		return nil, fmt.Errorf("项目 Artifact 文件在校验和打开之间发生替换")
	}
	bytes, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("读取项目 Artifact 文件失败: %w", err)
	}
	if int64(len(bytes)) > limit {
		return nil, fmt.Errorf("项目 Artifact 文件超过 %d 字节上限", limit)
	}
	if err := rejectDuplicateJSONKeys(bytes); err != nil {
		return nil, fmt.Errorf("项目 Artifact 文件 JSON 键无效: %w", err)
	}
	return bytes, nil
}

func rejectDuplicateJSONKeys(bytes []byte) error {
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	var read func() error
	read = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			keys := map[string]struct{}{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return fmt.Errorf("对象键不是字符串")
				}
				if _, exists := keys[name]; exists {
					return fmt.Errorf("存在重复键 %q", name)
				}
				keys[name] = struct{}{}
				if err := read(); err != nil {
					return err
				}
			}
			_, err := decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := read(); err != nil {
					return err
				}
			}
			_, err := decoder.Token()
			return err
		default:
			return fmt.Errorf("非法 JSON 分隔符")
		}
	}
	if err := read(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("存在额外 JSON 内容")
		}
		return err
	}
	return nil
}

func validateMountPath(path string) error {
	if strings.ContainsRune(path, '\x00') || !filepath.IsAbs(path) {
		return fmt.Errorf("artifactMount.mountPath 必须是无 NUL 的绝对路径")
	}
	if path != "/" && strings.HasSuffix(path, "/") {
		return fmt.Errorf("artifactMount.mountPath 不能有尾随斜线")
	}
	for _, segment := range strings.Split(path, "/")[1:] {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("artifactMount.mountPath 不能含空、. 或 .. 路径段")
		}
	}
	return nil
}

func validateArtifactFile(path string) error {
	if path == "" || strings.ContainsRune(path, '\x00') || filepath.IsAbs(path) {
		return fmt.Errorf("artifactMount.artifactFile 必须是无 NUL 的相对文件路径")
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("artifactMount.artifactFile 不能含空、. 或 .. 路径段")
		}
	}
	return nil
}

func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
