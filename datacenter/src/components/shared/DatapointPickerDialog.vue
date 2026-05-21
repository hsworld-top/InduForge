<template>
  <DcDialog
    v-model="visible"
    :title="title"
    width="860px"
    body-max-height="620px"
  >
    <div class="datapoint-picker-dialog">
      <div class="datapoint-picker-dialog__toolbar">
        <el-input
          v-model="keyword"
          size="small"
          clearable
          placeholder="搜索名称或路径"
          @keyup.enter="reloadDatapoints"
        />
        <select
          v-model="source"
          class="datapoint-picker-dialog__select"
          aria-label="来源类型"
          @change="reloadDatapoints"
        >
          <option value="">全部来源</option>
          <option value="mqtt.subscription">MQTT</option>
          <option value="db.query">数据库</option>
          <option value="http">HTTP</option>
          <option value="manual">手动</option>
        </select>
        <select
          v-model="dataType"
          class="datapoint-picker-dialog__select"
          aria-label="数据类型"
          @change="reloadDatapoints"
        >
          <option value="">全部类型</option>
          <option value="object">object</option>
          <option value="number">number</option>
          <option value="string">string</option>
          <option value="boolean">boolean</option>
        </select>
        <select
          v-model="status"
          class="datapoint-picker-dialog__select"
          aria-label="状态"
          @change="reloadDatapoints"
        >
          <option value="">全部状态</option>
          <option value="active">正常</option>
          <option value="inactive">停用</option>
          <option value="error">异常</option>
          <option value="unknown">未知</option>
        </select>
        <button
          type="button"
          class="datapoint-picker-dialog__small"
          @click="reloadDatapoints"
        >
          搜索
        </button>
      </div>

      <div v-if="loading" class="datapoint-picker-dialog__loading">
        <el-skeleton :rows="5" animated />
      </div>
      <div v-else class="datapoint-picker-dialog__table-wrap">
        <table
          v-if="options.length"
          class="datapoint-picker-dialog__table"
        >
          <thead>
            <tr>
              <th>名称</th>
              <th>路径</th>
              <th>类型</th>
              <th>来源</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="point in options"
              :key="point.id"
              :class="{ 'is-selected': selectedId === String(point.id) }"
              @click="selectedId = String(point.id)"
              @dblclick="confirm(point)"
            >
              <td>
                <strong>{{ point.name }}</strong>
              </td>
              <td>
                <code>{{ point.path }}</code>
              </td>
              <td>{{ point.dataType || "-" }}</td>
              <td>{{ sourceTypeText(point.sourceType) }}</td>
              <td>
                <span
                  class="datapoint-picker-dialog__status"
                  :class="`is-${point.status || 'unknown'}`"
                >
                  {{ datapointStatusText(point.status) }}
                </span>
              </td>
              <td>
                <span
                  class="datapoint-picker-dialog__row-action"
                  role="button"
                  tabindex="0"
                  @click.stop="confirm(point)"
                  @keydown.enter.stop.prevent="confirm(point)"
                >
                  {{ rowActionText }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
        <div
          v-if="options.length === 0"
          class="datapoint-picker-dialog__empty"
        >
          <strong>没有匹配的数据点</strong>
          <span>换个关键词，或放宽来源、类型、状态筛选。</span>
        </div>
      </div>

      <div class="datapoint-picker-dialog__footer">
        <div class="datapoint-picker-dialog__count">
          共 {{ total }} 条
          <span v-if="selectedDatapoint">已选 {{ selectedDatapoint.name }}</span>
        </div>
        <div class="datapoint-picker-dialog__pager">
          <button
            type="button"
            :disabled="page <= 1 || loading"
            @click="changePage(page - 1)"
          >
            上一页
          </button>
          <span>{{ page }} / {{ totalPages }}</span>
          <button
            type="button"
            :disabled="page >= totalPages || loading"
            @click="changePage(page + 1)"
          >
            下一页
          </button>
          <button
            type="button"
            class="datapoint-picker-dialog__confirm"
            :disabled="!selectedDatapoint"
            @click="confirm()"
            @mousedown.prevent="confirm()"
          >
            {{ confirmText }}
          </button>
        </div>
      </div>
    </div>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { getDatapoints } from "@/api/datapoint.api";
import type { Datapoint } from "@/api/schemas/datapoint.schema";
import DcDialog from "@/components/shared/DcDialog.vue";

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    projectId: string;
    title?: string;
    confirmText?: string;
    rowActionText?: string;
  }>(),
  {
    title: "数据点变量",
    confirmText: "选择",
    rowActionText: "选择",
  },
);

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  select: [datapoint: Datapoint];
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});

const keyword = ref("");
const source = ref("");
const dataType = ref("");
const status = ref("");
const page = ref(1);
const pageSize = 12;
const total = ref(0);
const options = ref<Datapoint[]>([]);
const loading = ref(false);
const selectedId = ref<string | null>(null);

const selectedDatapoint = computed(() =>
  options.value.find((point) => String(point.id) === selectedId.value),
);

const totalPages = computed(() =>
  Math.max(1, Math.ceil(total.value / pageSize)),
);

watch(
  () => props.modelValue,
  async (opened) => {
    if (!opened) {
      return;
    }
    selectedId.value = null;
    await reloadDatapoints();
  },
);

