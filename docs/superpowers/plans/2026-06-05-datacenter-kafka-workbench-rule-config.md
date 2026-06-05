# Kafka 工作台规则配置实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 Kafka 工作台明确收敛为“规则配置 + 临时预览辅助建点”的开发态界面，补齐 Topic 订阅的消费组、指定 Offset、右键菜单和临时连接/预览语义。

**架构：** Kafka 接入源只保存 Broker 与客户端参数；Topic 订阅保存节点侧持久运行所需的消费规则；工作台连接与预览只创建临时 Kafka reader，不推进正式消费组 offset。前后端同步扩展 Topic 订阅模型，右键菜单复用 MQTT 工作台交互模式，并保留 Kafka 的分区/offset 特性。

**技术栈：** Vue 3 + Element Plus + TypeScript + zod；Go data_service；PostgreSQL migrations；segmentio/kafka-go。

---

## 设计审查结论

当前设计方向基本正确，但必须固定以下边界，避免实现时混淆运行态：

- Kafka 工作台不是持久消费者，只负责配置规则与短时预览。
- 左上角“连接”是开发态临时连接开关；断开后不允许消息预览、变量预览等临时读取。
- Topic 订阅中的消费组名称是节点侧持久运行用的配置，工作台预览不得使用该消费组。
- 工作台预览打开时短时读取，关闭 Tab、断开连接或切换接入源后应取消/停止，不保留运行态订阅。
- 指定 Offset 只在具体分区内有意义，因此选择指定 Offset 时必须要求单分区和 offset。
- 新建 Topic 订阅不展示所属分组，默认根目录；分组归属由移动/编辑操作处理。

第一阶段不实现“查看分区信息”和“查看消费进度/lag”独立面板，也不在右键菜单中展示这两个入口。原因：它们需要额外 Kafka metadata/consumer group offset 接口，和本次配置规则闭环可以独立交付。

## 文件结构

- 修改 `data_service/internal/db/migrations/0030_kafka_workbench.sql`
  - 新库 Topic 订阅表增加 `consumer_group` 与 `start_offset`。
- 创建 `data_service/internal/db/migrations/0047_kafka_topic_subscription_runtime_fields.sql`
  - 既有库补充消费组和起始 Offset 字段。
- 创建 `data_service/internal/db/migrations/0047_kafka_topic_subscription_runtime_fields_down.sql`
  - 回滚新增字段。
- 修改 `data_service/internal/repository/kafka_workbench_repository.go`
  - Topic mapping record、create/update params、SQL scan 写入消费组与 startOffset。
- 修改 `data_service/internal/service/kafka_workbench_service.go`
  - 输入/输出模型增加消费组和 startOffset。
  - 创建/更新校验指定 Offset 必须带 partition 与 startOffset。
  - 预览 options 仍使用临时 reader，不传 consumerGroup。
- 修改 `data_service/internal/http/handler/kafka_workbench_handler.go`
  - 现有 decode 结构自动接收新增字段；如使用 raw body 标记保持 groupId 逻辑不变。
- 修改 `datacenter/src/api/schemas/kafka-workbench.schema.ts`
  - zod schema 增加 `consumerGroup` 与 `startOffset`。
- 修改 `datacenter/src/components/kafka/types.ts`
  - 复用 zod 类型，无需新增手写类型；只确认导出后类型可用。
- 修改 `datacenter/src/components/kafka/KafkaTopicMappingDialog.vue`
  - 表单文案调整为“Topic 订阅”。
  - 增加消费组名称。
  - 增加指定 Offset 输入和条件联动。
  - 新建时不展示所属分组；移动归属通过右键“移动到分组”完成。
- 修改 `datacenter/src/components/kafka/KafkaWorkbench.vue`
  - 增加 Topic/分组右键菜单。
  - 增加编辑、移动、删除、详情、复制 Topic、消息预览、变量管理等操作。
  - 将预览入口与连接状态绑定。
- 修改 `datacenter/src/components/kafka/KafkaTopicTreeBranch.vue`
  - 向上抛出 group/mapping contextmenu 事件。
- 修改 `datacenter/src/components/kafka/KafkaPreviewPanel.vue`
  - 接收 `connected` 或由父组件控制打开条件。
  - 关闭/切换时清理本地样本状态。
- 创建 `datacenter/src/components/kafka/KafkaTopicMoveDialog.vue`
  - Kafka 专用 Topic 订阅移动弹窗。

## 任务 1：后端持久模型增加消费组与起始 Offset

**文件：**
- 修改：`data_service/internal/db/migrations/0030_kafka_workbench.sql`
- 创建：`data_service/internal/db/migrations/0047_kafka_topic_subscription_runtime_fields.sql`
- 创建：`data_service/internal/db/migrations/0047_kafka_topic_subscription_runtime_fields_down.sql`

