<template>
  <DcDrawer
    v-model="visible"
    :width="uiPrefs.prefs.drawerWidth"
    :pinned="uiPrefs.prefs.pinnedDrawer"
    title=""
    @resize="uiPrefs.setDrawerWidth"
  >
    <!-- 头部：名称 + 状态 + 关闭 -->
    <template #default>
      <div class="dpd">
        <!-- 自定义头 -->
        <div class="dpd__header">
          <div class="dpd__header-left">
            <span class="dpd__name">{{ datapoint?.name || "-" }}</span>
            <StatusBadge
              v-if="datapoint?.status"
              :tone="resolveStatusTone(datapoint.status)"
              :text="formatStatus(datapoint.status)"
            />
          </div>
          <div class="dpd__header-right">
            <!-- 钉住按钮 -->
            <el-tooltip :content="uiPrefs.prefs.pinnedDrawer ? '取消钉住' : '钉住为旁路面板'" placement="bottom">
              <button
                type="button"
                class="dpd__icon-btn"
                :class="{ 'is-active': uiPrefs.prefs.pinnedDrawer }"
                :aria-label="uiPrefs.prefs.pinnedDrawer ? '取消钉住' : '钉住为旁路面板'"
                @click="togglePin"
              >
                <Paperclip />
              </button>
            </el-tooltip>
            <!-- 关闭按钮 -->
            <el-tooltip content="关闭" placement="bottom">
              <button
                type="button"
                class="dpd__icon-btn"
                aria-label="关闭详情抽屉"
                @click="handleClose"
              >
                <Close />
              </button>
            </el-tooltip>
          </div>
        </div>

        <!-- 路径行 -->
        <div class="dpd__path-row">
          <span class="dpd__path">{{ datapoint?.path || "-" }}</span>
          <el-tooltip content="复制路径" placement="bottom">
            <button
              type="button"
              class="dpd__icon-btn"
              aria-label="复制数据点路径"
              @click="copyPath"
            >
              <CopyDocument />
            </button>
          </el-tooltip>
        </div>

        <!-- 区块 1：基础属性 -->
        <section class="dpd__section">
          <h4 class="dpd__section-title">基础属性</h4>
          <dl class="dpd__grid">
            <div>
              <dt>来源类型</dt>
              <dd>{{ formatSourceType(datapoint?.sourceType) }}</dd>
            </div>
            <div>
              <dt>数据类型</dt>
              <dd class="dpd__mono">{{ datapoint?.dataType || "-" }}</dd>
            </div>
            <div>
              <dt>单位</dt>
              <dd>{{ datapoint?.unit || "-" }}</dd>
            </div>
            <div>
              <dt>精度</dt>
              <dd>{{ (datapoint as DataPointExtended)?.precision ?? "-" }}</dd>
            </div>
            <div class="dpd__grid-full">
              <dt>更新时间</dt>
              <dd class="dpd__mono">{{ formatTime(getUpdatedAt(datapoint)) }}</dd>
            </div>
          </dl>
        </section>

        <!-- 区块 2：标签 -->
        <section class="dpd__section">
          <div class="dpd__section-header">
            <h4 class="dpd__section-title">标签</h4>
            <button
              type="button"
              class="dpd__edit-btn"
              aria-label="编辑标签"
              @click="openTagDialog"
            >
              编辑
            </button>
          </div>
          <div class="dpd__tags">
            <el-tag
              v-for="tag in normalizeTags(datapoint?.tags)"
              :key="tag"
              size="small"
              effect="plain"
              class="dpd__tag"
            >
              {{ tag }}
            </el-tag>
            <span v-if="normalizeTags(datapoint?.tags).length === 0" class="dpd__muted">
              未设置
            </span>
          </div>
        </section>

        <!-- 区块 3：来源 -->
        <section class="dpd__section">
          <div class="dpd__section-header">
            <h4 class="dpd__section-title">来源</h4>
          </div>

          <!-- 来源 LinkChip -->
          <div v-if="datapoint?.sourceId" class="dpd__source-chip">
            <LinkChip
              :module="resolveSourceModule(datapoint.sourceType)"
              :object-id="String(datapoint.sourceId)"
              :label="formatSourceType(datapoint.sourceType)"
              @click="handleLinkChipClick"
            />
          </div>
          <p v-else class="dpd__muted">无来源信息</p>

          <!-- 失效原因（仅 invalid 时显示；无 invalidReason 时给兜底文案） -->
          <div
            v-if="datapoint?.status === 'invalid'"
            class="dpd__invalid-reason"
          >
            <span class="dpd__invalid-label">失效原因：</span>
            {{ (datapoint as DataPointExtended)?.invalidReason || '该数据点已失效，但后端未提供原因' }}
          </div>

          <!-- 配置 JSON 折叠区 -->
          <div v-if="hasSourceConfig" class="dpd__config-collapse">
            <button
              type="button"
              class="dpd__collapse-trigger"
              @click="configExpanded = !configExpanded"
            >
              <span>来源配置</span>
              <ArrowDown :class="{ 'is-expanded': configExpanded }" class="dpd__collapse-icon" />
            </button>
            <pre v-if="configExpanded" class="dpd__code">{{ formatSourceConfig((datapoint as DataPointExtended)?.sourceConfig) }}</pre>
          </div>

          <!-- 测试取值 -->
          <div class="dpd__test-value-row">
            <el-tooltip :content="testBtnTooltip" placement="top" :disabled="canTest && !testDisabledByCapability">
              <button
                type="button"
                class="dpd__test-btn"
                :class="{ 'dpd__test-btn--active': canTest && !testDisabledByCapability }"
                :disabled="!canTest || testTesting || testDisabledByCapability"
                :aria-label="canTest ? '测试取值' : testBtnTooltip"
                @click="handleTestValue"
              >
                {{ testTesting ? '测试中…' : '测试取值' }}
              </button>
            </el-tooltip>

            <!-- 测试结果区块 -->
            <div v-if="testResult" class="dpd__test-result">
              <template v-if="testResult.ok">
                <div class="dpd__test-result-row">
                  <span class="dpd__test-result-label">值：</span>
                  <pre
                    v-if="isLongValue(testResult.value)"
                    class="dpd__test-result-pre"
                  >{{ formatTestValue(testResult.value) }}</pre>
                  <span v-else class="dpd__test-result-value dpd__mono">{{ formatTestValue(testResult.value) }}</span>
                </div>
                <div class="dpd__test-result-row">
                  <span class="dpd__test-result-label">时间：</span>
                  <span class="dpd__test-result-value dpd__mono">{{ testResult.at }}</span>
                </div>
              </template>
              <template v-else>
                <div class="dpd__test-result-error">{{ testResult.message }}</div>
              </template>
            </div>
          </div>
        </section>

        <!-- 区块 4：引用（D2 占位；usages 为空时不显示） -->
        <section
          v-if="usages.length > 0"
          class="dpd__section"
        >
          <h4 class="dpd__section-title">引用</h4>
          <div
            v-for="group in usageGroups"
            :key="group.module"
            class="dpd__usage-group"
          >
            <p class="dpd__usage-group-label">{{ group.label }}</p>
            <div class="dpd__usage-chips">
              <LinkChip
                v-for="item in group.items"
                :key="item.objectId"
                :module="group.module"
                :object-id="item.objectId"
                :label="item.label"
                @click="handleLinkChipClick"
              />
            </div>
          </div>
        </section>

        <!-- 区块 5：运行态权限 -->
        <section class="dpd__section">
          <div class="dpd__section-header">
            <h4 class="dpd__section-title">运行态权限</h4>
            <button
              type="button"
              class="dpd__edit-btn"
              aria-label="编辑运行态权限"
              @click="openPermissionDialog"
            >
              编辑
            </button>
          </div>
          <div class="dpd__perm-summary">
            <span class="dpd__perm-label">写权限摘要：</span>
            <span class="dpd__perm-value">{{ writeSummary }}</span>
          </div>
          <div class="dpd__perm-badges">
            <StatusBadge tone="success" text="可读" />
            <StatusBadge tone="success" text="可订阅" />
          </div>
        </section>
      </div>
    </template>
  </DcDrawer>

  <!-- 标签 dialog -->
  <DataPointTagDialog
    :visible="tagDialogVisible"
    :datapoint="tagDialogDatapoint"
    :project-id="projectId"
    :tag-options="tagOptions"
    :saving="tagSaving"
    @submit="handleTagSubmit"
    @cancel="tagDialogVisible = false"
  />

  <!-- 写权限 dialog -->
  <RuntimePermissionDialog
    :visible="permDialogVisible"
    :datapoint="permDialogDatapoint"
    :project-id="projectId"
    :saving="permSaving"
    @submit="handlePermSubmit"
    @cancel="permDialogVisible = false"
  />
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { ElMessage } from "element-plus";
import {
  ArrowDown,
  Close,
  CopyDocument,
  Paperclip,
} from "@element-plus/icons-vue";
import dayjs from "dayjs";
import { TIME_FORMAT } from "@/constants";
import { updateDatapoint, updateDatapointRuntimeGrant } from "@/api/datapoint.api";
import {
  normalizeRuntimeGrantPayload,
  summarizeRuntimeGrant,
} from "@/utils/runtime-permission-grants";
import { useUiPrefsStore } from "@/stores/ui-prefs.store";
import { useDataCatalogStore } from "@/stores/data-catalog.store";
import type { TestValueResult } from "@/stores/data-catalog.store";
import DcDrawer from "@/components/shared/DcDrawer.vue";
import StatusBadge from "@/components/shared/StatusBadge.vue";
import LinkChip from "@/components/shared/LinkChip.vue";
import DataPointTagDialog from "./DataPointTagDialog.vue";
import RuntimePermissionDialog from "./RuntimePermissionDialog.vue";

