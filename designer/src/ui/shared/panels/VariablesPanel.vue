<!--
  VariablesPanel - 变量面板
  管理页面/全局变量，支持增删、导入导出
-->
<script setup lang="ts">
import type { ChangeTarget, VarsConfig } from "@/editor-core/document/types";
import { ElMessage, ElMessageBox } from "element-plus";
import { storeToRefs } from "pinia";
import { computed, ref, watch } from "vue";
import * as XLSX from "xlsx";
import IconEpDelete from "~icons/ep/delete";
import IconEpDownload from "~icons/ep/download";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpPlus from "~icons/ep/plus";
import IconEpUpload from "~icons/ep/upload";
import MonacoEditor from "@/ui/shared/components/common/monaco-editor-async";
import { useEditorStore } from "@/stores/editor-store";

const editorStore = useEditorStore();
const { doc, docVersion, currentPageId, pages } = storeToRefs(editorStore);
const SANITIZE_FILE_NAME_INVALID_RE = /[\\/:*?"<>|]+/g;
const SANITIZE_FILE_NAME_SPACE_RE = /\s+/g;

interface PageVarSourceLike {
  type?: string;
}

interface PageVarDefinitionLike {
  type?: string;
  default?: unknown;
  description?: string;
  access?: string;
  source?: PageVarSourceLike;
  mapped?: boolean;
  groupId?: string | null;
}

interface PageVarListItemLike {
  name: string;
  type: string;
  default: unknown;
  description: string;
}

interface PageVarEditorItemLike {
  name: string;
  type?: string;
  default?: unknown;
  description?: string;
  access?: string;
}

interface StructuredParseSuccess {
  ok: true;
  parsed: unknown;
}

interface StructuredParseFailure {
  ok: false;
  error: string;
}

type StructuredParseResult = StructuredParseSuccess | StructuredParseFailure;
interface ImportRowLike {
  [key: string]: unknown;
}

interface MonacoEditorExposeLike {
  format?: () => Promise<boolean> | boolean;
}

const editVisible = ref(false);
const editMode = ref(false);
const selectedVarName = ref("");
const originalName = ref("");
const formName = ref("");
const formType = ref("string");
const formDefaultText = ref("");
const formDefaultNumber = ref(0);
const formDefaultBoolean = ref(false);
const formDefaultDate = ref<string | Date | null>(null);
const formDescription = ref("");
const editValueEditorRef = ref<MonacoEditorExposeLike | null>(null);
const editValueHasErrors = ref(false);
const importInputRef = ref<HTMLInputElement | null>(null);
const importType = ref("json");

const typeOptions = [
  "string",
  "number",
  "boolean",
  "array",
  "object",
  "set",
  "map",
  "date",
  "regexp",
  "function",
];

const isEditorType = computed(() =>
  ["function", "array", "object", "set", "map"].includes(formType.value),
);
const isStructuredType = computed(() => ["array", "object", "set", "map"].includes(formType.value));
const isTextType = computed(() => ["string", "regexp"].includes(formType.value));
const editorLanguage = computed(() => (isStructuredType.value ? "json" : "javascript"));

const pageName = computed(() => {
  const pageId = currentPageId.value;
  if (!pageId) return "page";
  const page = pages.value.find((item) => item.id === pageId);
  return page?.name || "page";
});

const pageVars = computed<Record<string, PageVarDefinitionLike>>(() => {
  void docVersion.value;
  const pageId = currentPageId.value;
  if (!pageId || !doc.value) return {};
  const vars = doc.value.vars?.pages?.[pageId] as Record<string, PageVarDefinitionLike> | undefined;
  return vars && typeof vars === "object" ? vars : {};
});

const varList = computed<PageVarListItemLike[]>(() => {
  return Object.entries(pageVars.value).map(([name, def]) => ({
    name,
    type: def?.type || "string",
    default: def?.default,
    description: def?.description || "",
  }));
});

/**
 * 格式化默认值显示
 * @param {{ default: any }} item - 变量定义
 * @returns {string} 格式化后的文本
 */
function formatDefaultValue(item: { default?: unknown }): string {
  if (item.default === null || item.default === undefined) return "";
  if (typeof item.default === "object") {
    try {
      return JSON.stringify(item.default);
    } catch {
      return String(item.default);
    }
  }
  return String(item.default);
}

/**
 * 选择表格行
 * @param {string} name - 变量名
 */
function selectRow(name: string): void {
  selectedVarName.value = name || "";
}

/**
 * 打开新增弹窗
 */
function openCreateDialog(): void {
  editMode.value = false;
  formName.value = "";
  originalName.value = "";
  formType.value = "string";
  formDefaultText.value = "";
  formDefaultNumber.value = 0;
  formDefaultBoolean.value = false;
  formDefaultDate.value = null;
  formDescription.value = "";
  editVisible.value = true;
}

/**
 * 打开编辑弹窗
 * @param {{ name: string, type: string, default: any, description: string, access: string }} item - 变量信息
 */
function openEditDialog(item: PageVarEditorItemLike): void {
  if (!item) return;
  editMode.value = true;
  formName.value = item.name || "";
  originalName.value = item.name || "";
  formType.value = item.type || "string";
  formDescription.value = item.description || "";
  if (formType.value === "number") {
    formDefaultNumber.value = Number(item.default) || 0;
  } else if (formType.value === "boolean") {
    formDefaultBoolean.value = Boolean(item.default);
  } else if (formType.value === "date") {
    formDefaultDate.value = (item.default as string | Date | null | undefined) ?? null;
  } else {
    formDefaultText.value =
      item.default === undefined || item.default === null ? "" : String(item.default);
  }
  editVisible.value = true;
}

/**
 * 重置初始值输入
 */
function resetDefaultValue(): void {
  formDefaultText.value = "";
  formDefaultNumber.value = 0;
  formDefaultBoolean.value = false;
  formDefaultDate.value = null;
  editValueHasErrors.value = false;
}

function parseStructuredJson(value: unknown, type: string): StructuredParseResult {
  if (!isStructuredType.value) return { ok: true, parsed: value };
  if (typeof value !== "string") return { ok: true, parsed: value };
  try {
    const parsed = JSON.parse(value);
    if (type === "array" && !Array.isArray(parsed)) {
      return { ok: false, error: "数组类型需要 JSON 数组" };
    }
    if (type === "object") {
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
        return { ok: false, error: "对象类型需要 JSON 对象" };
      }
    }
    if (type === "set") {
      if (Array.isArray(parsed)) return { ok: true, parsed };
      if (parsed && typeof parsed === "object") {
        return { ok: true, parsed: Object.values(parsed) };
      }
      return { ok: false, error: "Set 需要 JSON 数组或对象" };
    }
    if (type === "map") {
      if (Array.isArray(parsed)) return { ok: true, parsed };
      if (parsed && typeof parsed === "object") {
        return { ok: true, parsed: Object.entries(parsed) };
      }
      return { ok: false, error: "Map 需要 JSON 数组或对象" };
    }
    return { ok: true, parsed };
  } catch {
    return { ok: false, error: "JSON 格式不正确" };
  }
}

