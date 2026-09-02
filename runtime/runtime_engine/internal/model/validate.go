package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

var canonicalUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// ValidateEngineConfig 只处理 Schema 无法表达的集合、fencing 和消费者拓扑约束。
func ValidateEngineConfig(config EngineConfig) error {
	// 配置版本决定部署边界，不能把 v1 的 k3s/release-pvc 语义隐式解释为本机进程。
	switch config.SchemaVersion {
	case "runtime-engine.config.v1":
		if config.ExecutionForm != "k3s-workload" || config.ArtifactMount.Source != "release-pvc" || config.NodeID != "" {
			return fmt.Errorf("runtime-engine.config.v1 必须使用 k3s-workload/release-pvc 且不得声明 nodeId")
		}
	case "runtime-engine.config.v2":
		if config.ExecutionForm != "native-linux" || config.ArtifactMount.Source != "native-release" || config.NodeID == "" {
			return fmt.Errorf("runtime-engine.config.v2 必须使用 nodeId、native-linux 和 native-release")
		}
	default:
		return fmt.Errorf("不支持的 runtime-engine config schemaVersion %q", config.SchemaVersion)
	}
	streams := []string{config.JetStream.DataRawStream, config.JetStream.DataDerivedStream, config.JetStream.EventStream, config.JetStream.CommandStream, config.JetStream.DeadLetterStream}
	seenStreams := map[string]struct{}{}
	for _, stream := range streams {
		if stream == "" {
			return fmt.Errorf("jetStream stream 不能为空")
		}
		if _, ok := seenStreams[stream]; ok {
			return fmt.Errorf("jetStream stream 必须为不同 stableId")
		}
		seenStreams[stream] = struct{}{}
	}
	if !canonicalUUID.MatchString(config.ProjectID) {
		return fmt.Errorf("config projectId 不是小写 canonical UUID: %q", config.ProjectID)
	}
	roles := make(map[string]struct{}, len(config.Roles))
	for _, role := range config.Roles {
		if _, exists := roles[role]; exists {
			return fmt.Errorf("roles 存在重复角色 %q", role)
		}
		roles[role] = struct{}{}
	}
	assignments := make(map[string]struct{}, len(config.RoleAssignments))
	for _, assignment := range config.RoleAssignments {
		if assignment.Ownership.Epoch < 1 {
			return fmt.Errorf("roleAssignment %q 的 ownership.epoch 必须至少为 1", assignment.Role)
		}
		if _, exists := assignments[assignment.Role]; exists {
			return fmt.Errorf("roleAssignments 存在重复角色 %q", assignment.Role)
		}
		assignments[assignment.Role] = struct{}{}
	}
	if !sameKeys(roles, assignments) {
		return fmt.Errorf("roles 与 roleAssignments 必须一一对应")
	}
	computeProducers := make(map[string]struct{})
	collectorProducers := make(map[string]struct{})
	collectorFiles := make(map[string]struct{})
	collectorArtifacts := make(map[string]struct{})
	alarmProducerCount := 0
	manualProducerCount := 0
	for _, producer := range config.ProducerAssignments {
		if producer.Ownership.Epoch < 1 {
			return fmt.Errorf("producerAssignment %q 的 ownership.epoch 必须至少为 1", producer.ProducerType)
		}
		switch producer.ProducerType {
		case "manual":
			if producer.ManualID != "runtime-api" {
				return fmt.Errorf("manual producer 必须使用 runtime-api 身份")
			}
			manualProducerCount++
			if manualProducerCount > 1 {
				return fmt.Errorf("producerAssignments 只能有一个 manual producer")
			}
		case "collector":
			if _, exists := collectorProducers[producer.CollectorID]; exists {
				return fmt.Errorf("producerAssignments 存在重复 collectorId %q", producer.CollectorID)
			}
			if producer.CollectorArtifact == nil || producer.CollectorArtifact.ArtifactFile == "" {
				return fmt.Errorf("collector producer %q 必须有唯一 collectorArtifact", producer.CollectorID)
			}
			if _, exists := collectorFiles[producer.CollectorArtifact.ArtifactFile]; exists {
				return fmt.Errorf("collectorArtifact 文件 %q 被多个 collector producer 引用", producer.CollectorArtifact.ArtifactFile)
			}
			artifactKey := producer.CollectorArtifact.Artifact.ArtifactID + "\x1f" + fmt.Sprint(producer.CollectorArtifact.Artifact.ArtifactRevision) + "\x1f" + producer.CollectorArtifact.Artifact.ArtifactDigest
			if _, exists := collectorArtifacts[artifactKey]; exists {
				return fmt.Errorf("collectorArtifact %q 被多个 collector producer 重复引用", producer.CollectorArtifact.Artifact.ArtifactID)
			}
			collectorFiles[producer.CollectorArtifact.ArtifactFile] = struct{}{}
			collectorArtifacts[artifactKey] = struct{}{}
			collectorProducers[producer.CollectorID] = struct{}{}
		case "compute":
			if _, enabled := roles["compute"]; !enabled {
				return fmt.Errorf("compute producer %q 引用未启用 compute role", producer.ComputeID)
			}
			if producer.Role != "compute" {
				return fmt.Errorf("compute producer %q 必须使用 compute role", producer.ComputeID)
			}
			if _, exists := computeProducers[producer.ComputeID]; exists {
				return fmt.Errorf("producerAssignments 存在重复 computeId %q", producer.ComputeID)
			}
			computeProducers[producer.ComputeID] = struct{}{}
		case "alarm":
			if _, enabled := roles["alarm"]; !enabled {
				return fmt.Errorf("alarm producer 引用未启用 alarm role")
			}
			if producer.Role != "alarm" {
				return fmt.Errorf("alarm producer 必须使用 alarm role")
			}
			alarmProducerCount++
			if alarmProducerCount > 1 {
				return fmt.Errorf("producerAssignments 只能有一个 alarm producer")
			}
		default:
			return fmt.Errorf("未知 producerType %q", producer.ProducerType)
		}
	}
	keys, durables := map[string]struct{}{}, map[string]struct{}{}
	coverage := map[string]map[string]bool{}
	for _, consumer := range config.JetStream.Consumers {
		if _, enabled := roles[consumer.Role]; !enabled {
			return fmt.Errorf("consumer %q 引用未启用角色 %q", consumer.ConsumerKey, consumer.Role)
		}
		if _, exists := keys[consumer.ConsumerKey]; exists {
			return fmt.Errorf("jetStream consumers 存在重复 consumerKey %q", consumer.ConsumerKey)
		}
		if _, exists := durables[consumer.DurableName]; exists {
			return fmt.Errorf("jetStream consumers 存在重复 durableName %q", consumer.DurableName)
		}
		keys[consumer.ConsumerKey], durables[consumer.DurableName] = struct{}{}, struct{}{}
		if len(consumer.BackoffMS) > consumer.MaxDeliver {
			return fmt.Errorf("consumer %q 的 backoff 长度不能超过 maxDeliver", consumer.ConsumerKey)
		}
		if consumer.MaxAckPending != 32 || consumer.MaxWaiting != 32 || consumer.MaxRequestBatch != 32 || consumer.MaxRequestExpiresMS != 5000 || consumer.MaxRequestMaxBytes != 1<<20 {
			return fmt.Errorf("consumer %q 的 pull 限制必须使用冻结值", consumer.ConsumerKey)
		}
		for i := 1; i < len(consumer.BackoffMS); i++ {
			if consumer.BackoffMS[i] < consumer.BackoffMS[i-1] {
				return fmt.Errorf("consumer %q 的 backoff 必须非递减", consumer.ConsumerKey)
			}
		}
		kind, expectedStream := subjectKind(consumer.FilterSubject, config.JetStream)
		if kind == "" {
			return fmt.Errorf("consumer %q 的 filterSubject 必须是已冻结 Runtime subject", consumer.ConsumerKey)
		}
		if consumer.Stream != expectedStream {
			return fmt.Errorf("consumer %q 的 stream %q 与 filterSubject %q 不匹配", consumer.ConsumerKey, consumer.Stream, consumer.FilterSubject)
		}
		if consumer.Role == "query" || consumer.Role == "coord" {
			return fmt.Errorf("%s role 不允许 JetStream consumer", consumer.Role)
		}
		if kind == "command" && consumer.Role != "compute" {
			return fmt.Errorf("command consumer 只能属于 compute role")
		}
		if coverage[consumer.Role] == nil {
			coverage[consumer.Role] = map[string]bool{}
		}
		if coverage[consumer.Role][kind] {
			return fmt.Errorf("role %q 的 %s consumer 重复", consumer.Role, kind)
		}
		keyKind := kind
		if keyKind == "computed" {
			keyKind = "derived"
		}
		expectedKey := consumer.Role + "-" + keyKind + "-v1"
		if consumer.ConsumerKey != expectedKey {
			return fmt.Errorf("consumer %q 必须使用冻结 consumerKey %q", consumer.ConsumerKey, expectedKey)
		}
		coverage[consumer.Role][kind] = true
	}
	for _, role := range []string{"writer", "alarm", "compute"} {
		_, enabled := roles[role]
		if enabled && (!coverage[role]["raw"] || !coverage[role]["computed"]) {
			return fmt.Errorf("已启用 %s role 必须恰有 raw 与 computed consumer", role)
		}
		if !enabled && len(coverage[role]) != 0 {
			return fmt.Errorf("未启用 %s role 不允许 consumer", role)
		}
		if role == "compute" && enabled && !coverage[role]["command"] {
			return fmt.Errorf("已启用 compute role 必须恰有 command consumer")
		}
	}
	if _, compute := roles["compute"]; compute && config.ComputeSandbox == nil {
		return fmt.Errorf("compute role 必须配置 computeSandbox")
	}
	return nil
}

