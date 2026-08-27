package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/repository"
)

const (
	alarmExcelItemSheet      = "报警项"
	alarmExcelConditionSheet = "报警条件"
	alarmExcelGuideSheet     = "填写说明"
)

type AlarmItemWorkbook struct {
	Content     []byte
	ContentType string
	FileName    string
}

type AlarmItemExcelIssue struct {
	Sheet   string `json:"sheet"`
	Row     int    `json:"row"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

type AlarmItemExcelPreview struct {
	Digest         string                `json:"digest"`
	CreateCount    int                   `json:"createCount"`
	UpdateCount    int                   `json:"updateCount"`
	UnchangedCount int                   `json:"unchangedCount"`
	ErrorCount     int                   `json:"errorCount"`
	WarningCount   int                   `json:"warningCount"`
	Issues         []AlarmItemExcelIssue `json:"issues"`
	WarningKeys    []string              `json:"warningKeys"`
}

type alarmExcelParsed struct {
	preview AlarmItemExcelPreview
	params  []repository.SaveAlarmItemParams
}

var alarmExcelItemHeaders = []string{"报警ID", "revision", "导入键", "数据点ID", "数据点路径", "显示名称", "报警类型", "计算方式", "启用", "目录ID", "目录路径", "通知模式", "触发通知", "恢复通知", "重复提醒秒", "通知渠道ID或名称", "消息模板"}
var alarmExcelConditionHeaders = []string{"报警ID", "导入键", "等级标签", "条件类型", "操作符", "参数JSON", "严重度", "死区", "触发延时(ms)", "清除延时(ms)"}

func (s *AlarmItemService) BuildImportTemplate(ctx context.Context, claims *auth.Claims, projectID string) (AlarmItemWorkbook, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return AlarmItemWorkbook{}, err
	}
	file := excelize.NewFile()
	defer file.Close()
	defaultSheet := file.GetSheetName(0)
	if err := file.SetSheetName(defaultSheet, alarmExcelItemSheet); err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成报警导入模板失败")
	}
	if _, err := file.NewSheet(alarmExcelConditionSheet); err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成报警导入模板失败")
	}
	if _, err := file.NewSheet(alarmExcelGuideSheet); err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成报警导入模板失败")
	}
	if err := writeAlarmExcelHeader(file, alarmExcelItemSheet, alarmExcelItemHeaders); err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成报警导入模板失败")
	}
	if err := writeAlarmExcelHeader(file, alarmExcelConditionSheet, alarmExcelConditionHeaders); err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成报警导入模板失败")
	}
	if err := writeAlarmExcelHeader(file, alarmExcelGuideSheet, []string{"条件类型", "适用数据", "操作符", "参数JSON示例", "说明"}); err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成报警导入模板失败")
	}
	itemExample := []any{"", "", "example-1", "", "产线.温度", "", "threshold", "highest_matching", "是", "", "根目录", "inherit", "", "", "", "", ""}
	conditionExample := []any{"", "example-1", "高", "threshold", "gt", `{"threshold":80}`, "warning", 0, 0, 0}
	for index, value := range itemExample {
		if err := setAlarmExcelCell(file, alarmExcelItemSheet, index+1, 2, value); err != nil {
			return AlarmItemWorkbook{}, badAlarm("生成报警导入模板失败")
		}
	}
	for index, value := range conditionExample {
		if err := setAlarmExcelCell(file, alarmExcelConditionSheet, index+1, 2, value); err != nil {
			return AlarmItemWorkbook{}, badAlarm("生成报警导入模板失败")
		}
	}
	guideRows := [][]any{
		{"threshold", "数值", "gt/gte/lt/lte", `{"threshold":80}`, "越限报警可写多行等级，并使用 highest_matching"},
		{"rate_of_change", "数值", "gt", `{"direction":"rise","limit":2,"windowMs":60000}`, "direction 为 rise/fall/absolute，速率单位为数值每秒"},
		{"quality", "全部类型", "in", `{"qualities":["bad","unknown"]}`, "只允许 bad、unknown，可选择一种或两种"},
		{"stale", "全部类型", "age_gte", `{"maxAgeMs":60000}`, "按源时间判断数据陈旧；缺失源时间持续达到期限也触发"},
		{"offline", "全部类型", "is_offline", `{}`, "数据点离线时触发"},
	}
	for rowIndex, values := range guideRows {
		if err := setAlarmExcelRow(file, alarmExcelGuideSheet, rowIndex+2, values); err != nil {
			return AlarmItemWorkbook{}, badAlarm("生成报警导入模板失败")
		}
	}
	if err := configureAlarmWorkbook(file, true); err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成报警导入模板失败")
	}
	buffer, err := file.WriteToBuffer()
	if err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成报警导入模板失败")
	}
	return alarmWorkbook(buffer.Bytes(), "报警项导入模板.xlsx"), nil
}

func (s *AlarmItemService) ExportWorkbook(ctx context.Context, claims *auth.Claims, projectID string, selection AlarmItemSelection) (AlarmItemWorkbook, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return AlarmItemWorkbook{}, err
	}
	items, err := s.resolveSelection(ctx, projectID, selection)
	if err != nil {
		return AlarmItemWorkbook{}, err
	}
	file := excelize.NewFile()
	defer file.Close()
	if err := file.SetSheetName(file.GetSheetName(0), alarmExcelItemSheet); err != nil {
		return AlarmItemWorkbook{}, badAlarm("导出报警项失败")
	}
	if _, err := file.NewSheet(alarmExcelConditionSheet); err != nil {
		return AlarmItemWorkbook{}, badAlarm("导出报警项失败")
	}
	if err := writeAlarmExcelHeader(file, alarmExcelItemSheet, alarmExcelItemHeaders); err != nil {
		return AlarmItemWorkbook{}, badAlarm("导出报警项失败")
	}
	if err := writeAlarmExcelHeader(file, alarmExcelConditionSheet, alarmExcelConditionHeaders); err != nil {
		return AlarmItemWorkbook{}, badAlarm("导出报警项失败")
	}
	itemRow, conditionRow := 2, 2
	for _, item := range items {
		if item.Mode != "point" || item.DatapointID == nil || item.Path == nil {
			continue
		}
		channels := strings.Join(item.NotificationChannelIDs, ",")
		values := []any{item.ID, item.Revision, "", *item.DatapointID, *item.Path, item.DisplayName, item.AlarmType, item.EvaluationMode, alarmExcelBool(item.IsEnabled), valueString(item.GroupID), valueString(item.GroupName), item.NotificationMode, alarmExcelOptionalBool(item.NotifyOnRaise), alarmExcelOptionalBool(item.NotifyOnClear), alarmExcelOptionalInt(item.RepeatIntervalSeconds), channels, item.MessageTemplate}
		if err := setAlarmExcelRow(file, alarmExcelItemSheet, itemRow, values); err != nil {
			return AlarmItemWorkbook{}, badAlarm("导出报警项失败")
		}
		itemRow++
		for _, condition := range item.Conditions {
			params, marshalErr := json.Marshal(condition.Params)
			if marshalErr != nil {
				return AlarmItemWorkbook{}, badAlarm("报警条件参数无法导出")
			}
			if err := setAlarmExcelRow(file, alarmExcelConditionSheet, conditionRow, []any{item.ID, "", condition.Label, condition.Kind, condition.Operator, string(params), condition.Severity, condition.Deadband, condition.TriggerDelayMS, condition.ClearDelayMS}); err != nil {
				return AlarmItemWorkbook{}, badAlarm("导出报警项失败")
			}
			conditionRow++
		}
	}
	if err := configureAlarmWorkbook(file, false); err != nil {
		return AlarmItemWorkbook{}, badAlarm("导出报警项失败")
	}
	buffer, err := file.WriteToBuffer()
	if err != nil {
		return AlarmItemWorkbook{}, badAlarm("导出报警项失败")
	}
	return alarmWorkbook(buffer.Bytes(), "报警项.xlsx"), nil
}

func (s *AlarmItemService) PreviewImport(ctx context.Context, claims *auth.Claims, projectID, fileName string, content []byte) (*AlarmItemExcelPreview, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return nil, err
	}
	parsed, err := s.parseAlarmWorkbook(ctx, projectID, fileName, content, nil)
	if err != nil {
		return nil, err
	}
	return &parsed.preview, nil
}

func (s *AlarmItemService) BuildImportErrorWorkbook(ctx context.Context, claims *auth.Claims, projectID, fileName string, content []byte) (AlarmItemWorkbook, error) {
	if err := s.readAccess(claims, projectID); err != nil {
		return AlarmItemWorkbook{}, err
	}
	parsed, err := s.parseAlarmWorkbook(ctx, projectID, fileName, content, nil)
	if err != nil {
		return AlarmItemWorkbook{}, err
	}
	file, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return AlarmItemWorkbook{}, badAlarm("无法解析报警导入文件")
	}
	defer file.Close()
	const issueSheet = "导入问题"
	if index, _ := file.GetSheetIndex(issueSheet); index >= 0 {
		if err := file.DeleteSheet(issueSheet); err != nil {
			return AlarmItemWorkbook{}, badAlarm("生成导入问题工作簿失败")
		}
	}
	if _, err := file.NewSheet(issueSheet); err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成导入问题工作簿失败")
	}
	if err := writeAlarmExcelHeader(file, issueSheet, []string{"工作表", "行", "类型", "原因"}); err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成导入问题工作簿失败")
	}
	for index, issue := range parsed.preview.Issues {
		if err := setAlarmExcelRow(file, issueSheet, index+2, []any{issue.Sheet, issue.Row, issue.Type, issue.Message}); err != nil {
			return AlarmItemWorkbook{}, badAlarm("生成导入问题工作簿失败")
		}
	}
	if err := file.SetColWidth(issueSheet, "A", "A", 16); err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成导入问题工作簿失败")
	}
	if err := file.SetColWidth(issueSheet, "B", "C", 10); err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成导入问题工作簿失败")
	}
	if err := file.SetColWidth(issueSheet, "D", "D", 72); err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成导入问题工作簿失败")
	}
	buffer, err := file.WriteToBuffer()
	if err != nil {
		return AlarmItemWorkbook{}, badAlarm("生成导入问题工作簿失败")
	}
	return alarmWorkbook(buffer.Bytes(), "报警项-导入问题.xlsx"), nil
}

func (s *AlarmItemService) ApplyImport(ctx context.Context, claims *auth.Claims, projectID, fileName string, content []byte, digest string, acknowledged []string) (*BatchAlarmItemResult, error) {
	if err := s.writeAccess(claims, projectID); err != nil {
		return nil, err
	}
	actual := alarmExcelDigest(content)
	if strings.TrimSpace(digest) == "" || !strings.EqualFold(strings.TrimSpace(digest), actual) {
		return nil, badAlarm("导入文件与预览文件不一致，请重新预览")
	}
	parsed, err := s.parseAlarmWorkbook(ctx, projectID, fileName, content, acknowledged)
	if err != nil {
		return nil, err
	}
	if parsed.preview.ErrorCount > 0 {
		return nil, badAlarm("导入文件仍有错误，不能提交")
	}
	ack := map[string]bool{}
	for _, key := range acknowledged {
		ack[key] = true
	}
	for _, key := range parsed.preview.WarningKeys {
		if !ack[key] {
			return nil, badAlarm("导入文件存在未确认的报警重叠警告")
		}
	}
	for index := range parsed.params {
		parsed.params[index].UserID = claims.UserID
	}
	if err = s.repository.BatchUpsertAlarmItems(ctx, projectID, claims.UserID, parsed.params); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(parsed.params))
	for _, params := range parsed.params {
		ids = append(ids, params.ID)
	}
	return &BatchAlarmItemResult{AffectedCount: len(ids), ItemIDs: ids}, nil
}

func (s *AlarmItemService) parseAlarmWorkbook(ctx context.Context, projectID, fileName string, content []byte, acknowledged []string) (*alarmExcelParsed, error) {
	if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(fileName)), ".xlsx") {
		return nil, badAlarm("报警导入仅支持 .xlsx 文件")
	}
	file, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, badAlarm("无法解析报警导入文件")
	}
	defer file.Close()
	for _, sheet := range []string{alarmExcelItemSheet, alarmExcelConditionSheet} {
		index, indexErr := file.GetSheetIndex(sheet)
		if indexErr != nil || index == -1 {
			return nil, badAlarm("导入文件缺少工作表：" + sheet)
		}
		if err = rejectAlarmExcelFormulas(file, sheet); err != nil {
			return nil, err
		}
	}
	itemRows, err := file.GetRows(alarmExcelItemSheet)
	if err != nil {
		return nil, badAlarm("读取报警项工作表失败")
	}
	conditionRows, err := file.GetRows(alarmExcelConditionSheet)
	if err != nil {
		return nil, badAlarm("读取报警条件工作表失败")
	}
	itemHeader, headerErr := alarmExcelHeader(itemRows, alarmExcelItemHeaders)
	if headerErr != nil {
		return nil, headerErr
	}
	conditionHeader, headerErr := alarmExcelHeader(conditionRows, alarmExcelConditionHeaders)
	if headerErr != nil {
		return nil, headerErr
	}
	conditionsByKey := map[string][]AlarmCondition{}
	conditionIssues := []AlarmItemExcelIssue{}
	for rowIndex, row := range conditionRows[1:] {
		if alarmExcelRowEmpty(row) {
			continue
		}
		key := alarmExcelLinkKey(cell(row, conditionHeader["报警ID"]), cell(row, conditionHeader["导入键"]))
		if key == "" {
			conditionIssues = append(conditionIssues, AlarmItemExcelIssue{Sheet: alarmExcelConditionSheet, Row: rowIndex + 2, Type: "error", Message: "报警ID和导入键不能同时为空"})
			continue
		}
		condition, parseErr := parseAlarmExcelCondition(row, conditionHeader)
		if parseErr != nil {
			conditionIssues = append(conditionIssues, AlarmItemExcelIssue{Sheet: alarmExcelConditionSheet, Row: rowIndex + 2, Type: "error", Message: parseErr.Error()})
			continue
		}
		conditionsByKey[key] = append(conditionsByKey[key], condition)
	}
	groups, _ := s.repository.ListAllGroups(ctx, projectID)
	groupLookup := map[string]string{}
	for _, group := range groups {
		groupLookup[group.ID], groupLookup[strings.ToLower(group.FullPath)] = group.ID, group.ID
		nameKey := strings.ToLower(group.Name)
		if existingID, exists := groupLookup[nameKey]; exists && existingID != group.ID {
			groupLookup[nameKey] = ""
		} else {
			groupLookup[nameKey] = group.ID
		}
	}
	channels, _ := s.repository.ListChannels(ctx, projectID)
	channelLookup := map[string]string{"runtime_inapp": "runtime_inapp"}
	for _, channel := range channels {
		channelLookup[channel.ID], channelLookup[strings.ToLower(channel.Name)] = channel.ID, channel.ID
	}
	preview := AlarmItemExcelPreview{Digest: alarmExcelDigest(content), Issues: conditionIssues, WarningKeys: []string{}}
	params := []repository.SaveAlarmItemParams{}
	seenKeys := map[string]bool{}
	for rowIndex, row := range itemRows[1:] {
		if alarmExcelRowEmpty(row) {
			continue
		}
		rowNumber := rowIndex + 2
		alarmID := strings.TrimSpace(cell(row, itemHeader["报警ID"]))
		importKey := strings.TrimSpace(cell(row, itemHeader["导入键"]))
		linkKey := alarmExcelLinkKey(alarmID, importKey)
		if linkKey == "" || seenKeys[linkKey] {
			preview.Issues = append(preview.Issues, AlarmItemExcelIssue{Sheet: alarmExcelItemSheet, Row: rowNumber, Type: "error", Message: "报警ID/导入键不能为空且不能重复"})
			continue
		}
		seenKeys[linkKey] = true
		conditions := conditionsByKey[linkKey]
		if len(conditions) == 0 {
			preview.Issues = append(preview.Issues, AlarmItemExcelIssue{Sheet: alarmExcelItemSheet, Row: rowNumber, Type: "error", Message: "未找到对应报警条件"})
			continue
		}
		draft, existing, parseErr := s.parseAlarmExcelItem(ctx, projectID, row, itemHeader, alarmID, conditions, groupLookup, channelLookup)
		if parseErr != nil {
			preview.Issues = append(preview.Issues, AlarmItemExcelIssue{Sheet: alarmExcelItemSheet, Row: rowNumber, Type: "error", Message: parseErr.Error()})
			continue
		}
		draft.AcknowledgedWarningKeys = acknowledged
		id := alarmID
		if id == "" {
			id = uuid.NewString()
		}
		normalized, normalizedParams, validation, normalizeErr := s.normalize(ctx, projectID, idIf(existing, id), draft)
		if normalizeErr != nil {
			preview.Issues = append(preview.Issues, AlarmItemExcelIssue{Sheet: alarmExcelItemSheet, Row: rowNumber, Type: "error", Message: normalizeErr.Error()})
			continue
		}
		if !existing {
			normalizedParams.ID, normalizedParams.IsCreate = id, true
			normalizedParams.Contract = buildAlarmItemContract(projectID, id, normalized, normalizedParams, normalizedParams.TriggerFingerprint)
		}
		for _, issue := range validation.Errors {
			preview.Issues = append(preview.Issues, AlarmItemExcelIssue{Sheet: alarmExcelItemSheet, Row: rowNumber, Type: "error", Message: issue.Message})
		}
		for _, issue := range validation.Warnings {
			preview.Issues = append(preview.Issues, AlarmItemExcelIssue{Sheet: alarmExcelItemSheet, Row: rowNumber, Type: "warning", Message: issue.Message})
			preview.WarningKeys = append(preview.WarningKeys, issue.AcknowledgementKey)
		}
		if len(validation.Errors) > 0 {
			continue
		}
		if existing {
			current, _ := s.repository.GetAlarmItem(ctx, projectID, alarmID)
			if current != nil && alarmExcelItemUnchanged(*current, normalizedParams) {
				preview.UnchangedCount++
				continue
			}
			preview.UpdateCount++
		} else {
			preview.CreateCount++
		}
		params = append(params, normalizedParams)
	}
	for key := range conditionsByKey {
		if !seenKeys[key] {
			preview.Issues = append(preview.Issues, AlarmItemExcelIssue{Sheet: alarmExcelConditionSheet, Type: "error", Message: "存在未关联到报警项的条件：" + key})
		}
	}
	for _, issue := range preview.Issues {
		if issue.Type == "warning" {
			preview.WarningCount++
		} else {
			preview.ErrorCount++
		}
	}
	preview.WarningKeys = alarmUniqueStrings(preview.WarningKeys)
	sort.Strings(preview.WarningKeys)
	return &alarmExcelParsed{preview: preview, params: params}, nil
}

func (s *AlarmItemService) parseAlarmExcelItem(ctx context.Context, projectID string, row []string, header map[string]int, alarmID string, conditions []AlarmCondition, groups, channels map[string]string) (SaveAlarmItemInput, bool, error) {
	var existing *repository.AlarmItemRecord
	var err error
	if alarmID != "" {
		existing, err = s.repository.GetAlarmItem(ctx, projectID, alarmID)
		if err != nil {
			return SaveAlarmItemInput{}, false, badAlarm("报警ID不存在或不属于当前工程")
		}
		if existing.Mode != "point" {
			return SaveAlarmItemInput{}, false, badAlarm("Excel 不支持更新组合报警")
		}
		revision, parseErr := strconv.ParseInt(strings.TrimSpace(cell(row, header["revision"])), 10, 64)
		if parseErr != nil || revision != existing.Revision {
			return SaveAlarmItemInput{}, false, badAlarm("revision 已过期，请重新导出后修改")
		}
	}
	datapointID := strings.TrimSpace(cell(row, header["数据点ID"]))
	path := strings.TrimSpace(cell(row, header["数据点路径"]))
	if existing != nil {
		datapointID = valueString(existing.DatapointID)
	} else if datapointID == "" {
		point, pointErr := s.datapoints.GetByProjectAndPath(ctx, projectID, path)
		if pointErr != nil {
			return SaveAlarmItemInput{}, false, badAlarm("数据点路径不存在或不属于当前工程")
		}
		datapointID = point.ID
	} else if _, pointErr := s.datapoints.GetByProjectAndID(ctx, projectID, datapointID); pointErr != nil {
		return SaveAlarmItemInput{}, false, badAlarm("数据点ID不存在或不属于当前工程")
	}
	displayName := strings.TrimSpace(cell(row, header["显示名称"]))
	if existing != nil && displayName == "" {
		displayName = existing.DisplayName
	}
	evaluationMode := strings.TrimSpace(cell(row, header["计算方式"]))
	if evaluationMode == "" {
		if strings.TrimSpace(cell(row, header["报警类型"])) == "threshold" {
			evaluationMode = "highest_matching"
		} else {
			evaluationMode = "single"
		}
	}
	enabled, err := parseAlarmExcelBool(cell(row, header["启用"]))
	if err != nil {
		return SaveAlarmItemInput{}, false, err
	}
	groupID, err := resolveAlarmExcelReference(cell(row, header["目录ID"]), cell(row, header["目录路径"]), groups, "目录")
	if err != nil {
		return SaveAlarmItemInput{}, false, err
	}
	notificationMode := strings.TrimSpace(cell(row, header["通知模式"]))
	if notificationMode == "" {
		notificationMode = "inherit"
	}
	channelIDs := []string{}
	for _, token := range strings.Split(cell(row, header["通知渠道ID或名称"]), ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		channelID, ok := channels[token]
		if !ok {
			channelID, ok = channels[strings.ToLower(token)]
		}
		if !ok {
			return SaveAlarmItemInput{}, false, badAlarm("通知渠道不存在：" + token)
		}
		channelIDs = append(channelIDs, channelID)
	}
	notifyRaise, err := parseAlarmExcelOptionalBool(cell(row, header["触发通知"]))
	if err != nil {
		return SaveAlarmItemInput{}, false, err
	}
	notifyClear, err := parseAlarmExcelOptionalBool(cell(row, header["恢复通知"]))
	if err != nil {
		return SaveAlarmItemInput{}, false, err
	}
	repeatSeconds, err := parseAlarmExcelOptionalInt(cell(row, header["重复提醒秒"]))
	if err != nil {
		return SaveAlarmItemInput{}, false, badAlarm("重复提醒秒必须是正整数或留空")
	}
	if notificationMode == "custom" {
		if notifyRaise == nil {
			value := true
			notifyRaise = &value
		}
		if notifyClear == nil {
			value := true
			notifyClear = &value
		}
	}
	draft := SaveAlarmItemInput{ItemID: alarmID, DatapointID: datapointID, DisplayName: displayName, GroupID: groupID, Mode: "point", EvaluationMode: evaluationMode, Conditions: conditions, Notification: AlarmNotificationSettings{Mode: notificationMode, NotifyOnRaise: notifyRaise, NotifyOnClear: notifyClear, RepeatIntervalSeconds: repeatSeconds, ChannelIDs: alarmUniqueStrings(channelIDs), MessageTemplate: cell(row, header["消息模板"])}, IsEnabled: &enabled}
	if existing != nil {
		draft.Description = existing.Description
		draft.Revision = existing.Revision
	}
	return draft, existing != nil, nil
}

func parseAlarmExcelCondition(row []string, header map[string]int) (AlarmCondition, error) {
	params := map[string]any{}
	if raw := strings.TrimSpace(cell(row, header["参数JSON"])); raw != "" {
		if err := json.Unmarshal([]byte(raw), &params); err != nil {
			return AlarmCondition{}, fmt.Errorf("参数JSON格式错误")
		}
	}
	deadband, err := parseAlarmExcelFloat(cell(row, header["死区"]), 0)
	if err != nil {
		return AlarmCondition{}, fmt.Errorf("死区必须是数字")
	}
	trigger, err := parseAlarmExcelInt(cell(row, header["触发延时(ms)"]), 0)
	if err != nil {
		return AlarmCondition{}, fmt.Errorf("触发延时必须是整数")
	}
	clearDelay, err := parseAlarmExcelInt(cell(row, header["清除延时(ms)"]), 0)
	if err != nil {
		return AlarmCondition{}, fmt.Errorf("清除延时必须是整数")
	}
	return AlarmCondition{Kind: strings.TrimSpace(cell(row, header["条件类型"])), Operator: strings.TrimSpace(cell(row, header["操作符"])), Label: strings.TrimSpace(cell(row, header["等级标签"])), Severity: strings.TrimSpace(cell(row, header["严重度"])), Params: params, Deadband: deadband, TriggerDelayMS: trigger, ClearDelayMS: clearDelay}, nil
}

func writeAlarmExcelHeader(file *excelize.File, sheet string, headers []string) error {
	for index, header := range headers {
		if err := setAlarmExcelCell(file, sheet, index+1, 1, header); err != nil {
			return err
		}
	}
	return file.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
}
func setAlarmExcelRow(file *excelize.File, sheet string, row int, values []any) error {
	for index, value := range values {
		if err := setAlarmExcelCell(file, sheet, index+1, row, value); err != nil {
			return err
		}
	}
	return nil
}
func setAlarmExcelCell(file *excelize.File, sheet string, column, row int, value any) error {
	cell, err := excelize.CoordinatesToCellName(column, row)
	if err != nil {
		return err
	}
	return file.SetCellValue(sheet, cell, value)
}
func configureAlarmWorkbook(file *excelize.File, includeGuide bool) error {
	if err := file.SetColVisible(alarmExcelItemSheet, "D", false); err != nil {
		return err
	}
	if err := file.SetColWidth(alarmExcelItemSheet, "A", "Q", 18); err != nil {
		return err
	}
	if err := file.SetColWidth(alarmExcelConditionSheet, "A", "J", 18); err != nil {
		return err
	}
	if includeGuide {
		if err := file.SetColWidth(alarmExcelGuideSheet, "A", "D", 24); err != nil {
			return err
		}
		if err := file.SetColWidth(alarmExcelGuideSheet, "E", "E", 60); err != nil {
			return err
		}
	}
	return nil
}
func alarmWorkbook(content []byte, name string) AlarmItemWorkbook {
	return AlarmItemWorkbook{Content: content, ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", FileName: name}
}
func alarmExcelDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
func alarmExcelHeader(rows [][]string, required []string) (map[string]int, error) {
	if len(rows) == 0 {
		return nil, badAlarm("导入工作表没有表头")
	}
	result := map[string]int{}
	for index, value := range rows[0] {
		result[strings.TrimSpace(value)] = index
	}
	for _, name := range required {
		if _, ok := result[name]; !ok {
			return nil, badAlarm("导入工作表缺少列：" + name)
		}
	}
	return result, nil
}
func rejectAlarmExcelFormulas(file *excelize.File, sheet string) error {
	rows, _ := file.GetRows(sheet)
	for rowIndex, row := range rows {
		for columnIndex := range row {
			cellName, _ := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+1)
			formula, _ := file.GetCellFormula(sheet, cellName)
			if formula != "" {
				return badAlarm("报警导入不接受公式或宏")
			}
		}
	}
	return nil
}
func alarmExcelLinkKey(alarmID, importKey string) string {
	if value := strings.TrimSpace(alarmID); value != "" {
		return "id:" + value
	}
	if value := strings.TrimSpace(importKey); value != "" {
		return "key:" + value
	}
	return ""
}
func resolveAlarmExcelReference(id, name string, lookup map[string]string, label string) (*string, error) {
	for _, raw := range []string{id, name} {
		value := strings.TrimSpace(raw)
		if value == "" || value == "根目录" {
			continue
		}
		if resolved, ok := lookup[value]; ok && resolved != "" {
			return &resolved, nil
		}
		if resolved, ok := lookup[strings.ToLower(value)]; ok && resolved != "" {
			return &resolved, nil
		}
		if resolved, ok := lookup[strings.ToLower(value)]; ok && resolved == "" {
			return nil, badAlarm(label + "名称不唯一，请填写完整路径或 ID：" + value)
		}
		return nil, badAlarm(label + "不存在：" + value)
	}
	return nil, nil
}
func parseAlarmExcelBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "是", "true", "1", "启用":
		return true, nil
	case "否", "false", "0", "停用":
		return false, nil
	default:
		return false, badAlarm("启用列只接受是/否")
	}
}
func alarmExcelBool(value bool) string {
	if value {
		return "是"
	}
	return "否"
}
func alarmExcelOptionalBool(value *bool) string {
	if value == nil {
		return ""
	}
	return alarmExcelBool(*value)
}
func alarmExcelOptionalInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}
func parseAlarmExcelOptionalBool(value string) (*bool, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := parseAlarmExcelBool(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
func parseAlarmExcelOptionalInt(value string) (*int, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return nil, fmt.Errorf("invalid positive integer")
	}
	return &parsed, nil
}
func alarmExcelItemUnchanged(current repository.AlarmItemRecord, next repository.SaveAlarmItemParams) bool {
	return current.DisplayName == next.DisplayName && valueString(current.GroupID) == valueString(next.GroupID) &&
		current.EvaluationMode == next.EvaluationMode && current.TriggerFingerprint == next.TriggerFingerprint &&
		current.NotificationMode == next.NotificationMode && current.MessageTemplate == next.MessageTemplate &&
		boolPointersEqual(current.NotifyOnRaise, next.NotifyOnRaise) && boolPointersEqual(current.NotifyOnClear, next.NotifyOnClear) &&
		intPointersEqual(current.RepeatIntervalSeconds, next.RepeatIntervalSeconds) &&
		strings.Join(current.NotificationChannelIDs, "\x00") == strings.Join(next.NotificationChannelIDs, "\x00") && current.IsEnabled == next.IsEnabled
}
func boolPointersEqual(left, right *bool) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && *left == *right)
}
func intPointersEqual(left, right *int) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && *left == *right)
}
func parseAlarmExcelFloat(value string, fallback float64) (float64, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(strings.TrimSpace(value), 64)
}
func parseAlarmExcelInt(value string, fallback int64) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	return strconv.ParseInt(strings.TrimSpace(value), 10, 64)
}
func cell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}
func alarmExcelRowEmpty(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}
func idIf(existing bool, id string) string {
	if existing {
		return id
	}
	return ""
}