// 扩展类型：除 schema 核心字段外，允许后端额外字段（passthrough）
interface DataPointExtended {
  id: string;
  path?: string;
  name?: string;
  status?: string;
  sourceType?: string;
  sourceId?: string | null;
  dataType?: string;
  unit?: string | null;
  precision?: number | string | null;
  tags?: unknown[];
  updatedAt?: string;
  updated_at?: string;
  createdAt?: string;
  created_at?: string;
  runtimePermissions?: Record<string, unknown>;
  runtimePermissionGrants?: Record<string, unknown>;
  writePermission?: Record<string, unknown>;
  runtimeGrant?: Record<string, unknown>;
  sourceConfig?: Record<string, unknown>;
  invalidReason?: string | null;
  /** 引用列表（后端未实现时为空数组） */
  usages?: UsageItem[];
}

interface UsageItem {
  module: "access-source" | "compute" | "alarm" | "datapoint";
  objectId: string;
  label: string;
}

// LinkChip 的 module 类型
type LinkChipModule = "datapoint" | "access-source" | "compute" | "alarm";

const props = defineProps<{
  /** 是否显示 */
  modelValue: boolean;
  /** 数据点详情 */
  datapoint: DataPointExtended | null;
  /** 项目 ID */
  projectId: string;
  /** 标签选项（由 DataPointList 传入） */
  tagOptions?: Array<{ value: string; name: string }>;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  /** 抽屉关闭 */
  close: [];
  /** 数据已更新，通知父组件刷新 */
  updated: [];
  /** 跳转到另一个模块 */
  navigate: [payload: { module: LinkChipModule; objectId: string }];
}>();

