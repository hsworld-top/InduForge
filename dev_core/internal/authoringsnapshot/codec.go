package authoringsnapshot

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/klauspost/compress/zstd"
)

const (
	SchemaVersion           = "authoring-snapshot.v1"
	MaxPlaintextSize  int64 = 128 << 20
	MaxCiphertextSize int64 = 128 << 20
)

var envelopeMagic = []byte("IFAS1\x00")

// Snapshot 是生产版本对应的完整开发态快照。Data 与 Scenes 分别由所属领域
// 生成强类型、已校验的规范 JSON；这里不解释领域字段，避免发布模块复制领域规则。
type Snapshot struct {
	SchemaVersion   string            `json:"schemaVersion"`
	TenantID        string            `json:"tenantId"`
	ProjectID       string            `json:"projectId"`
	ProjectRevision string            `json:"projectRevision"`
	CapturedAt      time.Time         `json:"capturedAt"`
	Workspace       map[string]string `json:"workspace"`
	WorkspaceModes  map[string]uint32 `json:"workspaceModes"`
	Scenes          json.RawMessage   `json:"scenes"`
	Data            json.RawMessage   `json:"data"`
}

type Sealed struct {
	Bytes          []byte
	ContentSHA256  string
	CipherSHA256   string
	PlaintextSize  int64
	CiphertextSize int64
}

type StoredMetadata struct {
	ContentSHA256, CipherSHA256 string
	CiphertextSize              int64
}

// Seal 先规范化并压缩快照，再使用 AES-256-GCM 加密。tenant/project/revision
// 被绑定为附加认证数据，密文被复制到其他版本或工程后无法通过认证。
func Seal(snapshot Snapshot, key []byte, random io.Reader) (Sealed, error) {
	if err := validate(snapshot); err != nil {
		return Sealed{}, err
	}
	if len(key) != 32 {
		return Sealed{}, fmt.Errorf("authoring snapshot key 必须为 32 字节")
	}
	if random == nil {
		random = rand.Reader
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return Sealed{}, fmt.Errorf("编码 authoring snapshot: %w", err)
	}
	if int64(len(raw)) > MaxPlaintextSize {
		return Sealed{}, fmt.Errorf("authoring snapshot 明文超过限制")
	}
	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed, zstd.WithEncoderConcurrency(1), zstd.WithEncoderCRC(false))
	if err != nil {
		return Sealed{}, err
	}
	if _, err = encoder.Write(raw); err != nil {
		encoder.Close()
		return Sealed{}, err
	}
	if err = encoder.Close(); err != nil {
		return Sealed{}, err
	}
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(random, nonce); err != nil {
		return Sealed{}, fmt.Errorf("生成 authoring snapshot nonce: %w", err)
	}
	aad := []byte(fmt.Sprintf("%s\x00%s\x00%s", snapshot.TenantID, snapshot.ProjectID, snapshot.ProjectRevision))
	result := append(append([]byte{}, envelopeMagic...), nonce...)
	result = gcm.Seal(result, nonce, compressed.Bytes(), aad)
	if int64(len(result)) > MaxCiphertextSize {
		return Sealed{}, fmt.Errorf("authoring snapshot 密文超过限制")
	}
	contentHash, cipherHash := sha256.Sum256(raw), sha256.Sum256(result)
	return Sealed{Bytes: result, ContentSHA256: hex.EncodeToString(contentHash[:]), CipherSHA256: hex.EncodeToString(cipherHash[:]), PlaintextSize: int64(len(raw)), CiphertextSize: int64(len(result))}, nil
}

func Open(sealed []byte, key []byte, tenantID, projectID, revision string) (Snapshot, error) {
	if int64(len(sealed)) > MaxCiphertextSize || len(key) != 32 || len(sealed) < len(envelopeMagic)+12 || !bytes.Equal(sealed[:len(envelopeMagic)], envelopeMagic) {
		return Snapshot{}, fmt.Errorf("authoring snapshot 密文无效")
	}
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	offset := len(envelopeMagic)
	nonce := sealed[offset : offset+gcm.NonceSize()]
	aad := []byte(fmt.Sprintf("%s\x00%s\x00%s", tenantID, projectID, revision))
	compressed, err := gcm.Open(nil, nonce, sealed[offset+gcm.NonceSize():], aad)
	if err != nil {
		return Snapshot{}, fmt.Errorf("校验 authoring snapshot: %w", err)
	}
	decoder, err := zstd.NewReader(bytes.NewReader(compressed), zstd.WithDecoderMaxMemory(1<<30))
	if err != nil {
		return Snapshot{}, err
	}
	defer decoder.Close()
	raw, err := io.ReadAll(io.LimitReader(decoder, MaxPlaintextSize+1))
	if err != nil || int64(len(raw)) > MaxPlaintextSize {
		return Snapshot{}, fmt.Errorf("展开 authoring snapshot 失败或超过限制")
	}
	var snapshot Snapshot
	if err = json.Unmarshal(raw, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("解析 authoring snapshot: %w", err)
	}
	if err = validate(snapshot); err != nil {
		return Snapshot{}, err
	}
	if snapshot.TenantID != tenantID || snapshot.ProjectID != projectID || snapshot.ProjectRevision != revision {
		return Snapshot{}, fmt.Errorf("authoring snapshot 归属不匹配")
	}
	return snapshot, nil
}

// VerifyAndOpen 在解密前后同时校验数据库记录的对象大小、密文摘要和明文内容摘要。
// key ID 由调用方用于选择密钥，不进入此函数，避免 codec 隐式访问密钥库。
func VerifyAndOpen(sealed []byte, key []byte, tenantID, projectID, revision string, metadata StoredMetadata) (Snapshot, error) {
	if metadata.CiphertextSize != int64(len(sealed)) {
		return Snapshot{}, fmt.Errorf("authoring snapshot 对象大小不匹配")
	}
	cipherHash := sha256.Sum256(sealed)
	if metadata.CipherSHA256 != hex.EncodeToString(cipherHash[:]) {
		return Snapshot{}, fmt.Errorf("authoring snapshot 密文摘要不匹配")
	}
	snapshot, err := Open(sealed, key, tenantID, projectID, revision)
	if err != nil {
		return Snapshot{}, err
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return Snapshot{}, err
	}
	contentHash := sha256.Sum256(raw)
	if metadata.ContentSHA256 != hex.EncodeToString(contentHash[:]) {
		return Snapshot{}, fmt.Errorf("authoring snapshot 内容摘要不匹配")
	}
	return snapshot, nil
}

func validate(snapshot Snapshot) error {
	if snapshot.SchemaVersion != SchemaVersion || snapshot.TenantID == "" || snapshot.ProjectID == "" || snapshot.ProjectRevision == "" || snapshot.CapturedAt.IsZero() || snapshot.Workspace == nil || snapshot.WorkspaceModes == nil || !jsonObject(snapshot.Scenes) || !jsonObject(snapshot.Data) {
		return fmt.Errorf("authoring snapshot 元数据或领域快照无效")
	}
	if len(snapshot.Workspace) != len(snapshot.WorkspaceModes) {
		return fmt.Errorf("authoring snapshot 工作空间模式不完整")
	}
	for name := range snapshot.Workspace {
		if _, ok := snapshot.WorkspaceModes[name]; !ok {
			return fmt.Errorf("authoring snapshot 工作空间模式不完整")
		}
	}
	return nil
}

func jsonObject(raw json.RawMessage) bool {
	if !json.Valid(raw) {
		return false
	}
	var object map[string]json.RawMessage
	return json.Unmarshal(raw, &object) == nil && object != nil
}
