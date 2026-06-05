# Kafka 工作台输出模式调整实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 Kafka 工作台调整为“消费规则树 + 两种数据点输出模式”，整包模式点击规则后手动拉取消息样本，字段模式点击规则后配置字段到数据点的映射，并支持样本/JSON 辅助解析。

**架构：** Kafka 接入源只描述 Broker 连接；Kafka 消费规则描述 Topic、消费组、读取参数和输出模式。`raw_message` 整包模式在规则上同步一个 `kafka.raw` 数据点；`field_mapping` 字段模式继续复用现有 `data_kafka_fields` 字段映射和 `kafka.field` 数据点。工作台不维护 MQTT 式连接状态，所有样本读取都是用户手动触发的临时 Kafka preview，不使用运行态消费组提交进度。

**技术栈：** Vue 3 + Element Plus + TypeScript + zod；Go data_service；PostgreSQL migrations；segmentio/kafka-go；现有 `WorkbenchStreamToolbar` / `WorkbenchStreamMessageList`。

---

## 设计边界

- 左侧树节点统一称为“消费规则”，不再叫 Topic 映射或 Topic 订阅。
- 新建消费规则时选择输出模式：
  - `raw_message`：整包数据点，一个消费规则同步一个数据点。
  - `field_mapping`：字段数据点，一个消费规则下维护多条字段到数据点映射。
- 整包模式右侧面板借鉴 MQTT 查看消息界面，但只提供手动拉取，不自动监听。
- 字段模式右侧面板保留表格，工具条增加“粘贴样本”“拉取样本”“测试映射”；测试映射时临时拉取 1 条消息并更新表格预览值。
- 工作台顶部 `连接/断开` 去掉或改为 `测试 Broker`，不作为进入面板或拉取样本的前置条件。
- 第一阶段只完成数据中心配置与开发态预览；节点侧长期消费执行器不在本计划实现，但保存的 `source_config` 必须能表达运行态需要的配置。

## 文件结构

- 修改 `data_service/internal/db/migrations/0030_kafka_workbench.sql`
  - 新库 Kafka 消费规则表增加输出模式字段。
  - 将 Topic 唯一约束调整为规则名称唯一，允许同一 Topic 被多条规则复用。
- 创建 `data_service/internal/db/migrations/0050_kafka_output_mode.sql`
  - 既有库补充 `output_mode`、`raw_output_scope`。
  - 调整唯一约束。
- 创建 `data_service/internal/db/migrations/0050_kafka_output_mode_down.sql`
  - 回滚输出模式字段和唯一约束。
- 修改 `data_service/internal/repository/kafka_workbench_repository.go`
  - Topic mapping record/params/scan 增加输出模式与整包数据点投影。
  - 创建/更新消费规则时同步整包模式数据点。
  - 删除消费规则时将整包数据点标记失效。
- 修改 `data_service/internal/service/kafka_workbench_service.go`
  - 输入/输出模型增加 `outputMode`、`rawOutputScope`、`rawDataPointId`、`rawDataPointPath`。
  - 增加输出模式归一化、整包数据点 source_config 生成。
  - 预览逻辑保持临时 reader，不依赖连接状态。
- 修改 `data_service/internal/service/kafka_workbench_service_test.go`
  - 覆盖输出模式归一化、字段模式不生成整包数据点配置、整包模式默认值。
- 修改 `datacenter/src/api/schemas/kafka-workbench.schema.ts`
  - schema 增加输出模式与整包数据点字段。
- 修改 `datacenter/src/components/kafka/types.ts`
  - 复用 schema 类型，无需新增复杂类型；补充本地样本来源类型。
- 修改 `datacenter/src/components/kafka/KafkaTopicMappingDialog.vue`
  - 文案改为“新建/编辑消费规则”。
  - 增加输出模式选择。
  - 整包模式显示目标数据点路径和输出内容。
  - 字段模式提示保存后进入字段配置。
- 修改 `datacenter/src/components/kafka/KafkaWorkbench.vue`
  - 左侧树文案调整。
  - 去掉连接状态对预览/字段面板的阻塞。
  - 点击规则后按 `outputMode` 展示整包输出面板或字段映射面板。
  - 右键菜单文案和动作调整。
- 创建 `datacenter/src/components/kafka/KafkaRawOutputPanel.vue`
  - 整包模式右侧面板，负责规则摘要、输出数据点、手动拉取样本和消息列表。
- 修改 `datacenter/src/components/kafka/KafkaPreviewPanel.vue`
  - 如保留作为整包样本列表底层组件，移除 `connected` prop；否则由 `KafkaRawOutputPanel` 替代后删除引用。
- 修改 `datacenter/src/components/kafka/KafkaFieldMappingPanel.vue`
  - 文案从“变量管理/变量预览”改为“字段映射/测试映射”。
  - 移除 `connected` prop。
  - 增加粘贴样本 JSON 弹窗与样本候选解析。
- 修改 `datacenter/src/components/kafka/KafkaTopicTreeBranch.vue`
  - 左侧树节点文案和 tooltip 从 Topic 映射改为消费规则。
- 修改 `datacenter/src/components/kafka/KafkaInspectorPanel.vue`
  - 展示当前规则输出模式、输出数据点、样本摘要。
- 可选修改 `datacenter/src/components/kafka/kafkaTopicTreeModel.ts`
  - 搜索匹配增加输出模式/消费组。

## 任务 1：后端消费规则模型增加输出模式

