# Kafka 工作台模式标签页改版实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 改造 Kafka 工作台的新建/编辑消费规则弹窗和左侧规则树右边的主工作区，让整包数据点与字段映射两种模式各自打开独立标签页，并用 JSON 样例自动展平辅助字段映射。

**架构：** 左侧 Kafka 消费规则树、分组和右键菜单保持现状；左侧树右边的主工作区改为动态标签页容器，只打开 `整包数据点` 和 `字段映射` 两类标签。前端优先复用 `WorkbenchStreamToolbar`、`WorkbenchStreamMessageList`、`WorkbenchStatusPill`、`MonacoEditor`、`BulkActionBar`，并参考 MQTT 工作台抽取 `WorkbenchTabBar` 与 `WorkbenchJsonSampleEditor` 两个公共组件；后端补一条输出模式不可修改的服务层防线。

**技术栈：** Vue 3、Element Plus、TypeScript、zod、Vitest、Monaco Editor、Go、PostgreSQL。

---

## 规格修正说明

本计划按用户最新确认执行：这里的“右侧面板”指 Kafka 左侧规则树右边的主工作区，而不是额外第三列静态 inspector。实现后 Kafka 工作台布局应参考 MQTT 的“左侧 explorer + 右侧 tab 工作区”交互，`KafkaInspectorPanel` 不再作为独立第三列展示。

## 文件结构

- 修改 `data_service/internal/service/kafka_workbench_service.go`
  - 在更新消费规则时禁止修改已有 `outputMode`。
- 修改 `data_service/internal/service/kafka_workbench_service_test.go`
  - 增加输出模式不可修改的服务层测试。
- 创建 `datacenter/src/components/kafka/kafkaSampleFields.ts`
  - 承载 Kafka 样例归一化、JSON 解析、字段自动展平和候选字段生成的纯函数。
- 创建 `datacenter/tests/kafka-sample-fields.test.ts`
  - 覆盖 JSON 对象、字符串 JSON、Kafka 完整 sample、数组路径和已存在字段标记。
- 创建 `datacenter/src/components/workbench/WorkbenchTabBar.vue`
  - 抽取公共、可关闭、紧凑的工作台标签栏，本轮只接入 Kafka，MQTT 保持现状。
- 创建 `datacenter/src/components/workbench/WorkbenchJsonSampleEditor.vue`
  - 从 MQTT 批量映射的样例消息编辑区抽取通用 JSON 样例编辑器交互，内部使用 `MonacoEditor`，不包含 MQTT/Kafka 业务字段。
- 修改 `datacenter/src/components/kafka/KafkaTopicMappingDialog.vue`
  - 输出模式改为单选卡片或 radio，编辑态禁用；运行参数收进高级配置区；整包数据点名称默认跟随规则名称。
- 修改 `datacenter/src/components/kafka/KafkaWorkbench.vue`
  - 保留左侧树；右侧主工作区改为动态模式标签页；删除第三列 inspector 依赖；维护样本缓存。
- 修改 `datacenter/src/components/kafka/KafkaRawOutputPanel.vue`
  - 作为整包数据点标签内容，复用公共消息流组件，只保留测试和样本操作。
- 修改 `datacenter/src/components/kafka/KafkaFieldMappingPanel.vue`
  - 改为字段映射标签内容；嵌入 Monaco JSON 样例编辑器；基于 `kafkaSampleFields.ts` 解析候选字段；复用公共组件。
- 修改 `datacenter/src/components/kafka/types.ts`
  - 如需要，补充 Kafka 模式标签和候选字段类型导出。

## 任务 1：后端禁止更新消费规则输出模式

**文件：**

- 修改：`data_service/internal/service/kafka_workbench_service.go`
- 修改：`data_service/internal/service/kafka_workbench_service_test.go`

- [ ] **步骤 1：编写失败的服务测试**

在 `data_service/internal/service/kafka_workbench_service_test.go` 扩展 import：

```go
import (
	"context"
	"strings"
	"testing"

	"github.com/indu-forge/data_service/internal/repository"
)
```

追加测试：

```go
func TestNormalizeUpdateTopicMappingRejectsOutputModeChange(t *testing.T) {
	service := &KafkaWorkbenchService{}
	current := repository.KafkaTopicMappingRecord{
		ID:             "mapping-1",
		ProjectID:      "project-1",
		ConnectionID:   "connection-1",
		Name:           "设备遥测",
		Topic:          "device.telemetry",
		ConsumerGroup:  "",
		OutputMode:     "raw_message",
		RawOutputScope: "value",
		PartitionMode:  "all",
		StartPosition:  "latest",
		Decode:         "json",
		SampleLimit:    100,
		TimeoutMS:      5000,
	}

	_, err := service.normalizeUpdateTopicMapping(
		context.Background(),
		"project-1",
		"user-1",
		current,
		UpdateKafkaTopicMappingInput{
			Name:           "设备遥测",
			Topic:          "device.telemetry",
			OutputMode:     "field_mapping",
			RawOutputScope: "value",
			PartitionMode:  "all",
			StartPosition:  "latest",
			Decode:         "json",
			SampleLimit:    100,
			TimeoutMS:      5000,
		},
	)

	if err == nil {
		t.Fatal("expected output mode change to fail")
	}
	if !strings.Contains(err.Error(), "输出模式创建后不可修改") {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **步骤 2：运行测试确认失败**

运行：

```powershell
Set-Location data_service
go test ./internal/service -run "TestNormalizeUpdateTopicMappingRejectsOutputModeChange" -count=1
```

预期：FAIL，因为当前 `normalizeUpdateTopicMapping` 仍允许从 `raw_message` 更新为 `field_mapping`。

- [ ] **步骤 3：实现最小后端防线**

在 `normalizeUpdateTopicMapping` 内，`decode := normalizeKafkaDecode(...)` 前增加：

```go
	requestedOutputMode := strings.ToLower(strings.TrimSpace(input.OutputMode))
	if requestedOutputMode != "" && requestedOutputMode != current.OutputMode {
		return repository.UpdateKafkaTopicMappingParams{}, apperrors.NewAppError(
			apperrors.ErrorCodeBadRequest,
			http.StatusBadRequest,
			"Kafka 输出模式创建后不可修改",
		)
	}
