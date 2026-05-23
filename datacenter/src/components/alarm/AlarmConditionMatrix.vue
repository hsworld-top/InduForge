<template>
  <section class="alarm-condition-modules">
    <div v-if="targetKind === 'unknown'" class="alarm-condition-modules__empty">
      选择目标点后，会按数据类型展示可配置的报警模板。
    </div>

    <template v-else-if="targetKind === 'number'">
      <section class="alarm-condition-modules__module">
        <ModuleHeader title="阈值报警" description="对低低、低、高、高高四个限值分别配置" />
        <div class="alarm-condition-modules__table is-threshold">
          <div class="alarm-condition-modules__row is-head">
            <ColumnLabel label="启用" tip="开启后该行条件才参与报警判断。" />
            <ColumnLabel label="类型" tip="阈值方向：低低、低、高、高高。" />
            <ColumnLabel label="报警限" tip="报警限带的中心值。实际触发和恢复会结合死区宽度计算。" />
            <ColumnLabel label="报警文本" tip="报警触发后展示的短文本。" />
            <ColumnLabel label="等级" tip="报警严重程度，用于颜色、筛选和处置优先级。" />
            <ColumnLabel label="优先级" tip="触发该条件时生成报警事件的初始优先级，范围 1～999；1 最低，999 最高。持续未恢复时，升级间隔用于等级阶段升级，重复推送间隔用于固定间隔提醒。" />
            <ColumnLabel label="触发延时" tip="条件成立后等待多久才报警，填 0 立即报警。" />
            <ColumnLabel
              label="死区宽度"
              tip="在报警限两侧形成同宽缓冲带。高限达到报警限+死区触发，回落到报警限-死区恢复；低限达到报警限-死区触发，回升到报警限+死区恢复。带内保持当前报警状态。"
            />
          </div>
          <div
            v-for="item in thresholdRows"
            :key="item.type"
            class="alarm-condition-modules__row"
            :class="{ 'is-disabled': !conditionFor(item.type).isEnabled }"
          >
            <input
              type="checkbox"
              :checked="conditionFor(item.type).isEnabled"
              @change="patchCondition(item.type, { isEnabled: checked($event) })"
            />
            <strong>{{ item.label }}</strong>
            <input
              type="number"
              step="any"
              :value="numberParam(conditionFor(item.type), 'limit')"
              @input="patchParam(item.type, 'limit', numberValue($event))"
            />
            <input
              type="text"
              :value="conditionFor(item.type).name"
              @input="patchCondition(item.type, { name: inputValue($event) })"
            />
            <SeveritySelect
              :model-value="conditionFor(item.type).severity"
              @update:model-value="patchSeverity(item.type, $event)"
            />
            <input
              type="number"
              min="1"
              max="999"
              step="1"
              :value="priorityParam(conditionFor(item.type))"
              @input="patchParam(item.type, 'priority', priorityValue($event))"
            />
            <input
              type="number"
              min="0"
              step="1"
              :value="numberParam(conditionFor(item.type), 'durationMs')"
              @input="patchParam(item.type, 'durationMs', optionalNumberValue($event))"
            />
            <input
              type="number"
              min="0"
              step="any"
              :value="numberParam(conditionFor(item.type), 'hysteresis')"
              @input="patchParam(item.type, 'hysteresis', optionalNumberValue($event))"
            />
          </div>
        </div>
      </section>

      <section class="alarm-condition-modules__module">
        <ModuleHeader title="偏差报警" description="相对基准值发生高偏差或低偏差时触发" />
        <div class="alarm-condition-modules__table is-deviation">
          <div class="alarm-condition-modules__row is-head">
            <ColumnLabel label="启用" tip="开启后该行条件才参与报警判断。" />
            <ColumnLabel label="类型" tip="偏差方向：高偏差或低偏差。" />
            <ColumnLabel label="基准值" tip="用于计算偏差的参考值。" />
            <ColumnLabel label="偏差值" tip="当前值相对基准值偏离超过该数值时触发。" />
            <ColumnLabel label="报警文本" tip="报警触发后展示的短文本。" />
            <ColumnLabel label="等级" tip="报警严重程度，用于颜色、筛选和处置优先级。" />
            <ColumnLabel label="优先级" tip="触发该条件时生成报警事件的初始优先级，范围 1～999；1 最低，999 最高。持续未恢复时，升级间隔用于等级阶段升级，重复推送间隔用于固定间隔提醒。" />
            <ColumnLabel label="触发延时" tip="条件成立后等待多久才报警，填 0 立即报警。" />
          </div>
          <div
            v-for="item in deviationRows"
            :key="item.type"
            class="alarm-condition-modules__row"
            :class="{ 'is-disabled': !conditionFor(item.type).isEnabled }"
          >
            <input type="checkbox" :checked="conditionFor(item.type).isEnabled" @change="patchCondition(item.type, { isEnabled: checked($event) })" />
            <strong>{{ item.label }}</strong>
            <input type="number" step="any" :value="numberParam(conditionFor(item.type), 'baselineValue')" @input="patchParam(item.type, 'baselineValue', numberValue($event))" />
            <input type="number" step="any" :value="numberParam(conditionFor(item.type), 'limit')" @input="patchParam(item.type, 'limit', numberValue($event))" />
            <input type="text" :value="conditionFor(item.type).name" @input="patchCondition(item.type, { name: inputValue($event) })" />
            <SeveritySelect :model-value="conditionFor(item.type).severity" @update:model-value="patchSeverity(item.type, $event)" />
            <input type="number" min="1" max="999" step="1" :value="priorityParam(conditionFor(item.type))" @input="patchParam(item.type, 'priority', priorityValue($event))" />
            <input type="number" min="0" step="1" :value="numberParam(conditionFor(item.type), 'durationMs')" @input="patchParam(item.type, 'durationMs', optionalNumberValue($event))" />
          </div>
        </div>
      </section>

      <section class="alarm-condition-modules__module">
        <ModuleHeader title="变化率报警" description="对上升过快、下降过快分别配置统计窗口" />
        <div class="alarm-condition-modules__table is-rate">
          <div class="alarm-condition-modules__row is-head">
            <ColumnLabel label="启用" tip="开启后该行条件才参与报警判断。" />
            <ColumnLabel label="类型" tip="变化方向：上升过快或下降过快。" />
            <ColumnLabel label="变化阈值" tip="统计窗口内变化量超过该数值时触发。" />
            <ColumnLabel label="统计窗口 ms" tip="用于计算变化率的时间窗口。" />
            <ColumnLabel label="报警文本" tip="报警触发后展示的短文本。" />
            <ColumnLabel label="等级" tip="报警严重程度，用于颜色、筛选和处置优先级。" />
            <ColumnLabel label="优先级" tip="触发该条件时生成报警事件的初始优先级，范围 1～999；1 最低，999 最高。持续未恢复时，升级间隔用于等级阶段升级，重复推送间隔用于固定间隔提醒。" />
          </div>
          <div
            v-for="item in rateRows"
            :key="item.key"
            class="alarm-condition-modules__row"
            :class="{ 'is-disabled': !conditionFor('rate_of_change', item.key).isEnabled }"
          >
            <input type="checkbox" :checked="conditionFor('rate_of_change', item.key).isEnabled" @change="patchCondition('rate_of_change', { isEnabled: checked($event) }, item.key)" />
            <strong>{{ item.label }}</strong>
            <input type="number" step="any" :value="numberParam(conditionFor('rate_of_change', item.key), 'limit')" @input="patchParam('rate_of_change', 'limit', numberValue($event), item.key)" />
            <input type="number" min="0" step="1" :value="numberParam(conditionFor('rate_of_change', item.key), 'windowMs')" @input="patchParam('rate_of_change', 'windowMs', numberValue($event), item.key)" />
            <input type="text" :value="conditionFor('rate_of_change', item.key).name" @input="patchCondition('rate_of_change', { name: inputValue($event) }, item.key)" />
            <SeveritySelect :model-value="conditionFor('rate_of_change', item.key).severity" @update:model-value="patchSeverity('rate_of_change', $event, item.key)" />
            <input type="number" min="1" max="999" step="1" :value="priorityParam(conditionFor('rate_of_change', item.key))" @input="patchParam('rate_of_change', 'priority', priorityValue($event), item.key)" />
          </div>
        </div>
      </section>

      <ExpressionModule
        :condition="conditionFor('cel')"
        @patch="patchCondition('cel', $event)"
        @patch-param="(field, value) => patchParam('cel', field, value)"
        @patch-severity="patchSeverity('cel', $event)"
      />
    </template>

    <template v-else-if="targetKind === 'boolean'">
      <section class="alarm-condition-modules__module">
        <ModuleHeader title="状态报警" description="当前值为 False 或 True 时触发" />
        <div class="alarm-condition-modules__table is-bool-state">
          <div class="alarm-condition-modules__row is-head">
            <ColumnLabel label="启用" tip="开启后该行条件才参与报警判断。" />
            <ColumnLabel label="状态" tip="当布尔点位值等于该状态时触发。" />
            <ColumnLabel label="报警文本" tip="报警触发后展示的短文本。" />
            <ColumnLabel label="等级" tip="报警严重程度，用于颜色、筛选和处置优先级。" />
            <ColumnLabel label="优先级" tip="触发该条件时生成报警事件的初始优先级，范围 1～999；1 最低，999 最高。持续未恢复时，升级间隔用于等级阶段升级，重复推送间隔用于固定间隔提醒。" />
            <ColumnLabel label="触发延时" tip="条件成立后等待多久才报警，填 0 立即报警。" />
          </div>
          <div v-for="item in boolStateRows" :key="item.key" class="alarm-condition-modules__row" :class="{ 'is-disabled': !conditionFor('bool_equal', item.key).isEnabled }">
            <input type="checkbox" :checked="conditionFor('bool_equal', item.key).isEnabled" @change="patchCondition('bool_equal', { isEnabled: checked($event) }, item.key)" />
            <strong>{{ item.label }}</strong>
            <input type="text" :value="conditionFor('bool_equal', item.key).name" @input="patchCondition('bool_equal', { name: inputValue($event) }, item.key)" />
            <SeveritySelect :model-value="conditionFor('bool_equal', item.key).severity" @update:model-value="patchSeverity('bool_equal', $event, item.key)" />
            <input type="number" min="1" max="999" step="1" :value="priorityParam(conditionFor('bool_equal', item.key))" @input="patchParam('bool_equal', 'priority', priorityValue($event), item.key)" />
            <input type="number" min="0" step="1" :value="numberParam(conditionFor('bool_equal', item.key), 'durationMs')" @input="patchParam('bool_equal', 'durationMs', optionalNumberValue($event), item.key)" />
          </div>
        </div>
      </section>

      <section class="alarm-condition-modules__module">
        <ModuleHeader title="变化报警" description="值从 True/False 发生跳变时触发" />
        <div class="alarm-condition-modules__table is-bool-transition">
          <div class="alarm-condition-modules__row is-head">
            <ColumnLabel label="启用" tip="开启后该行条件才参与报警判断。" />
            <ColumnLabel label="跳变" tip="布尔值从一个状态变化到另一个状态时触发。" />
            <ColumnLabel label="报警文本" tip="报警触发后展示的短文本。" />
            <ColumnLabel label="等级" tip="报警严重程度，用于颜色、筛选和处置优先级。" />
            <ColumnLabel label="优先级" tip="触发该条件时生成报警事件的初始优先级，范围 1～999；1 最低，999 最高。持续未恢复时，升级间隔用于等级阶段升级，重复推送间隔用于固定间隔提醒。" />
            <ColumnLabel label="触发延时" tip="状态变化后等待多久才报警，填 0 立即报警。" />
          </div>
          <div v-for="item in boolTransitionRows" :key="item.key" class="alarm-condition-modules__row" :class="{ 'is-disabled': !conditionFor('bool_transition', item.key).isEnabled }">
            <input type="checkbox" :checked="conditionFor('bool_transition', item.key).isEnabled" @change="patchCondition('bool_transition', { isEnabled: checked($event) }, item.key)" />
            <strong>{{ item.label }}</strong>
            <input type="text" :value="conditionFor('bool_transition', item.key).name" @input="patchCondition('bool_transition', { name: inputValue($event) }, item.key)" />
            <SeveritySelect :model-value="conditionFor('bool_transition', item.key).severity" @update:model-value="patchSeverity('bool_transition', $event, item.key)" />
            <input type="number" min="1" max="999" step="1" :value="priorityParam(conditionFor('bool_transition', item.key))" @input="patchParam('bool_transition', 'priority', priorityValue($event), item.key)" />
            <input type="number" min="0" step="1" :value="numberParam(conditionFor('bool_transition', item.key), 'durationMs')" @input="patchParam('bool_transition', 'durationMs', optionalNumberValue($event), item.key)" />
          </div>
        </div>
      </section>

      <ExpressionModule
        :condition="conditionFor('cel')"
        @patch="patchCondition('cel', $event)"
        @patch-param="(field, value) => patchParam('cel', field, value)"
        @patch-severity="patchSeverity('cel', $event)"
      />
    </template>

    <template v-else-if="targetKind === 'string'">
      <section class="alarm-condition-modules__module">
        <ModuleHeader title="文本报警" description="按等于、不等于、包含或正则匹配触发" />
        <div class="alarm-condition-modules__table is-string">
          <div class="alarm-condition-modules__row is-head">
            <ColumnLabel label="启用" tip="开启后该行条件才参与报警判断。" />
            <ColumnLabel label="类型" tip="文本判断方式：等于、不等于、包含或正则。" />
            <ColumnLabel label="匹配值" tip="用于和点位文本值比较的目标内容。" />
            <ColumnLabel label="报警文本" tip="报警触发后展示的短文本。" />
            <ColumnLabel label="等级" tip="报警严重程度，用于颜色、筛选和处置优先级。" />
            <ColumnLabel label="优先级" tip="触发该条件时生成报警事件的初始优先级，范围 1～999；1 最低，999 最高。持续未恢复时，升级间隔用于等级阶段升级，重复推送间隔用于固定间隔提醒。" />
            <ColumnLabel label="触发延时" tip="条件成立后等待多久才报警，填 0 立即报警。" />
          </div>
          <div v-for="item in stringRows" :key="item.type" class="alarm-condition-modules__row" :class="{ 'is-disabled': !conditionFor(item.type).isEnabled }">
            <input type="checkbox" :checked="conditionFor(item.type).isEnabled" @change="patchCondition(item.type, { isEnabled: checked($event) })" />
            <strong>{{ item.label }}</strong>
            <input type="text" :value="stringMatchValue(conditionFor(item.type))" @input="patchStringMatch(item.type, inputValue($event))" />
            <input type="text" :value="conditionFor(item.type).name" @input="patchCondition(item.type, { name: inputValue($event) })" />
            <SeveritySelect :model-value="conditionFor(item.type).severity" @update:model-value="patchSeverity(item.type, $event)" />
            <input type="number" min="1" max="999" step="1" :value="priorityParam(conditionFor(item.type))" @input="patchParam(item.type, 'priority', priorityValue($event))" />
            <input type="number" min="0" step="1" :value="numberParam(conditionFor(item.type), 'durationMs')" @input="patchParam(item.type, 'durationMs', optionalNumberValue($event))" />
          </div>
        </div>
      </section>

      <ExpressionModule
        :condition="conditionFor('cel')"
        @patch="patchCondition('cel', $event)"
        @patch-param="(field, value) => patchParam('cel', field, value)"
        @patch-severity="patchSeverity('cel', $event)"
      />
    </template>

    <ExpressionModule
      v-else
      :condition="conditionFor('cel')"
      @patch="patchCondition('cel', $event)"
      @patch-param="(field, value) => patchParam('cel', field, value)"
      @patch-severity="patchSeverity('cel', $event)"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, nextTick, ref, watch } from "vue";
