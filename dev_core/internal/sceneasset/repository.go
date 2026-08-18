package sceneasset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type contentObject struct {
	ID          string
	Hash        string
	Size        int64
	ContentType string
	JSON        []byte
	ObjectKey   string
}

type revisionFile struct {
	Path      string
	ObjectID  string
	Hash      string
	Content   []byte
	ObjectKey string
	Type      string
	Size      int64
}

type PostgreSQLRepository struct{ pool *pgxpool.Pool }

func NewPostgreSQLRepository(pool *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{pool: pool}
}

func (r *PostgreSQLRepository) GetProvider(ctx context.Context) (ProviderState, error) {
	var state ProviderState
	err := r.pool.QueryRow(ctx, `SELECT provider, provider_version, updated_at FROM scene_provider_state WHERE id = 1`).Scan(&state.Provider, &state.ProviderVersion, &state.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderState{}, ErrProviderNotSet
	}
	return state, err
}

func (r *PostgreSQLRepository) SetProvider(ctx context.Context, actor auth.User, provider, version string) (ProviderState, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ProviderState{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var count int64
	if err := tx.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM scene_documents WHERE deleted_at IS NULL) +
		(SELECT count(*) FROM scene_assets)`).Scan(&count); err != nil {
		return ProviderState{}, err
	}
	var current string
	err = tx.QueryRow(ctx, `SELECT provider FROM scene_provider_state WHERE id = 1 FOR UPDATE`).Scan(&current)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ProviderState{}, err
	}
	if count > 0 && current != "" && current != provider {
		return ProviderState{}, ErrProviderBusy
	}
	var state ProviderState
	err = tx.QueryRow(ctx, `
		INSERT INTO scene_provider_state (id, tenant_id, provider, provider_version, created_by, updated_by)
		VALUES (1, $1, $2, $3, $4, $4)
		ON CONFLICT (id) DO UPDATE SET provider = EXCLUDED.provider, provider_version = EXCLUDED.provider_version,
			updated_by = EXCLUDED.updated_by, updated_at = now()
		RETURNING provider, provider_version, updated_at`, actor.TenantID, provider, version, actor.ID).
		Scan(&state.Provider, &state.ProviderVersion, &state.UpdatedAt)
	if err != nil {
		return ProviderState{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProviderState{}, err
	}
	return state, nil
}

func (r *PostgreSQLRepository) CreateScene(ctx context.Context, actor auth.User, input SceneInput, provider string, initial contentObject) (Scene, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Scene{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	objectID, err := putContentObject(ctx, tx, actor, provider, initial)
	if err != nil {
		return Scene{}, err
	}
	var scene Scene
	contract, _ := json.Marshal(input.PublicContract)
	err = tx.QueryRow(ctx, `
		INSERT INTO scene_documents (tenant_id, project_id, scene_id, kind, name, provider, entry_path,
			public_contract, draft_version, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,1,$9,$9)
		RETURNING id::text, tenant_id::text, project_id::text, scene_id, kind, name, provider, entry_path,
			public_contract, datapoint_refs, current_revision, draft_version, committed_draft_version, created_at, updated_at`,
		actor.TenantID, input.ProjectID, input.SceneID, input.Kind, input.Name, provider, input.EntryPath, contract, actor.ID).
		Scan(&scene.ID, &scene.TenantID, &scene.ProjectID, &scene.SceneID, &scene.Kind, &scene.Name, &scene.Provider,
			&scene.EntryPath, &contract, &scene.DatapointRefs, &scene.CurrentRevision, &scene.DraftVersion,
			&scene.CommittedDraftVersion, &scene.CreatedAt, &scene.UpdatedAt)
	if err != nil {
		return Scene{}, mapDatabaseError(err)
	}
	if err := json.Unmarshal(contract, &scene.PublicContract); err != nil {
		return Scene{}, err
	}
	if err := ensureDirectories(ctx, tx, actor, scene, input.EntryPath); err != nil {
		return Scene{}, err
	}
	if _, err := tx.Exec(ctx, `
			INSERT INTO scene_file_nodes (tenant_id, project_id, scene_document_id, provider, logical_path, parent_path, node_type,
				content_object_id, content_hash, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,'file',$7,$8,$9,$9)
			ON CONFLICT (scene_document_id, logical_path) DO NOTHING`, actor.TenantID, input.ProjectID, scene.ID, provider,
		input.EntryPath, parentPath(input.EntryPath), objectID, initial.Hash, actor.ID); err != nil {
		return Scene{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Scene{}, err
	}
	return scene, nil
}

func (r *PostgreSQLRepository) ListScenes(ctx context.Context, tenantID, projectID string, filter ListFilter) ([]Scene, int64, error) {
	conditions := []string{"tenant_id = $1", "project_id = $2", "deleted_at IS NULL"}
	args := []any{tenantID, projectID}
	if filter.Kind != "" {
		args = append(args, filter.Kind)
		conditions = append(conditions, fmt.Sprintf("kind = $%d", len(args)))
	}
	if filter.Keyword != "" {
		args = append(args, "%"+filter.Keyword+"%")
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR scene_id ILIKE $%d)", len(args), len(args)))
	}
	where := strings.Join(conditions, " AND ")
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM scene_documents WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	sortColumn := map[string]string{"name": "name", "sceneId": "scene_id", "createdAt": "created_at", "updatedAt": "updated_at"}[filter.Sort]
	if sortColumn == "" {
		sortColumn = "updated_at"
	}
	order := "DESC"
	if strings.EqualFold(filter.Order, "asc") {
		order = "ASC"
	}
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	query := `SELECT id::text, tenant_id::text, project_id::text, scene_id, kind, name, provider, entry_path,
		public_contract, datapoint_refs, current_revision, draft_version, committed_draft_version, created_at, updated_at
		FROM scene_documents WHERE ` + where + ` ORDER BY ` + sortColumn + ` ` + order + `, id LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []Scene{}
	for rows.Next() {
		scene, err := scanScene(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, scene)
	}
	return items, total, rows.Err()
}

