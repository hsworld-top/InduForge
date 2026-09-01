package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/indu-forge/data_service/internal/repository"
	collectorsecurity "github.com/indu-forge/data_service/internal/security"
)

const (
	collectorBundleMaxFileBytes   = 16 << 20
	collectorBundleMaxSecretBytes = 1 << 20
	collectorBundleMaxTotalBytes  = 32 << 20
)

// ErrCollectorNotRequired 表示快照没有启用的采集连接，调用方无需安装采集器 bundle。
var ErrCollectorNotRequired = errors.New("工程不需要采集器运行 bundle")

// TenantProjectSnapshot 是已经过项目-租户绑定鉴权的快照输入。构建器不访问数据库，
// 因而调用方必须只传入同一租户边界内的权威快照。
type TenantProjectSnapshot struct {
	TenantID  string
	ProjectID string
	Snapshot  *repository.ProjectSnapshot
}

// DeploymentBindingInput 是 control-plane 已冻结的部署身份与站点引用。sourceSnapshot
// 必须是 data_service 构建 collector artifact 时返回的完整快照，避免用已变化的工程
// 定义装配旧 Release。
type DeploymentBindingInput struct {
	DeploymentID            string
	EnvironmentID           string
	NodeID                  string
	ReleaseID               string
	AccountID               string
	BindingRevision         int64
	OwnershipEpoch          int64
	NATSServerResourceRef   string
	NATSCredentialSecretRef string
	NATSEndpoint            string
	SourceSnapshot          json.RawMessage
}

// CollectorBindingBundle 是 NodeAgent 可写入安全目录的纯数据结果。Binding 与 Index
// 永远不含密钥值；SecretFiles 的 key 是 index 使用的规范相对路径。
type CollectorBindingBundle struct {
	Binding       json.RawMessage
	Index         json.RawMessage
	SecretFiles   map[string][]byte
	BindingSHA256 string
	BindingSize   int
	IndexSHA256   string
	IndexSize     int
}

// CollectorBindingBundleBuilder 只负责把已验证快照投影为 collector binding bundle。
// 它不接入 HTTP、数据库或站点 Secret 存储，便于发布流程重放和离线校验。
type CollectorBindingBundleBuilder struct {
	cipher *collectorsecurity.CollectorSecretCipher
}

func NewCollectorBindingBundleBuilder(cipher *collectorsecurity.CollectorSecretCipher) (*CollectorBindingBundleBuilder, error) {
	if cipher == nil {
		return nil, fmt.Errorf("采集连接密钥解密器不能为空")
	}
	return &CollectorBindingBundleBuilder{cipher: cipher}, nil
}