import type {
  AlarmCondition,
  AlarmConditionType,
  AlarmPolicyMode,
  AlarmSeverity,
  AlarmTargetRef,
} from "@/api/schemas/alarm.schema";
import { createThresholdCondition } from "@/components/alarm/alarmPolicyModel";
import MonacoEditor from "@/components/MonacoEditor.vue";

type ConditionKey = string;
type TargetKind = "number" | "boolean" | "string" | "object" | "mixed" | "unknown";

type RowConfig = {
  type: AlarmConditionType;
  key?: ConditionKey;
  label: string;
  params?: Record<string, unknown>;
  defaultName?: string;
};

const props = withDefaults(
  defineProps<{
    conditions: AlarmCondition[];
    mode?: AlarmPolicyMode;
    targets?: AlarmTargetRef[];
  }>(),
  {
    mode: "per_target",
    targets: () => [],
  },
);

const emit = defineEmits<{
  update: [conditions: AlarmCondition[]];
}>();

const thresholdRows: RowConfig[] = [
  { type: "LL", label: "低低", defaultName: "低低限" },
  { type: "L", label: "低", defaultName: "低限" },
  { type: "H", label: "高", defaultName: "高限" },
  { type: "HH", label: "高高", defaultName: "高高限" },
];

const deviationRows: RowConfig[] = [
  { type: "deviation_high", label: "高偏差", defaultName: "高偏差" },
  { type: "deviation_low", label: "低偏差", defaultName: "低偏差" },
];