// ValidateManualProducerFence 用固定 manual producer 防止 Runtime API 伪造或跨部署推进人工点位。
func ValidateManualProducerFence(config EngineConfig, ownership Ownership) error {
	for _, producer := range config.ProducerAssignments {
		if producer.ProducerType == "manual" && producer.ManualID == "runtime-api" && producer.Ownership == ownership {
			return nil
		}
	}
	return fmt.Errorf("manual producer 的 owner/epoch 未通过 Runtime API fence")
}

// ValidateWALCapacity 复核 Schema 无法比较的高水位与诊断预留容量边界。
func ValidateWALCapacity(capacity WALCapacity) error {
	if capacity.MaxBytes <= 0 || capacity.HighWatermarkBytes <= 0 || capacity.HighWatermarkBytes >= capacity.MaxBytes {
		return fmt.Errorf("WAL 容量必须满足 0 < highWatermarkBytes < maxBytes")
	}
	if capacity.DiagnosticReserveBytes <= 0 || capacity.DiagnosticReserveBytes >= capacity.MaxBytes {
		return fmt.Errorf("WAL diagnosticReserveBytes 必须为小于 maxBytes 的正整数")
	}
	if capacity.DataGapPolicy != "emit-alarm-event" {
		return fmt.Errorf("WAL dataGapPolicy 必须为 emit-alarm-event")
	}
	return nil
}

// ValidateCollectorArtifact 复核采集 Artifact 内 Schema 之外的映射集合约束。
func ValidateCollectorArtifact(artifact CollectorArtifact) error {
	if !canonicalUUID.MatchString(artifact.ProjectID) {
		return fmt.Errorf("collector artifact projectId 不是小写 canonical UUID: %q", artifact.ProjectID)
	}
	connections := make(map[string]struct{}, len(artifact.Connections))
	for _, connection := range artifact.Connections {
		if !canonicalUUID.MatchString(connection.ConnectionID) {
			return fmt.Errorf("collector artifact connectionId 不是小写 canonical UUID: %q", connection.ConnectionID)
		}
		if _, exists := connections[connection.ConnectionID]; exists {
			return fmt.Errorf("collector artifact 存在重复 connectionId %q", connection.ConnectionID)
		}
		connections[connection.ConnectionID] = struct{}{}
	}
	points := map[string]struct{}{}
	variables := map[string]struct{}{}
	for _, mapping := range artifact.PointMappings {
		if !canonicalUUID.MatchString(mapping.DatapointID) || !canonicalUUID.MatchString(mapping.ConnectionID) || !canonicalUUID.MatchString(mapping.VariableID) {
			return fmt.Errorf("collector artifact pointMapping 必须使用 canonical UUID")
		}
		if _, exists := connections[mapping.ConnectionID]; !exists {
			return fmt.Errorf("collector artifact mapping %q 引用不存在 connection %q", mapping.DatapointID, mapping.ConnectionID)
		}
		if _, exists := points[mapping.DatapointID]; exists {
			return fmt.Errorf("collector artifact 存在重复 datapointId %q", mapping.DatapointID)
		}
		variableKey := mapping.ConnectionID + "\x1f" + mapping.VariableID
		if _, exists := variables[variableKey]; exists {
			return fmt.Errorf("collector artifact 存在重复 connectionId/variableId 映射")
		}
		if !isScalarType(mapping.DataType) && mapping.DataType != "bytes" && mapping.DataType != "datetime" {
			return fmt.Errorf("collector artifact mapping %q 使用不支持 scalar dataType %q", mapping.DatapointID, mapping.DataType)
		}
		points[mapping.DatapointID] = struct{}{}
		variables[variableKey] = struct{}{}
	}
	return nil
}

