package sceneasset

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/jackc/pgx/v5"
)

// AuthoringSnapshot 只保存项目当前可编辑态和恢复当前提交语义所需的 revision，历史版本不属于该契约。
type AuthoringSnapshot struct {
	ProjectID string                     `json:"projectId"`
	Fence     string                     `json:"fence,omitempty"`
	Scenes    []AuthoringSceneSnapshot   `json:"scenes"`
	Assets    []AuthoringAssetSnapshot   `json:"assets"`
	Bindings  []AuthoringBindingSnapshot `json:"bindings"`
}

type AuthoringFileSnapshot struct {
	Path        string `json:"path"`
	Role        string `json:"role,omitempty"`
	ContentType string `json:"contentType"`
	Hash        string `json:"hash"`
	Content     []byte `json:"content"`
}

type AuthoringSceneSnapshot struct {
	Scene         AuthoringScene          `json:"scene"`
	DraftFiles    []AuthoringFileSnapshot `json:"draftFiles"`
	Revision      *AuthoringRevision      `json:"revision,omitempty"`
	RevisionFiles []AuthoringFileSnapshot `json:"revisionFiles,omitempty"`
}

type AuthoringAssetSnapshot struct {
	Asset           AuthoringAsset            `json:"asset"`
	Generation      *AuthoringAssetGeneration `json:"generation,omitempty"`
	GenerationFiles []AuthoringFileSnapshot   `json:"generationFiles"`
	DraftFiles      []AuthoringFileSnapshot   `json:"draftFiles"`
}

type AuthoringScene struct {
	ID                    string         `json:"id"`
	ProjectID             string         `json:"projectId"`
	SceneID               string         `json:"sceneId"`
	Kind                  string         `json:"kind"`
	Name                  string         `json:"name"`
	Provider              string         `json:"provider"`
	EntryPath             string         `json:"entryPath"`
	PublicContract        map[string]any `json:"publicContract"`
	ManagedContract       map[string]any `json:"managedContract"`
	DatapointRefs         []string       `json:"datapointRefs"`
	CurrentRevision       int64          `json:"currentRevision"`
	DraftVersion          int64          `json:"draftVersion"`
	CommittedDraftVersion int64          `json:"committedDraftVersion"`
}

type AuthoringRevision struct {
	ID              string         `json:"id"`
	Revision        int64          `json:"revision"`
	DraftVersion    int64          `json:"draftVersion"`
	Provider        string         `json:"provider"`
	ProviderVersion string         `json:"providerVersion"`
	EntryPath       string         `json:"entryPath"`
	RootHash        string         `json:"rootHash"`
	PublicContract  map[string]any `json:"publicContract"`
	DatapointRefs   []string       `json:"datapointRefs"`
}

type AuthoringAsset struct {
	ID                    string    `json:"id"`
	ProjectID             string    `json:"projectId"`
	Provider              string    `json:"provider"`
	Type                  AssetType `json:"type"`
	CompatibleKind        string    `json:"compatibleKind"`
	Name                  string    `json:"name"`
	EntryPath             string    `json:"entryPath"`
	CurrentGeneration     int64     `json:"currentGeneration"`
	DraftVersion          int64     `json:"draftVersion"`
	CommittedDraftVersion int64     `json:"committedDraftVersion"`
}

type AuthoringAssetGeneration struct {
	ID         string         `json:"id"`
	AssetID    string         `json:"assetId"`
	Generation int64          `json:"generation"`
	EntryPath  string         `json:"entryPath"`
	RootHash   string         `json:"rootHash"`
	Manifest   map[string]any `json:"manifest"`
}

