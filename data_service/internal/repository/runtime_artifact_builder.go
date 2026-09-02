package repository

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// RuntimeProjectArtifactV1 是运行时唯一接受的工程定义投影。它刻意不复用历史开发态
// ProjectArtifactV1：运行时契约需要完整、无歧义且可在节点上直接执行的对象。
type RuntimeProjectArtifactV1 struct {
	SchemaVersion          string `json:"schemaVersion"`
	ProjectArtifactVersion string `json:"projectArtifactVersion"`
	ProjectID              string `json:"projectId"`
	GeneratedAt            string `json:"generatedAt"`
	DataPoints             []any  `json:"dataPoints"`
	ComputeUnits           []any  `json:"computeUnits"`
	AlarmItems             []any  `json:"alarmItems"`
}

// CollectorRuntimeArtifactV1 是节点采集器可部署的无现场密钥逻辑定义。
type CollectorRuntimeArtifactV1 struct {
	SchemaVersion    string `json:"schemaVersion"`
	ArtifactID       string `json:"artifactId"`
	ArtifactRevision int64  `json:"artifactRevision"`
	ProjectID        string `json:"projectId"`
	CollectorVersion string `json:"collectorVersion"`
	Connections      []any  `json:"connections"`
	PointMappings    []any  `json:"pointMappings"`
	WAL              any    `json:"wal"`
}

// BuildRuntimeProjectArtifactV1 构建并再次用根 schema 校验运行态工程产物。
// generatedAt 必须由调用者的时钟注入，避免构建函数暗含不可重复的当前时间。
func BuildRuntimeProjectArtifactV1(schemaRoot, projectID string, snapshot *ProjectSnapshot, generatedAt time.Time) (*RuntimeProjectArtifactV1, error) {
	if snapshot == nil || generatedAt.IsZero() {
		return nil, fmt.Errorf("运行时工程产物缺少快照或生成时间")
	}
	points := make(map[string]DataPointRecord, len(snapshot.DataPoints))
	collectorConnections := make(map[string]string, len(snapshot.CollectorPoints))
	for _, point := range snapshot.CollectorPoints {
		collectorConnections[point.ID] = point.ConnectionID
	}
	dataPoints := make([]any, 0, len(snapshot.DataPoints))
	for _, point := range snapshot.DataPoints {
		if strings.TrimSpace(point.ID) == "" || points[point.ID].ID != "" {
			return nil, fmt.Errorf("数据点 ID 为空或重复")
		}
		sourceType, err := runtimeDataPointSourceType(point.SourceType)
		if err != nil {
			return nil, fmt.Errorf("数据点 %s: %w", point.ID, err)
		}
		defaultValue, err := parseTypedDefault(point.DefaultValue, point.DataType)
		if err != nil {
			return nil, fmt.Errorf("数据点 %s 默认值无效: %w", point.ID, err)
		}
		permissions := normalizeDataPointRuntimePermissions(point.RuntimePermissions)
		sourceConfig, err := runtimeSourceConfig(point, collectorConnections)
		if err != nil {
			return nil, err
		}
		dataPoints = append(dataPoints, map[string]any{
			"id": point.ID, "path": point.Path, "name": point.Name, "dataType": point.DataType,
			"sourceType": sourceType, "sourceId": nullableString(point.SourceID), "sourceConfig": sourceConfig,
			"runtimePermissions": map[string]any{"write": map[string]any{"allowRoles": stringSliceAny(permissions.Write.AllowRoles), "denyRoles": stringSliceAny(permissions.Write.DenyRoles), "inherit": permissions.Write.Inherit}},
			"refreshMode":        point.RefreshMode, "refreshIntervalMs": nullableInt(point.RefreshIntervalMS), "status": point.Status,
			"displayOrder": point.DisplayOrder, "unit": nullableString(point.Unit), "precisionNum": nullableInt(point.PrecisionNum),
			"defaultValue": defaultValue, "tags": cloneSnapshotAnyArray(point.Tags), "attributeDefaults": cloneSnapshotStringMap(point.AttributeDefaults),
		})
		points[point.ID] = point
	}
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].(map[string]any)["id"].(string) < dataPoints[j].(map[string]any)["id"].(string)
	})

	compute := make([]any, 0, len(snapshot.ComputeUnits))
	for _, unit := range snapshot.ComputeUnits {
		item, err := buildRuntimeComputeUnit(unit, points)
		if err != nil {
			return nil, err
		}
		compute = append(compute, item)
	}
	sort.Slice(compute, func(i, j int) bool {
		return compute[i].(map[string]any)["id"].(string) < compute[j].(map[string]any)["id"].(string)
	})

	alarms := make([]any, 0, len(snapshot.AlarmItems))
	for _, alarm := range snapshot.AlarmItems {
		item, err := buildRuntimeAlarm(alarm, points)
		if err != nil {
			return nil, err
		}
		alarms = append(alarms, item)
	}
	sort.Slice(alarms, func(i, j int) bool {
		return alarms[i].(map[string]any)["id"].(string) < alarms[j].(map[string]any)["id"].(string)
	})

	artifact := &RuntimeProjectArtifactV1{"runtime-project-artifact.v1", "1.0", projectID, generatedAt.UTC().Format(time.RFC3339Nano), dataPoints, compute, alarms}
	if err := validateRuntimeArtifactSchema(schemaRoot, "runtime-project-artifact.schema.json", artifact); err != nil {
		return nil, err
	}
	return artifact, nil
}

