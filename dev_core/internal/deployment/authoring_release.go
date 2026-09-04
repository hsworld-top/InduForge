package deployment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/authoringsnapshot"
	"github.com/indu-forge/dev_core/internal/objectstore"
	"github.com/indu-forge/dev_core/internal/releasebuilder"
	"github.com/indu-forge/dev_core/internal/sceneasset"
)

var ErrAuthoringSnapshotBuilderUnavailable = errors.New("开发态快照发布器未配置")

type authoringWorkspace interface {
	ExportAuthoring(string) (map[string]string, map[string]uint32, error)
	MaterializeBuildSnapshot(string, string, map[string]string, map[string]uint32, map[string][]byte) (string, func() error, error)
}

type authoringScenes interface {
	ExportAuthoringSnapshot(context.Context, auth.User, string, sceneasset.AuthoringSnapshotOptions) (sceneasset.AuthoringSnapshot, error)
}

type authoringData interface {
	GetAuthoringSnapshot(context.Context, string, string) (json.RawMessage, error)
	BuildArtifactsFromSnapshot(context.Context, string, string, string, time.Time, json.RawMessage) (json.RawMessage, json.RawMessage, json.RawMessage, error)
}

type authoringObjectStore interface {
	Put(context.Context, string, io.Reader, int64, string) (objectstore.ObjectRef, error)
	Delete(context.Context, string) error
	Open(context.Context, string) (objectstore.ObjectReader, error)
}

type AuthoringSnapshotMetadata struct {
	Bucket, Key, ContentHash, CipherHash, KeyID, ProjectRevision string
	Size                                                         int64
}

func (b *AuthoringReleaseBuilder) PersistBackup(ctx context.Context, project Project, taskID string, captured CapturedAuthoring) (AuthoringSnapshotMetadata, error) {
	key := path.Join("authoring-backups", project.TenantID, project.ID, taskID, captured.Sealed.ContentSHA256+".tar.zst")
	ref, err := b.store.Put(ctx, key, bytes.NewReader(captured.Sealed.Bytes), captured.Sealed.CiphertextSize, "application/vnd.induforge.authoring-snapshot.v1+encrypted")
	if err != nil {
		return AuthoringSnapshotMetadata{}, err
	}
	return AuthoringSnapshotMetadata{Bucket: ref.Bucket, Key: ref.Key, ContentHash: captured.Sealed.ContentSHA256, CipherHash: captured.Sealed.CipherSHA256, Size: captured.Sealed.CiphertextSize, KeyID: captured.KeyID, ProjectRevision: captured.Snapshot.ProjectRevision}, nil
}

func (b *AuthoringReleaseBuilder) OpenStored(ctx context.Context, project Project, metadata AuthoringSnapshotMetadata) (authoringsnapshot.Snapshot, error) {
	if metadata.Bucket == "" || metadata.Size <= 0 || metadata.Size > authoringsnapshot.MaxCiphertextSize {
		return authoringsnapshot.Snapshot{}, fmt.Errorf("开发态快照对象元数据无效")
	}
	object, err := b.store.Open(ctx, metadata.Key)
	if err != nil {
		return authoringsnapshot.Snapshot{}, err
	}
	defer object.Reader.Close()
	if object.Bucket != metadata.Bucket || object.Size != metadata.Size {
		return authoringsnapshot.Snapshot{}, fmt.Errorf("开发态快照对象存储位置或大小不匹配")
	}
	raw, err := io.ReadAll(io.LimitReader(object.Reader, metadata.Size+1))
	if err != nil || int64(len(raw)) != metadata.Size {
		return authoringsnapshot.Snapshot{}, fmt.Errorf("开发态快照对象大小无效")
	}
	key, err := b.keys.Resolve(metadata.KeyID)
	if err != nil {
		return authoringsnapshot.Snapshot{}, err
	}
	return authoringsnapshot.VerifyAndOpen(raw, key, project.TenantID, project.ID, metadata.ProjectRevision, authoringsnapshot.StoredMetadata{ContentSHA256: metadata.ContentHash, CipherSHA256: metadata.CipherHash, CiphertextSize: metadata.Size})
}

