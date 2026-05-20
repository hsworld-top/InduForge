<template>
  <aside class="alarm-rule-list">
    <header class="alarm-rule-list__head">
      <div class="alarm-rule-list__title">
        <strong>报警规则</strong>
        <span>{{ total }} 条</span>
      </div>
      <div class="alarm-rule-list__actions">
        <el-tooltip content="刷新报警规则" placement="bottom">
          <button
            type="button"
            class="alarm-rule-list__icon-button"
            :disabled="loading"
            aria-label="刷新报警规则"
            @click="emit('refresh')"
          >
            <Refresh />
          </button>
        </el-tooltip>
        <button
          type="button"
          class="alarm-rule-list__create"
          @click="emit('create')"
        >
          <Plus />
          <span>新建</span>
        </button>
      </div>
    </header>

    <div class="alarm-rule-list__filters">
      <label>
        <span>搜索</span>
        <input
          v-model="filters.search"
          type="search"
          placeholder="规则名或点位"
          @input="emitFilter"
        />
      </label>
      <label>
        <span>级别</span>
        <select v-model="filters.severity" @change="emitFilter">
          <option value="">全部级别</option>
          <option value="info">提示</option>
          <option value="warning">警告</option>
          <option value="major">重要</option>
          <option value="critical">紧急</option>
        </select>
      </label>
      <label>
        <span>状态</span>
        <select v-model="filters.enabled" @change="emitFilter">
          <option value="">全部状态</option>
          <option value="true">启用</option>
          <option value="false">停用</option>
        </select>
      </label>
    </div>

    <div v-if="error" class="alarm-rule-list__state is-error">
      <strong>列表不可用</strong>
      <span>{{ error }}</span>
      <button type="button" @click="emit('refresh')">重试</button>
    </div>
    <div v-else-if="loading" class="alarm-rule-list__state">
      <strong>正在读取</strong>
      <span>加载报警规则列表</span>
    </div>
    <div v-else-if="!rules.length" class="alarm-rule-list__state">
      <strong>暂无规则</strong>
      <span>新建规则后会显示在这里</span>
      <button type="button" @click="emit('create')">新建报警规则</button>
    </div>

    <div v-else class="alarm-rule-list__body">
      <button
        v-for="rule in rules"
        :key="rule.id"
        type="button"
        class="alarm-rule-list__row"
        :class="{
          'is-active': rule.id === selectedId,
          'is-disabled': !rule.isEnabled,
        }"
        @click="emit('select', rule.id)"
      >
        <i :class="`is-${rule.severity}`" />
        <span class="alarm-rule-list__main">
          <strong>{{ rule.name }}</strong>
          <code>{{ rule.targetPath }}</code>
        </span>
        <span class="alarm-rule-list__meta">
          <em :class="`is-${rule.severity}`">
            {{ severityLabel(rule.severity) }}
          </em>
          <small>{{ rule.isEnabled ? "启用" : "停用" }}</small>
        </span>
      </button>
    </div>

    <footer v-if="total > pageSize" class="alarm-rule-list__pager">
      <span>{{ page }} / {{ totalPages }}</span>
      <div>
        <button type="button" :disabled="page <= 1" @click="changePage(page - 1)">
          上一页
        </button>
        <button
          type="button"
          :disabled="page >= totalPages"
          @click="changePage(page + 1)"
        >
          下一页
        </button>
      </div>
    </footer>
  </aside>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { Plus, Refresh } from "@element-plus/icons-vue";
import type { AlarmRule, AlarmSeverity } from "@/api/schemas/alarm.schema";

const props = defineProps<{
  rules: AlarmRule[];
  total: number;
  selectedId: string;
  loading: boolean;
  error: string;
}>();

const emit = defineEmits<{
  refresh: [];
  create: [];
  select: [id: string];
  filter: [params: Record<string, string>];
}>();

const filters = reactive({
  search: "",
  severity: "",
  enabled: "",
});
const page = ref(1);
const pageSize = 20;

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / pageSize)));

const severityText: Record<AlarmSeverity, string> = {
  info: "提示",
  warning: "警告",
  major: "重要",
  critical: "紧急",
};

const severityLabel = (severity: AlarmSeverity) => severityText[severity] ?? severity;

const emitFilter = () => {
  emit("filter", {
    search: filters.search.trim(),
    severity: filters.severity,
    enabled: filters.enabled,
    page: String(page.value),
    pageSize: String(pageSize),
  });
};

