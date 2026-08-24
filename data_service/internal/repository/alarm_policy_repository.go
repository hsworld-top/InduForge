package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

type AlarmPolicyGroupRecord struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	ParentID    *string   `json:"parentId"`
	Description *string   `json:"description"`
	SortOrder   int       `json:"sortOrder"`
	FullPath    string    `json:"fullPath"`
	HasChildren bool      `json:"hasChildren"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AlarmProjectSettingsRecord struct {
	ProjectID              string    `json:"projectId"`
	NotifyOnRaise          bool      `json:"notifyOnRaise"`
	NotifyOnClear          bool      `json:"notifyOnClear"`
	RepeatIntervalSeconds  *int      `json:"repeatIntervalSeconds"`
	DefaultMessageTemplate string    `json:"defaultMessageTemplate"`
	DefaultChannelIDs      []string  `json:"defaultChannelIds"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

type AlarmHistorySettingsRecord struct {
	ProjectID                   string    `json:"projectId"`
	IsEnabled                   bool      `json:"isEnabled"`
	RetentionDays               *int      `json:"retentionDays"`
	StoreNotificationDeliveries bool      `json:"storeNotificationDeliveries"`
	CreatedAt                   time.Time `json:"createdAt"`
	UpdatedAt                   time.Time `json:"updatedAt"`
}

type AlarmNotificationChannelRecord struct {
	ID           string         `json:"id"`
	ProjectID    string         `json:"projectId"`
	Name         string         `json:"name"`
	ChannelType  string         `json:"channelType"`
	Config       map[string]any `json:"config"`
	SecretStatus map[string]any `json:"secretStatus"`
	IsEnabled    bool           `json:"isEnabled"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

type EncryptedAlarmChannelSecret struct {
	Key, KeyVersion string
	Value           []byte
}

type AlarmPolicyGroupListFilter struct {
	Search   string
	ParentID *string
	Page     int
	PageSize int
}

type SaveAlarmPolicyGroupParams struct {
	ID, ProjectID, UserID, Name string
	ParentID                    *string
	Description                 *string
	SortOrder                   int
}

type SaveAlarmProjectSettingsParams struct {
	ProjectID, UserID, DefaultMessageTemplate string
	NotifyOnRaise, NotifyOnClear              bool
	RepeatIntervalSeconds                     *int
	DefaultChannelIDs                         []string
}

type SaveAlarmHistorySettingsParams struct {
	ProjectID, UserID           string
	IsEnabled                   bool
	RetentionDays               *int
	StoreNotificationDeliveries bool
}

type SaveAlarmNotificationChannelParams struct {
	ID, ProjectID, UserID, Name, ChannelType string
	Config, SecretStatus                     map[string]any
	Secrets                                  []EncryptedAlarmChannelSecret
	DeleteSecretKeys                         []string
	IsEnabled                                bool
}

type AlarmConfigSyncStateRecord struct {
	ProjectID, SyncEpoch, LastIdempotencyKey string
	ConfigRevision, LastSequence             int64
	UpdatedAt                                time.Time
}

type AlarmConfigSyncOperationParams struct {
	Resource        string
	Action          string
	ID              string
	Group           *SaveAlarmPolicyGroupParams
	AlarmItem       *SaveAlarmItemParams
	Settings        *SaveAlarmProjectSettingsParams
	HistorySettings *SaveAlarmHistorySettingsParams
	Channel         *SaveAlarmNotificationChannelParams
}

type AlarmConfigSyncResultRecord struct {
	ConfigRevision   int64
	AcceptedSequence int64
	Idempotent       bool
}

type AlarmPolicyRepository struct{ pool *pgxpool.Pool }

func NewAlarmPolicyRepository(pool *pgxpool.Pool) *AlarmPolicyRepository {
	return &AlarmPolicyRepository{pool: pool}
}

func (r *AlarmPolicyRepository) ListGroups(ctx context.Context, projectID string, filter AlarmPolicyGroupListFilter) ([]AlarmPolicyGroupRecord, int, error) {
	page, size := normalizePageAndSize(filter.Page, filter.PageSize, 50, 200)
	args := []any{projectID}
	where := []string{"g.project_id = $1"}
	if filter.ParentID != nil {
		args = append(args, *filter.ParentID)
		where = append(where, fmt.Sprintf("g.parent_id = $%d", len(args)))
	} else if strings.TrimSpace(filter.Search) == "" {
		where = append(where, "g.parent_id IS NULL")
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		where = append(where, fmt.Sprintf("(g.name ILIKE $%d OR paths.full_path ILIKE $%d)", len(args), len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	cte := `WITH RECURSIVE paths AS (
        SELECT id, project_id, name, parent_id, name::text AS full_path FROM data_alarm_groups WHERE project_id = $1 AND parent_id IS NULL
        UNION ALL
        SELECT c.id, c.project_id, c.name, c.parent_id, (p.full_path || ' / ' || c.name) FROM data_alarm_groups c JOIN paths p ON p.id = c.parent_id WHERE c.project_id = $1
    ) `
	var total int
	if err := r.pool.QueryRow(ctx, cte+`SELECT COUNT(*) FROM data_alarm_groups g JOIN paths ON paths.id=g.id WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, wrapAlarmRepo("统计报警目录失败", err)
	}
	queryArgs := append(append([]any{}, args...), size, (page-1)*size)
	rows, err := r.pool.Query(ctx, cte+`SELECT g.id,g.project_id,g.name,g.parent_id,g.description,g.sort_order,paths.full_path,
        EXISTS(SELECT 1 FROM data_alarm_groups c WHERE c.parent_id=g.id),g.created_at,g.updated_at
        FROM data_alarm_groups g JOIN paths ON paths.id=g.id WHERE `+whereSQL+
		fmt.Sprintf(" ORDER BY g.sort_order,g.created_at LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2), queryArgs...)
	if err != nil {
		return nil, 0, wrapAlarmRepo("查询报警目录失败", err)
	}
	defer rows.Close()
	result := make([]AlarmPolicyGroupRecord, 0)
	for rows.Next() {
		var item AlarmPolicyGroupRecord
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Name, &item.ParentID, &item.Description, &item.SortOrder, &item.FullPath, &item.HasChildren, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, wrapAlarmRepo("读取报警目录失败", err)
		}
		result = append(result, item)
	}
	return result, total, rows.Err()
}