// runtimeDataPointSourceType 将开发态 sourceType 投影为冻结的 Runtime V1 白名单。
// db.query、mqtt.subscription 与 realtime.key 由基础引擎通过受控数据域访问，
// 运行工件只携带来源引用，绝不携带连接或密钥配置。
func runtimeDataPointSourceType(sourceType string) (string, error) {
	switch sourceType {
	case "manual":
		return "manual.input", nil
	case "collector.point", "calc.output", "db.query", "mqtt.subscription", "realtime.key":
		return sourceType, nil
	default:
		return "", fmt.Errorf("sourceType %s 不允许进入运行产物", sourceType)
	}
}

func runtimeSourceConfig(point DataPointRecord, collectorConnections map[string]string) (map[string]any, error) {
	forbidden := func(config map[string]any) bool {
		for key := range config {
			normalized := strings.ToLower(key)
			for _, token := range []string{"endpoint", "host", "credential", "password", "token", "secret", "username"} {
				if strings.Contains(normalized, token) {
					return true
				}
			}
		}
		return false
	}
	if forbidden(point.SourceConfig) {
		return nil, fmt.Errorf("数据点 %s sourceConfig 包含现场或密钥字段", point.ID)
	}
	switch point.SourceType {
	case "manual":
		if len(point.SourceConfig) != 0 || point.SourceID != nil {
			return nil, fmt.Errorf("手工数据点 %s sourceConfig/sourceId 必须为空", point.ID)
		}
		return map[string]any{}, nil
	case "collector.point":
		if point.SourceID == nil {
			return nil, fmt.Errorf("采集数据点 %s 缺少 sourceId", point.ID)
		}
		connectionID, ok := collectorConnections[*point.SourceID]
		if !ok {
			return nil, fmt.Errorf("采集数据点 %s sourceId 未映射采集点", point.ID)
		}
		if len(point.SourceConfig) != 1 || point.SourceConfig["connectionId"] != connectionID {
			return nil, fmt.Errorf("采集数据点 %s sourceConfig 必须且只能包含 connectionId", point.ID)
		}
		return map[string]any{"connectionId": connectionID}, nil
	case "calc.output":
		if point.SourceID == nil {
			return nil, fmt.Errorf("计算输出数据点 %s 缺少 sourceId", point.ID)
		}
		unitID, ok := point.SourceConfig["computeUnitId"].(string)
		key, ok2 := point.SourceConfig["outputKey"].(string)
		if !ok || !ok2 || strings.TrimSpace(key) == "" || unitID != *point.SourceID || len(point.SourceConfig) != 2 {
			return nil, fmt.Errorf("计算输出数据点 %s sourceConfig 无效", point.ID)
		}
		// data_service 内部需要 outputKey 来维护一对一输出；运行时冻结契约仅暴露
		// 稳定的 computeId，避免把内部索引模型带入节点制品。
		return map[string]any{"computeId": unitID}, nil
	case "db.query", "mqtt.subscription":
		if point.SourceID == nil {
			return nil, fmt.Errorf("基础数据点 %s 缺少 sourceId", point.ID)
		}
		return map[string]any{}, nil
	case "realtime.key":
		if point.SourceID == nil {
			return nil, fmt.Errorf("实时 Key 数据点 %s 缺少 sourceId", point.ID)
		}
		keyID, ok := point.SourceConfig["keyId"].(string)
		if !ok || strings.TrimSpace(keyID) == "" {
			return nil, fmt.Errorf("实时 Key 数据点 %s 缺少 keyId", point.ID)
		}
		return map[string]any{"keyId": keyID}, nil
	default:
		return nil, fmt.Errorf("数据点 %s sourceType %s 不允许进入运行产物", point.ID, point.SourceType)
	}
}

func buildRuntimeComputeUnit(unit ComputeUnitRecord, points map[string]DataPointRecord) (map[string]any, error) {
	if unit.Revision < 1 {
		return nil, fmt.Errorf("计算单元 %s revision 无效", unit.ID)
	}
	deps := make([]any, 0, len(unit.Dependencies))
	seenDeps := map[string]bool{}
	for _, raw := range unit.Dependencies {
		dep, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("计算单元 %s 依赖格式无效", unit.ID)
		}
		pkg, a := stringField(dep, "packageName")
		imp, b := stringField(dep, "importName")
		ver, c := stringField(dep, "version")
		lang, d := stringField(dep, "language")
		if !a || !b || !c || !d || (lang != "js" && lang != "python") || lang != unit.Language {
			return nil, fmt.Errorf("计算单元 %s 依赖不完整", unit.ID)
		}
		key := pkg + "\x00" + imp + "\x00" + ver + "\x00" + lang
		if seenDeps[key] {
			return nil, fmt.Errorf("计算单元 %s 依赖重复", unit.ID)
		}
		seenDeps[key] = true
		deps = append(deps, map[string]any{"packageName": pkg, "importName": imp, "version": ver, "language": lang})
	}
	sort.Slice(deps, func(i, j int) bool { return canonicalJSONKey(deps[i]) < canonicalJSONKey(deps[j]) })
	inputs, err := runtimeInputs(unit.InputBindings, points)
	if err != nil {
		return nil, fmt.Errorf("计算单元 %s: %w", unit.ID, err)
	}
	outputs := make([]any, 0, len(unit.Outputs))
	seenOutputs := map[string]bool{}
	for _, output := range unit.Outputs {
		point, ok := points[output.DatapointID]
		if !ok {
			return nil, fmt.Errorf("计算单元 %s 输出引用不存在数据点", unit.ID)
		}
		if seenOutputs[output.OutputKey] || seenOutputs["datapoint:"+output.DatapointID] {
			return nil, fmt.Errorf("计算单元 %s 输出键重复", unit.ID)
		}
		seenOutputs[output.OutputKey] = true
		seenOutputs["datapoint:"+output.DatapointID] = true
		policy, defaultValue, err := validateComputeOutputTarget(unit.ID, output, point)
		if err != nil {
			return nil, fmt.Errorf("计算单元 %s 输出 %s 投影无效: %w", unit.ID, output.OutputKey, err)
		}
		item := map[string]any{"datapointId": output.DatapointID, "outputKey": output.OutputKey, "path": output.Path, "name": output.Name, "description": nullableString(output.Description), "dataType": output.DataType, "unit": nullableString(output.Unit), "precisionNum": nullableInt(output.PrecisionNum), "sortOrder": output.SortOrder, "nullPolicy": policy}
		if policy == "default" {
			item["defaultValue"] = defaultValue
		}
		outputs = append(outputs, item)
	}
	if len(outputs) == 0 {
		return nil, fmt.Errorf("计算单元 %s 缺少输出", unit.ID)
	}
	sort.Slice(outputs, func(i, j int) bool {
		a, b := outputs[i].(map[string]any), outputs[j].(map[string]any)
		if a["sortOrder"].(int) == b["sortOrder"].(int) {
			return a["outputKey"].(string) < b["outputKey"].(string)
		}
		return a["sortOrder"].(int) < b["sortOrder"].(int)
	})
	trigger, err := runtimeTrigger(unit.TriggerType, unit.TriggerConfig, points, inputs)
	if err != nil {
		return nil, fmt.Errorf("计算单元 %s: %w", unit.ID, err)
	}
	return map[string]any{"id": unit.ID, "revision": unit.Revision, "name": unit.Name, "description": nullableString(unit.Description), "language": unit.Language, "scriptCode": unit.ScriptCode, "enabled": unit.IsEnabled, "timeoutMs": unit.TimeoutMS, "dependencies": deps, "inputs": inputs, "outputs": outputs, "trigger": trigger}, nil
}