```

把原来的：

```go
	outputMode, rawScope, err := normalizeKafkaOutputConfig(fallbackTrimmed(input.OutputMode, current.OutputMode), input.RawOutputScope)
```

改为：

```go
	outputMode, rawScope, err := normalizeKafkaOutputConfig(current.OutputMode, input.RawOutputScope)
```

这样编辑态仍可修改整包模式的 `rawOutputScope`，但不能改变 `outputMode`。

- [ ] **步骤 4：运行后端测试验证通过**

运行：

```powershell
Set-Location data_service
gofmt -w internal/service/kafka_workbench_service.go internal/service/kafka_workbench_service_test.go
go test ./internal/service -run "TestNormalizeKafka|TestNormalizeUpdateTopicMapping" -count=1
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add data_service/internal/service/kafka_workbench_service.go data_service/internal/service/kafka_workbench_service_test.go
git commit -m "fix(kafka): 禁止修改消费规则输出模式"
```

## 任务 2：抽取 Kafka 样例字段解析纯函数

**文件：**

- 创建：`datacenter/src/components/kafka/kafkaSampleFields.ts`
- 创建：`datacenter/tests/kafka-sample-fields.test.ts`

- [ ] **步骤 1：编写失败的前端测试**

创建 `datacenter/tests/kafka-sample-fields.test.ts`：

```ts
import { describe, expect, test } from 'vitest'
import {
  inferKafkaSampleFields,
  normalizeKafkaSampleEditorText,
} from '@/components/kafka/kafkaSampleFields'

describe('kafkaSampleFields', () => {
  test('从 Kafka sample.value 对象展平字段', () => {
    const fields = inferKafkaSampleFields(
      [
        {
          topic: 'device.telemetry',
          value: {
            deviceId: 'A001',
            temperature: 26.5,
            metrics: { pressure: 0.72 },
          },
        },
      ],
      [],
    )

    expect(fields.map((field) => field.path)).toEqual([
      'deviceId',
      'temperature',
      'metrics.pressure',
    ])
    expect(fields.find((field) => field.path === 'temperature')?.dataType).toBe('number')
  })

  test('支持 value 为 JSON 字符串并标记已存在字段', () => {
    const fields = inferKafkaSampleFields(
      [{ value: '{"deviceId":"A001","running":true}' }],
      ['deviceId'],
    )

    expect(fields.find((field) => field.path === 'deviceId')?.exists).toBe(true)
    expect(fields.find((field) => field.path === 'running')?.dataType).toBe('boolean')
  })

  test('数组按普通索引路径展开，不生成数组拆分规则', () => {
    const fields = inferKafkaSampleFields(
      [{ value: { items: [{ name: 'temp', value: 26.5 }] } }],
      [],
    )

    expect(fields.map((field) => field.path)).toContain('items.0.name')
    expect(fields.map((field) => field.path)).toContain('items.0.value')
  })

  test('编辑器文本格式化为 JSON 对象字符串', () => {
    const text = normalizeKafkaSampleEditorText({ value: { status: 'ok' } })
    expect(text).toContain('"status": "ok"')
  })
})
```

- [ ] **步骤 2：运行测试确认失败**

运行：

```powershell
pnpm --filter datacenter test -- tests/kafka-sample-fields.test.ts
```

预期：FAIL，因为 `kafkaSampleFields.ts` 还不存在。

- [ ] **步骤 3：实现样例解析纯函数**

创建 `datacenter/src/components/kafka/kafkaSampleFields.ts`：

```ts
export type KafkaSampleFieldType = 'string' | 'number' | 'boolean' | 'object' | 'array'

export type KafkaSampleFieldCandidate = {
  path: string
  name: string
  dataType: KafkaSampleFieldType
  sampleValue: unknown
  exists: boolean
}

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  Object.prototype.toString.call(value) === '[object Object]'

export const parseKafkaJsonValue = (value: unknown): unknown => {
  if (typeof value !== 'string') return value
  const trimmed = value.trim()
  if (!trimmed) return value
  try {
    return JSON.parse(trimmed)
  } catch {
    return value
  }
}