const rateRows: RowConfig[] = [
  { type: "rate_of_change", key: "up", label: "上升过快", defaultName: "上升过快", params: { direction: "up" } },
  { type: "rate_of_change", key: "down", label: "下降过快", defaultName: "下降过快", params: { direction: "down" } },
];

const boolStateRows: RowConfig[] = [
  { type: "bool_equal", key: "false", label: "False 报警", defaultName: "关闭报警", params: { expected: false } },
  { type: "bool_equal", key: "true", label: "True 报警", defaultName: "打开报警", params: { expected: true } },
];

const boolTransitionRows: RowConfig[] = [
  { type: "bool_transition", key: "true_false", label: "True -> False", defaultName: "开到关", params: { from: true, to: false } },
  { type: "bool_transition", key: "false_true", label: "False -> True", defaultName: "关到开", params: { from: false, to: true } },
];

const stringRows: RowConfig[] = [
  { type: "string_equal", label: "等于", defaultName: "文本等于" },
  { type: "string_not_equal", label: "不等于", defaultName: "文本不等于" },
  { type: "string_contains", label: "包含", defaultName: "文本包含" },
  { type: "string_regex", label: "正则", defaultName: "正则匹配" },
];

const severityOptions: Array<{ value: AlarmSeverity; label: string; priority: number }> = [
  { value: "info", label: "提示", priority: 100 },
  { value: "warning", label: "警告", priority: 300 },
  { value: "major", label: "重要", priority: 600 },
  { value: "critical", label: "紧急", priority: 900 },
];

