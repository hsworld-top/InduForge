<script setup lang="ts">
import type { ProjectI18nSettings, ProjectSchema } from "@/editor-core/document/types";
import type { I18nResourceRow, I18nScanPageInput, I18nScanStatus } from "@/editor-core/i18n/project-i18n";
import IconLucideInfo from "~icons/lucide/info";
import dayjs from "dayjs";
import { ElMessage } from "element-plus";
import { storeToRefs } from "pinia";
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { Serializer } from "@/editor-core/document/Serializer";
import {
  cloneProjectI18nSettings,
  findMutableNode,
  joinI18nResource,
  normalizeProjectI18nSettings,
  scanProjectI18nResources,
  summarizeI18nRows,
  updateI18nResourceValue,
} from "@/editor-core/i18n/project-i18n";
import { projectApi } from "@/services";
import { useEditorStore } from "@/stores/editor-store";
import { fetchResolvedProjectSchemaForPage } from "@/stores/editor/page-load-save-actions";
import { fetchNormalizedPageList } from "@/stores/editor/project-page-actions";
import { resolveProjectSchema } from "@/stores/editor/normalize-schema";

interface OpenPayload {
  scan?: boolean;
  pageId?: string;
  nodeId?: string;
  fieldPath?: string;
}

type StatusFilterValue = "" | "candidate" | "missing" | "issue" | "complete";

const STATUS_LABELS: Record<I18nScanStatus, string> = {
  candidate: "未配置",
  complete: "已完成",
  "missing-default": "待补翻译",
  "missing-current": "待补翻译",
  "source-changed": "原文有变化",
  orphan: "无效文案",
  "binding-conflict": "不可维护",
};

const STATUS_TIPS: Record<I18nScanStatus, string> = {
  candidate: "填写翻译并保存后，会自动纳入文案维护。",
  complete: "翻译语言已填写。",
  "missing-default": "翻译语言还没有填写。",
  "missing-current": "翻译语言还没有填写。",
  "source-changed": "页面里的原文和已维护文案不一致，需要确认是否更新。",
  orphan: "原页面、组件或字段已不存在，可以删除。",
  "binding-conflict": "这个字段使用了动态绑定，国际化只处理静态文案。",
};

const STATUS_TAG_TYPES: Record<I18nScanStatus, "primary" | "success" | "warning" | "danger" | "info"> = {
  candidate: "primary",
  complete: "success",
  "missing-default": "warning",
  "missing-current": "warning",
  "source-changed": "warning",
  orphan: "danger",
  "binding-conflict": "warning",
};

const editorStore = useEditorStore();
const { projectId, projectI18n, currentPageId, doc } = storeToRefs(editorStore);

const visible = ref(false);
const loading = ref(false);
const saving = ref(false);
const keyword = ref("");
const pageFilter = ref("");
const statusFilter = ref<StatusFilterValue>("");
const rows = ref<I18nResourceRow[]>([]);
const pageInputs = ref<I18nScanPageInput[]>([]);
const localSettings = ref<ProjectI18nSettings>(normalizeProjectI18nSettings(null));
const lastScanAt = ref("");
const targetAfterScan = ref<OpenPayload | null>(null);
const selectedRowId = ref("");
const affectedPageIds = ref<Set<string>>(new Set());
const draftValues = ref<Record<string, Record<string, string>>>({});

const enabledLocales = computed(() => localSettings.value.locales.filter((item) => item.enabled));
const defaultLocale = computed(() => localSettings.value.defaultLocale);
const currentLocale = computed(() => localSettings.value.currentLocale);
const pageOptions = computed(() => {
  const map = new Map<string, string>();
  rows.value.forEach((row) => {
    if (row.pageId) map.set(row.pageId, row.pageName || row.pageId);
  });
  return Array.from(map.entries()).map(([value, label]) => ({ value, label }));
});
const summary = computed(() => summarizeI18nRows(rows.value, localSettings.value));
const todoTranslateCount = computed(() => summary.value.missingCurrent);
const issueCount = computed(() => summary.value.sourceChanged + summary.value.orphan + summary.value.conflicts);
const filteredRows = computed(() => {
  const word = keyword.value.trim().toLowerCase();
  return rows.value.filter((row) => {
    if (pageFilter.value && row.pageId !== pageFilter.value) return false;
    if (statusFilter.value) {
      if (statusFilter.value === "missing") {
        if (row.status !== "missing-current") return false;
      } else if (statusFilter.value === "issue") {
        if (
          row.status !== "source-changed" &&
          row.status !== "orphan" &&
          row.status !== "binding-conflict"
        ) {
          return false;
        }
      } else if (row.status !== statusFilter.value) {
        return false;
      }
    }
    if (!word) return true;
    return [
      row.pageName,
      row.nodeLabel,
      row.nodeType,
      row.fieldPath,
      row.sourceText,
      row.defaultValue,
      row.currentValue,
      row.resourceKey,
    ]
      .join(" ")
      .toLowerCase()
      .includes(word);
  });
});