export const normalizeKafkaSampleRoot = (sample: unknown): unknown => {
  if (isPlainObject(sample) && Object.prototype.hasOwnProperty.call(sample, 'value')) {
    return parseKafkaJsonValue(sample.value)
  }
  return parseKafkaJsonValue(sample)
}

export const resolveKafkaSampleType = (value: unknown): KafkaSampleFieldType => {
  if (Array.isArray(value)) return 'array'
  if (value !== null && typeof value === 'object') return 'object'
  if (typeof value === 'number') return 'number'
  if (typeof value === 'boolean') return 'boolean'
  return 'string'
}

const defaultNameFromPath = (path: string) => {
  const last = path.split('.').filter(Boolean).at(-1)
  return last || path
}

export const inferKafkaSampleFields = (
  samples: unknown[],
  existingPaths: Iterable<string> = [],
): KafkaSampleFieldCandidate[] => {
  const existing = new Set(Array.from(existingPaths).map(String))
  const seen = new Map<string, KafkaSampleFieldCandidate>()

  const addCandidate = (path: string, value: unknown) => {
    if (!path || seen.has(path)) return
    seen.set(path, {
      path,
      name: defaultNameFromPath(path),
      dataType: resolveKafkaSampleType(value),
      sampleValue: value,
      exists: existing.has(path),
    })
  }

  const visit = (prefix: string, value: unknown) => {
    if (Array.isArray(value)) {
      if (value.length === 0) {
        addCandidate(prefix, value)
        return
      }
      value.forEach((item, index) => visit(prefix ? `${prefix}.${index}` : String(index), item))
      return
    }
    if (isPlainObject(value)) {
      const entries = Object.entries(value)
      if (entries.length === 0) {
        addCandidate(prefix, value)
        return
      }
      entries.forEach(([key, child]) => visit(prefix ? `${prefix}.${key}` : key, child))
      return
    }
    addCandidate(prefix, value)
  }

  samples.forEach((sample) => visit('', normalizeKafkaSampleRoot(sample)))
  return Array.from(seen.values())
}