func runtimeInputs(bindings map[string]any, points map[string]DataPointRecord) ([]any, error) {
	raw, ok := bindings["datapointVariables"]
	if !ok {
		return []any{}, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("inputs 格式无效")
	}
	seen := map[string]bool{}
	result := make([]any, 0, len(items))
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("input 格式无效")
		}
		id, a := stringField(item, "datapointId")
		alias, b := stringField(item, "alias")
		point, exists := points[id]
		if !a || !b || !exists || point.Status == "invalid" || seen[alias] || alias == "true" || alias == "false" {
			return nil, fmt.Errorf("input 引用或别名无效")
		}
		seen[alias] = true
		result = append(result, map[string]any{"alias": alias, "datapointId": id, "path": point.Path, "dataType": point.DataType})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].(map[string]any)["alias"].(string) < result[j].(map[string]any)["alias"].(string)
	})
	return result, nil
}

func runtimeTrigger(kind string, config map[string]any, points map[string]DataPointRecord, inputs []any) (map[string]any, error) {
	switch kind {
	case "manual":
		if len(config) != 0 && !(len(config) == 1 && config["kind"] == "manual") {
			return nil, fmt.Errorf("manual 触发配置无效")
		}
		return map[string]any{"kind": "manual"}, nil
	case "schedule":
		schedule := cloneSnapshotObject(config)
		if schedule["kind"] == nil {
			return nil, fmt.Errorf("schedule 缺少 kind")
		}
		return map[string]any{"kind": "schedule", "schedule": schedule}, nil
	case "datapoint_change":
		id, ok := stringField(config, "datapointId")
		point, exists := points[id]
		if !ok || !exists || point.Status == "invalid" {
			return nil, fmt.Errorf("点变触发引用无效")
		}
		out := cloneSnapshotObject(config)
		out["kind"] = "datapoint_change"
		out["datapointId"] = id
		out["path"] = point.Path
		out["dataType"] = point.DataType
		return out, nil
	case "condition":
		expression, ok := stringField(config, "expression")
		if !ok {
			return nil, fmt.Errorf("条件触发缺少表达式")
		}
		phases, ok := config["phases"].([]any)
		if !ok || len(phases) == 0 {
			return nil, fmt.Errorf("条件触发 phases 无效")
		}
		debounce, ok := integerAny(config["debounceMs"])
		if !ok {
			return nil, fmt.Errorf("条件触发 debounceMs 无效")
		}
		return map[string]any{"kind": "condition", "expression": expression, "phases": cloneSnapshotAnyArray(phases), "variables": inputs, "debounceMs": debounce}, nil
	default:
		return nil, fmt.Errorf("触发类型不受支持")
	}
}