func (r *AlarmPolicyRepository) ListAllGroups(ctx context.Context, projectID string) ([]AlarmPolicyGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `WITH RECURSIVE paths AS (
        SELECT id,project_id,name,parent_id,description,sort_order,name::text AS full_path,created_at,updated_at FROM data_alarm_groups WHERE project_id=$1 AND parent_id IS NULL
        UNION ALL SELECT c.id,c.project_id,c.name,c.parent_id,c.description,c.sort_order,(p.full_path || ' / ' || c.name),c.created_at,c.updated_at FROM data_alarm_groups c JOIN paths p ON p.id=c.parent_id WHERE c.project_id=$1
    ) SELECT id,project_id,name,parent_id,description,sort_order,full_path,EXISTS(SELECT 1 FROM data_alarm_groups c WHERE c.parent_id=paths.id),created_at,updated_at FROM paths ORDER BY full_path`, projectID)
	if err != nil {
		return nil, wrapAlarmRepo("查询报警目录失败", err)
	}
	defer rows.Close()
	items := make([]AlarmPolicyGroupRecord, 0)
	for rows.Next() {
		var item AlarmPolicyGroupRecord
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Name, &item.ParentID, &item.Description, &item.SortOrder, &item.FullPath, &item.HasChildren, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, wrapAlarmRepo("读取报警目录失败", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *AlarmPolicyRepository) GetGroup(ctx context.Context, projectID, id string) (*AlarmPolicyGroupRecord, error) {
	groups, err := r.ListAllGroups(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		if group.ID == id {
			copy := group
			return &copy, nil
		}
	}
	return nil, alarmNotFound("报警目录不存在")
}

func (r *AlarmPolicyRepository) SaveGroup(ctx context.Context, p SaveAlarmPolicyGroupParams) (*AlarmPolicyGroupRecord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, wrapAlarmRepo("开启报警目录事务失败", err)
	}
	defer tx.Rollback(ctx)
	var id string
	if p.ID == "" {
		err = tx.QueryRow(ctx, `INSERT INTO data_alarm_groups(project_id,name,parent_id,description,sort_order,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$6) RETURNING id`, p.ProjectID, p.Name, p.ParentID, p.Description, p.SortOrder, p.UserID).Scan(&id)
	} else {
		id = p.ID
		tag, execErr := tx.Exec(ctx, `UPDATE data_alarm_groups SET name=$3,parent_id=$4,description=$5,sort_order=$6,updated_by=$7,updated_at=now() WHERE project_id=$1 AND id=$2`, p.ProjectID, p.ID, p.Name, p.ParentID, p.Description, p.SortOrder, p.UserID)
		err = execErr
		if err == nil && tag.RowsAffected() == 0 {
			err = pgx.ErrNoRows
		}
	}
	if err != nil {
		return nil, translateAlarmWrite("保存报警目录失败", err)
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, p.ProjectID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, wrapAlarmRepo("提交报警目录事务失败", err)
	}
	return r.GetGroup(ctx, p.ProjectID, id)
}

func (r *AlarmPolicyRepository) DeleteGroup(ctx context.Context, projectID, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return wrapAlarmRepo("开启删除报警目录事务失败", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `DELETE FROM data_alarm_groups WHERE project_id=$1 AND id=$2`, projectID, id)
	if err != nil {
		return translateAlarmWrite("删除报警目录失败", err)
	}
	if tag.RowsAffected() == 0 {
		return alarmNotFound("报警目录不存在")
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, projectID); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return wrapAlarmRepo("提交删除报警目录事务失败", err)
	}
	return nil
}

