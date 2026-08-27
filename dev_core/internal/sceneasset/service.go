package sceneasset

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/indu-forge/dev_core/internal/objectstore"
	platformcache "github.com/indu-forge/dev_core/internal/platform/cache"
	"github.com/indu-forge/dev_core/internal/project"
	"github.com/indu-forge/dev_core/internal/scenecontract"
	"github.com/jackc/pgx/v5"
)

type projectReader interface {
	Get(context.Context, string, string) (project.Project, error)
}

type objectStore interface {
	Put(context.Context, string, io.Reader, int64, string) (objectstore.ObjectRef, error)
	Open(context.Context, string) (objectstore.ObjectReader, error)
	Delete(context.Context, string) error
}

type sessionStore interface {
	PutSceneSession(context.Context, string, string, time.Duration) error
	GetSceneSession(context.Context, string) (string, error)
	DeleteSceneSession(context.Context, string) error
}

type Service struct {
	repository *PostgreSQLRepository
	projects   projectReader
	objects    objectStore
	sessions   sessionStore
	providers  *Registry
}

func NewService(repository *PostgreSQLRepository, projects projectReader, objects objectStore, sessions sessionStore, providers *Registry) *Service {
	return &Service{repository: repository, projects: projects, objects: objects, sessions: sessions, providers: providers}
}

func (s *Service) GetProvider(ctx context.Context, actor auth.User) (ProviderState, ProviderCapabilities, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectRead); err != nil {
		return ProviderState{}, ProviderCapabilities{}, err
	}
	state, err := s.repository.GetProvider(ctx)
	if errors.Is(err, ErrProviderNotSet) {
		provider, _ := s.providers.Get(ProviderHT)
		return ProviderState{Provider: provider.Name(), ProviderVersion: provider.Version()}, provider.Capabilities(), nil
	}
	if err != nil {
		return ProviderState{}, ProviderCapabilities{}, err
	}
	provider, err := s.providers.Get(state.Provider)
	if err != nil {
		return ProviderState{}, ProviderCapabilities{}, err
	}
	return state, provider.Capabilities(), nil
}

func (s *Service) SetProvider(ctx context.Context, actor auth.User, name string) (ProviderState, error) {
	if !auth.IsPlatformAdmin(actor.Role) {
		return ProviderState{}, auth.ErrPermissionDenied
	}
	provider, err := s.providers.Get(name)
	if err != nil {
		return ProviderState{}, err
	}
	return s.repository.SetProvider(ctx, actor, provider.Name(), provider.Version())
}

func (s *Service) CreateScene(ctx context.Context, actor auth.User, projectID string, input SceneInput) (Scene, error) {
	if err := s.requireWrite(ctx, actor, projectID); err != nil {
		return Scene{}, err
	}
	input.ProjectID = projectID
	// 场景 ID 是页面源码和运行工件使用的稳定内部标识，不接受用户输入。
	input.SceneID = uuid.NewString()
	input.Kind = strings.ToLower(strings.TrimSpace(input.Kind))
	input.Name = strings.TrimSpace(input.Name)
	if input.Kind != "2d" && input.Kind != "3d" {
		return Scene{}, ErrInvalidKind
	}
	if err := validateSceneName(input.Name); err != nil {
		return Scene{}, err
	}
	nameExists, err := s.repository.ActiveSceneNameExists(ctx, actor.TenantID, projectID, input.Name, "")
	if err != nil {
		return Scene{}, err
	}
	if nameExists {
		return Scene{}, ErrSceneNameConflict
	}
	input.PublicContract, err = validatePublicContract(input.PublicContract)
	if err != nil {
		return Scene{}, err
	}
	entry, err := entryPath(input.Kind, input.SceneID, input.EntryPath)
	if err != nil {
		return Scene{}, err
	}
	input.EntryPath = entry
	state, err := s.repository.GetProvider(ctx)
	if errors.Is(err, ErrProviderNotSet) {
		state, err = s.repository.SetProvider(ctx, actor, ProviderHT, HTProviderVersion)
	}
	if err != nil {
		return Scene{}, err
	}
	provider, err := s.providers.Get(state.Provider)
	if err != nil {
		return Scene{}, err
	}
	if err := provider.ValidateEntry(input.Kind, entry); err != nil {
		return Scene{}, err
	}
	initial := []byte("{}")
	hash := hashBytes(initial)
	return s.repository.CreateScene(ctx, actor, input, state.Provider, contentObject{Hash: hash, Size: int64(len(initial)), ContentType: "application/json", JSON: initial})
}