func (b *CollectorBindingBundleBuilder) Build(snapshot TenantProjectSnapshot, input DeploymentBindingInput) (*CollectorBindingBundle, error) {
	if b == nil || b.cipher == nil {
		return nil, fmt.Errorf("采集器 bundle 构建器未初始化")
	}
	if err := validateBundleInput(snapshot, input); err != nil {
		return nil, err
	}
	artifact, err := validateArtifactSnapshot(snapshot, input.SourceSnapshot)
	if err != nil {
		return nil, err
	}

	connections, resources, secretFiles, err := b.connectionBundle(snapshot, input, artifact.ConnectionIDs)
	if err != nil {
		return nil, err
	}
	if len(connections) == 0 {
		return nil, ErrCollectorNotRequired
	}
	bindingID := derivedStableID("collector-binding", snapshot.TenantID, snapshot.ProjectID, input.DeploymentID, input.EnvironmentID, input.NodeID, input.ReleaseID)
	collectorID := derivedStableID("collector", snapshot.ProjectID, input.DeploymentID, input.NodeID)
	ownerID := derivedStableID("collector-owner", input.DeploymentID, input.NodeID)
	resources[input.NATSServerResourceRef] = mustRaw(map[string]string{"url": input.NATSEndpoint, "accountId": input.AccountID})
	index := struct {
		SchemaVersion string                     `json:"schemaVersion"`
		Resources     map[string]json.RawMessage `json:"resources"`
		Secrets       map[string]string          `json:"secrets"`
	}{"collector-runtime-index.v1", resources, secretIndex(secretFiles)}
	// NATS 文件由 NodeAgent 的站点凭据装配；构建器只发布 ref，绝不接收或回显 token。
	index.Secrets[input.NATSCredentialSecretRef] = "nats-credential.json"
	binding := collectorBindingDocument{
		SchemaVersion: "collector-runtime-binding.v1", BindingID: bindingID, BindingRevision: input.BindingRevision,
		DeploymentID: input.DeploymentID, AccountID: input.AccountID,
		Artifact: artifact.Ref, CollectorID: collectorID, Ownership: ownership{OwnerID: ownerID, Epoch: input.OwnershipEpoch},
		Connections: connections,
		NATS:        natsBinding{ServerResourceRef: input.NATSServerResourceRef, CredentialSecretRef: input.NATSCredentialSecretRef, RawSubjectPrefix: "data.raw"},
		WALCapacity: walCapacity{MaxBytes: 1 << 30, HighWatermarkBytes: 768 << 20, DiagnosticReserveBytes: 1 << 20, DataGapPolicy: "emit-alarm-event"},
	}
	bindingRaw, err := json.Marshal(binding)
	if err != nil {
		return nil, fmt.Errorf("序列化 collector binding 失败")
	}
	indexRaw, err := json.Marshal(index)
	if err != nil {
		return nil, fmt.Errorf("序列化 collector resolver index 失败")
	}
	if err := validateBundleSize(bindingRaw, indexRaw, secretFiles); err != nil {
		return nil, err
	}
	return &CollectorBindingBundle{Binding: bindingRaw, Index: indexRaw, SecretFiles: secretFiles, BindingSHA256: sha256Text(bindingRaw), BindingSize: len(bindingRaw), IndexSHA256: sha256Text(indexRaw), IndexSize: len(indexRaw)}, nil
}

type artifactReference struct {
	ArtifactID       string `json:"artifactId"`
	ArtifactRevision int64  `json:"artifactRevision"`
	ArtifactDigest   string `json:"artifactDigest"`
}
type ownership struct {
	OwnerID string `json:"ownerId"`
	Epoch   int64  `json:"epoch"`
}
type bindingConnection struct {
	ConnectionID string   `json:"connectionId"`
	ResourceRef  string   `json:"resourceRef"`
	SecretRefs   []string `json:"secretRefs,omitempty"`
}
type natsBinding struct {
	ServerResourceRef   string `json:"serverResourceRef"`
	CredentialSecretRef string `json:"credentialSecretRef"`
	RawSubjectPrefix    string `json:"rawSubjectPrefix"`
}
type walCapacity struct {
	MaxBytes               int64  `json:"maxBytes"`
	HighWatermarkBytes     int64  `json:"highWatermarkBytes"`
	DiagnosticReserveBytes int64  `json:"diagnosticReserveBytes"`
	DataGapPolicy          string `json:"dataGapPolicy"`
}
type collectorBindingDocument struct {
	SchemaVersion   string              `json:"schemaVersion"`
	BindingID       string              `json:"bindingId"`
	BindingRevision int64               `json:"bindingRevision"`
	DeploymentID    string              `json:"deploymentId"`
	AccountID       string              `json:"accountId"`
	Artifact        artifactReference   `json:"artifact"`
	CollectorID     string              `json:"collectorId"`
	Ownership       ownership           `json:"ownership"`
	Connections     []bindingConnection `json:"connections"`
	NATS            natsBinding         `json:"nats"`
	WALCapacity     walCapacity         `json:"walCapacity"`
}

type checkedArtifact struct {
	Ref           artifactReference
	ConnectionIDs map[string]struct{}
}