function validateStructuredValue(): boolean {
  if (!isStructuredType.value) return true;
  const result = parseStructuredJson(formDefaultText.value, formType.value);
  if (!result.ok) {
    ElMessage.error((result.error || "校验失败") as never);
    return false;
  }
  return true;
}

function handleEditValueMarkers(markers: unknown[]): void {
  if (!isEditorType.value) {
    editValueHasErrors.value = false;
    return;
  }
  editValueHasErrors.value = (markers || []).some(
    (marker) => (marker as { severity?: number }).severity === 8,
  );
}

/**
 * 保存变量
 */
function saveVar(): void {
  const name = formName.value.trim();
  if (!name) {
    ElMessage.warning("名称不能为空" as never);
    return;
  }
  if (!currentPageId.value || !doc.value) return;

  if (!editMode.value && pageVars.value[name]) {
    ElMessage.warning("变量名已存在" as never);
    return;
  }
  if (editMode.value && originalName.value && name !== originalName.value && pageVars.value[name]) {
    ElMessage.warning("变量名已存在" as never);
    return;
  }
  if (editValueHasErrors.value) {
    ElMessage.error("初始值存在语法错误，请先修正" as never);
    return;
  }
  if (isStructuredType.value && !validateStructuredValue()) {
    return;
  }

  const nextVars = { ...pageVars.value };
  const structured = parseStructuredJson(formDefaultText.value, formType.value);
  const value =
    formType.value === "number"
      ? Number(formDefaultNumber.value)
      : formType.value === "boolean"
        ? Boolean(formDefaultBoolean.value)
        : formType.value === "date"
          ? formDefaultDate.value
          : structured.ok
            ? structured.parsed
            : formDefaultText.value;
  if (editMode.value && originalName.value && name !== originalName.value) {
    delete nextVars[originalName.value];
  }
  const existing = editMode.value ? pageVars.value[originalName.value] : null;
  nextVars[name] = {
    ...(existing && typeof existing === "object" ? existing : {}),
    type: formType.value,
    default: value,
    description: formDescription.value,
  };

  commitPageVars(nextVars);
  selectedVarName.value = name;
  editVisible.value = false;
}