func (s *Service) ListScenes(ctx context.Context, actor auth.User, projectID string, filter ListFilter) ([]Scene, int64, error) {
	if err := s.requireRead(ctx, actor, projectID); err != nil {
		return nil, 0, err
	}
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	if filter.Kind != "" && filter.Kind != "2d" && filter.Kind != "3d" {
		return nil, 0, ErrInvalidKind
	}
	return s.repository.ListScenes(ctx, actor.TenantID, projectID, filter)
}

func (s *Service) GetScene(ctx context.Context, actor auth.User, projectID, kind, sceneID string) (Scene, error) {
	if err := s.requireRead(ctx, actor, projectID); err != nil {
		return Scene{}, err
	}
	if kind == "" {
		kind = "2d"
	}
	return s.repository.GetScene(ctx, actor.TenantID, projectID, kind, sceneID)
}

func (s *Service) UpdateScene(ctx context.Context, actor auth.User, projectID, kind, sceneID string, patch ScenePatch) (Scene, error) {
	if err := s.requireWrite(ctx, actor, projectID); err != nil {
		return Scene{}, err
	}
	scene, err := s.repository.GetScene(ctx, actor.TenantID, projectID, kind, sceneID)
	if err != nil {
		return Scene{}, err
	}
	if patch.Name != nil {
		name := strings.TrimSpace(*patch.Name)
		if err := validateSceneName(name); err != nil {
			return Scene{}, err
		}
		nameExists, err := s.repository.ActiveSceneNameExists(ctx, actor.TenantID, projectID, name, scene.ID)
		if err != nil {
			return Scene{}, err
		}
		if nameExists {
			return Scene{}, ErrSceneNameConflict
		}
		patch.Name = &name
	}
	if patch.PublicContract != nil {
		if patch.BaseDraftVersion == nil || *patch.BaseDraftVersion < 0 {
			return Scene{}, fmt.Errorf("%w: 修改交互契约必须提供 baseDraftVersion", ErrContractInvalid)
		}
		normalized, err := validatePublicContract(*patch.PublicContract)
		if err != nil {
			return Scene{}, err
		}
		patch.PublicContract = &normalized
	}
	return s.repository.UpdateScene(ctx, actor, scene, patch)
}

func validateSceneName(name string) error {
	if name == "" || len([]rune(name)) > 200 {
		return ErrInvalidSceneName
	}
	return nil
}

func (s *Service) DeleteScene(ctx context.Context, actor auth.User, projectID, kind, sceneID string) error {
	if err := s.requireWrite(ctx, actor, projectID); err != nil {
		return err
	}
	scene, err := s.repository.GetScene(ctx, actor.TenantID, projectID, kind, sceneID)
	if err != nil {
		return err
	}
	return s.repository.DeleteScene(ctx, actor, scene)
}