func validateBundleInput(s TenantProjectSnapshot, in DeploymentBindingInput) error {
	if s.Snapshot == nil || !canonicalUUID(s.ProjectID) || !validStableID(s.TenantID) || in.BindingRevision < 1 || in.OwnershipEpoch < 1 || len(in.SourceSnapshot) == 0 {
		return fmt.Errorf("采集器 bundle 输入无效")
	}
	for _, value := range []string{in.DeploymentID, in.EnvironmentID, in.NodeID, in.ReleaseID, in.AccountID} {
		if !validStableID(value) {
			return fmt.Errorf("采集器 bundle 部署身份无效")
		}
	}
	if !validRef(in.NATSServerResourceRef, "site-resource://") || !validRef(in.NATSCredentialSecretRef, "secret://") {
		return fmt.Errorf("采集器 bundle NATS 引用无效")
	}
	parsedNATS, err := url.Parse(in.NATSEndpoint)
	if err != nil || (parsedNATS.Scheme != "nats" && parsedNATS.Scheme != "tls") || parsedNATS.Host == "" || parsedNATS.User != nil || parsedNATS.RawQuery != "" || parsedNATS.Fragment != "" {
		return fmt.Errorf("采集器 bundle NATS endpoint 无效")
	}
	return nil
}

func validateArtifactSnapshot(s TenantProjectSnapshot, raw []byte) (checkedArtifact, error) {
	var source struct {
		SchemaVersion    string          `json:"schemaVersion"`
		ProjectID        string          `json:"projectId"`
		ArtifactRevision int64           `json:"artifactRevision"`
		SHA256           string          `json:"sha256"`
		Size             int             `json:"size"`
		Artifact         json.RawMessage `json:"artifact"`
	}
	if strictUnmarshal(raw, &source) != nil || source.SchemaVersion != "collector-runtime-artifact.v1" || source.ProjectID != s.ProjectID || source.ArtifactRevision < 1 || len(source.Artifact) == 0 || source.Size != len(source.Artifact) || source.SHA256 != sha256Text(source.Artifact) {
		return checkedArtifact{}, fmt.Errorf("采集器 sourceSnapshot 无效或已变化")
	}
	var artifact struct {
		SchemaVersion    string            `json:"schemaVersion"`
		ArtifactID       string            `json:"artifactId"`
		ArtifactRevision int64             `json:"artifactRevision"`
		ProjectID        string            `json:"projectId"`
		CollectorVersion string            `json:"collectorVersion"`
		Connections      []json.RawMessage `json:"connections"`
		PointMappings    json.RawMessage   `json:"pointMappings"`
		WAL              json.RawMessage   `json:"wal"`
	}
	if strictUnmarshal(source.Artifact, &artifact) != nil || artifact.SchemaVersion != "collector-runtime-artifact.v1" || !validStableID(artifact.ArtifactID) || artifact.ArtifactRevision != source.ArtifactRevision || artifact.ProjectID != s.ProjectID {
		return checkedArtifact{}, fmt.Errorf("采集器 sourceSnapshot artifact 无效")
	}
	expected := map[string]struct{}{}
	for _, c := range s.Snapshot.CollectorConnections {
		if c.IsEnabled {
			if !canonicalUUID(c.ID) {
				return checkedArtifact{}, fmt.Errorf("快照采集连接 ID 无效")
			}
			expected[c.ID] = struct{}{}
		}
	}
	if len(expected) == 0 {
		return checkedArtifact{ConnectionIDs: expected}, nil
	}
	got := map[string]struct{}{}
	for _, rawConnection := range artifact.Connections {
		var c struct {
			ConnectionID       string          `json:"connectionId"`
			ProtocolFamily     string          `json:"protocolFamily"`
			DriverID           string          `json:"driverId"`
			DriverVersion      string          `json:"driverVersion"`
			SchemaVersion      int             `json:"schemaVersion"`
			Enabled            bool            `json:"enabled"`
			RequiredOperations json.RawMessage `json:"requiredOperations"`
			DefaultAcquisition json.RawMessage `json:"defaultAcquisition"`
		}
		if strictUnmarshal(rawConnection, &c) != nil {
			return checkedArtifact{}, fmt.Errorf("采集器 sourceSnapshot artifact 连接无效")
		}
		if c.Enabled {
			if !canonicalUUID(c.ConnectionID) {
				return checkedArtifact{}, fmt.Errorf("采集器 artifact 连接 ID 无效")
			}
			got[c.ConnectionID] = struct{}{}
		}
	}
	if len(got) != len(expected) {
		return checkedArtifact{}, fmt.Errorf("采集器 sourceSnapshot 与权威快照不一致")
	}
	for id := range expected {
		if _, ok := got[id]; !ok {
			return checkedArtifact{}, fmt.Errorf("采集器 sourceSnapshot 与权威快照不一致")
		}
	}
	return checkedArtifact{Ref: artifactReference{artifact.ArtifactID, artifact.ArtifactRevision, sha256Text(source.Artifact)}, ConnectionIDs: got}, nil
}

