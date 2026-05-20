<template>
  <section class="alarm-condition-matrix">
    <header>
      <div>
        <strong>条件集</strong>
        <span>{{ conditions.length }} 项</span>
      </div>
      <button type="button" @click="addCondition('H')">新增条件</button>
    </header>

    <div class="alarm-condition-matrix__quick">
      <span>快速添加</span>
      <button
        v-for="item in quickTypes"
        :key="item.value"
        type="button"
        @click="addCondition(item.value)"
      >
        {{ item.label }}
      </button>
    </div>

    <div v-if="!conditions.length" class="alarm-condition-matrix__empty">
      尚未配置条件
    </div>

    <div v-else class="alarm-condition-matrix__rows">
      <div class="alarm-condition-matrix__labels" aria-hidden="true">
        <span>启用</span>
        <span>名称</span>
        <span>类型</span>
        <span>级别</span>
        <span>报警值 / 表达式</span>
        <span>恢复差值 / 观察时长</span>
        <span>连续满足 / 变化趋势</span>
        <span>操作</span>
      </div>
      <div
        v-for="condition in conditions"
        :key="condition.id"
        class="alarm-condition-matrix__row"
      >
        <label class="alarm-condition-matrix__switch">
          <input
            type="checkbox"
            :checked="condition.isEnabled"
            @change="patchCondition(condition.id, { isEnabled: checked($event) })"
          />
        </label>
        <input
          :value="condition.name"
          type="text"
          aria-label="条件名称"
          @input="patchCondition(condition.id, { name: value($event) })"
        />
        <select
          :value="condition.type"
          aria-label="条件类型"
          @change="changeType(condition.id, value($event) as AlarmConditionType)"
        >
          <option v-for="item in conditionTypes" :key="item.value" :value="item.value">
            {{ item.label }}
          </option>
        </select>
        <select
          :value="condition.severity"
          aria-label="严重级别"
          @change="patchCondition(condition.id, { severity: value($event) as AlarmSeverity })"
        >
          <option v-for="item in severityOptions" :key="item.value" :value="item.value">
            {{ item.label }}
          </option>
        </select>
        <template v-if="condition.type === 'cel'">
          <textarea
            :value="textParam(condition, 'expression')"
            rows="2"
            spellcheck="false"
            aria-label="自定义表达式"
            placeholder="例如：value > 80 && quality == 'good'"
            @input="patchParams(condition.id, { expression: value($event) })"
          />
        </template>
        <template v-else-if="condition.type === 'rate_of_change'">
          <input
            :value="numberParam(condition, 'limit')"
            type="number"
            step="any"
            aria-label="变化率阈值"
            placeholder="阈值"
            @input="patchNumberParam(condition.id, 'limit', value($event))"
          />
          <input
            :value="numberParam(condition, 'windowMs')"
            type="number"
            min="0"
            step="1"
            aria-label="观察时长毫秒"
            placeholder="观察时长 ms"
            @input="patchNumberParam(condition.id, 'windowMs', value($event))"
          />
          <select
            :value="textParam(condition, 'direction') || 'up'"
            aria-label="变化趋势"
            @change="patchParams(condition.id, { direction: value($event) })"
          >
            <option value="up">上升过快</option>
            <option value="down">下降过快</option>
          </select>
        </template>
        <template v-else>
          <input
            :value="numberParam(condition, 'limit')"
            type="number"
            step="any"
            aria-label="报警值"
            placeholder="报警值"
            @input="patchNumberParam(condition.id, 'limit', value($event))"
          />
          <input
            :value="numberParam(condition, 'hysteresis')"
            type="number"
            step="any"
            aria-label="恢复差值"
            placeholder="恢复差值"
            @input="patchOptionalNumberParam(condition.id, 'hysteresis', value($event))"
          />
          <input
            :value="numberParam(condition, 'durationMs')"
            type="number"
            min="0"
            step="1"
            aria-label="连续满足毫秒"
            placeholder="连续满足 ms"
            @input="patchOptionalNumberParam(condition.id, 'durationMs', value($event))"
          />
        </template>
        <button type="button" class="is-danger" @click="removeCondition(condition.id)">
          删除
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type {
  AlarmCondition,
  AlarmConditionType,
  AlarmSeverity,
} from "@/api/schemas/alarm.schema";
import { createThresholdCondition } from "@/components/alarm/alarmPolicyModel";

const props = defineProps<{
  conditions: AlarmCondition[];
}>();

const emit = defineEmits<{
  update: [conditions: AlarmCondition[]];
}>();

const conditionTypes: Array<{ value: AlarmConditionType; label: string }> = [
  { value: "HH", label: "高高限" },
  { value: "H", label: "高限" },
  { value: "L", label: "低限" },
  { value: "LL", label: "低低限" },
  { value: "deviation_high", label: "高偏差" },
  { value: "deviation_low", label: "低偏差" },
  { value: "rate_of_change", label: "变化率" },
  { value: "cel", label: "自定义" },
];