- [ ] **步骤 1：编写迁移**

在 `0030_kafka_workbench.sql` 的 `data_kafka_topic_mappings` 表中加入：

```sql
    consumer_group text NOT NULL DEFAULT '' CHECK (char_length(consumer_group) <= 200),
    start_offset bigint,
```

并在表约束中加入：

```sql
    CONSTRAINT data_kafka_topic_mappings_offset_check
        CHECK (start_position <> 'offset' OR (partition_mode = 'single' AND partition IS NOT NULL AND start_offset IS NOT NULL)),
```

创建 `0047_kafka_topic_subscription_runtime_fields.sql`：

```sql
ALTER TABLE data_kafka_topic_mappings
    ADD COLUMN IF NOT EXISTS consumer_group text NOT NULL DEFAULT '' CHECK (char_length(consumer_group) <= 200),
    ADD COLUMN IF NOT EXISTS start_offset bigint;

ALTER TABLE data_kafka_topic_mappings
    DROP CONSTRAINT IF EXISTS data_kafka_topic_mappings_offset_check,
    ADD CONSTRAINT data_kafka_topic_mappings_offset_check
        CHECK (start_position <> 'offset' OR (partition_mode = 'single' AND partition IS NOT NULL AND start_offset IS NOT NULL));
```

创建 `0047_kafka_topic_subscription_runtime_fields_down.sql`：

```sql
ALTER TABLE data_kafka_topic_mappings
    DROP CONSTRAINT IF EXISTS data_kafka_topic_mappings_offset_check,
    DROP COLUMN IF EXISTS start_offset,
    DROP COLUMN IF EXISTS consumer_group;
```

- [ ] **步骤 2：检查 SQL**

运行：

```powershell
rg -n "consumer_group|start_offset|offset_check" data_service/internal/db/migrations/0030_kafka_workbench.sql data_service/internal/db/migrations/0047_kafka_topic_subscription_runtime_fields*.sql
```

预期：三个文件均能看到对应字段或约束。

- [ ] **步骤 3：Commit**

```powershell
git add data_service/internal/db/migrations/0030_kafka_workbench.sql data_service/internal/db/migrations/0047_kafka_topic_subscription_runtime_fields.sql data_service/internal/db/migrations/0047_kafka_topic_subscription_runtime_fields_down.sql
git commit -m "feat(kafka): 扩展Topic订阅运行配置字段"
```

## 任务 2：后端 Topic 订阅读写模型同步字段

**文件：**
- 修改：`data_service/internal/repository/kafka_workbench_repository.go`
- 修改：`data_service/internal/service/kafka_workbench_service.go`
- 测试：`data_service/internal/service/kafka_workbench_service_test.go`，如不存在则创建。

- [ ] **步骤 1：编写服务校验测试**

创建或追加 `data_service/internal/service/kafka_workbench_service_test.go`：

```go
package service

import "testing"

func TestNormalizeKafkaTopicMappingRequiresOffsetForOffsetStart(t *testing.T) {
	partition := 0
	_, err := normalizeKafkaTopicMappingRuntimeFields("offset", "single", &partition, nil)
	if err == nil {
		t.Fatal("expected offset start without startOffset to fail")
	}
}

func TestNormalizeKafkaTopicMappingAcceptsOffsetWithSinglePartition(t *testing.T) {
	partition := 0
	offset := int64(42)
	result, err := normalizeKafkaTopicMappingRuntimeFields("offset", "single", &partition, &offset)
	if err != nil {
		t.Fatalf("expected valid offset config: %v", err)
	}
	if result.StartOffset == nil || *result.StartOffset != 42 {
		t.Fatalf("unexpected start offset: %#v", result.StartOffset)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
go test ./internal/service -run "TestNormalizeKafkaTopicMapping" -count=1
```

预期：FAIL，提示 `normalizeKafkaTopicMappingRuntimeFields` 未定义。

- [ ] **步骤 3：扩展 service 类型与校验**

在 `CreateKafkaTopicMappingInput` 和 `UpdateKafkaTopicMappingInput` 增加：

```go
ConsumerGroup string `json:"consumerGroup"`
StartOffset   *int64 `json:"startOffset"`
```

在 `KafkaTopicMapping` 增加：

```go
ConsumerGroup string `json:"consumerGroup"`
StartOffset   *int64 `json:"startOffset"`
```

新增 helper：