export const normalizeKafkaSampleEditorText = (sample: unknown): string => {
  const root = normalizeKafkaSampleRoot(sample)
  if (typeof root === 'string') return root
  try {
    return JSON.stringify(root ?? {}, null, 2)
  } catch {
    return String(root ?? '')
  }
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：

```powershell
pnpm --filter datacenter test -- tests/kafka-sample-fields.test.ts
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add datacenter/src/components/kafka/kafkaSampleFields.ts datacenter/tests/kafka-sample-fields.test.ts
git commit -m "feat(kafka): 抽取样例字段解析模型"
```

## 任务 3：抽取公共工作台标签栏与 JSON 样例编辑器

**文件：**

- 创建：`datacenter/src/components/workbench/WorkbenchTabBar.vue`
- 创建：`datacenter/src/components/workbench/WorkbenchJsonSampleEditor.vue`

- [ ] **步骤 1：创建公共组件**

创建 `datacenter/src/components/workbench/WorkbenchTabBar.vue`：

```vue
<template>
  <nav class="workbench-tabbar" aria-label="工作台标签页">
    <button
      v-for="tab in tabs"
      :key="tab.id"
      type="button"
      class="workbench-tabbar__tab"
      :class="{ 'is-active': tab.id === activeId }"
      @click="$emit('update:activeId', tab.id)"
    >
      <component :is="tab.icon" v-if="tab.icon" />
      <span>{{ tab.title }}</span>
      <button
        v-if="closable"
        type="button"
        class="workbench-tabbar__close"
        :aria-label="`关闭 ${tab.title}`"
        @click.stop="$emit('close', tab.id)"
      >
        <IconTablerX />
      </button>
    </button>
  </nav>
</template>

<script setup lang="ts">
import IconTablerX from '~icons/tabler/x'

export type WorkbenchTabBarItem = {
  id: string
  title: string
  icon?: any
}

withDefaults(
  defineProps<{
    tabs: WorkbenchTabBarItem[]
    activeId: string
    closable?: boolean
  }>(),
  {
    closable: true,
  },
)

defineEmits<{
  (event: 'update:activeId', value: string): void
  (event: 'close', value: string): void
}>()
</script>

<style scoped>
.workbench-tabbar {
  min-height: 38px;
  display: flex;
  overflow-x: auto;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.workbench-tabbar:empty {
  display: none;
}

.workbench-tabbar__tab {
  min-width: 132px;
  max-width: 240px;
  height: 38px;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 0 9px;
  border: 0;
  border-right: 1px solid var(--dc-border);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.workbench-tabbar__tab.is-active {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  font-weight: 700;
}

.workbench-tabbar__tab span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workbench-tabbar__tab svg {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

.workbench-tabbar__close {
  width: 18px;
  height: 18px;
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-muted);
}

.workbench-tabbar__close:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-danger);
}
</style>
```

- [ ] **步骤 2：创建公共 JSON 样例编辑器**

创建 `datacenter/src/components/workbench/WorkbenchJsonSampleEditor.vue`。这个组件复用 MQTT 批量映射中“样例消息 + Monaco + 操作按钮”的交互形态，但 props 和事件保持通用，避免引入 MQTT 语义：

```vue
<template>
  <section class="workbench-json-sample-editor">
    <header class="workbench-json-sample-editor__header">
      <strong>{{ title }}</strong>
      <div class="workbench-json-sample-editor__actions">
        <el-button size="small" :disabled="fillLatestDisabled" @click="$emit('fillLatest')">
          {{ fillLatestText }}
        </el-button>
        <el-button size="small" @click="formatEditor">格式化</el-button>
        <el-button type="primary" size="small" @click="$emit('parse')">
          {{ parseText }}
        </el-button>
      </div>
    </header>
    <MonacoEditor
      ref="editorRef"
      class="workbench-json-sample-editor__monaco"
      :model-value="modelValue"
      language="json"
      theme="vs"
      height="100%"
      :options="editorOptions"
      @update:model-value="$emit('update:modelValue', $event)"
    />
    <p v-if="error" class="workbench-json-sample-editor__error">{{ error }}</p>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import MonacoEditor from '@/components/MonacoEditor.vue'

const props = withDefaults(
  defineProps<{
    modelValue: string
    title?: string
    fillLatestText?: string
    fillLatestDisabled?: boolean
    parseText?: string
    error?: string
  }>(),
  {
    title: 'JSON 样例',
    fillLatestText: '填入最近样本',
    fillLatestDisabled: false,
    parseText: '解析字段',
    error: '',
  },
)

defineEmits<{
  (event: 'update:modelValue', value: string): void
  (event: 'fillLatest'): void
  (event: 'parse'): void
}>()

const editorRef = ref<InstanceType<typeof MonacoEditor> | null>(null)
const editorOptions = {
  minimap: { enabled: false },
  lineNumbers: 'on',
  wordWrap: 'on',
  scrollBeyondLastLine: false,
}

const formatEditor = async () => {
  await editorRef.value?.format?.()
}

defineExpose({
  format: formatEditor,
  focus: () => editorRef.value?.focus?.(),
})
</script>

<style scoped>
.workbench-json-sample-editor {
  min-height: 0;
  min-width: 0;
  display: grid;
  grid-template-rows: auto minmax(220px, 1fr) auto;
  gap: 8px;
}

.workbench-json-sample-editor__header,
.workbench-json-sample-editor__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.workbench-json-sample-editor__header {
  justify-content: space-between;
}

.workbench-json-sample-editor__header strong {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workbench-json-sample-editor__actions {
  flex-shrink: 0;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.workbench-json-sample-editor__monaco {
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
}

.workbench-json-sample-editor__error {
  margin: 0;
  color: var(--dc-danger);
  font-size: 12px;
  line-height: 1.5;
}
</style>
```

- [ ] **步骤 3：运行类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 4：Commit**

```powershell
git add datacenter/src/components/workbench/WorkbenchTabBar.vue datacenter/src/components/workbench/WorkbenchJsonSampleEditor.vue
git commit -m "feat(workbench): 抽取工作台通用组件"
```

## 任务 4：改造消费规则新建/编辑弹窗

**文件：**

- 修改：`datacenter/src/components/kafka/KafkaTopicMappingDialog.vue`

- [ ] **步骤 1：改造输出模式 UI**

将输出模式从 `el-segmented` 改为模式卡片。模板中替换输出模式表单项核心内容：

```vue
<div class="kafka-topic-dialog__mode-options" :class="{ 'is-readonly': mode === 'edit' }">
  <button
    v-for="option in outputModeOptions"
    :key="option.value"
    type="button"
    class="kafka-topic-dialog__mode-option"
    :class="{ 'is-active': form.outputMode === option.value }"
    :disabled="mode === 'edit'"
    @click="form.outputMode = option.value"
  >
    <span class="kafka-topic-dialog__mode-radio">
      {{ form.outputMode === option.value ? '●' : '○' }}
    </span>
    <span>
      <strong>{{ option.label }}</strong>
      <em>{{ option.description }}</em>
    </span>
  </button>
</div>
<p v-if="mode === 'edit'" class="kafka-topic-dialog__hint">
  输出模式创建后不可修改；如需切换模式，请删除后重新创建消费规则。
</p>
```

把 `outputModeOptions` 改为：

```ts
const outputModeOptions = [
  {
    label: '整包数据点',
    value: 'raw_message',
    description: '整条 Kafka 消息作为一个数据点输出，数据点名称默认使用消费规则名称。',
  },
  {
    label: '字段映射',
    value: 'field_mapping',
    description: '从消息 value 中提取字段，保存后在字段映射标签页生成数据点。',
  },
] as const
```

- [ ] **步骤 2：收起复杂运行参数**

用 `el-collapse` 包裹当前 “运行消费与样本参数” 区块：

```vue
<el-collapse class="kafka-topic-dialog__advanced">
  <el-collapse-item title="高级配置：消费起点与测试参数" name="runtime">
    <div class="kafka-topic-dialog__grid">
      <!-- 保留现有分区、起始位置、offset、解码、样本上限、预览超时表单项 -->
    </div>
  </el-collapse-item>
</el-collapse>
```

字段映射模式提示保留，但文案改为：

```vue
<div v-else class="kafka-topic-dialog__mode-note">
  保存后会打开字段映射标签页，可通过测试拉取或粘贴 JSON 样例解析字段。
</div>
```

- [ ] **步骤 3：保证编辑态提交不会切换模式**

在 `submit` 中生成 payload 前增加本地保护：

```ts
const outputMode =
  props.mode === 'edit' ? props.mapping?.outputMode || form.outputMode : form.outputMode
```

并把 payload 中的 `outputMode: form.outputMode` 改为：

```ts
outputMode,
```

整包相关字段按 `outputMode === 'raw_message'` 判断。

- [ ] **步骤 4：补充样式**

追加样式：

```css
.kafka-topic-dialog__mode-options {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.kafka-topic-dialog__mode-option {
  min-height: 86px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  text-align: left;
}

.kafka-topic-dialog__mode-option.is-active {
  border-color: color-mix(in oklch, var(--dc-primary) 44%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.kafka-topic-dialog__mode-option:disabled {
  cursor: not-allowed;
  opacity: 0.78;
}

.kafka-topic-dialog__mode-option strong,
.kafka-topic-dialog__mode-option em {
  display: block;
}

.kafka-topic-dialog__mode-option em {
  margin-top: 4px;
  color: var(--dc-text-muted);
  font-size: 12px;
  font-style: normal;
  line-height: 1.5;
}

.kafka-topic-dialog__hint {
  margin: 6px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.5;
}

.kafka-topic-dialog__advanced :deep(.el-collapse-item__content) {
  padding-bottom: 0;
}
```

- [ ] **步骤 5：运行类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add datacenter/src/components/kafka/KafkaTopicMappingDialog.vue
git commit -m "feat(kafka): 锁定消费规则输出模式"
```

## 任务 5：将 Kafka 主工作区改为模式标签页

**文件：**

- 修改：`datacenter/src/components/kafka/KafkaWorkbench.vue`
- 可删除引用：`datacenter/src/components/kafka/KafkaInspectorPanel.vue`

- [ ] **步骤 1：新增标签页状态和类型**

在 `KafkaWorkbench.vue` script 中加入：

```ts
type KafkaModeTab = {
  id: string
  type: 'raw' | 'fields'
  title: string
  mappingId: string
}

const modeTabs = ref<KafkaModeTab[]>([])
const activeModeTabId = ref('')
```

新增 helper：

```ts
const modeTabId = (mapping: KafkaTopicMapping) =>
  mapping.outputMode === 'raw_message' ? `kafka-raw-${mapping.id}` : `kafka-fields-${mapping.id}`

const modeTabType = (mapping: KafkaTopicMapping): KafkaModeTab['type'] =>
  mapping.outputMode === 'raw_message' ? 'raw' : 'fields'

const openModeTab = (mapping: KafkaTopicMapping) => {
  const id = modeTabId(mapping)
  if (!modeTabs.value.some((tab) => tab.id === id)) {
    modeTabs.value.push({
      id,
      type: modeTabType(mapping),
      title: mapping.name || mapping.topic,
      mappingId: String(mapping.id),
    })
  }
  selectedMapping.value = mapping
  activeMapping.value = mapping
  activeModeTabId.value = id
}

const closeModeTab = (tabId: string) => {
  const index = modeTabs.value.findIndex((tab) => tab.id === tabId)
  if (index < 0) return
  modeTabs.value = modeTabs.value.filter((tab) => tab.id !== tabId)
  if (activeModeTabId.value === tabId) {
    activeModeTabId.value = modeTabs.value[index - 1]?.id || modeTabs.value[index]?.id || ''
  }
}
```

- [ ] **步骤 2：让选择消费规则打开模式标签**

将当前：

```ts
const selectMapping = (mapping: KafkaTopicMapping) => {
  selectedMapping.value = mapping
  activeMapping.value = mapping
}
```

改为：

```ts
const selectMapping = (mapping: KafkaTopicMapping) => {
  openModeTab(mapping)
}
```

右键菜单的 `open` 和 `pullSamples` 分支继续调用 `selectMapping(mapping)`；`pullSamples` 后保留 `samplePullRequestId.value += 1`。

- [ ] **步骤 3：替换主内容模板**

导入公共组件和图标：

```ts
import WorkbenchTabBar from '@/components/workbench/WorkbenchTabBar.vue'
import IconTablerBraces from '~icons/tabler/braces'
```

将 `<main class="kafka-workbench__main">` 内部替换为：

```vue
<main class="kafka-workbench__main">
  <WorkbenchTabBar
    v-if="modeTabs.length > 0"
    v-model:active-id="activeModeTabId"
    :tabs="modeTabs.map((tab) => ({
      id: tab.id,
      title: tab.title,
      icon: tab.type === 'raw' ? IconTablerMessages : IconTablerBraces,
    }))"
    @close="closeModeTab"
  />

  <div class="kafka-workbench__content">
    <template v-if="modeTabs.length > 0">
      <template v-for="tab in modeTabs" :key="tab.id">
        <KafkaRawOutputPanel
          v-if="tab.type === 'raw' && getMappingById(tab.mappingId)"
          class="kafka-workbench__content-panel"
          :class="{ 'is-active': activeModeTabId === tab.id }"
          :project-id="projectId"
          :mapping="getMappingById(tab.mappingId)!"
          :pull-request-id="samplePullRequestId"
          @samples="handlePreviewSamples"
        />
        <KafkaFieldMappingPanel
          v-else-if="tab.type === 'fields' && getMappingById(tab.mappingId)"
          class="kafka-workbench__content-panel"
          :class="{ 'is-active': activeModeTabId === tab.id }"
          :project-id="projectId"
          :mapping="getMappingById(tab.mappingId)!"
          :samples="getPreviewSamples(tab.mappingId)"
          :pull-request-id="samplePullRequestId"
          @samples="handlePreviewSamples"
        />
      </template>
    </template>
    <div v-else class="kafka-workbench__placeholder">
      <IconTablerMessages />
      <strong>选择消费规则</strong>
      <span>整包规则可测试拉取样本，字段规则可通过 JSON 样例生成字段映射。</span>
    </div>
  </div>