func (r *PostgreSQLRepository) GetScene(ctx context.Context, tenantID, projectID, kind, sceneID string) (Scene, error) {
	row := r.pool.QueryRow(ctx, `SELECT id::text, tenant_id::text, project_id::text, scene_id, kind, name, provider, entry_path,
		public_contract, datapoint_refs, current_revision, draft_version, committed_draft_version, created_at, updated_at
		FROM scene_documents WHERE tenant_id=$1 AND project_id=$2 AND kind=$3 AND scene_id=$4 AND deleted_at IS NULL`, tenantID, projectID, kind, sceneID)
	scene, err := scanScene(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Scene{}, ErrNotFound
	}
	return scene, err
}

func (r *PostgreSQLRepository) UpdateScene(ctx context.Context, actor auth.User, scene Scene, patch ScenePatch) (Scene, error) {
	name := scene.Name
	if patch.Name != nil {
		name = strings.TrimSpace(*patch.Name)
	}
	contract := scene.PublicContract
	if patch.PublicContract != nil {
		contract = *patch.PublicContract
	}
	contractJSON, _ := json.Marshal(contract)
	row := r.pool.QueryRow(ctx, `UPDATE scene_documents SET name=$1, public_contract=$2, updated_by=$3, updated_at=now()
		WHERE id=$4 AND tenant_id=$5 AND deleted_at IS NULL
		RETURNING id::text, tenant_id::text, project_id::text, scene_id, kind, name, provider, entry_path,
		public_contract, datapoint_refs, current_revision, draft_version, committed_draft_version, created_at, updated_at`,
		name, contractJSON, actor.ID, scene.ID, actor.TenantID)
	updated, err := scanScene(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Scene{}, ErrNotFound
	}
	return updated, err
}

func (r *PostgreSQLRepository) DeleteScene(ctx context.Context, actor auth.User, scene Scene) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `UPDATE scene_documents SET deleted_at=now(), updated_by=$1, updated_at=now()
		WHERE id=$2 AND tenant_id=$3 AND deleted_at IS NULL`, actor.ID, scene.ID, actor.TenantID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE scene_file_nodes SET deleted_at=now(), updated_by=$1, updated_at=now()
		WHERE scene_document_id=$2 AND deleted_at IS NULL`, actor.ID, scene.ID); err != nil {
		return err
	}
	if err := markUnreferencedObjects(ctx, tx, actor.TenantID, scene.Provider); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgreSQLRepository) FindContent(ctx context.Context, tenantID, provider, hash string) (contentObject, error) {
	var item contentObject
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT id::text, content_hash, content_size, content_type, json_content, COALESCE(object_key,'')
		FROM scene_content_objects WHERE tenant_id=$1 AND provider=$2 AND content_hash=$3`, tenantID, provider, hash).
		Scan(&item.ID, &item.Hash, &item.Size, &item.ContentType, &raw, &item.ObjectKey)
	item.JSON = raw
	return item, err
}