func (s *Service) CreateEditorSession(ctx context.Context, actor auth.User, projectID, kind, sceneID string) (EditorSessionResponse, error) {
	if err := s.requireWrite(ctx, actor, projectID); err != nil {
		return EditorSessionResponse{}, err
	}
	scene, err := s.repository.GetScene(ctx, actor.TenantID, projectID, kind, sceneID)
	if err != nil {
		return EditorSessionResponse{}, err
	}
	provider, err := s.providers.Get(scene.Provider)
	if err != nil {
		return EditorSessionResponse{}, err
	}
	now := time.Now().UTC()
	session := EditorSession{ID: uuid.NewString(), UserID: actor.ID, TenantID: actor.TenantID, ProjectID: projectID,
		SceneID: sceneID, SceneName: scene.Name, Kind: kind, Provider: scene.Provider, EntryPath: scene.EntryPath, ExpiresAt: now.Add(SessionTTL)}
	encoded, _ := json.Marshal(session)
	if err := s.sessions.PutSceneSession(ctx, session.ID, string(encoded), SessionTTL); err != nil {
		return EditorSessionResponse{}, err
	}
	return EditorSessionResponse{SessionID: session.ID, SceneName: session.SceneName, URL: provider.EditorURL(session), ExpiresAt: session.ExpiresAt}, nil
}

func (s *Service) CreateViewerSession(ctx context.Context, actor auth.User, projectID, kind, sceneID string) (ViewerSessionResponse, error) {
	if err := s.requireRead(ctx, actor, projectID); err != nil {
		return ViewerSessionResponse{}, err
	}
	scene, err := s.repository.GetScene(ctx, actor.TenantID, projectID, kind, sceneID)
	if err != nil {
		return ViewerSessionResponse{}, err
	}
	revision, err := s.repository.GetCurrentRevision(ctx, scene)
	if err != nil {
		return ViewerSessionResponse{}, err
	}
	provider, err := s.providers.Get(revision.Provider)
	if err != nil {
		return ViewerSessionResponse{}, err
	}
	now := time.Now().UTC()
	session := ViewerSession{
		ID: uuid.NewString(), TenantID: actor.TenantID, ProjectID: projectID, SceneDocumentID: scene.ID,
		SceneID: scene.SceneID, Kind: scene.Kind, Provider: revision.Provider, RevisionID: revision.ID,
		Revision: revision.Revision, EntryPath: revision.EntryPath, Contract: revision.PublicContract, DatapointRefs: revision.DatapointRefs,
		ExpiresAt: now.Add(ViewerSessionTTL), AbsoluteExpiresAt: now.Add(ViewerAbsoluteTTL),
	}
	encoded, _ := json.Marshal(session)
	if err := s.sessions.PutSceneSession(ctx, session.ID, string(encoded), ViewerSessionTTL); err != nil {
		return ViewerSessionResponse{}, err
	}
	return ViewerSessionResponse{SessionID: session.ID, URL: provider.ViewerURL(session), ExpiresAt: session.ExpiresAt,
		SceneID: session.SceneID, Kind: session.Kind, Revision: session.Revision, Contract: session.Contract, DatapointRefs: session.DatapointRefs}, nil
}

func (s *Service) ResolveViewerSession(ctx context.Context, sessionID string) (ViewerSession, error) {
	encoded, err := s.sessions.GetSceneSession(ctx, sessionID)
	if errors.Is(err, platformcache.ErrMiss) {
		return ViewerSession{}, ErrSessionExpired
	}
	if err != nil {
		return ViewerSession{}, err
	}
	var session ViewerSession
	if err := json.Unmarshal([]byte(encoded), &session); err != nil || session.ID != sessionID {
		return ViewerSession{}, ErrSessionExpired
	}
	now := time.Now().UTC()
	if now.After(session.ExpiresAt) || now.After(session.AbsoluteExpiresAt) {
		_ = s.sessions.DeleteSceneSession(ctx, sessionID)
		return ViewerSession{}, ErrSessionExpired
	}
	session.ExpiresAt = now.Add(ViewerSessionTTL)
	if session.ExpiresAt.After(session.AbsoluteExpiresAt) {
		session.ExpiresAt = session.AbsoluteExpiresAt
	}
	encodedSession, _ := json.Marshal(session)
	if err := s.sessions.PutSceneSession(ctx, session.ID, string(encodedSession), time.Until(session.ExpiresAt)); err != nil {
		return ViewerSession{}, err
	}
	return session, nil
}