func buildRuntimeAlarm(alarm AlarmItemRecord, points map[string]DataPointRecord) (map[string]any, error) {
	if alarm.Revision < 1 {
		return nil, fmt.Errorf("报警项 %s revision 无效", alarm.ID)
	}
	type alarmInputOrder struct {
		item  map[string]any
		order int
	}
	orderedInputs := make([]alarmInputOrder, 0, len(alarm.Inputs))
	seen := map[string]bool{}
	seenDatapoints := map[string]bool{}
	for _, input := range alarm.Inputs {
		point, ok := points[input.DatapointID]
		if !ok || point.Status == "invalid" || seen[input.InputKey] || seenDatapoints[input.DatapointID] || point.Path != input.Path || point.DataType != input.DataType {
			return nil, fmt.Errorf("报警项 %s 输入引用无效", alarm.ID)
		}
		seen[input.InputKey] = true
		seenDatapoints[input.DatapointID] = true
		orderedInputs = append(orderedInputs, alarmInputOrder{item: map[string]any{"alias": input.InputKey, "datapointId": input.DatapointID, "path": input.Path, "dataType": input.DataType}, order: input.SortOrder})
	}
	sort.Slice(orderedInputs, func(i, j int) bool {
		if orderedInputs[i].order == orderedInputs[j].order {
			return orderedInputs[i].item["alias"].(string) < orderedInputs[j].item["alias"].(string)
		}
		return orderedInputs[i].order < orderedInputs[j].order
	})
	inputs := make([]any, 0, len(orderedInputs))
	for _, input := range orderedInputs {
		inputs = append(inputs, input.item)
	}
	conditions := make([]any, 0, len(alarm.Conditions))
	for _, condition := range alarm.Conditions {
		if math.IsNaN(condition.Deadband) || math.IsInf(condition.Deadband, 0) {
			return nil, fmt.Errorf("报警项 %s 存在非有限 deadband", alarm.ID)
		}
		operator := condition.Operator
		// 开发态旧基线的 offline 使用 is_offline；运行契约唯一接受 is。
		if condition.Kind == "offline" && operator == "is_offline" {
			operator = "is"
		}
		conditions = append(conditions, map[string]any{"id": condition.ID, "kind": condition.Kind, "operator": operator, "params": cloneSnapshotObject(condition.Params), "severity": condition.Severity, "triggerDelayMs": condition.TriggerDelayMS, "clearDelayMs": condition.ClearDelayMS, "deadband": condition.Deadband, "_sortOrder": condition.SortOrder})
	}
	sort.Slice(conditions, func(i, j int) bool {
		a, b := conditions[i].(map[string]any), conditions[j].(map[string]any)
		if a["_sortOrder"].(int) == b["_sortOrder"].(int) {
			return a["id"].(string) < b["id"].(string)
		}
		return a["_sortOrder"].(int) < b["_sortOrder"].(int)
	})
	for _, raw := range conditions {
		delete(raw.(map[string]any), "_sortOrder")
	}
	if alarm.EvaluationMode == "highest_matching" {
		if alarm.Mode != "point" || !strictThresholdOrder(conditions) {
			return nil, fmt.Errorf("报警项 %s highest_matching 阈值必须唯一且按严重度顺序向外单调", alarm.ID)
		}
	}
	result := map[string]any{"id": alarm.ID, "revision": alarm.Revision, "displayName": alarm.DisplayName, "enabled": alarm.IsEnabled, "mode": alarm.Mode, "evaluationMode": alarm.EvaluationMode, "inputs": inputs, "conditions": conditions}
	if alarm.Mode == "point" {
		if alarm.DatapointID == nil || len(inputs) != 1 || inputs[0].(map[string]any)["datapointId"] != *alarm.DatapointID {
			return nil, fmt.Errorf("报警项 %s 普通点引用无效", alarm.ID)
		}
	} else if alarm.Mode == "derived" {
		if strings.TrimSpace(alarm.DerivedExpression) == "" {
			return nil, fmt.Errorf("报警项 %s 组合表达式为空", alarm.ID)
		}
		result["derivedExpression"] = alarm.DerivedExpression
	} else {
		return nil, fmt.Errorf("报警项 %s 模式无效", alarm.ID)
	}
	return result, nil
}

func strictThresholdOrder(conditions []any) bool {
	if len(conditions) < 1 {
		return false
	}
	direction := 0
	var previous *big.Rat
	seen := map[string]bool{}
	for index, raw := range conditions {
		condition := raw.(map[string]any)
		kind, _ := condition["kind"].(string)
		operator, _ := condition["operator"].(string)
		params, _ := condition["params"].(map[string]any)
		threshold, ok := exactNumber(params["threshold"])
		if kind != "threshold" || !ok || seen[threshold.RatString()] {
			return false
		}
		seen[threshold.RatString()] = true
		currentDirection := 0
		if operator == "gt" || operator == "gte" {
			currentDirection = 1
		} else if operator == "lt" || operator == "lte" {
			currentDirection = -1
		} else {
			return false
		}
		if index > 0 && (currentDirection != direction || (direction == 1 && threshold.Cmp(previous) <= 0) || (direction == -1 && threshold.Cmp(previous) >= 0)) {
			return false
		}
		direction = currentDirection
		previous = threshold
	}
	return true
}
func exactNumber(value any) (*big.Rat, bool) {
	switch n := value.(type) {
	case json.Number:
		parsed, ok := new(big.Rat).SetString(n.String())
		return parsed, ok
	case int:
		return new(big.Rat).SetInt64(int64(n)), true
	case int64:
		return new(big.Rat).SetInt64(n), true
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 9007199254740991 {
			return nil, false
		}
		parsed, ok := new(big.Rat).SetString(strconv.FormatFloat(n, 'g', -1, 64))
		return parsed, ok
	}
	return nil, false
}

