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
	SchemaVersion           string        `json:"schemaVersion"`
	ReleaseID               string        `json:"releaseId"`
	RuntimeArtifactPath     string        `json:"runtimeArtifactPath"`
	RuntimeArtifactSHA256   string        `json:"runtimeArtifactSha256"`
	CollectorArtifactPath   string        `json:"collectorArtifactPath,omitempty"`
	CollectorArtifactSHA256 string        `json:"collectorArtifactSha256,omitempty"`
	ArtifactDir             string        `json:"artifactDir"`
	BundleDir               string        `json:"bundleDir"`
	Binding                 binding.Input `json:"binding"`
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
	if in.Binding.ArtifactMountPath != "/work/artifact" || in.Binding.ArtifactFile != "runtime-project-artifact.json" {
		return fmt.Errorf("项目制品目标与配置不一致")
	}
	f, err := os.Open(in.RuntimeArtifactPath)
	if err != nil {
		return fmt.Errorf("读取运行制品失败")
	}
	defer f.Close()
	if err = verifyArtifactChecksum(f, in.RuntimeArtifactSHA256); err != nil {
		return fmt.Errorf("运行制品摘要不匹配")
	}
	if _, err = f.Seek(0, 0); err != nil {
		return fmt.Errorf("读取运行制品失败")
	}
	if err = provision.UnpackRuntimeArtifact(f, in.ArtifactDir, provision.Limits{MaxFiles: 128, MaxFileBytes: 128 << 20, MaxTotalBytes: 512 << 20}); err != nil {
		return fmt.Errorf("安全解包失败")
	}
	var collectorRaw []byte
	if in.CollectorArtifactPath != "" {
		cf, openErr := os.Open(in.CollectorArtifactPath)
		if openErr != nil {
			return fmt.Errorf("读取采集制品失败")
		}
		defer cf.Close()
		if copyErr := verifyArtifactChecksum(cf, in.CollectorArtifactSHA256); copyErr != nil {
			return fmt.Errorf("采集制品摘要不匹配")
		}
		if _, seekErr := cf.Seek(0, 0); seekErr != nil {
			return fmt.Errorf("读取采集制品失败")
		}
		collectorDir := filepath.Join(in.ArtifactDir, "collector")
		if unpackErr := provision.UnpackArtifact(cf, collectorDir, "collector-runtime-artifact.json", provision.Limits{MaxFiles: 128, MaxFileBytes: 128 << 20, MaxTotalBytes: 512 << 20}); unpackErr != nil {
			return fmt.Errorf("安全解包采集制品失败")
		}
		collectorRaw, err = os.ReadFile(filepath.Join(collectorDir, "collector-runtime-artifact.json"))
		if err != nil {
			return fmt.Errorf("读取采集制品失败")
		}
	}
	artifactPath := filepath.Join(in.ArtifactDir, "runtime-project-artifact.json")
	b, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("读取项目制品失败")
	}
	derived, err := deriveBuildInput(in.Binding, b, collectorRaw)
	if err != nil {
		return fmt.Errorf("artifact-role-binding: %w", err)
	}
	config, err := binding.BuildEngineConfig(derived)
	if err != nil {
		return fmt.Errorf("engine-config-binding: %w", err)
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
		// 原始 loader 错误可能携带项目路径或制品摘要；日志只保留可行动的校验阶段。
		return fmt.Errorf("runtime-config-loader: %s", loaderErrorClass(err))
	}
	index, err := binding.BuildResolverIndex(in.Binding)
	if err != nil {
		return fmt.Errorf("索引构造失败: %w", err)
	}
	if err := binding.WriteBundle(config, index, in.BundleDir); err != nil {
		return fmt.Errorf("写入 bundle 失败")
	}
	if err := exposeArtifactToSandbox(artifactPath, in.Binding.Role); err != nil {
		return err
	}
	return nil
}

// exposeArtifactToSandbox 仅让 compute Pod 内同组 sandbox 读取已校验工件；其他角色继续 owner-only。
func exposeArtifactToSandbox(path, role string) error {
	if role != "compute" {
		return nil
	}
	if err := os.Chmod(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("设置计算工件目录共享权限失败")
	}
	if err := os.Chmod(path, 0o640); err != nil {
		return fmt.Errorf("设置计算工件共享权限失败")
	}
	return nil
}
func verifyArtifactChecksum(reader io.Reader, expected string) error {
	h := sha256.New()
	if _, err := io.Copy(h, reader); err != nil {
		return err
	}
	if "sha256:"+hex.EncodeToString(h.Sum(nil)) != expected {
		return fmt.Errorf("摘要不匹配")
	}
	return nil
}

func loaderErrorClass(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "runtime-engine-config"):
		return "engine-config-schema"
	case strings.Contains(message, "项目 Artifact 原始字节"):
		return "project-artifact-digest"
	case strings.Contains(message, "runtime-project-artifact"):
		return "project-artifact-schema"
	case strings.Contains(message, "Artifact") || strings.Contains(message, "artifact"):
		return "project-artifact-model"
	default:
		return "engine-config-model"
	}
}

