package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthoringFenceRecord struct {
	Token          string    `json:"fenceToken"`
	ExpiresAt      time.Time `json:"expiresAt"`
	AuthoringEpoch string    `json:"authoringEpoch"`
	Mode           string    `json:"mode"`
}

type AuthoringEpochConflict struct {
	ProjectID string
	Current   int64
}

func (e *AuthoringEpochConflict) Error() string {
	return "工程开发态已变化，请刷新后重试"
}

func formatRepositoryEpoch(epoch int64) string { return "epoch-" + strconv.FormatInt(epoch, 10) }
func parseRepositoryEpoch(value string) (int64, error) {
	if !strings.HasPrefix(value, "epoch-") {
		return 0, errors.New("authoring epoch invalid")
	}
	epoch, err := strconv.ParseInt(strings.TrimPrefix(value, "epoch-"), 10, 64)
	if err != nil || epoch < 1 {
		return 0, errors.New("authoring epoch invalid")
	}
	return epoch, nil
}

type AuthoringFenceRepository struct {
	pool     *pgxpool.Pool
	lockPool *pgxpool.Pool
}

func NewAuthoringFenceRepository(pool, lockPool *pgxpool.Pool) *AuthoringFenceRepository {
	return &AuthoringFenceRepository{pool: pool, lockPool: lockPool}
}

func (r *AuthoringFenceRepository) ResolveQueryProject(ctx context.Context, queryID, tenantID string) (string, error) {
	var projectID string
	err := r.pool.QueryRow(ctx, `SELECT q.project_id::text FROM data_queries q WHERE q.id=$1 AND (NOT EXISTS(SELECT 1 FROM data_project_tenant_bindings b WHERE b.project_id=q.project_id) OR EXISTS(SELECT 1 FROM data_project_tenant_bindings b WHERE b.project_id=q.project_id AND b.tenant_id=$2))`, queryID, tenantID).Scan(&projectID)
	return projectID, err
}

func advisoryFenceKeys(projectID string) (int32, int32) {
	sum := sha256.Sum256([]byte(projectID))
	return int32(binary.BigEndian.Uint32(sum[0:4])), int32(binary.BigEndian.Uint32(sum[4:8]))
}

func (r *AuthoringFenceRepository) withExclusive(ctx context.Context, projectID string, fn func() error) error {
	conn, err := r.lockPool.Acquire(ctx)
	if err != nil {
		return err
	}
	k1, k2 := advisoryFenceKeys(projectID)
	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock($1,$2)`, k1, k2); err != nil {
		conn.Release()
		return err
	}
	defer func() {
		if unlockFenceConn(conn, false, k1, k2) {
			conn.Release()
		}
	}()
	return fn()
}

func unlockFenceConn(conn *pgxpool.Conn, shared bool, k1, k2 int32) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	statement := `SELECT pg_advisory_unlock($1,$2)`
	if shared {
		statement = `SELECT pg_advisory_unlock_shared($1,$2)`
	}
	if _, err := conn.Exec(ctx, statement, k1, k2); err != nil {
		raw := conn.Hijack()
		_ = raw.Close(ctx)
		return false
	}
	return true
}

// GateNormalWrite 与 fence acquire 使用同一会话级 advisory lock；共享锁一直持有到 HTTP handler 返回，
// 从而消除“检查未锁定后才开始写入”的竞态。
func (r *AuthoringFenceRepository) GateNormalWrite(ctx context.Context, projectID, tenantID, expectedEpoch string) (func(), error) {
	conn, err := r.lockPool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	k1, k2 := advisoryFenceKeys(projectID)
	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock_shared($1,$2)`, k1, k2); err != nil {
		conn.Release()
		return nil, err
	}
	release := func() {
		if unlockFenceConn(conn, true, k1, k2) {
			conn.Release()
		}
	}
	var currentEpoch int64
	var blocked bool
	if err = conn.QueryRow(ctx, `SELECT b.authoring_epoch,EXISTS(SELECT 1 FROM data_authoring_fences f WHERE f.project_id=b.project_id AND f.tenant_id=b.tenant_id AND (f.expires_at>now() OR f.mode='restore')) FROM data_project_tenant_bindings b WHERE b.project_id=$1 AND b.tenant_id=$2`, projectID, tenantID).Scan(&currentEpoch, &blocked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 中心在项目创建后通过内部接口初始化绑定；无绑定仅保留给独立 data_service
			// 开发/测试场景，不能伪造一个与中心脱节的 epoch 事实源。
			return release, nil
		}
		release()
		return nil, err
	}
	expected, parseErr := parseRepositoryEpoch(strings.TrimSpace(expectedEpoch))
	if blocked || parseErr != nil || expected != currentEpoch {
		release()
		return nil, &AuthoringEpochConflict{ProjectID: projectID, Current: currentEpoch}
	}
	return release, nil
}