func snapshotScene(scene Scene) AuthoringScene {
	return AuthoringScene{ID: scene.ID, ProjectID: scene.ProjectID, SceneID: scene.SceneID, Kind: scene.Kind, Name: scene.Name, Provider: scene.Provider, EntryPath: scene.EntryPath, PublicContract: scene.PublicContract, ManagedContract: scene.ManagedContract, DatapointRefs: scene.DatapointRefs, CurrentRevision: scene.CurrentRevision, DraftVersion: scene.DraftVersion, CommittedDraftVersion: scene.CommittedDraftVersion}
}
func storedScene(scene AuthoringScene, tenantID string) Scene {
	return Scene{ID: scene.ID, TenantID: tenantID, ProjectID: scene.ProjectID, SceneID: scene.SceneID, Kind: scene.Kind, Name: scene.Name, Provider: scene.Provider, EntryPath: scene.EntryPath, PublicContract: scene.PublicContract, ManagedContract: scene.ManagedContract, DatapointRefs: scene.DatapointRefs, CurrentRevision: scene.CurrentRevision, DraftVersion: scene.DraftVersion, CommittedDraftVersion: scene.CommittedDraftVersion}
}
func snapshotRevision(revision Revision) AuthoringRevision {
	return AuthoringRevision{ID: revision.ID, Revision: revision.Revision, DraftVersion: revision.DraftVersion, Provider: revision.Provider, ProviderVersion: revision.ProviderVersion, EntryPath: revision.EntryPath, RootHash: revision.RootHash, PublicContract: revision.PublicContract, DatapointRefs: revision.DatapointRefs}
}
func snapshotAsset(asset SceneAsset) AuthoringAsset {
	return AuthoringAsset{ID: asset.ID, ProjectID: asset.ProjectID, Provider: asset.Provider, Type: asset.Type, CompatibleKind: asset.CompatibleKind, Name: asset.Name, EntryPath: asset.EntryPath, CurrentGeneration: asset.CurrentGeneration, DraftVersion: asset.DraftVersion, CommittedDraftVersion: asset.CommittedDraftVersion}
}
func storedAsset(asset AuthoringAsset, tenantID string) SceneAsset {
	return SceneAsset{ID: asset.ID, TenantID: tenantID, ProjectID: asset.ProjectID, Provider: asset.Provider, Type: asset.Type, CompatibleKind: asset.CompatibleKind, Name: asset.Name, EntryPath: asset.EntryPath, CurrentGeneration: asset.CurrentGeneration, DraftVersion: asset.DraftVersion, CommittedDraftVersion: asset.CommittedDraftVersion}
}
func snapshotGeneration(g AssetGeneration) AuthoringAssetGeneration {
	return AuthoringAssetGeneration{ID: g.ID, AssetID: g.AssetID, Generation: g.Generation, EntryPath: g.EntryPath, RootHash: g.RootHash, Manifest: g.Manifest}
}

type AuthoringBindingSnapshot struct {
	ID              string `json:"id"`
	SceneDocumentID string `json:"sceneDocumentId"`
	AssetID         string `json:"assetId"`
	Generation      int64  `json:"generation"`
	MountPath       string `json:"mountPath"`
}

type AuthoringSnapshotOptions struct {
	ExpectedProjectID string
	Fence             string
}

// RuntimeFiles 从不可变场景快照投影发布运行文件，发布构建不得重新读取活动场景表。
func RuntimeFiles(snapshot AuthoringSnapshot) (map[string][]byte, error) {
	if err := validateAuthoringSnapshot(snapshot); err != nil {
		return nil, err
	}
	result := map[string][]byte{}
	for _, scene := range snapshot.Scenes {
		if scene.Revision == nil {
			continue
		}
		for _, file := range scene.RevisionFiles {
			result[file.Path] = append([]byte(nil), file.Content...)
		}
	}
	assets := map[string]AuthoringAssetSnapshot{}
	for _, asset := range snapshot.Assets {
		assets[asset.Asset.ID] = asset
	}
	for _, binding := range snapshot.Bindings {
		asset, ok := assets[binding.AssetID]
		if !ok || asset.Generation == nil || asset.Generation.Generation != binding.Generation {
			return nil, ErrInvalidArchive
		}
		for _, file := range asset.GenerationFiles {
			name := path.Join(binding.MountPath, file.Path)
			if previous, exists := result[name]; exists && !bytes.Equal(previous, file.Content) {
				return nil, ErrInvalidArchive
			}
			result[name] = append([]byte(nil), file.Content...)
		}
	}
	return result, nil
}

func validateAuthoringSnapshotProject(projectID string, options AuthoringSnapshotOptions) error {
	if projectID == "" || (options.ExpectedProjectID != "" && options.ExpectedProjectID != projectID) {
		return fmt.Errorf("场景可编辑态快照工程归属不匹配")
	}
	return nil
}