</main>
```

新增：

```ts
const getMappingById = (mappingId: string | number) =>
  mappings.value.find((mapping) => String(mapping.id) === String(mappingId)) ||
  (activeMapping.value && String(activeMapping.value.id) === String(mappingId)
    ? activeMapping.value
    : null)
```

- [ ] **步骤 4：移除第三列 inspector**

删除模板中的：

```vue
<KafkaInspectorPanel
  :connection="inspectorConnection"
  :mapping="selectedMapping"
  :preview="activePreview"
/>
```

删除 `KafkaInspectorPanel` import 和 `inspectorConnection` computed。把 `.kafka-workbench` grid 从三列改为两列：

```css
.kafka-workbench {
  grid-template-columns: 260px minmax(0, 1fr);
}
```

增加 tab 内容切换样式：

```css
.kafka-workbench__content-panel {
  position: absolute;
  inset: 0;
  visibility: hidden;
  pointer-events: none;
  opacity: 0;
}

.kafka-workbench__content-panel.is-active {
  visibility: visible;
  pointer-events: auto;
  opacity: 1;
}

.kafka-workbench__content {
  position: relative;
}
```

- [ ] **步骤 5：删除规则时关闭对应标签**

在 `deleteMapping` 成功后补充：

```ts
modeTabs.value = modeTabs.value.filter((tab) => tab.mappingId !== String(mapping.id))
if (!modeTabs.value.some((tab) => tab.id === activeModeTabId.value)) {
  activeModeTabId.value = modeTabs.value[0]?.id || ''
}
```

在 connection id watch 中补充：

```ts
modeTabs.value = []
activeModeTabId.value = ''
```

- [ ] **步骤 6：运行类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 7：Commit**

```powershell
git add datacenter/src/components/kafka/KafkaWorkbench.vue
git commit -m "feat(kafka): 使用模式标签页承载工作区"
```

## 任务 6：整理整包数据点标签页

**文件：**

- 修改：`datacenter/src/components/kafka/KafkaRawOutputPanel.vue`

- [ ] **步骤 1：补充整包规则摘要与测试动作**

保留现有 `WorkbenchStreamToolbar` 和 `WorkbenchStreamMessageList`。在 header chips 中补充消费参数：

```vue
<WorkbenchStatusPill
  :label="mapping.rawOutputScope === 'full_message' ? '完整消息' : '消息体'"
  tone="info"
