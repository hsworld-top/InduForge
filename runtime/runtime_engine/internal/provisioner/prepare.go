// Package provisioner implements the local, secret-free RuntimeEngine bundle preparation boundary.
package provisioner

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/indu-forge/runtime-engine/internal/binding"
	"github.com/indu-forge/runtime-engine/internal/loader"
	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/provision"
)

const (
	SchemaVersion = "runtime-binding.input.v1"
	SecretRoot    = "/var/run/induforge/secrets"
	WorkRoot      = "/work"
)

type Input struct {
	SchemaVersion         string        `json:"schemaVersion"`
	ReleaseID             string        `json:"releaseId"`
	RuntimeArtifactPath   string        `json:"runtimeArtifactPath"`
	RuntimeArtifactSHA256 string        `json:"runtimeArtifactSha256"`
	ArtifactDir           string        `json:"artifactDir"`
	BundleDir             string        `json:"bundleDir"`
	Binding               binding.Input `json:"binding"`
}

func Decode(raw []byte) (Input, error) {
	var in Input
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&in); err != nil || d.Decode(&struct{}{}) != io.EOF {
		return Input{}, fmt.Errorf("输入格式非法")
	}
	if in.SchemaVersion != SchemaVersion {
		return Input{}, fmt.Errorf("输入版本非法")
	}
	return in, nil
}

// Prepare verifies the release sub-artifact, materializes it under WorkRoot,
// verifies it through the formal loader, then atomically writes the bundle.
func Prepare(in Input) error {
	if err := validateInput(in); err != nil {
		return err
	}
	if in.Binding.ArtifactMountPath != in.ArtifactDir || in.Binding.ArtifactFile != "runtime-project-artifact.json" {
		return fmt.Errorf("项目制品目标与配置不一致")
	}
	f, err := os.Open(in.RuntimeArtifactPath)
	if err != nil {
		return fmt.Errorf("读取运行制品失败")
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return fmt.Errorf("校验运行制品失败")
	}
	if "sha256:"+hex.EncodeToString(h.Sum(nil)) != in.RuntimeArtifactSHA256 {
		return fmt.Errorf("运行制品摘要不匹配")
	}
	if _, err = f.Seek(0, 0); err != nil {
		return fmt.Errorf("读取运行制品失败")
	}
	if err = provision.UnpackRuntimeArtifact(f, in.ArtifactDir, provision.Limits{MaxFiles: 128, MaxFileBytes: 128 << 20, MaxTotalBytes: 512 << 20}); err != nil {
		return fmt.Errorf("安全解包失败")
	}
	artifactPath := filepath.Join(in.ArtifactDir, "runtime-project-artifact.json")
	b, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("读取项目制品失败")
	}
	derived, err := deriveBuildInput(in.Binding, b)
	if err != nil {
		return fmt.Errorf("派生项目制品角色绑定失败: %w", err)
	}
	config, err := binding.BuildEngineConfig(derived)
	if err != nil {
		return fmt.Errorf("配置构造失败: %w", err)
	}
	configRaw, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败")
	}
	check, err := os.CreateTemp(in.ArtifactDir, ".runtime-config-check-")
	if err != nil {
		return fmt.Errorf("创建配置校验文件失败")
	}
	checkPath := check.Name()
	defer os.Remove(checkPath)
	if err := check.Chmod(0o600); err != nil || func() error {
		_, e := check.Write(configRaw)
		if e != nil {
			return e
		}
		return check.Close()
	}() != nil {
		check.Close()
		return fmt.Errorf("写入配置校验文件失败")
	}
	readOnly := false
	_, err = loader.Load(loader.Options{ConfigPath: checkPath, ConfigRoot: in.ArtifactDir, RequireReadOnlyMount: &readOnly})
	if err != nil {
		return fmt.Errorf("项目制品与角色分配不匹配")
	}
	index, err := binding.BuildResolverIndex(in.Binding)
	if err != nil {
		return fmt.Errorf("索引构造失败: %w", err)
	}
	if err := binding.WriteBundle(config, index, in.BundleDir); err != nil {
		return fmt.Errorf("写入 bundle 失败")
	}
	return nil
}