const targetKind = computed<TargetKind>(() => {
  if (props.mode === "derived") {
    return "number";
  }
  if (!props.targets.length) {
    return "unknown";
  }
  if (props.targets.every((target) => isNumericType(target.dataType))) {
    return "number";
  }
  if (props.targets.every((target) => isBooleanType(target.dataType))) {
    return "boolean";
  }
  if (props.targets.every((target) => isStringType(target.dataType))) {
    return "string";
  }
  if (props.targets.every((target) => isObjectType(target.dataType))) {
    return "object";
  }
  return "mixed";
});

const allRowConfigs = computed<RowConfig[]>(() => [
  ...thresholdRows,
  ...deviationRows,
  ...rateRows,
  ...boolStateRows,
  ...boolTransitionRows,
  ...stringRows,
  { type: "cel", label: "判断表达式", defaultName: "判断表达式" },
]);

const update = (conditions: AlarmCondition[]) => emit("update", conditions);

const conditionKey = (type: AlarmConditionType, key?: ConditionKey) => key ? `${type}:${key}` : type;

const existingByKey = computed(() => {
  const result = new Map<string, AlarmCondition>();
  for (const condition of props.conditions) {
    const key = conditionKey(condition.type, condition.params?.key as string | undefined);
    if (!result.has(key)) {
      result.set(key, condition);
    }
  }
  return result;
});

const conditionFor = (type: AlarmConditionType, key?: ConditionKey) => {
  const existing = existingByKey.value.get(conditionKey(type, key));
  return existing ?? createDraftCondition(type, key);
};

const createDraftCondition = (type: AlarmConditionType, key?: ConditionKey): AlarmCondition => {
  const row = allRowConfigs.value.find((item) => item.type === type && item.key === key)
    ?? allRowConfigs.value.find((item) => item.type === type);
  const base = createThresholdCondition(type);
  const severity = base.severity;
  const priority = defaultPriority(severity);
  const params = {
    ...base.params,
    ...(row?.params || {}),
    key,
    priority,
  };
  if (type === "deviation_high" || type === "deviation_low") {
    params.baselineValue = 0;
  }
  if (type === "cel") {
    params.expression = "value > 0";
  }
  return {
    ...base,
    name: row?.defaultName || base.name,
    isEnabled: false,
    params,
  };
};