func (r *PostgreSQLRepository) PutFile(ctx context.Context, actor auth.User, scene Scene, file FileWrite, content contentObject) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	objectID, err := putContentObject(ctx, tx, actor, scene.Provider, content)
	if err != nil {
		return 0, err
	}
	var nextVersion int64
	err = tx.QueryRow(ctx, `UPDATE scene_documents SET draft_version=draft_version+1, updated_by=$1, updated_at=now()
		WHERE id=$2 AND tenant_id=$3 AND deleted_at IS NULL AND draft_version=$4 RETURNING draft_version`,
		actor.ID, scene.ID, actor.TenantID, file.BaseDraftVersion).Scan(&nextVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrDraftConflict
	}
	if err != nil {
		return 0, err
	}
	if err := ensureDirectories(ctx, tx, actor, scene, file.Path); err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO scene_file_nodes (tenant_id, project_id, scene_document_id, provider, logical_path, parent_path,
		node_type, content_object_id, content_hash, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,'file',$7,$8,$9,$9)
		ON CONFLICT (scene_document_id, logical_path) DO UPDATE SET parent_path=EXCLUDED.parent_path,
			node_type='file', content_object_id=EXCLUDED.content_object_id, content_hash=EXCLUDED.content_hash,
			deleted_at=NULL, updated_by=EXCLUDED.updated_by, updated_at=now()`, actor.TenantID, scene.ProjectID, scene.ID,
		scene.Provider, file.Path, parentPath(file.Path), objectID, content.Hash, actor.ID)
	if err != nil {
		return 0, err
	}
	if err := markUnreferencedObjects(ctx, tx, actor.TenantID, scene.Provider); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return nextVersion, nil
}

// ReplaceDraftFiles 在一个事务中替换场景草稿文件，确保 ZIP 任一文件失败时草稿保持不变。
func (r *PostgreSQLRepository) ReplaceDraftFiles(ctx context.Context, actor auth.User, scene Scene, baseDraft int64, files map[string]contentObject) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var next int64
	err = tx.QueryRow(ctx, `UPDATE scene_documents SET draft_version=draft_version+1, updated_by=$1, updated_at=now()
		WHERE id=$2 AND tenant_id=$3 AND deleted_at IS NULL AND draft_version=$4 RETURNING draft_version`,
		actor.ID, scene.ID, actor.TenantID, baseDraft).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrDraftConflict
	}
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `UPDATE scene_file_nodes SET deleted_at=now(), updated_by=$1, updated_at=now()
		WHERE scene_document_id=$2 AND deleted_at IS NULL`, actor.ID, scene.ID); err != nil {
		return 0, err
	}
	paths := make([]string, 0, len(files))
	for filename := range files {
		paths = append(paths, filename)
	}
	sort.Strings(paths)
	for _, filename := range paths {
		item := files[filename]
		objectID, err := putContentObject(ctx, tx, actor, scene.Provider, item)
		if err != nil {
			return 0, err
		}
		if err := ensureDirectories(ctx, tx, actor, scene, filename); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO scene_file_nodes (tenant_id, project_id, scene_document_id, provider,
			logical_path, parent_path, node_type, content_object_id, content_hash, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,'file',$7,$8,$9,$9)
			ON CONFLICT (scene_document_id, logical_path) DO UPDATE SET parent_path=EXCLUDED.parent_path,
			node_type='file', content_object_id=EXCLUDED.content_object_id, content_hash=EXCLUDED.content_hash,
			deleted_at=NULL, updated_by=EXCLUDED.updated_by, updated_at=now()`, actor.TenantID, scene.ProjectID,
			scene.ID, scene.Provider, filename, parentPath(filename), objectID, item.Hash, actor.ID); err != nil {
			return 0, err
		}
	}
	rows, err := tx.Query(ctx, `SELECT generation_id::text, mount_path FROM scene_asset_bindings
		WHERE scene_document_id=$1 ORDER BY mount_path`, scene.ID)
	if err != nil {
		return 0, err
	}
	type binding struct{ generationID, mountPath string }
	bindings := []binding{}
	for rows.Next() {
		var item binding
		if err := rows.Scan(&item.generationID, &item.mountPath); err != nil {
			rows.Close()
			return 0, err
		}
		bindings = append(bindings, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	for _, item := range bindings {
		if err := materializeAssetGeneration(ctx, tx, actor, scene, item.generationID, item.mountPath); err != nil {
			return 0, err
		}
	}
	if err := markUnreferencedObjects(ctx, tx, actor.TenantID, scene.Provider); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return next, nil
}

func (r *PostgreSQLRepository) ListFiles(ctx context.Context, scene Scene, filter FileListFilter) ([]FileNode, int64, error) {
	conditions := []string{"n.tenant_id=$1", "n.scene_document_id=$2", "n.provider=$3", "n.parent_path=$4", "n.deleted_at IS NULL"}
	args := []any{scene.TenantID, scene.ID, scene.Provider, filter.Directory}
	if filter.Keyword != "" {
		args = append(args, "%"+filter.Keyword+"%")
		conditions = append(conditions, fmt.Sprintf("n.logical_path ILIKE $%d", len(args)))
	}
	where := strings.Join(conditions, " AND ")
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM scene_file_nodes n WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	sortColumn := map[string]string{"name": "n.logical_path", "size": "o.content_size", "updatedAt": "n.updated_at"}[filter.Sort]
	if sortColumn == "" {
		sortColumn = "n.logical_path"
	}
	order := "ASC"
	if strings.EqualFold(filter.Order, "desc") {
		order = "DESC"
	}
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, `SELECT n.logical_path, n.parent_path, n.node_type, COALESCE(n.content_hash,''),
		COALESCE(o.content_size,0), COALESCE(o.content_type,''), n.updated_at
		FROM scene_file_nodes n LEFT JOIN scene_content_objects o ON o.id=n.content_object_id
		WHERE `+where+` ORDER BY `+sortColumn+` `+order+`, n.id LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []FileNode{}
	for rows.Next() {
		var item FileNode
		if err := rows.Scan(&item.Path, &item.ParentPath, &item.Type, &item.ContentHash, &item.Size, &item.ContentType, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		parts := strings.Split(item.Path, "/")
		item.Name = parts[len(parts)-1]
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *PostgreSQLRepository) GetFile(ctx context.Context, scene Scene, filename string) (contentObject, error) {
	var item contentObject
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT o.id::text, o.content_hash, o.content_size, o.content_type, o.json_content, COALESCE(o.object_key,'')
		FROM scene_file_nodes n JOIN scene_content_objects o ON o.id=n.content_object_id
		WHERE n.tenant_id=$1 AND n.scene_document_id=$2 AND n.provider=$3 AND n.logical_path=$4 AND n.node_type='file' AND n.deleted_at IS NULL`,
		scene.TenantID, scene.ID, scene.Provider, filename).Scan(&item.ID, &item.Hash, &item.Size, &item.ContentType, &raw, &item.ObjectKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return contentObject{}, ErrNotFound
	}
	item.JSON = raw
	return item, err
}