type AuthoringReleaseResult struct {
	Source                                                       ReleaseSource
	Bucket, Key, ContentHash, CipherHash, KeyID, ProjectRevision string
	Size                                                         int64
}

// CapturedAuthoring 是短临界区内取得的三域同一逻辑快照。后续耗时构建只允许读取这里的值。
type CapturedAuthoring struct {
	Snapshot authoringsnapshot.Snapshot
	Scenes   sceneasset.AuthoringSnapshot
	Sealed   authoringsnapshot.Sealed
	KeyID    string
}

type AuthoringReleaseBuilder struct {
	workspace authoringWorkspace
	scenes    authoringScenes
	data      authoringData
	store     authoringObjectStore
	keys      *authoringsnapshot.Keyring
	source    *ProjectReleaseSourceBuilder
	now       func() time.Time
}

func NewAuthoringReleaseBuilder(workspace authoringWorkspace, scenes authoringScenes, data authoringData, store authoringObjectStore, keys *authoringsnapshot.Keyring, source *ProjectReleaseSourceBuilder) *AuthoringReleaseBuilder {
	return &AuthoringReleaseBuilder{workspace: workspace, scenes: scenes, data: data, store: store, keys: keys, source: source, now: time.Now}
}

func (b *AuthoringReleaseBuilder) Build(ctx context.Context, actor auth.User, project Project, version Version, authorization string) (AuthoringReleaseResult, error) {
	captured, err := b.Capture(ctx, actor, project)
	if err != nil {
		return AuthoringReleaseResult{}, err
	}
	return b.BuildCaptured(ctx, project, version, captured)
}

func (b *AuthoringReleaseBuilder) Capture(ctx context.Context, actor auth.User, project Project) (CapturedAuthoring, error) {
	if b == nil || b.workspace == nil || b.scenes == nil || b.data == nil || b.store == nil || b.keys == nil || b.source == nil {
		return CapturedAuthoring{}, ErrAuthoringSnapshotBuilderUnavailable
	}
	workspace, modes, err := b.workspace.ExportAuthoring(project.WorkspacePath)
	if err != nil {
		return CapturedAuthoring{}, err
	}
	scenes, err := b.scenes.ExportAuthoringSnapshot(ctx, actor, project.ID, sceneasset.AuthoringSnapshotOptions{ExpectedProjectID: project.ID})
	if err != nil {
		return CapturedAuthoring{}, err
	}
	sceneJSON, err := json.Marshal(scenes)
	if err != nil {
		return CapturedAuthoring{}, err
	}
	dataJSON, err := b.data.GetAuthoringSnapshot(ctx, project.ID, project.TenantID)
	if err != nil {
		return CapturedAuthoring{}, err
	}
	revisionRaw, err := json.Marshal(struct {
		Workspace map[string]string `json:"workspace"`
		Modes     map[string]uint32 `json:"modes"`
		Scenes    json.RawMessage   `json:"scenes"`
		Data      json.RawMessage   `json:"data"`
	}{workspace, modes, sceneJSON, dataJSON})
	if err != nil {
		return CapturedAuthoring{}, err
	}
	revisionSum := sha256.Sum256(revisionRaw)
	revision := "sha256:" + hex.EncodeToString(revisionSum[:])
	snapshot := authoringsnapshot.Snapshot{SchemaVersion: authoringsnapshot.SchemaVersion, TenantID: project.TenantID, ProjectID: project.ID, ProjectRevision: revision, CapturedAt: b.now().UTC(), Workspace: workspace, WorkspaceModes: modes, Scenes: sceneJSON, Data: dataJSON}
	keyID, key, err := b.keys.Current()
	if err != nil {
		return CapturedAuthoring{}, err
	}
	sealed, err := authoringsnapshot.Seal(snapshot, key, nil)
	if err != nil {
		return CapturedAuthoring{}, err
	}
	return CapturedAuthoring{Snapshot: snapshot, Scenes: scenes, Sealed: sealed, KeyID: keyID}, nil
}