const quickTypes = conditionTypes.filter((item) =>
  ["HH", "H", "L", "LL"].includes(item.value),
);

const severityOptions: Array<{ value: AlarmSeverity; label: string }> = [
  { value: "info", label: "提示" },
  { value: "warning", label: "警告" },
  { value: "major", label: "重要" },
  { value: "critical", label: "紧急" },
];

const value = (event: Event) =>
  (event.target as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement).value;
const checked = (event: Event) => (event.target as HTMLInputElement).checked;

const update = (conditions: AlarmCondition[]) => emit("update", conditions);

const addCondition = (type: AlarmConditionType) => {
  update([...props.conditions, createThresholdCondition(type)]);
};

const removeCondition = (id: string) => {
  update(props.conditions.filter((condition) => condition.id !== id));
};

const patchCondition = (id: string, patch: Partial<AlarmCondition>) => {
  update(
    props.conditions.map((condition) =>
      condition.id === id ? { ...condition, ...patch } : condition,
    ),
  );
};

const changeType = (id: string, type: AlarmConditionType) => {
  const template = createThresholdCondition(type);
  patchCondition(id, {
    type,
    name: template.name,
    params: template.params,
  });
};

const patchParams = (id: string, patch: Record<string, unknown>) => {
  update(
    props.conditions.map((condition) =>
      condition.id === id
        ? { ...condition, params: { ...condition.params, ...patch } }
        : condition,
    ),
  );
};

const patchNumberParam = (id: string, field: string, raw: string) => {
  patchParams(id, { [field]: raw === "" ? undefined : Number(raw) });
};

const patchOptionalNumberParam = (id: string, field: string, raw: string) => {
  const condition = props.conditions.find((item) => item.id === id);
  if (!condition) {
    return;
  }
  const params = { ...condition.params };
  if (raw === "") {
    delete params[field];
  } else {
    params[field] = Number(raw);
  }
  patchCondition(id, { params });
};

const numberParam = (condition: AlarmCondition, field: string) => {
  const param = condition.params[field];
  return typeof param === "number" && Number.isFinite(param) ? String(param) : "";
};

const textParam = (condition: AlarmCondition, field: string) => {
  const param = condition.params[field];
  return typeof param === "string" ? param : "";
};
</script>

<style scoped>
.alarm-condition-matrix {
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.alarm-condition-matrix header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.alarm-condition-matrix header div {
  min-width: 0;
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.alarm-condition-matrix strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-condition-matrix header span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.alarm-condition-matrix button,
.alarm-condition-matrix input,
.alarm-condition-matrix select,
.alarm-condition-matrix textarea {
  font-family: inherit;
}

.alarm-condition-matrix header button,
.alarm-condition-matrix__quick button,
.alarm-condition-matrix__row button {
  height: 30px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
  padding: 0 9px;
}

.alarm-condition-matrix__quick {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  padding: 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.alarm-condition-matrix__quick > span {
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 700;
  margin-right: 2px;
}

.alarm-condition-matrix__empty {
  padding: 10px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  color: var(--dc-text-muted);
  font-size: 13px;
}

.alarm-condition-matrix__rows {
  display: grid;
  gap: 8px;
}

.alarm-condition-matrix__labels,
.alarm-condition-matrix__row {
  grid-template-columns:
    28px minmax(116px, 1fr) 104px 84px repeat(3, minmax(90px, 1fr)) 48px;
}

.alarm-condition-matrix__labels {
  display: grid;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.alarm-condition-matrix__row {
  display: grid;
  align-items: center;
  gap: 8px;
  padding: 9px 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}

.alarm-condition-matrix__switch {
  display: flex;
  justify-content: center;
}

.alarm-condition-matrix input,
.alarm-condition-matrix select,
.alarm-condition-matrix textarea {
  width: 100%;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  font-size: 12px;
  outline: none;
}

.alarm-condition-matrix input:focus,
.alarm-condition-matrix select:focus,
.alarm-condition-matrix textarea:focus {
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.1);
}

.alarm-condition-matrix input,
.alarm-condition-matrix select {
  height: 30px;
  padding: 0 8px;
}

.alarm-condition-matrix textarea {
  grid-column: span 3;
  padding: 7px 8px;
  resize: vertical;
}

.alarm-condition-matrix__row .is-danger {
  color: var(--dc-danger, #b91c1c);
}

@media (max-width: 1100px) {
  .alarm-condition-matrix__labels {
    display: none;
  }

  .alarm-condition-matrix__row {
    grid-template-columns: 24px repeat(2, minmax(0, 1fr));
  }

  .alarm-condition-matrix textarea {
    grid-column: span 2;
  }
}
</style>