const changePage = (nextPage: number) => {
  page.value = Math.min(Math.max(nextPage, 1), totalPages.value);
  emitFilter();
};
</script>

<style scoped>
.alarm-rule-list {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
}

.alarm-rule-list__head {
  min-height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--dc-border);
}

.alarm-rule-list__title strong,
.alarm-rule-list__title span {
  display: block;
}

.alarm-rule-list__title strong {
  font-size: 14px;
}

.alarm-rule-list__title span {
  margin-top: 3px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.alarm-rule-list__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.alarm-rule-list__filters {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 86px 86px;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.alarm-rule-list__filters label {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.alarm-rule-list__filters span {
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.alarm-rule-list__filters input,
.alarm-rule-list__filters select {
  width: 100%;
  height: 30px;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  font-family: inherit;
  font-size: 12px;
  outline: none;
}

.alarm-rule-list__filters input {
  padding: 0 9px;
}

.alarm-rule-list__filters select {
  padding: 0 6px;
}

.alarm-rule-list button {
  font-family: inherit;
}

.alarm-rule-list__icon-button,
.alarm-rule-list__create {
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
}

.alarm-rule-list__icon-button {
  width: 30px;
  padding: 0;
}

.alarm-rule-list__create {
  gap: 5px;
  padding: 0 10px;
  color: var(--dc-primary);
}

.alarm-rule-list__icon-button svg,
.alarm-rule-list__create svg {
  width: 15px;
  height: 15px;
}

.alarm-rule-list__icon-button:hover:not(:disabled),
.alarm-rule-list__create:hover {
  border-color: rgba(37, 99, 235, 0.26);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.alarm-rule-list__icon-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.alarm-rule-list__body {
  min-height: 0;
  padding: 8px;
  overflow-y: auto;
}

.alarm-rule-list__pager {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  border-top: 1px solid var(--dc-border);
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.alarm-rule-list__pager div {
  display: flex;
  gap: 6px;
}

.alarm-rule-list__pager button {
  height: 26px;
  padding: 0 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
}

.alarm-rule-list__pager button:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.alarm-rule-list__row {
  width: 100%;
  display: grid;
  grid-template-columns: 4px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  margin: 0 0 8px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  cursor: pointer;
  text-align: left;
}

.alarm-rule-list__row:hover {
  border-color: rgba(37, 99, 235, 0.24);
  background: var(--dc-surface-raised);
}

.alarm-rule-list__row.is-active {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
}

.alarm-rule-list__row.is-disabled {
  opacity: 0.68;
}

.alarm-rule-list__row i {
  width: 4px;
  height: 38px;
  border-radius: 2px;
  background: var(--dc-border);
}

.alarm-rule-list__row i.is-info {
  background: #3b82f6;
}

.alarm-rule-list__row i.is-warning {
  background: #d97706;
}

.alarm-rule-list__row i.is-major {
  background: #ea580c;
}

.alarm-rule-list__row i.is-critical {
  background: #dc2626;
}

.alarm-rule-list__main {
  min-width: 0;
}

.alarm-rule-list__main strong,
.alarm-rule-list__main code {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-rule-list__main strong {
  font-size: 13px;
}

.alarm-rule-list__main code {
  margin-top: 4px;
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

.alarm-rule-list__meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 5px;
}

.alarm-rule-list__meta em,
.alarm-rule-list__meta small {
  font-style: normal;
  white-space: nowrap;
}

.alarm-rule-list__meta em {
  font-size: 12px;
  font-weight: 700;
}

.alarm-rule-list__meta em.is-info {
  color: #2563eb;
}

.alarm-rule-list__meta em.is-warning {
  color: #b45309;
}

.alarm-rule-list__meta em.is-major {
  color: #c2410c;
}

.alarm-rule-list__meta em.is-critical {
  color: #b91c1c;
}

.alarm-rule-list__meta small {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.alarm-rule-list__state {
  display: grid;
  gap: 8px;
  margin: 12px;
  padding: 14px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.alarm-rule-list__state strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-rule-list__state button {
  justify-self: start;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
}

.alarm-rule-list__state.is-error {
  border-color: rgba(220, 38, 38, 0.32);
}

.alarm-rule-list__state.is-error strong,
.alarm-rule-list__state.is-error span {
  color: var(--dc-danger, #b91c1c);
}

@media (max-width: 920px) {
  .alarm-rule-list {
    max-height: 320px;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }
}
</style>
