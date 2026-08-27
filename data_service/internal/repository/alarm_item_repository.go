package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type AlarmItemInputRecord struct {
	ID          string `json:"id"`
	DatapointID string `json:"datapointId"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	DataType    string `json:"dataType"`
	InputKey    string `json:"inputKey"`
	SortOrder   int    `json:"sortOrder"`
}

type AlarmItemConditionRecord struct {
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

type AlarmItemRecord struct {
	ID                     string                     `json:"id"`
	ProjectID              string                     `json:"projectId"`
	DisplayName            string                     `json:"displayName"`
	NameKey                string                     `json:"nameKey"`
	Mode                   string                     `json:"mode"`
	AlarmType              string                     `json:"alarmType"`
	PresetSlot             *string                    `json:"presetSlot"`
	EvaluationMode         string                     `json:"evaluationMode"`
	DerivedExpression      string                     `json:"derivedExpression"`
	TriggerFingerprint     string                     `json:"triggerFingerprint"`
	DatapointID            *string                    `json:"datapointId"`
	Path                   *string                    `json:"path"`
	DatapointName          *string                    `json:"datapointName"`
	DataType               *string                    `json:"dataType"`
	GroupID                *string                    `json:"groupId"`
	GroupName              *string                    `json:"groupName"`
	Description            *string                    `json:"description"`
	NotificationMode       string                     `json:"notificationMode"`
	MessageTemplate        string                     `json:"messageTemplate"`
	NotifyOnRaise          *bool                      `json:"notifyOnRaise"`
	NotifyOnClear          *bool                      `json:"notifyOnClear"`
	RepeatIntervalSeconds  *int                       `json:"repeatIntervalSeconds"`
	NotificationChannelIDs []string                   `json:"notificationChannelIds"`
	IsEnabled              bool                       `json:"isEnabled"`
	Revision               int64                      `json:"revision"`
	Contract               map[string]any             `json:"contract"`
	Inputs                 []AlarmItemInputRecord     `json:"inputs"`
	Conditions             []AlarmItemConditionRecord `json:"conditions"`
	CreatedAt              time.Time                  `json:"createdAt"`
	UpdatedAt              time.Time                  `json:"updatedAt"`
}

type AlarmItemListFilter struct {
	Search, Severity, AlarmType, Mode, DatapointID string
	GroupID                                        *string
	Enabled                                        *bool
	Page, PageSize                                 int
}

type SaveAlarmItemParams struct {
	ID, ProjectID, UserID, DisplayName, NameKey, Mode, AlarmType, EvaluationMode string
	PresetSlot                                                                   *string
	DerivedExpression, TriggerFingerprint                                        string
	DatapointID, GroupID, Description                                            *string
	NotificationMode, MessageTemplate                                            string
	NotifyOnRaise, NotifyOnClear                                                 *bool
	RepeatIntervalSeconds                                                        *int
	NotificationChannelIDs                                                       []string
	IsEnabled                                                                    bool
	Revision                                                                     int64
	Contract                                                                     map[string]any
	Inputs                                                                       []AlarmItemInputRecord
	Conditions                                                                   []AlarmItemConditionRecord
	IsCreate                                                                     bool
}

type AlarmItemConflictRecord struct {
	Kind, DatapointID, DatapointName, AlarmItemID, AlarmItemName string
}

type AlarmItemOverlapRecord struct {
	DatapointID, DatapointName, AlarmItemID, AlarmItemName string
	Conditions                                             []AlarmItemConditionRecord
}

func (r *AlarmRepository) ListAlarmItems(ctx context.Context, projectID string, filter AlarmItemListFilter) ([]AlarmItemRecord, int, error) {
	page, size := normalizePageAndSize(filter.Page, filter.PageSize, 20, 100)
	where, args := buildAlarmItemWhere(projectID, filter)
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_alarm_items ai LEFT JOIN data_points dp ON dp.id=ai.datapoint_id WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, wrapAlarmRepo("统计报警项失败", err)
	}
	queryArgs := append(append([]any{}, args...), size, (page-1)*size)
	rows, err := r.pool.Query(ctx, alarmItemBaseSelect()+` WHERE `+where+fmt.Sprintf(` ORDER BY ai.updated_at DESC,ai.created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2), queryArgs...)
	if err != nil {
		return nil, 0, wrapAlarmRepo("查询报警项失败", err)
	}
	defer rows.Close()
	items, err := scanAlarmItemRows(rows)
	if err != nil {
		return nil, 0, err
	}
	if err = r.loadAlarmItemDetails(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *AlarmRepository) ListAllAlarmItems(ctx context.Context, projectID string) ([]AlarmItemRecord, error) {
	rows, err := r.pool.Query(ctx, alarmItemBaseSelect()+` WHERE ai.project_id=$1 ORDER BY ai.created_at`, projectID)
	if err != nil {
		return nil, wrapAlarmRepo("查询全部报警项失败", err)
	}
	defer rows.Close()
	items, err := scanAlarmItemRows(rows)
	if err != nil {
		return nil, err
	}
	if err = r.loadAlarmItemDetails(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

// ListAlarmItemsForBatch 由服务端解析显式 ID 或当前筛选结果，避免前端为“全部结果”拉取全量数据。
func (r *AlarmRepository) ListAlarmItemsForBatch(ctx context.Context, projectID string, ids []string, filter *AlarmItemListFilter) ([]AlarmItemRecord, error) {
	where := "ai.project_id=$1"
	args := []any{projectID}
	if len(ids) > 0 {
		where += " AND ai.id=ANY($2::uuid[])"
		args = append(args, ids)
	} else if filter != nil {
		where, args = buildAlarmItemWhere(projectID, *filter)
	} else {
		return []AlarmItemRecord{}, nil
	}
	rows, err := r.pool.Query(ctx, alarmItemBaseSelect()+` WHERE `+where+` ORDER BY ai.created_at,ai.id`, args...)
	if err != nil {
		return nil, wrapAlarmRepo("查询批量报警项失败", err)
	}
	defer rows.Close()
	items, err := scanAlarmItemRows(rows)
	if err != nil {
		return nil, err
	}
	if err = r.loadAlarmItemDetails(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AlarmRepository) GetAlarmItem(ctx context.Context, projectID, id string) (*AlarmItemRecord, error) {
	item, err := scanAlarmItem(r.pool.QueryRow(ctx, alarmItemBaseSelect()+` WHERE ai.project_id=$1 AND ai.id=$2`, projectID, id))
	if err != nil {
		return nil, err
	}
	items := []AlarmItemRecord{item}
	if err = r.loadAlarmItemDetails(ctx, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *AlarmRepository) ListAlarmItemsByDatapoint(ctx context.Context, projectID, datapointID string) ([]AlarmItemRecord, error) {
	rows, err := r.pool.Query(ctx, alarmItemBaseSelect()+` WHERE ai.project_id=$1 AND (ai.datapoint_id=$2 OR EXISTS(SELECT 1 FROM data_alarm_item_inputs i WHERE i.alarm_item_id=ai.id AND i.datapoint_id=$2)) ORDER BY ai.display_name`, projectID, datapointID)
	if err != nil {
		return nil, wrapAlarmRepo("查询数据点报警项失败", err)
	}
	defer rows.Close()
	items, err := scanAlarmItemRows(rows)
	if err != nil {
		return nil, err
	}
	if err = r.loadAlarmItemDetails(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

// ListPresetAlarmItemsForDatapoints 只返回默认配置生成的报警项，高级配置的普通报警项不会参与覆盖保存。
func (r *AlarmRepository) ListPresetAlarmItemsForDatapoints(ctx context.Context, projectID string, datapointIDs []string) ([]AlarmItemRecord, error) {
	rows, err := r.pool.Query(ctx, alarmItemBaseSelect()+` WHERE ai.project_id=$1 AND ai.datapoint_id=ANY($2::uuid[]) AND ai.preset_slot IS NOT NULL ORDER BY ai.datapoint_id,ai.preset_slot`, projectID, datapointIDs)
	if err != nil {
		return nil, wrapAlarmRepo("查询默认报警配置失败", err)
	}
	defer rows.Close()
	items, err := scanAlarmItemRows(rows)
	if err != nil {
		return nil, err
	}
	if err = r.loadAlarmItemDetails(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AlarmRepository) FindAlarmItemConflicts(ctx context.Context, projectID, excludeID string, datapointID *string, nameKey, fingerprint string, derived bool) ([]AlarmItemConflictRecord, error) {
	if derived {
		rows, err := r.pool.Query(ctx, `SELECT CASE WHEN name_key=$3 THEN 'name' ELSE 'duplicate' END,id,display_name FROM data_alarm_items WHERE project_id=$1 AND mode='derived' AND id<>$2 AND (name_key=$3 OR trigger_fingerprint=$4)`, projectID, excludeID, nameKey, fingerprint)
		if err != nil {
			return nil, wrapAlarmRepo("检查组合报警冲突失败", err)
		}
		defer rows.Close()
		result := []AlarmItemConflictRecord{}
		for rows.Next() {
			var item AlarmItemConflictRecord
			if err = rows.Scan(&item.Kind, &item.AlarmItemID, &item.AlarmItemName); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}
	if datapointID == nil {
		return []AlarmItemConflictRecord{}, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT CASE WHEN ai.name_key=$4 THEN 'name' ELSE 'duplicate' END,ai.datapoint_id,dp.name,ai.id,ai.display_name FROM data_alarm_items ai JOIN data_points dp ON dp.id=ai.datapoint_id WHERE ai.project_id=$1 AND ai.id<>$2 AND ai.datapoint_id=$3 AND (ai.name_key=$4 OR ai.trigger_fingerprint=$5)`, projectID, excludeID, *datapointID, nameKey, fingerprint)
	if err != nil {
		return nil, wrapAlarmRepo("检查数据点报警冲突失败", err)
	}
	defer rows.Close()
	result := []AlarmItemConflictRecord{}
	for rows.Next() {
		var item AlarmItemConflictRecord
		if err = rows.Scan(&item.Kind, &item.DatapointID, &item.DatapointName, &item.AlarmItemID, &item.AlarmItemName); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *AlarmRepository) FindAlarmItemOverlapCandidates(ctx context.Context, projectID, excludeID, datapointID, fingerprint string) ([]AlarmItemOverlapRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT ai.datapoint_id,dp.name,ai.id,ai.display_name FROM data_alarm_items ai JOIN data_points dp ON dp.id=ai.datapoint_id WHERE ai.project_id=$1 AND ai.mode='point' AND ai.id<>$2 AND ai.datapoint_id=$3 AND ai.trigger_fingerprint<>$4 ORDER BY ai.created_at`, projectID, excludeID, datapointID, fingerprint)
	if err != nil {
		return nil, wrapAlarmRepo("检查报警条件重叠失败", err)
	}
	defer rows.Close()
	result := []AlarmItemOverlapRecord{}
	for rows.Next() {
		var item AlarmItemOverlapRecord
		if err = rows.Scan(&item.DatapointID, &item.DatapointName, &item.AlarmItemID, &item.AlarmItemName); err != nil {
			return nil, err
		}
		conditions, conditionErr := r.listAlarmItemConditions(ctx, item.AlarmItemID)
		if conditionErr != nil {
			return nil, conditionErr
		}
		item.Conditions = conditions
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *AlarmRepository) NextAlarmItemDisplayName(ctx context.Context, projectID, datapointID, base string) (string, error) {
	rows, err := r.pool.Query(ctx, `SELECT display_name FROM data_alarm_items WHERE project_id=$1 AND datapoint_id=$2 AND (display_name=$3 OR display_name LIKE $3 || ' %')`, projectID, datapointID, base)
	if err != nil {
		return "", wrapAlarmRepo("生成报警显示名称失败", err)
	}
	defer rows.Close()
	used := map[string]bool{}
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return "", err
		}
		used[name] = true
	}
	if !used[base] {
		return base, rows.Err()
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s %d", base, index)
		if !used[candidate] {
			return candidate, rows.Err()
		}
	}
}

func (r *AlarmRepository) SaveAlarmItem(ctx context.Context, p SaveAlarmItemParams) (*AlarmItemRecord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, wrapAlarmRepo("开启报警项事务失败", err)
	}
	defer tx.Rollback(ctx)
	if err = saveAlarmItemTx(ctx, tx, p); err != nil {
		return nil, err
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, p.ProjectID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, wrapAlarmRepo("提交报警项事务失败", err)
	}
	return r.GetAlarmItem(ctx, p.ProjectID, p.ID)
}

func (r *AlarmRepository) BatchCreateAlarmItems(ctx context.Context, projectID, userID string, items []SaveAlarmItemParams) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, wrapAlarmRepo("开启批量报警事务失败", err)
	}
	defer tx.Rollback(ctx)
	ids := make([]string, 0, len(items))
	for _, item := range items {
		item.ProjectID, item.UserID, item.IsCreate = projectID, userID, true
		if err = saveAlarmItemTx(ctx, tx, item); err != nil {
			return nil, err
		}
		ids = append(ids, item.ID)
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, projectID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, wrapAlarmRepo("提交批量报警事务失败", err)
	}
	return ids, nil
}

func (r *AlarmRepository) BatchSaveAlarmItems(ctx context.Context, projectID, userID string, items []SaveAlarmItemParams) error {
	for index := range items {
		items[index].IsCreate = false
	}
	return r.BatchUpsertAlarmItems(ctx, projectID, userID, items)
}

func (r *AlarmRepository) BatchUpsertAlarmItems(ctx context.Context, projectID, userID string, items []SaveAlarmItemParams) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return wrapAlarmRepo("开启批量修改事务失败", err)
	}
	defer tx.Rollback(ctx)
	for _, item := range items {
		item.ProjectID, item.UserID = projectID, userID
		if err = saveAlarmItemTx(ctx, tx, item); err != nil {
			return err
		}
	}
	if len(items) > 0 {
		if _, err = incrementAlarmConfigRevision(ctx, tx, projectID); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return wrapAlarmRepo("提交批量修改事务失败", err)
	}
	return nil
}

// ReplacePresetAlarmItems 在一个事务内替换所选数据点的全部默认报警配置；preset_slot 为空的高级报警项保持不变。
func (r *AlarmRepository) ReplacePresetAlarmItems(ctx context.Context, projectID, userID string, datapointIDs []string, items []SaveAlarmItemParams) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, wrapAlarmRepo("开启默认报警配置事务失败", err)
	}
	defer tx.Rollback(ctx)
	keepIDs := make([]string, 0, len(items))
	for _, item := range items {
		if !item.IsCreate {
			keepIDs = append(keepIDs, item.ID)
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM data_alarm_items WHERE project_id=$1 AND datapoint_id=ANY($2::uuid[]) AND preset_slot IS NOT NULL AND NOT (id=ANY($3::uuid[]))`, projectID, datapointIDs, keepIDs); err != nil {
		return nil, translateAlarmWrite("清理原默认报警配置失败", err)
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		item.ProjectID, item.UserID = projectID, userID
		if err = saveAlarmItemTx(ctx, tx, item); err != nil {
			return nil, err
		}
		ids = append(ids, item.ID)
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, projectID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, wrapAlarmRepo("提交默认报警配置失败", err)
	}
	return ids, nil
}

func (r *AlarmRepository) BatchDeleteAlarmItems(ctx context.Context, projectID string, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, wrapAlarmRepo("开启批量删除事务失败", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `DELETE FROM data_alarm_items WHERE project_id=$1 AND id=ANY($2::uuid[])`, projectID, ids)
	if err != nil {
		return 0, translateAlarmWrite("批量删除报警项失败", err)
	}
	if int(tag.RowsAffected()) != len(ids) {
		return 0, alarmNotFound("部分报警项不存在，请刷新后重试")
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, projectID); err != nil {
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, wrapAlarmRepo("提交批量删除事务失败", err)
	}
	return int(tag.RowsAffected()), nil
}

func saveAlarmItemTx(ctx context.Context, tx pgx.Tx, p SaveAlarmItemParams) error {
	channels, contract, err := marshalAlarmJSON(p.NotificationChannelIDs, p.Contract)
	if err != nil {
		return err
	}
	if p.IsCreate {
		_, err = tx.Exec(ctx, `INSERT INTO data_alarm_items(id,project_id,datapoint_id,group_id,display_name,name_key,description,mode,alarm_type,preset_slot,evaluation_mode,derived_expression,trigger_fingerprint,notification_mode,notify_on_raise,notify_on_clear,repeat_interval_seconds,notification_channel_ids,message_template,is_enabled,contract,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18::jsonb,$19,$20,$21::jsonb,$22,$22)`, p.ID, p.ProjectID, p.DatapointID, p.GroupID, p.DisplayName, p.NameKey, p.Description, p.Mode, p.AlarmType, p.PresetSlot, p.EvaluationMode, p.DerivedExpression, p.TriggerFingerprint, p.NotificationMode, p.NotifyOnRaise, p.NotifyOnClear, p.RepeatIntervalSeconds, string(channels), p.MessageTemplate, p.IsEnabled, string(contract), p.UserID)
	} else {
		var currentRevision int64
		if err = tx.QueryRow(ctx, `SELECT revision FROM data_alarm_items WHERE project_id=$1 AND id=$2 FOR UPDATE`, p.ProjectID, p.ID).Scan(&currentRevision); err != nil {
			return firstAlarmError(err, alarmNotFound("报警项不存在"))
		}
		if p.Revision > 0 && p.Revision != currentRevision {
			return translateAlarmWrite("报警项已被其他操作修改，请刷新后重试", fmt.Errorf("revision conflict"))
		}
		_, err = tx.Exec(ctx, `UPDATE data_alarm_items SET group_id=$3,display_name=$4,name_key=$5,description=$6,alarm_type=$7,preset_slot=$8,evaluation_mode=$9,derived_expression=$10,trigger_fingerprint=$11,notification_mode=$12,notify_on_raise=$13,notify_on_clear=$14,repeat_interval_seconds=$15,notification_channel_ids=$16::jsonb,message_template=$17,is_enabled=$18,contract=$19::jsonb,revision=revision+1,updated_by=$20,updated_at=now() WHERE project_id=$1 AND id=$2`, p.ProjectID, p.ID, p.GroupID, p.DisplayName, p.NameKey, p.Description, p.AlarmType, p.PresetSlot, p.EvaluationMode, p.DerivedExpression, p.TriggerFingerprint, p.NotificationMode, p.NotifyOnRaise, p.NotifyOnClear, p.RepeatIntervalSeconds, string(channels), p.MessageTemplate, p.IsEnabled, string(contract), p.UserID)
	}
	if err != nil {
		return translateAlarmWrite("保存报警项失败", err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM data_alarm_item_inputs WHERE alarm_item_id=$1`, p.ID); err != nil {
		return wrapAlarmRepo("清理组合报警输入失败", err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM data_alarm_item_conditions WHERE alarm_item_id=$1`, p.ID); err != nil {
		return wrapAlarmRepo("清理报警条件失败", err)
	}
	for _, input := range p.Inputs {
		if _, err = tx.Exec(ctx, `INSERT INTO data_alarm_item_inputs(id,alarm_item_id,datapoint_id,input_key,sort_order) VALUES($1,$2,$3,$4,$5)`, input.ID, p.ID, input.DatapointID, input.InputKey, input.SortOrder); err != nil {
			return translateAlarmWrite("保存组合报警输入失败", err)
		}
	}
	for _, condition := range p.Conditions {
		raw, marshalErr := json.Marshal(condition.Params)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = tx.Exec(ctx, `INSERT INTO data_alarm_item_conditions(id,alarm_item_id,kind,operator,label,severity,params,trigger_delay_ms,clear_delay_ms,deadband,sort_order) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11)`, condition.ID, p.ID, condition.Kind, condition.Operator, condition.Label, condition.Severity, string(raw), condition.TriggerDelayMS, condition.ClearDelayMS, condition.Deadband, condition.SortOrder); err != nil {
			return translateAlarmWrite("保存报警条件失败", err)
		}
	}
	return nil
}

func (r *AlarmRepository) SetAlarmItemEnabled(ctx context.Context, projectID, id, userID string, enabled bool) (*AlarmItemRecord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE data_alarm_items SET is_enabled=$3,contract=jsonb_set(contract,'{isEnabled}',to_jsonb($3::boolean),true),revision=revision+1,updated_by=$4,updated_at=now() WHERE project_id=$1 AND id=$2`, projectID, id, enabled, userID)
	if err != nil {
		return nil, translateAlarmWrite("更新报警状态失败", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, alarmNotFound("报警项不存在")
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, projectID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetAlarmItem(ctx, projectID, id)
}

func (r *AlarmRepository) DeleteAlarmItem(ctx context.Context, projectID, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `DELETE FROM data_alarm_items WHERE project_id=$1 AND id=$2`, projectID, id)
	if err != nil {
		return translateAlarmWrite("删除报警项失败", err)
	}
	if tag.RowsAffected() == 0 {
		return alarmNotFound("报警项不存在")
	}
	if _, err = incrementAlarmConfigRevision(ctx, tx, projectID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func buildAlarmItemWhere(projectID string, filter AlarmItemListFilter) (string, []any) {
	args := []any{projectID}
	where := []string{"ai.project_id=$1"}
	add := func(value any, clause string) {
		args = append(args, value)
		where = append(where, strings.ReplaceAll(clause, "%d", fmt.Sprint(len(args))))
	}
	if value := strings.TrimSpace(filter.Search); value != "" {
		add("%"+value+"%", `(ai.display_name ILIKE $%d OR ai.description ILIKE $%d OR dp.name ILIKE $%d OR dp.path ILIKE $%d)`)
	}
	if filter.Mode != "" {
		add(filter.Mode, `ai.mode=$%d`)
	}
	if filter.AlarmType != "" {
		add(filter.AlarmType, `ai.alarm_type=$%d`)
	}
	if filter.Enabled != nil {
		add(*filter.Enabled, `ai.is_enabled=$%d`)
	}
	if filter.DatapointID != "" {
		add(filter.DatapointID, `(ai.datapoint_id=$%d OR EXISTS(SELECT 1 FROM data_alarm_item_inputs x WHERE x.alarm_item_id=ai.id AND x.datapoint_id=$%d))`)
	}
	if filter.GroupID != nil {
		add(*filter.GroupID, `ai.group_id IN (WITH RECURSIVE alarm_group_tree AS (SELECT id FROM data_alarm_groups WHERE project_id=$1 AND id=$%d UNION ALL SELECT child.id FROM data_alarm_groups child JOIN alarm_group_tree parent ON child.parent_id=parent.id WHERE child.project_id=$1) SELECT id FROM alarm_group_tree)`)
	}
	if filter.Severity != "" {
		add(filter.Severity, `EXISTS(SELECT 1 FROM data_alarm_item_conditions c WHERE c.alarm_item_id=ai.id AND c.severity=$%d)`)
	}
	return strings.Join(where, " AND "), args
}

func alarmItemBaseSelect() string {
	return `SELECT ai.id,ai.project_id,ai.datapoint_id,dp.path,dp.name,dp.data_type,ai.group_id,g.name,ai.display_name,ai.name_key,ai.description,ai.mode,ai.alarm_type,ai.preset_slot,ai.evaluation_mode,ai.derived_expression,ai.trigger_fingerprint,ai.notification_mode,ai.notify_on_raise,ai.notify_on_clear,ai.repeat_interval_seconds,ai.notification_channel_ids,ai.message_template,ai.is_enabled,ai.revision,ai.contract,ai.created_at,ai.updated_at FROM data_alarm_items ai LEFT JOIN data_points dp ON dp.id=ai.datapoint_id LEFT JOIN data_alarm_groups g ON g.id=ai.group_id`
}

type alarmItemScanner interface{ Scan(...any) error }

func scanAlarmItem(scanner alarmItemScanner) (AlarmItemRecord, error) {
	var item AlarmItemRecord
	var channels, contract []byte
	err := scanner.Scan(&item.ID, &item.ProjectID, &item.DatapointID, &item.Path, &item.DatapointName, &item.DataType, &item.GroupID, &item.GroupName, &item.DisplayName, &item.NameKey, &item.Description, &item.Mode, &item.AlarmType, &item.PresetSlot, &item.EvaluationMode, &item.DerivedExpression, &item.TriggerFingerprint, &item.NotificationMode, &item.NotifyOnRaise, &item.NotifyOnClear, &item.RepeatIntervalSeconds, &channels, &item.MessageTemplate, &item.IsEnabled, &item.Revision, &contract, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, firstAlarmError(err, wrapAlarmRepo("读取报警项失败", err))
	}
	if err = json.Unmarshal(channels, &item.NotificationChannelIDs); err != nil {
		return item, err
	}
	if err = json.Unmarshal(contract, &item.Contract); err != nil {
		return item, err
	}
	return item, nil
}
func scanAlarmItemRows(rows pgx.Rows) ([]AlarmItemRecord, error) {
	items := []AlarmItemRecord{}
	for rows.Next() {
		item, err := scanAlarmItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *AlarmRepository) loadAlarmItemDetails(ctx context.Context, items []AlarmItemRecord) error {
	for index := range items {
		inputs, err := r.listAlarmItemInputs(ctx, items[index].ID)
		if err != nil {
			return err
		}
		conditions, err := r.listAlarmItemConditions(ctx, items[index].ID)
		if err != nil {
			return err
		}
		items[index].Inputs = inputs
		items[index].Conditions = conditions
	}
	return nil
}
func (r *AlarmRepository) listAlarmItemInputs(ctx context.Context, id string) ([]AlarmItemInputRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT i.id,i.datapoint_id,p.path,p.name,p.data_type,i.input_key,i.sort_order FROM data_alarm_item_inputs i JOIN data_points p ON p.id=i.datapoint_id WHERE i.alarm_item_id=$1 ORDER BY i.sort_order,i.created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AlarmItemInputRecord{}
	for rows.Next() {
		var item AlarmItemInputRecord
		if err = rows.Scan(&item.ID, &item.DatapointID, &item.Path, &item.Name, &item.DataType, &item.InputKey, &item.SortOrder); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *AlarmRepository) listAlarmItemConditions(ctx context.Context, id string) ([]AlarmItemConditionRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,kind,operator,label,severity,params,trigger_delay_ms,clear_delay_ms,deadband,sort_order FROM data_alarm_item_conditions WHERE alarm_item_id=$1 ORDER BY sort_order,created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AlarmItemConditionRecord{}
	for rows.Next() {
		var item AlarmItemConditionRecord
		var raw []byte
		if err = rows.Scan(&item.ID, &item.Kind, &item.Operator, &item.Label, &item.Severity, &raw, &item.TriggerDelayMS, &item.ClearDelayMS, &item.Deadband, &item.SortOrder); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &item.Params); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
