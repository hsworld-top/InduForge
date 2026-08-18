package sceneasset

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
	platformcache "github.com/indu-forge/dev_core/internal/platform/cache"
	"github.com/jackc/pgx/v5"
)

const maxAssetArchiveSize = int64(512 << 20)
const maxAssetArchiveFiles = 10000

func (s *Service) ListAssets(ctx context.Context, actor auth.User, projectID string, filter AssetListFilter) ([]SceneAsset, int64, error) {
	if err := s.requireRead(ctx, actor, projectID); err != nil {
		return nil, 0, err
	}
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	if filter.Type != "" && !validAssetType(filter.Type) {
		return nil, 0, ErrInvalidAssetType
	}
	if filter.Kind != "" && filter.Kind != "2d" && filter.Kind != "3d" {
		return nil, 0, ErrInvalidKind
	}
	provider := ProviderHT
	if state, err := s.repository.GetProvider(ctx); err == nil {
		provider = state.Provider
	} else if !errors.Is(err, ErrProviderNotSet) {
		return nil, 0, err
	}
	items, total, err := s.repository.ListAssets(ctx, actor.TenantID, projectID, provider, "", filter)
	if err != nil {
		return nil, 0, err
	}
	setAssetThumbnailURLs(projectID, items)
	return items, total, nil
}

func (s *Service) ListSessionAssets(ctx context.Context, actor auth.User, sessionID string, filter AssetListFilter) ([]SceneAsset, []AssetBinding, int64, Scene, error) {
	_, scene, err := s.ResolveSession(ctx, actor, sessionID)
	if err != nil {
		return nil, nil, 0, Scene{}, err
	}
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	filter.Kind = scene.Kind
	if filter.Type != "" && !validAssetType(filter.Type) {
		return nil, nil, 0, Scene{}, ErrInvalidAssetType
	}
	items, total, err := s.repository.ListAssets(ctx, actor.TenantID, scene.ProjectID, scene.Provider, scene.ID, filter)
	if err != nil {
		return nil, nil, 0, Scene{}, err
	}
	bindings, err := s.repository.ListAssetBindings(ctx, scene)
	if err != nil {
		return nil, nil, 0, Scene{}, err
	}
	setAssetThumbnailURLs(scene.ProjectID, items)
	return items, bindings, total, scene, nil
}

func setAssetThumbnailURLs(projectID string, items []SceneAsset) {
	for index := range items {
		items[index].ThumbnailURL = "/api/v1/projects/" + projectID + "/scene-assets/" + items[index].ID + "/thumbnail"
	}
}

func (s *Service) GetAsset(ctx context.Context, actor auth.User, projectID, assetID string) (SceneAsset, error) {
	if err := s.requireRead(ctx, actor, projectID); err != nil {
		return SceneAsset{}, err
	}
	asset, err := s.repository.GetAsset(ctx, actor.TenantID, projectID, assetID)
	if err == nil {
		asset.ThumbnailURL = "/api/v1/projects/" + projectID + "/scene-assets/" + asset.ID + "/thumbnail"
	}
	return asset, err
}

