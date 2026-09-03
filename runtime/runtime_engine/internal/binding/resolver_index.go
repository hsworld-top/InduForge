package binding

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/indu-forge/runtime-engine/internal/model"
)

const (
	configFileName = "runtime-engine-config.json"
	indexFileName  = "site-index.json"
)

// ResolverIndex 是 collector-runtime-index.v1 的无凭据表示。Secrets 的值只能
// 是相对于 index 文件的受控挂载路径，Resolver 会在运行时以 no-follow 方式读取。
type ResolverIndex struct {
	SchemaVersion string                     `json:"schemaVersion"`
	Resources     map[string]json.RawMessage `json:"resources"`
	Secrets       map[string]string          `json:"secrets"`
}

// BuildResolverIndex 从同一部署输入构造 resolver 可接受的 source index。它只
// 写入 NATS/Sandbox endpoint 和 resource 元数据，以及挂载 secret 文件路径；
// token、password、DSN 等值保持在运行时只读 secret 文件中。
func BuildResolverIndex(input Input) (ResolverIndex, error) {
	if err := validateResolverInput(input); err != nil {
		return ResolverIndex{}, err
	}

	resources := map[string]json.RawMessage{}
	secrets := map[string]string{}
	if err := addResource(resources, input.JetStream.ServerResourceRef, map[string]string{
		"url":       input.JetStream.Endpoint,
		"accountId": input.AccountID,
	}); err != nil {
		return ResolverIndex{}, err
	}
	if err := addSecret(secrets, input.JetStream.CredentialSecretRef, input.JetStream.CredentialSecretFile); err != nil {
		return ResolverIndex{}, err
	}
	// 当前 Resolver.ResolvePostgres 只消费 dsnSecretRef，但正式 index 允许
	// 保存非敏感资源描述。写入 schema 让部署预检上下文与引用保持同一快照。
	if err := addResource(resources, input.StateStore.ResourceRef, map[string]string{"schema": input.StateStore.Schema}); err != nil {
		return ResolverIndex{}, err
	}
	if err := addSecret(secrets, input.StateStore.CredentialSecretRef, input.StateStore.CredentialSecretFile); err != nil {
		return ResolverIndex{}, err
	}
	if input.Role == roleCompute {
		if err := addResource(resources, input.ComputeSandbox.ServerResourceRef, map[string]string{"url": input.ComputeSandboxEndpoint}); err != nil {
			return ResolverIndex{}, err
		}
		if err := addSecret(secrets, input.ComputeSandbox.CredentialSecretRef, input.ComputeSandboxSecretFile); err != nil {
			return ResolverIndex{}, err
		}
	}
	index := ResolverIndex{SchemaVersion: "collector-runtime-index.v1", Resources: resources, Secrets: secrets}
	if err := validateResolverIndex(index); err != nil {
		return ResolverIndex{}, err
	}
	return index, nil
}

// WriteBundle 先在 targetDir 的同级目录完整写入并 fsync 两个 0600 文件，再以
// rename 安装该目录。替换已有 bundle 时，失败路径会恢复旧目录，不会留下半写配置。
func WriteBundle(config model.EngineConfig, index ResolverIndex, targetDir string) error {
	if err := model.ValidateEngineConfig(config); err != nil {
		return fmt.Errorf("写入 bundle 前 RuntimeEngine 配置未通过正式校验: %w", err)
	}
	if err := validateResolverIndex(index); err != nil {
		return err
	}
	if !filepath.IsAbs(targetDir) || filepath.Clean(targetDir) != targetDir || targetDir == string(filepath.Separator) {
		return fmt.Errorf("bundle 目标目录必须是规范绝对路径")
	}
	parent, name := filepath.Dir(targetDir), filepath.Base(targetDir)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("创建 bundle 父目录失败: %w", err)
	}
	temporary, err := os.MkdirTemp(parent, "."+name+".tmp-")
	if err != nil {
		return fmt.Errorf("创建 bundle 临时目录失败: %w", err)
	}
	defer os.RemoveAll(temporary)
	// 主运行容器将 /work 作为只读 bind mount，再把受控 Secret 投影到
	// bundle/secrets。挂载点必须由仍可写的 init 容器预先创建，不能在
	// 只读父挂载下由 OCI 运行时临时 mkdir。
	if err = os.Mkdir(filepath.Join(temporary, "secrets"), 0o700); err != nil {
		return fmt.Errorf("创建 bundle secret 挂载点失败: %w", err)
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化 RuntimeEngine 配置失败: %w", err)
	}
	indexBytes, err := json.Marshal(index)
	if err != nil {
		return fmt.Errorf("序列化 resolver index 失败: %w", err)
	}
	if err := writePrivateFile(filepath.Join(temporary, configFileName), configBytes); err != nil {
		return err
	}
	if err := writePrivateFile(filepath.Join(temporary, indexFileName), indexBytes); err != nil {
		return err
	}
	if err := syncDirectory(temporary); err != nil {
		return fmt.Errorf("同步 bundle 临时目录失败: %w", err)
	}
	if err := installBundle(temporary, targetDir); err != nil {
		return err
	}
	return nil
}