function resolveTableRowClassName({ row }: { row: I18nResourceRow }): string {
  return row.id === selectedRowId.value ? "is-selected-row" : "";
}

function nowText(): string {
  return dayjs().format("YYYY-MM-DD HH:mm:ss");
}

function markPageAffected(pageId: string) {
  if (!pageId) return;
  const next = new Set(affectedPageIds.value);
  next.add(pageId);
  affectedPageIds.value = next;
}

function emitOpenFromPanel(payload: OpenPayload = {}) {
  targetAfterScan.value = payload;
  visible.value = true;
  if (payload.pageId) pageFilter.value = payload.pageId;
  if (payload.scan !== false || rows.value.length === 0) {
    void scanProject();
  } else {
    selectTargetRow(payload);
  }
}

function handleOpenEvent(event: Event) {
  const payload = ((event as CustomEvent<OpenPayload>).detail || {}) as OpenPayload;
  emitOpenFromPanel(payload);
}

function rebuildRows() {
  const result = scanProjectI18nResources(pageInputs.value, localSettings.value);
  rows.value = result.rows;
}

function setDraftValue(rowId: string, locale: string, value: string) {
  draftValues.value = {
    ...draftValues.value,
    [rowId]: {
      ...(draftValues.value[rowId] || {}),
      [locale]: value,
    },
  };
}

function buildCurrentPageInput(): I18nScanPageInput | null {
  if (!doc.value || !currentPageId.value) return null;
  const schema = editorStore.serializer.exportToSchema(doc.value);
  return {
    pageId: currentPageId.value,
    pageName: schema.pagesById[currentPageId.value]?.name || currentPageId.value,
    schema,
  };
}