func (r *PostgreSQLRepository) LoadRevisionFiles(ctx context.Context, scene Scene) (map[string]revisionFile, error) {
	rows, err := r.pool.Query(ctx, `SELECT n.logical_path, o.id::text, o.content_hash, o.json_content,
		COALESCE(o.object_key,''), o.content_type, o.content_size FROM scene_file_nodes n
		JOIN scene_content_objects o ON o.id=n.content_object_id WHERE n.tenant_id=$1 AND n.scene_document_id=$2
		AND n.provider=$3 AND n.node_type='file' AND n.deleted_at IS NULL`, scene.TenantID, scene.ID, scene.Provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	files := map[string]revisionFile{}
	for rows.Next() {
		var file revisionFile
		if err := rows.Scan(&file.Path, &file.ObjectID, &file.Hash, &file.Content, &file.ObjectKey, &file.Type, &file.Size); err != nil {
			return nil, err
		}
		files[file.Path] = file
	}
	return files, rows.Err()
}

func (r *PostgreSQLRepository) Commit(ctx context.Context, actor auth.User, scene Scene, baseDraft int64, providerVersion, rootHash string, refs []string, dependencies []string) (Revision, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Revision{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var currentDraft, currentRevision int64
	if err := tx.QueryRow(ctx, `SELECT draft_version, current_revision FROM scene_documents WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL FOR UPDATE`, scene.ID, actor.TenantID).Scan(&currentDraft, &currentRevision); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Revision{}, ErrNotFound
		}
		return Revision{}, err
	}
	if currentDraft != baseDraft {
		return Revision{}, ErrDraftConflict
	}
	refJSON, _ := json.Marshal(refs)
	contractJSON, _ := json.Marshal(scene.PublicContract)
	revisionID := uuid.NewString()
	nextRevision := currentRevision + 1
	var created time.Time
	err = tx.QueryRow(ctx, `INSERT INTO scene_revisions (id, tenant_id, project_id, scene_document_id, revision, draft_version,
		provider, provider_version, entry_path, public_contract, datapoint_refs, root_hash, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13) RETURNING created_at`, revisionID, actor.TenantID,
		scene.ProjectID, scene.ID, nextRevision, currentDraft, scene.Provider, providerVersion, scene.EntryPath,
		contractJSON, refJSON, rootHash, actor.ID).Scan(&created)
	if err != nil {
		return Revision{}, err
	}
	for _, filename := range dependencies {
		command, err := tx.Exec(ctx, `INSERT INTO scene_revision_files (tenant_id, project_id, scene_document_id, revision_id,
			logical_path, content_object_id, content_hash, created_by, updated_by)
			SELECT $1,$2,$3,$4,n.logical_path,n.content_object_id,n.content_hash,$5,$5 FROM scene_file_nodes n
			WHERE n.tenant_id=$1 AND n.scene_document_id=$3 AND n.provider=$6 AND n.logical_path=$7 AND n.deleted_at IS NULL`,
			actor.TenantID, scene.ProjectID, scene.ID, revisionID, actor.ID, scene.Provider, filename)
		if err != nil {
			return Revision{}, err
		}
		if command.RowsAffected() != 1 {
			return Revision{}, fmt.Errorf("%w: %s", ErrDependencyMissing, filename)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO scene_revision_assets (tenant_id, project_id, scene_document_id,
		revision_id, asset_id, generation_id, mount_path, created_by, updated_by)
		SELECT tenant_id, project_id, scene_document_id, $1, asset_id, generation_id, mount_path, $2, $2
		FROM scene_asset_bindings WHERE scene_document_id=$3`, revisionID, actor.ID, scene.ID); err != nil {
		return Revision{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE scene_documents SET current_revision=$1, committed_draft_version=$2,
		datapoint_refs=$3, updated_by=$4, updated_at=now() WHERE id=$5`, nextRevision, currentDraft, refJSON, actor.ID, scene.ID)
	if err != nil {
		return Revision{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Revision{}, err
	}
	return Revision{Revision: nextRevision, DraftVersion: currentDraft, Provider: scene.Provider, ProviderVersion: providerVersion, EntryPath: scene.EntryPath, RootHash: rootHash, CreatedAt: created}, nil
}

func (r *PostgreSQLRepository) CountUncommitted(ctx context.Context, tenantID, projectID string) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM scene_documents WHERE tenant_id=$1 AND project_id=$2
		AND deleted_at IS NULL AND draft_version<>committed_draft_version`, tenantID, projectID).Scan(&count)
	return count, err
}

func (r *PostgreSQLRepository) ReleaseScenes(ctx context.Context, tenantID, projectID string) ([]map[string]any, error) {
	rows, err := r.pool.Query(ctx, `SELECT d.scene_id, d.kind, r.revision, r.entry_path, r.root_hash,
		r.provider, r.provider_version FROM scene_documents d JOIN scene_revisions r
		ON r.scene_document_id=d.id AND r.revision=d.current_revision
		WHERE d.tenant_id=$1 AND d.project_id=$2 AND d.deleted_at IS NULL ORDER BY d.kind, d.scene_id`, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var sceneID, kind, entry, rootHash, provider, providerVersion string
		var revision int64
		if err := rows.Scan(&sceneID, &kind, &revision, &entry, &rootHash, &provider, &providerVersion); err != nil {
			return nil, err
		}
		assets, err := r.releaseAssets(ctx, tenantID, projectID, sceneID, kind, revision)
		if err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"sceneId": sceneID, "kind": kind, "revision": revision,
			"entry": entry, "digest": rootHash, "provider": PublicProvider, "engineVersion": PublicVersion, "assets": assets})
	}
	return items, rows.Err()
}

func (r *PostgreSQLRepository) releaseAssets(ctx context.Context, tenantID, projectID, sceneID, kind string, revision int64) ([]map[string]any, error) {
	rows, err := r.pool.Query(ctx, `SELECT a.id::text, a.asset_type, g.generation, ra.mount_path, g.root_hash
		FROM scene_revision_assets ra JOIN scene_assets a ON a.id=ra.asset_id
		JOIN scene_asset_generations g ON g.id=ra.generation_id
		JOIN scene_revisions r ON r.id=ra.revision_id JOIN scene_documents d ON d.id=ra.scene_document_id
		WHERE ra.tenant_id=$1 AND ra.project_id=$2 AND d.scene_id=$3 AND d.kind=$4 AND r.revision=$5
		ORDER BY a.asset_type, a.id`, tenantID, projectID, sceneID, kind, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, assetType, mountPath, rootHash string
		var generation int64
		if err := rows.Scan(&id, &assetType, &generation, &mountPath, &rootHash); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"assetId": id, "type": assetType, "generation": generation,
			"mountPath": mountPath, "digest": rootHash})
	}
	return items, rows.Err()
}

func (r *PostgreSQLRepository) OrphanCandidates(ctx context.Context, before time.Time, limit int) ([]contentObject, error) {
	rows, err := r.pool.Query(ctx, `SELECT o.id::text, o.content_hash, o.content_size, o.content_type,
		o.json_content, COALESCE(o.object_key,'') FROM scene_content_objects o
		WHERE o.orphaned_at IS NOT NULL AND o.orphaned_at < $1
		AND NOT EXISTS (SELECT 1 FROM scene_file_nodes n WHERE n.content_object_id=o.id AND n.deleted_at IS NULL)
		AND NOT EXISTS (SELECT 1 FROM scene_revision_files f WHERE f.content_object_id=o.id)
		AND NOT EXISTS (SELECT 1 FROM scene_asset_generation_files f WHERE f.content_object_id=o.id)
		AND NOT EXISTS (SELECT 1 FROM scene_asset_draft_files f WHERE f.content_object_id=o.id)
		AND NOT EXISTS (SELECT 1 FROM scene_asset_generations g WHERE g.thumbnail_content_object_id=o.id)
		ORDER BY o.orphaned_at LIMIT $2`, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []contentObject{}
	for rows.Next() {
		var item contentObject
		if err := rows.Scan(&item.ID, &item.Hash, &item.Size, &item.ContentType, &item.JSON, &item.ObjectKey); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgreSQLRepository) DeleteOrphan(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM scene_content_objects o WHERE o.id=$1 AND o.orphaned_at IS NOT NULL
		AND NOT EXISTS (SELECT 1 FROM scene_file_nodes n WHERE n.content_object_id=o.id AND n.deleted_at IS NULL)
		AND NOT EXISTS (SELECT 1 FROM scene_revision_files f WHERE f.content_object_id=o.id)
		AND NOT EXISTS (SELECT 1 FROM scene_asset_generation_files f WHERE f.content_object_id=o.id)
		AND NOT EXISTS (SELECT 1 FROM scene_asset_draft_files f WHERE f.content_object_id=o.id)
		AND NOT EXISTS (SELECT 1 FROM scene_asset_generations g WHERE g.thumbnail_content_object_id=o.id)`, id)
	return err
}