/**
 * 删除变量
 */
function handleDelete(): void {
  if (!selectedVarName.value) {
    ElMessage.info("请选择需要删除的变量" as never);
    return;
  }
  ElMessageBox.confirm(`确认删除变量 "${selectedVarName.value}" 吗？`, "删除确认", {
    confirmButtonText: "删除",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      if (!currentPageId.value || !doc.value) return;
      const nextVars = { ...pageVars.value };
      delete nextVars[selectedVarName.value];
      commitPageVars(nextVars);
      selectedVarName.value = "";
    })
    .catch(() => {});
}

/**
 * 提交页面变量
 * @param {Record<string, any>} vars - 变量定义
 */
function commitPageVars(vars: Record<string, PageVarDefinitionLike>): void {
  if (!doc.value || !currentPageId.value) return;
  const oldValue = doc.value.schema.vars as VarsConfig;
  const nextVars = {
    ...oldValue,
    pages: {
      ...(oldValue?.pages || {}),
      [currentPageId.value]: vars as Record<string, unknown>,
    },
  } as VarsConfig;
  doc.value.schema.vars = nextVars;
  doc.value._emitChange?.({
    type: "update",
    target: "page" as ChangeTarget,
    oldValue,
    newValue: nextVars,
  } as never);
}

const importAccept = computed(() => {
  if (importType.value === "csv") return ".csv";
  if (importType.value === "xlsx") return ".xlsx,.xls";
  return ".json";
});