func (r *AlarmPolicyRepository) GetProjectSettings(ctx context.Context, projectID string) (*AlarmProjectSettingsRecord, error) {
	item, err := scanAlarmProjectSettings(r.pool.QueryRow(ctx, `SELECT project_id,notify_on_raise,notify_on_clear,repeat_interval_seconds,default_message_template,default_channel_ids,created_at,updated_at FROM data_alarm_project_settings WHERE project_id=$1`, projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AlarmPolicyRepository) SaveProjectSettings(ctx context.Context, p SaveAlarmProjectSettingsParams) (*AlarmProjectSettingsRecord, error) {
	channels, err := json.Marshal(p.DefaultChannelIDs)
	if err != nil {
		return nil, wrapAlarmRepo("编码默认通知渠道失败", err)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, wrapAlarmRepo("开启报警设置事务失败", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO data_alarm_project_settings(project_id,notify_on_raise,notify_on_clear,repeat_interval_seconds,default_message_template,default_channel_ids,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7,$7) ON CONFLICT(project_id) DO UPDATE SET notify_on_raise=EXCLUDED.notify_on_raise,notify_on_clear=EXCLUDED.notify_on_clear,repeat_interval_seconds=EXCLUDED.repeat_interval_seconds,default_message_template=EXCLUDED.default_message_template,default_channel_ids=EXCLUDED.default_channel_ids,updated_by=EXCLUDED.updated_by,updated_at=now()`, p.ProjectID, p.NotifyOnRaise, p.NotifyOnClear, p.RepeatIntervalSeconds, p.DefaultMessageTemplate, string(channels), p.UserID)
	if err != nil {
		return nil, translateAlarmWrite("保存报警设置失败", err)
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, p.ProjectID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, wrapAlarmRepo("提交报警设置失败", err)
	}
	return r.GetProjectSettings(ctx, p.ProjectID)
}

func (r *AlarmPolicyRepository) GetHistorySettings(ctx context.Context, projectID string) (*AlarmHistorySettingsRecord, error) {
	item, err := scanAlarmHistorySettings(r.pool.QueryRow(ctx, `SELECT project_id,is_enabled,retention_days,store_notification_deliveries,created_at,updated_at FROM data_alarm_history_settings WHERE project_id=$1`, projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AlarmPolicyRepository) SaveHistorySettings(ctx context.Context, p SaveAlarmHistorySettingsParams) (*AlarmHistorySettingsRecord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, wrapAlarmRepo("开启报警历史设置事务失败", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO data_alarm_history_settings(project_id,is_enabled,retention_days,store_notification_deliveries,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$5) ON CONFLICT(project_id) DO UPDATE SET is_enabled=EXCLUDED.is_enabled,retention_days=EXCLUDED.retention_days,store_notification_deliveries=EXCLUDED.store_notification_deliveries,updated_by=EXCLUDED.updated_by,updated_at=now()`, p.ProjectID, p.IsEnabled, p.RetentionDays, p.StoreNotificationDeliveries, p.UserID)
	if err != nil {
		return nil, translateAlarmWrite("保存报警历史设置失败", err)
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, p.ProjectID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, wrapAlarmRepo("提交报警历史设置失败", err)
	}
	return r.GetHistorySettings(ctx, p.ProjectID)
}

func (r *AlarmPolicyRepository) ListChannels(ctx context.Context, projectID string) ([]AlarmNotificationChannelRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,project_id,name,channel_type,config,secret_status,is_enabled,created_at,updated_at FROM data_alarm_notification_channels WHERE project_id=$1 ORDER BY updated_at DESC`, projectID)
	if err != nil {
		return nil, wrapAlarmRepo("查询通知渠道失败", err)
	}
	defer rows.Close()
	items := make([]AlarmNotificationChannelRecord, 0)
	for rows.Next() {
		item, scanErr := scanAlarmChannel(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *AlarmPolicyRepository) GetChannel(ctx context.Context, projectID, id string) (*AlarmNotificationChannelRecord, error) {
	item, err := scanAlarmChannel(r.pool.QueryRow(ctx, `SELECT id,project_id,name,channel_type,config,secret_status,is_enabled,created_at,updated_at FROM data_alarm_notification_channels WHERE project_id=$1 AND id=$2`, projectID, id))
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AlarmPolicyRepository) SaveChannel(ctx context.Context, p SaveAlarmNotificationChannelParams) (*AlarmNotificationChannelRecord, error) {
	config, status, err := marshalAlarmJSON(p.Config, p.SecretStatus)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, wrapAlarmRepo("开启通知渠道事务失败", err)
	}
	defer tx.Rollback(ctx)
	id := p.ID
	if id == "" {
		err = tx.QueryRow(ctx, `INSERT INTO data_alarm_notification_channels(project_id,name,channel_type,config,secret_status,is_enabled,created_by,updated_by) VALUES($1,$2,$3,$4::jsonb,$5::jsonb,$6,$7,$7) RETURNING id`, p.ProjectID, p.Name, p.ChannelType, string(config), string(status), p.IsEnabled, p.UserID).Scan(&id)
	} else {
		tag, execErr := tx.Exec(ctx, `UPDATE data_alarm_notification_channels SET name=$3,channel_type=$4,config=$5::jsonb,secret_status=$6::jsonb,is_enabled=$7,updated_by=$8,updated_at=now() WHERE project_id=$1 AND id=$2`, p.ProjectID, id, p.Name, p.ChannelType, string(config), string(status), p.IsEnabled, p.UserID)
		err = execErr
		if err == nil && tag.RowsAffected() == 0 {
			err = pgx.ErrNoRows
		}
	}
	if err != nil {
		return nil, translateAlarmWrite("保存通知渠道失败", err)
	}
	for _, key := range p.DeleteSecretKeys {
		if _, err = tx.Exec(ctx, `DELETE FROM data_alarm_notification_channel_secrets WHERE channel_id=$1 AND secret_key=$2`, id, key); err != nil {
			return nil, wrapAlarmRepo("删除通知渠道密钥失败", err)
		}
	}
	for _, secret := range p.Secrets {
		if _, err = tx.Exec(ctx, `INSERT INTO data_alarm_notification_channel_secrets(channel_id,secret_key,encrypted_value,encryption_key_version) VALUES($1,$2,$3,$4) ON CONFLICT(channel_id,secret_key) DO UPDATE SET encrypted_value=EXCLUDED.encrypted_value,encryption_key_version=EXCLUDED.encryption_key_version,updated_at=now()`, id, secret.Key, secret.Value, secret.KeyVersion); err != nil {
			return nil, wrapAlarmRepo("保存通知渠道密钥失败", err)
		}
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, p.ProjectID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, wrapAlarmRepo("提交通知渠道失败", err)
	}
	return r.GetChannel(ctx, p.ProjectID, id)
}

func (r *AlarmPolicyRepository) DeleteChannel(ctx context.Context, projectID, id string) error {
	var used bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM data_alarm_project_settings WHERE project_id=$1 AND default_channel_ids ? $2) OR EXISTS(SELECT 1 FROM data_alarm_items WHERE project_id=$1 AND notification_channel_ids ? $2)`, projectID, id).Scan(&used)
	if err != nil {
		return wrapAlarmRepo("检查通知渠道占用失败", err)
	}
	if used {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "通知渠道正在被报警设置使用")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return wrapAlarmRepo("开启删除通知渠道事务失败", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `DELETE FROM data_alarm_notification_channels WHERE project_id=$1 AND id=$2`, projectID, id)
	if err != nil {
		return translateAlarmWrite("删除通知渠道失败", err)
	}
	if tag.RowsAffected() == 0 {
		return alarmNotFound("通知渠道不存在")
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, projectID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *AlarmPolicyRepository) GetSyncState(ctx context.Context, projectID string) (AlarmConfigSyncStateRecord, error) {
	var item AlarmConfigSyncStateRecord
	err := r.pool.QueryRow(ctx, `SELECT project_id,config_revision,sync_epoch,last_sequence,last_idempotency_key,updated_at FROM data_alarm_config_sync_state WHERE project_id=$1`, projectID).Scan(&item.ProjectID, &item.ConfigRevision, &item.SyncEpoch, &item.LastSequence, &item.LastIdempotencyKey, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AlarmConfigSyncStateRecord{ProjectID: projectID}, nil
	}
	if err != nil {
		return item, wrapAlarmRepo("查询报警同步状态失败", err)
	}
	return item, nil
}

// ApplyConfigSync 在单一事务中应用节点回写的开发态报警配置。
func (r *AlarmPolicyRepository) ApplyConfigSync(ctx context.Context, projectID, actorID, syncEpoch string, sequence int64, idempotencyKey string, operations []AlarmConfigSyncOperationParams) (AlarmConfigSyncResultRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AlarmConfigSyncResultRecord{}, wrapAlarmRepo("开启报警配置同步事务失败", err)
	}
	defer tx.Rollback(ctx)

	var previousRevision, previousSequence int64
	err = tx.QueryRow(ctx, `SELECT config_revision,sequence FROM data_alarm_config_sync_requests WHERE project_id=$1 AND idempotency_key=$2`, projectID, idempotencyKey).Scan(&previousRevision, &previousSequence)
	if err == nil {
		return AlarmConfigSyncResultRecord{ConfigRevision: previousRevision, AcceptedSequence: previousSequence, Idempotent: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return AlarmConfigSyncResultRecord{}, wrapAlarmRepo("检查报警配置同步幂等记录失败", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO data_alarm_config_sync_state(project_id) VALUES($1) ON CONFLICT(project_id) DO NOTHING`, projectID); err != nil {
		return AlarmConfigSyncResultRecord{}, wrapAlarmRepo("初始化报警配置同步状态失败", err)
	}
	var currentEpoch string
	var currentSequence int64
	if err = tx.QueryRow(ctx, `SELECT sync_epoch,last_sequence FROM data_alarm_config_sync_state WHERE project_id=$1 FOR UPDATE`, projectID).Scan(&currentEpoch, &currentSequence); err != nil {
		return AlarmConfigSyncResultRecord{}, wrapAlarmRepo("锁定报警配置同步状态失败", err)
	}
	// 并发重复请求可能在等待项目锁期间由另一事务完成，锁定后必须再次确认幂等记录。
	err = tx.QueryRow(ctx, `SELECT config_revision,sequence FROM data_alarm_config_sync_requests WHERE project_id=$1 AND idempotency_key=$2`, projectID, idempotencyKey).Scan(&previousRevision, &previousSequence)
	if err == nil {
		return AlarmConfigSyncResultRecord{ConfigRevision: previousRevision, AcceptedSequence: previousSequence, Idempotent: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return AlarmConfigSyncResultRecord{}, wrapAlarmRepo("锁定后检查报警配置同步幂等记录失败", err)
	}
	if currentEpoch == syncEpoch && sequence <= currentSequence {
		return AlarmConfigSyncResultRecord{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "报警配置同步序号已过期")
	}
	if currentEpoch == "" && sequence != 1 {
		return AlarmConfigSyncResultRecord{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "首次报警配置同步必须从序号 1 开始")
	}
	if currentEpoch != "" && currentEpoch != syncEpoch {
		if sequence != 1 {
			return AlarmConfigSyncResultRecord{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "新的报警配置同步批次必须从序号 1 开始")
		}
		var epochSeen bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM data_alarm_config_sync_requests WHERE project_id=$1 AND sync_epoch=$2)`, projectID, syncEpoch).Scan(&epochSeen); err != nil {
			return AlarmConfigSyncResultRecord{}, wrapAlarmRepo("检查报警配置同步批次失败", err)
		}
		if epochSeen {
			return AlarmConfigSyncResultRecord{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "报警配置同步批次已结束")
		}
	}

	for _, operation := range operations {
		if err = applyAlarmSyncOperation(ctx, tx, projectID, actorID, operation); err != nil {
			return AlarmConfigSyncResultRecord{}, err
		}
	}
	if err = validateAlarmSyncReferences(ctx, tx, projectID); err != nil {
		return AlarmConfigSyncResultRecord{}, err
	}
	configRevision, err := incrementAlarmConfigRevision(ctx, tx, projectID)
	if err != nil {
		return AlarmConfigSyncResultRecord{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE data_alarm_config_sync_state SET sync_epoch=$2,last_sequence=$3,last_idempotency_key=$4,updated_at=now() WHERE project_id=$1`, projectID, syncEpoch, sequence, idempotencyKey); err != nil {
		return AlarmConfigSyncResultRecord{}, wrapAlarmRepo("更新报警配置同步状态失败", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO data_alarm_config_sync_requests(project_id,idempotency_key,sync_epoch,sequence,config_revision) VALUES($1,$2,$3,$4,$5)`, projectID, idempotencyKey, syncEpoch, sequence, configRevision); err != nil {
		return AlarmConfigSyncResultRecord{}, translateAlarmWrite("保存报警配置同步幂等记录失败", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return AlarmConfigSyncResultRecord{}, wrapAlarmRepo("提交报警配置同步事务失败", err)
	}
	return AlarmConfigSyncResultRecord{ConfigRevision: configRevision, AcceptedSequence: sequence}, nil
}

func applyAlarmSyncOperation(ctx context.Context, tx pgx.Tx, projectID, actorID string, operation AlarmConfigSyncOperationParams) error {
	if operation.Action == "delete" {
		return deleteAlarmSyncResource(ctx, tx, projectID, operation.Resource, operation.ID)
	}
	switch operation.Resource {
	case "group":
		p := operation.Group
		tag, err := tx.Exec(ctx, `INSERT INTO data_alarm_groups(id,project_id,parent_id,name,description,sort_order,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$7) ON CONFLICT(id) DO UPDATE SET parent_id=EXCLUDED.parent_id,name=EXCLUDED.name,description=EXCLUDED.description,sort_order=EXCLUDED.sort_order,updated_by=EXCLUDED.updated_by,updated_at=now() WHERE data_alarm_groups.project_id=EXCLUDED.project_id`, operation.ID, projectID, p.ParentID, p.Name, p.Description, p.SortOrder, actorID)
		if err != nil || tag.RowsAffected() == 0 {
			return translateAlarmWrite("同步报警目录失败", firstAlarmError(err, pgx.ErrNoRows))
		}
	case "alarm_item":
		p := operation.AlarmItem
		channels, contract, err := marshalAlarmJSON(p.NotificationChannelIDs, p.Contract)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO data_alarm_items(id,project_id,datapoint_id,group_id,display_name,name_key,description,mode,alarm_type,evaluation_mode,derived_expression,trigger_fingerprint,notification_mode,notify_on_raise,notify_on_clear,repeat_interval_seconds,notification_channel_ids,message_template,is_enabled,contract,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17::jsonb,$18,$19,$20::jsonb,$21,$21) ON CONFLICT(id) DO UPDATE SET datapoint_id=EXCLUDED.datapoint_id,group_id=EXCLUDED.group_id,display_name=EXCLUDED.display_name,name_key=EXCLUDED.name_key,description=EXCLUDED.description,mode=EXCLUDED.mode,alarm_type=EXCLUDED.alarm_type,evaluation_mode=EXCLUDED.evaluation_mode,derived_expression=EXCLUDED.derived_expression,trigger_fingerprint=EXCLUDED.trigger_fingerprint,notification_mode=EXCLUDED.notification_mode,notify_on_raise=EXCLUDED.notify_on_raise,notify_on_clear=EXCLUDED.notify_on_clear,repeat_interval_seconds=EXCLUDED.repeat_interval_seconds,notification_channel_ids=EXCLUDED.notification_channel_ids,message_template=EXCLUDED.message_template,is_enabled=EXCLUDED.is_enabled,contract=EXCLUDED.contract,revision=data_alarm_items.revision+1,updated_by=EXCLUDED.updated_by,updated_at=now() WHERE data_alarm_items.project_id=EXCLUDED.project_id`, operation.ID, projectID, p.DatapointID, p.GroupID, p.DisplayName, p.NameKey, p.Description, p.Mode, p.AlarmType, p.EvaluationMode, p.DerivedExpression, p.TriggerFingerprint, p.NotificationMode, p.NotifyOnRaise, p.NotifyOnClear, p.RepeatIntervalSeconds, string(channels), p.MessageTemplate, p.IsEnabled, string(contract), actorID)
		if err != nil || tag.RowsAffected() == 0 {
			return translateAlarmWrite("同步报警配置失败", firstAlarmError(err, pgx.ErrNoRows))
		}
		if _, err = tx.Exec(ctx, `DELETE FROM data_alarm_item_inputs WHERE alarm_item_id=$1`, operation.ID); err != nil {
			return wrapAlarmRepo("清理同步报警输入失败", err)
		}
		if _, err = tx.Exec(ctx, `DELETE FROM data_alarm_item_conditions WHERE alarm_item_id=$1`, operation.ID); err != nil {
			return wrapAlarmRepo("清理同步报警条件失败", err)
		}
		for _, input := range p.Inputs {
			if _, err = tx.Exec(ctx, `INSERT INTO data_alarm_item_inputs(id,alarm_item_id,datapoint_id,input_key,sort_order) VALUES($1,$2,$3,$4,$5)`, input.ID, operation.ID, input.DatapointID, input.InputKey, input.SortOrder); err != nil {
				return translateAlarmWrite("同步报警输入失败", err)
			}
		}
		for _, condition := range p.Conditions {
			raw, marshalErr := json.Marshal(condition.Params)
			if marshalErr != nil {
				return marshalErr
			}
			if _, err = tx.Exec(ctx, `INSERT INTO data_alarm_item_conditions(id,alarm_item_id,kind,operator,label,severity,params,trigger_delay_ms,clear_delay_ms,deadband,sort_order) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11)`, condition.ID, operation.ID, condition.Kind, condition.Operator, condition.Label, condition.Severity, string(raw), condition.TriggerDelayMS, condition.ClearDelayMS, condition.Deadband, condition.SortOrder); err != nil {
				return translateAlarmWrite("同步报警条件失败", err)
			}
		}
		if len(p.Conditions) == 0 {
			return err
		}
	case "project_settings":
		p := operation.Settings
		channels, err := json.Marshal(p.DefaultChannelIDs)
		if err != nil {
			return wrapAlarmRepo("编码同步报警默认渠道失败", err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO data_alarm_project_settings(project_id,notify_on_raise,notify_on_clear,repeat_interval_seconds,default_message_template,default_channel_ids,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7,$7) ON CONFLICT(project_id) DO UPDATE SET notify_on_raise=EXCLUDED.notify_on_raise,notify_on_clear=EXCLUDED.notify_on_clear,repeat_interval_seconds=EXCLUDED.repeat_interval_seconds,default_message_template=EXCLUDED.default_message_template,default_channel_ids=EXCLUDED.default_channel_ids,updated_by=EXCLUDED.updated_by,updated_at=now()`, projectID, p.NotifyOnRaise, p.NotifyOnClear, p.RepeatIntervalSeconds, p.DefaultMessageTemplate, string(channels), actorID)
		if err != nil {
			return translateAlarmWrite("同步报警默认设置失败", err)
		}
	case "history_settings":
		p := operation.HistorySettings
		_, err := tx.Exec(ctx, `INSERT INTO data_alarm_history_settings(project_id,is_enabled,retention_days,store_notification_deliveries,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$5) ON CONFLICT(project_id) DO UPDATE SET is_enabled=EXCLUDED.is_enabled,retention_days=EXCLUDED.retention_days,store_notification_deliveries=EXCLUDED.store_notification_deliveries,updated_by=EXCLUDED.updated_by,updated_at=now()`, projectID, p.IsEnabled, p.RetentionDays, p.StoreNotificationDeliveries, actorID)
		if err != nil {
			return translateAlarmWrite("同步报警历史设置失败", err)
		}
	case "channel":
		p := operation.Channel
		config, status, err := marshalAlarmJSON(p.Config, p.SecretStatus)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO data_alarm_notification_channels(id,project_id,name,channel_type,config,secret_status,is_enabled,created_by,updated_by) VALUES($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7,$8,$8) ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,channel_type=EXCLUDED.channel_type,config=EXCLUDED.config,secret_status=EXCLUDED.secret_status,is_enabled=EXCLUDED.is_enabled,updated_by=EXCLUDED.updated_by,updated_at=now() WHERE data_alarm_notification_channels.project_id=EXCLUDED.project_id`, operation.ID, projectID, p.Name, p.ChannelType, string(config), string(status), p.IsEnabled, actorID)
		if err != nil || tag.RowsAffected() == 0 {
			return translateAlarmWrite("同步报警通知渠道失败", firstAlarmError(err, pgx.ErrNoRows))
		}
		for _, key := range p.DeleteSecretKeys {
			if _, err = tx.Exec(ctx, `DELETE FROM data_alarm_notification_channel_secrets WHERE channel_id=$1 AND secret_key=$2`, operation.ID, key); err != nil {
				return wrapAlarmRepo("同步删除报警渠道密钥失败", err)
			}
		}
		for _, secret := range p.Secrets {
			if _, err = tx.Exec(ctx, `INSERT INTO data_alarm_notification_channel_secrets(channel_id,secret_key,encrypted_value,encryption_key_version) VALUES($1,$2,$3,$4) ON CONFLICT(channel_id,secret_key) DO UPDATE SET encrypted_value=EXCLUDED.encrypted_value,encryption_key_version=EXCLUDED.encryption_key_version,updated_at=now()`, operation.ID, secret.Key, secret.Value, secret.KeyVersion); err != nil {
				return wrapAlarmRepo("同步保存报警渠道密钥失败", err)
			}
		}
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "报警配置同步资源类型不受支持")
	}
	return nil
}

func deleteAlarmSyncResource(ctx context.Context, tx pgx.Tx, projectID, resource, id string) error {
	var sqlText string
	switch resource {
	case "group":
		sqlText = `DELETE FROM data_alarm_groups WHERE project_id=$1 AND id=$2`
	case "alarm_item":
		sqlText = `DELETE FROM data_alarm_items WHERE project_id=$1 AND id=$2`
	case "project_settings":
		sqlText = `DELETE FROM data_alarm_project_settings WHERE project_id=$1`
	case "history_settings":
		sqlText = `DELETE FROM data_alarm_history_settings WHERE project_id=$1`
	case "channel":
		var used bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM data_alarm_project_settings WHERE project_id=$1 AND default_channel_ids ? $2) OR EXISTS(SELECT 1 FROM data_alarm_items WHERE project_id=$1 AND notification_channel_ids ? $2)`, projectID, id).Scan(&used); err != nil {
			return wrapAlarmRepo("检查同步通知渠道占用失败", err)
		}
		if used {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "通知渠道正在被报警设置使用")
		}
		sqlText = `DELETE FROM data_alarm_notification_channels WHERE project_id=$1 AND id=$2`
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "报警配置同步资源类型不受支持")
	}
	var tag pgconn.CommandTag
	var err error
	if resource == "project_settings" || resource == "history_settings" {
		tag, err = tx.Exec(ctx, sqlText, projectID)
	} else {
		tag, err = tx.Exec(ctx, sqlText, projectID, id)
	}
	if err != nil {
		return translateAlarmWrite("同步删除报警配置失败", err)
	}
	if tag.RowsAffected() == 0 {
		// 报警历史无显式记录本身就表示使用默认值，删除操作因此天然幂等。
		if resource == "history_settings" {
			return nil
		}
		return alarmNotFound("同步删除的报警配置不存在")
	}
	return nil
}