// RestoreSnapshot 持有独占工程锁直至覆盖事务提交。即使 fence TTL 在大快照恢复中途到期，
// 普通写请求仍无法取得共享锁，因此不会与恢复事务并发。
func (r *AuthoringFenceRepository) RestoreSnapshot(ctx context.Context, snapshots *ProjectSnapshotRepository, projectID, tenantID, actorID, ownerID, token, expectedEpoch, targetEpoch, direction string, snapshot ProjectSnapshot) error {
	for _, connection := range snapshot.Connections {
		if connectionConfigContainsPlaintextSecret(connection.Config, "") {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "工程快照连接配置包含明文密钥")
		}
	}
	if err := validateSnapshotComputeOutputTargets(snapshot); err != nil {
		return err
	}
	return r.withExclusive(ctx, projectID, func() error {
		expected, err := parseRepositoryEpoch(expectedEpoch)
		if err != nil {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "expectedAuthoringEpoch 格式无效")
		}
		target, err := parseRepositoryEpoch(targetEpoch)
		if err != nil || (direction != "forward" && direction != "compensate") || (direction == "forward" && target != expected+1) || (direction == "compensate" && target != expected-1) {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "工程开发态代次变化无效")
		}
		var stored []byte
		var current int64
		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		err = tx.QueryRow(ctx, `SELECT f.token_hash,b.authoring_epoch FROM data_authoring_fences f JOIN data_project_tenant_bindings b ON b.project_id=f.project_id AND b.tenant_id=f.tenant_id WHERE f.project_id=$1 AND f.tenant_id=$2 AND f.owner_id=$3 AND f.mode='restore' AND f.expires_at>now()`, projectID, tenantID, ownerID).Scan(&stored, &current)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "工程开发内容写保护不存在")
		}
		if err != nil {
			return err
		}
		if subtle.ConstantTimeCompare(stored, fenceHash(token)) != 1 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "工程开发内容写保护令牌无效")
		}
		if current != expected {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "工程开发态代次不一致")
		}
		if err = snapshots.replaceProjectDataTx(ctx, tx, projectID, actorID, snapshot); err != nil {
			return err
		}
		result, err := tx.Exec(ctx, `UPDATE data_project_tenant_bindings SET authoring_epoch=$4,updated_at=now() WHERE project_id=$1 AND tenant_id=$2 AND authoring_epoch=$3`, projectID, tenantID, expected, target)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "工程开发态代次不一致")
		}
		return tx.Commit(ctx)
	})
}

func newFenceToken() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	return token, hash[:], nil
}