func (b *AuthoringReleaseBuilder) BuildCaptured(ctx context.Context, project Project, version Version, captured CapturedAuthoring) (AuthoringReleaseResult, error) {
	return b.buildCaptured(ctx, project, version, captured, true)
}

func (b *AuthoringReleaseBuilder) BuildDevelopmentCaptured(ctx context.Context, project Project, version Version, captured CapturedAuthoring) (AuthoringReleaseResult, error) {
	return b.buildCaptured(ctx, project, version, captured, false)
}

func (b *AuthoringReleaseBuilder) buildCaptured(ctx context.Context, project Project, version Version, captured CapturedAuthoring, persistSnapshot bool) (AuthoringReleaseResult, error) {
	// 先只投影 runtime；只有其权威引擎要求包含 collector 时才构建采集产物。
	runtimeJSON, _, _, err := b.data.BuildArtifactsFromSnapshot(ctx, project.ID, project.TenantID, "", captured.Snapshot.CapturedAt, captured.Snapshot.Data)
	if err != nil {
		return AuthoringReleaseResult{}, err
	}
	var artifact map[string]any
	if json.Unmarshal(runtimeJSON, &artifact) != nil {
		return AuthoringReleaseResult{}, fmt.Errorf("快照运行工件无效")
	}
	var collectorJSON, collectorSource json.RawMessage
	if containsEngine(releasebuilder.DeriveEngineRequirements(artifact), "collector") {
		_, collectorJSON, collectorSource, err = b.data.BuildArtifactsFromSnapshot(ctx, project.ID, project.TenantID, version.ID, captured.Snapshot.CapturedAt, captured.Snapshot.Data)
		if err != nil {
			return AuthoringReleaseResult{}, err
		}
		if len(collectorJSON) == 0 || len(collectorSource) == 0 {
			return AuthoringReleaseResult{}, fmt.Errorf("冻结快照缺少采集运行产物")
		}
	}
	overlays, err := sceneasset.RuntimeFiles(captured.Scenes)
	if err != nil {
		return AuthoringReleaseResult{}, err
	}
	staging, cleanup, err := b.workspace.MaterializeBuildSnapshot(project.ID, version.ID, captured.Snapshot.Workspace, captured.Snapshot.WorkspaceModes, overlays)
	if err != nil {
		return AuthoringReleaseResult{}, err
	}
	defer cleanup()
	frozenProject := project
	frozenProject.WorkspacePath = staging
	source, err := b.source.buildReleaseSourceFromCaptured(ctx, frozenProject, version, artifact, runtimeJSON, collectorJSON, collectorSource)
	if err != nil {
		return AuthoringReleaseResult{}, err
	}
	if !persistSnapshot {
		return AuthoringReleaseResult{Source: source, ContentHash: captured.Sealed.ContentSHA256, CipherHash: captured.Sealed.CipherSHA256, Size: captured.Sealed.CiphertextSize, KeyID: captured.KeyID, ProjectRevision: captured.Snapshot.ProjectRevision}, nil
	}
	// 版本专属 key 不与其他快照共享；在全部构建成功后才上传，避免构建失败留下孤儿对象。
	objectKey := path.Join("authoring-snapshots", project.TenantID, project.ID, version.ID, captured.Sealed.ContentSHA256+".tar.zst")
	ref, err := b.store.Put(ctx, objectKey, bytes.NewReader(captured.Sealed.Bytes), captured.Sealed.CiphertextSize, "application/vnd.induforge.authoring-snapshot.v1+encrypted")
	if err != nil {
		return AuthoringReleaseResult{}, err
	}
	return AuthoringReleaseResult{Source: source, Bucket: ref.Bucket, Key: ref.Key, ContentHash: captured.Sealed.ContentSHA256, CipherHash: captured.Sealed.CipherSHA256, Size: captured.Sealed.CiphertextSize, KeyID: captured.KeyID, ProjectRevision: captured.Snapshot.ProjectRevision}, nil
}

func (b *AuthoringReleaseBuilder) DeleteSnapshot(ctx context.Context, key string) error {
	if b == nil || b.store == nil || key == "" {
		return nil
	}
	return b.store.Delete(ctx, key)
}