**文件：**
- 修改：`data_service/internal/db/migrations/0030_kafka_workbench.sql`
- 创建：`data_service/internal/db/migrations/0050_kafka_output_mode.sql`
- 创建：`data_service/internal/db/migrations/0050_kafka_output_mode_down.sql`
- 修改：`data_service/internal/repository/kafka_workbench_repository.go`
- 修改：`data_service/internal/service/kafka_workbench_service.go`
- 测试：`data_service/internal/service/kafka_workbench_service_test.go`

- [ ] **步骤 1：编写服务归一化测试**

在 `data_service/internal/service/kafka_workbench_service_test.go` 追加：

```go
func TestNormalizeKafkaOutputModeDefaultsToFieldMapping(t *testing.T) {
	mode, scope, err := normalizeKafkaOutputConfig("", "")
	if err != nil {
		t.Fatalf("expected default output config: %v", err)
	}
	if mode != "field_mapping" || scope != "value" {
		t.Fatalf("unexpected output config: %s %s", mode, scope)
	}
}

func TestNormalizeKafkaOutputModeAcceptsRawMessage(t *testing.T) {
	mode, scope, err := normalizeKafkaOutputConfig("raw_message", "full_message")
	if err != nil {
		t.Fatalf("expected raw output config: %v", err)
	}
	if mode != "raw_message" || scope != "full_message" {
		t.Fatalf("unexpected raw output config: %s %s", mode, scope)
	}
}

func TestNormalizeKafkaOutputModeRejectsUnknownMode(t *testing.T) {
	if _, _, err := normalizeKafkaOutputConfig("topic_value", "value"); err == nil {
		t.Fatal("expected unknown output mode to fail")
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

工作目录：`data_service`

```powershell
go test ./internal/service -run "TestNormalizeKafkaOutputMode" -count=1
```

预期：FAIL，提示 `normalizeKafkaOutputConfig` 未定义。

- [ ] **步骤 3：编写迁移**

修改 `0030_kafka_workbench.sql` 的 `data_kafka_topic_mappings` 表，新增字段：

```sql
    output_mode text NOT NULL DEFAULT 'field_mapping' CHECK (output_mode IN ('raw_message', 'field_mapping')),
    raw_output_scope text NOT NULL DEFAULT 'value' CHECK (raw_output_scope IN ('value', 'full_message')),
```

将原约束：

```sql
CONSTRAINT data_kafka_topic_mappings_topic_key
    UNIQUE (project_id, connection_id, topic)
```

替换为：

```sql
CONSTRAINT data_kafka_topic_mappings_name_key
    UNIQUE (project_id, connection_id, name)
```

创建 `0050_kafka_output_mode.sql`：

```sql
ALTER TABLE data_kafka_topic_mappings
    ADD COLUMN IF NOT EXISTS output_mode text NOT NULL DEFAULT 'field_mapping' CHECK (output_mode IN ('raw_message', 'field_mapping')),
    ADD COLUMN IF NOT EXISTS raw_output_scope text NOT NULL DEFAULT 'value' CHECK (raw_output_scope IN ('value', 'full_message'));

ALTER TABLE data_kafka_topic_mappings
    DROP CONSTRAINT IF EXISTS data_kafka_topic_mappings_topic_key,
    DROP CONSTRAINT IF EXISTS data_kafka_topic_mappings_name_key,
    ADD CONSTRAINT data_kafka_topic_mappings_name_key
        UNIQUE (project_id, connection_id, name);
```

创建 `0050_kafka_output_mode_down.sql`：

```sql
ALTER TABLE data_kafka_topic_mappings
    DROP CONSTRAINT IF EXISTS data_kafka_topic_mappings_name_key,
    ADD CONSTRAINT data_kafka_topic_mappings_topic_key
        UNIQUE (project_id, connection_id, topic);

ALTER TABLE data_kafka_topic_mappings
    DROP COLUMN IF EXISTS raw_output_scope,
    DROP COLUMN IF EXISTS output_mode;
```

- [ ] **步骤 4：扩展 service 类型**

在 `KafkaTopicMapping` 增加：

```go
OutputMode       string `json:"outputMode"`
RawOutputScope   string `json:"rawOutputScope"`
RawDataPointID   string `json:"rawDataPointId"`
RawDataPointPath string `json:"rawDataPointPath"`
```

在 `CreateKafkaTopicMappingInput` 和 `UpdateKafkaTopicMappingInput` 增加：

```go
OutputMode       string `json:"outputMode"`
RawOutputScope   string `json:"rawOutputScope"`
RawDataPointPath string `json:"rawDataPointPath"`
```

增加常量集合：

```go
var (
	allowedKafkaOutputModes     = map[string]struct{}{"raw_message": {}, "field_mapping": {}}
	allowedKafkaRawOutputScopes = map[string]struct{}{"value": {}, "full_message": {}}
)
```

增加归一化 helper：

```go
func normalizeKafkaOutputConfig(mode, scope string) (string, string, error) {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode == "" {
		normalizedMode = "field_mapping"
	}
	if _, ok := allowedKafkaOutputModes[normalizedMode]; !ok {
		return "", "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka 输出模式不合法")
	}
	normalizedScope := strings.ToLower(strings.TrimSpace(scope))
	if normalizedScope == "" {
		normalizedScope = "value"
	}
	if normalizedMode != "raw_message" {
		return normalizedMode, "value", nil
	}
	if _, ok := allowedKafkaRawOutputScopes[normalizedScope]; !ok {
		return "", "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka 整包输出内容不合法")
	}
	return normalizedMode, normalizedScope, nil
}
```

- [ ] **步骤 5：扩展 repository 类型与 SQL**

在 `KafkaTopicMappingRecord`、`CreateKafkaTopicMappingParams`、`UpdateKafkaTopicMappingParams` 增加：

```go
OutputMode       string
RawOutputScope   string
RawDataPointID   *string
RawDataPointPath *string
RawDataPointPathInput string
RawSourceConfig  map[string]any
```

注意：`RawDataPointPathInput` 和 `RawSourceConfig` 只用于写入，不需要出现在 record scan 中。

修改 `ListTopicMappings` 和 `GetTopicMapping` 的 SELECT，增加 data_points 左连接：

```sql
LEFT JOIN data_points dp
  ON dp.project_id = m.project_id
 AND dp.source_type = 'kafka.raw'
 AND dp.source_config->>'topicMappingId' = m.id::text