```go
type kafkaRuntimeStartConfig struct {
	StartOffset *int64
}

func normalizeKafkaTopicMappingRuntimeFields(startPosition, partitionMode string, partition *int, startOffset *int64) (kafkaRuntimeStartConfig, error) {
	if startPosition != "offset" {
		return kafkaRuntimeStartConfig{StartOffset: nil}, nil
	}
	if partitionMode != "single" || partition == nil {
		return kafkaRuntimeStartConfig{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "指定 Offset 必须选择单分区并填写分区编号")
	}
	if startOffset == nil || *startOffset < 0 {
		return kafkaRuntimeStartConfig{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "指定 Offset 必须填写非负 offset")
	}
	return kafkaRuntimeStartConfig{StartOffset: startOffset}, nil
}
```

在创建/更新 Topic mapping 归一化中：

```go
runtimeStart, err := normalizeKafkaTopicMappingRuntimeFields(startPosition, partitionMode, partition, input.StartOffset)
if err != nil {
	return repository.CreateKafkaTopicMappingParams{}, err
}
consumerGroup := strings.TrimSpace(input.ConsumerGroup)
```

创建时允许 `consumerGroup == ""`，表示节点侧按默认策略生成消费组名称；本计划不把消费组设为必填。

- [ ] **步骤 4：扩展 repository 字段**

在 `KafkaTopicMappingRecord`、`CreateKafkaTopicMappingParams`、`UpdateKafkaTopicMappingParams` 增加：

```go
ConsumerGroup string
StartOffset   *int64
```

修改 insert SQL：

```sql
consumer_group, start_offset
```

修改 update SQL：

```sql
consumer_group = $13,
start_offset = $14,
```

同步 `RETURNING` 和 `scanKafkaTopicMappingRecord`，确保扫描顺序一致。

- [ ] **步骤 5：运行测试验证通过**

运行：

```powershell
gofmt -w data_service/internal/service/kafka_workbench_service.go data_service/internal/service/kafka_workbench_service_test.go data_service/internal/repository/kafka_workbench_repository.go
go test ./internal/service ./internal/repository ./internal/http/handler
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add data_service/internal/service/kafka_workbench_service.go data_service/internal/service/kafka_workbench_service_test.go data_service/internal/repository/kafka_workbench_repository.go
git commit -m "feat(kafka): 保存Topic订阅消费规则"
```

## 任务 3：前端 schema 与 Topic 订阅表单补齐消费组和 Offset

**文件：**
- 修改：`datacenter/src/api/schemas/kafka-workbench.schema.ts`
- 修改：`datacenter/src/components/kafka/KafkaTopicMappingDialog.vue`

- [ ] **步骤 1：扩展 zod schema**

在 `KafkaTopicMappingSchema` 增加：

```ts
consumerGroup: z.string().default(''),
startOffset: z.number().nullable().optional(),
```

- [ ] **步骤 2：调整 Topic 订阅表单模型**

在 `KafkaTopicMappingDialog.vue` 的 `form` 增加：

```ts
consumerGroup: '',
startOffset: null as number | null,
```

`resetForm` 中增加：

```ts
form.consumerGroup = props.mapping?.consumerGroup || ''
form.startOffset = props.mapping?.startOffset ?? null
```

`submit` payload 增加：

```ts
consumerGroup: form.consumerGroup.trim(),
startOffset: form.startPosition === 'offset' ? form.startOffset : null,
```

- [ ] **步骤 3：调整表单 UI**

基础信息分组中，在 Topic 后增加：

```vue
<el-form-item>
  <template #label>
    <span class="kafka-topic-dialog__field-label">
      消费组名称
      <el-tooltip
        content="节点侧持久运行时用于记录消费进度；工作台预览不会使用该消费组，避免影响真实进度。"
        placement="top"
      >
        <IconTablerHelpCircle class="kafka-topic-dialog__field-help" />
      </el-tooltip>
    </span>
  </template>
  <el-input v-model="form.consumerGroup" clearable placeholder="留空时由节点侧默认策略生成" />
</el-form-item>
```

读取参数中，当 `form.startPosition === 'offset'` 时显示：

```vue
<el-form-item label="起始 Offset" required>
  <el-input-number v-model="form.startOffset" :min="0" :step="1" />
</el-form-item>
```

增加 watch：

```ts
watch(
  () => form.startPosition,
  (value) => {
    if (value === 'offset') {
      form.partitionMode = 'single'
      if (form.startOffset === null) form.startOffset = 0
    } else {
      form.startOffset = null
    }
  },
)
```

`canSubmit` 增加：

```ts
if (form.startPosition === 'offset' && (form.partitionMode !== 'single' || form.startOffset === null || form.startOffset < 0)) return false
```