func (s *Service) ViewerBootstrap(ctx context.Context, sessionID string) (ViewerBootstrap, error) {
	session, err := s.ResolveViewerSession(ctx, sessionID)
	if err != nil {
		return ViewerBootstrap{}, err
	}
	return ViewerBootstrap{SceneID: session.SceneID, Kind: session.Kind, Revision: session.Revision,
		EntryPath: session.EntryPath, ExpiresAt: session.ExpiresAt, DatapointRefs: session.DatapointRefs}, nil
}

func (s *Service) OpenViewerFile(ctx context.Context, sessionID, filename string) (FileContent, error) {
	session, err := s.ResolveViewerSession(ctx, sessionID)
	if err != nil {
		return FileContent{}, err
	}
	filename, err = NormalizePath(filename, false)
	if err != nil {
		return FileContent{}, err
	}
	item, err := s.repository.GetRevisionFile(ctx, session, filename)
	if err != nil {
		return FileContent{}, err
	}
	if len(item.JSON) > 0 {
		return FileContent{Bytes: item.JSON, Size: int64(len(item.JSON)), ContentType: item.ContentType, Hash: item.Hash}, nil
	}
	object, err := s.objects.Open(ctx, item.ObjectKey)
	if err != nil {
		return FileContent{}, err
	}
	return FileContent{Reader: object.Reader, Size: object.Size, ContentType: item.ContentType, Hash: item.Hash}, nil
}

func (s *Service) ResolveSession(ctx context.Context, actor auth.User, sessionID string) (EditorSession, Scene, error) {
	encoded, err := s.sessions.GetSceneSession(ctx, sessionID)
	if errors.Is(err, platformcache.ErrMiss) {
		return EditorSession{}, Scene{}, ErrSessionExpired
	}
	if err != nil {
		return EditorSession{}, Scene{}, err
	}
	var session EditorSession
	if err := json.Unmarshal([]byte(encoded), &session); err != nil {
		return EditorSession{}, Scene{}, ErrSessionExpired
	}
	if session.ID != sessionID || session.UserID != actor.ID || session.TenantID != actor.TenantID || time.Now().After(session.ExpiresAt) {
		return EditorSession{}, Scene{}, ErrSessionForbidden
	}
	if err := s.requireWrite(ctx, actor, session.ProjectID); err != nil {
		return EditorSession{}, Scene{}, err
	}
	scene, err := s.repository.GetScene(ctx, actor.TenantID, session.ProjectID, session.Kind, session.SceneID)
	if err != nil {
		return EditorSession{}, Scene{}, err
	}
	if scene.Provider != session.Provider {
		return EditorSession{}, Scene{}, ErrSessionForbidden
	}
	return session, scene, nil
}

func (s *Service) ListFiles(ctx context.Context, actor auth.User, sessionID string, filter FileListFilter) ([]FileNode, int64, Scene, error) {
	_, scene, err := s.ResolveSession(ctx, actor, sessionID)
	if err != nil {
		return nil, 0, Scene{}, err
	}
	filter.Directory, err = NormalizePath(filter.Directory, true)
	if err != nil {
		return nil, 0, Scene{}, err
	}
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	items, total, err := s.repository.ListFiles(ctx, scene, filter)
	return items, total, scene, err
}

func (s *Service) OpenFile(ctx context.Context, actor auth.User, sessionID, filename string) (FileContent, error) {
	_, scene, err := s.ResolveSession(ctx, actor, sessionID)
	if err != nil {
		return FileContent{}, err
	}
	filename, err = NormalizePath(filename, false)
	if err != nil {
		return FileContent{}, err
	}
	item, err := s.repository.GetFile(ctx, scene, filename)
	if err != nil {
		return FileContent{}, err
	}
	if len(item.JSON) > 0 {
		return FileContent{Bytes: item.JSON, Size: int64(len(item.JSON)), ContentType: item.ContentType, Hash: item.Hash}, nil
	}
	object, err := s.objects.Open(ctx, item.ObjectKey)
	if err != nil {
		return FileContent{}, err
	}
	return FileContent{Reader: object.Reader, Size: object.Size, ContentType: item.ContentType, Hash: item.Hash}, nil
}