const uiPrefs = useUiPrefsStore();
const catalog = useDataCatalogStore();

// 来源配置折叠状态
const configExpanded = ref(false);

// 测试取值状态（drawer 本地，切换数据点时重置）
const testTesting = ref(false);
const testResult = ref<TestValueResult | null>(null);
// 因"能力未启用"临时禁用按钮（仅本次会话内）
const testDisabledByCapability = ref(false);
const testCapabilityMsg = ref("");

/** 支持测试取值的 sourceType 列表 */
const TEST_SUPPORTED_TYPES = ["db.query", "mqtt.tag", "calc.output"] as const;

const canTest = computed(() => {
  const t = props.datapoint?.sourceType;
  return !!t && (TEST_SUPPORTED_TYPES as readonly string[]).includes(t);
});

const testBtnTooltip = computed(() => {
  if (testDisabledByCapability.value) return testCapabilityMsg.value;
  if (!canTest.value) {
    const t = props.datapoint?.sourceType;
    if (t === "alarm.state") return "报警状态不支持测试取值";
    if (t === "mqtt.subscription") return "MQTT 订阅暂不支持单点测试取值";
    return "该来源类型暂不支持测试取值";
  }
  return "";
});

// 标签 dialog
const tagDialogVisible = ref(false);
const tagDialogDatapoint = ref<DataPointExtended | null>(null);
const tagSaving = ref(false);

// 权限 dialog
const permDialogVisible = ref(false);
const permDialogDatapoint = ref<DataPointExtended | null>(null);
const permSaving = ref(false);