// ExportAuthoringSnapshot 导出项目当前态；对象存储文件会被读取为 bytes，不泄漏 object_key。
func (s *Service) ExportAuthoringSnapshot(ctx context.Context, actor auth.User, projectID string, options AuthoringSnapshotOptions) (AuthoringSnapshot, error) {
	if err := validateAuthoringSnapshotProject(projectID, options); err != nil {
		return AuthoringSnapshot{}, err
	}
	if err := s.requireRead(ctx, actor, projectID); err != nil {
		return AuthoringSnapshot{}, err
	}
	snapshot, objects, err := s.repository.exportAuthoringSnapshot(ctx, actor.TenantID, projectID)
	if err != nil {
		return AuthoringSnapshot{}, err
	}
	snapshot.Fence = options.Fence
	objectByKey := make(map[string]contentObject, len(objects))
	for _, target := range objects {
		objectByKey[target.provider+"\x00"+target.object.Hash] = target.object
	}
	hydrate := func(provider string, file *AuthoringFileSnapshot) error {
		object, ok := objectByKey[provider+"\x00"+file.Hash]
		if !ok {
			return ErrInvalidArchive
		}
		if len(object.JSON) > 0 {
			file.Content = append([]byte(nil), object.JSON...)
			return nil
		}
		opened, openErr := s.objects.Open(ctx, object.ObjectKey)
		if openErr != nil {
			return openErr
		}
		content, readErr := io.ReadAll(io.LimitReader(opened.Reader, MaxFileSize+1))
		closeErr := opened.Reader.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if int64(len(content)) > MaxFileSize {
			return ErrFileTooLarge
		}
		file.Content = content
		return nil
	}
	for index := range snapshot.Scenes {
		for fileIndex := range snapshot.Scenes[index].DraftFiles {
			if err := hydrate(snapshot.Scenes[index].Scene.Provider, &snapshot.Scenes[index].DraftFiles[fileIndex]); err != nil {
				return AuthoringSnapshot{}, err
			}
		}
		for fileIndex := range snapshot.Scenes[index].RevisionFiles {
			if err := hydrate(snapshot.Scenes[index].Scene.Provider, &snapshot.Scenes[index].RevisionFiles[fileIndex]); err != nil {
				return AuthoringSnapshot{}, err
			}
		}
	}
	for index := range snapshot.Assets {
		for fileIndex := range snapshot.Assets[index].GenerationFiles {
			if err := hydrate(snapshot.Assets[index].Asset.Provider, &snapshot.Assets[index].GenerationFiles[fileIndex]); err != nil {
				return AuthoringSnapshot{}, err
			}
		}
		for fileIndex := range snapshot.Assets[index].DraftFiles {
			if err := hydrate(snapshot.Assets[index].Asset.Provider, &snapshot.Assets[index].DraftFiles[fileIndex]); err != nil {
				return AuthoringSnapshot{}, err
			}
		}
	}
	return snapshot, nil
}

// ImportAuthoringSnapshot 先完成外部对象上传，再用一个数据库事务替换项目当前态；事务失败会清理本次预上传对象。
func (s *Service) ImportAuthoringSnapshot(ctx context.Context, actor auth.User, projectID string, options AuthoringSnapshotOptions, snapshot AuthoringSnapshot) error {
	if err := validateAuthoringSnapshotProject(projectID, options); err != nil {
		return err
	}
	if snapshot.ProjectID != projectID {
		return fmt.Errorf("场景可编辑态快照工程归属不匹配")
	}
	if options.Fence != "" && snapshot.Fence != "" && options.Fence != snapshot.Fence {
		return fmt.Errorf("场景可编辑态快照 fence 不匹配")
	}
	if options.Fence != "" && ctx.Value(restoreFenceContextKey{}) != options.Fence {
		return fmt.Errorf("场景恢复 fence 无效")
	}
	if err := s.requireWrite(ctx, actor, projectID); err != nil {
		return err
	}
	if err := validateAuthoringSnapshot(snapshot); err != nil {
		return err
	}
	objects, _, err := s.prepareAuthoringSnapshotObjects(ctx, actor, snapshot)
	if err != nil {
		return err
	}
	if err = s.repository.replaceAuthoringSnapshot(ctx, actor, projectID, snapshot, objects); err != nil {
		return err
	}
	return nil
}

