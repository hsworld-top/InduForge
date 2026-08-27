package sceneasset

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
)

const (
	ProviderHT        = "ht"
	HTProviderVersion = "5.0"
	PublicProvider    = "induforge"
	PublicVersion     = "1"
	MaxFileSize       = int64(64 << 20)
	SessionTTL        = 20 * time.Minute
	ViewerSessionTTL  = 30 * time.Minute
	ViewerAbsoluteTTL = 8 * time.Hour
)

var (
	ErrNotFound          = errors.New("场景不存在")
	ErrProviderNotSet    = errors.New("场景 Provider 尚未配置")
	ErrProviderBusy      = errors.New("场景或资源库非空，不能切换 Provider")
	ErrInvalidProvider   = errors.New("不支持的场景 Provider")
	ErrInvalidKind       = errors.New("场景类型必须为 2d 或 3d")
	ErrInvalidSceneID    = errors.New("场景 ID 格式无效")
	ErrInvalidSceneName  = errors.New("场景名称不能为空且不能超过 200 个字符")
	ErrSceneNameConflict = errors.New("场景名称已存在")
	ErrInvalidPath       = errors.New("场景文件路径无效")
	ErrProtectedPath     = errors.New("场景文件路径为系统保留路径")
	ErrFileTooLarge      = errors.New("场景文件超过大小限制")
	ErrInvalidJSON       = errors.New("JSON 场景文件格式无效")
	ErrDraftConflict     = errors.New("草稿已被其他编辑会话修改")
	ErrUncommittedDraft  = errors.New("存在未提交的场景草稿")
	ErrSceneNotCommitted = errors.New("场景尚未提交，不能创建 Viewer")
	ErrContractInvalid   = errors.New("场景交互契约无效")
	ErrSessionExpired    = errors.New("编辑会话不存在或已过期")
	ErrSessionForbidden  = errors.New("编辑会话与当前用户或场景不匹配")
	ErrEntryMissing      = errors.New("场景入口文件不存在")
	ErrDependencyMissing = errors.New("场景依赖文件不存在")
	ErrAssetNotFound     = errors.New("场景资源不存在")
	ErrAssetArchived     = errors.New("场景资源已归档")
	ErrInvalidAssetType  = errors.New("场景资源类型无效")
	ErrInvalidAssetName  = errors.New("场景资源名称无效")
	ErrAssetConflict     = errors.New("场景资源已存在")
	ErrAssetNotEditable  = errors.New("仅 Symbol 和 Component 支持在线编辑")
	ErrAssetInUse        = errors.New("场景资源仍被当前草稿引用")
	ErrDuplicateFile     = errors.New("资源包包含重复文件路径")
	ErrInvalidArchive    = errors.New("资源包格式无效")
)