- [ ] **步骤 4：运行前端类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add datacenter/src/api/schemas/kafka-workbench.schema.ts datacenter/src/components/kafka/KafkaTopicMappingDialog.vue
git commit -m "feat(kafka): 完善Topic订阅表单"
```

## 任务 4：Kafka 工作台 Topic 基础右键菜单

**文件：**
- 修改：`datacenter/src/components/kafka/KafkaWorkbench.vue`
- 修改：`datacenter/src/components/kafka/KafkaTopicTreeBranch.vue`

- [ ] **步骤 1：树节点抛出 Topic 右键事件**

在 mapping 按钮增加：

```vue
@contextmenu.prevent.stop="$emit('mappingContextmenu', $event, mapping)"
```

`defineEmits` 增加：

```ts
(event: 'mappingContextmenu', mouseEvent: MouseEvent, mapping: KafkaTopicMapping): void
```

递归子组件透传 `mappingContextmenu` 事件。

- [ ] **步骤 2：工作台根目录 Topic 绑定右键**

在 `KafkaWorkbench.vue` 根目录 mapping 按钮增加：

```vue
@contextmenu.prevent.stop="openMappingMenu($event, mapping)"
```

子组件增加：

```vue
@mapping-contextmenu="openMappingMenu"
```

- [ ] **步骤 3：增加 contextMenu 状态**

在 script 中增加：

```ts
const contextMenu = ref<{
  visible: boolean
  x: number
  y: number
  mapping: KafkaTopicMapping | null
}>({
  visible: false,
  x: 0,
  y: 0,
  mapping: null,
})
```

- [ ] **步骤 4：增加菜单模板**

在 `KafkaWorkbench.vue` 末尾加入 Teleport 菜单：

```vue
<Teleport to="body">
  <div
    v-if="contextMenu.visible"
    class="kafka-workbench__menu-mask"
    @click="closeContextMenu"
    @contextmenu.prevent="closeContextMenu"
  >
    <div
      class="kafka-workbench__context-menu"
      :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
      @click.stop
    >
      <button type="button" @click="emitContextAction('preview')">
        <IconTablerMessages class="kafka-workbench__menu-icon" />
        <span>消息预览</span>
      </button>
      <button type="button" @click="emitContextAction('fields')">
        <IconTablerSchema class="kafka-workbench__menu-icon" />
        <span>变量管理</span>
      </button>
      <button type="button" @click="emitContextAction('edit')">
        <IconTablerPencil class="kafka-workbench__menu-icon" />
        <span>编辑订阅</span>
      </button>
      <button type="button" @click="emitContextAction('copyTopic')">
        <IconTablerCopy class="kafka-workbench__menu-icon" />
        <span>复制 Topic</span>
      </button>
    </div>
  </div>
</Teleport>
```

导入图标：

```ts
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerPencil from '~icons/tabler/pencil'
```

- [ ] **步骤 5：实现菜单行为**

增加：

```ts
function openMappingMenu(event: MouseEvent, mapping: KafkaTopicMapping) {
  contextMenu.value = {
    visible: true,
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 180),
    mapping,
  }
}

function closeContextMenu() {
  contextMenu.value.visible = false
}
```

实现 action：

```ts
async function emitContextAction(
  action: 'preview' | 'fields' | 'edit' | 'copyTopic',
) {
  const { mapping } = contextMenu.value
  closeContextMenu()
  if (!mapping) return
  if (action === 'preview') {
    openPreview(mapping)
    return
  }
  if (action === 'fields') {
    openVariables(mapping)
    return
  }
  if (action === 'edit') {
    mappingDialogMode.value = 'edit'
    editingMapping.value = mapping
    mappingDialogVisible.value = true
    return
  }
  if (action === 'copyTopic') {
    await navigator.clipboard.writeText(mapping.topic)
    ElMessage.success('Topic 已复制')
  }
}
```

任务 4 只要求 Topic 菜单显示、右键定位、消息预览、变量管理、编辑订阅和复制 Topic 可用；移动、删除、分组右键在任务 5 接入真实操作。

- [ ] **步骤 6：增加菜单样式**

参考 `MqttWorkbench.vue` 的 context menu 样式，复制到 Kafka 并改类名前缀为 `kafka-workbench__`。

- [ ] **步骤 7：运行前端类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 8：Commit**

```powershell
git add datacenter/src/components/kafka/KafkaWorkbench.vue datacenter/src/components/kafka/KafkaTopicTreeBranch.vue
git commit -m "feat(kafka): 增加Topic右键菜单"
```

## 任务 5：实现移动、编辑分组与删除操作

**文件：**
- 创建：`datacenter/src/components/kafka/KafkaTopicMoveDialog.vue`
- 修改：`datacenter/src/components/kafka/KafkaWorkbench.vue`
- 修改：`datacenter/src/components/kafka/KafkaTopicGroupDialog.vue`
- 修改：`datacenter/src/components/kafka/KafkaTopicTreeBranch.vue`

- [ ] **步骤 1：树节点抛出分组右键事件**

在 `KafkaTopicTreeBranch.vue` 分组按钮增加：

```vue
@contextmenu.prevent.stop="$emit('groupContextmenu', $event, node)"
```

`defineEmits` 增加：

```ts
(event: 'groupContextmenu', mouseEvent: MouseEvent, group: KafkaTopicGroupNode): void
```

递归子组件透传 `groupContextmenu` 事件。

- [ ] **步骤 2：扩展右键菜单状态与模板**

把 `KafkaWorkbench.vue` 的 contextMenu 状态改为：

```ts
const contextMenu = ref<{
  visible: boolean
  type: 'mapping' | 'group' | null
  x: number
  y: number
  mapping: KafkaTopicMapping | null
  group: KafkaTopicGroupNode | null
}>({
  visible: false,
  type: null,
  x: 0,
  y: 0,
  mapping: null,
  group: null,
})
```

`openMappingMenu` 补充 `type` 和 `group`：

```ts
function openMappingMenu(event: MouseEvent, mapping: KafkaTopicMapping) {
  contextMenu.value = {
    visible: true,
    type: 'mapping',
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 240),
    mapping,
    group: null,
  }
}
```

增加分组右键入口并在子组件绑定：

```ts
function openGroupMenu(event: MouseEvent, group: KafkaTopicGroupNode) {
  contextMenu.value = {
    visible: true,
    type: 'group',
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 160),
    mapping: null,
    group,
  }
}
```

```vue
@group-contextmenu="openGroupMenu"
```

菜单模板增加：

```vue
<button type="button" @click="emitContextAction('move')">
  <IconTablerFolderSymlink class="kafka-workbench__menu-icon" />
  <span>移动到分组</span>