```

SELECT 字段中在 `raw_output_scope` 后增加：

```sql
dp.id AS raw_data_point_id,
dp.path AS raw_data_point_path,
```

修改 insert/update `RETURNING` 后通过同一个 scan 顺序返回：

```sql
output_mode, raw_output_scope
```

`scanKafkaTopicMappingRecord` 对应增加：

```go
&record.OutputMode,
&record.RawOutputScope,
&record.RawDataPointID,
&record.RawDataPointPath,
```

- [ ] **步骤 6：同步整包模式数据点**

在 repository 增加事务化方法，替代原 `CreateTopicMapping` / `UpdateTopicMapping` 在 service 中的调用：

```go
func (r *KafkaWorkbenchRepository) CreateTopicMappingWithDataPoint(ctx context.Context, params CreateKafkaTopicMappingParams) (*KafkaTopicMappingRecord, error)
func (r *KafkaWorkbenchRepository) UpdateTopicMappingWithDataPoint(ctx context.Context, params UpdateKafkaTopicMappingParams) (*KafkaTopicMappingRecord, error)
```

实现要求：
- 先创建/更新 `data_kafka_topic_mappings`。
- `OutputMode == "raw_message"` 时调用 `upsertKafkaRawDataPoint`。
- `OutputMode != "raw_message"` 时将同 mapping 的 `kafka.raw` 数据点标记为 `invalid`。
- 返回 record 时带上 `RawDataPointID` 和 `RawDataPointPath`。

新增 helper：

```go
func upsertKafkaRawDataPoint(ctx context.Context, tx pgx.Tx, mapping KafkaTopicMappingRecord, path string, sourceConfig map[string]any, userID string) (string, string, error) {
	sourceConfigWithMapping := cloneProtocolMap(sourceConfig)
	sourceConfigWithMapping["topicMappingId"] = mapping.ID
	sourceConfigPayload, err := json.Marshal(sourceConfigWithMapping)
	if err != nil {
		return "", "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka 整包数据点 sourceConfig 格式无效", err)
	}
	allocatedPath, err := allocateGeneratedDataPointPath(ctx, tx, mapping.ProjectID, path, "kafka.raw", mapping.ID)
	if err != nil {
		return "", "", err
	}
	dataType := "object"
	if mapping.Decode != "json" {
		dataType = "string"
	}
	var dataPointID, dataPointPath string
	err = tx.QueryRow(ctx, `
		UPDATE data_points
		SET path = $2,
		    name = $3,
		    source_id = $4,
		    source_config = $5::jsonb,
		    data_type = $6,
		    refresh_mode = 'subscription',
		    status = 'active',
		    updated_by = $7,
		    updated_at = now()
		WHERE project_id = $1
		  AND source_type = 'kafka.raw'
		  AND source_config->>'topicMappingId' = $8
		RETURNING id, path
	`, mapping.ProjectID, path, mapping.Name, mapping.ConnectionID, string(sourceConfigPayload), dataType, userID, mapping.ID).Scan(&dataPointID, &dataPointPath)
	if err == nil {
		return dataPointID, dataPointPath, nil
	}
	if err != pgx.ErrNoRows {
		return "", "", apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步 Kafka 整包数据点失败", err)
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO data_points (
			project_id, path, name, source_type, source_id, source_config,
			data_type, refresh_mode, status, display_order, created_by, updated_by
		)
		VALUES (
			$1, $2, $3, 'kafka.raw', $4, $5::jsonb, $6, 'subscription', 'active',
			COALESCE((SELECT MAX(display_order) + 1 FROM data_points WHERE project_id = $1), 0),
			$7, $7
		)
		RETURNING id, path
	`, mapping.ProjectID, allocatedPath, mapping.Name, mapping.ConnectionID, string(sourceConfigPayload), dataType, userID).Scan(&dataPointID, &dataPointPath)
	if err != nil {
		return "", "", apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步 Kafka 整包数据点失败", err)
	}
	return dataPointID, dataPointPath, nil
}
```

- [ ] **步骤 7：service 归一化写入输出模式**

在 `normalizeCreateTopicMapping` 和 `normalizeUpdateTopicMapping` 中调用：

```go
outputMode, rawScope, err := normalizeKafkaOutputConfig(input.OutputMode, input.RawOutputScope)
if err != nil {
	return repository.CreateKafkaTopicMappingParams{}, err
}
```

创建参数中增加：

```go
OutputMode:       outputMode,
RawOutputScope:   rawScope,
RawDataPointPathInput: buildKafkaRawDataPointPath(name),
RawSourceConfig:  kafkaRawSourceConfig(connectionID, topic, consumerGroup, partitionMode, partition, startPosition, runtimeStart.StartOffset, decode, rawScope),
```

更新参数中 `RawDataPointPathInput` 使用：

```go
rawPath := strings.TrimSpace(input.RawDataPointPath)
if rawPath == "" && current.RawDataPointPath != nil {
	rawPath = *current.RawDataPointPath
}
if rawPath == "" {
	rawPath = buildKafkaRawDataPointPath(name)
}
```

新增 helper：

```go
func kafkaRawSourceConfig(connectionID, topic, consumerGroup, partitionMode string, partition *int, startPosition string, startOffset *int64, decode, scope string) map[string]any {
	return map[string]any{
		"connectionId":   connectionID,
		"topic":          topic,
		"consumerGroup":  strings.TrimSpace(consumerGroup),
		"partitionMode":  partitionMode,
		"partition":      partition,
		"startPosition":  startPosition,
		"startOffset":    startOffset,
		"decode":         decode,
		"rawOutputScope": scope,
	}
}

func buildKafkaRawDataPointPath(mappingName string) string {
	return "kafka." + normalizeDatapointSegment(mappingName) + ".message"
}
```

- [ ] **步骤 8：替换 service 调用**

在 `CreateTopicMapping` 中把：

```go
record, err := s.repository.CreateTopicMapping(ctx, params)
```

替换为：

```go
record, err := s.repository.CreateTopicMappingWithDataPoint(ctx, params)
```

在 `UpdateTopicMapping` 中把：

```go
record, err := s.repository.UpdateTopicMapping(ctx, params)
```

替换为：

```go
record, err := s.repository.UpdateTopicMappingWithDataPoint(ctx, params)
```

在 `toKafkaTopicMapping` 中补充：

```go
OutputMode:       record.OutputMode,
RawOutputScope:   record.RawOutputScope,
RawDataPointID:   kafkaStringValue(record.RawDataPointID),
RawDataPointPath: kafkaStringValue(record.RawDataPointPath),
```

- [ ] **步骤 9：删除消费规则时标记整包数据点失效**

在 `DeleteTopicMapping` 对应 repository 方法中，删除 mapping 前增加：

```sql
UPDATE data_points
SET status = 'invalid', updated_at = now()
WHERE project_id = $1
  AND source_type = 'kafka.raw'
  AND source_config->>'topicMappingId' = $2
```

字段模式现有 `data_kafka_fields` CASCADE 和 `kafka.field` 数据点失效逻辑如未覆盖，应在本步骤检查；若删除 Topic mapping 未标记字段数据点，补充同类更新：

```sql
UPDATE data_points
SET status = 'invalid', updated_at = now()
WHERE project_id = $1
  AND source_type = 'kafka.field'
  AND source_config->>'topicMappingId' = $2
```

- [ ] **步骤 10：运行后端测试**

工作目录：`data_service`

```powershell
gofmt -w internal/repository/kafka_workbench_repository.go internal/service/kafka_workbench_service.go internal/service/kafka_workbench_service_test.go
go test ./internal/service -run "TestNormalizeKafkaOutputMode|TestNormalizeKafkaTopicMapping" -count=1
go test ./internal/service ./internal/repository ./internal/http/handler
```

预期：全部 PASS。

- [ ] **步骤 11：Commit**

```powershell
git status --short
git add data_service/internal/db/migrations/0030_kafka_workbench.sql data_service/internal/db/migrations/0050_kafka_output_mode.sql data_service/internal/db/migrations/0050_kafka_output_mode_down.sql data_service/internal/repository/kafka_workbench_repository.go data_service/internal/service/kafka_workbench_service.go data_service/internal/service/kafka_workbench_service_test.go
git commit -m "feat(kafka): 增加消费规则输出模式"
```

提交前确认没有混入非 Kafka 输出模式相关文件。

## 任务 2：前端 schema 与新建消费规则弹窗

**文件：**
- 修改：`datacenter/src/api/schemas/kafka-workbench.schema.ts`
- 修改：`datacenter/src/components/kafka/types.ts`
- 修改：`datacenter/src/components/kafka/KafkaTopicMappingDialog.vue`

- [ ] **步骤 1：扩展 zod schema**

在 `KafkaTopicMappingSchema` 增加：

```ts
outputMode: z.enum(['raw_message', 'field_mapping']).default('field_mapping'),
rawOutputScope: z.enum(['value', 'full_message']).default('value'),
rawDataPointId: IdSchema.optional().or(z.literal('')),
rawDataPointPath: z.string().default('').optional(),
```

- [ ] **步骤 2：表单模型增加输出模式**

在 `KafkaTopicMappingDialog.vue` 的 `form` 增加：

```ts
outputMode: 'raw_message',
rawOutputScope: 'value',
rawDataPointPath: '',
```

`resetForm` 中增加：

```ts
form.outputMode = props.mapping?.outputMode || 'raw_message'
form.rawOutputScope = props.mapping?.rawOutputScope || 'value'
form.rawDataPointPath = props.mapping?.rawDataPointPath || buildDefaultRawPath(form.name)
```

新增 helper：

```ts
const normalizePathSegment = (value: string) =>
  value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, '_')
    .replace(/^_+|_+$/g, '')

const buildDefaultRawPath = (name: string) => `kafka.${normalizePathSegment(name) || 'message'}.message`
```

- [ ] **步骤 3：弹窗文案改为消费规则**

将标题改为：

```vue
:title="mode === 'create' ? '新建消费规则' : '编辑消费规则'"
```

将“基础信息”中的名称 tooltip 改为：

```vue
<el-tooltip content="消费规则会出现在左侧树中，建议使用业务流名称。" placement="top">
```

将保存成功消息调用方统一改为 `消费规则已保存`，本步骤只调整弹窗内部文案。

- [ ] **步骤 4：增加输出模式选择**

在基础信息 section 顶部加入：

```vue
<el-form-item>
  <template #label>
    <span class="kafka-topic-dialog__field-label">
      输出模式
      <el-tooltip content="整包数据点会把最新消息写入一个数据点；字段数据点会把消息字段拆成多个数据点。" placement="top">
        <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
      </el-tooltip>
    </span>
  </template>
  <el-segmented v-model="form.outputMode" :options="outputModeOptions" />
</el-form-item>
```

script 中增加：

```ts
const outputModeOptions = [
  { label: '整包数据点', value: 'raw_message' },
  { label: '字段数据点', value: 'field_mapping' },
]
```

- [ ] **步骤 5：整包模式显示输出配置**

新增 section：

```vue
<div v-if="form.outputMode === 'raw_message'" class="kafka-topic-dialog__section">
  <div class="kafka-topic-dialog__section-title">数据点输出</div>
  <el-form-item>
    <template #label>
      <span class="kafka-topic-dialog__field-label">
        输出内容
        <el-tooltip content="消息体只写入 payload/value；完整消息会包含 key、headers、partition、offset 和 payload。" placement="top">
          <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
        </el-tooltip>
      </span>
    </template>
    <el-select v-model="form.rawOutputScope">
      <el-option label="消息体 payload" value="value" />
      <el-option label="完整消息" value="full_message" />
    </el-select>
  </el-form-item>
  <el-form-item>
    <template #label>
      <span class="kafka-topic-dialog__field-label">
        目标数据点路径
        <el-tooltip content="留空时按规则名称自动生成；保存后设计中心可查询或订阅这个数据点。" placement="top">
          <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
        </el-tooltip>
      </span>
    </template>
    <el-input v-model="form.rawDataPointPath" clearable placeholder="kafka.device_telemetry.message" />
  </el-form-item>
</div>
```

字段模式显示提示：

```vue
<div v-else class="kafka-topic-dialog__mode-note">
  保存后进入字段映射面板，可粘贴 JSON 样本或临时拉取样本生成数据点映射。
</div>
```

- [ ] **步骤 6：提交 payload**

`submit` payload 增加：

```ts
outputMode: form.outputMode,
rawOutputScope: form.outputMode === 'raw_message' ? form.rawOutputScope : 'value',
rawDataPointPath:
  form.outputMode === 'raw_message'
    ? (form.rawDataPointPath.trim() || buildDefaultRawPath(form.name))
    : '',
```

- [ ] **步骤 7：运行前端类型检查**

工作目录：仓库根目录

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 8：Commit**

```powershell
git status --short
git add datacenter/src/api/schemas/kafka-workbench.schema.ts datacenter/src/components/kafka/types.ts datacenter/src/components/kafka/KafkaTopicMappingDialog.vue
git commit -m "feat(kafka): 支持消费规则输出模式表单"
```

## 任务 3：整包数据点右侧面板

**文件：**
- 创建：`datacenter/src/components/kafka/KafkaRawOutputPanel.vue`
- 修改：`datacenter/src/components/kafka/KafkaWorkbench.vue`
- 修改：`datacenter/src/components/kafka/KafkaInspectorPanel.vue`

- [ ] **步骤 1：创建整包输出面板**

创建 `KafkaRawOutputPanel.vue`，核心结构：

```vue
<template>
  <section class="kafka-raw-output-panel">
    <header class="kafka-raw-output-panel__summary">
      <div>
        <strong>{{ mapping.name || mapping.topic }}</strong>
        <span>{{ mapping.topic }}</span>
      </div>
      <div class="kafka-raw-output-panel__chips">
        <WorkbenchStatusPill label="整包数据点" tone="info" />
        <WorkbenchStatusPill :label="mapping.rawDataPointPath || '未生成数据点'" :tone="mapping.rawDataPointPath ? 'success' : 'neutral'" />
      </div>
    </header>

    <WorkbenchStreamToolbar
      :title="'样本测试'"
      :subtitle="toolbarSubtitle"
      :loading="loading"
      :status-label="statusLabel"
      :status-tone="statusTone"
      :search="search"
      :limit="displayLimit"
      :format-json="formatJson"
      :auto-scroll="false"
      :show-timestamp="showTimestamp"
      @update:search="search = $event"
      @update:limit="displayLimit = $event"
      @update:format-json="formatJson = $event"
      @update:show-timestamp="showTimestamp = $event"
      @refresh="pullSamples"
      @clear="clearSamples"
    />
    <WorkbenchStreamMessageList
      :messages="displayMessages"
      :loading="loading"
      :format-json="formatJson"
      :show-timestamp="showTimestamp"
      empty-text="暂无 Kafka 样本"
      empty-hint="点击拉取样本执行一次临时读取"
      @copy="copyMessage"
    />
  </section>
</template>
```

script 使用 `dataAPI.previewKafkaTopicMapping`：

```ts
const pullSamples = async () => {
  loading.value = true
  lastError.value = ''
  try {
    const preview = await dataAPI.previewKafkaTopicMapping(props.projectId, props.mapping.id, {
      limit: displayLimit.value,
      timeoutMs: props.mapping.timeoutMs,
      decode: props.mapping.decode,
    })
    samples.value = preview.samples || []
    emit('samples', { mappingId: String(props.mapping.id), samples: samples.value, preview })
  } catch (error) {
    lastError.value = getApiErrorMessage(error, 'Kafka 样本拉取失败')
    ElMessage.error(lastError.value)
  } finally {
    loading.value = false
  }
}
```

- [ ] **步骤 2：Workbench 按输出模式展示面板**

在 `KafkaWorkbench.vue` import：

```ts
import KafkaRawOutputPanel from './KafkaRawOutputPanel.vue'
```

将 content 区域改为：

```vue
<KafkaRawOutputPanel
  v-if="activeMapping?.outputMode === 'raw_message'"
  :project-id="projectId"
  :mapping="activeMapping"
  @samples="handlePreviewSamples"
/>
<KafkaFieldMappingPanel
  v-else-if="activeMapping?.outputMode === 'field_mapping'"
  :project-id="projectId"
  :mapping="activeMapping"
  :samples="getPreviewSamples(activeMapping.id)"
  @open-preview="pullFieldSamples"
  @samples="handlePreviewSamples"
/>
```

用 `activeMapping` 取代 tab 作为主显示状态：

```ts
const activeMapping = ref<KafkaTopicMapping | null>(null)

const selectMapping = (mapping: KafkaTopicMapping) => {
  selectedMapping.value = mapping
  activeMapping.value = mapping
}
```

左侧点击和双击均调用 `selectMapping`。

- [ ] **步骤 3：保留必要 tab 状态清理**

如果 `tabs` 只服务旧预览/变量页面，本任务删除 `tabs`、`activeTabId`、`openPreview`、`openVariables` 相关主流程。右键菜单动作用：

```ts
if (action === 'open') {
  selectMapping(mapping)
  return
}
if (action === 'pullSamples') {
  selectMapping(mapping)
  pendingPullMappingId.value = String(mapping.id)
  return
}
```

如果为了降低风险暂时保留 tabbar，则必须保证默认点击规则进入对应模式面板，而不是打开“变量” tab。

- [ ] **步骤 4：Inspector 增加输出模式信息**

在 `KafkaInspectorPanel.vue` 当前规则区域增加：

```ts
{ label: '输出模式', value: mapping.outputMode === 'raw_message' ? '整包数据点' : '字段数据点' },
{ label: '输出数据点', value: mapping.rawDataPointPath || '-' },
{ label: '消费组', value: mapping.consumerGroup || '默认生成' },
```

- [ ] **步骤 5：运行类型检查**

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git status --short
git add datacenter/src/components/kafka/KafkaRawOutputPanel.vue datacenter/src/components/kafka/KafkaWorkbench.vue datacenter/src/components/kafka/KafkaInspectorPanel.vue
git commit -m "feat(kafka): 增加整包数据点输出面板"
```

## 任务 4：字段数据点面板改为字段映射与样本辅助

**文件：**
- 修改：`datacenter/src/components/kafka/KafkaFieldMappingPanel.vue`
- 修改：`datacenter/src/components/kafka/KafkaWorkbench.vue`

- [ ] **步骤 1：移除 connected prop**

在 `KafkaFieldMappingPanel.vue` props 中删除：

```ts
connected: boolean
```

删除所有 `props.connected` 判断。将“请先连接后再预览变量”类提示替换为：

```ts
ElMessage.warning('请先粘贴样本或点击拉取样本')
```

- [ ] **步骤 2：文案改为字段映射**

模板替换：

```text
变量管理 -> 字段映射
新建变量 -> 新建映射
从样本创建 -> 从样本生成
变量预览 -> 测试映射
消息预览 -> 拉取样本
变量组 -> 映射分组
变量名 -> 映射名称
```

表格列建议调整为：

```vue
<el-table-column prop="name" label="映射名称" min-width="140" show-overflow-tooltip />
<el-table-column prop="valuePath" label="字段路径" min-width="170" show-overflow-tooltip />
<el-table-column prop="dataPointPath" label="输出数据点" min-width="190" show-overflow-tooltip />
<el-table-column label="预览值" min-width="140" show-overflow-tooltip>
```

保留 `最后值/质量/更新时间` 字段，但展示文案改为“最近测试值”。

- [ ] **步骤 3：增加粘贴样本弹窗**

新增状态：

```ts
const sampleDialogVisible = ref(false)
const sampleText = ref('')
```

工具条增加按钮：

```vue
<el-button size="small" @click="openSampleDialog">
  <IconTablerBraces class="kafka-variable-panel__button-icon" />
  粘贴样本
</el-button>
```

新增弹窗：

```vue
<DcDialog v-model="sampleDialogVisible" title="粘贴 JSON 样本" width="680px">
  <el-input
    v-model="sampleText"
    type="textarea"
    :rows="14"
    placeholder='{"deviceId":"A001","temperature":26.5}'
  />
  <template #footer>
    <el-button @click="sampleDialogVisible = false">取消</el-button>
    <el-button type="primary" @click="applySampleText">解析样本</el-button>
  </template>
</DcDialog>
```

新增方法：

```ts
const openSampleDialog = () => {
  sampleText.value = ''
  sampleDialogVisible.value = true
}

const applySampleText = () => {
  try {
    const parsed = JSON.parse(sampleText.value)
    const sample = { topic: props.mapping.topic, value: parsed, timestamp: dayjs().format(TIME_FORMAT) }
    emit('samples', {
      mappingId: String(props.mapping.id),
      samples: [sample],
      preview: {
        status: 'success',
        topic: props.mapping.topic,
        samples: [sample],
        schema: [],
        diagnostics: { source: 'manual' },
      },
    })
    sampleDialogVisible.value = false
    ElMessage.success('样本已解析')
  } catch {
    ElMessage.error('JSON 样本格式无效')
  }
}
```

- [ ] **步骤 4：拉取样本按钮**

将原 `openMessagePreview` 改为 `pullSamples`：

```ts
const pullSamples = async () => {
  previewing.value = true
  try {
    const preview = await dataAPI.previewKafkaTopicMapping(props.projectId, props.mapping.id, {
      limit: props.mapping.sampleLimit,
      timeoutMs: props.mapping.timeoutMs,
      decode: props.mapping.decode,
    })
    emit('samples', {
      mappingId: String(props.mapping.id),
      samples: preview.samples || [],
      preview,
    })
    ElMessage.success('样本已拉取')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, 'Kafka 样本拉取失败'))
  } finally {
    previewing.value = false
  }
}
```

- [ ] **步骤 5：测试映射拉取一条并更新预览值**

将 `runVariablePreview` 改名为 `testMappings`，请求 limit 固定为 1：

```ts
const testMappings = async () => {
  previewing.value = true
  try {
    const preview = await dataAPI.previewKafkaTopicMapping(props.projectId, props.mapping.id, {
      limit: 1,
      timeoutMs: props.mapping.timeoutMs,
      decode: props.mapping.decode,
    })
    emit('samples', {
      mappingId: String(props.mapping.id),
      samples: preview.samples || [],
      preview,
    })
    ElMessage.success('映射测试已完成')
    await loadFields()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '测试映射失败'))
  } finally {
    previewing.value = false
  }
}
```

说明：后端现有 `PreviewTopicMapping` 会调用 `syncKafkaFieldLastValues` 写回字段最后值，本步骤复用该能力。

- [ ] **步骤 6：从样本生成映射时支持手动样本**

确认 `createFromSamples` 使用的 `props.samples` 同时接收：
- 整包面板拉取样本。
- 字段面板粘贴样本。
- 字段面板拉取样本。

当没有样本时提示：

```ts
ElMessage.warning('请先粘贴样本或拉取样本后再生成映射')
```

- [ ] **步骤 7：父组件事件接线**

在 `KafkaWorkbench.vue` 中传给 `KafkaFieldMappingPanel`：

```vue
@samples="handlePreviewSamples"
```

删除 `:connected` 与 `@open-preview` 旧接线。如保留 `@open-preview` 名称，语义必须改为拉取样本，不打开独立预览 tab。

- [ ] **步骤 8：运行类型检查**

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 9：Commit**

```powershell
git status --short
git add datacenter/src/components/kafka/KafkaFieldMappingPanel.vue datacenter/src/components/kafka/KafkaWorkbench.vue
git commit -m "feat(kafka): 优化字段数据点映射面板"
```

## 任务 5：工作台去连接化与右键菜单语义调整

**文件：**
- 修改：`datacenter/src/components/kafka/KafkaWorkbench.vue`
- 修改：`datacenter/src/components/kafka/KafkaTopicTreeBranch.vue`
- 修改：`datacenter/src/components/kafka/kafkaTopicTreeModel.ts`

- [ ] **步骤 1：左上角连接按钮改为测试 Broker**

将 header status 中按钮文案改为：

```vue
<button
  type="button"
  class="kafka-workbench__connect-action"
  :title="'测试 Broker 连通性'"
  @click="testBroker"