const visible = computed({
  get: () => props.modelValue,
  set: (val: boolean) => {
    emit("update:modelValue", val);
    if (!val) emit("close");
  },
});

// 折叠重置（切换数据点时）
watch(
  () => props.datapoint?.id,
  () => {
    configExpanded.value = false;
    // 切换数据点时重置测试取值状态
    testResult.value = null;
    testDisabledByCapability.value = false;
    testCapabilityMsg.value = "";
  },
);

// ── 来源配置相关 ──────────────────────────────────────────────────────────

const hasSourceConfig = computed(() => {
  const cfg = (props.datapoint as DataPointExtended)?.sourceConfig;
  return !!cfg && Object.keys(cfg).length > 0;
});

function formatSourceConfig(value?: Record<string, unknown>): string {
  if (!value || Object.keys(value).length === 0) return "{}";
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return "{}";
  }
}

// ── 引用区块 ────────────────────────────────────────────────────────────

const usages = computed<UsageItem[]>(
  () => (props.datapoint as DataPointExtended)?.usages ?? [],
);

// 按模块分组
const MODULE_LABELS: Record<string, string> = {
  "access-source": "接入源",
  compute: "计算单元",
  alarm: "报警规则",
  datapoint: "数据点",
};

const usageGroups = computed(() => {
  const groupMap = new Map<string, { module: LinkChipModule; label: string; items: UsageItem[] }>();
  for (const item of usages.value) {
    if (!groupMap.has(item.module)) {
      groupMap.set(item.module, {
        module: item.module,
        label: MODULE_LABELS[item.module] || item.module,
        items: [],
      });
    }
    groupMap.get(item.module)!.items.push(item);
  }
  return Array.from(groupMap.values());
});

// ── 权限摘要 ────────────────────────────────────────────────────────────

const writeSummary = computed(() => {
  const dp = props.datapoint;
  if (!dp) return "-";
  const rp = dp.runtimePermissions as { write?: unknown } | undefined;
  const rpg = dp.runtimePermissionGrants as { write?: unknown } | undefined;
  const grant = normalizeRuntimeGrantPayload(
    rp?.write || rpg?.write || dp.writePermission || dp.runtimeGrant || {},
  );
  return summarizeRuntimeGrant(grant);
});

// ── 格式化工具 ────────────────────────────────────────────────────────────

function resolveStatusTone(status?: string): "success" | "danger" | "warning" | "muted" {
  switch (status) {
    case "active": return "success";
    case "invalid": return "danger";
    case "error": return "warning";
    default: return "muted";
  }
}

function formatStatus(status?: string): string {
  switch (status) {
    case "active": return "活跃";
    case "invalid": return "失效";
    case "error": return "错误";
    case "inactive": return "停用";
    default: return "未知";
  }
}

const sourceTypeLabels: Record<string, string> = {
  "db.query": "数据库查询",
  "mqtt.tag": "MQTT 变量",
  "mqtt.subscription": "MQTT 订阅",
  "calc.output": "计算输出",
  "static.var": "静态变量",
};

function formatSourceType(sourceType?: string): string {
  if (!sourceType) return "-";
  return sourceTypeLabels[sourceType] || sourceType;
}

function formatTime(value?: string | null): string {
  if (!value) return "-";
  return dayjs(value).format(TIME_FORMAT);
}

function getUpdatedAt(dp: DataPointExtended | null): string | undefined {
  if (!dp) return undefined;
  return dp.updatedAt || dp.updated_at || dp.createdAt || dp.created_at || undefined;
}

function normalizeTags(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  const result: string[] = [];
  for (const item of value) {
    let label = "";
    if (typeof item === "string") {
      label = item;
    } else if (item && typeof item === "object") {
      const r = item as Record<string, unknown>;
      label = String(r.label || r.name || r.value || "");
    }
    const n = label.trim();
    if (n && !result.includes(n)) result.push(n);
  }
  return result;
}

/** sourceType → LinkChip module 映射 */
function resolveSourceModule(sourceType?: string): LinkChipModule {
  if (!sourceType) return "access-source";
  if (sourceType.startsWith("calc")) return "compute";
  if (sourceType.startsWith("alarm")) return "alarm";
  return "access-source";
}

// ── 交互 ──────────────────────────────────────────────────────────────────