// ValidateCollectorIngressMappings 将经过加载器认证的 Collector Artifacts 与项目快照逐项对齐，
// 使 raw ingress 可按 collectorId + source mapping 执行 fencing，而不混淆 consumer token。
func ValidateCollectorIngressMappings(config EngineConfig, project ProjectArtifact, artifacts map[string]CollectorArtifact) error {
	collectorAssignments := map[string]struct{}{}
	for _, producer := range config.ProducerAssignments {
		if producer.ProducerType == "collector" {
			collectorAssignments[producer.CollectorID] = struct{}{}
		}
	}
	if len(collectorAssignments) != len(artifacts) {
		return fmt.Errorf("collector producer assignments 与已验证 artifacts 集合不一致")
	}
	projectPoints := map[string]DataPoint{}
	for _, point := range project.DataPoints {
		projectPoints[point.ID] = point
	}
	seenActive := map[string]int{}
	for collectorID, artifact := range artifacts {
		if _, exists := collectorAssignments[collectorID]; !exists {
			return fmt.Errorf("未配置 collector producer %q 的 artifact", collectorID)
		}
		if artifact.ProjectID != config.ProjectID || artifact.ProjectID != project.ProjectID {
			return fmt.Errorf("collector %q artifact projectId 与 Engine project 不一致", collectorID)
		}
		if err := ValidateCollectorArtifact(artifact); err != nil {
			return fmt.Errorf("collector %q artifact 无效: %w", collectorID, err)
		}
		for _, mapping := range artifact.PointMappings {
			if !mapping.Enabled {
				continue
			}
			point, exists := projectPoints[mapping.DatapointID]
			if !exists || point.Status != "active" || point.SourceType != "collector.point" || point.SourceID == nil {
				return fmt.Errorf("collector %q enabled mapping %q 未对应 active collector.point", collectorID, mapping.DatapointID)
			}
			if *point.SourceID != mapping.VariableID || point.DataType != mapping.DataType {
				return fmt.Errorf("collector %q mapping %q 的 sourceId/dataType 与项目数据点不一致", collectorID, mapping.DatapointID)
			}
			connectionID, err := collectorPointConnectionID(point.SourceConfig)
			if err != nil || connectionID != mapping.ConnectionID {
				return fmt.Errorf("collector %q mapping %q 的 sourceConfig.connectionId 不一致", collectorID, mapping.DatapointID)
			}
			seenActive[mapping.DatapointID]++
		}
	}
	for _, point := range project.DataPoints {
		if point.Status == "active" && point.SourceType == "collector.point" && seenActive[point.ID] != 1 {
			return fmt.Errorf("active collector.point %q 必须恰有一个受信 enabled mapping", point.ID)
		}
	}
	return nil
}

// ValidateRawSourceMapping 在 event ingress 时以受信 collector Artifact 复核 payload source，
// 防止其他已配置 collector 或同连接下的变量伪造 pointId。
func ValidateRawSourceMapping(artifacts map[string]CollectorArtifact, collectorID, pointID, connectionID, variableID string) error {
	artifact, exists := artifacts[collectorID]
	if !exists {
		return fmt.Errorf("raw collector %q 没有受信 artifact", collectorID)
	}
	for _, mapping := range artifact.PointMappings {
		if mapping.Enabled && mapping.DatapointID == pointID && mapping.ConnectionID == connectionID && mapping.VariableID == variableID {
			return nil
		}
	}
	return fmt.Errorf("raw source 不匹配 collector %q 的受信 point mapping", collectorID)
}

// ValidateRawProducerFence 防止非配置 Collector 或过期 owner/epoch 推进 raw 事件副作用。
func ValidateRawProducerFence(config EngineConfig, collectorID string, ownership Ownership) error {
	for _, producer := range config.ProducerAssignments {
		if producer.ProducerType == "collector" && producer.CollectorID == collectorID && producer.Ownership == ownership {
			return nil
		}
	}
	return fmt.Errorf("raw producer %q 的 owner/epoch 未通过 collector fence", collectorID)
}

// ValidateComputedProducerFence 防止非配置 Compute owner 伪造 computed 事件。
func ValidateComputedProducerFence(config EngineConfig, computeID string, ownership Ownership) error {
	for _, producer := range config.ProducerAssignments {
		if producer.ProducerType == "compute" && producer.ComputeID == computeID && producer.Ownership == ownership {
			return nil
		}
	}
	return fmt.Errorf("computed producer %q 的 owner/epoch 未通过 compute fence", computeID)
}

// ValidateAlarmProducerFence 按事件种类使用契约规定的 producer fence。
func ValidateAlarmProducerFence(config EngineConfig, kind, collectorID string, ownership Ownership) error {
	if kind == "data-gap" {
		return ValidateRawProducerFence(config, collectorID, ownership)
	}
	for _, producer := range config.ProducerAssignments {
		if producer.ProducerType == "alarm" && producer.Ownership == ownership {
			return nil
		}
	}
	return fmt.Errorf("alarm-transition producer 的 owner/epoch 未通过 alarm fence")
}