func (b *CollectorBindingBundleBuilder) connectionBundle(s TenantProjectSnapshot, in DeploymentBindingInput, ids map[string]struct{}) ([]bindingConnection, map[string]json.RawMessage, map[string][]byte, error) {
	byID := map[string]repository.SnapshotCollectorConnectionRecord{}
	for _, c := range s.Snapshot.CollectorConnections {
		if c.IsEnabled {
			byID[c.ID] = c
		}
	}
	secretByConnection := map[string][]repository.SnapshotCollectorSecretRecord{}
	for _, secret := range s.Snapshot.CollectorSecrets {
		if _, ok := ids[secret.ConnectionID]; !ok {
			return nil, nil, nil, fmt.Errorf("采集连接密钥引用未知连接")
		}
		secretByConnection[secret.ConnectionID] = append(secretByConnection[secret.ConnectionID], secret)
	}
	keys := make([]string, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	connections := make([]bindingConnection, 0, len(keys))
	resources := map[string]json.RawMessage{}
	files := map[string][]byte{}
	for _, id := range keys {
		c, ok := byID[id]
		if !ok {
			return nil, nil, nil, fmt.Errorf("采集器 artifact 连接不在权威快照中")
		}
		resource, err := connectionResource(c)
		if err != nil {
			return nil, nil, nil, err
		}
		resourceRef := "site-resource://collector/" + id
		resources[resourceRef] = mustRaw(resource)
		item := bindingConnection{ConnectionID: id, ResourceRef: resourceRef}
		if records := secretByConnection[id]; len(records) > 0 {
			file, err := b.decryptConnectionSecrets(id, records)
			if err != nil {
				return nil, nil, nil, err
			}
			name := "connection-" + id + ".json"
			files[name] = file
			item.SecretRefs = []string{"secret://collector/" + id}
		}
		connections = append(connections, item)
	}
	return connections, resources, files, nil
}

func (b *CollectorBindingBundleBuilder) decryptConnectionSecrets(connectionID string, records []repository.SnapshotCollectorSecretRecord) ([]byte, error) {
	values := map[string]json.RawMessage{}
	for _, record := range records {
		if record.EncryptionKeyVersion != b.cipher.KeyVersion() || !validSecretKey(record.SecretKey) {
			return nil, fmt.Errorf("采集连接 %s 密钥元数据无效", connectionID)
		}
		if _, exists := values[record.SecretKey]; exists {
			return nil, fmt.Errorf("采集连接 %s 密钥重复", connectionID)
		}
		plain, err := b.cipher.Decrypt(record.EncryptedValue)
		if err != nil {
			return nil, fmt.Errorf("采集连接 %s 密钥解密失败", connectionID)
		}
		if len(plain) > collectorBundleMaxSecretBytes {
			return nil, fmt.Errorf("采集连接 %s 密钥过大", connectionID)
		}
		if json.Valid(plain) {
			values[record.SecretKey] = append(json.RawMessage(nil), plain...)
		} else {
			values[record.SecretKey] = mustRaw(string(plain))
		}
	}
	raw, err := json.Marshal(values)
	if err != nil || len(raw) > collectorBundleMaxSecretBytes {
		return nil, fmt.Errorf("采集连接 %s 密钥文件无效或过大", connectionID)
	}
	return raw, nil
}

func connectionResource(c repository.SnapshotCollectorConnectionRecord) (any, error) {
	host, ok := c.Config["host"].(string)
	if !ok || strings.TrimSpace(host) == "" {
		return nil, fmt.Errorf("采集连接 %s 缺少 host", c.ID)
	}
	port, ok := integer(c.Config["port"])
	if !ok || port < 1 || port > 65535 {
		return nil, fmt.Errorf("采集连接 %s 缺少有效 port", c.ID)
	}
	switch c.DriverID {
	case "modbus.tcp":
		return map[string]any{"host": host, "port": port}, nil
	case "opcua.standard":
		path, ok := c.Config["endpointPath"].(string)
		if !ok || strings.ContainsAny(path, "?#") {
			return nil, fmt.Errorf("采集连接 %s OPC UA endpointPath 无效", c.ID)
		}
		if c.Config["securityMode"] != "None" || c.Config["securityPolicy"] != "None" || c.Config["authenticationType"] != "anonymous" {
			return nil, fmt.Errorf("采集连接 %s OPC UA 安全模式不受当前运行时支持", c.ID)
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		endpoint := (&url.URL{Scheme: "opc.tcp", Host: net.JoinHostPort(host, fmt.Sprint(port)), Path: path}).String()
		return map[string]string{"url": endpoint}, nil
	default:
		return nil, fmt.Errorf("采集连接 %s 使用未支持的运行驱动", c.ID)
	}
}

func secretIndex(files map[string][]byte) map[string]string {
	out := map[string]string{}
	for file := range files {
		id := strings.TrimSuffix(strings.TrimPrefix(file, "connection-"), ".json")
		out["secret://collector/"+id] = file
	}
	return out
}
func validateBundleSize(binding, index []byte, files map[string][]byte) error {
	total := len(binding) + len(index)
	if len(binding) > collectorBundleMaxFileBytes || len(index) > collectorBundleMaxFileBytes {
		return fmt.Errorf("采集器 bundle 文件超过限制")
	}
	for name, v := range files {
		if len(v) > collectorBundleMaxSecretBytes {
			return fmt.Errorf("采集器 secret 文件 %s 超过限制", name)
		}
		total += len(v)
	}
	if total > collectorBundleMaxTotalBytes {
		return fmt.Errorf("采集器 bundle 总量超过限制")
	}
	return nil
}
func mustRaw(value any) json.RawMessage { raw, _ := json.Marshal(value); return raw }
func sha256Text(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func derivedStableID(prefix string, values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "\x1f")))
	return prefix + "-" + hex.EncodeToString(sum[:20])
}

var stableIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var secretKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func validStableID(s string) bool  { return stableIDPattern.MatchString(s) }
func validSecretKey(s string) bool { return secretKeyPattern.MatchString(s) }
func canonicalUUID(s string) bool {
	parsed, err := uuid.Parse(s)
	return err == nil && parsed.String() == s
}
func validRef(s, prefix string) bool {
	if !strings.HasPrefix(s, prefix) {
		return false
	}
	tail := strings.TrimPrefix(s, prefix)
	if len(tail) == 0 || len(tail) > 256 || !((tail[0] >= 'A' && tail[0] <= 'Z') || (tail[0] >= 'a' && tail[0] <= 'z') || (tail[0] >= '0' && tail[0] <= '9')) {
		return false
	}
	for _, r := range tail[1:] {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || strings.ContainsRune("._:/@-", r)) {
			return false
		}
	}
	return true
}
func integer(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		if n >= -1<<31 && n <= 1<<31-1 {
			return int(n), true
		}
	case float64:
		if n == float64(int(n)) {
			return int(n), true
		}
	case json.Number:
		parsed, e := n.Int64()
		if e == nil && parsed >= -1<<31 && parsed <= 1<<31-1 {
			return int(parsed), true
		}
	}
	return 0, false
}
func strictUnmarshal(raw []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("JSON 存在额外内容")
	}
	return nil
}