>
  <IconTablerLoader2 v-if="testingBroker" />
  <span>{{ testingBroker ? '测试中' : '测试 Broker' }}</span>
</button>
```

script 中把 `connected/connecting/toggleConnection` 替换为：

```ts
const testingBroker = ref(false)

const testBroker = async () => {
  if (testingBroker.value) return
  testingBroker.value = true
  try {
    await dataAPI.previewKafkaConnection(props.projectId, props.connection.id, {
      limit: 1,
      timeoutMs: 1000,
      probe: true,
    })
    ElMessage.success('Kafka Broker 可连接')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, 'Kafka Broker 测试失败'))
  } finally {
    testingBroker.value = false
  }
}
```

删除断开时关闭 preview tab 的逻辑，因为工作台不再维护长期连接。

- [ ] **步骤 2：左侧树文案改为消费规则**

替换：

```text
筛选 Topic -> 筛选消费规则
新建 Topic 映射 -> 新建消费规则
暂无 Topic 映射 -> 暂无消费规则
没有匹配的 Topic -> 没有匹配的消费规则
加载 Topic... -> 加载消费规则...
```

左侧根节点 icon 可继续使用 `IconTablerMessages`，tooltip 内容展示：

```text
Topic: xxx
消费组: yyy
输出: 整包数据点/字段数据点
```

- [ ] **步骤 3：右键菜单按模式显示**

将 mapping 右键菜单调整为：

```vue
<button v-if="contextMenu.type === 'mapping'" type="button" @click="emitContextAction('open')">
  <IconTablerSchema class="kafka-workbench__menu-icon" />
  <span>{{ contextMenu.mapping?.outputMode === 'raw_message' ? '打开输出配置' : '打开字段映射' }}</span>
