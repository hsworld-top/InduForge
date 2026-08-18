package sceneasset

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/jackc/pgx/v5"
)

func (r *PostgreSQLRepository) CreateAsset(ctx context.Context, actor auth.User, projectID, provider string, input AssetImportInput, analysis AssetAnalysis, rootHash string, files map[string]contentObject) (SceneAsset, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SceneAsset{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	assetID := uuid.NewString()
	var asset SceneAsset
	err = tx.QueryRow(ctx, `INSERT INTO scene_assets (id, tenant_id, project_id, provider, asset_type,
		compatible_kind, name, entry_path, created_by, updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)
		RETURNING id::text, tenant_id::text, project_id::text, provider, asset_type, compatible_kind, name,
		entry_path, current_generation, draft_version, committed_draft_version, archived_at IS NOT NULL, created_at, updated_at`,
		assetID, actor.TenantID, projectID, provider, input.Type, analysis.CompatibleKind, input.Name, analysis.EntryPath, actor.ID).
		Scan(&asset.ID, &asset.TenantID, &asset.ProjectID, &asset.Provider, &asset.Type, &asset.CompatibleKind,
			&asset.Name, &asset.EntryPath, &asset.CurrentGeneration, &asset.DraftVersion,
			&asset.CommittedDraftVersion, &asset.Archived, &asset.CreatedAt, &asset.UpdatedAt)
	if err != nil {
		return SceneAsset{}, mapAssetDatabaseError(err)
	}
	generationID, err := insertAssetGeneration(ctx, tx, actor, asset, 1, analysis, rootHash, files)
	if err != nil {
		return SceneAsset{}, err
	}
	if input.Type == AssetSymbol || input.Type == AssetComponent {
		if err := replaceAssetDraftFiles(ctx, tx, actor, asset, generationID); err != nil {
			return SceneAsset{}, err
		}
	}
	err = tx.QueryRow(ctx, `UPDATE scene_assets SET current_generation=1, draft_version=1,
		committed_draft_version=1, updated_by=$1, updated_at=now() WHERE id=$2
		RETURNING current_generation, draft_version, committed_draft_version, updated_at`, actor.ID, asset.ID).
		Scan(&asset.CurrentGeneration, &asset.DraftVersion, &asset.CommittedDraftVersion, &asset.UpdatedAt)
	if err != nil {
		return SceneAsset{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SceneAsset{}, err
	}
	return asset, nil
}

func (r *PostgreSQLRepository) ReplaceAsset(ctx context.Context, actor auth.User, asset SceneAsset, analysis AssetAnalysis, rootHash string, files map[string]contentObject) (SceneAsset, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SceneAsset{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var current, draft, committed int64
	var archived bool
	err = tx.QueryRow(ctx, `SELECT current_generation, draft_version, committed_draft_version, archived_at IS NOT NULL
		FROM scene_assets WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, asset.ID, actor.TenantID).
		Scan(&current, &draft, &committed, &archived)
	if errors.Is(err, pgx.ErrNoRows) {
		return SceneAsset{}, ErrAssetNotFound
	}
	if err != nil {
		return SceneAsset{}, err
	}
	if archived {
		return SceneAsset{}, ErrAssetArchived
	}
	if draft != committed {
		return SceneAsset{}, ErrDraftConflict
	}
	next := current + 1
	generationID, err := insertAssetGeneration(ctx, tx, actor, asset, next, analysis, rootHash, files)
	if err != nil {
		return SceneAsset{}, err
	}
	if asset.Type == AssetSymbol || asset.Type == AssetComponent {
		if err := replaceAssetDraftFiles(ctx, tx, actor, asset, generationID); err != nil {
			return SceneAsset{}, err
		}
	}
	err = tx.QueryRow(ctx, `UPDATE scene_assets SET current_generation=$1, entry_path=$2,
		draft_version=draft_version+1, committed_draft_version=draft_version+1, updated_by=$3, updated_at=now()
		WHERE id=$4 RETURNING current_generation, draft_version, committed_draft_version, updated_at`,
		next, analysis.EntryPath, actor.ID, asset.ID).
		Scan(&asset.CurrentGeneration, &asset.DraftVersion, &asset.CommittedDraftVersion, &asset.UpdatedAt)
	if err != nil {
		return SceneAsset{}, err
	}
	asset.EntryPath = analysis.EntryPath
	if err := tx.Commit(ctx); err != nil {
		return SceneAsset{}, err
	}
	return asset, nil
}

func insertAssetGeneration(ctx context.Context, tx pgx.Tx, actor auth.User, asset SceneAsset, generation int64, analysis AssetAnalysis, rootHash string, files map[string]contentObject) (string, error) {
	generationID := uuid.NewString()
	manifest, _ := json.Marshal(analysis.Manifest)
	if _, err := tx.Exec(ctx, `INSERT INTO scene_asset_generations (id, tenant_id, project_id, asset_id,
		generation, entry_path, root_hash, manifest, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`, generationID, actor.TenantID, asset.ProjectID,
		asset.ID, generation, analysis.EntryPath, rootHash, manifest, actor.ID); err != nil {
		return "", err
	}
	paths := make([]string, 0, len(files))
	for filename := range files {
		paths = append(paths, filename)
	}
	sort.Strings(paths)
	for _, filename := range paths {
		item := files[filename]
		objectID, err := putContentObject(ctx, tx, actor, asset.Provider, item)
		if err != nil {
			return "", err
		}
		role := "dependency"
		if filename == analysis.EntryPath {
			role = "entry"
		}
		if _, err := tx.Exec(ctx, `INSERT INTO scene_asset_generation_files (tenant_id, project_id, asset_id,
			generation_id, logical_path, file_role, content_object_id, content_hash, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`, actor.TenantID, asset.ProjectID, asset.ID,
			generationID, filename, role, objectID, item.Hash, actor.ID); err != nil {
			return "", err
		}
	}
	return generationID, nil
}

func replaceAssetDraftFiles(ctx context.Context, tx pgx.Tx, actor auth.User, asset SceneAsset, generationID string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM scene_asset_draft_files WHERE asset_id=$1`, asset.ID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO scene_asset_draft_files (tenant_id, project_id, asset_id, logical_path,
		content_object_id, content_hash, created_by, updated_by)
		SELECT tenant_id, project_id, asset_id, logical_path, content_object_id, content_hash, $1, $1
		FROM scene_asset_generation_files WHERE generation_id=$2`, actor.ID, generationID)
	return err
}

func (r *PostgreSQLRepository) ListAssets(ctx context.Context, tenantID, projectID, provider, sceneDocumentID string, filter AssetListFilter) ([]SceneAsset, int64, error) {
	conditions := []string{"a.tenant_id=$1", "a.project_id=$2", "a.provider=$3"}
	args := []any{tenantID, projectID, provider}
	if filter.Archived {
		conditions = append(conditions, "a.archived_at IS NOT NULL")
	} else {
		conditions = append(conditions, "a.archived_at IS NULL")
	}
	if filter.Type != "" {
		args = append(args, filter.Type)
		conditions = append(conditions, fmt.Sprintf("a.asset_type=$%d", len(args)))
	}
	if filter.Kind != "" {
		args = append(args, filter.Kind)
		conditions = append(conditions, fmt.Sprintf("a.compatible_kind IN ($%d,'both')", len(args)))
	}
	if filter.Keyword != "" {
		args = append(args, "%"+filter.Keyword+"%")
		conditions = append(conditions, fmt.Sprintf("a.name ILIKE $%d", len(args)))
	}
	where := strings.Join(conditions, " AND ")
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM scene_assets a WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	sortColumn := map[string]string{"name": "a.name", "type": "a.asset_type", "createdAt": "a.created_at", "updatedAt": "a.updated_at"}[filter.Sort]
	if sortColumn == "" {
		sortColumn = "a.updated_at"
	}
	order := "DESC"
	if strings.EqualFold(filter.Order, "asc") {
		order = "ASC"
	}
	args = append(args, sceneDocumentID, filter.Limit, (filter.Page-1)*filter.Limit)
	sceneArg, limitArg, offsetArg := len(args)-2, len(args)-1, len(args)
	rows, err := r.pool.Query(ctx, `SELECT a.id::text, a.tenant_id::text, a.project_id::text, a.provider,
		a.asset_type, a.compatible_kind, a.name, a.entry_path, a.current_generation, a.draft_version,
		a.committed_draft_version, a.archived_at IS NOT NULL, b.id IS NOT NULL,
		COALESCE(bg.generation <> a.current_generation, false), a.created_at, a.updated_at
		FROM scene_assets a LEFT JOIN scene_asset_bindings b ON b.asset_id=a.id AND b.scene_document_id=NULLIF($`+fmt.Sprint(sceneArg)+`, '')::uuid
		LEFT JOIN scene_asset_generations bg ON bg.id=b.generation_id WHERE `+where+`
		ORDER BY `+sortColumn+` `+order+`, a.id LIMIT $`+fmt.Sprint(limitArg)+` OFFSET $`+fmt.Sprint(offsetArg), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []SceneAsset{}
	for rows.Next() {
		var item SceneAsset
		if err := rows.Scan(&item.ID, &item.TenantID, &item.ProjectID, &item.Provider, &item.Type,
			&item.CompatibleKind, &item.Name, &item.EntryPath, &item.CurrentGeneration, &item.DraftVersion,
			&item.CommittedDraftVersion, &item.Archived, &item.Bound, &item.UpdateAvailable,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *PostgreSQLRepository) GetAsset(ctx context.Context, tenantID, projectID, assetID string) (SceneAsset, error) {
	var asset SceneAsset
	err := r.pool.QueryRow(ctx, `SELECT id::text, tenant_id::text, project_id::text, provider, asset_type,
		compatible_kind, name, entry_path, current_generation, draft_version, committed_draft_version,
		archived_at IS NOT NULL, created_at, updated_at FROM scene_assets
		WHERE id=$1 AND tenant_id=$2 AND project_id=$3`, assetID, tenantID, projectID).
		Scan(&asset.ID, &asset.TenantID, &asset.ProjectID, &asset.Provider, &asset.Type, &asset.CompatibleKind,
			&asset.Name, &asset.EntryPath, &asset.CurrentGeneration, &asset.DraftVersion,
			&asset.CommittedDraftVersion, &asset.Archived, &asset.CreatedAt, &asset.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SceneAsset{}, ErrAssetNotFound
	}
	return asset, err
}

func (r *PostgreSQLRepository) UpdateAsset(ctx context.Context, actor auth.User, asset SceneAsset, name string) (SceneAsset, error) {
	err := r.pool.QueryRow(ctx, `UPDATE scene_assets SET name=$1, updated_by=$2, updated_at=now()
		WHERE id=$3 AND tenant_id=$4 AND archived_at IS NULL RETURNING name, updated_at`,
		name, actor.ID, asset.ID, actor.TenantID).Scan(&asset.Name, &asset.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SceneAsset{}, ErrAssetNotFound
	}
	if err != nil {
		return SceneAsset{}, mapAssetDatabaseError(err)
	}
	return asset, nil
}

func (r *PostgreSQLRepository) ArchiveAsset(ctx context.Context, actor auth.User, asset SceneAsset) error {
	command, err := r.pool.Exec(ctx, `UPDATE scene_assets SET archived_at=now(), updated_by=$1, updated_at=now()
		WHERE id=$2 AND tenant_id=$3 AND archived_at IS NULL`, actor.ID, asset.ID, actor.TenantID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrAssetNotFound
	}
	return nil
}

func (r *PostgreSQLRepository) LoadAssetGenerationFiles(ctx context.Context, asset SceneAsset, generation int64) (AssetGeneration, map[string]revisionFile, error) {
	var result AssetGeneration
	var manifest []byte
	err := r.pool.QueryRow(ctx, `SELECT id::text, asset_id::text, generation, entry_path, root_hash, manifest, created_at
		FROM scene_asset_generations WHERE asset_id=$1 AND generation=$2`, asset.ID, generation).
		Scan(&result.ID, &result.AssetID, &result.Generation, &result.EntryPath, &result.RootHash, &manifest, &result.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AssetGeneration{}, nil, ErrAssetNotFound
	}
	if err != nil {
		return AssetGeneration{}, nil, err
	}
	_ = json.Unmarshal(manifest, &result.Manifest)
	rows, err := r.pool.Query(ctx, `SELECT f.logical_path, o.id::text, o.content_hash, o.json_content,
		COALESCE(o.object_key,''), o.content_type, o.content_size FROM scene_asset_generation_files f
		JOIN scene_content_objects o ON o.id=f.content_object_id WHERE f.generation_id=$1`, result.ID)
	if err != nil {
		return AssetGeneration{}, nil, err
	}
	defer rows.Close()
	files := map[string]revisionFile{}
	for rows.Next() {
		var item revisionFile
		if err := rows.Scan(&item.Path, &item.ObjectID, &item.Hash, &item.Content, &item.ObjectKey, &item.Type, &item.Size); err != nil {
			return AssetGeneration{}, nil, err
		}
		files[item.Path] = item
	}
	return result, files, rows.Err()
}

func (r *PostgreSQLRepository) ApplyAssetAction(ctx context.Context, actor auth.User, scene Scene, action AssetAction, mountPath string) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var next int64
	err = tx.QueryRow(ctx, `UPDATE scene_documents SET draft_version=draft_version+1, updated_by=$1, updated_at=now()
		WHERE id=$2 AND tenant_id=$3 AND draft_version=$4 AND deleted_at IS NULL RETURNING draft_version`,
		actor.ID, scene.ID, actor.TenantID, action.BaseDraftVersion).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrDraftConflict
	}
	if err != nil {
		return 0, err
	}
	if action.Action == "detach" {
		var existingMount string
		err = tx.QueryRow(ctx, `DELETE FROM scene_asset_bindings WHERE scene_document_id=$1 AND asset_id=$2
			RETURNING mount_path`, scene.ID, action.AssetID).Scan(&existingMount)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrAssetNotFound
		}
		if err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `UPDATE scene_file_nodes SET deleted_at=now(), updated_by=$1, updated_at=now()
			WHERE scene_document_id=$2 AND deleted_at IS NULL AND (logical_path=$3 OR logical_path LIKE $3 || '/%')`,
			actor.ID, scene.ID, existingMount); err != nil {
			return 0, err
		}
		if err := markUnreferencedObjects(ctx, tx, actor.TenantID, scene.Provider); err != nil {
			return 0, err
		}
		if err := tx.Commit(ctx); err != nil {
			return 0, err
		}
		return next, nil
	}
	var assetType AssetType
	var compatibleKind string
	var generationID string
	var archived bool
	err = tx.QueryRow(ctx, `SELECT a.asset_type, a.compatible_kind, g.id::text, a.archived_at IS NOT NULL
		FROM scene_assets a JOIN scene_asset_generations g ON g.asset_id=a.id AND g.generation=a.current_generation
		WHERE a.id=$1 AND a.tenant_id=$2 AND a.project_id=$3 AND a.provider=$4`,
		action.AssetID, actor.TenantID, scene.ProjectID, scene.Provider).Scan(&assetType, &compatibleKind, &generationID, &archived)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrAssetNotFound
	}
	if err != nil {
		return 0, err
	}
	if archived && action.Action == "attach" {
		return 0, ErrAssetArchived
	}
	if compatibleKind != "both" && compatibleKind != scene.Kind {
		return 0, ErrInvalidAssetType
	}
	if action.Action == "attach" {
		command, err := tx.Exec(ctx, `INSERT INTO scene_asset_bindings (tenant_id, project_id, scene_document_id,
			asset_id, generation_id, mount_path, created_by, updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$7)
			ON CONFLICT (scene_document_id, asset_id) DO NOTHING`, actor.TenantID, scene.ProjectID, scene.ID,
			action.AssetID, generationID, mountPath, actor.ID)
		if err != nil {
			return 0, mapAssetDatabaseError(err)
		}
		if command.RowsAffected() == 0 {
			return 0, ErrAssetConflict
		}
	} else if action.Action == "update" {
		var existingMount string
		err := tx.QueryRow(ctx, `UPDATE scene_asset_bindings SET generation_id=$1, updated_by=$2, updated_at=now()
			WHERE scene_document_id=$3 AND asset_id=$4 RETURNING mount_path`, generationID, actor.ID, scene.ID, action.AssetID).
			Scan(&existingMount)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrAssetNotFound
		}
		if err != nil {
			return 0, err
		}
		mountPath = existingMount
		if _, err := tx.Exec(ctx, `UPDATE scene_file_nodes SET deleted_at=now(), updated_by=$1, updated_at=now()
			WHERE scene_document_id=$2 AND deleted_at IS NULL AND (logical_path=$3 OR logical_path LIKE $3 || '/%')`,
			actor.ID, scene.ID, mountPath); err != nil {
			return 0, err
		}
	} else {
		return 0, ErrInvalidPath
	}
	if err := materializeAssetGeneration(ctx, tx, actor, scene, generationID, mountPath); err != nil {
		return 0, err
	}
	if err := markUnreferencedObjects(ctx, tx, actor.TenantID, scene.Provider); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return next, nil
}

func materializeAssetGeneration(ctx context.Context, tx pgx.Tx, actor auth.User, scene Scene, generationID, mountPath string) error {
	rows, err := tx.Query(ctx, `SELECT logical_path, content_object_id::text, content_hash
		FROM scene_asset_generation_files WHERE generation_id=$1 ORDER BY logical_path`, generationID)
	if err != nil {
		return err
	}
	type mountedFile struct{ path, objectID, hash string }
	files := []mountedFile{}
	for rows.Next() {
		var item mountedFile
		if err := rows.Scan(&item.path, &item.objectID, &item.hash); err != nil {
			rows.Close()
			return err
		}
		files = append(files, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range files {
		target := mountPath + "/" + item.path
		if err := ensureDirectories(ctx, tx, actor, scene, target); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO scene_file_nodes (tenant_id, project_id, scene_document_id, provider,
			logical_path, parent_path, node_type, content_object_id, content_hash, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,'file',$7,$8,$9,$9)
			ON CONFLICT (scene_document_id, logical_path) DO UPDATE SET content_object_id=EXCLUDED.content_object_id,
			content_hash=EXCLUDED.content_hash, deleted_at=NULL, updated_by=EXCLUDED.updated_by, updated_at=now()`,
			actor.TenantID, scene.ProjectID, scene.ID, scene.Provider, target, parentPath(target), item.objectID, item.hash, actor.ID)
		if err != nil {
			return mapAssetDatabaseError(err)
		}
	}
	return nil
}

func (r *PostgreSQLRepository) ListAssetBindings(ctx context.Context, scene Scene) ([]AssetBinding, error) {
	rows, err := r.pool.Query(ctx, `SELECT b.id::text, a.id::text, a.name, a.asset_type, b.mount_path,
		bg.generation<>a.current_generation FROM scene_asset_bindings b JOIN scene_assets a ON a.id=b.asset_id
		JOIN scene_asset_generations bg ON bg.id=b.generation_id WHERE b.scene_document_id=$1 ORDER BY a.asset_type, a.name`, scene.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AssetBinding{}
	for rows.Next() {
		var item AssetBinding
		if err := rows.Scan(&item.ID, &item.AssetID, &item.Name, &item.Type, &item.MountPath, &item.UpdateAvailable); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgreSQLRepository) ListDependencies(ctx context.Context, scene Scene, filter FileListFilter) ([]DependencyNode, int64, error) {
	conditions := []string{"n.tenant_id=$1", "n.scene_document_id=$2", "n.provider=$3", "n.node_type='file'", "n.deleted_at IS NULL"}
	args := []any{scene.TenantID, scene.ID, scene.Provider}
	if filter.Directory != "" {
		args = append(args, filter.Directory+"/%")
		conditions = append(conditions, fmt.Sprintf("n.logical_path LIKE $%d", len(args)))
	}
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
	rows, err := r.pool.Query(ctx, `SELECT n.logical_path, n.content_hash, o.content_size, o.content_type,
		COALESCE(source.asset_id::text,''), COALESCE(source.asset_name,'')
		FROM scene_file_nodes n JOIN scene_content_objects o ON o.id=n.content_object_id
		LEFT JOIN LATERAL (
			SELECT b.asset_id, a.name AS asset_name FROM scene_asset_bindings b
			JOIN scene_assets a ON a.id=b.asset_id
			WHERE b.scene_document_id=n.scene_document_id
			AND (n.logical_path=b.mount_path OR n.logical_path LIKE b.mount_path || '/%')
			ORDER BY char_length(b.mount_path) DESC LIMIT 1
		) source ON true
		WHERE `+where+` ORDER BY `+sortColumn+` `+order+`, n.id
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []DependencyNode{}
	for rows.Next() {
		var item DependencyNode
		if err := rows.Scan(&item.Path, &item.ContentHash, &item.Size, &item.ContentType,
			&item.SourceAssetID, &item.SourceAssetName); err != nil {
			return nil, 0, err
		}
		item.Name = path.Base(item.Path)
		item.SourceType = "scene"
		if item.SourceAssetID != "" {
			item.SourceType = "asset"
		}
		item.Status = "available"
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *PostgreSQLRepository) ListAssetDraftFiles(ctx context.Context, asset SceneAsset, filter FileListFilter) ([]FileNode, int64, error) {
	conditions := []string{"f.asset_id=$1"}
	args := []any{asset.ID}
	if filter.Directory != "" {
		args = append(args, filter.Directory+"/%")
		conditions = append(conditions, fmt.Sprintf("f.logical_path LIKE $%d", len(args)))
	}
	if filter.Keyword != "" {
		args = append(args, "%"+filter.Keyword+"%")
		conditions = append(conditions, fmt.Sprintf("f.logical_path ILIKE $%d", len(args)))
	}
	where := strings.Join(conditions, " AND ")
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM scene_asset_draft_files f WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, `SELECT f.logical_path, o.content_hash, o.content_size, o.content_type, f.updated_at
		FROM scene_asset_draft_files f JOIN scene_content_objects o ON o.id=f.content_object_id WHERE `+where+`
		ORDER BY f.logical_path LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []FileNode{}
	for rows.Next() {
		var item FileNode
		if err := rows.Scan(&item.Path, &item.ContentHash, &item.Size, &item.ContentType, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		item.ParentPath = parentPath(item.Path)
		item.Type = "file"
		parts := strings.Split(item.Path, "/")
		item.Name = parts[len(parts)-1]
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *PostgreSQLRepository) GetAssetDraftFile(ctx context.Context, asset SceneAsset, filename string) (contentObject, error) {
	var item contentObject
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT o.id::text, o.content_hash, o.content_size, o.content_type,
		o.json_content, COALESCE(o.object_key,'') FROM scene_asset_draft_files f
		JOIN scene_content_objects o ON o.id=f.content_object_id WHERE f.asset_id=$1 AND f.logical_path=$2`,
		asset.ID, filename).Scan(&item.ID, &item.Hash, &item.Size, &item.ContentType, &raw, &item.ObjectKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return contentObject{}, ErrNotFound
	}
	item.JSON = raw
	return item, err
}

func (r *PostgreSQLRepository) PutAssetDraftFile(ctx context.Context, actor auth.User, asset SceneAsset, file FileWrite, content contentObject) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	objectID, err := putContentObject(ctx, tx, actor, asset.Provider, content)
	if err != nil {
		return 0, err
	}
	var next int64
	err = tx.QueryRow(ctx, `UPDATE scene_assets SET draft_version=draft_version+1, updated_by=$1, updated_at=now()
		WHERE id=$2 AND tenant_id=$3 AND draft_version=$4 AND archived_at IS NULL RETURNING draft_version`,
		actor.ID, asset.ID, actor.TenantID, file.BaseDraftVersion).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrDraftConflict
	}
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO scene_asset_draft_files (tenant_id, project_id, asset_id, logical_path,
		content_object_id, content_hash, created_by, updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$7)
		ON CONFLICT (asset_id, logical_path) DO UPDATE SET content_object_id=EXCLUDED.content_object_id,
		content_hash=EXCLUDED.content_hash, updated_by=EXCLUDED.updated_by, updated_at=now()`,
		actor.TenantID, asset.ProjectID, asset.ID, file.Path, objectID, content.Hash, actor.ID)
	if err != nil {
		return 0, err
	}
	if err := markUnreferencedObjects(ctx, tx, actor.TenantID, asset.Provider); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return next, nil
}

func (r *PostgreSQLRepository) LoadAssetDraft(ctx context.Context, asset SceneAsset) (map[string]revisionFile, error) {
	rows, err := r.pool.Query(ctx, `SELECT f.logical_path, o.id::text, o.content_hash, o.json_content,
		COALESCE(o.object_key,''), o.content_type, o.content_size FROM scene_asset_draft_files f
		JOIN scene_content_objects o ON o.id=f.content_object_id WHERE f.asset_id=$1`, asset.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	files := map[string]revisionFile{}
	for rows.Next() {
		var item revisionFile
		if err := rows.Scan(&item.Path, &item.ObjectID, &item.Hash, &item.Content, &item.ObjectKey, &item.Type, &item.Size); err != nil {
			return nil, err
		}
		files[item.Path] = item
	}
	return files, rows.Err()
}

func (r *PostgreSQLRepository) CommitAssetDraft(ctx context.Context, actor auth.User, asset SceneAsset, baseDraft int64, analysis AssetAnalysis, rootHash string, files map[string]contentObject) (SceneAsset, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SceneAsset{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var current, draft int64
	err = tx.QueryRow(ctx, `SELECT current_generation, draft_version FROM scene_assets
		WHERE id=$1 AND tenant_id=$2 AND archived_at IS NULL FOR UPDATE`, asset.ID, actor.TenantID).Scan(&current, &draft)
	if errors.Is(err, pgx.ErrNoRows) {
		return SceneAsset{}, ErrAssetNotFound
	}
	if err != nil {
		return SceneAsset{}, err
	}
	if draft != baseDraft {
		return SceneAsset{}, ErrDraftConflict
	}
	if _, err := insertAssetGeneration(ctx, tx, actor, asset, current+1, analysis, rootHash, files); err != nil {
		return SceneAsset{}, err
	}
	err = tx.QueryRow(ctx, `UPDATE scene_assets SET current_generation=$1, committed_draft_version=$2,
		entry_path=$3, updated_by=$4, updated_at=now() WHERE id=$5
		RETURNING current_generation, committed_draft_version, updated_at`, current+1, draft, analysis.EntryPath, actor.ID, asset.ID).
		Scan(&asset.CurrentGeneration, &asset.CommittedDraftVersion, &asset.UpdatedAt)
	if err != nil {
		return SceneAsset{}, err
	}
	asset.DraftVersion = draft
	asset.EntryPath = analysis.EntryPath
	if err := tx.Commit(ctx); err != nil {
		return SceneAsset{}, err
	}
	return asset, nil
}

func mapAssetDatabaseError(err error) error {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "scene_assets_active_name_uidx") || strings.Contains(message, "duplicate") {
		return fmt.Errorf("%w: 同分类下名称重复", ErrAssetConflict)
	}
	return err
}

func assetRootHash(files map[string]ProviderFile) string {
	paths := make([]string, 0, len(files))
	for filename := range files {
		paths = append(paths, filename)
	}
	sort.Strings(paths)
	hasher := sha256.New()
	for _, filename := range paths {
		_, _ = hasher.Write([]byte(filename + "\x00" + files[filename].Hash + "\n"))
	}
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