// BuildCollectorRuntimeArtifactV1 仅投影可部署的逻辑采集信息；现场连接配置只校验，绝不写入输出。
// artifactID/revision 必须由未来 dev_core 的受信、不可变 Release 版本对象提供；data_service 不提供可伪造它们的公网入口。
func BuildCollectorRuntimeArtifactV1(schemaRoot, projectID, artifactID string, revision int64, collectorVersion string, snapshot *ProjectSnapshot) (*CollectorRuntimeArtifactV1, error) {
	if snapshot == nil || revision < 1 || strings.TrimSpace(artifactID) == "" || strings.TrimSpace(collectorVersion) == "" {
		return nil, fmt.Errorf("采集器产物参数无效")
	}
	connections := make([]any, 0)
	byID := map[string]SnapshotCollectorConnectionRecord{}
	for _, conn := range snapshot.CollectorConnections {
		if !conn.IsEnabled {
			continue
		}
		if err := validateRuntimeCollectorConnection(schemaRoot, conn); err != nil {
			return nil, err
		}
		defaultAcquisition, err := strictAcquisition(conn.DefaultAcquisition)
		if err != nil {
			return nil, fmt.Errorf("连接 %s 默认采集策略无效: %w", conn.ID, err)
		}
		byID[conn.ID] = conn
		connections = append(connections, map[string]any{"connectionId": conn.ID, "protocolFamily": conn.ProtocolFamily, "driverId": conn.DriverID, "driverVersion": conn.DriverVersion, "schemaVersion": conn.SchemaVersion, "enabled": true, "requiredOperations": []any{"point.read"}, "defaultAcquisition": defaultAcquisition})
	}
	if len(connections) == 0 {
		return nil, fmt.Errorf("没有启用的受支持采集连接")
	}
	sort.Slice(connections, func(i, j int) bool {
		return connections[i].(map[string]any)["connectionId"].(string) < connections[j].(map[string]any)["connectionId"].(string)
	})
	dataPointsByVariable := map[string][]DataPointRecord{}
	collectorPointsByID := map[string]SnapshotCollectorPointRecord{}
	for _, point := range snapshot.CollectorPoints {
		collectorPointsByID[point.ID] = point
	}
	for _, point := range snapshot.DataPoints {
		if point.SourceType == "collector.point" && point.Status == "active" {
			if point.SourceID == nil {
				return nil, fmt.Errorf("活动采集数据点 %s 缺少有效变量反向引用", point.ID)
			}
			variable, exists := collectorPointsByID[*point.SourceID]
			if !exists || !variable.Enabled {
				return nil, fmt.Errorf("活动采集数据点 %s 反向引用了不存在或未启用的采集变量", point.ID)
			}
			dataPointsByVariable[*point.SourceID] = append(dataPointsByVariable[*point.SourceID], point)
		}
	}
	mappings := make([]any, 0)
	seenDataPoint := map[string]bool{}
	for _, point := range snapshot.CollectorPoints {
		if !point.Enabled {
			continue
		}
		conn, ok := byID[point.ConnectionID]
		if !ok {
			return nil, fmt.Errorf("启用采集点 %s 引用了未启用或不支持连接", point.ID)
		}
		candidates := dataPointsByVariable[point.ID]
		if len(candidates) != 1 || seenDataPoint[candidates[0].ID] {
			return nil, fmt.Errorf("采集点 %s 没有唯一正式数据点", point.ID)
		}
		dataPoint := candidates[0]
		if _, err := runtimeSourceConfig(dataPoint, map[string]string{point.ID: point.ConnectionID}); err != nil {
			return nil, err
		}
		seenDataPoint[dataPoint.ID] = true
		if err := validateRuntimeCollectorPoint(schemaRoot, conn, point); err != nil {
			return nil, err
		}
		effective, err := strictAcquisition(conn.DefaultAcquisition)
		if err != nil {
			return nil, fmt.Errorf("采集点 %s 默认采集策略无效: %w", point.ID, err)
		}
		m := map[string]any{"datapointId": dataPoint.ID, "connectionId": point.ConnectionID, "variableId": point.ID, "dataType": point.DataType, "elementCount": point.ElementCount, "addressSchemaVersion": point.AddressSchemaVersion, "address": cloneSnapshotObject(point.Address), "readOptions": cloneSnapshotObject(point.ReadOptions), "acquisitionMode": point.AcquisitionMode, "effectiveAcquisition": effective, "enabled": true}
		if point.AcquisitionMode == "override" {
			override, err := strictAcquisitionOverlay(conn.DefaultAcquisition, point.AcquisitionOverrides)
			if err != nil {
				return nil, fmt.Errorf("采集点 %s 采集覆盖无效: %w", point.ID, err)
			}
			m["acquisitionOverrides"] = cloneSnapshotObject(point.AcquisitionOverrides)
			m["effectiveAcquisition"] = override
		} else if point.AcquisitionMode != "inherit" {
			return nil, fmt.Errorf("采集点 %s acquisitionMode 无效", point.ID)
		}
		mappings = append(mappings, m)
	}
	if len(mappings) == 0 {
		return nil, fmt.Errorf("没有启用采集点")
	}
	sort.Slice(mappings, func(i, j int) bool {
		return mappings[i].(map[string]any)["datapointId"].(string) < mappings[j].(map[string]any)["datapointId"].(string)
	})
	artifact := &CollectorRuntimeArtifactV1{"collector-runtime-artifact.v1", artifactID, revision, projectID, collectorVersion, connections, mappings, map[string]any{"fsync": "before-publish", "checksum": "crc32c", "commitMarker": "after-jetstream-puback"}}
	if err := validateRuntimeArtifactSchema(schemaRoot, "collector-runtime-artifact.schema.json", artifact); err != nil {
		return nil, err
	}
	return artifact, nil
}