</button>
<button v-if="contextMenu.type === 'mapping'" type="button" @click="emitContextAction('pullSamples')">
  <IconTablerMessages class="kafka-workbench__menu-icon" />
  <span>拉取样本</span>
</button>
<button v-if="contextMenu.type === 'mapping'" type="button" @click="emitContextAction('edit')">
  <IconTablerPencil class="kafka-workbench__menu-icon" />
  <span>编辑规则</span>
</button>
<button v-if="contextMenu.type === 'mapping'" type="button" @click="emitContextAction('copyTopic')">
  <IconTablerCopy class="kafka-workbench__menu-icon" />
  <span>复制 Topic</span>
</button>
```

动作类型更新为：

```ts
type KafkaContextAction =
  | 'open'
  | 'pullSamples'
  | 'edit'
  | 'copyTopic'
  | 'move'
  | 'delete'
  | 'createChildGroup'
  | 'renameGroup'
  | 'deleteGroup'
```

- [ ] **步骤 4：拉取样本右键行为**

为了不跨组件直接调用子组件内部方法，右键 `pullSamples` 只选中规则并标记：

```ts
const samplePullRequestId = ref(0)

if (action === 'pullSamples') {
  selectMapping(mapping)
  samplePullRequestId.value += 1
  return
}
```

传给两个面板：

```vue
:pull-request-id="samplePullRequestId"
```

在 `KafkaRawOutputPanel` 和 `KafkaFieldMappingPanel` 监听：

```ts
watch(
  () => props.pullRequestId,
  (value, oldValue) => {
    if (value !== oldValue && value > 0) void pullSamples()
  },
)
```

- [ ] **步骤 5：搜索模型增加消费组和输出模式**

在 `kafkaTopicTreeModel.ts` 的 `mappingMatches` 中扩展：

```ts
const modeLabel = mapping.outputMode === 'raw_message' ? '整包数据点' : '字段数据点'
return [mapping.name, mapping.topic, mapping.consumerGroup, modeLabel]
  .filter(Boolean)
  .some((value) => String(value).toLowerCase().includes(keyword))