func (r *AuthoringFenceRepository) Acquire(ctx context.Context, projectID, tenantID, ownerID, mode, expectedEpoch string, ttl time.Duration) (*AuthoringFenceRecord, error) {
	token, hash, err := newFenceToken()
	if err != nil {
		return nil, err
	}
	var stored time.Time
	var currentEpoch int64
	err = r.withExclusive(ctx, projectID, func() error {
		tx, beginErr := r.pool.Begin(ctx)
		if beginErr != nil {
			return beginErr
		}
		defer tx.Rollback(ctx)
		expected, parseErr := parseRepositoryEpoch(expectedEpoch)
		if parseErr != nil {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "工程开发态代次不一致")
		}
		if queryErr := tx.QueryRow(ctx, `SELECT authoring_epoch FROM data_project_tenant_bindings WHERE project_id=$1 AND tenant_id=$2 FOR UPDATE`, projectID, tenantID).Scan(&currentEpoch); queryErr != nil {
			return queryErr
		}
		var existingOwner, existingMode string
		var existingExpired bool
		queryErr := tx.QueryRow(ctx, `SELECT owner_id::text,mode,expires_at<=now() FROM data_authoring_fences WHERE project_id=$1 FOR UPDATE`, projectID).Scan(&existingOwner, &existingMode, &existingExpired)
		if queryErr != nil && !errors.Is(queryErr, pgx.ErrNoRows) {
			return queryErr
		}
		recoveringRestore := queryErr == nil && existingMode == "restore" && mode == "restore" && existingOwner == ownerID
		// 同一恢复任务可能在 data 已推进而 core 尚未推进时崩溃。重入请求允许携带
		// 任务的 expected 或 target（两者严格相邻），并始终回显数据库实际 epoch，
		// 由控制面据此决定继续完成还是执行补偿。其他 acquire 仍要求完全匹配。
		recoveryEpochMatches := recoveringRestore &&
			((currentEpoch == expected+1) || (expected == currentEpoch+1))
		if currentEpoch != expected && !recoveryEpochMatches {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "工程开发态代次不一致")
		}
		if queryErr == nil {
			expiredCapture := existingMode == "capture" && existingExpired
			if !recoveringRestore && !expiredCapture {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "工程开发内容正在被发布或恢复，请稍后重试")
			}
			if queryErr = tx.QueryRow(ctx, `UPDATE data_authoring_fences SET tenant_id=$2,owner_id=$3,mode=$4,token_hash=$5,expires_at=now()+make_interval(secs => $6),updated_at=now() WHERE project_id=$1 RETURNING expires_at`, projectID, tenantID, ownerID, mode, hash, int(ttl/time.Second)).Scan(&stored); queryErr != nil {
				return queryErr
			}
		} else if queryErr = tx.QueryRow(ctx, `INSERT INTO data_authoring_fences(project_id,tenant_id,owner_id,mode,token_hash,expires_at) VALUES($1,$2,$3,$4,$5,now()+make_interval(secs => $6)) RETURNING expires_at`, projectID, tenantID, ownerID, mode, hash, int(ttl/time.Second)).Scan(&stored); queryErr != nil {
			return queryErr
		}
		return tx.Commit(ctx)
	})
	if err != nil {
		return nil, err
	}
	return &AuthoringFenceRecord{Token: token, ExpiresAt: stored, AuthoringEpoch: formatRepositoryEpoch(currentEpoch), Mode: mode}, nil
}

func fenceHash(token string) []byte { sum := sha256.Sum256([]byte(token)); return sum[:] }

func (r *AuthoringFenceRepository) Renew(ctx context.Context, projectID, tenantID, ownerID, token string, ttl time.Duration) (*AuthoringFenceRecord, error) {
	var expires time.Time
	var epoch int64
	var mode string
	err := r.pool.QueryRow(ctx, `WITH renewed AS (UPDATE data_authoring_fences SET expires_at=now()+make_interval(secs => $5),updated_at=now() WHERE project_id=$1 AND tenant_id=$2 AND owner_id=$3 AND token_hash=$4 AND expires_at>now() RETURNING project_id,tenant_id,mode,expires_at) SELECT renewed.expires_at,b.authoring_epoch,renewed.mode FROM renewed JOIN data_project_tenant_bindings b USING(project_id,tenant_id)`, projectID, tenantID, ownerID, fenceHash(token), int(ttl/time.Second)).Scan(&expires, &epoch, &mode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "工程开发内容写保护已失效")
	}
	if err != nil {
		return nil, err
	}
	return &AuthoringFenceRecord{Token: token, ExpiresAt: expires, AuthoringEpoch: formatRepositoryEpoch(epoch), Mode: mode}, nil
}

func (r *AuthoringFenceRepository) Release(ctx context.Context, projectID, tenantID, ownerID, token string) error {
	return r.withExclusive(ctx, projectID, func() error {
		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		var storedOwner string
		var storedHash []byte
		err = tx.QueryRow(ctx, `SELECT owner_id::text,token_hash FROM data_authoring_fences WHERE project_id=$1 AND tenant_id=$2 FOR UPDATE`, projectID, tenantID).Scan(&storedOwner, &storedHash)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if storedOwner != ownerID || subtle.ConstantTimeCompare(storedHash, fenceHash(token)) != 1 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "工程开发内容写保护令牌无效")
		}
		if _, err = tx.Exec(ctx, `DELETE FROM data_authoring_fences WHERE project_id=$1`, projectID); err != nil {
			return err
		}
		return tx.Commit(ctx)
	})
}
