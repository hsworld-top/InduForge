package repository

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/security"
)

// ConnectionSecretRepository 负责外部接入源密钥的加解密持久化；调用方只能按明确键名读取。
type ConnectionSecretRepository struct {
	pool   *pgxpool.Pool
	cipher *security.ConnectionSecretCipher
}

func NewConnectionSecretRepository(pool *pgxpool.Pool, cipher *security.ConnectionSecretCipher) *ConnectionSecretRepository {
	return &ConnectionSecretRepository{pool: pool, cipher: cipher}
}

func (r *ConnectionSecretRepository) Status(ctx context.Context, connectionID string) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx, `SELECT secret_key FROM data_connection_secrets WHERE connection_id = $1 ORDER BY secret_key`, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取接入源密钥状态失败", err)
	}
	defer rows.Close()
	result := map[string]bool{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		result[key] = true
	}
	return result, rows.Err()
}

func (r *ConnectionSecretRepository) StatusByConnections(ctx context.Context, connectionIDs []string) (map[string]map[string]bool, error) {
	result := map[string]map[string]bool{}
	if len(connectionIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT connection_id::text, secret_key FROM data_connection_secrets WHERE connection_id = ANY($1::uuid[]) ORDER BY connection_id, secret_key`, connectionIDs)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "批量读取接入源密钥状态失败", err)
	}
	defer rows.Close()
	for rows.Next() {
		var connectionID, key string
		if err := rows.Scan(&connectionID, &key); err != nil {
			return nil, err
		}
		if result[connectionID] == nil {
			result[connectionID] = map[string]bool{}
		}
		result[connectionID][key] = true
	}
	return result, rows.Err()
}

func (r *ConnectionSecretRepository) ResolveAll(ctx context.Context, connectionID string) (map[string]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT secret_key, encrypted_value, encryption_key_version FROM data_connection_secrets WHERE connection_id = $1 ORDER BY secret_key`, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取接入源密钥失败", err)
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var key, version string
		var encrypted []byte
		if err := rows.Scan(&key, &encrypted, &version); err != nil {
			return nil, err
		}
		if r.cipher == nil || version != r.cipher.KeyVersion() {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "接入源密钥版本不可用")
		}
		plaintext, err := r.cipher.Decrypt(connectionID, key, encrypted)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解密接入源密钥失败", err)
		}
		result[key] = string(plaintext)
	}
	return result, rows.Err()
}

func (r *ConnectionSecretRepository) Resolve(ctx context.Context, connectionID, key string) (string, bool, error) {
	var encrypted []byte
	var version string
	err := r.pool.QueryRow(ctx, `SELECT encrypted_value, encryption_key_version FROM data_connection_secrets WHERE connection_id = $1 AND secret_key = $2`, connectionID, key).Scan(&encrypted, &version)
	if err == pgx.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取接入源密钥失败", err)
	}
	if r.cipher == nil || version != r.cipher.KeyVersion() {
		return "", false, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "接入源密钥版本不可用")
	}
	plaintext, err := r.cipher.Decrypt(connectionID, key, encrypted)
	if err != nil {
		return "", false, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解密接入源密钥失败", err)
	}
	return string(plaintext), true, nil
}

func applyPlainConnectionSecretsTx(ctx context.Context, tx pgx.Tx, cipher *security.ConnectionSecretCipher, connectionID string, secrets map[string]string, clearKeys []string) error {
	if cipher == nil && (len(secrets) > 0 || len(clearKeys) > 0) {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "接入源密钥服务未初始化")
	}
	deleteSet := map[string]struct{}{}
	for _, key := range clearKeys {
		key = strings.TrimSpace(key)
		if key != "" {
			deleteSet[key] = struct{}{}
		}
	}
	deleteKeys := make([]string, 0, len(deleteSet))
	for key := range deleteSet {
		deleteKeys = append(deleteKeys, key)
	}
	sort.Strings(deleteKeys)
	if len(deleteKeys) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM data_connection_secrets WHERE connection_id = $1 AND secret_key = ANY($2::text[])`, connectionID, deleteKeys); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除接入源密钥失败", err)
		}
	}
	keys := make([]string, 0, len(secrets))
	for key := range secrets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		key = strings.TrimSpace(key)
		value := secrets[key]
		if key == "" || value == "" {
			continue
		}
		encrypted, err := cipher.Encrypt(connectionID, key, []byte(value))
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "加密接入源密钥失败", err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO data_connection_secrets (connection_id, secret_key, encrypted_value, encryption_key_version)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (connection_id, secret_key) DO UPDATE
			SET encrypted_value = EXCLUDED.encrypted_value,
			    encryption_key_version = EXCLUDED.encryption_key_version,
			    updated_at = now()
		`, connectionID, key, encrypted, cipher.KeyVersion())
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "保存接入源密钥失败", err)
		}
	}
	return nil
}

func replaceScopedConnectionSecretsTx(ctx context.Context, tx pgx.Tx, cipher *security.ConnectionSecretCipher, connectionID, prefix string, secrets map[string]string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM data_connection_secrets WHERE connection_id=$1 AND secret_key LIKE $2`, connectionID, prefix+"%"); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "替换工作台密钥失败", err)
	}
	prefixed := make(map[string]string, len(secrets))
	for key, value := range secrets {
		prefixed[prefix+key] = value
	}
	return applyPlainConnectionSecretsTx(ctx, tx, cipher, connectionID, prefixed, nil)
}