const upsertCondition = (next: AlarmCondition) => {
  const key = conditionKey(next.type, next.params?.key as string | undefined);
  const exists = props.conditions.some(
    (condition) => conditionKey(condition.type, condition.params?.key as string | undefined) === key,
  );
  if (exists) {
    update(
      props.conditions.map((condition) =>
        conditionKey(condition.type, condition.params?.key as string | undefined) === key
          ? next
          : condition,
      ),
    );
    return;
  }
  update([...props.conditions, next]);
};

const patchCondition = (type: AlarmConditionType, patch: Partial<AlarmCondition>, key?: ConditionKey) => {
  const current = conditionFor(type, key);
  upsertCondition({ ...current, ...patch, params: { ...current.params, ...(patch.params || {}) } });
};

const patchParam = (type: AlarmConditionType, field: string, value: unknown, key?: ConditionKey) => {
  const current = conditionFor(type, key);
  const params = { ...current.params };
  if (value === undefined || value === "") {
    delete params[field];
  } else {
    params[field] = value;
  }
  upsertCondition({ ...current, params });
};

const patchSeverity = (type: AlarmConditionType, severity: AlarmSeverity, key?: ConditionKey) => {
  const current = conditionFor(type, key);
  upsertCondition({
    ...current,
    severity,
    params: { ...current.params, priority: defaultPriority(severity) },
  });
};

const patchStringMatch = (type: AlarmConditionType, value: string) => {
  patchParam(type, type === "string_regex" ? "pattern" : "expected", value);
};

const checked = (event: Event) => (event.target as HTMLInputElement).checked;
const inputValue = (event: Event) =>
  (event.target as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement).value;
const numberValue = (event: Event) => Number(inputValue(event));
const optionalNumberValue = (event: Event) => {
  const value = inputValue(event);
  return value === "" ? undefined : Number(value);
};
const priorityValue = (event: Event) => {
  const value = Math.round(Number(inputValue(event)) || 1);
  return Math.min(999, Math.max(1, value));
};

const numberParam = (condition: AlarmCondition, field: string) => {
  const value = condition.params[field];
  return typeof value === "number" && Number.isFinite(value) ? String(value) : "";
};

const textParam = (condition: AlarmCondition, field: string) => {
  const value = condition.params[field];
  return typeof value === "string" ? value : "";
};

const priorityParam = (condition: AlarmCondition) => {
  const value = condition.params.priority;
  return typeof value === "number" && Number.isFinite(value)
    ? String(value)
    : String(defaultPriority(condition.severity));
};

const stringMatchValue = (condition: AlarmCondition) =>
  condition.type === "string_regex"
    ? textParam(condition, "pattern")
    : textParam(condition, "expected");

const defaultPriority = (severity: AlarmSeverity) =>
  severityOptions.find((item) => item.value === severity)?.priority ?? 300;

const isNumericType = (dataType: string) =>
  ["number", "integer", "int", "float", "double", "decimal"].includes(
    dataType.trim().toLowerCase(),
  );

const isBooleanType = (dataType: string) =>
  ["boolean", "bool"].includes(dataType.trim().toLowerCase());

const isStringType = (dataType: string) =>
  ["string", "text"].includes(dataType.trim().toLowerCase());

const isObjectType = (dataType: string) =>
  ["object", "json"].includes(dataType.trim().toLowerCase());

const ModuleHeader = defineComponent({
  props: {
    title: { type: String, required: true },
    description: { type: String, required: true },
  },
  setup(componentProps) {
    return () => h("header", { class: "alarm-condition-modules__module-head" }, [
      h("strong", componentProps.title),
      h("button", {
        type: "button",
        class: "alarm-condition-modules__help",
        title: componentProps.description,
        "aria-label": `${componentProps.title}说明`,
      }, "?"),
    ]);
  },
});

const ColumnLabel = defineComponent({
  props: {
    label: { type: String, required: true },
    tip: { type: String, required: true },
  },
  setup(componentProps) {
    return () => h("span", {
      class: "alarm-condition-modules__column-label",
      title: componentProps.tip,
    }, componentProps.label);
  },
});

const SeveritySelect = defineComponent({
  props: {
    modelValue: { type: String, required: true },
  },
  emits: ["update:modelValue"],
  setup(componentProps, { emit: componentEmit }) {
    return () => h("select", {
      value: componentProps.modelValue,
      onChange: (event: Event) => componentEmit("update:modelValue", (event.target as HTMLSelectElement).value),
    }, severityOptions.map((item) => h("option", { value: item.value }, item.label)));
  },
});

type ExpressionDiagnostic = {
  severity?: string;
  message: string;
  line: number;
  column: number;
  endLine?: number;
  endColumn?: number;
  source?: string;
};

const expressionEditorOptions = {
  lineNumbers: "off",
  glyphMargin: false,
  folding: false,
  lineDecorationsWidth: 0,
  lineNumbersMinChars: 0,
  minimap: { enabled: false },
  scrollbar: {
    vertical: "hidden",
    horizontal: "auto",
    alwaysConsumeMouseWheel: false,
  },
  overviewRulerLanes: 0,
  hideCursorInOverviewRuler: true,
  renderLineHighlight: "none",
  fontSize: 12,
  tabSize: 2,
  wordWrap: "on",
  scrollBeyondLastLine: false,
  automaticLayout: true,
};