async function reloadDatapoints() {
  page.value = 1;
  await loadDatapoints();
}

async function loadDatapoints() {
  if (!props.projectId) return;
  loading.value = true;
  try {
    const params: Record<string, unknown> = {
      search: keyword.value,
      page: page.value,
      pageSize,
    };
    if (source.value) {
      params.type = source.value;
      params.sourceType = source.value;
    }
    if (dataType.value) {
      params.dataType = dataType.value;
    }
    if (status.value) {
      params.status = status.value;
    }
    const result = await getDatapoints(props.projectId, params);
    options.value = result.list;
    total.value = Number(result.pagination?.total ?? result.list.length);
    if (
      selectedId.value &&
      !result.list.some((point) => String(point.id) === selectedId.value)
    ) {
      selectedId.value = null;
    }
  } finally {
    loading.value = false;
  }
}

async function changePage(next: number) {
  const nextPage = Math.min(Math.max(1, next), totalPages.value);
  if (nextPage === page.value) return;
  page.value = nextPage;
  await loadDatapoints();
}

function confirm(point?: Datapoint) {
  const target = point || selectedDatapoint.value;
  if (!target) return;
  emit("select", target);
  visible.value = false;
}

const datapointStatusText = (value?: string) => {
  const map: Record<string, string> = {
    active: "正常",
    inactive: "停用",
    error: "异常",
    unknown: "未知",
  };
  return map[value || ""] || "未知";
};

const sourceTypeText = (value?: string) => {
  const map: Record<string, string> = {
    "mqtt.subscription": "MQTT",
    "db.query": "数据库",
    http: "HTTP",
    manual: "手动",
  };
  return map[value || ""] || value || "-";
};
</script>

<style scoped>
.datapoint-picker-dialog {
  display: grid;
  gap: 12px;
}

.datapoint-picker-dialog__toolbar {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) 130px 120px 110px auto;
  gap: 8px;
}

.datapoint-picker-dialog__select {
  height: 32px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  font-size: 12px;
  outline: none;
  padding: 0 8px;
}

.datapoint-picker-dialog__small {
  height: 32px;
  padding: 0 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 800;
}

.datapoint-picker-dialog__table-wrap {
  min-height: 320px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  overflow: auto;
}

.datapoint-picker-dialog__table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.datapoint-picker-dialog__table th,
.datapoint-picker-dialog__table td {
  padding: 9px 10px;
  border-bottom: 1px solid var(--dc-border);
  text-align: left;
  vertical-align: middle;
}

.datapoint-picker-dialog__table th {
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 800;
}

.datapoint-picker-dialog__table tbody tr {
  cursor: pointer;
}

.datapoint-picker-dialog__table tbody tr:hover,
.datapoint-picker-dialog__table tbody tr.is-selected {
  background: var(--dc-primary-soft);
}

.datapoint-picker-dialog__table th:nth-child(1),
.datapoint-picker-dialog__table td:nth-child(1) {
  width: 150px;
}

.datapoint-picker-dialog__table th:nth-child(3),
.datapoint-picker-dialog__table td:nth-child(3),
.datapoint-picker-dialog__table th:nth-child(4),
.datapoint-picker-dialog__table td:nth-child(4),
.datapoint-picker-dialog__table th:nth-child(5),
.datapoint-picker-dialog__table td:nth-child(5),
.datapoint-picker-dialog__table th:nth-child(6),
.datapoint-picker-dialog__table td:nth-child(6) {
  width: 86px;
}

.datapoint-picker-dialog__table strong,
.datapoint-picker-dialog__table code,
.datapoint-picker-dialog__table td {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-picker-dialog__table strong {
  display: block;
  color: var(--dc-text);
  font-size: 13px;
}

.datapoint-picker-dialog__table code {
  display: block;
  color: var(--dc-text-muted);
  font-family: Consolas, "Courier New", monospace;
  font-size: 12px;
}

.datapoint-picker-dialog__status {
  height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 800;
}

.datapoint-picker-dialog__status.is-active {
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.datapoint-picker-dialog__status.is-error {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.datapoint-picker-dialog__status.is-inactive {
  color: var(--dc-text-muted);
}

.datapoint-picker-dialog__row-action {
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 9px;
  border: 1px solid rgba(29, 78, 216, 0.28);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 800;
}

.datapoint-picker-dialog__empty {
  margin: 10px;
  display: grid;
  gap: 5px;
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.datapoint-picker-dialog__empty strong {
  color: var(--dc-text);
}

.datapoint-picker-dialog__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.datapoint-picker-dialog__count {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.datapoint-picker-dialog__count span {
  max-width: 240px;
  overflow: hidden;
  color: var(--dc-primary);
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.datapoint-picker-dialog__pager {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.datapoint-picker-dialog__pager button {
  height: 30px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 800;
}

.datapoint-picker-dialog__pager button:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.datapoint-picker-dialog__pager span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.datapoint-picker-dialog__confirm,
.datapoint-picker-dialog__pager .datapoint-picker-dialog__confirm {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
}

@media (max-width: 760px) {
  .datapoint-picker-dialog__toolbar {
    grid-template-columns: 1fr 1fr;
  }

  .datapoint-picker-dialog__footer {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