func validateAuthoringSnapshot(snapshot AuthoringSnapshot) error {
	scenes := map[string]struct{}{}
	assets := map[string]*AuthoringAssetSnapshot{}
	validateFiles := func(files []AuthoringFileSnapshot) error {
		seen := map[string]struct{}{}
		for _, file := range files {
			if file.Path == "" || file.Hash == "" || hashBytes(file.Content) != file.Hash {
				return ErrInvalidArchive
			}
			if _, exists := seen[file.Path]; exists {
				return ErrDuplicateFile
			}
			seen[file.Path] = struct{}{}
		}
		return nil
	}
	for index := range snapshot.Scenes {
		item := &snapshot.Scenes[index]
		if item.Scene.ID == "" || item.Scene.ProjectID != snapshot.ProjectID || item.Scene.Provider == "" {
			return ErrInvalidArchive
		}
		if _, exists := scenes[item.Scene.ID]; exists {
			return ErrInvalidArchive
		}
		scenes[item.Scene.ID] = struct{}{}
		if err := validateFiles(item.DraftFiles); err != nil {
			return err
		}
		if err := validateFiles(item.RevisionFiles); err != nil {
			return err
		}
		if (item.Scene.CurrentRevision == 0) != (item.Revision == nil) {
			return ErrInvalidArchive
		}
		if item.Revision != nil && (item.Revision.ID == "" || item.Revision.Revision != item.Scene.CurrentRevision) {
			return ErrInvalidArchive
		}
		if item.Revision != nil && item.Revision.Provider != item.Scene.Provider {
			return ErrInvalidArchive
		}
	}
	for index := range snapshot.Assets {
		item := &snapshot.Assets[index]
		if item.Asset.ID == "" || item.Asset.ProjectID != snapshot.ProjectID || item.Asset.Provider == "" {
			return ErrInvalidArchive
		}
		if _, exists := assets[item.Asset.ID]; exists {
			return ErrInvalidArchive
		}
		assets[item.Asset.ID] = item
		if err := validateFiles(item.GenerationFiles); err != nil {
			return err
		}
		if err := validateFiles(item.DraftFiles); err != nil {
			return err
		}
		if item.Asset.CurrentGeneration == 0 {
			if item.Generation != nil || len(item.GenerationFiles) != 0 {
				return ErrInvalidArchive
			}
		} else if item.Generation == nil || item.Generation.ID == "" || item.Generation.Generation != item.Asset.CurrentGeneration {
			return ErrInvalidArchive
		}
		if item.Generation != nil {
			if item.Generation.AssetID != item.Asset.ID {
				return ErrInvalidArchive
			}
			entryFound := false
			for _, file := range item.GenerationFiles {
				if file.Role != "entry" && file.Role != "dependency" {
					return ErrInvalidArchive
				}
				if file.Role == "entry" && file.Path == item.Generation.EntryPath {
					entryFound = true
				}
			}
			if !entryFound {
				return ErrInvalidArchive
			}
		}
	}
	for _, binding := range snapshot.Bindings {
		if binding.ID == "" || binding.MountPath == "" {
			return ErrInvalidArchive
		}
		if _, exists := scenes[binding.SceneDocumentID]; !exists {
			return ErrInvalidArchive
		}
		asset, exists := assets[binding.AssetID]
		if !exists || asset.Generation == nil || asset.Generation.Generation != binding.Generation {
			return ErrInvalidArchive
		}
	}
	return nil
}

func (s *Service) prepareAuthoringSnapshotObjects(ctx context.Context, actor auth.User, snapshot AuthoringSnapshot) (map[string]contentObject, []string, error) {
	objects := map[string]contentObject{}
	uploaded := []string{}
	visit := func(provider string, file *AuthoringFileSnapshot) error {
		if file.Path == "" || int64(len(file.Content)) > MaxFileSize || hashBytes(file.Content) != file.Hash {
			return ErrInvalidArchive
		}
		key := provider + "\x00" + file.Hash
		if _, ok := objects[key]; ok {
			return nil
		}
		item := contentObject{Hash: file.Hash, Size: int64(len(file.Content)), ContentType: file.ContentType}
		if path.Ext(file.Path) == ".json" {
			if !json.Valid(file.Content) {
				return ErrInvalidJSON
			}
			item.JSON = append([]byte(nil), file.Content...)
		} else {
			item.ObjectKey = objectKey(actor.TenantID, provider, file.Hash)
			if _, err := s.objects.Put(ctx, item.ObjectKey, bytes.NewReader(file.Content), item.Size, item.ContentType); err != nil {
				return err
			}
			uploaded = append(uploaded, item.ObjectKey)
		}
		objects[key] = item
		return nil
	}
	for i := range snapshot.Scenes {
		for _, files := range [][]AuthoringFileSnapshot{snapshot.Scenes[i].DraftFiles, snapshot.Scenes[i].RevisionFiles} {
			for j := range files {
				if err := visit(snapshot.Scenes[i].Scene.Provider, &files[j]); err != nil {
					return nil, uploaded, err
				}
			}
		}
	}
	for i := range snapshot.Assets {
		for _, files := range [][]AuthoringFileSnapshot{snapshot.Assets[i].GenerationFiles, snapshot.Assets[i].DraftFiles} {
			for j := range files {
				if err := visit(snapshot.Assets[i].Asset.Provider, &files[j]); err != nil {
					return nil, uploaded, err
				}
			}
		}
	}
	return objects, uploaded, nil
}

type authoringObjectTarget struct {
	file     *AuthoringFileSnapshot
	provider string
	object   contentObject
}

func appendAuthoringFile(target *[]AuthoringFileSnapshot, objects *[]authoringObjectTarget, provider string, file AuthoringFileSnapshot, object contentObject) {
	object.Hash = file.Hash
	object.ContentType = file.ContentType
	*target = append(*target, file)
	*objects = append(*objects, authoringObjectTarget{file: &(*target)[len(*target)-1], provider: provider, object: object})
}