// 对表达式做前端快速校验，保证明显的脚本语句和未闭合符号能在编辑时立即反馈。
const validateExpression = (value: string): ExpressionDiagnostic[] => {
  const diagnostics: ExpressionDiagnostic[] = [];
  const text = value.trim();
  if (!text) {
    diagnostics.push({
      severity: "warning",
      message: "表达式为空，启用后不会产生有效判断。",
      line: 1,
      column: 1,
      endColumn: 2,
      source: "alarm-expression",
    });
    return diagnostics;
  }

  const statementMatch = text.match(/\b(function|return|throw|const|let|var|class|import|export)\b|;/);
  if (statementMatch?.index !== undefined) {
    diagnostics.push({
      message: "这里只填写判断表达式，不填写完整脚本语句。",
      line: 1,
      column: statementMatch.index + 1,
      endColumn: statementMatch.index + statementMatch[0].length + 1,
      source: "alarm-expression",
    });
  }

  const pairs: Record<string, string> = { "(": ")", "[": "]", "{": "}" };
  const stack: Array<{ char: string; index: number }> = [];
  let quote: '"' | "'" | "`" | null = null;
  let escaping = false;
  for (let index = 0; index < value.length; index += 1) {
    const char = value[index];
    if (quote) {
      if (escaping) {
        escaping = false;
      } else if (char === "\\") {
        escaping = true;
      } else if (char === quote) {
        quote = null;
      }
      continue;
    }
    if (char === "\"" || char === "'" || char === "`") {
      quote = char;
      continue;
    }
    if (pairs[char]) {
      stack.push({ char, index });
      continue;
    }
    if ([")", "]", "}"].includes(char)) {
      const last = stack.pop();
      if (!last || pairs[last.char] !== char) {
        diagnostics.push({
          message: "括号不匹配。",
          line: 1,
          column: index + 1,
          endColumn: index + 2,
          source: "alarm-expression",
        });
      }
    }
  }
  const unclosed = stack.pop();
  if (unclosed) {
    diagnostics.push({
      message: "括号未闭合。",
      line: 1,
      column: unclosed.index + 1,
      endColumn: unclosed.index + 2,
      source: "alarm-expression",
    });
  }
  if (quote) {
    diagnostics.push({
      message: "字符串未闭合。",
      line: 1,
      column: Math.max(1, value.lastIndexOf(quote) + 1),
      endColumn: Math.max(2, value.lastIndexOf(quote) + 2),
      source: "alarm-expression",
    });
  }
  return diagnostics;
};

const ExpressionModule = defineComponent({
  props: {
    condition: { type: Object as () => AlarmCondition, required: true },
  },
  emits: ["patch", "patchParam", "patchSeverity"],
  setup(componentProps, { emit: componentEmit }) {
    const editorRef = ref<{
      setDiagnostics?: (diagnostics: ExpressionDiagnostic[], owner?: string) => void;
    } | null>(null);
    const expressionText = computed(() => textParam(componentProps.condition, "expression"));
    const diagnostics = computed(() => validateExpression(expressionText.value));

    const syncDiagnostics = () => {
      void nextTick(() => {
        editorRef.value?.setDiagnostics?.(diagnostics.value, "alarm-expression");
      });
    };

    watch(expressionText, syncDiagnostics, { immediate: true });

    return () => h("section", { class: "alarm-condition-modules__module" }, [
      h(ModuleHeader, { title: "判断表达式", description: "用表达式处理固定模板无法覆盖的逻辑" }),
      h("div", { class: "alarm-condition-modules__expression" }, [
        h("label", { class: "alarm-condition-modules__expression-enable" }, [
          h("input", {
            type: "checkbox",
            checked: componentProps.condition.isEnabled,
            title: "启用判断表达式",
            "aria-label": "启用判断表达式",
            onChange: (event: Event) => componentEmit("patch", { isEnabled: (event.target as HTMLInputElement).checked }),
          }),
        ]),
        h("div", { class: "alarm-condition-modules__expression-code" }, [
          h(MonacoEditor, {
            ref: editorRef,
            modelValue: expressionText.value,
            language: "javascript",
            theme: "vs",
            height: "96px",
            options: expressionEditorOptions,
            "onUpdate:modelValue": (value: string) => {
              if (value !== expressionText.value) {
                componentEmit("patchParam", "expression", value);
              }
            },
            onChange: syncDiagnostics,
          }),
          h("div", { class: "alarm-condition-modules__expression-status" }, [
            diagnostics.value.length
              ? h("span", { class: "is-error", title: diagnostics.value.map((item) => item.message).join("\n") }, `${diagnostics.value.length} 个问题`)
              : h("span", { class: "is-clean" }, "表达式可用"),
            h("em", "示例：value > 80 && quality == \"good\""),
          ]),
        ]),
        h("div", { class: "alarm-condition-modules__expression-side" }, [
          h("label", [
            h("span", "报警文本"),
            h("input", {
              value: componentProps.condition.name,
              onInput: (event: Event) => componentEmit("patch", { name: (event.target as HTMLInputElement).value }),
            }),
          ]),
          h("div", { class: "alarm-condition-modules__expression-inline" }, [
            h("label", [
              h("span", "等级"),
              h(SeveritySelect, {
                modelValue: componentProps.condition.severity,
                "onUpdate:modelValue": (value: AlarmSeverity) => componentEmit("patchSeverity", value),
              }),
            ]),
            h("label", [
              h("span", "优先级"),
              h("input", {
                type: "number",
                min: 1,
                max: 999,
                step: 1,
                value: priorityParam(componentProps.condition),
                onInput: (event: Event) => componentEmit("patchParam", "priority", priorityValue(event)),
              }),
            ]),
          ]),
        ]),
      ]),
    ]);
  },
});
</script>