function downloadBlob(content: BlobPart, name: string, type: string): void {
  const blob = new Blob([content], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  link.click();
  URL.revokeObjectURL(url);
}

function sanitizeFileName(name: unknown): string {
  return String(name || "page")
    .trim()
    .replace(SANITIZE_FILE_NAME_INVALID_RE, "-")
    .replace(SANITIZE_FILE_NAME_SPACE_RE, "-");
}

function getExportBaseName(): string {
  return `${sanitizeFileName(pageName.value)}-variable`;
}

function buildExportRows(): PageVarListItemLike[] {
  return Object.entries(pageVars.value).map(([name, detail]) => ({
    name,
    type: detail?.type || "string",
    default: formatDefaultValue({ default: detail?.default }),
    description: detail?.description || "",
  }));
}

function normalizeRowKey(row: ImportRowLike, key: string): unknown {
  const lowerKey = key.toLowerCase();
  const hit = Object.keys(row).find((k) => k.toLowerCase() === lowerKey);
  return hit ? row[hit] : "";
}

function mergeImportedRows(rows: ImportRowLike[]): void {
  const nextVars = { ...pageVars.value };
  let added = 0;
  let skipped = 0;

  rows.forEach((row) => {
    const name = String(normalizeRowKey(row, "name") || "").trim();
    if (!name) return;
    if (nextVars[name]) {
      skipped += 1;
      return;
    }
    const type = String(normalizeRowKey(row, "type") || "string").trim();
    const defaultRaw = normalizeRowKey(row, "default");
    const description = String(normalizeRowKey(row, "description") || "");
    const value =
      type === "number"
        ? Number(defaultRaw)
        : type === "boolean"
          ? Boolean(defaultRaw === true || String(defaultRaw).toLowerCase() === "true")
          : type === "date"
            ? defaultRaw || null
            : defaultRaw;
    nextVars[name] = {
      type,
      default: value,
      description,
    };
    added += 1;
  });

  commitPageVars(nextVars);
  ElMessage.success(`导入完成，新增 ${added} 项，跳过 ${skipped} 项` as never);
}

function handleExport(format: string): void {
  const rows = buildExportRows();
  const baseName = getExportBaseName();
  if (format === "json") {
    const payload = {
      pageId: currentPageId.value || "",
      vars: pageVars.value || {},
    };
    downloadBlob(JSON.stringify(payload, null, 2), `${baseName}.json`, "application/json");
    return;
  }

  const worksheet = XLSX.utils.json_to_sheet(rows);
  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, "variables");
  if (format === "csv") {
    const csv = XLSX.utils.sheet_to_csv(worksheet);
    downloadBlob(csv, `${baseName}.csv`, "text/csv");
    return;
  }
  const buffer = XLSX.write(workbook, { bookType: "xlsx", type: "array" });
  downloadBlob(
    buffer,
    `${baseName}.xlsx`,
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  );
}

function handleImport(format: string): void {
  importType.value = format;
  if (importInputRef.value) {
    importInputRef.value.value = "";
    importInputRef.value.click();
  }
}

async function handleFileChange(event: Event): Promise<void> {
  const file = (event.target as HTMLInputElement | null)?.files?.[0];
  if (!file) return;
  if (importType.value === "json") {
    const text = await file.text();
    try {
      const data = JSON.parse(text);
      if (data && typeof data === "object" && data.vars) {
        const vars = data.vars as Record<string, PageVarDefinitionLike>;
        mergeImportedRows(
          Object.entries(vars).map(([name, detail]) => ({
            name,
            type: detail?.type || "string",
            default: detail?.default,
            description: detail?.description || "",
          })),
        );
        return;
      }
      if (Array.isArray(data)) {
        mergeImportedRows(data as ImportRowLike[]);
        return;
      }
      ElMessage.error("JSON 格式不支持" as never);
    } catch {
      ElMessage.error("JSON 解析失败" as never);
    }
    return;
  }

  const buffer = await file.arrayBuffer();
  const workbook = XLSX.read(buffer, { type: "array" });
  const sheetName = workbook.SheetNames[0];
  if (!sheetName) {
    ElMessage.error("文件中没有数据表" as never);
    return;
  }
  const sheet = workbook.Sheets[sheetName];
  if (!sheet) {
    ElMessage.error("文件中没有数据表" as never);
    return;
  }
  const rows = XLSX.utils.sheet_to_json(sheet, {
    defval: "",
  }) as ImportRowLike[];
  mergeImportedRows(rows);
}

watch(currentPageId, () => {
  selectedVarName.value = "";
});
</script>

