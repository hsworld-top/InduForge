<template>
  <header class="workbench-stream-toolbar">
    <div class="workbench-stream-toolbar__identity">
      <component :is="icon" v-if="icon" class="workbench-stream-toolbar__icon" />
      <div class="workbench-stream-toolbar__title">
        <strong>{{ title }}</strong>
        <span v-if="subtitle">{{ subtitle }}</span>
      </div>
      <WorkbenchStatusPill :label="statusLabel" :tone="statusTone" />
    </div>

    <div class="workbench-stream-toolbar__actions">
      <el-input
        :model-value="search"
        class="workbench-stream-toolbar__search"
        size="small"
        placeholder="搜索 Topic 或 Payload"
        clearable
        @update:model-value="$emit('update:search', $event)"
      >
        <template #prefix>
          <IconTablerSearch />
        </template>
      </el-input>

      <label class="workbench-stream-toolbar__limit">
        <span>显示</span>
        <el-input-number
          :model-value="limit"
          :min="10"
          :max="1000"
          :step="10"
          size="small"
          controls-position="right"
          @update:model-value="$emit('update:limit', Number($event || 10))"
        />
      </label>

      <el-tooltip content="刷新历史消息" placement="top">
        <button
          type="button"
          class="workbench-stream-toolbar__icon-btn"
          :disabled="loading"
          @click="$emit('refresh')"
        >
          <IconTablerRefresh />
        </button>
      </el-tooltip>

      <el-tooltip :content="formatJson ? '关闭 JSON 格式化' : '开启 JSON 格式化'" placement="top">
        <button
          type="button"
          class="workbench-stream-toolbar__icon-btn"
          :class="{ 'is-active': formatJson }"
          @click="$emit('update:formatJson', !formatJson)"
        >
          <IconTablerBraces />
        </button>
      </el-tooltip>

      <el-tooltip :content="autoScroll ? '关闭自动滚动' : '开启自动滚动'" placement="top">
        <button
          type="button"
          class="workbench-stream-toolbar__icon-btn"
          :class="{ 'is-active': autoScroll }"
          @click="$emit('update:autoScroll', !autoScroll)"
        >
          <IconTablerArrowBarToUp />
        </button>
      </el-tooltip>

      <el-tooltip :content="showTimestamp ? '隐藏时间戳' : '显示时间戳'" placement="top">
        <button
          type="button"
          class="workbench-stream-toolbar__icon-btn"
          :class="{ 'is-active': showTimestamp }"
          @click="$emit('update:showTimestamp', !showTimestamp)"
        >
          <IconTablerClock />
        </button>
      </el-tooltip>

      <el-tooltip content="清空消息" placement="top">
        <button
          type="button"
          class="workbench-stream-toolbar__icon-btn"
          @click="$emit('clear')"
        >
          <IconTablerTrash />
        </button>
      </el-tooltip>
    </div>
  </header>
</template>

<script setup lang="ts">
import WorkbenchStatusPill from "./WorkbenchStatusPill.vue";
import IconTablerArrowBarToUp from "~icons/tabler/arrow-bar-to-up";
import IconTablerClock from "~icons/tabler/clock";
import IconTablerBraces from "~icons/tabler/braces";
import IconTablerRefresh from "~icons/tabler/refresh";
import IconTablerSearch from "~icons/tabler/search";
import IconTablerTrash from "~icons/tabler/trash";

withDefaults(
  defineProps<{
    title: string;
    subtitle?: string;
    icon?: any;
    statusLabel: string;
    statusTone?: "neutral" | "success" | "warning" | "danger" | "info";
    search: string;
    limit: number;
    loading?: boolean;
    formatJson: boolean;
    autoScroll: boolean;
    showTimestamp: boolean;
  }>(),
  {
    subtitle: "",
    icon: null,
    statusTone: "neutral",
    loading: false,
  },
);

defineEmits<{
  (event: "update:search", value: string): void;
  (event: "update:limit", value: number): void;
  (event: "update:formatJson", value: boolean): void;
  (event: "update:autoScroll", value: boolean): void;
  (event: "update:showTimestamp", value: boolean): void;
  (event: "refresh"): void;
  (event: "clear"): void;
}>();
</script>

<style scoped>
.workbench-stream-toolbar {
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border, #e2e8f0);
  background: var(--dc-surface-subtle, #f8fafc);
}

.workbench-stream-toolbar__identity,
.workbench-stream-toolbar__actions,
.workbench-stream-toolbar__limit,
.workbench-stream-toolbar__icon-btn {
  display: inline-flex;
  align-items: center;
}

.workbench-stream-toolbar__identity {
  min-width: 0;
  gap: 8px;
}

.workbench-stream-toolbar__icon {
  width: 17px;
  height: 17px;
  color: var(--dc-primary, #2563eb);
  flex-shrink: 0;
}

.workbench-stream-toolbar__title {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.workbench-stream-toolbar__title strong {
  overflow: hidden;
  color: var(--dc-text, #0f172a);
  font-size: 13px;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workbench-stream-toolbar__title span {
  overflow: hidden;
  color: var(--dc-text-muted, #64748b);
  font-size: 11px;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workbench-stream-toolbar__actions {
  justify-content: flex-end;
  gap: 7px;
  min-width: 0;
  flex-wrap: wrap;
}

.workbench-stream-toolbar__search {
  width: 220px;
}

.workbench-stream-toolbar__limit {
  gap: 6px;
  color: var(--dc-text-muted, #64748b);
  font-size: 11px;
  font-weight: 700;
}

.workbench-stream-toolbar__limit :deep(.el-input-number) {
  width: 104px;
}

.workbench-stream-toolbar__icon-btn {
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid var(--dc-border, #e2e8f0);
  border-radius: var(--dc-radius-sm, 6px);
  background: var(--dc-surface-raised, #fff);
  color: var(--dc-text-secondary, #334155);
}

.workbench-stream-toolbar__icon-btn svg {
  width: 15px;
  height: 15px;
}

.workbench-stream-toolbar__icon-btn:hover,
.workbench-stream-toolbar__icon-btn.is-active {
  border-color: color-mix(in oklch, var(--dc-primary, #2563eb) 30%, var(--dc-border, #e2e8f0));
  color: var(--dc-primary, #2563eb);
}

.workbench-stream-toolbar__icon-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.workbench-stream-toolbar :deep(.el-input__wrapper) {
  border-radius: var(--dc-radius-sm, 6px);
}

.workbench-stream-toolbar :deep(.el-input__prefix svg) {
  width: 14px;
  height: 14px;
}

@media (max-width: 980px) {
  .workbench-stream-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .workbench-stream-toolbar__actions {
    justify-content: flex-start;
  }

  .workbench-stream-toolbar__search {
    width: min(100%, 320px);
  }
}
</style>