func subjectKind(subject string, streams JetStream) (string, string) {
	switch subject {
	case "data.raw.>":
		return "raw", streams.DataRawStream
	case "data.computed.>":
		return "computed", streams.DataDerivedStream
	case "compute.command.>":
		return "command", streams.CommandStream
	default:
		return "", ""
	}
}

// ValidateProjectArtifact 复核 Artifact 内 UUID、引用、DAG、触发器和报警的跨字段约束。
func ValidateProjectArtifact(artifact ProjectArtifact, config EngineConfig) error {
	if !canonicalUUID.MatchString(artifact.ProjectID) {
		return fmt.Errorf("artifact projectId 不是小写 canonical UUID: %q", artifact.ProjectID)
	}
	if err := validateUTCTimestamp(artifact.GeneratedAt, "artifact generatedAt"); err != nil {
		return err
	}
	if len(artifact.AlarmItems) > MaxAlarmItems {
		return fmt.Errorf("alarmItems 超过 V1 上限 %d", MaxAlarmItems)
	}
	points := make(map[string]DataPoint, len(artifact.DataPoints))
	paths := map[string]string{}
	for _, point := range artifact.DataPoints {
		if !canonicalUUID.MatchString(point.ID) {
			return fmt.Errorf("dataPoint id 不是小写 canonical UUID: %q", point.ID)
		}
		if !isRuntimeDataType(point.DataType) {
			return fmt.Errorf("dataPoint %q 使用不支持 dataType %q", point.ID, point.DataType)
		}
		if err := validateDataPointSource(point); err != nil {
			return fmt.Errorf("dataPoint %q source 无效: %w", point.ID, err)
		}
		if point.RefreshMode != "auto" && point.RefreshMode != "manual" && point.RefreshMode != "subscription" {
			return fmt.Errorf("dataPoint %q refreshMode 无效", point.ID)
		}
		if point.Status != "active" && point.Status != "inactive" && point.Status != "invalid" {
			return fmt.Errorf("dataPoint %q status 无效", point.ID)
		}
		if err := validateWriteGrant(point.RuntimePermission.Write, point.ID); err != nil {
			return err
		}
		if _, exists := points[point.ID]; exists {
			return fmt.Errorf("dataPoints 存在重复 id %q", point.ID)
		}
		if old, exists := paths[point.Path]; exists {
			return fmt.Errorf("dataPoints path %q 同时属于 %q 与 %q", point.Path, old, point.ID)
		}
		points[point.ID], paths[point.Path] = point, point.ID
	}
	computes := map[string]ComputeUnit{}
	computeOutputOwner := map[string]string{}
	for _, unit := range artifact.ComputeUnits {
		if !canonicalUUID.MatchString(unit.ID) {
			return fmt.Errorf("compute id 不是小写 canonical UUID: %q", unit.ID)
		}
		if _, exists := computes[unit.ID]; exists {
			return fmt.Errorf("computeUnits 存在重复 id %q", unit.ID)
		}
		if err := validateCompute(unit, points); err != nil {
			return err
		}
		computes[unit.ID] = unit
		for _, output := range unit.Outputs {
			if owner, exists := computeOutputOwner[output.DatapointID]; exists {
				return fmt.Errorf("datapoint %q 同时被 compute %q 与 %q 输出，写入语义歧义", output.DatapointID, owner, unit.ID)
			}
			computeOutputOwner[output.DatapointID] = unit.ID
		}
	}
	for _, point := range artifact.DataPoints {
		if point.SourceType == "calc.output" {
			if point.SourceID == nil {
				return fmt.Errorf("calc.output datapoint %q 必须有 sourceId", point.ID)
			}
			if _, exists := computes[*point.SourceID]; !exists {
				return fmt.Errorf("calc.output datapoint %q sourceId 未引用 compute", point.ID)
			}
		}
	}
	for computeID := range computeIDs(config) {
		if _, exists := computes[computeID]; !exists {
			return fmt.Errorf("compute producer %q 未在项目 Artifact 中定义", computeID)
		}
	}
	// compute/alarm 被拆成独立工作负载。只有 compute 角色负责所有启用计算单元的
	// producer fence；alarm 只消费已验证的 derived 事件，不应要求挂载 compute producer。
	if hasRole(config, "compute") {
		for _, unit := range artifact.ComputeUnits {
			_, assigned := computeIDs(config)[unit.ID]
			if unit.Enabled && !assigned {
				return fmt.Errorf("enabled compute %q 必须且只能有一个 producer assignment", unit.ID)
			}
			if !unit.Enabled && assigned {
				return fmt.Errorf("disabled compute %q 不得保留 producer assignment", unit.ID)
			}
	}
	}
	if err := validateComputeDAG(computes, computeOutputOwner); err != nil {
		return err
	}
	alarmIDs := map[string]struct{}{}
	conditionIDs := map[string]struct{}{}
	for _, alarm := range artifact.AlarmItems {
		if !canonicalUUID.MatchString(alarm.ID) {
			return fmt.Errorf("alarm id 不是小写 canonical UUID: %q", alarm.ID)
		}
		if _, exists := alarmIDs[alarm.ID]; exists {
			return fmt.Errorf("alarmItems 存在重复 id %q", alarm.ID)
		}
		alarmIDs[alarm.ID] = struct{}{}
		if err := validateAlarm(alarm, points, conditionIDs); err != nil {
			return err
		}
	}
	return nil
}

func hasRole(config EngineConfig, wanted string) bool {
	for _, role := range config.Roles {
		if role == wanted {
			return true
		}
	}
	return false
}

func validateWriteGrant(grant WriteGrant, pointID string) error {
	seen := map[string]struct{}{}
	for _, role := range append(append([]string{}, grant.AllowRoles...), grant.DenyRoles...) {
		if _, exists := seen[role]; exists {
			return fmt.Errorf("dataPoint %q write grant role %q 同时重复/冲突", pointID, role)
		}
		seen[role] = struct{}{}
	}
	return nil
}