func (s *Service) ImportAsset(ctx context.Context, actor auth.User, projectID string, input AssetImportInput) (SceneAsset, error) {
	if err := s.requireWrite(ctx, actor, projectID); err != nil {
		return SceneAsset{}, err
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len([]rune(input.Name)) > 200 {
		return SceneAsset{}, ErrInvalidAssetName
	}
	if !validAssetType(input.Type) {
		return SceneAsset{}, ErrInvalidAssetType
	}
	state, err := s.repository.GetProvider(ctx)
	if errors.Is(err, ErrProviderNotSet) {
		state, err = s.repository.SetProvider(ctx, actor, ProviderHT, HTProviderVersion)
	}
	if err != nil {
		return SceneAsset{}, err
	}
	provider, err := s.providers.Get(state.Provider)
	if err != nil {
		return SceneAsset{}, err
	}
	providerFiles, err := unpackAsset(input)
	if err != nil {
		return SceneAsset{}, err
	}
	analysis, err := provider.AnalyzeAsset(ctx, input.Type, providerFiles)
	if err != nil {
		return SceneAsset{}, err
	}
	stored, uploaded, err := s.storeAssetFiles(ctx, actor, state.Provider, providerFiles)
	if err != nil {
		s.cleanupUncommittedObjects(ctx, actor.TenantID, state.Provider, uploaded)
		return SceneAsset{}, err
	}
	result, err := s.repository.CreateAsset(ctx, actor, projectID, state.Provider, input, analysis, assetRootHash(providerFiles), stored)
	if err != nil {
		s.cleanupUncommittedObjects(ctx, actor.TenantID, state.Provider, uploaded)
		return SceneAsset{}, err
	}
	result.ThumbnailURL = "/api/v1/projects/" + projectID + "/scene-assets/" + result.ID + "/thumbnail"
	return result, nil
}

func (s *Service) ImportSessionAsset(ctx context.Context, actor auth.User, sessionID string, input AssetImportInput) (SceneAsset, error) {
	session, _, err := s.ResolveSession(ctx, actor, sessionID)
	if err != nil {
		return SceneAsset{}, err
	}
	return s.ImportAsset(ctx, actor, session.ProjectID, input)
}

func (s *Service) ReplaceSessionAsset(ctx context.Context, actor auth.User, sessionID, assetID string, input AssetImportInput) (SceneAsset, error) {
	session, _, err := s.ResolveSession(ctx, actor, sessionID)
	if err != nil {
		return SceneAsset{}, err
	}
	return s.ReplaceAsset(ctx, actor, session.ProjectID, assetID, input)
}

func (s *Service) ArchiveSessionAsset(ctx context.Context, actor auth.User, sessionID, assetID string) error {
	session, _, err := s.ResolveSession(ctx, actor, sessionID)
	if err != nil {
		return err
	}
	return s.ArchiveAsset(ctx, actor, session.ProjectID, assetID)
}

func (s *Service) CreateSessionAssetEditor(ctx context.Context, actor auth.User, sessionID, assetID string) (EditorSessionResponse, error) {
	session, _, err := s.ResolveSession(ctx, actor, sessionID)
	if err != nil {
		return EditorSessionResponse{}, err
	}
	return s.CreateAssetEditorSession(ctx, actor, session.ProjectID, assetID)
}

func (s *Service) ReplaceAsset(ctx context.Context, actor auth.User, projectID, assetID string, input AssetImportInput) (SceneAsset, error) {
	if err := s.requireWrite(ctx, actor, projectID); err != nil {
		return SceneAsset{}, err
	}
	asset, err := s.repository.GetAsset(ctx, actor.TenantID, projectID, assetID)
	if err != nil {
		return SceneAsset{}, err
	}
	if asset.Archived {
		return SceneAsset{}, ErrAssetArchived
	}
	input.Type = asset.Type
	input.Name = asset.Name
	provider, err := s.providers.Get(asset.Provider)
	if err != nil {
		return SceneAsset{}, err
	}
	providerFiles, err := unpackAsset(input)
	if err != nil {
		return SceneAsset{}, err
	}
	analysis, err := provider.AnalyzeAsset(ctx, asset.Type, providerFiles)
	if err != nil {
		return SceneAsset{}, err
	}
	stored, uploaded, err := s.storeAssetFiles(ctx, actor, asset.Provider, providerFiles)
	if err != nil {
		s.cleanupUncommittedObjects(ctx, actor.TenantID, asset.Provider, uploaded)
		return SceneAsset{}, err
	}
	result, err := s.repository.ReplaceAsset(ctx, actor, asset, analysis, assetRootHash(providerFiles), stored)
	if err != nil {
		s.cleanupUncommittedObjects(ctx, actor.TenantID, asset.Provider, uploaded)
		return SceneAsset{}, err
	}
	result.ThumbnailURL = "/api/v1/projects/" + projectID + "/scene-assets/" + result.ID + "/thumbnail"
	return result, nil
}

func (s *Service) UpdateAsset(ctx context.Context, actor auth.User, projectID, assetID string, patch AssetPatch) (SceneAsset, error) {
	if err := s.requireWrite(ctx, actor, projectID); err != nil {
		return SceneAsset{}, err
	}
	name := strings.TrimSpace(patch.Name)
	if name == "" || len([]rune(name)) > 200 {
		return SceneAsset{}, ErrInvalidAssetName
	}
	asset, err := s.repository.GetAsset(ctx, actor.TenantID, projectID, assetID)
	if err != nil {
		return SceneAsset{}, err
	}
	return s.repository.UpdateAsset(ctx, actor, asset, name)
}

func (s *Service) ArchiveAsset(ctx context.Context, actor auth.User, projectID, assetID string) error {
	if err := s.requireWrite(ctx, actor, projectID); err != nil {
		return err
	}
	asset, err := s.repository.GetAsset(ctx, actor.TenantID, projectID, assetID)
	if err != nil {
		return err
	}
	return s.repository.ArchiveAsset(ctx, actor, asset)
}

func (s *Service) ApplyAssetAction(ctx context.Context, actor auth.User, sessionID string, action AssetAction) (int64, error) {
	_, scene, err := s.ResolveSession(ctx, actor, sessionID)
	if err != nil {
		return 0, err
	}
	action.Action = strings.ToLower(strings.TrimSpace(action.Action))
	if action.Action != "attach" && action.Action != "update" && action.Action != "detach" {
		return 0, ErrInvalidPath
	}
	asset, err := s.repository.GetAsset(ctx, actor.TenantID, scene.ProjectID, action.AssetID)
	if err != nil {
		return 0, err
	}
	if asset.Archived && action.Action != "detach" {
		return 0, ErrAssetArchived
	}
	provider, err := s.providers.Get(scene.Provider)
	if err != nil {
		return 0, err
	}
	return s.repository.ApplyAssetAction(ctx, actor, scene, action, provider.MountPath(asset))
}

func (s *Service) ListDependencies(ctx context.Context, actor auth.User, sessionID string, filter FileListFilter) ([]DependencyNode, int64, Scene, error) {
	_, scene, err := s.ResolveSession(ctx, actor, sessionID)
	if err != nil {
		return nil, 0, Scene{}, err
	}
	filter.Directory, err = NormalizePath(filter.Directory, true)
	if err != nil {
		return nil, 0, Scene{}, err
	}
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	items, total, err := s.repository.ListDependencies(ctx, scene, filter)
	return items, total, scene, err
}

func (s *Service) OpenAssetThumbnail(ctx context.Context, actor auth.User, projectID, assetID string) (FileContent, error) {
	asset, err := s.GetAsset(ctx, actor, projectID, assetID)
	if err != nil {
		return FileContent{}, err
	}
	if asset.Type != AssetImage {
		return FileContent{}, ErrNotFound
	}
	_, files, err := s.repository.LoadAssetGenerationFiles(ctx, asset, asset.CurrentGeneration)
	if err != nil {
		return FileContent{}, err
	}
	item := files[asset.EntryPath]
	return s.openStoredContent(ctx, item)
}

func (s *Service) CreateAssetEditorSession(ctx context.Context, actor auth.User, projectID, assetID string) (EditorSessionResponse, error) {
	if err := s.requireWrite(ctx, actor, projectID); err != nil {
		return EditorSessionResponse{}, err
	}
	asset, err := s.repository.GetAsset(ctx, actor.TenantID, projectID, assetID)
	if err != nil {
		return EditorSessionResponse{}, err
	}
	if asset.Archived {
		return EditorSessionResponse{}, ErrAssetArchived
	}
	if asset.Type != AssetSymbol && asset.Type != AssetComponent {
		return EditorSessionResponse{}, ErrAssetNotEditable
	}
	provider, err := s.providers.Get(asset.Provider)
	if err != nil {
		return EditorSessionResponse{}, err
	}
	now := time.Now().UTC()
	session := AssetEditorSession{ID: uuid.NewString(), UserID: actor.ID, TenantID: actor.TenantID,
		ProjectID: projectID, AssetID: asset.ID, Provider: asset.Provider, ExpiresAt: now.Add(SessionTTL)}
	encoded, _ := json.Marshal(session)
	if err := s.sessions.PutSceneSession(ctx, session.ID, string(encoded), SessionTTL); err != nil {
		return EditorSessionResponse{}, err
	}
	return EditorSessionResponse{SessionID: session.ID, URL: provider.AssetEditorURL(session), ExpiresAt: session.ExpiresAt}, nil
}

func (s *Service) ResolveAssetSession(ctx context.Context, actor auth.User, sessionID string) (AssetEditorSession, SceneAsset, error) {
	encoded, err := s.sessions.GetSceneSession(ctx, sessionID)
	if errors.Is(err, platformcache.ErrMiss) {
		return AssetEditorSession{}, SceneAsset{}, ErrSessionExpired
	}
	if err != nil {
		return AssetEditorSession{}, SceneAsset{}, err
	}
	var session AssetEditorSession
	if err := json.Unmarshal([]byte(encoded), &session); err != nil {
		return AssetEditorSession{}, SceneAsset{}, ErrSessionExpired
	}
	if session.ID != sessionID || session.UserID != actor.ID || session.TenantID != actor.TenantID || time.Now().After(session.ExpiresAt) {
		return AssetEditorSession{}, SceneAsset{}, ErrSessionForbidden
	}
	if err := s.requireWrite(ctx, actor, session.ProjectID); err != nil {
		return AssetEditorSession{}, SceneAsset{}, err
	}
	asset, err := s.repository.GetAsset(ctx, actor.TenantID, session.ProjectID, session.AssetID)
	if err != nil {
		return AssetEditorSession{}, SceneAsset{}, err
	}
	if asset.Provider != session.Provider || (asset.Type != AssetSymbol && asset.Type != AssetComponent) {
		return AssetEditorSession{}, SceneAsset{}, ErrSessionForbidden
	}
	if asset.Archived {
		return AssetEditorSession{}, SceneAsset{}, ErrAssetArchived
	}
	return session, asset, nil
}

func (s *Service) ListAssetDraftFiles(ctx context.Context, actor auth.User, sessionID string, filter FileListFilter) ([]FileNode, int64, SceneAsset, error) {
	_, asset, err := s.ResolveAssetSession(ctx, actor, sessionID)
	if err != nil {
		return nil, 0, SceneAsset{}, err
	}
	filter.Directory, err = NormalizePath(filter.Directory, true)
	if err != nil {
		return nil, 0, SceneAsset{}, err
	}
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	items, total, err := s.repository.ListAssetDraftFiles(ctx, asset, filter)
	return items, total, asset, err
}

func (s *Service) OpenAssetDraftFile(ctx context.Context, actor auth.User, sessionID, filename string) (FileContent, error) {
	_, asset, err := s.ResolveAssetSession(ctx, actor, sessionID)
	if err != nil {
		return FileContent{}, err
	}
	filename, err = NormalizePath(filename, false)
	if err != nil {
		return FileContent{}, err
	}
	item, err := s.repository.GetAssetDraftFile(ctx, asset, filename)
	if err != nil {
		return FileContent{}, err
	}
	return s.openContentObject(ctx, item)
}

func (s *Service) PutAssetDraftFile(ctx context.Context, actor auth.User, sessionID string, input FileWrite) (int64, error) {
	_, asset, err := s.ResolveAssetSession(ctx, actor, sessionID)
	if err != nil {
		return 0, err
	}
	input.Path, err = NormalizePath(input.Path, false)
	if err != nil {
		return 0, err
	}
	if input.Path != asset.EntryPath || !strings.EqualFold(path.Ext(input.Path), ".json") {
		return 0, ErrInvalidPath
	}
	providerFiles := map[string]ProviderFile{input.Path: {Path: input.Path, ContentType: input.ContentType, Content: input.Content}}
	provider, err := s.providers.Get(asset.Provider)
	if err != nil {
		return 0, err
	}
	if err := provider.ValidateFile(input.Path, input.Content); err != nil {
		return 0, err
	}
	stored, uploaded, err := s.storeAssetFiles(ctx, actor, asset.Provider, providerFiles)
	if err != nil {
		s.cleanupUncommittedObjects(ctx, actor.TenantID, asset.Provider, uploaded)
		return 0, err
	}
	next, err := s.repository.PutAssetDraftFile(ctx, actor, asset, input, stored[input.Path])
	if err != nil {
		s.cleanupUncommittedObjects(ctx, actor.TenantID, asset.Provider, uploaded)
	}
	return next, err
}

func (s *Service) CommitAssetDraft(ctx context.Context, actor auth.User, sessionID string, baseDraft int64) (SceneAsset, error) {
	_, asset, err := s.ResolveAssetSession(ctx, actor, sessionID)
	if err != nil {
		return SceneAsset{}, err
	}
	stored, err := s.repository.LoadAssetDraft(ctx, asset)
	if err != nil {
		return SceneAsset{}, err
	}
	providerFiles := make(map[string]ProviderFile, len(stored))
	objects := make(map[string]contentObject, len(stored))
	for filename, item := range stored {
		providerFiles[filename] = ProviderFile{Path: filename, ContentType: item.Type, Content: item.Content, Hash: item.Hash}
		objects[filename] = contentObject{ID: item.ObjectID, Hash: item.Hash, Size: item.Size, ContentType: item.Type,
			JSON: item.Content, ObjectKey: item.ObjectKey}
	}
	provider, err := s.providers.Get(asset.Provider)
	if err != nil {
		return SceneAsset{}, err
	}
	analysis, err := provider.AnalyzeAsset(ctx, asset.Type, providerFiles)
	if err != nil {
		return SceneAsset{}, err
	}
	return s.repository.CommitAssetDraft(ctx, actor, asset, baseDraft, analysis, assetRootHash(providerFiles), objects)
}

func unpackAsset(input AssetImportInput) (map[string]ProviderFile, error) {
	if int64(len(input.Content)) > maxAssetArchiveSize {
		return nil, ErrFileTooLarge
	}
	if strings.EqualFold(path.Ext(input.Filename), ".zip") || strings.Contains(strings.ToLower(input.ContentType), "zip") {
		reader, err := zip.NewReader(bytes.NewReader(input.Content), int64(len(input.Content)))
		if err != nil || len(reader.File) == 0 || len(reader.File) > maxAssetArchiveFiles {
			return nil, ErrInvalidArchive
		}
		files := make(map[string]ProviderFile, len(reader.File))
		var total uint64
		for _, zipped := range reader.File {
			if zipped.FileInfo().Mode()&os.ModeSymlink != 0 {
				return nil, ErrInvalidArchive
			}
			if zipped.FileInfo().IsDir() {
				continue
			}
			filename, err := NormalizePath(strings.TrimSuffix(zipped.Name, "/"), false)
			if err != nil {
				return nil, err
			}
			if _, exists := files[filename]; exists {
				return nil, ErrDuplicateFile
			}
			total += zipped.UncompressedSize64
			if zipped.UncompressedSize64 > uint64(MaxFileSize) || total > uint64(maxAssetArchiveSize) {
				return nil, ErrFileTooLarge
			}
			file, err := zipped.Open()
			if err != nil {
				return nil, ErrInvalidArchive
			}
			content, readErr := io.ReadAll(io.LimitReader(file, MaxFileSize+1))
			_ = file.Close()
			if readErr != nil || int64(len(content)) > MaxFileSize {
				return nil, ErrFileTooLarge
			}
			files[filename] = canonicalProviderFile(filename, content, "")
		}
		if len(files) == 0 {
			return nil, ErrInvalidArchive
		}
		if err := validateProviderFilePaths(files); err != nil {
			return nil, err
		}
		return files, nil
	}
	if int64(len(input.Content)) > MaxFileSize {
		return nil, ErrFileTooLarge
	}
	filename := path.Base(strings.ReplaceAll(strings.TrimSpace(input.Filename), `\`, "/"))
	if filename == "." || filename == "" {
		return nil, ErrInvalidPath
	}
	return map[string]ProviderFile{filename: canonicalProviderFile(filename, input.Content, input.ContentType)}, nil
}

// validateProviderFilePaths 拒绝 a 与 a/b 这类文件祖先冲突，避免非法 ZIP 进入数据库事务后才失败。
func validateProviderFilePaths(files map[string]ProviderFile) error {
	for filename := range files {
		for parent := parentPath(filename); parent != ""; parent = parentPath(parent) {
			if _, collision := files[parent]; collision {
				return ErrDuplicateFile
			}
		}
	}
	return nil
}

func canonicalProviderFile(filename string, content []byte, contentType string) ProviderFile {
	if strings.EqualFold(path.Ext(filename), ".json") && json.Valid(content) {
		var value any
		_ = json.Unmarshal(content, &value)
		content, _ = json.Marshal(value)
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = mime.TypeByExtension(strings.ToLower(path.Ext(filename)))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return ProviderFile{Path: filename, ContentType: contentType, Content: content, Hash: hashBytes(content)}
}

func (s *Service) storeAssetFiles(ctx context.Context, actor auth.User, provider string, files map[string]ProviderFile) (map[string]contentObject, []string, error) {
	stored := make(map[string]contentObject, len(files))
	uploaded := []string{}
	paths := make([]string, 0, len(files))
	for filename := range files {
		paths = append(paths, filename)
	}
	sort.Strings(paths)
	for _, filename := range paths {
		file := files[filename]
		item := contentObject{Hash: file.Hash, Size: int64(len(file.Content)), ContentType: file.ContentType}
		if strings.EqualFold(path.Ext(filename), ".json") {
			if !json.Valid(file.Content) {
				return nil, uploaded, ErrInvalidJSON
			}
			item.JSON = file.Content
		} else if existing, err := s.repository.FindContent(ctx, actor.TenantID, provider, file.Hash); err == nil {
			item = existing
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, uploaded, err
		} else {
			item.ObjectKey = objectKey(actor.TenantID, provider, file.Hash)
			if _, err := s.objects.Put(ctx, item.ObjectKey, bytes.NewReader(file.Content), item.Size, item.ContentType); err != nil {
				return nil, uploaded, err
			}
			uploaded = append(uploaded, item.ObjectKey)
		}
		stored[filename] = item
	}
	return stored, uploaded, nil
}

func (s *Service) cleanupUncommittedObjects(ctx context.Context, tenantID, provider string, keys []string) {
	for _, key := range keys {
		hash := path.Base(key)
		if _, err := s.repository.FindContent(ctx, tenantID, provider, hash); errors.Is(err, pgx.ErrNoRows) {
			_ = s.objects.Delete(ctx, key)
		}
	}
}

func (s *Service) openStoredContent(ctx context.Context, item revisionFile) (FileContent, error) {
	if len(item.Content) > 0 {
		return FileContent{Bytes: item.Content, Size: int64(len(item.Content)), ContentType: item.Type, Hash: item.Hash}, nil
	}
	object, err := s.objects.Open(ctx, item.ObjectKey)
	if err != nil {
		return FileContent{}, err
	}
	return FileContent{Reader: object.Reader, Size: object.Size, ContentType: item.Type, Hash: item.Hash}, nil
}

func (s *Service) openContentObject(ctx context.Context, item contentObject) (FileContent, error) {
	return s.openStoredContent(ctx, revisionFile{Hash: item.Hash, Content: item.JSON, ObjectKey: item.ObjectKey, Type: item.ContentType, Size: item.Size})
}

func validAssetType(value AssetType) bool {
	switch value {
	case AssetImage, AssetFont, AssetModel, AssetMaterial, AssetSymbol, AssetComponent:
		return true
	default:
		return false
	}
}

var _ = fmt.Sprintf