func validateRuntimeCollectorConnection(schemaRoot string, c SnapshotCollectorConnectionRecord) error {
	if c.DriverID != "modbus.tcp" && c.DriverID != "opcua.standard" {
		return fmt.Errorf("连接 %s 使用未认证运行驱动 %s", c.ID, c.DriverID)
	}
	if c.SchemaVersion != 1 || c.DriverVersion != "1.0.0" || (c.DriverID == "modbus.tcp" && c.ProtocolFamily != "modbus") || (c.DriverID == "opcua.standard" && c.ProtocolFamily != "opcua") {
		return fmt.Errorf("连接 %s 驱动契约版本或协议族无效", c.ID)
	}
	if err := validateCollectorProtocolSchema(schemaRoot, c.DriverID, "connection.schema.json", c.Config); err != nil {
		return fmt.Errorf("连接 %s: %w", c.ID, err)
	}
	if c.DriverID == "modbus.tcp" {
		if err := validateExactKeys(c.Config, []string{"host", "port", "connectTimeoutMs", "receiveTimeoutMs", "dataFormat"}, []string{"host", "port"}); err != nil {
			return err
		}
		if !nonEmptyString(c.Config["host"]) || !boundedInt(c.Config["port"], 1, 65535) || optionalBoundedInt(c.Config, "connectTimeoutMs", 100, 120000) == false || optionalBoundedInt(c.Config, "receiveTimeoutMs", 100, 120000) == false || !optionalEnum(c.Config, "dataFormat", "ABCD", "BADC", "CDAB", "DCBA") {
			return fmt.Errorf("连接 %s Modbus 逻辑配置不符合协议", c.ID)
		}
		return nil
	}
	if err := validateExactKeys(c.Config, []string{"host", "port", "endpointPath", "securityMode", "securityPolicy", "authenticationType", "timeoutMs"}, []string{"host", "port", "endpointPath", "securityMode", "securityPolicy", "authenticationType"}); err != nil {
		return err
	}
	if !nonEmptyString(c.Config["host"]) || !boundedInt(c.Config["port"], 1, 65535) || !stringValue(c.Config["endpointPath"]) || !equalsString(c.Config["securityMode"], "None") || !equalsString(c.Config["securityPolicy"], "None") || !equalsString(c.Config["authenticationType"], "anonymous") || !optionalBoundedInt(c.Config, "timeoutMs", 1000, 120000) {
		return fmt.Errorf("连接 %s OPC UA 逻辑配置不符合协议", c.ID)
	}
	return nil
}
func validateRuntimeCollectorPoint(schemaRoot string, c SnapshotCollectorConnectionRecord, p SnapshotCollectorPointRecord) error {
	if p.AddressSchemaVersion != 1 || p.ElementCount < 1 || p.ElementCount > 4096 {
		return fmt.Errorf("采集点 %s 地址版本或元素数量无效", p.ID)
	}
	if p.DataType == "object" || p.DataType == "array" || p.DataType == "bytes" && p.ElementCount != 1 {
		return fmt.Errorf("采集点 %s 数据类型或元素数量无效", p.ID)
	}
	if (c.DriverID == "modbus.tcp" && !containsString([]string{"bool", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float32", "float64", "string", "bytes"}, p.DataType)) || (c.DriverID == "opcua.standard" && !containsString([]string{"bool", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float32", "float64", "decimal", "string", "bytes", "datetime"}, p.DataType)) {
		return fmt.Errorf("采集点 %s 数据类型不受驱动支持", p.ID)
	}
	if err := validateCollectorProtocolSchema(schemaRoot, c.DriverID, "address.schema.json", p.Address); err != nil {
		return fmt.Errorf("采集点 %s: %w", p.ID, err)
	}
	if err := validateCollectorProtocolSchema(schemaRoot, c.DriverID, "read-options.schema.json", p.ReadOptions); err != nil {
		return fmt.Errorf("采集点 %s readOptions: %w", p.ID, err)
	}
	if c.DriverID == "modbus.tcp" {
		if err := validateExactKeys(p.Address, []string{"station", "area", "address", "bitIndex"}, []string{"station", "area", "address"}); err != nil {
			return err
		}
		if !boundedInt(p.Address["station"], 0, 255) || !boundedInt(p.Address["address"], 0, 65535) || !enumValue(p.Address["area"], "coil", "discreteInput", "inputRegister", "holdingRegister") || (p.Address["bitIndex"] != nil && !boundedInt(p.Address["bitIndex"], 0, 15)) {
			return fmt.Errorf("采集点 %s Modbus 地址无效", p.ID)
		}
		if (p.Address["area"] == "coil" || p.Address["area"] == "discreteInput") && p.DataType != "bool" {
			return fmt.Errorf("采集点 %s Modbus 线圈只支持 bool", p.ID)
		}
		return nil
	}
	if err := validateExactKeys(p.Address, []string{"nodeId"}, []string{"nodeId"}); err != nil {
		return err
	}
	if !nonEmptyString(p.Address["nodeId"]) {
		return fmt.Errorf("采集点 %s OPC UA nodeId 无效", p.ID)
	}
	return nil
}