function togglePin() {
  uiPrefs.setPinnedDrawer(!uiPrefs.prefs.pinnedDrawer);
}

function handleClose() {
  visible.value = false;
}

async function copyPath() {
  const path = props.datapoint?.path;
  if (!path) return;
  try {
    await navigator.clipboard.writeText(path);
    ElMessage.success("已复制数据点路径");
  } catch {
    ElMessage.error("复制失败，请手动复制");
  }
}

// ── 标签 dialog ────────────────────────────────────────────────────────────

function openTagDialog() {
  if (!props.datapoint) return;
  tagDialogDatapoint.value = props.datapoint;
  tagDialogVisible.value = true;
}

async function handleTagSubmit(tags: string[]) {
  if (!props.datapoint) return;
  tagSaving.value = true;
  try {
    await updateDatapoint(props.projectId, String(props.datapoint.id), { tags } as never);
    ElMessage.success("标签已保存");
    tagDialogVisible.value = false;
    emit("updated");
  } catch {
    ElMessage.error("保存标签失败");
  } finally {
    tagSaving.value = false;
  }
}

// ── 权限 dialog ────────────────────────────────────────────────────────────

function openPermissionDialog() {
  if (!props.datapoint) return;
  permDialogDatapoint.value = props.datapoint;
  permDialogVisible.value = true;
}

async function handlePermSubmit(grant: Record<string, unknown>) {
  if (!props.datapoint) return;
  permSaving.value = true;
  try {
    await updateDatapointRuntimeGrant(props.projectId, String(props.datapoint.id), { write: grant });
    ElMessage.success("写权限已保存");
    permDialogVisible.value = false;
    emit("updated");
  } catch {
    ElMessage.error("保存运行态权限失败");
  } finally {
    permSaving.value = false;
  }
}

// ── 测试取值 ──────────────────────────────────────────────────────────────

async function handleTestValue() {
  if (!props.datapoint || testTesting.value) return;
  testTesting.value = true;
  testResult.value = null;
  const dp = props.datapoint as DataPointExtended;
  const result = await catalog.testDatapointValue(props.projectId, {
    id: String(dp.id),
    sourceType: dp.sourceType,
    sourceId: dp.sourceId ?? null,
    sourceConfig: dp.sourceConfig,
  });
  testTesting.value = false;
  testResult.value = result;
  // 能力未启用：临时禁用按钮
  if (!result.ok && result.reason === "capability-disabled") {
    testDisabledByCapability.value = true;
    testCapabilityMsg.value = result.message;
  }
}

/** 判断值是否超过两行（简单判定：字符数 > 80 或含换行） */
function isLongValue(value: unknown): boolean {
  const s = formatTestValue(value);
  return s.length > 80 || s.includes("\n");
}

function formatTestValue(value: unknown): string {
  if (value === null || value === undefined) return "null";
  if (typeof value === "object") {
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return String(value);
    }
  }
  return String(value);
}

// ── LinkChip 跳转 ──────────────────────────────────────────────────────────

function handleLinkChipClick(payload: { module: LinkChipModule; objectId: string }) {
  emit("navigate", { module: payload.module, objectId: payload.objectId });
}
</script>

<style scoped>
/* ── 整体容器 ── */
.dpd {
  display: flex;
  flex-direction: column;
  gap: 0;
  height: 100%;
  overflow-y: auto;
}

/* ── 头部 ── */
.dpd__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 16px 10px;
  border-bottom: 1px solid var(--dc-border);
}

.dpd__header-left {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.dpd__header-right {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.dpd__name {
  font-size: 16px;
  font-weight: 700;
  color: var(--dc-text);
  overflow-wrap: anywhere;
}

/* ── 路径行 ── */
.dpd__path-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 16px 12px;
  border-bottom: 1px solid var(--dc-border);
}

.dpd__path {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
  line-height: 1.6;
}

/* ── icon 按钮 ── */
.dpd__icon-btn {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--dc-text-muted);
  cursor: pointer;
  transition: background 0.18s, color 0.18s, border-color 0.18s;
}

.dpd__icon-btn svg {
  width: 15px;
  height: 15px;
}

.dpd__icon-btn:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.dpd__icon-btn.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

/* ── 区块通用 ── */
.dpd__section {
  padding: 16px;
  border-bottom: 1px solid var(--dc-border);
}

.dpd__section:last-child {
  border-bottom: 0;
}

