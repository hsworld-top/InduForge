package deployment

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

func (r *PostgreSQLRepository) AcquireAuthoringFence(ctx context.Context, tenantID, projectID, ownerKind, ownerID string, ttl time.Duration) (string, error) {
	if ttl <= 0 || (ownerKind != "build" && ownerKind != "restore") {
		return "", fmt.Errorf("工程写栅栏参数无效")
	}
	tokenRaw := make([]byte, 32)
	if _, err := rand.Read(tokenRaw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenRaw)
	sum := sha256.Sum256([]byte(token))
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, projectID); err != nil {
		return "", err
	}
	var allowed bool
	if ownerKind == "build" {
		err = tx.QueryRow(ctx, `SELECT NOT EXISTS(SELECT 1 FROM authoring_restore_tasks WHERE project_id=$1 AND state IN ('queued','staging','restoring_workspace','restoring_scenes','restoring_data','finalizing','compensating'))`, projectID).Scan(&allowed)
	} else {
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM authoring_restore_tasks WHERE id=$1 AND project_id=$2 AND tenant_id=$3 AND state IN ('queued','staging','restoring_workspace','restoring_scenes','restoring_data','finalizing','compensating','succeeded'))`, ownerID, projectID, tenantID).Scan(&allowed)
	}
	if err != nil {
		return "", err
	}
	if !allowed {
		return "", fmt.Errorf("工程正在发布或恢复")
	}
	command, err := tx.Exec(ctx, `INSERT INTO authoring_project_fences(project_id,tenant_id,owner_kind,owner_id,fence_token_hash,expires_at) VALUES($1,$2,$3,$4,$5,now()+$6::interval) ON CONFLICT(project_id) DO UPDATE SET tenant_id=EXCLUDED.tenant_id,owner_kind=EXCLUDED.owner_kind,owner_id=EXCLUDED.owner_id,fence_token_hash=EXCLUDED.fence_token_hash,expires_at=EXCLUDED.expires_at,updated_at=now() WHERE authoring_project_fences.expires_at<=now()`, projectID, tenantID, ownerKind, ownerID, hex.EncodeToString(sum[:]), fmt.Sprintf("%d seconds", int(ttl.Seconds())))
	if err != nil {
		return "", err
	}
	if command.RowsAffected() != 1 {
		return "", fmt.Errorf("工程正在发布或恢复")
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return token, nil
}

func (r *PostgreSQLRepository) ReleaseAuthoringFence(ctx context.Context, tenantID, projectID, ownerID, token string) error {
	sum := sha256.Sum256([]byte(token))
	_, err := r.pool.Exec(ctx, `DELETE FROM authoring_project_fences WHERE tenant_id=$1 AND project_id=$2 AND owner_id=$3 AND fence_token_hash=$4`, tenantID, projectID, ownerID, hex.EncodeToString(sum[:]))
	return err
}

func (r *PostgreSQLRepository) RenewAuthoringFence(ctx context.Context, tenantID, projectID, ownerID, token string, ttl time.Duration) error {
	sum := sha256.Sum256([]byte(token))
	command, err := r.pool.Exec(ctx, `UPDATE authoring_project_fences SET expires_at=now()+$1::interval,updated_at=now() WHERE tenant_id=$2 AND project_id=$3 AND owner_id=$4 AND fence_token_hash=$5 AND expires_at>now()`, fmt.Sprintf("%d seconds", int(ttl.Seconds())), tenantID, projectID, ownerID, hex.EncodeToString(sum[:]))
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("工程写栅栏已失效")
	}
	return nil
}