func validateCompute(unit ComputeUnit, points map[string]DataPoint) error {
	inputsByAlias := map[string]Input{}
	for _, input := range unit.Inputs {
		if err := validateInput(input, points, "compute "+unit.ID); err != nil {
			return err
		}
		if _, exists := inputsByAlias[input.Alias]; exists {
			return fmt.Errorf("compute %q 的 input alias %q 重复", unit.ID, input.Alias)
		}
		inputsByAlias[input.Alias] = input
	}
	dependencies := map[string]struct{}{}
	for _, dependency := range unit.Dependencies {
		key := dependency.Language + "\x1f" + dependency.PackageName + "\x1f" + dependency.ImportName
		if _, exists := dependencies[key]; exists {
			return fmt.Errorf("compute %q 的 dependency %q/%q 重复", unit.ID, dependency.PackageName, dependency.ImportName)
		}
		if dependency.Language != unit.Language {
			return fmt.Errorf("compute %q 的 dependency %q language 与 compute 不一致", unit.ID, dependency.PackageName)
		}
		dependencies[key] = struct{}{}
	}
	outputKeys, outputPoints := map[string]struct{}{}, map[string]struct{}{}
	for _, output := range unit.Outputs {
		point, ok := points[output.DatapointID]
		if !ok {
			return fmt.Errorf("compute %q 的 output 引用不存在 datapoint %q", unit.ID, output.DatapointID)
		}
		if output.Path != point.Path || output.DataType != point.DataType {
			return fmt.Errorf("compute %q 的 output %q 与目标 datapoint 元数据不一致", unit.ID, output.OutputKey)
		}
		if point.Status != "active" || point.SourceType != "calc.output" || point.SourceID == nil || *point.SourceID != unit.ID {
			return fmt.Errorf("compute %q 的 output %q 必须对应当前 active calc.output datapoint", unit.ID, output.OutputKey)
		}
		if err := validateComputeOutputSource(point.SourceConfig, unit.ID); err != nil {
			return fmt.Errorf("compute %q 的 output %q sourceConfig 不匹配: %w", unit.ID, output.OutputKey, err)
		}
		if !isRuntimeDataType(output.DataType) {
			return fmt.Errorf("compute %q 的 output %q 使用不支持 dataType %q", unit.ID, output.OutputKey, output.DataType)
		}
		if _, exists := outputKeys[output.OutputKey]; exists {
			return fmt.Errorf("compute %q 的 outputKey %q 重复", unit.ID, output.OutputKey)
		}
		if _, exists := outputPoints[output.DatapointID]; exists {
			return fmt.Errorf("compute %q 对 datapoint %q 存在歧义重复输出", unit.ID, output.DatapointID)
		}
		if output.NullPolicy == "default" {
			if len(output.DefaultValue) == 0 || string(output.DefaultValue) == "null" || ValidateDataPointValue(output.DefaultValue, output.DataType, false) != nil {
				return fmt.Errorf("compute %q 的 default nullPolicy 必须提供匹配类型的非空 defaultValue", unit.ID)
			}
		} else if len(output.DefaultValue) > 0 {
			return fmt.Errorf("compute %q 的 output %q 仅 nullPolicy=default 可携带 defaultValue", unit.ID, output.OutputKey)
		}
		outputKeys[output.OutputKey], outputPoints[output.DatapointID] = struct{}{}, struct{}{}
	}
	return validateTrigger(unit, points, inputsByAlias)
}

// calc.output sourceConfig V1 freezes only computeId. outputKey is carried by
// the ComputeUnit output mapping itself; accepting additional source metadata
// would let a point impersonate another producer after deployment.
func validateComputeOutputSource(raw json.RawMessage, computeID string) error {
	var source struct {
		ComputeID string `json:"computeId"`
	}
	if err := decodeStrictJSONObject(raw, &source); err != nil || source.ComputeID != computeID || !canonicalUUID.MatchString(source.ComputeID) {
		return fmt.Errorf("computeId 必须等于当前 compute")
	}
	return nil
}

// validateDataPointSource 让模型验证与 Schema 的 source discriminator 完全一致。
// 它也保护直接调用 ValidateProjectArtifact 的调用方，不依赖 loader 先做 Schema 校验。
func validateDataPointSource(point DataPoint) error {
	switch point.SourceType {
	case "manual.input":
		if point.SourceID != nil {
			return fmt.Errorf("manual.input 的 sourceId 必须为 null")
		}
		if err := decodeStrictJSONObject(point.SourceConfig, &struct{}{}); err != nil {
			return fmt.Errorf("manual.input 的 sourceConfig 必须是严格空对象")
		}
	case "collector.point":
		if point.SourceID == nil || !canonicalUUID.MatchString(*point.SourceID) {
			return fmt.Errorf("collector.point 的 sourceId 必须是 canonical UUID")
		}
		if _, err := collectorPointConnectionID(point.SourceConfig); err != nil {
			return fmt.Errorf("collector.point 的 sourceConfig 无效: %w", err)
		}
	case "calc.output":
		if point.SourceID == nil || !canonicalUUID.MatchString(*point.SourceID) {
			return fmt.Errorf("calc.output 的 sourceId 必须是 canonical UUID")
		}
		if err := validateComputeOutputSource(point.SourceConfig, *point.SourceID); err != nil {
			return err
		}
	case "db.query", "mqtt.subscription":
		if point.SourceID == nil || !canonicalUUID.MatchString(*point.SourceID) {
			return fmt.Errorf("%s 的 sourceId 必须是 canonical UUID", point.SourceType)
		}
		if err := decodeStrictJSONObject(point.SourceConfig, &struct{}{}); err != nil {
			return fmt.Errorf("%s 的 sourceConfig 必须是严格空对象", point.SourceType)
		}
	case "realtime.key":
		if point.SourceID == nil || !canonicalUUID.MatchString(*point.SourceID) {
			return fmt.Errorf("realtime.key 的 sourceId 必须是 canonical UUID")
		}
		if err := validateRealtimeKeySource(point.SourceConfig); err != nil {
			return fmt.Errorf("realtime.key 的 sourceConfig 无效: %w", err)
		}
	default:
		return fmt.Errorf("未知 sourceType %q", point.SourceType)
	}
	return nil
}

func validateRealtimeKeySource(raw json.RawMessage) error {
	var source struct {
		KeyID string `json:"keyId"`
	}
	if err := decodeStrictJSONObject(raw, &source); err != nil || !canonicalUUID.MatchString(source.KeyID) {
		return fmt.Errorf("keyId 必须是 canonical UUID")
	}
	return nil
}