</button>
<button v-if="contextMenu.type === 'group'" type="button" @click="emitContextAction('createChildGroup')">
  <IconTablerFolderPlus class="kafka-workbench__menu-icon" />
  <span>新建子分组</span>
</button>
<button v-if="contextMenu.type === 'group'" type="button" @click="emitContextAction('renameGroup')">
  <IconTablerPencil class="kafka-workbench__menu-icon" />
  <span>重命名分组</span>
</button>
<button v-if="contextMenu.type === 'group'" type="button" class="is-danger" @click="emitContextAction('deleteGroup')">
  <IconTablerTrash class="kafka-workbench__menu-icon" />
  <span>删除分组</span>
</button>
<button v-if="contextMenu.type === 'mapping'" type="button" class="is-danger" @click="emitContextAction('delete')">
  <IconTablerTrash class="kafka-workbench__menu-icon" />
  <span>删除</span>
</button>
```

导入图标：

```ts
import IconTablerFolderSymlink from '~icons/tabler/folder-symlink'
import IconTablerTrash from '~icons/tabler/trash'
```

- [ ] **步骤 3：创建移动弹窗**

创建 `KafkaTopicMoveDialog.vue`，同时支持移动 Topic 订阅和移动分组：

```vue
<template>
  <DcDialog v-model="visible" :title="title" width="420px" :close-disabled="loading">
    <el-form label-position="top">
      <el-form-item label="目标分组">
        <el-select v-model="targetGroupId" class="kafka-topic-move-dialog__select" clearable placeholder="根目录">
          <el-option label="根目录" :value="null" />
          <el-option v-for="group in groupOptions" :key="group.id" :label="group.label" :value="group.id" />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="kafka-topic-move-dialog__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="loading" @click="submit">移动</el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { KafkaTopicGroup, KafkaTopicGroupNode, KafkaTopicMapping } from './types'