func validateResolverInput(input Input) error {
	for label, value := range map[string]string{
		"tenantId":                        input.TenantID,
		"accountId":                       input.AccountID,
		"jetStream.endpoint":              input.JetStream.Endpoint,
		"jetStream.serverResourceRef":     input.JetStream.ServerResourceRef,
		"jetStream.credentialSecretRef":   input.JetStream.CredentialSecretRef,
		"jetStream.credentialSecretFile":  input.JetStream.CredentialSecretFile,
		"stateStore.resourceRef":          input.StateStore.ResourceRef,
		"stateStore.credentialSecretRef":  input.StateStore.CredentialSecretRef,
		"stateStore.credentialSecretFile": input.StateStore.CredentialSecretFile,
		"stateStore.schema":               input.StateStore.Schema,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("构造 resolver index 缺少 %s", label)
		}
	}
	if !safeSecretPath(input.JetStream.CredentialSecretFile) || !safeSecretPath(input.StateStore.CredentialSecretFile) {
		return fmt.Errorf("resolver index secret 文件路径必须为规范相对路径")
	}
	if input.Role == roleCompute {
		if input.ComputeSandbox == nil || strings.TrimSpace(input.ComputeSandboxEndpoint) == "" || strings.TrimSpace(input.ComputeSandboxSecretFile) == "" || !safeSecretPath(input.ComputeSandboxSecretFile) || !validSandboxEndpoint(input.ComputeSandboxEndpoint) {
			return fmt.Errorf("compute resolver index 缺少 sandbox endpoint 或挂载 secret 文件")
		}
	} else if input.Role != roleWriter && input.Role != roleAlarm {
		return fmt.Errorf("RuntimeEngine role 必须为 writer、compute 或 alarm")
	}
	if !validNATSEndpoint(input.JetStream.Endpoint) {
		return fmt.Errorf("resolver index NATS endpoint 非法")
	}
	return nil
}

func validNATSEndpoint(endpoint string) bool {
	parsed, err := url.Parse(endpoint)
	return err == nil && (parsed.Scheme == "nats" || parsed.Scheme == "tls") && parsed.Host != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == ""
}

func validSandboxEndpoint(endpoint string) bool {
	parsed, err := url.Parse(endpoint)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == "" && parsed.Path == ""
}

func validateResolverIndex(index ResolverIndex) error {
	if index.SchemaVersion != "collector-runtime-index.v1" || index.Resources == nil || index.Secrets == nil {
		return fmt.Errorf("resolver index 格式无效")
	}
	for ref, raw := range index.Resources {
		if !validReference(ref, "site-resource://") || len(raw) == 0 || !json.Valid(raw) {
			return fmt.Errorf("resolver index resource 无效")
		}
	}
	for ref, path := range index.Secrets {
		if !validReference(ref, "secret://") || !safeSecretPath(path) {
			return fmt.Errorf("resolver index secret 引用无效")
		}
	}
	return nil
}

func addResource(resources map[string]json.RawMessage, ref string, value any) error {
	if _, exists := resources[ref]; exists {
		return fmt.Errorf("resolver index resourceRef 重复: %s", ref)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	resources[ref] = raw
	return nil
}

func addSecret(secrets map[string]string, ref, path string) error {
	if _, exists := secrets[ref]; exists {
		return fmt.Errorf("resolver index credentialSecretRef 重复: %s", ref)
	}
	secrets[ref] = path
	return nil
}

func validReference(value, prefix string) bool {
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	tail := strings.TrimPrefix(value, prefix)
	if len(tail) == 0 || len(tail) > 256 || !asciiAlphaNumeric(tail[0]) {
		return false
	}
	for _, character := range tail[1:] {
		if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || strings.ContainsRune("._:/@-", character)) {
			return false
		}
	}
	return true
}

func asciiAlphaNumeric(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9'
}

func safeSecretPath(path string) bool {
	if path == "" || filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsAny(path, "\\:") || strings.ContainsRune(path, '\x00') || path == "." || path == ".." || strings.HasPrefix(path, "../") {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func writePrivateFile(path string, value []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("创建 bundle 文件失败: %w", err)
	}
	if _, err := file.Write(value); err != nil {
		file.Close()
		return fmt.Errorf("写入 bundle 文件失败: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("同步 bundle 文件失败: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("关闭 bundle 文件失败: %w", err)
	}
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func installBundle(temporary, target string) error {
	info, err := os.Lstat(target)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("检查旧 bundle 失败: %w", err)
	}
	if os.IsNotExist(err) {
		if err := os.Rename(temporary, target); err != nil {
			return fmt.Errorf("安装 bundle 失败: %w", err)
		}
		if err := syncDirectory(filepath.Dir(target)); err != nil {
			return fmt.Errorf("同步 bundle 父目录失败: %w", err)
		}
		return nil
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("旧 bundle 必须是非链接目录")
	}
	backup, err := os.MkdirTemp(filepath.Dir(target), "."+filepath.Base(target)+".old-")
	if err != nil {
		return fmt.Errorf("创建 bundle 备份路径失败: %w", err)
	}
	if err := os.Remove(backup); err != nil {
		return fmt.Errorf("准备 bundle 备份路径失败: %w", err)
	}
	if err := renameBundle(target, backup); err != nil {
		return fmt.Errorf("备份旧 bundle 失败: %w", err)
	}
	if err := renameBundle(temporary, target); err != nil {
		if restoreErr := renameBundle(backup, target); restoreErr != nil {
			return fmt.Errorf("安装 bundle 失败且恢复旧 bundle 失败: %w", restoreErr)
		}
		return fmt.Errorf("安装 bundle 失败，已恢复旧 bundle: %w", err)
	}
	if err := syncDirectory(filepath.Dir(target)); err != nil {
		return fmt.Errorf("同步 bundle 父目录失败: %w", err)
	}
	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("清理旧 bundle 备份失败: %w", err)
	}
	if err := syncDirectory(filepath.Dir(target)); err != nil {
		return fmt.Errorf("同步 bundle 清理结果失败: %w", err)
	}
	return nil
}

// renameBundle 为单测注入“安装第二步失败”保留的窄边界；生产使用 os.Rename。
var renameBundle = os.Rename