```

- [ ] **步骤 6：运行类型检查**

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 7：Commit**

```powershell
git status --short
git add datacenter/src/components/kafka/KafkaWorkbench.vue datacenter/src/components/kafka/KafkaTopicTreeBranch.vue datacenter/src/components/kafka/kafkaTopicTreeModel.ts
git commit -m "feat(kafka): 调整工作台为消费规则视图"
```

## 任务 6：最终验证与人工验收

**文件：**
- 不新增代码文件。

- [ ] **步骤 1：前端类型检查**

工作目录：仓库根目录

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 2：后端测试**

工作目录：`data_service`

```powershell
go test ./internal/service ./internal/repository ./internal/http/handler
```

预期：PASS。

- [ ] **步骤 3：人工验收整包模式**

使用本地 Kafka/Redpanda 测试环境：

1. 新建 Kafka 接入源，只配置 Broker。
2. 打开 Kafka 工作台，确认左上角是 `测试 Broker`，没有连接/断开状态阻塞。
3. 新建消费规则，选择 `整包数据点`。
4. 填写 Topic、消费组、输出内容、目标数据点路径并保存。
5. 点击左侧规则，右侧显示整包输出面板。
6. 点击 `拉取样本`，确认消息列表展示 partition、offset、timestamp、payload。
7. 到数据点列表确认生成 `kafka.raw` 数据点，路径与表单一致。

- [ ] **步骤 4：人工验收字段模式**

1. 新建消费规则，选择 `字段数据点`。
2. 点击左侧规则，右侧显示字段映射面板。
3. 点击 `粘贴样本`，输入：

```json
{
  "deviceId": "A001",
  "temperature": 26.5,
  "status": "running"
}
```

4. 点击 `从样本生成`，确认生成字段映射。
5. 点击 `测试映射`，确认拉取 1 条样本并刷新预览值/质量。
6. 到数据点列表确认生成 `kafka.field` 数据点。

- [ ] **步骤 5：人工验收 Topic 可复用、名称唯一**

1. 新建规则 A：Topic `device.telemetry`，消费组 `group-a`。
2. 新建规则 B：Topic `device.telemetry`，消费组 `group-b`。
3. 确认两条规则都可保存。
4. 新建规则 C：Topic `device.telemetry`，消费组 `group-a`，但规则名称与规则 A 不同。
5. 确认规则 C 也可保存；同 Topic/同消费组的运行影响由 tooltip 提醒，不用数据库硬拦截。
6. 新建规则 D：规则名称与规则 A 相同。
7. 确认后端返回唯一约束错误或前端提示保存失败。

- [ ] **步骤 6：最终状态检查**

```powershell
git status --short
```

确认只包含本计划相关文件。若 `data_service/tmp/`、构建产物、非本任务文件出现在变更中，不要提交。

- [ ] **步骤 7：最终 Commit**

```powershell
git add data_service/internal/db/migrations/0030_kafka_workbench.sql data_service/internal/db/migrations/0050_kafka_output_mode.sql data_service/internal/db/migrations/0050_kafka_output_mode_down.sql data_service/internal/repository/kafka_workbench_repository.go data_service/internal/service/kafka_workbench_service.go data_service/internal/service/kafka_workbench_service_test.go datacenter/src/api/schemas/kafka-workbench.schema.ts datacenter/src/components/kafka
git commit -m "feat(kafka): 重构工作台消费规则输出模式"
```

提交前必须再次确认没有混入非本次 Kafka 工作台输出模式调整的文件。

## 自检结果

- 规格覆盖：已覆盖两种输出模式、新建弹窗、整包手动拉取、字段映射工具条、JSON 样本辅助、去连接化、Topic 可复用但规则名称唯一、数据点同步和验证路径。
- 占位符扫描：计划没有使用“待定/TODO/后续实现”等占位指令；节点侧长期消费明确排除在第一阶段外。
- 类型一致性：统一使用 `outputMode`、`rawOutputScope`、`rawDataPointId`、`rawDataPointPath`；输出模式枚举统一为 `raw_message` 和 `field_mapping`。
- 风险提示：当前仓库已有较多未提交改动，执行时必须逐次 `git status --short` 核对，只提交本计划涉及文件。