func validateAlarmSyncReferences(ctx context.Context, tx pgx.Tx, projectID string) error {
	var invalid bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(
        SELECT 1 FROM data_alarm_items c LEFT JOIN data_alarm_groups g ON g.id=c.group_id
        WHERE c.project_id=$1 AND c.group_id IS NOT NULL AND (g.id IS NULL OR g.project_id<>$1)
    ) OR EXISTS(
        SELECT 1 FROM data_alarm_items c JOIN data_points d ON d.id=c.datapoint_id
        WHERE c.project_id=$1 AND c.mode='point' AND d.project_id<>$1
    ) OR EXISTS(
        SELECT 1 FROM data_alarm_item_inputs i JOIN data_alarm_items c ON c.id=i.alarm_item_id JOIN data_points d ON d.id=i.datapoint_id
        WHERE c.project_id=$1 AND d.project_id<>$1
    ) OR EXISTS(
		SELECT 1 FROM data_alarm_items c, jsonb_array_elements_text(c.notification_channel_ids) AS ids(channel_id)
		WHERE c.project_id=$1 AND ids.channel_id<>'runtime_inapp' AND NOT EXISTS(
			SELECT 1 FROM data_alarm_notification_channels c WHERE c.project_id=$1 AND c.id::text=ids.channel_id AND c.is_enabled
		)
	) OR EXISTS(
		SELECT 1 FROM data_alarm_project_settings s, jsonb_array_elements_text(s.default_channel_ids) AS ids(channel_id)
		WHERE s.project_id=$1 AND ids.channel_id<>'runtime_inapp' AND NOT EXISTS(
			SELECT 1 FROM data_alarm_notification_channels c WHERE c.project_id=$1 AND c.id::text=ids.channel_id AND c.is_enabled
        )
    )`, projectID).Scan(&invalid)
	if err != nil {
		return wrapAlarmRepo("校验同步报警配置引用失败", err)
	}
	if invalid {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "同步报警配置包含跨工程点位、目录或无效通知渠道")
	}
	return nil
}

func firstAlarmError(err, fallback error) error {
	if err != nil {
		return err
	}
	return fallback
}

type alarmScanner interface{ Scan(...any) error }

func incrementAlarmConfigRevision(ctx context.Context, tx pgx.Tx, projectID string) (int64, error) {
	var rev int64
	err := tx.QueryRow(ctx, `INSERT INTO data_alarm_config_sync_state(project_id,config_revision) VALUES($1,1) ON CONFLICT(project_id) DO UPDATE SET config_revision=data_alarm_config_sync_state.config_revision+1,updated_at=now() RETURNING config_revision`, projectID).Scan(&rev)
	if err != nil {
		return 0, wrapAlarmRepo("更新报警配置版本失败", err)
	}
	return rev, nil
}

func scanAlarmProjectSettings(s alarmScanner) (AlarmProjectSettingsRecord, error) {
	var item AlarmProjectSettingsRecord
	var ids []byte
	err := s.Scan(&item.ProjectID, &item.NotifyOnRaise, &item.NotifyOnClear, &item.RepeatIntervalSeconds, &item.DefaultMessageTemplate, &ids, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	if err = json.Unmarshal(ids, &item.DefaultChannelIDs); err != nil {
		return item, wrapAlarmRepo("解析默认通知渠道失败", err)
	}
	return item, nil
}
func scanAlarmHistorySettings(s alarmScanner) (AlarmHistorySettingsRecord, error) {
	var item AlarmHistorySettingsRecord
	err := s.Scan(&item.ProjectID, &item.IsEnabled, &item.RetentionDays, &item.StoreNotificationDeliveries, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, err
	}
	if err != nil {
		return item, wrapAlarmRepo("读取报警历史设置失败", err)
	}
	return item, nil
}
func scanAlarmChannel(s alarmScanner) (AlarmNotificationChannelRecord, error) {
	var item AlarmNotificationChannelRecord
	var config, status []byte
	err := s.Scan(&item.ID, &item.ProjectID, &item.Name, &item.ChannelType, &config, &status, &item.IsEnabled, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, alarmNotFound("通知渠道不存在")
	}
	if err != nil {
		return item, wrapAlarmRepo("读取通知渠道失败", err)
	}
	if err = json.Unmarshal(config, &item.Config); err != nil {
		return item, wrapAlarmRepo("解析通知渠道配置失败", err)
	}
	if err = json.Unmarshal(status, &item.SecretStatus); err != nil {
		return item, wrapAlarmRepo("解析通知渠道密钥状态失败", err)
	}
	return item, nil
}
func marshalAlarmJSON(a, b any) ([]byte, []byte, error) {
	first, err := json.Marshal(a)
	if err != nil {
		return nil, nil, wrapAlarmRepo("编码报警数据失败", err)
	}
	second, err := json.Marshal(b)
	if err != nil {
		return nil, nil, wrapAlarmRepo("编码报警数据失败", err)
	}
	return first, second, nil
}
func alarmNotFound(message string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, message)
}
func wrapAlarmRepo(message string, err error) error {
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}
func translateAlarmWrite(message string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return alarmNotFound(strings.TrimSuffix(message, "失败") + "对象不存在")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "报警配置仍被其他资源引用或引用对象不存在")
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "报警名称或触发语义与已有配置冲突")
		case "23514", "22P02":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "报警配置参数不合法")
		}
	}
	return wrapAlarmRepo(message, err)
}