import { flattenKafkaTopicGroups } from './kafkaTopicTreeModel'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    targetType: 'mapping' | 'group'
    groups: KafkaTopicGroup[]
    mapping?: KafkaTopicMapping | null
    group?: KafkaTopicGroupNode | null
    loading?: boolean
  }>(),
  {
    mapping: null,
    group: null,
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', groupId: string | null): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const targetGroupId = ref<string | null>(null)
const blockedIds = computed(() => (props.targetType === 'group' && props.group ? collectGroupIds(props.group) : new Set<string>()))
const groupOptions = computed(() => flattenKafkaTopicGroups(props.groups, blockedIds.value))
const title = computed(() => (props.targetType === 'group' ? '移动分组' : '移动 Topic 订阅'))

function submit() {
  emit('submit', targetGroupId.value || null)
}

function collectGroupIds(group: KafkaTopicGroupNode) {
  const result = new Set<string>([String(group.id)])
  group.children.forEach((child) => {
    collectGroupIds(child).forEach((id) => result.add(id))
  })
  return result
}

watch(
  () => [props.modelValue, props.mapping?.id, props.group?.id, props.targetType] as const,
  () => {
    if (props.modelValue) {
      targetGroupId.value =
        props.targetType === 'group'
          ? props.group?.parentId
            ? String(props.group.parentId)
            : null
          : props.mapping?.groupId
            ? String(props.mapping.groupId)
            : null
    }
  },
)
</script>

<style scoped>
.kafka-topic-move-dialog__select {
  width: 100%;
}

.kafka-topic-move-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
```

- [ ] **步骤 4：工作台接入移动弹窗**

在 `KafkaWorkbench.vue` import：

```ts
import { ElMessage, ElMessageBox } from 'element-plus'
import KafkaTopicMoveDialog from './KafkaTopicMoveDialog.vue'
```

增加 state：

```ts
const moveDialogVisible = ref(false)
const moveTargetType = ref<'mapping' | 'group'>('mapping')
const movingMapping = ref<KafkaTopicMapping | null>(null)
const movingGroup = ref<KafkaTopicGroupNode | null>(null)
const moving = ref(false)
```

模板加入：

```vue
<KafkaTopicMoveDialog
  v-model="moveDialogVisible"
  :target-type="moveTargetType"
  :groups="groups"
  :mapping="movingMapping"
  :group="movingGroup"
  :loading="moving"
  @submit="handleMoveSubmit"
/>
```

- [ ] **步骤 5：实现 Topic 订阅移动和删除**

增加：

```ts
function openMoveMappingDialog(mapping: KafkaTopicMapping) {
  moveTargetType.value = 'mapping'
  movingMapping.value = mapping
  movingGroup.value = null
  moveDialogVisible.value = true
}

async function handleMoveSubmit(groupId: string | null) {
  if (!movingMapping.value) return
  moving.value = true
  try {
    await dataAPI.updateKafkaTopicMapping(props.projectId, movingMapping.value.id, {
      ...buildMappingPayload(movingMapping.value),
      groupId,
    })
    ElMessage.success('Topic 订阅已移动')
    moveDialogVisible.value = false
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '移动 Topic 订阅失败'))
  } finally {
    moving.value = false
  }
}

function buildMappingPayload(mapping: KafkaTopicMapping) {
  return {
    name: mapping.name,
    topic: mapping.topic,
    groupId: mapping.groupId || null,
    consumerGroup: mapping.consumerGroup || '',
    partitionMode: mapping.partitionMode,
    partition: mapping.partition ?? null,
    startPosition: mapping.startPosition,
    startOffset: mapping.startOffset ?? null,
    decode: mapping.decode,
    sampleLimit: mapping.sampleLimit,
    timeoutMs: mapping.timeoutMs,
    description: mapping.description || '',
  }
}

async function deleteMapping(mapping: KafkaTopicMapping) {
  await ElMessageBox.confirm(`删除 Topic 订阅“${mapping.name || mapping.topic}”？相关变量配置会一并删除。`, '删除 Topic 订阅', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning',
  })
  await dataAPI.deleteKafkaTopicMapping(props.projectId, mapping.id)
  ElMessage.success('Topic 订阅已删除')
  await loadWorkbench()
}
```

将任务 4 中 `move` action 接到：

```ts
async function emitContextAction(
  action:
    | 'preview'
    | 'fields'
    | 'edit'
    | 'copyTopic'
    | 'move'
    | 'delete'
    | 'createChildGroup'
    | 'renameGroup'
    | 'deleteGroup',
) {
  const { type, mapping, group } = contextMenu.value
  closeContextMenu()
  if (type === 'mapping' && mapping) {
    if (action === 'preview') {
      openPreview(mapping)
      return
    }
    if (action === 'fields') {
      openVariables(mapping)
      return
    }
    if (action === 'edit') {
      mappingDialogMode.value = 'edit'
      editingMapping.value = mapping
      mappingDialogVisible.value = true
      return
    }
    if (action === 'copyTopic') {
      await navigator.clipboard.writeText(mapping.topic)
      ElMessage.success('Topic 已复制')
      return
    }
    if (action === 'move') {
      openMoveMappingDialog(mapping)
      return
    }
    if (action === 'delete') {
      await deleteMapping(mapping)
      return
    }
  }
  if (type === 'group' && group) {
    if (action === 'createChildGroup') {
      openCreateChildGroup(group)
      return
    }
    if (action === 'renameGroup') {
      openRenameGroup(group)
      return
    }
    if (action === 'move') {
      openMoveGroupDialog(group)
      return
    }
    if (action === 'deleteGroup') {
      await deleteGroup(group)
      return
    }
  }
}
```

其中 Topic 移动分支为：

```ts
if (action === 'move' && mapping) {
  openMoveMappingDialog(mapping)
  return
}
```

Topic 删除分支为：

```ts
if (action === 'delete') {
  await deleteMapping(mapping)
  return
}
```

- [ ] **步骤 6：实现分组新建子分组、重命名、移动和删除**

在 `KafkaWorkbench.vue` 增加状态：

```ts
const groupDialogMode = ref<'create' | 'edit'>('create')
const editingGroup = ref<KafkaTopicGroupNode | null>(null)
const pendingParentGroupId = ref<string | null>(null)
```

把分组弹窗模板改为：

```vue
<KafkaTopicGroupDialog
  v-model="groupDialogVisible"
  :mode="groupDialogMode"
  :groups="groups"
  :group="editingGroup"
  :loading="groupSaving"
  @submit="saveGroup"