type ProviderState struct {
	Provider        string    `json:"provider"`
	ProviderVersion string    `json:"providerVersion"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type Scene struct {
	ID                    string         `json:"id"`
	TenantID              string         `json:"-"`
	ProjectID             string         `json:"projectId"`
	SceneID               string         `json:"sceneId"`
	Kind                  string         `json:"kind"`
	Name                  string         `json:"name"`
	Provider              string         `json:"-"`
	EntryPath             string         `json:"-"`
	PublicContract        map[string]any `json:"publicContract"`
	ManagedContract       map[string]any `json:"managedContract"`
	DatapointRefs         []string       `json:"datapointRefs"`
	CurrentRevision       int64          `json:"currentRevision"`
	DraftVersion          int64          `json:"draftVersion"`
	CommittedDraftVersion int64          `json:"committedDraftVersion"`
	CreatedAt             time.Time      `json:"createdAt"`
	UpdatedAt             time.Time      `json:"updatedAt"`
}

type SceneInput struct {
	ProjectID      string         `json:"-"`
	SceneID        string         `json:"-"`
	Kind           string         `json:"kind"`
	Name           string         `json:"name"`
	EntryPath      string         `json:"-"`
	PublicContract map[string]any `json:"publicContract"`
}

type ScenePatch struct {
	Name             *string         `json:"name"`
	PublicContract   *map[string]any `json:"publicContract"`
	BaseDraftVersion *int64          `json:"baseDraftVersion"`
}

type FileNode struct {
	Path        string    `json:"path"`
	ParentPath  string    `json:"parentPath"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	ContentHash string    `json:"contentHash,omitempty"`
	Size        int64     `json:"size,omitempty"`
	ContentType string    `json:"contentType,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// DependencyNode 是面向编辑器高级诊断面板的只读投影，不提供文件操作语义。
type DependencyNode struct {
	Path            string `json:"path"`
	Name            string `json:"name"`
	ContentHash     string `json:"contentHash"`
	Size            int64  `json:"size"`
	ContentType     string `json:"contentType"`
	SourceType      string `json:"sourceType"`
	SourceAssetID   string `json:"sourceAssetId,omitempty"`
	SourceAssetName string `json:"sourceAssetName,omitempty"`
	Status          string `json:"status"`
}

type FileContent struct {
	Reader      io.ReadCloser
	Bytes       []byte
	Size        int64
	ContentType string
	Hash        string
}

type EditorSession struct {
	ID        string    `json:"sessionId"`
	UserID    string    `json:"userId"`
	TenantID  string    `json:"tenantId"`
	ProjectID string    `json:"projectId"`
	SceneID   string    `json:"sceneId"`
	SceneName string    `json:"sceneName"`
	Kind      string    `json:"kind"`
	Provider  string    `json:"provider"`
	EntryPath string    `json:"entryPath"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type EditorSessionResponse struct {
	SessionID string    `json:"sessionId"`
	SceneName string    `json:"sceneName"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type CommitInput struct {
	BaseDraftVersion int64 `json:"baseDraftVersion"`
}

type Revision struct {
	ID              string         `json:"-"`
	Revision        int64          `json:"revision"`
	DraftVersion    int64          `json:"draftVersion"`
	Provider        string         `json:"-"`
	ProviderVersion string         `json:"-"`
	EntryPath       string         `json:"-"`
	RootHash        string         `json:"-"`
	PublicContract  map[string]any `json:"-"`
	DatapointRefs   []string       `json:"-"`
	CreatedAt       time.Time      `json:"createdAt"`
}

// ViewerSession 是 revision 固定的只读能力凭证，不携带用户长期令牌。
type ViewerSession struct {
	ID                string         `json:"sessionId"`
	TenantID          string         `json:"tenantId"`
	ProjectID         string         `json:"projectId"`
	SceneDocumentID   string         `json:"sceneDocumentId"`
	SceneID           string         `json:"sceneId"`
	Kind              string         `json:"kind"`
	Provider          string         `json:"provider"`
	RevisionID        string         `json:"revisionId"`
	Revision          int64          `json:"revision"`
	EntryPath         string         `json:"entryPath"`
	Contract          map[string]any `json:"contract"`
	DatapointRefs     []string       `json:"datapointRefs"`
	ExpiresAt         time.Time      `json:"expiresAt"`
	AbsoluteExpiresAt time.Time      `json:"absoluteExpiresAt"`
}

type ViewerSessionResponse struct {
	SessionID     string         `json:"sessionId"`
	URL           string         `json:"url"`
	ExpiresAt     time.Time      `json:"expiresAt"`
	SceneID       string         `json:"sceneId"`
	Kind          string         `json:"kind"`
	Revision      int64          `json:"revision"`
	Contract      map[string]any `json:"contract"`
	DatapointRefs []string       `json:"datapointRefs"`
}

type ViewerBootstrap struct {
	SceneID       string    `json:"sceneId"`
	Kind          string    `json:"kind"`
	Revision      int64     `json:"revision"`
	EntryPath     string    `json:"entryPath"`
	ExpiresAt     time.Time `json:"expiresAt"`
	DatapointRefs []string  `json:"datapointRefs"`
}

type ListFilter struct {
	Kind    string
	Keyword string
	Page    int
	Limit   int
	Sort    string
	Order   string
}

type FileListFilter struct {
	Directory string
	Keyword   string
	Page      int
	Limit     int
	Sort      string
	Order     string
}

type FileWrite struct {
	Path             string
	ContentType      string
	Content          []byte
	BaseDraftVersion int64
}

type AssetType string

const (
	AssetImage     AssetType = "image"
	AssetFont      AssetType = "font"
	AssetModel     AssetType = "model"
	AssetMaterial  AssetType = "material"
	AssetSymbol    AssetType = "symbol"
	AssetComponent AssetType = "component"
)

type SceneAsset struct {
	ID                    string    `json:"assetId"`
	TenantID              string    `json:"-"`
	ProjectID             string    `json:"projectId"`
	Provider              string    `json:"-"`
	Type                  AssetType `json:"type"`
	CompatibleKind        string    `json:"compatibleKind"`
	Name                  string    `json:"name"`
	EntryPath             string    `json:"entryFile"`
	CurrentGeneration     int64     `json:"-"`
	DraftVersion          int64     `json:"draftVersion,omitempty"`
	CommittedDraftVersion int64     `json:"committedDraftVersion,omitempty"`
	Archived              bool      `json:"archived"`
	Bound                 bool      `json:"bound,omitempty"`
	UpdateAvailable       bool      `json:"updateAvailable,omitempty"`
	ThumbnailURL          string    `json:"thumbnailUrl,omitempty"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type AssetGeneration struct {
	ID         string         `json:"-"`
	AssetID    string         `json:"assetId"`
	Generation int64          `json:"-"`
	EntryPath  string         `json:"entryPath"`
	RootHash   string         `json:"rootHash"`
	Manifest   map[string]any `json:"manifest"`
	CreatedAt  time.Time      `json:"createdAt"`
}

type AssetFile struct {
	Path        string
	ContentType string
	Content     []byte
	Hash        string
	ObjectID    string
	ObjectKey   string
	Size        int64
}

type AssetImportInput struct {
	Name        string
	Type        AssetType
	Filename    string
	ContentType string
	Content     []byte
}

type AssetSelectionInput struct {
	Name      string         `json:"name"`
	Type      AssetType      `json:"type"`
	Selection map[string]any `json:"selection"`
}

type AssetListFilter struct {
	Type     AssetType
	Kind     string
	Keyword  string
	Page     int
	Limit    int
	Sort     string
	Order    string
	Archived bool
}

type AssetPatch struct {
	Name string `json:"name"`
}

type AssetAction struct {
	Action           string `json:"action"`
	AssetID          string `json:"assetId"`
	BaseDraftVersion int64  `json:"baseDraftVersion"`
}

type AssetBinding struct {
	ID              string    `json:"bindingId"`
	AssetID         string    `json:"assetId"`
	Name            string    `json:"name"`
	Type            AssetType `json:"type"`
	MountPath       string    `json:"mountPath"`
	UpdateAvailable bool      `json:"updateAvailable"`
}

type AssetEditorSession struct {
	ID        string    `json:"sessionId"`
	UserID    string    `json:"userId"`
	TenantID  string    `json:"tenantId"`
	ProjectID string    `json:"projectId"`
	AssetID   string    `json:"assetId"`
	Provider  string    `json:"provider"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type AssetAnalysis struct {
	EntryPath      string
	CompatibleKind string
	Dependencies   []string
	Manifest       map[string]any
}

type ProjectRepository interface {
	RequireWrite(context.Context, auth.User, string) error
}