func validateCollectorProtocolSchema(schemaRoot, driverID, file string, value any) error {
	manifestBytes, err := os.ReadFile(filepath.Join(schemaRoot, "..", "collector-protocols", driverID, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest struct {
		ProtocolFamily, DriverID, DriverVersion string
		SchemaVersion                           int
		Operations, DataTypes                   []string
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return err
	}
	if manifest.DriverID != driverID || manifest.SchemaVersion != 1 || manifest.DriverVersion != "1.0.0" || !containsString(manifest.Operations, "point.read") {
		return fmt.Errorf("manifest 不满足运行态要求")
	}
	schemaBytes, err := os.ReadFile(filepath.Join(schemaRoot, "..", "collector-protocols", driverID, file))
	if err != nil {
		return err
	}
	var schemaDoc any
	schemaDecoder := json.NewDecoder(bytes.NewReader(schemaBytes))
	schemaDecoder.UseNumber()
	if err := schemaDecoder.Decode(&schemaDoc); err != nil {
		return err
	}
	var schemaExtra any
	if err := schemaDecoder.Decode(&schemaExtra); err != io.EOF {
		return fmt.Errorf("协议 schema 存在多余 JSON 值")
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	resource := "urn:induforge:runtime:" + driverID + ":" + file
	if err := compiler.AddResource(resource, schemaDoc); err != nil {
		return err
	}
	schema, err := compiler.Compile(resource)
	if err != nil {
		return err
	}
	// jsonschema 接收的实际值也必须保留 json.Number；否则 schema 的数值
	// 边界会在 float64 中先丢失精度，和运行制品的 typed default 语义不一致。
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("协议 schema 输入存在多余 JSON 值")
	}
	if err := schema.Validate(document); err != nil {
		return err
	}
	return nil
}
func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func stringValue(value any) bool { _, ok := value.(string); return ok }
func nonEmptyString(value any) bool {
	text, ok := value.(string)
	return ok && strings.TrimSpace(text) != ""
}
func equalsString(value any, expected string) bool {
	text, ok := value.(string)
	return ok && text == expected
}
func enumValue(value any, options ...string) bool {
	text, ok := value.(string)
	if !ok {
		return false
	}
	for _, option := range options {
		if text == option {
			return true
		}
	}
	return false
}
func optionalEnum(values map[string]any, key string, options ...string) bool {
	value, ok := values[key]
	return !ok || enumValue(value, options...)
}
func boundedInt(value any, min, max int) bool {
	number, ok := integerAny(value)
	return ok && number >= min && number <= max
}
func optionalBoundedInt(values map[string]any, key string, min, max int) bool {
	value, ok := values[key]
	return !ok || boundedInt(value, min, max)
}
func validateExactKeys(v map[string]any, allowed, required []string) error {
	set := map[string]bool{}
	for _, key := range allowed {
		set[key] = true
	}
	for key := range v {
		if !set[key] {
			return fmt.Errorf("存在未定义字段 %s", key)
		}
	}
	for _, key := range required {
		if _, ok := v[key]; !ok {
			return fmt.Errorf("缺少必填字段 %s", key)
		}
	}
	return nil
}
func strictAcquisition(v map[string]any) (map[string]any, error) {
	return strictAcquisitionOverlay(map[string]any{"intervalMs": 1000, "deadband": 0.0, "changeOnly": false}, v)
}
func strictAcquisitionOverlay(base, over map[string]any) (map[string]any, error) {
	if err := validateAcquisitionFields(base, false); err != nil {
		return nil, err
	}
	if err := validateAcquisitionFields(over, true); err != nil {
		return nil, err
	}
	result := map[string]any{"intervalMs": 1000, "deadband": 0.0, "changeOnly": false}
	for key, value := range base {
		result[key] = value
	}
	for key, value := range over {
		result[key] = value
	}
	if err := validateAcquisitionFields(result, false); err != nil {
		return nil, err
	}
	return result, nil
}

func validateAcquisitionFields(values map[string]any, requireNonEmpty bool) error {
	if requireNonEmpty && len(values) == 0 {
		return fmt.Errorf("覆盖不能为空")
	}
	for key := range values {
		if key != "intervalMs" && key != "deadband" && key != "changeOnly" {
			return fmt.Errorf("不支持字段 %s", key)
		}
	}
	if value, ok := values["intervalMs"]; ok && !boundedInt(value, 1, 86400000) {
		return fmt.Errorf("intervalMs 必须是 1..86400000 的整数")
	}
	if value, ok := values["deadband"]; ok {
		number, ok := finiteAcquisitionNumber(value)
		if !ok || number < 0 {
			return fmt.Errorf("deadband 必须是非负有限数字")
		}
	}
	if value, ok := values["changeOnly"]; ok {
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("changeOnly 必须是布尔值")
		}
	}
	return nil
}

func finiteAcquisitionNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, !math.IsNaN(number) && !math.IsInf(number, 0)
	case float32:
		return float64(number), !math.IsNaN(float64(number)) && !math.IsInf(float64(number), 0)
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case json.Number:
		parsed, err := strconv.ParseFloat(number.String(), 64)
		return parsed, err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
	default:
		return 0, false
	}
}

func parseTypedDefault(raw *string, dataType string) (any, error) {
	if raw == nil {
		return nil, nil
	}
	decoder := json.NewDecoder(strings.NewReader(*raw))
	decoder.UseNumber()
	var result any
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("存在多余 JSON")
	}
	if err := ensureFiniteJSON(result); err != nil {
		return nil, err
	}
	if err := validateTypedDefaultValue(result, dataType); err != nil {
		return nil, err
	}
	return result, nil
}

// ParseTypedJSONDefault 是计算输出、快照和运行制品共用的无损 JSON 默认值解析入口。
func ParseTypedJSONDefault(raw, dataType string) (any, error) {
	return parseTypedDefault(&raw, dataType)
}
func ensureFiniteJSON(v any) error {
	switch x := v.(type) {
	case float64:
		if math.IsInf(x, 0) || math.IsNaN(x) {
			return fmt.Errorf("非有限数字")
		}
	case []any:
		for _, i := range x {
			if err := ensureFiniteJSON(i); err != nil {
				return err
			}
		}
	case map[string]any:
		for _, i := range x {
			if err := ensureFiniteJSON(i); err != nil {
				return err
			}
		}
	}
	return nil
}
func nullableString(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}
func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}
func stringSliceAny(v []string) []any {
	r := make([]any, len(v))
	for i, x := range v {
		r[i] = x
	}
	sort.Slice(r, func(i, j int) bool { return r[i].(string) < r[j].(string) })
	return r
}
func stringField(m map[string]any, key string) (string, bool) {
	v, ok := m[key].(string)
	return strings.TrimSpace(v), ok && strings.TrimSpace(v) != ""
}
func integerAny(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), int64(int(n)) == n
	case float64:
		return int(n), math.Trunc(n) == n
	}
	return 0, false
}
func runtimeNullPolicy(v string) string {
	switch v {
	case "skip":
		return "skip"
	case "default":
		return "default"
	case "error", "propagate", "":
		return "propagate"
	default:
		return ""
	}
}

func isExactComputeOutputSourceConfig(config map[string]any, unitID, outputKey string) bool {
	return len(config) == 2 && config["computeUnitId"] == unitID && config["outputKey"] == outputKey
}