func collectorPointConnectionID(raw json.RawMessage) (string, error) {
	var source struct {
		ConnectionID string `json:"connectionId"`
	}
	if err := decodeStrictJSONObject(raw, &source); err != nil {
		return "", err
	}
	if !canonicalUUID.MatchString(source.ConnectionID) {
		return "", fmt.Errorf("connectionId 必须是 canonical UUID")
	}
	return source.ConnectionID, nil
}

// decodeStrictJSONObject 拒绝 non-object、重复键、未知字段和 trailing JSON。
// SourceConfig 是生产者身份边界，不允许 json.Unmarshal 静默吞掉额外或重复字段。
func decodeStrictJSONObject(raw json.RawMessage, target any) error {
	if !uniqueJSONKeys(raw) {
		return fmt.Errorf("存在重复键或非法 JSON")
	}
	var object map[string]json.RawMessage
	objectDecoder := json.NewDecoder(bytes.NewReader(raw))
	if err := objectDecoder.Decode(&object); err != nil || object == nil {
		return fmt.Errorf("必须是 JSON 对象")
	}
	if err := objectDecoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("存在额外 JSON 内容")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("存在额外 JSON 内容")
	}
	return nil
}

func uniqueJSONKeys(raw json.RawMessage) bool {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var read func() bool
	read = func() bool {
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		delim, isDelim := token.(json.Delim)
		if !isDelim {
			return true
		}
		switch delim {
		case '{':
			seen := map[string]struct{}{}
			for decoder.More() {
				key, err := decoder.Token()
				name, ok := key.(string)
				if err != nil || !ok {
					return false
				}
				if _, exists := seen[name]; exists {
					return false
				}
				seen[name] = struct{}{}
				if !read() {
					return false
				}
			}
			_, err := decoder.Token()
			return err == nil
		case '[':
			for decoder.More() {
				if !read() {
					return false
				}
			}
			_, err := decoder.Token()
			return err == nil
		default:
			return false
		}
	}
	return read() && func() bool { _, err := decoder.Token(); return err == io.EOF }()
}

func validateInput(input Input, points map[string]DataPoint, owner string) error {
	if !canonicalUUID.MatchString(input.DatapointID) {
		return fmt.Errorf("%s 引用的 datapointId 不是小写 canonical UUID: %q", owner, input.DatapointID)
	}
	point, ok := points[input.DatapointID]
	if !ok {
		return fmt.Errorf("%s 引用不存在 datapoint %q", owner, input.DatapointID)
	}
	if point.Path != input.Path || point.DataType != input.DataType {
		return fmt.Errorf("%s 的 datapoint %q path/dataType 与定义不一致", owner, input.DatapointID)
	}
	if !isRuntimeDataType(input.DataType) {
		return fmt.Errorf("%s 的 datapoint %q 使用不支持 dataType %q", owner, input.DatapointID, input.DataType)
	}
	return nil
}

func validateTrigger(unit ComputeUnit, points map[string]DataPoint, inputAliases map[string]Input) error {
	trigger := unit.Trigger
	switch trigger.Kind {
	case "manual":
		return nil
	case "schedule":
		if trigger.Schedule == nil {
			return fmt.Errorf("compute %q 缺少 schedule", unit.ID)
		}
		s := trigger.Schedule
		if s.Kind == "interval" && (s.Every == nil || s.Unit == nil) {
			return fmt.Errorf("compute %q 的 interval schedule 缺少 every/unit", unit.ID)
		}
		if s.Kind != "interval" && (s.Timezone == nil || s.Time == nil) {
			return fmt.Errorf("compute %q 的 %s schedule 缺少 timezone/time", unit.ID, s.Kind)
		}
		if s.Kind == "weekly" && len(s.Weekdays) == 0 {
			return fmt.Errorf("compute %q 的 weekly schedule 缺少 weekdays", unit.ID)
		}
		if (s.Kind == "monthly" || s.Kind == "yearly") && s.DayRule == nil {
			return fmt.Errorf("compute %q 的 %s schedule 缺少 dayRule", unit.ID, s.Kind)
		}
		if s.DayRule != nil && *s.DayRule == "day" && s.DayOfMonth == nil {
			return fmt.Errorf("compute %q 的 dayRule=day 缺少 dayOfMonth", unit.ID)
		}
		if s.DayRule != nil && *s.DayRule == "weekday" && s.WeekOfMonth == nil {
			return fmt.Errorf("compute %q 的 dayRule=weekday 缺少 weekOfMonth", unit.ID)
		}
		if s.DayRule != nil && *s.DayRule == "weekday" && s.Weekday == nil {
			return fmt.Errorf("compute %q 的 dayRule=weekday 缺少 weekday", unit.ID)
		}
		if s.Timezone != nil {
			if _, err := time.LoadLocation(*s.Timezone); err != nil || !strings.Contains(*s.Timezone, "/") {
				return fmt.Errorf("compute %q 的 timezone 必须为有效 IANA 时区", unit.ID)
			}
		}
		if s.StartAt != nil {
			if err := validateUTCTimestamp(*s.StartAt, "compute "+unit.ID+" schedule startAt"); err != nil {
				return err
			}
		}
		if s.EndAt != nil {
			if err := validateUTCTimestamp(*s.EndAt, "compute "+unit.ID+" schedule endAt"); err != nil {
				return err
			}
		}
		if s.StartAt != nil && s.EndAt != nil {
			start, _ := time.Parse(time.RFC3339Nano, *s.StartAt)
			end, _ := time.Parse(time.RFC3339Nano, *s.EndAt)
			if !end.After(start) {
				return fmt.Errorf("compute %q 的 schedule endAt 必须晚于 startAt", unit.ID)
			}
		}
		if s.MaxRuns != nil && (*s.MaxRuns < 1 || *s.MaxRuns > 1_000_000) {
			return fmt.Errorf("compute %q 的 schedule maxRuns 必须为 1-1000000", unit.ID)
		}
		return nil
	case "datapoint_change":
		if err := validateInput(Input{DatapointID: trigger.DatapointID, Path: trigger.Path, DataType: trigger.DataType}, points, "compute "+unit.ID+" trigger"); err != nil {
			return err
		}
		found := false
		for _, input := range unit.Inputs {
			if input.DatapointID == trigger.DatapointID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("compute %q trigger datapoint 必须同时声明为 input", unit.ID)
		}
		if !triggerModeCompatible(trigger.Mode, trigger.DataType) {
			return fmt.Errorf("compute %q 的 datapoint_change mode 与数据点类型不兼容", unit.ID)
		}
		if len(trigger.Deadband) > 0 && (!isNumericType(trigger.DataType) || (trigger.Mode != "value_change" && trigger.Mode != "increase" && trigger.Mode != "decrease")) {
			return fmt.Errorf("compute %q 的 datapoint_change deadband 仅适用于数值 value_change/increase/decrease", unit.ID)
		}
		if len(trigger.Deadband) > 0 {
			if _, ok := ParseNonNegativeNumber(trigger.Deadband); !ok {
				return fmt.Errorf("compute %q 的 datapoint_change deadband 必须为有限非负精确数字", unit.ID)
			}
		}
		return nil
	case "condition":
		variables := map[string]struct{}{}
		for _, variable := range trigger.Variables {
			// true/false are expression literals, so allowing either alias would
			// make a declared datapoint unreachable at evaluation time.
			if variable.Alias == "true" || variable.Alias == "false" {
				return fmt.Errorf("compute %q condition 变量 alias %q 与布尔字面量冲突", unit.ID, variable.Alias)
			}
			if err := validateInput(variable, points, "compute "+unit.ID+" condition trigger"); err != nil {
				return err
			}
			if _, exists := variables[variable.Alias]; exists {
				return fmt.Errorf("compute %q condition 变量 alias %q 重复", unit.ID, variable.Alias)
			}
			variables[variable.Alias] = struct{}{}
			input, exists := inputAliases[variable.Alias]
			if !exists {
				return fmt.Errorf("compute %q condition 变量 %q 未在 inputs 中声明", unit.ID, variable.Alias)
			}
			if input.DatapointID != variable.DatapointID || input.Path != variable.Path || input.DataType != variable.DataType {
				return fmt.Errorf("compute %q condition 变量 %q 必须与同 alias input 精确一致", unit.ID, variable.Alias)
			}
		}
		if len(variables) != len(inputAliases) {
			return fmt.Errorf("compute %q condition variables 必须与 inputs 一一对应", unit.ID)
		}
		return nil
	default:
		return fmt.Errorf("compute %q 未知 trigger kind %q", unit.ID, trigger.Kind)
	}
}

