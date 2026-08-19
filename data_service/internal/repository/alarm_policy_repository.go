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

type AlarmPolicyBindingRecord struct {
	ID          string  `json:"id"`
	DatapointID string  `json:"datapointId"`
	Path        string  `json:"path"`
	Name        string  `json:"name"`
	DataType    string  `json:"dataType"`
	Role        string  `json:"role"`
	InputKey    *string `json:"inputKey"`
	SortOrder   int     `json:"sortOrder"`
}

type AlarmPolicyConditionRecord struct {
	ID             string         `json:"id"`
	Kind           string         `json:"kind"`
	Operator       string         `json:"operator"`
	Label          string         `json:"label"`
	Severity       string         `json:"severity"`
	Params         map[string]any `json:"params"`
	TriggerDelayMS int64          `json:"triggerDelayMs"`
	ClearDelayMS   int64          `json:"clearDelayMs"`
	Deadband       float64        `json:"deadband"`
	SortOrder      int            `json:"sortOrder"`
}

type AlarmPolicyRecord struct {
	ID                     string                       `json:"id"`
	ProjectID              string                       `json:"projectId"`
	Name                   string                       `json:"name"`
	Mode                   string                       `json:"mode"`
	DerivedExpression      string                       `json:"derivedExpression"`
	GroupID                *string                      `json:"groupId"`
	GroupName              *string                      `json:"groupName"`
	Description            *string                      `json:"description"`
	NotificationMode       string                       `json:"notificationMode"`
	NotifyOnRaise          *bool                        `json:"notifyOnRaise"`
	NotifyOnClear          *bool                        `json:"notifyOnClear"`
	RepeatIntervalSeconds  *int                         `json:"repeatIntervalSeconds"`
	NotificationChannelIDs []string                     `json:"notificationChannelIds"`
	MessageTemplate        string                       `json:"messageTemplate"`
	IsEnabled              bool                         `json:"isEnabled"`
	Revision               int64                        `json:"revision"`
	Contract               map[string]any               `json:"contract"`
	Bindings               []AlarmPolicyBindingRecord   `json:"bindings"`
	Conditions             []AlarmPolicyConditionRecord `json:"conditions"`
	CreatedAt              time.Time                    `json:"createdAt"`
	UpdatedAt              time.Time                    `json:"updatedAt"`
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

type AlarmPolicyListFilter struct {
	Search, Severity, ConditionKind, Mode string
	GroupID                               *string
	Enabled                               *bool
	DatapointID                           string
	Page, PageSize                        int
}

type AlarmPolicyGroupListFilter struct {
	Search   string
	ParentID *string
	Page     int
	PageSize int
}

type SaveAlarmPolicyParams struct {
	ID, ProjectID, UserID, Name, Mode, DerivedExpression string
	IsCreate                                             bool
	GroupID, Description                                 *string
	NotificationMode                                     string
	NotifyOnRaise, NotifyOnClear                         *bool
	RepeatIntervalSeconds                                *int
	NotificationChannelIDs                               []string
	MessageTemplate                                      string
	IsEnabled                                            bool
	Contract                                             map[string]any
	Bindings                                             []AlarmPolicyBindingRecord
	Conditions                                           []AlarmPolicyConditionRecord
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
	Policy          *SaveAlarmPolicyParams
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
        SELECT id, project_id, name, parent_id, name::text AS full_path FROM data_alarm_policy_groups WHERE project_id = $1 AND parent_id IS NULL
        UNION ALL
        SELECT c.id, c.project_id, c.name, c.parent_id, (p.full_path || ' / ' || c.name) FROM data_alarm_policy_groups c JOIN paths p ON p.id = c.parent_id WHERE c.project_id = $1
    ) `
	var total int
	if err := r.pool.QueryRow(ctx, cte+`SELECT COUNT(*) FROM data_alarm_policy_groups g JOIN paths ON paths.id=g.id WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, wrapAlarmRepo("统计报警目录失败", err)
	}
	queryArgs := append(append([]any{}, args...), size, (page-1)*size)
	rows, err := r.pool.Query(ctx, cte+`SELECT g.id,g.project_id,g.name,g.parent_id,g.description,g.sort_order,paths.full_path,
        EXISTS(SELECT 1 FROM data_alarm_policy_groups c WHERE c.parent_id=g.id),g.created_at,g.updated_at
        FROM data_alarm_policy_groups g JOIN paths ON paths.id=g.id WHERE `+whereSQL+
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
        SELECT id,project_id,name,parent_id,description,sort_order,name::text AS full_path,created_at,updated_at FROM data_alarm_policy_groups WHERE project_id=$1 AND parent_id IS NULL
        UNION ALL SELECT c.id,c.project_id,c.name,c.parent_id,c.description,c.sort_order,(p.full_path || ' / ' || c.name),c.created_at,c.updated_at FROM data_alarm_policy_groups c JOIN paths p ON p.id=c.parent_id WHERE c.project_id=$1
    ) SELECT id,project_id,name,parent_id,description,sort_order,full_path,EXISTS(SELECT 1 FROM data_alarm_policy_groups c WHERE c.parent_id=paths.id),created_at,updated_at FROM paths ORDER BY full_path`, projectID)
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
		err = tx.QueryRow(ctx, `INSERT INTO data_alarm_policy_groups(project_id,name,parent_id,description,sort_order,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$6) RETURNING id`, p.ProjectID, p.Name, p.ParentID, p.Description, p.SortOrder, p.UserID).Scan(&id)
	} else {
		id = p.ID
		tag, execErr := tx.Exec(ctx, `UPDATE data_alarm_policy_groups SET name=$3,parent_id=$4,description=$5,sort_order=$6,updated_by=$7,updated_at=now() WHERE project_id=$1 AND id=$2`, p.ProjectID, p.ID, p.Name, p.ParentID, p.Description, p.SortOrder, p.UserID)
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
	tag, err := tx.Exec(ctx, `DELETE FROM data_alarm_policy_groups WHERE project_id=$1 AND id=$2`, projectID, id)
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

func (r *AlarmPolicyRepository) ListPolicies(ctx context.Context, projectID string, f AlarmPolicyListFilter) ([]AlarmPolicyRecord, int, error) {
	page, size := normalizePageAndSize(f.Page, f.PageSize, 20, 100)
	where, args := buildAlarmPolicyWhere(projectID, f)
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_alarm_policies ap WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, wrapAlarmRepo("统计报警策略失败", err)
	}
	queryArgs := append(append([]any{}, args...), size, (page-1)*size)
	rows, err := r.pool.Query(ctx, alarmPolicyBaseSelect()+` WHERE `+where+fmt.Sprintf(` ORDER BY ap.updated_at DESC,ap.created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2), queryArgs...)
	if err != nil {
		return nil, 0, wrapAlarmRepo("查询报警策略失败", err)
	}
	defer rows.Close()
	items, err := scanAlarmPolicyRows(rows)
	if err != nil {
		return nil, 0, err
	}
	if err = r.loadPolicyDetails(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *AlarmPolicyRepository) ListAllPolicies(ctx context.Context, projectID string) ([]AlarmPolicyRecord, error) {
	rows, err := r.pool.Query(ctx, alarmPolicyBaseSelect()+` WHERE ap.project_id=$1 ORDER BY ap.created_at`, projectID)
	if err != nil {
		return nil, wrapAlarmRepo("查询全部报警策略失败", err)
	}
	defer rows.Close()
	items, err := scanAlarmPolicyRows(rows)
	if err != nil {
		return nil, err
	}
	if err = r.loadPolicyDetails(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AlarmPolicyRepository) GetPolicy(ctx context.Context, projectID, id string) (*AlarmPolicyRecord, error) {
	item, err := scanAlarmPolicy(r.pool.QueryRow(ctx, alarmPolicyBaseSelect()+` WHERE ap.project_id=$1 AND ap.id=$2`, projectID, id))
	if err != nil {
		return nil, err
	}
	items := []AlarmPolicyRecord{item}
	if err = r.loadPolicyDetails(ctx, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *AlarmPolicyRepository) ListPoliciesByDatapoint(ctx context.Context, projectID, datapointID string) ([]AlarmPolicyRecord, error) {
	rows, err := r.pool.Query(ctx, alarmPolicyBaseSelect()+` WHERE ap.project_id=$1 AND EXISTS(SELECT 1 FROM data_alarm_policy_bindings b WHERE b.policy_id=ap.id AND b.datapoint_id=$2) ORDER BY ap.name`, projectID, datapointID)
	if err != nil {
		return nil, wrapAlarmRepo("查询数据点报警策略失败", err)
	}
	defer rows.Close()
	items, err := scanAlarmPolicyRows(rows)
	if err != nil {
		return nil, err
	}
	if err = r.loadPolicyDetails(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AlarmPolicyRepository) SavePolicy(ctx context.Context, p SaveAlarmPolicyParams) (*AlarmPolicyRecord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, wrapAlarmRepo("开启报警策略事务失败", err)
	}
	defer tx.Rollback(ctx)
	channelsJSON, contractJSON, err := marshalAlarmJSON(p.NotificationChannelIDs, p.Contract)
	if err != nil {
		return nil, err
	}
	id := p.ID
	if p.IsCreate {
		err = tx.QueryRow(ctx, `INSERT INTO data_alarm_policies(id,project_id,group_id,name,description,mode,derived_expression,notification_mode,notify_on_raise,notify_on_clear,repeat_interval_seconds,notification_channel_ids,message_template,is_enabled,contract,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13,$14,$15::jsonb,$16,$16) RETURNING id`, id, p.ProjectID, p.GroupID, p.Name, p.Description, p.Mode, p.DerivedExpression, p.NotificationMode, p.NotifyOnRaise, p.NotifyOnClear, p.RepeatIntervalSeconds, string(channelsJSON), p.MessageTemplate, p.IsEnabled, string(contractJSON), p.UserID).Scan(&id)
	} else {
		tag, execErr := tx.Exec(ctx, `UPDATE data_alarm_policies SET group_id=$3,name=$4,description=$5,mode=$6,derived_expression=$7,notification_mode=$8,notify_on_raise=$9,notify_on_clear=$10,repeat_interval_seconds=$11,notification_channel_ids=$12::jsonb,message_template=$13,is_enabled=$14,contract=$15::jsonb,revision=revision+1,updated_by=$16,updated_at=now() WHERE project_id=$1 AND id=$2`, p.ProjectID, id, p.GroupID, p.Name, p.Description, p.Mode, p.DerivedExpression, p.NotificationMode, p.NotifyOnRaise, p.NotifyOnClear, p.RepeatIntervalSeconds, string(channelsJSON), p.MessageTemplate, p.IsEnabled, string(contractJSON), p.UserID)
		err = execErr
		if err == nil && tag.RowsAffected() == 0 {
			err = pgx.ErrNoRows
		}
	}
	if err != nil {
		return nil, translateAlarmWrite("保存报警策略失败", err)
	}
	if err = replaceAlarmPolicyDetails(ctx, tx, id, p.Bindings, p.Conditions); err != nil {
		return nil, err
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, p.ProjectID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, wrapAlarmRepo("提交报警策略事务失败", err)
	}
	return r.GetPolicy(ctx, p.ProjectID, id)
}

func (r *AlarmPolicyRepository) SetPolicyEnabled(ctx context.Context, projectID, id, userID string, enabled bool) (*AlarmPolicyRecord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, wrapAlarmRepo("开启报警策略事务失败", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE data_alarm_policies SET is_enabled=$3,revision=revision+1,updated_by=$4,updated_at=now() WHERE project_id=$1 AND id=$2`, projectID, id, enabled, userID)
	if err != nil {
		return nil, translateAlarmWrite("更新报警策略状态失败", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, alarmNotFound("报警策略不存在")
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, projectID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, wrapAlarmRepo("提交报警策略状态失败", err)
	}
	return r.GetPolicy(ctx, projectID, id)
}

func (r *AlarmPolicyRepository) DeletePolicy(ctx context.Context, projectID, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return wrapAlarmRepo("开启删除报警策略事务失败", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `DELETE FROM data_alarm_policies WHERE project_id=$1 AND id=$2`, projectID, id)
	if err != nil {
		return translateAlarmWrite("删除报警策略失败", err)
	}
	if tag.RowsAffected() == 0 {
		return alarmNotFound("报警策略不存在")
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, projectID); err != nil {
		return err
	}
	return tx.Commit(ctx)
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
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM data_alarm_project_settings WHERE project_id=$1 AND default_channel_ids ? $2) OR EXISTS(SELECT 1 FROM data_alarm_policies WHERE project_id=$1 AND notification_channel_ids ? $2)`, projectID, id).Scan(&used)
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
		tag, err := tx.Exec(ctx, `INSERT INTO data_alarm_policy_groups(id,project_id,parent_id,name,description,sort_order,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$7) ON CONFLICT(id) DO UPDATE SET parent_id=EXCLUDED.parent_id,name=EXCLUDED.name,description=EXCLUDED.description,sort_order=EXCLUDED.sort_order,updated_by=EXCLUDED.updated_by,updated_at=now() WHERE data_alarm_policy_groups.project_id=EXCLUDED.project_id`, operation.ID, projectID, p.ParentID, p.Name, p.Description, p.SortOrder, actorID)
		if err != nil || tag.RowsAffected() == 0 {
			return translateAlarmWrite("同步报警目录失败", firstAlarmError(err, pgx.ErrNoRows))
		}
	case "policy":
		p := operation.Policy
		channels, contract, err := marshalAlarmJSON(p.NotificationChannelIDs, p.Contract)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO data_alarm_policies(id,project_id,group_id,name,description,mode,derived_expression,notification_mode,notify_on_raise,notify_on_clear,repeat_interval_seconds,notification_channel_ids,message_template,is_enabled,contract,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13,$14,$15::jsonb,$16,$16) ON CONFLICT(id) DO UPDATE SET group_id=EXCLUDED.group_id,name=EXCLUDED.name,description=EXCLUDED.description,mode=EXCLUDED.mode,derived_expression=EXCLUDED.derived_expression,notification_mode=EXCLUDED.notification_mode,notify_on_raise=EXCLUDED.notify_on_raise,notify_on_clear=EXCLUDED.notify_on_clear,repeat_interval_seconds=EXCLUDED.repeat_interval_seconds,notification_channel_ids=EXCLUDED.notification_channel_ids,message_template=EXCLUDED.message_template,is_enabled=EXCLUDED.is_enabled,contract=EXCLUDED.contract,revision=data_alarm_policies.revision+1,updated_by=EXCLUDED.updated_by,updated_at=now() WHERE data_alarm_policies.project_id=EXCLUDED.project_id`, operation.ID, projectID, p.GroupID, p.Name, p.Description, p.Mode, p.DerivedExpression, p.NotificationMode, p.NotifyOnRaise, p.NotifyOnClear, p.RepeatIntervalSeconds, string(channels), p.MessageTemplate, p.IsEnabled, string(contract), actorID)
		if err != nil || tag.RowsAffected() == 0 {
			return translateAlarmWrite("同步报警策略失败", firstAlarmError(err, pgx.ErrNoRows))
		}
		if err = replaceAlarmPolicyDetails(ctx, tx, operation.ID, p.Bindings, p.Conditions); err != nil {
			return err
		}
	case "settings":
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
		sqlText = `DELETE FROM data_alarm_policy_groups WHERE project_id=$1 AND id=$2`
	case "policy":
		sqlText = `DELETE FROM data_alarm_policies WHERE project_id=$1 AND id=$2`
	case "settings":
		sqlText = `DELETE FROM data_alarm_project_settings WHERE project_id=$1`
	case "history_settings":
		sqlText = `DELETE FROM data_alarm_history_settings WHERE project_id=$1`
	case "channel":
		var used bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM data_alarm_project_settings WHERE project_id=$1 AND default_channel_ids ? $2) OR EXISTS(SELECT 1 FROM data_alarm_policies WHERE project_id=$1 AND notification_channel_ids ? $2)`, projectID, id).Scan(&used); err != nil {
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
	if resource == "settings" || resource == "history_settings" {
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
        SELECT 1 FROM data_alarm_policies p LEFT JOIN data_alarm_policy_groups g ON g.id=p.group_id
        WHERE p.project_id=$1 AND p.group_id IS NOT NULL AND (g.id IS NULL OR g.project_id<>$1)
    ) OR EXISTS(
        SELECT 1 FROM data_alarm_policy_bindings b JOIN data_alarm_policies p ON p.id=b.policy_id JOIN data_points d ON d.id=b.datapoint_id
        WHERE p.project_id=$1 AND d.project_id<>$1
    ) OR EXISTS(
		SELECT 1 FROM data_alarm_policies p, jsonb_array_elements_text(p.notification_channel_ids) AS ids(channel_id)
		WHERE p.project_id=$1 AND ids.channel_id<>'runtime_inapp' AND NOT EXISTS(
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

func buildAlarmPolicyWhere(projectID string, f AlarmPolicyListFilter) (string, []any) {
	args := []any{projectID}
	parts := []string{"ap.project_id=$1"}
	add := func(sql string, v any) {
		args = append(args, v)
		position := len(args)
		parts = append(parts, fmt.Sprintf(sql, position, position, position))
	}
	if v := strings.TrimSpace(f.Search); v != "" {
		add("(ap.name ILIKE $%d OR EXISTS(SELECT 1 FROM data_alarm_policy_bindings b JOIN data_points dp ON dp.id=b.datapoint_id WHERE b.policy_id=ap.id AND (dp.name ILIKE $%d OR dp.path ILIKE $%d)))", "%"+v+"%")
	}
	if f.GroupID != nil {
		add("ap.group_id IN (WITH RECURSIVE gt AS (SELECT id FROM data_alarm_policy_groups WHERE project_id=$1 AND id=$%d UNION ALL SELECT c.id FROM data_alarm_policy_groups c JOIN gt ON c.parent_id=gt.id WHERE c.project_id=$1) SELECT id FROM gt)", *f.GroupID)
	}
	if f.Enabled != nil {
		add("ap.is_enabled=$%d", *f.Enabled)
	}
	if f.Mode != "" {
		add("ap.mode=$%d", f.Mode)
	}
	if f.Severity != "" {
		add("EXISTS(SELECT 1 FROM data_alarm_policy_conditions c WHERE c.policy_id=ap.id AND c.severity=$%d)", f.Severity)
	}
	if f.ConditionKind != "" {
		add("EXISTS(SELECT 1 FROM data_alarm_policy_conditions c WHERE c.policy_id=ap.id AND c.kind=$%d)", f.ConditionKind)
	}
	if f.DatapointID != "" {
		add("EXISTS(SELECT 1 FROM data_alarm_policy_bindings b WHERE b.policy_id=ap.id AND b.datapoint_id=$%d)", f.DatapointID)
	}
	return strings.Join(parts, " AND "), args
}

func alarmPolicyBaseSelect() string {
	return `SELECT ap.id,ap.project_id,ap.group_id,g.name,ap.name,ap.description,ap.mode,ap.derived_expression,ap.notification_mode,ap.notify_on_raise,ap.notify_on_clear,ap.repeat_interval_seconds,ap.notification_channel_ids,ap.message_template,ap.is_enabled,ap.revision,ap.contract,ap.created_at,ap.updated_at FROM data_alarm_policies ap LEFT JOIN data_alarm_policy_groups g ON g.id=ap.group_id`
}

type alarmScanner interface{ Scan(...any) error }

func scanAlarmPolicy(s alarmScanner) (AlarmPolicyRecord, error) {
	var item AlarmPolicyRecord
	var channels, contract []byte
	err := s.Scan(&item.ID, &item.ProjectID, &item.GroupID, &item.GroupName, &item.Name, &item.Description, &item.Mode, &item.DerivedExpression, &item.NotificationMode, &item.NotifyOnRaise, &item.NotifyOnClear, &item.RepeatIntervalSeconds, &channels, &item.MessageTemplate, &item.IsEnabled, &item.Revision, &contract, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, alarmNotFound("报警策略不存在")
	}
	if err != nil {
		return item, wrapAlarmRepo("读取报警策略失败", err)
	}
	if err = json.Unmarshal(channels, &item.NotificationChannelIDs); err != nil {
		return item, wrapAlarmRepo("解析通知渠道失败", err)
	}
	if err = json.Unmarshal(contract, &item.Contract); err != nil {
		return item, wrapAlarmRepo("解析报警契约失败", err)
	}
	return item, nil
}
func scanAlarmPolicyRows(rows pgx.Rows) ([]AlarmPolicyRecord, error) {
	items := make([]AlarmPolicyRecord, 0)
	for rows.Next() {
		item, err := scanAlarmPolicy(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapAlarmRepo("遍历报警策略失败", err)
	}
	return items, nil
}

func (r *AlarmPolicyRepository) loadPolicyDetails(ctx context.Context, items []AlarmPolicyRecord) error {
	for i := range items {
		bindings, err := r.loadBindings(ctx, items[i].ID)
		if err != nil {
			return err
		}
		conditions, err := r.loadConditions(ctx, items[i].ID)
		if err != nil {
			return err
		}
		items[i].Bindings = bindings
		items[i].Conditions = conditions
	}
	return nil
}
func (r *AlarmPolicyRepository) loadBindings(ctx context.Context, policyID string) ([]AlarmPolicyBindingRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT b.id,b.datapoint_id,dp.path,dp.name,dp.data_type,b.role,b.input_key,b.sort_order FROM data_alarm_policy_bindings b JOIN data_points dp ON dp.id=b.datapoint_id WHERE b.policy_id=$1 ORDER BY b.role DESC,b.sort_order,b.created_at`, policyID)
	if err != nil {
		return nil, wrapAlarmRepo("查询报警绑定失败", err)
	}
	defer rows.Close()
	items := make([]AlarmPolicyBindingRecord, 0)
	for rows.Next() {
		var item AlarmPolicyBindingRecord
		if err := rows.Scan(&item.ID, &item.DatapointID, &item.Path, &item.Name, &item.DataType, &item.Role, &item.InputKey, &item.SortOrder); err != nil {
			return nil, wrapAlarmRepo("读取报警绑定失败", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *AlarmPolicyRepository) loadConditions(ctx context.Context, policyID string) ([]AlarmPolicyConditionRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,kind,operator,label,severity,params,trigger_delay_ms,clear_delay_ms,deadband,sort_order FROM data_alarm_policy_conditions WHERE policy_id=$1 ORDER BY sort_order,created_at`, policyID)
	if err != nil {
		return nil, wrapAlarmRepo("查询报警条件失败", err)
	}
	defer rows.Close()
	items := make([]AlarmPolicyConditionRecord, 0)
	for rows.Next() {
		var item AlarmPolicyConditionRecord
		var params []byte
		if err := rows.Scan(&item.ID, &item.Kind, &item.Operator, &item.Label, &item.Severity, &params, &item.TriggerDelayMS, &item.ClearDelayMS, &item.Deadband, &item.SortOrder); err != nil {
			return nil, wrapAlarmRepo("读取报警条件失败", err)
		}
		if err = json.Unmarshal(params, &item.Params); err != nil {
			return nil, wrapAlarmRepo("解析报警条件失败", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func replaceAlarmPolicyDetails(ctx context.Context, tx pgx.Tx, policyID string, bindings []AlarmPolicyBindingRecord, conditions []AlarmPolicyConditionRecord) error {
	if _, err := tx.Exec(ctx, `DELETE FROM data_alarm_policy_bindings WHERE policy_id=$1`, policyID); err != nil {
		return wrapAlarmRepo("替换报警绑定失败", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM data_alarm_policy_conditions WHERE policy_id=$1`, policyID); err != nil {
		return wrapAlarmRepo("替换报警条件失败", err)
	}
	for i, b := range bindings {
		if _, err := tx.Exec(ctx, `INSERT INTO data_alarm_policy_bindings(policy_id,datapoint_id,role,input_key,sort_order) VALUES($1,$2,$3,$4,$5)`, policyID, b.DatapointID, b.Role, b.InputKey, i); err != nil {
			return translateAlarmWrite("保存报警绑定失败", err)
		}
	}
	for i, c := range conditions {
		params, err := json.Marshal(c.Params)
		if err != nil {
			return wrapAlarmRepo("编码报警条件失败", err)
		}
		if c.ID == "" {
			_, err = tx.Exec(ctx, `INSERT INTO data_alarm_policy_conditions(policy_id,kind,operator,label,severity,params,trigger_delay_ms,clear_delay_ms,deadband,sort_order) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10)`, policyID, c.Kind, c.Operator, c.Label, c.Severity, string(params), c.TriggerDelayMS, c.ClearDelayMS, c.Deadband, i)
		} else {
			_, err = tx.Exec(ctx, `INSERT INTO data_alarm_policy_conditions(id,policy_id,kind,operator,label,severity,params,trigger_delay_ms,clear_delay_ms,deadband,sort_order) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11)`, c.ID, policyID, c.Kind, c.Operator, c.Label, c.Severity, string(params), c.TriggerDelayMS, c.ClearDelayMS, c.Deadband, i)
		}
		if err != nil {
			return translateAlarmWrite("保存报警条件失败", err)
		}
	}
	return nil
}

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
		case "23505", "23514", "22P02":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "报警配置不合法或名称重复")
		}
	}
	return wrapAlarmRepo(message, err)
}