func putContentObject(ctx context.Context, tx pgx.Tx, actor auth.User, provider string, item contentObject) (string, error) {
	var id string
	var jsonValue any
	if len(item.JSON) > 0 {
		jsonValue = item.JSON
	}
	var objectKey any
	if item.ObjectKey != "" {
		objectKey = item.ObjectKey
	}
	err := tx.QueryRow(ctx, `INSERT INTO scene_content_objects (tenant_id, provider, content_hash, content_size,
		content_type, json_content, object_key, created_by, updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8)
		ON CONFLICT (tenant_id, provider, content_hash) DO UPDATE SET orphaned_at=NULL, updated_by=EXCLUDED.updated_by,
		updated_at=now() RETURNING id::text`, actor.TenantID, provider, item.Hash, item.Size, item.ContentType, jsonValue, objectKey, actor.ID).Scan(&id)
	return id, err
}

func markUnreferencedObjects(ctx context.Context, tx pgx.Tx, tenantID, provider string) error {
	_, err := tx.Exec(ctx, `UPDATE scene_content_objects o SET orphaned_at=COALESCE(orphaned_at, now()), updated_at=now()
		WHERE o.tenant_id=$1 AND o.provider=$2
		AND NOT EXISTS (SELECT 1 FROM scene_file_nodes n WHERE n.content_object_id=o.id AND n.deleted_at IS NULL)
		AND NOT EXISTS (SELECT 1 FROM scene_revision_files f WHERE f.content_object_id=o.id)
		AND NOT EXISTS (SELECT 1 FROM scene_asset_generation_files f WHERE f.content_object_id=o.id)
		AND NOT EXISTS (SELECT 1 FROM scene_asset_draft_files f WHERE f.content_object_id=o.id)
		AND NOT EXISTS (SELECT 1 FROM scene_asset_generations g WHERE g.thumbnail_content_object_id=o.id)`, tenantID, provider)
	return err
}