<template>
  <div class="variables-panel">
    <div class="vars-toolbar">
      <el-button class="toolbar-button" size="small" circle @click="openCreateDialog">
        <IconEpPlus />
      </el-button>
      <el-tooltip content="删除" placement="top">
        <el-button class="toolbar-button" size="small" circle @click="handleDelete">
          <IconEpDelete />
        </el-button>
      </el-tooltip>
      <el-dropdown class="toolbar-dropdown" trigger="hover" @command="handleExport">
        <el-button class="toolbar-button" size="small" circle>
          <IconEpUpload />
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="csv">导出 CSV</el-dropdown-item>
            <el-dropdown-item command="xlsx">导出 XLSX</el-dropdown-item>
            <el-dropdown-item command="json">导出 JSON</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-dropdown class="toolbar-dropdown" trigger="hover" @command="handleImport">
        <el-button class="toolbar-button" size="small" circle>
          <IconEpDownload />
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="csv">导入 CSV</el-dropdown-item>
            <el-dropdown-item command="xlsx">导入 XLSX</el-dropdown-item>
            <el-dropdown-item command="json">导入 JSON</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <input
        ref="importInputRef"
        class="hidden-file-input"
        type="file"
        :accept="importAccept"
        @change="handleFileChange"
      />
    </div>

    <div class="vars-table">
      <div class="vars-header">
        <div class="vars-col vars-name">变量名</div>
        <div class="vars-col vars-default">默认值</div>
      </div>
      <div
        v-for="item in varList"
        :key="item.name"
        class="vars-row"
        :class="{ 'is-active': selectedVarName === item.name }"
        @click="selectRow(item.name)"
        @dblclick="openEditDialog(item)"
      >
        <div class="vars-col vars-name">{{ item.name }}</div>
        <div class="vars-col vars-default">
          <span class="default-value">{{ formatDefaultValue(item) }}</span>
          <el-button class="edit-button" size="small" circle @click.stop="openEditDialog(item)">
            <IconEpEditPen />
          </el-button>
        </div>
      </div>
      <div v-if="varList.length === 0" class="empty-block">暂无数据</div>
    </div>
  </div>

  <el-dialog
    v-model="editVisible"
    :title="editMode ? '编辑变量' : '新增变量'"
    width="520px"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <el-form label-width="80px">
      <el-form-item label="变量名">
        <el-input v-model="formName" />
      </el-form-item>
      <el-form-item label="类型">
        <el-select v-model="formType" @change="resetDefaultValue">
          <el-option v-for="t in typeOptions" :key="t" :label="t" :value="t" />
        </el-select>
      </el-form-item>
      <el-form-item label="初始值">
        <div v-if="isEditorType" class="edit-value-block">
          <MonacoEditor
            ref="editValueEditorRef"
            v-model="formDefaultText"
            :language="editorLanguage"
            height="220px"
            @markers="handleEditValueMarkers"
          />
        </div>
        <el-input v-else-if="isTextType" v-model="formDefaultText" type="textarea" :rows="6" />
        <el-input-number
          v-else-if="formType === 'number'"
          v-model="formDefaultNumber"
          style="width: 100%"
        />
        <el-switch v-else-if="formType === 'boolean'" v-model="formDefaultBoolean" />
        <el-date-picker
          v-else-if="formType === 'date'"
          v-model="formDefaultDate"
          type="datetime"
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="formDescription" type="textarea" :rows="2" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="editVisible = false">取消</el-button>
      <el-button type="primary" @click="saveVar">确定</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.variables-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.vars-toolbar {
  display: flex;
  gap: 6px;
  align-items: center;
}

.vars-toolbar > * {
  margin: 0;
}

.toolbar-button {
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  color: #606266;
}

.toolbar-button:hover {
  background: #eef2ff;
  color: #4f46e5;
}

.toolbar-dropdown {
  display: inline-flex;
}

.hidden-file-input {
  display: none;
}

.vars-table {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  overflow: hidden;
}

.vars-header,
.vars-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  padding: 8px 10px;
}

.vars-header {
  font-size: 12px;
  font-weight: 600;
  background: #f7f8fa;
  color: var(--el-text-color-regular);
  border-bottom: 1px solid #e4e7ed;
}

.vars-row {
  font-size: 12px;
  color: var(--el-text-color-regular);
  cursor: pointer;
  border-bottom: 1px solid #f0f2f5;
}

.vars-row:last-of-type {
  border-bottom: none;
}

.vars-row.is-active {
  background: #eef2ff;
}

.vars-col {
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.vars-default {
  display: flex;
  align-items: center;
  gap: 6px;
}

.default-value {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.edit-button {
  border: none;
  background: #eef2ff;
  color: #4f46e5;
}

.edit-button:hover {
  background: #e0e7ff;
}

.empty-hint,
.empty-block {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  text-align: center;
  padding: 16px 0;
}

.edit-value-block {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.edit-value-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