/>
```

实现：

```ts
function openCreateGroup() {
  groupDialogMode.value = 'create'
  editingGroup.value = null
  pendingParentGroupId.value = null
  groupDialogVisible.value = true
}

function openCreateChildGroup(group: KafkaTopicGroupNode) {
  groupDialogMode.value = 'create'
  editingGroup.value = null
  pendingParentGroupId.value = String(group.id)
  groupDialogVisible.value = true
}

function openRenameGroup(group: KafkaTopicGroupNode) {
  groupDialogMode.value = 'edit'
  editingGroup.value = group
  pendingParentGroupId.value = null
  groupDialogVisible.value = true
}

function openMoveGroupDialog(group: KafkaTopicGroupNode) {
  moveTargetType.value = 'group'
  movingGroup.value = group
  movingMapping.value = null
  moveDialogVisible.value = true
}

async function saveGroup(payload: { name: string; parentId: string | null }) {
  groupSaving.value = true
  try {
    if (groupDialogMode.value === 'edit' && editingGroup.value) {
      await dataAPI.updateKafkaTopicGroup(props.projectId, editingGroup.value.id, {
        name: payload.name,
        parentId: editingGroup.value.parentId || null,
      })
      ElMessage.success('Topic 分组已更新')
    } else {
      await dataAPI.createKafkaTopicGroup(props.projectId, props.connection.id, {
        name: payload.name,
        parentId: pendingParentGroupId.value || payload.parentId,
      })
      ElMessage.success('Topic 分组已创建')
    }
    groupDialogVisible.value = false
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存 Topic 分组失败'))
  } finally {
    groupSaving.value = false
  }
}
```

扩展 `handleMoveSubmit`：

```ts
async function handleMoveSubmit(groupId: string | null) {
  moving.value = true
  try {
    if (moveTargetType.value === 'mapping' && movingMapping.value) {
      await dataAPI.updateKafkaTopicMapping(props.projectId, movingMapping.value.id, {
        ...buildMappingPayload(movingMapping.value),
        groupId,
      })
      ElMessage.success('Topic 订阅已移动')
    } else if (moveTargetType.value === 'group' && movingGroup.value) {
      await dataAPI.updateKafkaTopicGroup(props.projectId, movingGroup.value.id, {
        name: movingGroup.value.name,
        parentId: groupId,
      })
      ElMessage.success('Topic 分组已移动')
    }
    moveDialogVisible.value = false
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '移动失败'))
  } finally {
    moving.value = false
  }
}
```

删除分组：

```ts
async function deleteGroup(group: KafkaTopicGroupNode) {
  await ElMessageBox.confirm(`确认删除分组「${group.name}」？组内 Topic 订阅会回到根目录。`, '删除分组', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning',
  })
  await dataAPI.deleteKafkaTopicGroup(props.projectId, group.id)
  ElMessage.success('Topic 分组已删除')
  await loadWorkbench()
}
```

在 `KafkaTopicGroupDialog.vue` 中，编辑模式不显示上级分组，只修改名称；创建模式保留上级分组选择。

分组 action 分支为：

```ts
if (action === 'createChildGroup') {
  openCreateChildGroup(group)
  return
}
if (action === 'renameGroup') {
  openRenameGroup(group)
  return
}
if (action === 'move') {
  openMoveGroupDialog(group)
  return
}
if (action === 'deleteGroup') {
  await deleteGroup(group)
  return
}
```

- [ ] **步骤 7：运行前端类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 8：Commit**

```powershell
git add datacenter/src/components/kafka/KafkaTopicMoveDialog.vue datacenter/src/components/kafka/KafkaWorkbench.vue datacenter/src/components/kafka/KafkaTopicGroupDialog.vue datacenter/src/components/kafka/KafkaTopicTreeBranch.vue
git commit -m "feat(kafka): 支持Topic和分组右键操作"
```

## 任务 6：连接状态约束临时预览入口

**文件：**
- 修改：`datacenter/src/components/kafka/KafkaWorkbench.vue`
- 修改：`datacenter/src/components/kafka/KafkaPreviewPanel.vue`
- 修改：`datacenter/src/components/kafka/KafkaFieldMappingPanel.vue`

- [ ] **步骤 1：工作台打开预览前检查连接状态**

在 `openPreview` 开头增加：

```ts
if (!connected.value) {
  ElMessage.warning('请先点击左上角连接，再执行 Kafka 消息预览')
  return
}
```

在 `openVariables` 保持允许打开变量管理，但变量预览按钮由 `KafkaFieldMappingPanel` 的 `connected` prop 控制。

- [ ] **步骤 2：PreviewPanel 接收 connected**

在 `KafkaPreviewPanel.vue` props 增加：

```ts
connected: boolean
```

`runPreview` 开头增加：

```ts
if (!props.connected) {
  ElMessage.warning('请先连接 Kafka 后再预览消息')
  return
}
```

`onMounted(runPreview)` 改为：

```ts
onMounted(() => {
  if (props.connected) void runPreview()
})
```

watch mapping 时也仅在 connected 时执行。

- [ ] **步骤 3：父组件传入 connected**

在 `KafkaWorkbench.vue`：

```vue
<KafkaPreviewPanel
  v-if="activeTab?.type === 'preview'"
  :project-id="projectId"
  :mapping="activeTab.mapping"
  :connected="connected"
  @samples="handlePreviewSamples"