func ensureDirectories(ctx context.Context, tx pgx.Tx, actor auth.User, scene Scene, filename string) error {
	parts := strings.Split(filename, "/")
	for index := 1; index < len(parts); index++ {
		directory := strings.Join(parts[:index], "/")
		if _, err := tx.Exec(ctx, `INSERT INTO scene_file_nodes (tenant_id, project_id, scene_document_id, provider, logical_path,
			parent_path, node_type, created_by, updated_by) VALUES ($1,$2,$3,$4,$5,$6,'directory',$7,$7)
			ON CONFLICT (scene_document_id, logical_path) DO UPDATE SET deleted_at=NULL,
			updated_by=EXCLUDED.updated_by, updated_at=now()`, actor.TenantID, scene.ProjectID, scene.ID, scene.Provider,
			directory, parentPath(directory), actor.ID); err != nil {
			return err
		}
	}
	return nil
}

type rowScanner interface{ Scan(...any) error }

func scanScene(row rowScanner) (Scene, error) {
	var scene Scene
	var contractJSON, refsJSON []byte
	err := row.Scan(&scene.ID, &scene.TenantID, &scene.ProjectID, &scene.SceneID, &scene.Kind, &scene.Name, &scene.Provider,
		&scene.EntryPath, &contractJSON, &refsJSON, &scene.CurrentRevision, &scene.DraftVersion,
		&scene.CommittedDraftVersion, &scene.CreatedAt, &scene.UpdatedAt)
	if err != nil {
		return Scene{}, err
	}
	if len(contractJSON) > 0 {
		_ = json.Unmarshal(contractJSON, &scene.PublicContract)
	}
	if len(refsJSON) > 0 {
		_ = json.Unmarshal(refsJSON, &scene.DatapointRefs)
	}
	if scene.PublicContract == nil {
		scene.PublicContract = map[string]any{}
	}
	if scene.DatapointRefs == nil {
		scene.DatapointRefs = []string{}
	}
	return scene, nil
}

func parentPath(filename string) string {
	index := strings.LastIndex(filename, "/")
	if index < 0 {
		return ""
	}
	return filename[:index]
}

func mapDatabaseError(err error) error {
	message := err.Error()
	if strings.Contains(message, "unique") || strings.Contains(message, "duplicate") {
		return fmt.Errorf("资源已存在: %w", err)
	}
	return err
}

func sortedRevisionFiles(files map[string]revisionFile) []revisionFile {
	result := make([]revisionFile, 0, len(files))
	for _, file := range files {
		result = append(result, file)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}