.dpd__section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.dpd__section-title {
  margin: 0 0 10px;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.dpd__section-header .dpd__section-title {
  margin: 0;
}

/* ── 编辑按钮 ── */
.dpd__edit-btn {
  height: 24px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: 8px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 600;
  transition: background 0.18s, border-color 0.18s, color 0.18s;
}

.dpd__edit-btn:hover {
  border-color: rgba(29, 78, 216, 0.24);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

/* ── 基础属性 grid ── */
.dpd__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin: 0;
}

.dpd__grid > div {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface);
}

.dpd__grid-full {
  grid-column: 1 / -1;
}

.dpd__grid dt {
  margin: 0 0 4px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.dpd__grid dd {
  margin: 0;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 600;
  overflow-wrap: anywhere;
}

.dpd__mono {
  font-family: var(--dc-font-mono);
}

/* ── 标签 ── */
.dpd__tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-height: 28px;
}

.dpd__tag {
  --el-tag-bg-color: var(--dc-primary-soft);
  --el-tag-border-color: transparent;
  --el-tag-text-color: var(--dc-primary);
  border-radius: 999px;
  font-weight: 700;
}

.dpd__muted {
  color: var(--dc-text-muted);
  font-size: 12px;
}

/* ── 来源区块 ── */
.dpd__source-chip {
  margin-bottom: 8px;
}

.dpd__invalid-reason {
  padding: 8px 10px;
  border-radius: var(--dc-radius-md);
  background: rgba(220, 38, 38, 0.06);
  color: var(--dc-danger, #dc2626);
  font-size: 12px;
  line-height: 1.6;
  margin-bottom: 8px;
}

.dpd__invalid-label {
  font-weight: 600;
}

/* ── 来源配置折叠 ── */
.dpd__config-collapse {
  margin-top: 8px;
}

.dpd__collapse-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 600;
}

.dpd__collapse-icon {
  width: 14px;
  height: 14px;
  transition: transform 0.2s;
}

.dpd__collapse-icon.is-expanded {
  transform: rotate(180deg);
}

.dpd__code {
  max-height: 200px;
  overflow: auto;
  margin: 8px 0 0;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

/* ── 测试取值 ── */
.dpd__test-value-row {
  margin-top: 12px;
}

.dpd__test-btn {
  height: 28px;
  padding: 0 14px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  cursor: not-allowed;
  font-family: inherit;
  font-size: 13px;
  font-weight: 600;
  opacity: 0.55;
  transition: background 0.18s, border-color 0.18s, color 0.18s, opacity 0.18s;
}

.dpd__test-btn--active {
  border-color: rgba(29, 78, 216, 0.32);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  cursor: pointer;
  opacity: 1;
}

.dpd__test-btn--active:hover {
  border-color: var(--dc-primary);
}

.dpd__test-btn--active:disabled {
  cursor: not-allowed;
  opacity: 0.65;
}

.dpd__test-result {
  margin-top: 10px;
  padding: 10px 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-muted);
  font-size: 12px;
  line-height: 1.6;
}

.dpd__test-result-row {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin-bottom: 4px;
}

.dpd__test-result-row:last-child {
  margin-bottom: 0;
}

.dpd__test-result-label {
  flex-shrink: 0;
  font-weight: 600;
  color: var(--dc-text-muted);
}

.dpd__test-result-value {
  color: var(--dc-text);
  overflow-wrap: anywhere;
}

.dpd__test-result-pre {
  max-height: 160px;
  overflow: auto;
  margin: 0;
  padding: 8px;
  border: 1px solid var(--dc-border);
  border-radius: 6px;
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 11px;
  line-height: 1.5;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  flex: 1;
}

.dpd__test-result-error {
  color: var(--dc-danger, #dc2626);
  font-weight: 500;
}

/* ── 引用区块 ── */
.dpd__usage-group {
  margin-bottom: 10px;
}

.dpd__usage-group-label {
  margin: 0 0 6px;
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 600;
}

.dpd__usage-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

/* ── 权限区块 ── */
.dpd__perm-summary {
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--dc-text-secondary);
}

.dpd__perm-label {
  font-weight: 600;
  color: var(--dc-text-muted);
}

.dpd__perm-value {
  color: var(--dc-text);
}

.dpd__perm-badges {
  display: flex;
  gap: 6px;
}
</style>