func (r *PostgreSQLRepository) exportAuthoringSnapshot(ctx context.Context, tenantID, projectID string) (AuthoringSnapshot, []authoringObjectTarget, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return AuthoringSnapshot{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	snapshot, objects, err := exportAuthoringSnapshotTx(ctx, tx, tenantID, projectID)
	if err != nil {
		return AuthoringSnapshot{}, nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AuthoringSnapshot{}, nil, err
	}
	return snapshot, objects, nil
}

type authoringQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func exportAuthoringSnapshotTx(ctx context.Context, queryer authoringQuerier, tenantID, projectID string) (AuthoringSnapshot, []authoringObjectTarget, error) {
	snapshot := AuthoringSnapshot{ProjectID: projectID, Scenes: []AuthoringSceneSnapshot{}, Assets: []AuthoringAssetSnapshot{}, Bindings: []AuthoringBindingSnapshot{}}
	objects := []authoringObjectTarget{}
	rows, err := queryer.Query(ctx, `SELECT id::text,tenant_id::text,project_id::text,scene_id,kind,name,provider,entry_path,public_contract,provider_contract,datapoint_refs,current_revision,draft_version,committed_draft_version,created_at,updated_at FROM scene_documents WHERE tenant_id=$1 AND project_id=$2 AND deleted_at IS NULL ORDER BY id`, tenantID, projectID)
	if err != nil {
		return snapshot, nil, err
	}
	for rows.Next() {
		scene, scanErr := scanScene(rows)
		if scanErr != nil {
			rows.Close()
			return snapshot, nil, scanErr
		}
		snapshot.Scenes = append(snapshot.Scenes, AuthoringSceneSnapshot{Scene: snapshotScene(scene)})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return snapshot, nil, err
	}
	rows.Close()
	for i := range snapshot.Scenes {
		scene := storedScene(snapshot.Scenes[i].Scene, tenantID)
		files, loadErr := loadAuthoringFiles(ctx, queryer, `SELECT n.logical_path,o.content_hash,o.json_content,COALESCE(o.object_key,''),o.content_type,o.content_size,'' FROM scene_file_nodes n JOIN scene_content_objects o ON o.id=n.content_object_id WHERE n.tenant_id=$1 AND n.scene_document_id=$2 AND n.deleted_at IS NULL AND n.node_type='file' ORDER BY n.logical_path`, tenantID, scene.ID)
		if loadErr != nil {
			return snapshot, nil, loadErr
		}
		for _, name := range sortedRevisionFileKeys(files) {
			f := files[name]
			appendAuthoringFile(&snapshot.Scenes[i].DraftFiles, &objects, snapshot.Scenes[i].Scene.Provider, AuthoringFileSnapshot{Path: name, ContentType: f.Type, Hash: f.Hash}, contentObject{JSON: f.Content, ObjectKey: f.ObjectKey})
		}
		if snapshot.Scenes[i].Scene.CurrentRevision > 0 {
			revision, revisionErr := loadAuthoringRevision(ctx, queryer, scene)
			if revisionErr != nil {
				return snapshot, nil, revisionErr
			}
			revisionSnapshot := snapshotRevision(revision)
			snapshot.Scenes[i].Revision = &revisionSnapshot
			revisionFiles, revisionErr := loadAuthoringFiles(ctx, queryer, `SELECT f.logical_path,o.content_hash,o.json_content,COALESCE(o.object_key,''),o.content_type,o.content_size,'' FROM scene_revision_files f JOIN scene_content_objects o ON o.id=f.content_object_id WHERE f.tenant_id=$1 AND f.revision_id=$2 ORDER BY f.logical_path`, tenantID, revision.ID)
			if revisionErr != nil {
				return snapshot, nil, revisionErr
			}
			for _, f := range revisionFiles {
				appendAuthoringFile(&snapshot.Scenes[i].RevisionFiles, &objects, snapshot.Scenes[i].Scene.Provider, AuthoringFileSnapshot{Path: f.Path, ContentType: f.Type, Hash: f.Hash}, contentObject{JSON: f.Content, ObjectKey: f.ObjectKey})
			}
		}
	}
	rows, err = queryer.Query(ctx, `SELECT id::text,tenant_id::text,project_id::text,provider,asset_type,compatible_kind,name,entry_path,current_generation,draft_version,committed_draft_version,false,created_at,updated_at FROM scene_assets WHERE tenant_id=$1 AND project_id=$2 AND archived_at IS NULL ORDER BY id`, tenantID, projectID)
	if err != nil {
		return snapshot, nil, err
	}
	for rows.Next() {
		var asset SceneAsset
		scanErr := rows.Scan(&asset.ID, &asset.TenantID, &asset.ProjectID, &asset.Provider, &asset.Type, &asset.CompatibleKind, &asset.Name, &asset.EntryPath, &asset.CurrentGeneration, &asset.DraftVersion, &asset.CommittedDraftVersion, &asset.Archived, &asset.CreatedAt, &asset.UpdatedAt)
		if scanErr != nil {
			rows.Close()
			return snapshot, nil, scanErr
		}
		item := AuthoringAssetSnapshot{Asset: snapshotAsset(asset)}
		if asset.CurrentGeneration > 0 {
			generation, files, loadErr := loadAuthoringAssetGeneration(ctx, queryer, asset)
			if loadErr != nil {
				rows.Close()
				return snapshot, nil, loadErr
			}
			generationSnapshot := snapshotGeneration(generation)
			item.Generation = &generationSnapshot
			for _, f := range files {
				appendAuthoringFile(&item.GenerationFiles, &objects, asset.Provider, AuthoringFileSnapshot{Path: f.Path, Role: f.Role, ContentType: f.Type, Hash: f.Hash}, contentObject{Hash: f.Hash, JSON: f.Content, ObjectKey: f.ObjectKey})
			}
		}
		draft, loadErr := loadAuthoringFiles(ctx, queryer, `SELECT f.logical_path,o.content_hash,o.json_content,COALESCE(o.object_key,''),o.content_type,o.content_size,'' FROM scene_asset_draft_files f JOIN scene_content_objects o ON o.id=f.content_object_id WHERE f.tenant_id=$1 AND f.asset_id=$2 ORDER BY f.logical_path`, tenantID, asset.ID)
		if loadErr != nil {
			rows.Close()
			return snapshot, nil, loadErr
		}
		for _, name := range sortedRevisionFileKeys(draft) {
			f := draft[name]
			appendAuthoringFile(&item.DraftFiles, &objects, asset.Provider, AuthoringFileSnapshot{Path: name, ContentType: f.Type, Hash: f.Hash}, contentObject{JSON: f.Content, ObjectKey: f.ObjectKey})
		}
		snapshot.Assets = append(snapshot.Assets, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return snapshot, nil, err
	}
	rows.Close()
	rows, err = queryer.Query(ctx, `SELECT b.id::text,b.scene_document_id::text,b.asset_id::text,g.generation,b.mount_path FROM scene_asset_bindings b JOIN scene_asset_generations g ON g.id=b.generation_id WHERE b.tenant_id=$1 AND b.project_id=$2 ORDER BY b.id`, tenantID, projectID)
	if err != nil {
		return snapshot, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b AuthoringBindingSnapshot
		if err = rows.Scan(&b.ID, &b.SceneDocumentID, &b.AssetID, &b.Generation, &b.MountPath); err != nil {
			return snapshot, nil, err
		}
		snapshot.Bindings = append(snapshot.Bindings, b)
	}
	return snapshot, objects, rows.Err()
}

func sortedRevisionFileKeys(files map[string]revisionFile) []string {
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func loadAuthoringFiles(ctx context.Context, queryer authoringQuerier, sql string, args ...any) (map[string]revisionFile, error) {
	rows, err := queryer.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := map[string]revisionFile{}
	for rows.Next() {
		var f revisionFile
		if err = rows.Scan(&f.Path, &f.Hash, &f.Content, &f.ObjectKey, &f.Type, &f.Size, &f.Role); err != nil {
			return nil, err
		}
		items[f.Path] = f
	}
	return items, rows.Err()
}

func loadAuthoringRevision(ctx context.Context, queryer authoringQuerier, scene Scene) (Revision, error) {
	var item Revision
	var contractJSON, refsJSON []byte
	err := queryer.QueryRow(ctx, `SELECT id::text,revision,draft_version,provider,provider_version,entry_path,public_contract,datapoint_refs,root_hash,created_at FROM scene_revisions WHERE tenant_id=$1 AND scene_document_id=$2 AND revision=$3`, scene.TenantID, scene.ID, scene.CurrentRevision).Scan(&item.ID, &item.Revision, &item.DraftVersion, &item.Provider, &item.ProviderVersion, &item.EntryPath, &contractJSON, &refsJSON, &item.RootHash, &item.CreatedAt)
	if err != nil {
		return item, err
	}
	if err = json.Unmarshal(contractJSON, &item.PublicContract); err != nil {
		return item, err
	}
	if err = json.Unmarshal(refsJSON, &item.DatapointRefs); err != nil {
		return item, err
	}
	return item, nil
}

func loadAuthoringAssetGeneration(ctx context.Context, queryer authoringQuerier, asset SceneAsset) (AssetGeneration, []revisionFile, error) {
	var generation AssetGeneration
	var manifest []byte
	err := queryer.QueryRow(ctx, `SELECT id::text,asset_id::text,generation,entry_path,root_hash,manifest,created_at FROM scene_asset_generations WHERE tenant_id=$1 AND asset_id=$2 AND generation=$3`, asset.TenantID, asset.ID, asset.CurrentGeneration).Scan(&generation.ID, &generation.AssetID, &generation.Generation, &generation.EntryPath, &generation.RootHash, &manifest, &generation.CreatedAt)
	if err != nil {
		return generation, nil, err
	}
	if err = json.Unmarshal(manifest, &generation.Manifest); err != nil {
		return generation, nil, err
	}
	files, err := loadAuthoringFiles(ctx, queryer, `SELECT f.logical_path,o.content_hash,o.json_content,COALESCE(o.object_key,''),o.content_type,o.content_size,f.file_role FROM scene_asset_generation_files f JOIN scene_content_objects o ON o.id=f.content_object_id WHERE f.tenant_id=$1 AND f.generation_id=$2 ORDER BY f.logical_path`, asset.TenantID, generation.ID)
	if err != nil {
		return generation, nil, err
	}
	ordered := make([]revisionFile, 0, len(files))
	for _, key := range sortedRevisionFileKeys(files) {
		ordered = append(ordered, files[key])
	}
	return generation, ordered, nil
}

func (r *PostgreSQLRepository) replaceAuthoringSnapshot(ctx context.Context, actor auth.User, projectID string, snapshot AuthoringSnapshot, objects map[string]contentObject) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if ctx.Value(restoreFenceContextKey{}) != nil {
		if _, err = tx.Exec(ctx, `SET LOCAL induforge.authoring_restore='on'`); err != nil {
			return err
		}
	}
	if err = r.replaceAuthoringSnapshotTx(ctx, tx, actor, projectID, snapshot, objects); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgreSQLRepository) replaceAuthoringSnapshotTx(ctx context.Context, tx pgx.Tx, actor auth.User, projectID string, snapshot AuthoringSnapshot, objects map[string]contentObject) error {
	if _, err := tx.Exec(ctx, `DELETE FROM scene_documents WHERE tenant_id=$1 AND project_id=$2`, actor.TenantID, projectID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM scene_assets WHERE tenant_id=$1 AND project_id=$2`, actor.TenantID, projectID); err != nil {
		return err
	}
	var err error
	objectIDs := map[string]string{}
	put := func(provider string, file AuthoringFileSnapshot) (string, error) {
		key := provider + "\x00" + file.Hash
		if id := objectIDs[key]; id != "" {
			return id, nil
		}
		item, ok := objects[key]
		if !ok {
			return "", ErrInvalidArchive
		}
		id, putErr := putContentObject(ctx, tx, actor, provider, item)
		if putErr == nil {
			objectIDs[key] = id
		}
		return id, putErr
	}
	for _, item := range snapshot.Assets {
		a := item.Asset
		if _, err = tx.Exec(ctx, `INSERT INTO scene_assets(id,tenant_id,project_id,provider,asset_type,compatible_kind,name,entry_path,current_generation,draft_version,committed_draft_version,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12)`, a.ID, actor.TenantID, projectID, a.Provider, a.Type, a.CompatibleKind, a.Name, a.EntryPath, a.CurrentGeneration, a.DraftVersion, a.CommittedDraftVersion, actor.ID); err != nil {
			return err
		}
		generationID := ""
		if item.Generation != nil {
			generationID = item.Generation.ID
			if generationID == "" {
				return ErrInvalidArchive
			}
			manifest, marshalErr := json.Marshal(item.Generation.Manifest)
			if marshalErr != nil {
				return marshalErr
			}
			if _, err = tx.Exec(ctx, `INSERT INTO scene_asset_generations(id,tenant_id,project_id,asset_id,generation,entry_path,root_hash,manifest,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`, generationID, actor.TenantID, projectID, a.ID, item.Generation.Generation, item.Generation.EntryPath, item.Generation.RootHash, manifest, actor.ID); err != nil {
				return err
			}
		} else if a.CurrentGeneration != 0 || len(item.GenerationFiles) != 0 {
			return ErrInvalidArchive
		}
		for _, f := range item.GenerationFiles {
			id, putErr := put(a.Provider, f)
			if putErr != nil {
				return putErr
			}
			role := f.Role
			if role == "" {
				role = "dependency"
			}
			if _, err = tx.Exec(ctx, `INSERT INTO scene_asset_generation_files(tenant_id,project_id,asset_id,generation_id,logical_path,file_role,content_object_id,content_hash,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`, actor.TenantID, projectID, a.ID, generationID, f.Path, role, id, f.Hash, actor.ID); err != nil {
				return err
			}
		}
		for _, f := range item.DraftFiles {
			id, putErr := put(a.Provider, f)
			if putErr != nil {
				return putErr
			}
			if _, err = tx.Exec(ctx, `INSERT INTO scene_asset_draft_files(tenant_id,project_id,asset_id,logical_path,content_object_id,content_hash,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$7)`, actor.TenantID, projectID, a.ID, f.Path, id, f.Hash, actor.ID); err != nil {
				return err
			}
		}
	}
	for _, item := range snapshot.Scenes {
		s := item.Scene
		stored := storedScene(s, actor.TenantID)
		contract, marshalErr := json.Marshal(s.PublicContract)
		if marshalErr != nil {
			return marshalErr
		}
		managed, marshalErr := json.Marshal(s.ManagedContract)
		if marshalErr != nil {
			return marshalErr
		}
		refs, marshalErr := json.Marshal(s.DatapointRefs)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = tx.Exec(ctx, `INSERT INTO scene_documents(id,tenant_id,project_id,scene_id,kind,name,provider,entry_path,public_contract,provider_contract,datapoint_refs,current_revision,draft_version,committed_draft_version,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$15)`, s.ID, actor.TenantID, projectID, s.SceneID, s.Kind, s.Name, s.Provider, s.EntryPath, contract, managed, refs, s.CurrentRevision, s.DraftVersion, s.CommittedDraftVersion, actor.ID); err != nil {
			return err
		}
		for _, f := range item.DraftFiles {
			id, putErr := put(s.Provider, f)
			if putErr != nil {
				return putErr
			}
			if err = ensureDirectories(ctx, tx, actor, stored, f.Path); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO scene_file_nodes(tenant_id,project_id,scene_document_id,provider,logical_path,parent_path,node_type,content_object_id,content_hash,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,'file',$7,$8,$9,$9)`, actor.TenantID, projectID, s.ID, s.Provider, f.Path, parentPath(f.Path), id, f.Hash, actor.ID); err != nil {
				return err
			}
		}
		if item.Revision != nil {
			rev := item.Revision
			revisionContract, marshalErr := json.Marshal(rev.PublicContract)
			if marshalErr != nil {
				return marshalErr
			}
			revisionRefs, marshalErr := json.Marshal(rev.DatapointRefs)
			if marshalErr != nil {
				return marshalErr
			}
			if _, err = tx.Exec(ctx, `INSERT INTO scene_revisions(id,tenant_id,project_id,scene_document_id,revision,draft_version,provider,provider_version,entry_path,public_contract,datapoint_refs,root_hash,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13)`, rev.ID, actor.TenantID, projectID, s.ID, rev.Revision, rev.DraftVersion, rev.Provider, rev.ProviderVersion, rev.EntryPath, revisionContract, revisionRefs, rev.RootHash, actor.ID); err != nil {
				return err
			}
			for _, f := range item.RevisionFiles {
				id, putErr := put(s.Provider, f)
				if putErr != nil {
					return putErr
				}
				if _, err = tx.Exec(ctx, `INSERT INTO scene_revision_files(tenant_id,project_id,scene_document_id,revision_id,logical_path,content_object_id,content_hash,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8)`, actor.TenantID, projectID, s.ID, rev.ID, f.Path, id, f.Hash, actor.ID); err != nil {
					return err
				}
			}
		}
	}
	for _, b := range snapshot.Bindings {
		var generationID string
		if err = tx.QueryRow(ctx, `SELECT id::text FROM scene_asset_generations WHERE tenant_id=$1 AND project_id=$2 AND asset_id=$3 AND generation=$4`, actor.TenantID, projectID, b.AssetID, b.Generation).Scan(&generationID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO scene_asset_bindings(id,tenant_id,project_id,scene_document_id,asset_id,generation_id,mount_path,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8)`, b.ID, actor.TenantID, projectID, b.SceneDocumentID, b.AssetID, generationID, b.MountPath, actor.ID); err != nil {
			return err
		}
	}
	return nil
}

type PreparedAuthoringRestore struct {
	objects  map[string]contentObject
	snapshot AuthoringSnapshot
}

func (s *Service) PrepareAuthoringRestore(ctx context.Context, actor auth.User, projectID string, options AuthoringSnapshotOptions, snapshot AuthoringSnapshot) (*PreparedAuthoringRestore, error) {
	if err := validateAuthoringSnapshotProject(projectID, options); err != nil {
		return nil, err
	}
	if snapshot.ProjectID != projectID {
		return nil, fmt.Errorf("场景可编辑态快照工程归属不匹配")
	}
	if options.Fence == "" || ctx.Value(restoreFenceContextKey{}) != options.Fence {
		return nil, fmt.Errorf("场景恢复 fence 无效")
	}
	if err := validateAuthoringSnapshot(snapshot); err != nil {
		return nil, err
	}
	objects, _, err := s.prepareAuthoringSnapshotObjects(ctx, actor, snapshot)
	if err != nil {
		return nil, err
	}
	return &PreparedAuthoringRestore{objects: objects, snapshot: snapshot}, nil
}
func (s *Service) ApplyPreparedAuthoringRestore(ctx context.Context, tx pgx.Tx, actor auth.User, projectID, fence string, prepared *PreparedAuthoringRestore) error {
	if prepared == nil || fence == "" || ctx.Value(restoreFenceContextKey{}) != fence {
		return fmt.Errorf("场景恢复准备无效")
	}
	if _, err := tx.Exec(ctx, `SET LOCAL induforge.authoring_restore='on'`); err != nil {
		return err
	}
	return s.repository.replaceAuthoringSnapshotTx(ctx, tx, actor, projectID, prepared.snapshot, prepared.objects)
}