/>
```

- [ ] **步骤 4：断开连接时关闭预览 Tab 并清空样本**

在 `toggleConnection` 的断开逻辑中保留现有：

```ts
tabs.value = tabs.value.filter((tab) => tab.type !== 'preview')
```

并补充：

```ts
activePreview.value = null
previewSamplesByMapping.value = new Map()
```

- [ ] **步骤 5：运行前端类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add datacenter/src/components/kafka/KafkaWorkbench.vue datacenter/src/components/kafka/KafkaPreviewPanel.vue datacenter/src/components/kafka/KafkaFieldMappingPanel.vue
git commit -m "fix(kafka): 约束工作台临时预览连接状态"
```

## 任务 7：最终验证与人工验收清单

**文件：**
- 不新增代码文件。

- [ ] **步骤 1：运行前端类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 2：运行后端相关测试**

运行：

```powershell
go test ./internal/service ./internal/repository ./internal/http/handler
```

工作目录：`data_service`

预期：PASS。

- [ ] **步骤 3：人工验收 Kafka Topic 订阅表单**

使用 Redpanda/Kafka 测试环境：

1. 新建 Kafka 接入源，只配置 Broker。
2. 进入 Kafka 工作台，点击左上角连接。
3. 新建 Topic 订阅，确认表单不显示所属分组。
4. 填写消费组名称，保存后重新打开编辑，确认回显。
5. 选择起始位置“指定 Offset”，确认自动要求单分区并显示起始 Offset。
6. 不填 offset 时保存按钮不可用或后端返回明确错误。

- [ ] **步骤 4：人工验收右键菜单**

1. 根目录 Topic 右键，确认出现消息预览、变量管理、编辑订阅、复制 Topic、移动到分组、删除。
2. 分组内 Topic 右键，确认行为一致。
3. 分组右键，确认能打开新建子分组、重命名、移动到分组、删除。
4. 删除 Topic 前有确认弹窗。
5. 移动 Topic 到分组后树刷新，Topic 出现在目标分组。

- [ ] **步骤 5：人工验收临时预览语义**

1. 未连接时打开消息预览，应提示先连接。
2. 连接后打开消息预览，刷新可读取样本。
3. 关闭预览 Tab 后样本不再继续追加。
4. 断开连接后预览 Tab 被关闭或无法刷新。
5. 预览不使用表单里的消费组名称；该消费组只保存为节点侧运行配置。

- [ ] **步骤 6：最终 Commit**

```powershell
git status --short
git add datacenter/src/components/kafka data_service/internal data_service/go.mod data_service/go.sum
git commit -m "feat(kafka): 完善工作台订阅规则配置"
```

提交前必须确认 `git status --short` 中没有混入非本任务文件。

## 自检结果

- 规格覆盖：消费组、指定 Offset、临时连接/预览、右键菜单、移动/删除、验证路径均有任务覆盖。
- 占位符扫描：计划未使用“待定/TODO/后续实现”等占位指令；分区信息和消费进度面板已明确排除在第一阶段外。
- 类型一致性：统一使用 `consumerGroup`、`startOffset`、`partitionMode`、`partition`、`startPosition` 字段名；前端 zod、Vue 表单、Go service/repository 保持一致。