/>
<WorkbenchStatusPill :label="partitionLabel" tone="info" />
```

新增 computed：

```ts
const partitionLabel = computed(() =>
  props.mapping.partitionMode === 'single'
    ? `partition ${props.mapping.partition ?? 0}`
    : '全部分区',
)
```

把 toolbar `title` 改为：

```vue
title="整包样本测试"
```

- [ ] **步骤 2：统一测试拉取结果文案**

在 `pullSamples` 成功后补充：

```ts
ElMessage.success(`已拉取 ${samples.value.length} 条样本`)
```

如果样本为空，仍然成功但提示：

```ts
if (samples.value.length === 0) {
  ElMessage.warning('本次未拉取到样本，可调整起始位置、样本上限或超时后重试')
  return
}
```

- [ ] **步骤 3：运行类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 4：Commit**

```powershell
git add datacenter/src/components/kafka/KafkaRawOutputPanel.vue
git commit -m "feat(kafka): 优化整包数据点测试标签"
```

## 任务 7：改造字段映射标签页和样例编辑器

**文件：**

- 修改：`datacenter/src/components/kafka/KafkaFieldMappingPanel.vue`
- 使用：`datacenter/src/components/kafka/kafkaSampleFields.ts`
- 使用：`datacenter/src/components/workbench/WorkbenchJsonSampleEditor.vue`
- 使用：`datacenter/src/components/shared/BulkActionBar.vue`

- [ ] **步骤 1：引入公共组件和样例模型**

在 script 中新增 import：

```ts
import BulkActionBar from '@/components/shared/BulkActionBar.vue'
import WorkbenchJsonSampleEditor from '@/components/workbench/WorkbenchJsonSampleEditor.vue'
import {
  inferKafkaSampleFields,
  normalizeKafkaSampleEditorText,
  type KafkaSampleFieldCandidate,
} from './kafkaSampleFields'
```

删除 `MonacoEditor` 直接 import，字段映射面板通过 `WorkbenchJsonSampleEditor` 使用 Monaco。

新增状态：

```ts
const sampleEditorVisible = ref(true)
const sampleEditorText = ref('')
const sampleParseError = ref('')
const selectedCandidatePaths = ref<Set<string>>(new Set())
const candidateRows = ref<KafkaSampleFieldCandidate[]>([])
```

- [ ] **步骤 2：把弹窗式粘贴样本替换为嵌入式编辑器**

删除模板中的 `sampleDialogVisible` 对应 `DcDialog`。在表格右侧增加可收起样例编辑器区域，结构参考：

```vue
<section class="kafka-variable-panel__body" :class="{ 'has-editor': sampleEditorVisible }">
  <div class="kafka-variable-panel__result">
    <!-- 保留字段映射表格和分页 -->
  </div>
  <div v-if="sampleEditorVisible" class="kafka-variable-panel__editor">
    <WorkbenchJsonSampleEditor
      ref="sampleEditorRef"
      v-model="sampleEditorText"
      :fill-latest-disabled="props.samples.length === 0"
      :error="sampleParseError"
      @fill-latest="fillEditorFromLatestSample"
      @parse="parseEditorFields"
    />
  </div>