// validateComputeOutputTarget 是快照写入和运行制品构建共同使用的输出投影边界。
// 开发态 sourceConfig 保留 outputKey 以维护一对一关系；产物层随后才会冻结为 {computeId}。
func validateComputeOutputTarget(unitID string, output ComputeOutputRecord, point DataPointRecord) (string, any, error) {
	if point.Status != "active" {
		return "", nil, fmt.Errorf("目标数据点必须为 active")
	}
	if point.SourceType != "calc.output" || point.SourceID == nil || *point.SourceID != unitID || !isExactComputeOutputSourceConfig(point.SourceConfig, unitID, output.OutputKey) {
		return "", nil, fmt.Errorf("目标数据点 calc.output 来源投影不一致")
	}
	if point.Path != output.Path || point.DataType != output.DataType {
		return "", nil, fmt.Errorf("目标数据点 path 或 dataType 与输出不一致")
	}
	storagePolicy := strings.TrimSpace(output.NullPolicy)
	if storagePolicy == "" {
		storagePolicy = "error"
	}
	if storagePolicy != "error" && storagePolicy != "skip" && storagePolicy != "default" {
		return "", nil, fmt.Errorf("nullPolicy 无效")
	}
	if storagePolicy != "default" {
		if output.DefaultValue != nil || point.DefaultValue != nil {
			return "", nil, fmt.Errorf("非 default 输出不能携带默认值")
		}
		return runtimeNullPolicy(storagePolicy), nil, nil
	}
	if output.DefaultValue == nil || point.DefaultValue == nil || !jsonPayloadEqual([]byte(*output.DefaultValue), []byte(*point.DefaultValue)) {
		return "", nil, fmt.Errorf("default 输出默认值与数据点投影不一致")
	}
	parsed, err := parseTypedDefault(output.DefaultValue, output.DataType)
	if err != nil || parsed == nil {
		if err != nil {
			return "", nil, fmt.Errorf("default 输出默认值无效: %w", err)
		}
		return "", nil, fmt.Errorf("default 输出默认值不能为 null")
	}
	return runtimeNullPolicy(storagePolicy), parsed, nil
}
func validateTypedDefaultValue(value any, dataType string) error {
	if value == nil {
		return nil
	}
	integerRange := func(min, max int64) bool {
		n, ok := value.(json.Number)
		if !ok {
			return false
		}
		parsed, err := n.Int64()
		return err == nil && parsed >= min && parsed <= max
	}
	uintRange := func(max uint64) bool {
		n, ok := value.(json.Number)
		if !ok {
			return false
		}
		parsed, err := strconv.ParseUint(n.String(), 10, 64)
		return err == nil && parsed <= max
	}
	switch dataType {
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("必须是 bool")
		}
	case "int8":
		if !integerRange(-128, 127) {
			return fmt.Errorf("超出 int8 范围")
		}
	case "int16":
		if !integerRange(-32768, 32767) {
			return fmt.Errorf("超出 int16 范围")
		}
	case "int32":
		if !integerRange(-2147483648, 2147483647) {
			return fmt.Errorf("超出 int32 范围")
		}
	case "int64":
		if !integerRange(math.MinInt64, math.MaxInt64) {
			return fmt.Errorf("超出 int64 范围")
		}
	case "uint8":
		if !uintRange(255) {
			return fmt.Errorf("超出 uint8 范围")
		}
	case "uint16":
		if !uintRange(65535) {
			return fmt.Errorf("超出 uint16 范围")
		}
	case "uint32":
		if !uintRange(4294967295) {
			return fmt.Errorf("超出 uint32 范围")
		}
	case "uint64":
		if !uintRange(math.MaxUint64) {
			return fmt.Errorf("超出 uint64 范围")
		}
	case "float32", "float64":
		number, ok := value.(json.Number)
		if !ok {
			return fmt.Errorf("必须是数字")
		}
		parsed, err := strconv.ParseFloat(number.String(), 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return fmt.Errorf("必须是有限数字")
		}
		if dataType == "float32" && (parsed > math.MaxFloat32 || parsed < -math.MaxFloat32) {
			return fmt.Errorf("超出 float32 范围")
		}
	case "decimal":
		number, ok := value.(json.Number)
		if !ok {
			return fmt.Errorf("必须是数字")
		}
		// 不经 float64，保留大指数、高精小数和 2^53 以上 JSON decimal 的精确语义。
		if _, ok := new(big.Rat).SetString(number.String()); !ok {
			return fmt.Errorf("decimal 数字无效")
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("必须是字符串")
		}
	case "bytes":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("必须是 base64 字符串")
		}
		decoded, err := base64.StdEncoding.DecodeString(text)
		if err != nil || base64.StdEncoding.EncodeToString(decoded) != text {
			return fmt.Errorf("base64 无效")
		}
	case "datetime":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("必须是 UTC 时间字符串")
		}
		parsed, err := time.Parse(time.RFC3339Nano, text)
		if err != nil || parsed.Location() != time.UTC || !strings.HasSuffix(text, "Z") {
			return fmt.Errorf("必须是 UTC RFC3339")
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("必须是数组")
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("必须是对象")
		}
	default:
		return fmt.Errorf("未知数据类型")
	}
	return nil
}
func canonicalJSONKey(v any) string { b, _ := json.Marshal(v); return string(b) }

func validateRuntimeArtifactSchema(dir, file string, artifact any) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("必须显式提供 runtime schema 根目录")
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	for _, name := range []string{"common.schema.json", file} {
		bytes, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			return readErr
		}
		var doc any
		if err := json.Unmarshal(bytes, &doc); err != nil {
			return err
		}
		url := "https://induforge.dev/contracts/runtime/" + name
		if err := compiler.AddResource(url, doc); err != nil {
			return err
		}
	}
	schema, err := compiler.Compile("https://induforge.dev/contracts/runtime/" + file)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(artifact)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var doc any
	if err := decoder.Decode(&doc); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("artifact JSON 存在多余值")
	}
	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("%s 验证失败: %w", file, err)
	}
	return nil
}