<style scoped>
.alarm-condition-modules {
  display: grid;
  gap: 10px;
  padding: 0;
  background: transparent;
}

.alarm-condition-modules__module-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.alarm-condition-modules :deep(.alarm-condition-modules__module-head) {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  min-height: 24px;
  padding: 0 0 6px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.12);
}

.alarm-condition-modules strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-condition-modules span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.alarm-condition-modules__empty {
  padding: 10px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  color: var(--dc-text-muted);
  font-size: 12px;
}

.alarm-condition-modules__module {
  display: grid;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: linear-gradient(180deg, var(--dc-surface-raised), var(--dc-surface));
}

.alarm-condition-modules__module-head {
  min-height: 24px;
  justify-content: flex-start;
  gap: 6px;
  padding: 0;
  border-bottom: 1px solid rgba(148, 163, 184, 0.12);
  padding-bottom: 6px;
}

.alarm-condition-modules__module-head strong,
.alarm-condition-modules :deep(.alarm-condition-modules__module-head strong) {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
  line-height: 1.3;
}

.alarm-condition-modules__help,
.alarm-condition-modules :deep(.alarm-condition-modules__help) {
  width: 18px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid var(--dc-border);
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  cursor: help;
  font-family: inherit;
  font-size: 11px;
  font-weight: 800;
}

.alarm-condition-modules__table {
  display: grid;
  gap: 0;
  overflow-x: auto;
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.alarm-condition-modules__row {
  min-width: 760px;
  display: grid;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 4px 10px;
  border-radius: 0;
  background: transparent;
}

.alarm-condition-modules__row + .alarm-condition-modules__row {
  border-top: 1px solid rgba(148, 163, 184, 0.12);
}

.alarm-condition-modules__row.is-head {
  min-height: 30px;
  background: rgba(248, 250, 252, 0.78);
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0;
}

.alarm-condition-modules__row.is-disabled {
  opacity: 0.68;
}

.alarm-condition-modules__row.is-head span {
  color: inherit;
  font-size: inherit;
}

.alarm-condition-modules__row.is-head > :first-child {
  justify-self: center;
}

.alarm-condition-modules__column-label {
  cursor: help;
  text-decoration: underline dotted rgba(100, 116, 139, 0.45);
  text-underline-offset: 3px;
}

.alarm-condition-modules :deep(.alarm-condition-modules__column-label) {
  cursor: help;
  text-decoration: underline dotted rgba(100, 116, 139, 0.45);
  text-underline-offset: 3px;
}

.alarm-condition-modules__row strong {
  font-size: 12px;
  color: var(--dc-text-secondary);
  white-space: nowrap;
}

.alarm-condition-modules__table.is-threshold .alarm-condition-modules__row {
  grid-template-columns: 34px 54px minmax(120px, 0.8fr) minmax(180px, 1.2fr) 96px 86px 92px 92px;
}

.alarm-condition-modules__table.is-deviation .alarm-condition-modules__row {
  grid-template-columns: 34px 72px minmax(120px, 0.8fr) minmax(120px, 0.8fr) minmax(180px, 1.2fr) 96px 86px 92px;
}

.alarm-condition-modules__table.is-rate .alarm-condition-modules__row {
  grid-template-columns: 34px 86px minmax(140px, 0.8fr) minmax(140px, 0.9fr) minmax(180px, 1.2fr) 96px 86px;
}

.alarm-condition-modules__table.is-bool-state .alarm-condition-modules__row,
.alarm-condition-modules__table.is-bool-transition .alarm-condition-modules__row {
  grid-template-columns: 34px 120px minmax(220px, 1fr) 96px 86px 92px;
}

.alarm-condition-modules__table.is-string .alarm-condition-modules__row {
  grid-template-columns: 34px 72px minmax(220px, 1fr) minmax(220px, 1fr) 96px 86px 92px;
}

.alarm-condition-modules input,
.alarm-condition-modules select,
.alarm-condition-modules textarea,
.alarm-condition-modules button {
  font-family: inherit;
}

.alarm-condition-modules input,
.alarm-condition-modules select,
.alarm-condition-modules textarea {
  width: 100%;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: rgba(255, 255, 255, 0.82);
  color: var(--dc-text);
  font-size: 12px;
  outline: none;
}

.alarm-condition-modules :deep(input),
.alarm-condition-modules :deep(select),
.alarm-condition-modules :deep(textarea) {
  width: 100%;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: rgba(255, 255, 255, 0.82);
  color: var(--dc-text);
  font-family: inherit;
  font-size: 12px;
  outline: none;
}

.alarm-condition-modules input,
.alarm-condition-modules select {
  height: 28px;
  padding: 0 8px;
}

.alarm-condition-modules :deep(input),
.alarm-condition-modules :deep(select) {
  height: 28px;
  padding: 0 8px;
}

.alarm-condition-modules input[type="checkbox"] {
  width: 15px;
  height: 15px;
  justify-self: center;
  padding: 0;
  accent-color: var(--dc-primary);
}

.alarm-condition-modules :deep(input[type="checkbox"]) {
  width: 15px;
  height: 15px;
  justify-self: center;
  padding: 0;
  accent-color: var(--dc-primary);
}

.alarm-condition-modules textarea {
  padding: 7px;
  resize: vertical;
  line-height: 1.45;
}

.alarm-condition-modules :deep(textarea) {
  padding: 7px;
  resize: vertical;
  line-height: 1.45;
}

.alarm-condition-modules input:focus,
.alarm-condition-modules select:focus,
.alarm-condition-modules textarea:focus {
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.1);
}