// deriveBuildInput 只以已安全解包的 Artifact 原始 bytes 决定 Artifact ref 与
// producer fencing；部署输入不能覆盖 compute/alarm 的 ID 或 owner。owner
// 由 deployment+role 固定推导，epoch 只接受控制面冻结的服务 generation。
func deriveBuildInput(input binding.Input, raw []byte, collectorRaw ...[]byte) (binding.BuildInput, error) {
	var artifact model.ProjectArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return binding.BuildInput{}, fmt.Errorf("项目制品 JSON 非法")
	}
	digest := sha256.Sum256(raw)
	ownership := model.Ownership{OwnerID: stableRoleOwner(input.DeploymentID, input.Role), Epoch: input.FencingEpoch}
	build := binding.BuildInput{
		Input: input,
		ProjectArtifact: model.ArtifactRef{
			ArtifactID:       artifact.ProjectID,
			ArtifactRevision: artifactRevision(artifact.ProjectArtifactVersion),
			ArtifactDigest:   "sha256:" + hex.EncodeToString(digest[:]),
		},
		RoleOwnership:   ownership,
		ManualOwnership: model.Ownership{OwnerID: input.ManualOwner, Epoch: input.ManualEpoch},
	}
	if build.ProjectArtifact.ArtifactRevision < 1 {
		return binding.BuildInput{}, fmt.Errorf("项目制品版本非法")
	}
	if len(collectorRaw) > 0 && len(collectorRaw[0]) > 0 {
		var collector model.CollectorArtifact
		if json.Unmarshal(collectorRaw[0], &collector) != nil || collector.SchemaVersion != "collector-runtime-artifact.v1" {
			return binding.BuildInput{}, fmt.Errorf("采集制品 JSON 非法")
		}
		digest := sha256.Sum256(collectorRaw[0])
		build.CollectorProducers = []binding.CollectorProducer{{CollectorID: stableCollectorID(input.ProjectID, input.DeploymentID, input.NodeID), Artifact: model.CollectorArtifactBinding{Artifact: model.ArtifactRef{ArtifactID: collector.ArtifactID, ArtifactRevision: collector.ArtifactRevision, ArtifactDigest: "sha256:" + hex.EncodeToString(digest[:])}, ArtifactFile: "collector/collector-runtime-artifact.json"}, Ownership: model.Ownership{OwnerID: stableCollectorOwner(input.DeploymentID, input.NodeID), Epoch: input.FencingEpoch}}}
	}
	if build.ManualOwnership.OwnerID == "" && build.ManualOwnership.Epoch == 0 {
		// 仅保留旧 binding 的本地测试兼容；控制面一旦发布该字段必须原样传递。
		build.ManualOwnership = model.Ownership{OwnerID: "runtime-api", Epoch: 1}
	}
	// 所有 DERIVED 消费者都从同一项目制品派生 compute producer 身份；身份固定为
	// compute 角色而非当前消费者角色，避免 writer/alarm 误接受或误拒绝派生事实。
	for _, unit := range artifact.ComputeUnits {
		if !unit.Enabled {
			continue
		}
		if unit.Revision < 1 {
			return binding.BuildInput{}, fmt.Errorf("compute %q revision 非法", unit.ID)
		}
		build.ComputeProducers = append(build.ComputeProducers, binding.ComputeProducer{ComputeID: unit.ID, Ownership: model.Ownership{OwnerID: stableRoleOwner(input.DeploymentID, "compute"), Epoch: input.FencingEpoch}})
	}
	switch input.Role {
	case "writer":
	case "compute":
		if len(build.ComputeProducers) == 0 {
			return binding.BuildInput{}, fmt.Errorf("compute 角色缺少可用 producer")
		}
	case "alarm":
		enabled := false
		for _, item := range artifact.AlarmItems {
			if item.Enabled {
				enabled = true
			}
		}
		if !enabled {
			return binding.BuildInput{}, fmt.Errorf("alarm 角色缺少可用 producer")
		}
		build.AlarmOwnership = ownership
	default:
		return binding.BuildInput{}, fmt.Errorf("RuntimeEngine role 必须为 writer、compute 或 alarm")
	}
	return build, nil
}

func stableRoleOwner(deploymentID, role string) string {
	sum := sha256.Sum256([]byte(deploymentID + "\x00" + role))
	return "if-" + role + "-" + hex.EncodeToString(sum[:])[:16]
}
func stableCollectorID(projectID, deploymentID, nodeID string) string {
	return stableCollectorValue("collector", projectID, deploymentID, nodeID)
}
func stableCollectorOwner(deploymentID, nodeID string) string {
	return stableCollectorValue("collector-owner", deploymentID, nodeID)
}
func stableCollectorValue(prefix string, values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "\x1f")))
	return prefix + "-" + hex.EncodeToString(sum[:20])
}

func artifactRevision(version string) int64 {
	if version == "1.0" {
		return 1
	}
	return 0
}

func validateInput(in Input) error {
	if strings.TrimSpace(in.ReleaseID) == "" || strings.TrimSpace(in.Binding.InstanceID) == "" || in.Binding.FencingEpoch < 1 || !under(in.RuntimeArtifactPath, "/opt/induforge/release") || !under(in.ArtifactDir, WorkRoot) || !under(in.BundleDir, WorkRoot) || !validSHA256(in.RuntimeArtifactSHA256) || (in.CollectorArtifactPath != "" && (!under(in.CollectorArtifactPath, "/opt/induforge/release") || !validSHA256(in.CollectorArtifactSHA256))) {
		return fmt.Errorf("输入路径或 Release 身份非法")
	}
	for _, p := range []string{in.Binding.JetStream.CredentialSecretFile, in.Binding.StateStore.CredentialSecretFile, in.Binding.ComputeSandboxSecretFile} {
		if p != "" && (!strings.HasPrefix(p, "secrets/") || filepath.IsAbs(p) || filepath.Clean(p) != p || strings.Contains(p, "..")) {
			return fmt.Errorf("secret 路径越界")
		}
	}
	return nil
}
func validSHA256(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, r := range value[7:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}
func under(path, root string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && (path == root || strings.HasPrefix(path, root+"/"))
}