// deriveBuildInput 只以已安全解包的 Artifact 原始 bytes 决定 Artifact ref 与
// producer fencing；部署输入不能覆盖 compute/alarm 的 ID、revision 或 ownership。
func deriveBuildInput(input binding.Input, raw []byte) (binding.BuildInput, error) {
	var artifact model.ProjectArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return binding.BuildInput{}, fmt.Errorf("项目制品 JSON 非法")
	}
	digest := sha256.Sum256(raw)
	build := binding.BuildInput{
		Input: input,
		ProjectArtifact: model.ArtifactRef{
			ArtifactID:       artifact.ProjectID,
			ArtifactRevision: artifactRevision(artifact.ProjectArtifactVersion),
			ArtifactDigest:   "sha256:" + hex.EncodeToString(digest[:]),
		},
		RoleOwnership:   model.Ownership{OwnerID: input.InstanceID, Epoch: 1},
		ManualOwnership: model.Ownership{OwnerID: input.ManualOwner, Epoch: input.ManualEpoch},
	}
	if build.ProjectArtifact.ArtifactRevision < 1 {
		return binding.BuildInput{}, fmt.Errorf("项目制品版本非法")
	}
	if build.ManualOwnership.OwnerID == "" && build.ManualOwnership.Epoch == 0 {
		// 仅保留旧 binding 的本地测试兼容；控制面一旦发布该字段必须原样传递。
		build.ManualOwnership = model.Ownership{OwnerID: "runtime-api", Epoch: 1}
	}
	switch input.Role {
	case "compute":
		for _, unit := range artifact.ComputeUnits {
			if !unit.Enabled {
				continue
			}
			if unit.Revision < 1 {
				return binding.BuildInput{}, fmt.Errorf("compute %q revision 非法", unit.ID)
			}
			build.ComputeProducers = append(build.ComputeProducers, binding.ComputeProducer{ComputeID: unit.ID, Ownership: model.Ownership{OwnerID: input.InstanceID, Epoch: unit.Revision}})
		}
		if len(build.ComputeProducers) == 0 {
			return binding.BuildInput{}, fmt.Errorf("compute 角色缺少可用 producer")
		}
	case "alarm":
		var epoch int64
		for _, item := range artifact.AlarmItems {
			if item.Enabled && item.Revision > epoch {
				epoch = item.Revision
			}
		}
		if epoch < 1 {
			return binding.BuildInput{}, fmt.Errorf("alarm 角色缺少可用 producer")
		}
		build.AlarmOwnership = model.Ownership{OwnerID: input.InstanceID, Epoch: epoch}
	default:
		return binding.BuildInput{}, fmt.Errorf("RuntimeEngine role 必须为 compute 或 alarm")
	}
	return build, nil
}

func artifactRevision(version string) int64 {
	if version == "1.0" {
		return 1
	}
	return 0
}

func validateInput(in Input) error {
	if strings.TrimSpace(in.ReleaseID) == "" || strings.TrimSpace(in.Binding.InstanceID) == "" || !under(in.RuntimeArtifactPath, "/opt/induforge/release") || !under(in.ArtifactDir, WorkRoot) || !under(in.BundleDir, WorkRoot) || !strings.HasPrefix(in.RuntimeArtifactSHA256, "sha256:") {
		return fmt.Errorf("输入路径或 Release 身份非法")
	}
	for _, p := range []string{in.Binding.JetStream.CredentialSecretFile, in.Binding.StateStore.CredentialSecretFile, in.Binding.ComputeSandboxSecretFile} {
		if p != "" && (!strings.HasPrefix(p, "secrets/") || filepath.IsAbs(p) || filepath.Clean(p) != p || strings.Contains(p, "..")) {
			return fmt.Errorf("secret 路径越界")
		}
	}
	return nil
}
func under(path, root string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && (path == root || strings.HasPrefix(path, root+"/"))
}