</section>
```

- [ ] **步骤 3：实现样例编辑器动作**

新增：

```ts
const sampleEditorRef = ref<InstanceType<typeof WorkbenchJsonSampleEditor> | null>(null)

const fillEditorFromLatestSample = () => {
  const sample = props.samples[0]
  if (!sample) {
    ElMessage.warning('暂无最近样本，请先测试拉取或手动粘贴 JSON')
    return
  }
  sampleEditorText.value = normalizeKafkaSampleEditorText(sample)
}

const parseEditorFields = () => {
  sampleParseError.value = ''
  try {
    const parsed = JSON.parse(sampleEditorText.value)
    candidateRows.value = inferKafkaSampleFields(
      [{ value: parsed }],
      fields.value.map((field) => field.valuePath),
    )
    selectedCandidatePaths.value = new Set(
      candidateRows.value
        .filter((candidate) => !candidate.exists)
        .map((candidate) => candidate.path),
    )
    candidates.value = candidateRows.value
    ElMessage.success(`已解析 ${candidateRows.value.length} 个字段候选`)
  } catch {
    sampleParseError.value = 'JSON 样例格式无效'
  }
}
```

保留 `pullSamples`，成功后补充：

```ts
const firstSample = preview.samples?.[0]
if (firstSample) {
  sampleEditorText.value = normalizeKafkaSampleEditorText(firstSample)
  candidateRows.value = inferKafkaSampleFields(
    preview.samples || [],
    fields.value.map((field) => field.valuePath),
  )
}
```

- [ ] **步骤 4：改造从样本生成逻辑**

将 `createFromSamples` 的候选来源改为优先使用编辑器候选：

```ts
const createFromSamples = async () => {
  const selectedPaths = selectedCandidatePaths.value
  const sourceCandidates =
    candidateRows.value.length > 0
      ? candidateRows.value
      : inferKafkaSampleFields(
          props.samples,
          fields.value.map((field) => field.valuePath),
        )
  const nextCandidates = sourceCandidates.filter(
    (candidate) =>
      !candidate.exists && (selectedPaths.size === 0 || selectedPaths.has(candidate.path)),
  )

  if (nextCandidates.length === 0) {
    ElMessage.warning('暂无可保存的字段候选，请先解析 JSON 样例或测试拉取样本')
    return
  }

  batchSaving.value = true
  try {
    await dataAPI.createKafkaFieldsBatch(
      props.projectId,
      props.mapping.id,
      nextCandidates.map((candidate) => ({
        name: candidate.name,
        valuePath: candidate.path,
        dataType: candidate.dataType,
        enabled: true,
        groupId:
          selectedGroupId.value && selectedGroupId.value !== '__ungrouped'
            ? selectedGroupId.value
            : null,
      })),
    )
    ElMessage.success('字段映射已保存')
    await reloadAfterMutation()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存字段映射失败'))
  } finally {
    batchSaving.value = false
  }
}
```

- [ ] **步骤 5：增加候选字段表格区**

在字段映射表格上方或编辑器下方增加候选字段小表格：

```vue
<el-table
  v-if="candidateRows.length > 0"
  class="kafka-variable-panel__candidate-table"
  :data="candidateRows"
  max-height="220"
  row-key="path"
  @selection-change="handleCandidateSelectionChange"
>
  <el-table-column type="selection" width="42" :selectable="(row) => !row.exists" />
  <el-table-column prop="name" label="映射名称" min-width="120" />
  <el-table-column prop="path" label="字段路径" min-width="150" show-overflow-tooltip />
  <el-table-column prop="dataType" label="类型" width="88" />
  <el-table-column label="样例值" min-width="120" show-overflow-tooltip>
    <template #default="{ row }">
      <span class="kafka-variable-panel__mono">{{ formatLastValue(row.sampleValue) }}</span>
    </template>
  </el-table-column>
  <el-table-column label="状态" width="88">
    <template #default="{ row }">
      <WorkbenchStatusPill :label="row.exists ? '已存在' : '候选'" :tone="row.exists ? 'neutral' : 'info'" />
    </template>
  </el-table-column>