func (s *Service) PutFile(ctx context.Context, actor auth.User, sessionID string, input FileWrite) (int64, error) {
	_, scene, err := s.ResolveSession(ctx, actor, sessionID)
	if err != nil {
		return 0, err
	}
	input.Path, err = NormalizePath(input.Path, false)
	if err != nil {
		return 0, err
	}
	provider, err := s.providers.Get(scene.Provider)
	if err != nil {
		return 0, err
	}
	if err := provider.ValidateDraftFile(scene.Kind, scene.EntryPath, input.Path, input.Content); err != nil {
		return 0, err
	}
	content := input.Content
	isJSON := strings.EqualFold(path.Ext(input.Path), ".json")
	if isJSON {
		var parsed any
		if err := json.Unmarshal(content, &parsed); err != nil {
			return 0, ErrInvalidJSON
		}
		content, _ = json.Marshal(parsed)
	}
	hash := hashBytes(content)
	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = mime.TypeByExtension(strings.ToLower(path.Ext(input.Path)))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	object := contentObject{Hash: hash, Size: int64(len(content)), ContentType: contentType}
	if isJSON {
		object.JSON = content
	} else {
		existing, findErr := s.repository.FindContent(ctx, actor.TenantID, scene.Provider, hash)
		if findErr == nil {
			object = existing
		} else if !errors.Is(findErr, pgx.ErrNoRows) {
			return 0, findErr
		} else {
			object.ObjectKey = objectKey(actor.TenantID, scene.Provider, hash)
			if _, err := s.objects.Put(ctx, object.ObjectKey, bytes.NewReader(content), int64(len(content)), contentType); err != nil {
				return 0, err
			}
		}
	}
	input.Content = nil
	var metadata *Analysis
	if input.Path == scene.EntryPath {
		extracted, extractErr := provider.ExtractMetadata(content)
		if extractErr != nil {
			return 0, extractErr
		}
		metadata = &extracted
	}
	next, err := s.repository.PutFile(ctx, actor, scene, input, object, metadata)
	if err != nil && object.ObjectKey != "" {
		s.cleanupUncommittedObjects(ctx, actor.TenantID, scene.Provider, []string{object.ObjectKey})
	}
	return next, err
}

func (s *Service) Commit(ctx context.Context, actor auth.User, projectID, kind, sceneID string, baseDraft int64, validators ...func(Analysis) error) (Revision, error) {
	if err := s.requireWrite(ctx, actor, projectID); err != nil {
		return Revision{}, err
	}
	scene, err := s.repository.GetScene(ctx, actor.TenantID, projectID, kind, sceneID)
	if err != nil {
		return Revision{}, err
	}
	if scene.DraftVersion != baseDraft {
		return Revision{}, ErrDraftConflict
	}
	provider, err := s.providers.Get(scene.Provider)
	if err != nil {
		return Revision{}, err
	}
	stored, err := s.repository.LoadRevisionFiles(ctx, scene)
	if err != nil {
		return Revision{}, err
	}
	files := make(map[string]ProviderFile, len(stored))
	for filename, item := range stored {
		files[filename] = ProviderFile{Path: filename, ContentType: item.Type, Content: item.Content, Hash: item.Hash}
	}
	analysis, err := provider.Analyze(ctx, scene.Kind, scene.EntryPath, files)
	if err != nil {
		return Revision{}, err
	}
	for _, validate := range validators {
		if validate != nil {
			if err := validate(analysis); err != nil {
				return Revision{}, err
			}
		}
	}
	hasher := sha256.New()
	for _, filename := range analysis.Dependencies {
		item, ok := stored[filename]
		if !ok {
			return Revision{}, fmt.Errorf("%w: %s", ErrDependencyMissing, filename)
		}
		_, _ = io.WriteString(hasher, filename+"\x00"+item.Hash+"\n")
	}
	effectiveContract, err := mergeSceneContracts(scene.PublicContract, analysis.ManagedContract)
	if err != nil {
		return Revision{}, err
	}
	return s.repository.Commit(ctx, actor, scene, baseDraft, provider.Version(), hex.EncodeToString(hasher.Sum(nil)), effectiveContract, analysis.DatapointRefs, analysis.Dependencies)
}

