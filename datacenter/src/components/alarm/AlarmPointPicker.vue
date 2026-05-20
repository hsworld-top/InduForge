<template>
  <div class="alarm-point-picker">
    <DataPointPicker
      v-model="selectedId"
      :project-id="projectId"
      @select="appendDatapoint"
    />

    <div
      v-if="items.length"
      class="alarm-point-picker__items"
      :class="{ 'has-key': withKey }"
    >
      <div class="alarm-point-picker__labels" aria-hidden="true">
        <span>点位路径</span>
        <span v-if="withKey">变量名</span>
        <span>数据类型</span>
        <span>操作</span>
      </div>
      <div
        v-for="item in items"
        :key="item.datapointId"
        class="alarm-point-picker__item"
      >
        <span>
          <strong>{{ item.name || item.path }}</strong>
          <code>{{ item.path }}</code>
        </span>
        <input
          v-if="withKey"
          :value="inputKey(item)"
          type="text"
          placeholder="变量名"
          @input="updateKey(item.datapointId, inputValue($event))"
        />
        <em>{{ item.dataType || "-" }}</em>
        <button type="button" @click="removeItem(item.datapointId)">移除</button>
      </div>
    </div>
    <div v-else class="alarm-point-picker__empty">
      {{ withKey ? "请选择计算输入点" : "请选择目标点" }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import type {
  AlarmInputRef,
  AlarmTargetRef,
} from "@/api/schemas/alarm.schema";
import type { Datapoint } from "@/api/schemas/datapoint.schema";
import DataPointPicker from "./DataPointPicker.vue";

const props = withDefaults(
  defineProps<{
    projectId: string;
    items: Array<AlarmInputRef | AlarmTargetRef>;
    withKey?: boolean;
  }>(),
  {
    withKey: false,
  },
);

const emit = defineEmits<{
  update: [items: Array<AlarmInputRef | AlarmTargetRef>];
}>();

const selectedId = ref("");

const inputValue = (event: Event) => (event.target as HTMLInputElement).value;

const makeKey = (datapoint: Datapoint) => {
  const source = datapoint.name || datapoint.path.split("/").filter(Boolean).at(-1) || "v";
  const key = source.replace(/[^A-Za-z0-9_]/g, "_").replace(/^[^A-Za-z_]+/, "");
  return key || `v${props.items.length + 1}`;
};

const appendDatapoint = (datapoint: Datapoint) => {
  if (props.items.some((item) => item.datapointId === datapoint.id)) {
    return;
  }
  const base = {
    datapointId: datapoint.id,
    path: datapoint.path,
    name: datapoint.name,
    dataType: datapoint.dataType,
  };
  const next = props.withKey ? { ...base, key: makeKey(datapoint) } : base;
  emit("update", [...props.items, next]);
  selectedId.value = "";
};

const removeItem = (datapointId: string) => {
  emit("update", props.items.filter((item) => item.datapointId !== datapointId));
};

const inputKey = (item: AlarmInputRef | AlarmTargetRef) =>
  "key" in item ? item.key : "";

const updateKey = (datapointId: string, key: string) => {
  emit(
    "update",
    props.items.map((item) =>
      item.datapointId === datapointId && "key" in item ? { ...item, key } : item,
    ),
  );
};
</script>

<style scoped>
.alarm-point-picker {
  display: grid;
  gap: 10px;
}

.alarm-point-picker__items {
  display: grid;
  gap: 6px;
}

.alarm-point-picker__labels {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.alarm-point-picker__items.has-key .alarm-point-picker__labels {
  grid-template-columns: minmax(0, 1fr) minmax(90px, 130px) auto auto;
}

.alarm-point-picker__item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}

.alarm-point-picker__items.has-key .alarm-point-picker__item {
  grid-template-columns: minmax(0, 1fr) minmax(90px, 130px) auto auto;
}

.alarm-point-picker__item strong,
.alarm-point-picker__item code {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-point-picker__item strong {
  font-size: 13px;
}

.alarm-point-picker__item code {
  margin-top: 3px;
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

.alarm-point-picker__item input {
  width: 100%;
  height: 30px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  font-family: var(--dc-font-mono);
  font-size: 12px;
  outline: none;
  padding: 0 8px;
}

.alarm-point-picker__item em {
  color: var(--dc-text-muted);
  font-size: 12px;
  font-style: normal;
}

.alarm-point-picker__item button {
  height: 28px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-danger, #b91c1c);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 700;
  padding: 0 8px;
}

.alarm-point-picker__empty {
  padding: 12px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 13px;
}

@media (max-width: 760px) {
  .alarm-point-picker__labels {
    display: none;
  }

  .alarm-point-picker__item {
    grid-template-columns: 1fr;
  }
}
</style>