</el-table>
```

新增：

```ts
const handleCandidateSelectionChange = (rows: KafkaSampleFieldCandidate[]) => {
  selectedCandidatePaths.value = new Set(rows.map((row) => row.path))
}
```

- [ ] **步骤 6：补充布局样式**

将 `.kafka-variable-panel__body` 改为可分栏布局：

```css
.kafka-variable-panel__body {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
}

.kafka-variable-panel__body.has-editor {
  grid-template-columns: minmax(0, 1fr) minmax(340px, 420px);
}

.kafka-variable-panel__result,
.kafka-variable-panel__editor {
  min-height: 0;
  min-width: 0;
}

.kafka-variable-panel__result {
  display: flex;
  flex-direction: column;
}

.kafka-variable-panel__editor {
  display: grid;
  grid-template-rows: auto minmax(220px, 1fr) auto;
  gap: 8px;
  padding: 10px;
  border-left: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.kafka-variable-panel__candidate-table {
  margin-bottom: 8px;
}
```

- [ ] **步骤 7：运行测试和类型检查**

运行：

```powershell
pnpm --filter datacenter test -- tests/kafka-sample-fields.test.ts
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 8：Commit**

```powershell
git add datacenter/src/components/kafka/KafkaFieldMappingPanel.vue datacenter/src/components/kafka/kafkaSampleFields.ts datacenter/tests/kafka-sample-fields.test.ts
git commit -m "feat(kafka): 增加字段映射样例编辑器"
```

## 任务 8：最终验证和人工验收

**文件：**

- 不新增文件。

- [ ] **步骤 1：运行前端测试**

运行：

```powershell
pnpm --filter datacenter test -- tests/kafka-sample-fields.test.ts tests/kafka-topic-tree-model.test.ts
```

预期：PASS。

- [ ] **步骤 2：运行前端类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 3：运行后端相关测试**

运行：

```powershell
Set-Location data_service
go test ./internal/service -run "TestNormalizeKafka|TestNormalizeUpdateTopicMapping" -count=1
```

预期：PASS。

- [ ] **步骤 4：人工验收新建/编辑弹窗**

验收路径：

1. 打开 Kafka 工作台，左侧树样式和右键菜单保持原样。
2. 点击新建消费规则，确认输出模式是单选卡片或 radio。
3. 新建整包数据点规则，保存后重新右键编辑，确认输出模式不可切换。
4. 新建字段映射规则，保存后重新右键编辑，确认输出模式不可切换。
5. 编辑 Topic、消费组、分区、解码、样本上限，确认可保存并回显。

- [ ] **步骤 5：人工验收模式标签页**

验收路径：

1. 点击整包模式消费规则，确认右侧主工作区打开整包数据点标签页。
2. 点击字段映射模式消费规则，确认打开字段映射标签页。
3. 重复点击同一消费规则，确认激活已有标签页，不重复打开。
4. 删除消费规则后，确认对应标签页关闭。
5. 切换 Kafka 接入源后，确认标签页和样本缓存清空。

- [ ] **步骤 6：人工验收整包样本**

验收路径：

1. 在整包标签页点击测试拉取。
2. 成功时展示 payload、partition、offset、timestamp。
3. 复制 Payload 可用。
4. 清空样本可用。
5. 拉取失败时保留上一次成功样本并展示错误消息。

- [ ] **步骤 7：人工验收字段映射样例编辑器**

验收路径：

1. 在字段映射标签页粘贴 JSON 样例。
2. 点击解析字段，确认对象字段被自动展平。
3. 粘贴包含数组的 JSON，确认产生 `items.0.xxx` 这类普通路径，不出现数组拆分规则表单。
4. 勾选候选字段并保存，确认字段映射表刷新。
5. 点击测试拉取，确认最近样本可填入样例编辑器。

- [ ] **步骤 8：最终 Commit**

若前面任务已经逐项 commit，本步骤只检查工作区：

```powershell
git status --short
```

预期：只剩用户已有的未跟踪或无关文件；不要把 `data_service/tmp/` 之类非本任务文件纳入提交。

## 自检结果

- 规格覆盖：新建/编辑模式锁定、左侧树不动、两类动态模式标签页、整包测试拉取、字段映射样例编辑器、JSON 自动展平、无数组拆分规则、验证路径均有任务覆盖。
- 公共组件：计划复用 `WorkbenchStreamToolbar`、`WorkbenchStreamMessageList`、`WorkbenchStatusPill`、`MonacoEditor`、`BulkActionBar`，并抽取 `WorkbenchTabBar`、`WorkbenchJsonSampleEditor`；未把 Kafka 业务语义放进公共组件。
- 类型一致性：前端统一使用 `raw_message`、`field_mapping`、`KafkaSampleFieldCandidate`、`modeTabs`、`activeModeTabId`；后端只增加 `outputMode` 更新防线，不改接口字段名。
- 范围控制：不实现 Kafka 诊断、metadata、consumer lag、数组拆分、JSONPath 或脚本解析。