func (s *Service) Export(ctx context.Context, actor auth.User, sessionID string) ([]byte, error) {
	_, scene, err := s.ResolveSession(ctx, actor, sessionID)
	if err != nil {
		return nil, err
	}
	stored, err := s.repository.LoadRevisionFiles(ctx, scene)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for _, item := range sortedRevisionFiles(stored) {
		writer, err := archive.Create(item.Path)
		if err != nil {
			return nil, err
		}
		if len(item.Content) > 0 {
			_, err = writer.Write(item.Content)
		} else {
			object, openErr := s.objects.Open(ctx, item.ObjectKey)
			if openErr != nil {
				return nil, openErr
			}
			_, err = io.Copy(writer, object.Reader)
			_ = object.Reader.Close()
		}
		if err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func (s *Service) Import(ctx context.Context, actor auth.User, sessionID string, archiveContent []byte, baseDraft int64) (int64, error) {
	_, scene, err := s.ResolveSession(ctx, actor, sessionID)
	if err != nil {
		return 0, err
	}
	if scene.DraftVersion != baseDraft {
		return 0, ErrDraftConflict
	}
	if int64(len(archiveContent)) > maxAssetArchiveSize {
		return 0, ErrFileTooLarge
	}
	reader, err := zip.NewReader(bytes.NewReader(archiveContent), int64(len(archiveContent)))
	if err != nil || len(reader.File) == 0 || len(reader.File) > maxAssetArchiveFiles {
		return 0, ErrInvalidArchive
	}
	provider, err := s.providers.Get(scene.Provider)
	if err != nil {
		return 0, err
	}
	files := make(map[string]ProviderFile, len(reader.File))
	var total uint64
	for _, zipped := range reader.File {
		if zipped.FileInfo().Mode()&os.ModeSymlink != 0 {
			return 0, ErrInvalidArchive
		}
		if zipped.FileInfo().IsDir() {
			continue
		}
		filename, err := NormalizePath(strings.TrimSuffix(zipped.Name, "/"), false)
		if err != nil {
			return 0, err
		}
		if _, exists := files[filename]; exists {
			return 0, ErrDuplicateFile
		}
		total += zipped.UncompressedSize64
		if zipped.UncompressedSize64 > uint64(MaxFileSize) || total > uint64(maxAssetArchiveSize) {
			return 0, ErrFileTooLarge
		}
		file, err := zipped.Open()
		if err != nil {
			return 0, ErrInvalidArchive
		}
		content, readErr := io.ReadAll(io.LimitReader(file, MaxFileSize+1))
		_ = file.Close()
		if readErr != nil || int64(len(content)) > MaxFileSize {
			return 0, ErrFileTooLarge
		}
		if err := provider.ValidateDraftFile(scene.Kind, scene.EntryPath, filename, content); err != nil {
			return 0, err
		}
		files[filename] = canonicalProviderFile(filename, content, "")
	}
	if len(files) == 0 {
		return 0, ErrInvalidArchive
	}
	if err := validateProviderFilePaths(files); err != nil {
		return 0, err
	}
	analysis, err := provider.Analyze(ctx, scene.Kind, scene.EntryPath, files)
	if err != nil {
		return 0, err
	}
	stored, uploaded, err := s.storeAssetFiles(ctx, actor, scene.Provider, files)
	if err != nil {
		s.cleanupUncommittedObjects(ctx, actor.TenantID, scene.Provider, uploaded)
		return 0, err
	}
	next, err := s.repository.ReplaceDraftFiles(ctx, actor, scene, baseDraft, stored, analysis)
	if err != nil {
		s.cleanupUncommittedObjects(ctx, actor.TenantID, scene.Provider, uploaded)
	}
	return next, err
}

func (s *Service) EnsurePublishable(ctx context.Context, actor auth.User, projectID string) error {
	if err := s.requireRead(ctx, actor, projectID); err != nil {
		return err
	}
	count, err := s.repository.CountUncommitted(ctx, actor.TenantID, projectID)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrUncommittedDraft
	}
	return nil
}

func (s *Service) CleanupOrphans(ctx context.Context, grace time.Duration) error {
	if grace < time.Minute {
		grace = time.Hour
	}
	items, err := s.repository.OrphanCandidates(ctx, time.Now().Add(-grace), 200)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ObjectKey != "" {
			if err := s.objects.Delete(ctx, item.ObjectKey); err != nil {
				continue
			}
		}
		if err := s.repository.DeleteOrphan(ctx, item.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ValidateRelease(ctx context.Context, actor auth.User, projectID string) (map[string]any, error) {
	if err := s.EnsurePublishable(ctx, actor, projectID); err != nil {
		return nil, err
	}
	_, err := s.repository.GetProvider(ctx)
	if err != nil && !errors.Is(err, ErrProviderNotSet) {
		return nil, err
	}
	scenes, err := s.repository.ReleaseScenes(ctx, actor.TenantID, projectID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"provider": PublicProvider, "engineVersion": PublicVersion, "scenes": scenes}, nil
}

// List 将数据库中的公开契约投影成 AI 上下文格式，不暴露 HT 私有画布 JSON。
func (s *Service) List(ctx context.Context, actor auth.User, projectID string) (scenecontract.Snapshot, error) {
	items, _, err := s.ListScenes(ctx, actor, projectID, ListFilter{Page: 1, Limit: 200, Sort: "sceneId", Order: "asc"})
	if err != nil {
		return scenecontract.Snapshot{}, err
	}
	contracts := make([]scenecontract.Contract, 0, len(items))
	for _, item := range items {
		if item.CurrentRevision == 0 {
			continue
		}
		revision, err := s.repository.GetCurrentRevision(ctx, item)
		if err != nil {
			return scenecontract.Snapshot{}, err
		}
		encoded, _ := json.Marshal(revision.PublicContract)
		var contract scenecontract.Contract
		_ = json.Unmarshal(encoded, &contract)
		contract.ID = item.SceneID
		contract.Kind = item.Kind
		contract.Name = item.Name
		contract.DatapointRefs = append([]string(nil), revision.DatapointRefs...)
		contract.ContractVersion = fmt.Sprintf("%d-%s", revision.Revision, hashBytes(encoded))
		contracts = append(contracts, contract)
	}
	encoded, _ := json.Marshal(contracts)
	return scenecontract.Snapshot{ContractVersion: hashBytes(encoded), Contracts: contracts}, nil
}

func (s *Service) requireRead(ctx context.Context, actor auth.User, projectID string) error {
	item, err := s.projects.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return err
	}
	if !project.CanRead(actor, item) {
		return auth.ErrPermissionDenied
	}
	return auth.RequireCapability(actor, auth.CapabilityProjectRead)
}

func (s *Service) requireWrite(ctx context.Context, actor auth.User, projectID string) error {
	item, err := s.projects.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return err
	}
	return project.RequireCapability(actor, item, auth.CapabilityProjectWrite)
}

func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	return page, limit
}

func hashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func objectKey(tenantID, provider, hash string) string {
	return "scenes/" + tenantID + "/" + provider + "/" + hash[:2] + "/" + hash
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