async function scanProject() {
  if (!projectId.value) {
    ElMessage.warning({ message: "缺少工程信息" } as never);
    return;
  }
  loading.value = true;
  try {
    localSettings.value = cloneProjectI18nSettings(projectI18n.value);
    const { pages } = await fetchNormalizedPageList(projectId.value, projectApi);
    const currentInput = buildCurrentPageInput();
    const inputs: I18nScanPageInput[] = [];
    for (const page of pages) {
      if (currentInput && page.id === currentInput.pageId) {
        inputs.push(currentInput);
        continue;
      }
      const schema = await fetchResolvedProjectSchemaForPage(
        projectApi,
        projectId.value,
        page.id,
        resolveProjectSchema,
      );
      inputs.push({
        pageId: page.id,
        pageName: page.name || schema.pagesById[page.id]?.name || page.id,
        schema,
      });
    }
    pageInputs.value = inputs;
    draftValues.value = {};
    rebuildRows();
    lastScanAt.value = nowText();
    if (targetAfterScan.value) {
      selectTargetRow(targetAfterScan.value);
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : "扫描失败";
    ElMessage.error({ message } as never);
  } finally {
    loading.value = false;
  }
}

function selectTargetRow(payload: OpenPayload) {
  if (!payload.pageId && !payload.nodeId && !payload.fieldPath) return;
  const matched = rows.value.find((row) => {
    if (payload.pageId && row.pageId !== payload.pageId) return false;
    if (payload.nodeId && row.nodeId !== payload.nodeId) return false;
    if (payload.fieldPath && row.fieldPath !== payload.fieldPath) return false;
    return true;
  });
  if (!matched) return;
  selectedRowId.value = matched.id;
  pageFilter.value = payload.pageId || pageFilter.value;
  statusFilter.value = "";
}

function getMutableNode(row: I18nResourceRow) {
  return findMutableNode(pageInputs.value, row.pageId, row.nodeId);
}

function syncCurrentNodeToStore(row: I18nResourceRow) {
  if (row.pageId !== currentPageId.value) return;
  const target = getMutableNode(row);
  if (!target?.node) return;
  editorStore.updateNode(row.nodeId, {
    i18n: target.node.i18n || { props: {} },
  });
}

function ensureResourceForRow(row: I18nResourceRow): boolean {
  if (row.status === "binding-conflict") {
    ElMessage.warning({ message: "该字段使用了动态绑定，不能维护静态翻译" } as never);
    return false;
  }
  const target = getMutableNode(row);
  if (!target?.node) {
    ElMessage.warning({ message: "未找到组件" } as never);
    return false;
  }
  localSettings.value = joinI18nResource(localSettings.value, target.node, row, nowText());
  markPageAffected(row.pageId);
  syncCurrentNodeToStore(row);
  return true;
}

function handleValueChange(row: I18nResourceRow, locale: string, value: string) {
  setDraftValue(row.id, locale, value);
  if (locale === defaultLocale.value) row.defaultValue = value;
  else row.currentValue = value;
  selectedRowId.value = row.id;
}

function applyDraftValues() {
  const entries = Object.entries(draftValues.value);
  if (!entries.length) return;
  entries.forEach(([rowId, localeValues]) => {
    const row = rows.value.find((item) => item.id === rowId);
    if (!row) return;
    Object.entries(localeValues).forEach(([locale, value]) => {
      const nextRow = {
        ...row,
        defaultValue: locale === defaultLocale.value ? value : row.defaultValue,
        currentValue: locale === defaultLocale.value ? row.currentValue : value,
      };
      if (!row.resourceKey || !localSettings.value.resources[row.resourceKey]) {
        if (!ensureResourceForRow(nextRow)) return;
      } else {
        localSettings.value = updateI18nResourceValue(
          localSettings.value,
          row.resourceKey,
          locale,
          value,
          nowText(),
        );
      }
    });
  });
  draftValues.value = {};
  rebuildRows();
}

async function saveAffectedPages() {
  const ids = Array.from(affectedPageIds.value);
  for (const pageId of ids) {
    if (pageId === currentPageId.value) {
      await editorStore.saveCurrentPage();
      continue;
    }
    const page = pageInputs.value.find((item) => item.pageId === pageId);
    if (!page) continue;
    const pageSerializer = new Serializer();
    const pageDoc = pageSerializer.importFromSchema(page.schema);
    const payload = pageSerializer.exportPage(pageDoc, pageId);
    await projectApi.updatePage(projectId.value, pageId, payload);
  }
}

async function handleSave(closeAfterSave = false) {
  if (!projectId.value) return;
  saving.value = true;
  try {
    applyDraftValues();
    editorStore.setProjectI18n(localSettings.value);
    const settingsResult = await editorStore.saveProjectSettings();
    if (!settingsResult.ok) {
      throw settingsResult.error || new Error("保存工程国际化失败");
    }
    await saveAffectedPages();
    affectedPageIds.value = new Set();
    ElMessage.success({ message: "国际化资源已保存" } as never);
    if (closeAfterSave) visible.value = false;
  } catch (error) {
    const message = error instanceof Error ? error.message : "保存失败";
    ElMessage.error({ message } as never);
  } finally {
    saving.value = false;
  }
}

function handleDefaultLocaleChange(value: string) {
  localSettings.value = {
    ...localSettings.value,
    defaultLocale: value,
  };
  rebuildRows();
}

function handleCurrentLocaleChange(value: string) {
  localSettings.value = {
    ...localSettings.value,
    currentLocale: value,
  };
  rebuildRows();
}

function statusCount(status: I18nScanStatus): number {
  return rows.value.filter((row) => row.status === status).length;
}

function selectStatus(status: StatusFilterValue) {
  statusFilter.value = status;
}

onMounted(() => {
  window.addEventListener("designer:i18n-open-resource", handleOpenEvent);
});

onBeforeUnmount(() => {
  window.removeEventListener("designer:i18n-open-resource", handleOpenEvent);
});

defineExpose({
  open: emitOpenFromPanel,
  scanProject,
});
</script>

<template>
  <el-dialog
    v-model="visible"
    title="工程国际化资源"
    width="1120px"
    top="4vh"
    append-to-body
    :close-on-click-modal="false"
  >
    <div class="i18n-dialog">
      <div class="dialog-toolbar">
        <label class="toolbar-field">
          <span>翻译语言</span>
          <el-select
            :model-value="currentLocale"
            size="small"
            class="toolbar-select"
            @update:model-value="handleCurrentLocaleChange"
          >
            <el-option
              v-for="localeItem in enabledLocales"
              :key="localeItem.code"
              :label="`${localeItem.name} ${localeItem.code}`"
              :value="localeItem.code"
            />
          </el-select>
        </label>
        <el-select v-model="pageFilter" size="small" class="toolbar-select" clearable placeholder="全部页面">
          <el-option
            v-for="page in pageOptions"
            :key="page.value"
            :label="page.label"
            :value="page.value"
          />
        </el-select>
        <el-select v-model="statusFilter" size="small" class="toolbar-select" clearable placeholder="全部状态">
          <el-option label="未配置" value="candidate" />
          <el-option label="待补翻译" value="missing" />
          <el-option label="待处理" value="issue" />
          <el-option label="已完成" value="complete" />
        </el-select>
        <el-input v-model="keyword" size="small" placeholder="搜索文案、组件、字段" clearable />
        <el-button size="small" :loading="loading" @click="scanProject">重新扫描工程</el-button>
      </div>

      <div class="dialog-main">
        <aside class="status-side">
          <div class="status-card" :class="{ 'is-active': !statusFilter }" @click="selectStatus('')">
            <span>全部</span>
            <strong>{{ rows.length }}</strong>
          </div>
          <div
            class="status-card"
            :class="{ 'is-active': statusFilter === 'candidate' }"
            @click="selectStatus('candidate')"
          >
            <span>未配置</span>
            <strong>{{ summary.candidates }}</strong>
          </div>
          <div
            class="status-card"
            :class="{ 'is-active': statusFilter === 'missing' }"
            @click="selectStatus('missing')"
          >
            <span>待补翻译</span>
            <strong>{{ todoTranslateCount }}</strong>
          </div>
          <div
            class="status-card"
            :class="{ 'is-active': statusFilter === 'issue' }"
            @click="selectStatus('issue')"
          >
            <span>待处理</span>
            <strong>{{ issueCount }}</strong>
          </div>
          <div class="status-meta">
            <div>已维护：{{ summary.totalResources }}</div>
            <div>上次扫描：{{ lastScanAt || "-" }}</div>
            <div>待保存页面：{{ affectedPageIds.size }}</div>
          </div>
        </aside>

        <el-table
          v-loading="loading"
          :data="filteredRows"
          height="520"
          row-key="id"
          :row-class-name="resolveTableRowClassName"
          border
        >
          <el-table-column label="状态" width="116">
            <template #default="{ row }">
              <el-tooltip :content="STATUS_TIPS[row.status as I18nScanStatus]" placement="top">
                <el-tag :type="STATUS_TAG_TYPES[row.status as I18nScanStatus]" effect="light">
                  {{ STATUS_LABELS[row.status as I18nScanStatus] }}
                </el-tag>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column label="位置" min-width="230">
            <template #default="{ row }">
              <div class="main-text">{{ row.pageName }} / {{ row.nodeLabel }}</div>
              <div class="sub-text">
                {{ row.fieldPath }}
                <el-tooltip placement="top">
                  <template #content>
                    <div>组件 ID：{{ row.nodeId }}</div>
                    <div>资源 key：{{ row.resourceKey || "-" }}</div>
                  </template>
                  <IconLucideInfo class="info-icon" />
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="sourceText" label="原文" min-width="150" show-overflow-tooltip />
          <el-table-column :label="`翻译：${currentLocale}`" min-width="220">
            <template #default="{ row }">
              <el-input
                :model-value="row.currentValue"
                size="small"
                @update:model-value="(value: string) => handleValueChange(row, currentLocale, value)"
              />
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button :loading="saving" @click="handleSave(false)">保存</el-button>
      <el-button type="primary" :loading="saving" @click="handleSave(true)">保存并关闭</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.i18n-dialog {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dialog-toolbar {
  display: grid;
  grid-template-columns: 170px 150px 130px minmax(180px, 1fr) auto;
  gap: 8px;
  align-items: end;
}

.toolbar-select {
  width: 100%;
}

.toolbar-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.toolbar-field > span {
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-label);
}

.dialog-main {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  gap: 10px;
  min-height: 0;
}

.status-side {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.status-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 34px;
  padding: 0 10px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-shell-surface);
  cursor: pointer;
}

.status-card.is-active {
  border-color: var(--designer-primary-border);
  background: var(--designer-primary-soft);
  color: var(--designer-primary-text);
}

.status-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-label);
}

.main-text {
  font-weight: 600;
}

.sub-text {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 2px;
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-label);
}

.info-icon {
  width: 13px;
  height: 13px;
  color: var(--designer-text-tertiary);
  cursor: help;
}

:deep(.is-selected-row td) {
  background: var(--designer-primary-soft) !important;
}
</style>