func validateUTCTimestamp(value, label string) error {
	if !strings.HasSuffix(value, "Z") {
		return fmt.Errorf("%s 必须为 UTC RFC3339 Z 时间", label)
	}
	if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		return fmt.Errorf("%s 不是有效 UTC RFC3339 时间: %w", label, err)
	}
	return nil
}

func isNumericType(dataType string) bool {
	switch strings.ToLower(dataType) {
	case "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float32", "float64", "decimal":
		return true
	default:
		return false
	}
}

func isRuntimeDataType(dataType string) bool {
	switch dataType {
	case "bool", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float32", "float64", "decimal", "string", "bytes", "datetime", "object", "array":
		return true
	default:
		return false
	}
}

func isScalarType(dataType string) bool {
	return dataType == "bool" || dataType == "string" || isNumericType(dataType)
}

func triggerModeCompatible(mode, dataType string) bool {
	switch mode {
	case "any", "value_change":
		return isScalarType(dataType)
	case "increase", "decrease":
		return isNumericType(dataType)
	case "rising_edge", "falling_edge":
		return dataType == "bool"
	default:
		return false
	}
}

func validateComputeDAG(computes map[string]ComputeUnit, outputOwner map[string]string) error {
	edges := map[string][]string{}
	for id, unit := range computes {
		for _, input := range unit.Inputs {
			if owner, ok := outputOwner[input.DatapointID]; ok && owner != id {
				edges[owner] = append(edges[owner], id)
			}
		}
	}
	state := map[string]int{}
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("compute DAG 存在环，涉及 compute %q", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, next := range edges[id] {
			if err := visit(next); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range computes {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func validateAlarm(alarm AlarmItem, points map[string]DataPoint, conditionIDs map[string]struct{}) error {
	aliases := map[string]struct{}{}
	for _, input := range alarm.Inputs {
		if err := validateInput(input, points, "alarm "+alarm.ID); err != nil {
			return err
		}
		if _, exists := aliases[input.Alias]; exists {
			return fmt.Errorf("alarm %q 的 input alias %q 重复", alarm.ID, input.Alias)
		}
		aliases[input.Alias] = struct{}{}
	}
	if alarm.Mode == "point" && len(alarm.Inputs) != 1 {
		return fmt.Errorf("point alarm %q 必须恰有一个 input", alarm.ID)
	}
	if alarm.Mode == "derived" && (len(alarm.Inputs) < 2 || alarm.DerivedExpression == nil || alarm.EvaluationMode != "single" || len(alarm.Conditions) != 1) {
		return fmt.Errorf("derived alarm %q 的 inputs/expression/evaluationMode/conditions 组合无效", alarm.ID)
	}
	if alarm.EvaluationMode == "single" && len(alarm.Conditions) != 1 {
		return fmt.Errorf("single alarm %q 必须恰有一个 condition", alarm.ID)
	}
	if alarm.EvaluationMode == "highest_matching" && alarm.Mode != "point" {
		return fmt.Errorf("highest_matching 仅适用于 point alarm %q", alarm.ID)
	}
	pointDataType := ""
	if alarm.Mode == "point" {
		pointDataType = alarm.Inputs[0].DataType
	}
	for _, condition := range alarm.Conditions {
		if !canonicalUUID.MatchString(condition.ID) {
			return fmt.Errorf("alarm condition id 不是小写 canonical UUID: %q", condition.ID)
		}
		if _, exists := conditionIDs[condition.ID]; exists {
			return fmt.Errorf("alarm conditions 存在重复 id %q", condition.ID)
		}
		conditionIDs[condition.ID] = struct{}{}
		if alarm.EvaluationMode == "highest_matching" && condition.Kind != "threshold" {
			return fmt.Errorf("highest_matching alarm %q 仅允许 threshold condition", alarm.ID)
		}
		if alarm.Mode == "derived" && (condition.Kind == "offline" || condition.Kind == "quality" || condition.Kind == "stale") {
			return fmt.Errorf("derived alarm %q 禁止 %s condition", alarm.ID, condition.Kind)
		}
		if err := validateCondition(condition); err != nil {
			return fmt.Errorf("alarm %q: %w", alarm.ID, err)
		}
		if pointDataType != "" {
			if err := validatePointAlarmConditionType(pointDataType, condition); err != nil {
				return fmt.Errorf("alarm %q: %w", alarm.ID, err)
			}
		}
	}
	return nil
}

// Point alarms have a declared wire type, so parameter incompatibilities are
// rejected at load time. Derived expressions intentionally have no declared
// output type; their result remains a runtime fail-closed check.
func validatePointAlarmConditionType(dataType string, condition AlarmCondition) error {
	numeric := isNumericType(dataType)
	switch condition.Kind {
	case "threshold", "range", "rate_of_change", "deviation":
		if !numeric {
			return fmt.Errorf("%s condition 仅支持数值 datapoint", condition.Kind)
		}
	case "text_match":
		if dataType != "string" {
			return fmt.Errorf("text_match condition 仅支持 string datapoint")
		}
	case "state":
		value, ok := strictConditionObject(condition.Params)
		if !ok || len(value) != 1 || value["expected"] == nil || ValidateDataPointValue(value["expected"], dataType, false) != nil {
			return fmt.Errorf("state expected 必须匹配 datapoint dataType")
		}
	case "transition":
		switch condition.Operator {
		case "changed":
			// Any Runtime V1 value can participate in a change transition.
		case "rising", "falling":
			if dataType != "bool" {
				return fmt.Errorf("transition %s 仅支持 bool datapoint", condition.Operator)
			}
		case "from_to":
			value, ok := strictConditionObject(condition.Params)
			if !ok || len(value) != 2 || value["from"] == nil || value["to"] == nil || ValidateDataPointValue(value["from"], dataType, false) != nil || ValidateDataPointValue(value["to"], dataType, false) != nil {
				return fmt.Errorf("transition from_to 必须匹配 datapoint dataType")
			}
		}
	}
	return nil
}

func validateCondition(condition AlarmCondition) error {
	if _, ok := ParseNonNegativeNumber(condition.Deadband); !ok {
		return fmt.Errorf("alarm deadband 必须为有限非负精确数字")
	}
	if condition.Kind == "range" {
		value, ok := strictConditionObject(condition.Params)
		if !ok || len(value) != 2 {
			return fmt.Errorf("range params 无法解析")
		}
		lower, lowerOK := value["lower"]
		upper, upperOK := value["upper"]
		lowerValue, lowerNumber := ParseNonNegativeOrSignedNumber(lower)
		upperValue, upperNumber := ParseNonNegativeOrSignedNumber(upper)
		if !lowerOK || !upperOK || !lowerNumber || !upperNumber || lowerValue.Cmp(upperValue) >= 0 {
			return fmt.Errorf("range params 必须 lower < upper")
		}
	}
	if condition.Kind == "transition" && condition.Operator == "from_to" {
		value, ok := strictConditionObject(condition.Params)
		if !ok || len(value) != 2 {
			return fmt.Errorf("transition params 无法解析")
		}
		from, fromOK := value["from"]
		to, toOK := value["to"]
		if !fromOK || !toOK || !strictConditionValue(from) || !strictConditionValue(to) {
			return fmt.Errorf("transition params 无法解析")
		}
		if conditionValueEqual(from, to) {
			return fmt.Errorf("transition from_to 的 from 与 to 不得相等")
		}
	}
	if condition.Kind == "transition" && condition.Operator != "from_to" {
		var value map[string]json.RawMessage
		if err := json.Unmarshal(condition.Params, &value); err != nil {
			return fmt.Errorf("transition params 无法解析: %w", err)
		}
		if len(value) != 0 {
			return fmt.Errorf("transition %s 的 params 必须为空对象", condition.Operator)
		}
	}
	if condition.Kind == "deviation" && condition.Operator != "gt" {
		return fmt.Errorf("deviation operator 必须为 gt")
	}
	return nil
}

func strictConditionObject(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	var value map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if decoder.Decode(&value) != nil || decoder.Decode(&struct{}{}) != io.EOF || value == nil {
		return nil, false
	}
	return value, true
}

func strictConditionValue(raw json.RawMessage) bool {
	_, ok := strictJSONValue(raw)
	return ok
}

// conditionValueEqual keeps JSON numbers as exact rationals.  This avoids
// merging adjacent uint64 values or high-precision decimals through float64,
// while non-numeric jsonValue values retain structural equality semantics.
func conditionValueEqual(left, right json.RawMessage) bool {
	leftNumber, leftIsNumber := ParseNonNegativeOrSignedNumber(left)
	rightNumber, rightIsNumber := ParseNonNegativeOrSignedNumber(right)
	if leftIsNumber && rightIsNumber {
		return leftNumber.Cmp(rightNumber) == 0
	}
	leftValue, leftOK := strictJSONValue(left)
	rightValue, rightOK := strictJSONValue(right)
	if !leftOK || !rightOK {
		return false
	}
	leftRaw, leftErr := json.Marshal(leftValue)
	rightRaw, rightErr := json.Marshal(rightValue)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftRaw, rightRaw)
}

func computeIDs(config EngineConfig) map[string]struct{} {
	result := map[string]struct{}{}
	for _, producer := range config.ProducerAssignments {
		if producer.ProducerType == "compute" {
			result[producer.ComputeID] = struct{}{}
		}
	}
	return result
}

func sameKeys(left, right map[string]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for key := range left {
		if _, ok := right[key]; !ok {
			return false
		}
	}
	return true
}

// SortedComputeIDs 提供稳定诊断/测试输出，不参与运行逻辑。
func SortedComputeIDs(artifact ProjectArtifact) []string {
	ids := make([]string, 0, len(artifact.ComputeUnits))
	for _, unit := range artifact.ComputeUnits {
		ids = append(ids, unit.ID)
	}
	sort.Strings(ids)
	return ids
}