.alarm-condition-modules :deep(input:focus),
.alarm-condition-modules :deep(select:focus),
.alarm-condition-modules :deep(textarea:focus) {
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.1);
}

.alarm-condition-modules__expression {
  display: grid;
  grid-template-columns: 34px minmax(360px, 1fr) minmax(260px, 360px);
  align-items: start;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.alarm-condition-modules :deep(.alarm-condition-modules__expression) {
  display: grid;
  grid-template-columns: 34px minmax(360px, 1fr) minmax(260px, 360px);
  align-items: start;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.alarm-condition-modules__expression label,
.alarm-condition-modules :deep(.alarm-condition-modules__expression label) {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.alarm-condition-modules__expression label > span,
.alarm-condition-modules :deep(.alarm-condition-modules__expression label > span) {
  color: var(--dc-text-secondary);
  font-size: 11px;
  font-weight: 700;
}

.alarm-condition-modules__expression-code,
.alarm-condition-modules :deep(.alarm-condition-modules__expression-code) {
  min-height: 0;
  display: grid;
  gap: 5px;
  overflow: hidden;
}

.alarm-condition-modules__expression-enable,
.alarm-condition-modules :deep(.alarm-condition-modules__expression-enable) {
  display: flex;
  justify-content: center;
  padding-top: 7px;
}

.alarm-condition-modules__expression-enable input,
.alarm-condition-modules :deep(.alarm-condition-modules__expression-enable input) {
  justify-self: center;
}

.alarm-condition-modules__expression-code :deep(.monaco-editor-container) {
  height: 96px !important;
  min-height: 96px !important;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: #fff;
}

.alarm-condition-modules :deep(.alarm-condition-modules__expression-code .monaco-editor-container) {
  height: 96px !important;
  min-height: 96px !important;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: #fff;
}

.alarm-condition-modules__expression-code :deep(.monaco-editor),
.alarm-condition-modules__expression-code :deep(.monaco-editor-background),
.alarm-condition-modules__expression-code :deep(.margin) {
  background-color: #fff;
}

.alarm-condition-modules :deep(.alarm-condition-modules__expression-code .monaco-editor),
.alarm-condition-modules :deep(.alarm-condition-modules__expression-code .monaco-editor-background),
.alarm-condition-modules :deep(.alarm-condition-modules__expression-code .margin) {
  background-color: #fff;
}

.alarm-condition-modules__expression-status,
.alarm-condition-modules :deep(.alarm-condition-modules__expression-status) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 16px;
  color: var(--dc-text-muted);
  font-size: 11px;
}

.alarm-condition-modules__expression-status span,
.alarm-condition-modules :deep(.alarm-condition-modules__expression-status span) {
  font-size: 11px;
  font-weight: 700;
}

.alarm-condition-modules__expression-status .is-clean,
.alarm-condition-modules :deep(.alarm-condition-modules__expression-status .is-clean) {
  color: #15803d;
}

.alarm-condition-modules__expression-status .is-error,
.alarm-condition-modules :deep(.alarm-condition-modules__expression-status .is-error) {
  color: var(--dc-danger, #b91c1c);
}

.alarm-condition-modules__expression-status em,
.alarm-condition-modules :deep(.alarm-condition-modules__expression-status em) {
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 11px;
  font-style: normal;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-condition-modules__expression-side,
.alarm-condition-modules :deep(.alarm-condition-modules__expression-side) {
  min-width: 0;
  display: grid;
  align-content: start;
  gap: 8px;
}

.alarm-condition-modules__expression-inline,
.alarm-condition-modules :deep(.alarm-condition-modules__expression-inline) {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 92px;
  gap: 8px;
}

@media (max-width: 900px) {
  .alarm-condition-modules__module-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .alarm-condition-modules__expression {
    grid-template-columns: 34px minmax(0, 1fr);
  }

  .alarm-condition-modules :deep(.alarm-condition-modules__expression) {
    grid-template-columns: 34px minmax(0, 1fr);
  }

  .alarm-condition-modules__expression-side,
  .alarm-condition-modules :deep(.alarm-condition-modules__expression-side) {
    grid-column: 2;
  }

  .alarm-condition-modules__expression-inline {
    grid-template-columns: 1fr;
  }

  .alarm-condition-modules :deep(.alarm-condition-modules__expression-inline) {
    grid-template-columns: 1fr;
  }
}
</style>



