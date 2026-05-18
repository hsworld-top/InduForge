<template>
  <DcDrawer
    v-model="visible"
    :width="uiPrefs.prefs.drawerWidth"
    :pinned="uiPrefs.prefs.pinnedDrawer"
    title=""
    @resize="uiPrefs.setDrawerWidth"
  >
    <template #default>
      <div class="asd">
        <!-- 头部 -->
        <div class="asd__header">
          <div class="asd__header-left">
            <span class="asd__name">{{ connection?.name || "未命名连接" }}</span>
            <StatusBadge
              :tone="resolveStatusTone(connection?.status)"
              :text="resolveStatusText(connection?.status)"
            />
          </div>
          <div class="asd__header-right">
            <!-- 钉住 -->
            <el-tooltip
              :content="uiPrefs.prefs.pinnedDrawer ? '取消钉住' : '钉住为旁路面板'"
              placement="bottom"
            >
              <button
                type="button"
                class="asd__icon-btn"
                :class="{ 'is-active': uiPrefs.prefs.pinnedDrawer }"
                :aria-label="uiPrefs.prefs.pinnedDrawer ? '取消钉住' : '钉住为旁路面板'"
                @click="togglePin"
              >
                <Paperclip />
              </button>
            </el-tooltip>
            <!-- 关闭 -->
            <el-tooltip content="关闭" placement="bottom">
              <button
                type="button"
                class="asd__icon-btn"
                aria-label="关闭详情抽屉"
                @click="handleClose"
              >
                <Close />
              </button>
            </el-tooltip>
          </div>
        </div>

        <!-- 区块 1：基础属性 -->
        <section class="asd__section">
          <h4 class="asd__section-title">基础属性</h4>
          <dl class="asd__grid">
            <div>
              <dt>类型</dt>
              <dd>{{ resolveConnectionType(connection) }}</dd>
            </div>
            <div>
              <dt>状态</dt>
              <dd>{{ resolveStatusText(connection?.status) }}</dd>
            </div>
            <div class="asd__grid-full">
              <dt>地址</dt>
              <dd class="asd__mono">{{ resolveConnectionEndpoint(connection) }}</dd>
            </div>
            <div class="asd__grid-full">
              <dt>更新时间</dt>
              <dd class="asd__mono">{{ formatTime(connection?.updatedAt) }}</dd>
            </div>
          </dl>
        </section>

        <!-- 区块 2：操作 -->
        <section class="asd__section">
          <h4 class="asd__section-title">操作</h4>
          <div class="asd__actions">
            <!-- 编辑配置 -->
            <button
              type="button"
              class="asd__action-btn"
              @click="handleEdit"
            >
              编辑配置
            </button>

            <!-- 测试连接 -->
            <el-tooltip
              :content="testBtnTooltip"
              placement="top"
              :disabled="!isIndustrialType"
            >
              <span>
                <button
                  type="button"
                  class="asd__action-btn"
                  :disabled="isIndustrialType || testTesting"
                  @click="handleTest"
                >
                  {{ testTesting ? "测试中…" : "测试连接" }}
                </button>
              </span>
            </el-tooltip>

            <!-- 关联数据点 -->
            <button
              type="button"
              class="asd__action-btn"
              @click="$emit('navigate-datapoints', connection)"
            >
              关联数据点
            </button>

            <!-- 删除接入源 -->
            <button
              type="button"
              class="asd__action-btn asd__action-btn--danger"
              :disabled="deleting"
              @click="handleDelete"
            >
              {{ deleting ? "删除中…" : "删除接入源" }}
            </button>
          </div>
        </section>

        <!-- 区块 3：连通性测试结果（点过测试后显示） -->
        <section v-if="testResult !== null" class="asd__section">
          <h4 class="asd__section-title">连通性测试结果</h4>
          <div class="asd__test-result">
            <!-- 进行中 -->
            <template v-if="testResult.status === 'testing'">
              <StatusBadge tone="info" text="正在测试" />
              <span class="asd__test-msg">正在测试…</span>
            </template>

            <!-- 成功 -->
            <template v-else-if="testResult.status === 'success'">
              <StatusBadge tone="success" :text="`测试通过，耗时 ${testResult.durationMs}ms · ${testResult.testedAt}`" />
            </template>

            <!-- 失败 -->
            <template v-else-if="testResult.status === 'error'">
              <StatusBadge tone="danger" :text="testResult.shortMsg" />
              <details v-if="testResult.detail" class="asd__test-detail">
                <summary class="asd__test-detail-summary">展开详情</summary>
                <pre class="asd__code">{{ testResult.detail }}</pre>
              </details>
            </template>
          </div>
        </section>

        <!-- 区块 4：配置（折叠，默认展开） -->
        <section class="asd__section">
          <button
            type="button"
            class="asd__collapse-trigger"
            @click="configExpanded = !configExpanded"
          >
            <span>配置</span>
            <ArrowDown :class="{ 'is-expanded': configExpanded }" class="asd__collapse-icon" />
          </button>
          <pre v-if="configExpanded" class="asd__code asd__code--config">{{ maskedConfig }}</pre>
        </section>
      </div>
    </template>
  </DcDrawer>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { ElMessage } from "element-plus";
import { ArrowDown, Close, Paperclip } from "@element-plus/icons-vue";
import dayjs from "dayjs";
import { TIME_FORMAT } from "@/constants";
import { deleteAccessSource, testAccessSource } from "@/api/access-source.api";
import { getApiErrorMessage } from "@/utils/request";
import { useUiPrefsStore } from "@/stores/ui-prefs.store";
import { useConfirm } from "@/composables/useConfirm";
import DcDrawer from "@/components/shared/DcDrawer.vue";
import StatusBadge from "@/components/shared/StatusBadge.vue";

type AccessSourceConnection = {
  id: string;
  name?: string;
  type?: string;
  status?: string;
  updatedAt?: string;
  relationalConfig?: {
    dbType?: string;
    host?: string;
    port?: number | string;
    database?: string;
  };
  mqttConfig?: {
    protocol?: string;
    brokerUrl?: string;
    host?: string;
    port?: number | string;
    topic?: string;
    defaultTopic?: string;
  };
  config?: Record<string, unknown>;
  [key: string]: unknown;
};

type TestResult =
  | { status: "testing" }
  | { status: "success"; durationMs: number; testedAt: string }
  | { status: "error"; shortMsg: string; detail: string };

const props = defineProps<{
  modelValue: boolean;
  connection: AccessSourceConnection | null;
  projectId: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  close: [];
  edit: [connection: AccessSourceConnection];
  deleted: [];
  "navigate-datapoints": [connection: AccessSourceConnection];
}>();

const uiPrefs = useUiPrefsStore();
const { confirm } = useConfirm();

const configExpanded = ref(true);
const testTesting = ref(false);
const deleting = ref(false);
const testResult = ref<TestResult | null>(null);

const visible = computed({
  get: () => props.modelValue,
  set: (val: boolean) => {
    emit("update:modelValue", val);
    if (!val) emit("close");
  },
});

// 切换 connection 时重置测试结果
watch(
  () => props.connection?.id,
  () => {
    testResult.value = null;
    configExpanded.value = true;
  },
);

// ── 工业协议列表（测试按钮禁用）──────────────────────────────────────────
const INDUSTRIAL_TYPES = new Set(["opcua", "opcda", "s7", "modbus", "tdengine"]);

const isIndustrialType = computed(() =>
  INDUSTRIAL_TYPES.has(props.connection?.type || ""),
);

const testBtnTooltip = computed(() =>
  isIndustrialType.value
    ? "该协议测试由节点侧执行，暂不支持开发态联调"
    : "",
);

// ── 格式化工具（内联，不 import AccessSourceCard 私有函数）────────────────

function resolveConnectionType(connection: AccessSourceConnection | null): string {
  if (!connection) return "-";
  if (connection.type === "relational") {
    const dbType = connection.relationalConfig?.dbType || "";
    const labels: Record<string, string> = {
      mysql: "MySQL",
      postgresql: "PostgreSQL",
      sqlserver: "SQL Server",
    };
    return labels[dbType] || dbType || "数据库/时序库";
  }
  if (connection.type === "mqtt") return "MQTT Broker";
  const labels: Record<string, string> = {
    kafka: "Kafka Topic",
    http: "HTTP Source",
    websocket: "WebSocket",
    redis: "Redis",
    opcua: "OPC UA",
    opcda: "OPC DA",
    s7: "Siemens S7",
    modbus: "Modbus",
    tdengine: "TDengine",
  };
  return (connection.type && labels[connection.type]) || connection.type || "未知类型";
}

function resolveConnectionEndpoint(connection: AccessSourceConnection | null): string {
  if (!connection) return "-";
  if (connection.type === "relational") {
    const cfg = connection.relationalConfig;
    if (!cfg) return "未配置数据库地址";
    const host = [cfg.host, cfg.port].filter(Boolean).join(":");
    return [host, cfg.database].filter(Boolean).join(" / ") || "未配置数据库";
  }
  if (connection.type === "mqtt") {
    const cfg = connection.mqttConfig;
    if (!cfg) return "未配置 Broker";
    const host = cfg.brokerUrl || cfg.host;
    const endpoint = [host, cfg.port].filter(Boolean).join(":");
    const topic = cfg.topic || cfg.defaultTopic;
    return [endpoint, topic].filter(Boolean).join(" / ") || "未配置 Topic";
  }
  const cfg = (connection.config || {}) as Record<string, unknown>;
  if (connection.type === "kafka") {
    return [cfg["brokers"], cfg["topic"]].filter(Boolean).join(" / ") || "未配置 Topic";
  }
  if (connection.type === "http") {
    return [cfg["method"] || "GET", cfg["baseUrl"]].filter(Boolean).join(" ") || "未配置 URL";
  }
  if (connection.type === "websocket") {
    return [cfg["url"], cfg["topic"]].filter(Boolean).join(" / ") || "未配置 WebSocket";
  }
  if (connection.type === "redis") {
    return [cfg["address"], cfg["keyPattern"] || "*"].filter(Boolean).join(" / ") || "未配置 Redis";
  }
  if (connection.type === "opcua") {
    return [cfg["endpoint"], cfg["securityMode"] || "none"].filter(Boolean).join(" / ") || "未配置 OPC UA";
  }
  if (connection.type === "opcda") {
    return String(cfg["serverProgId"] || cfg["host"] || "未配置 OPC DA");
  }
  if (connection.type === "s7") {
    return [cfg["host"], cfg["port"]].filter(Boolean).join(":") || "未配置 S7";
  }
  if (connection.type === "modbus") {
    if (cfg["mode"] === "rtu") return "RTU / 串口配置";
    return [cfg["host"], cfg["port"]].filter(Boolean).join(":") || "未配置 Modbus";
  }
  if (connection.type === "tdengine") {
    return [cfg["dsn"], cfg["database"]].filter(Boolean).join(" / ") || "未配置 TDengine";
  }
  return "等待接入配置";
}

function resolveStatusTone(status?: string): "success" | "danger" | "warning" | "muted" {
  switch (status) {
    case "connected": return "success";
    case "disconnected": return "muted";
    case "error":
    case "degraded": return "danger";
    default: return "muted";
  }
}

function resolveStatusText(status?: string): string {
  const labels: Record<string, string> = {
    connected: "在线",
    disconnected: "离线",
    error: "异常",
    degraded: "降级",
    unknown: "未知",
  };
  return labels[status || "unknown"] || status || "未知";
}

function formatTime(value?: string | null): string {
  if (!value) return "-";
  return dayjs(value).format(TIME_FORMAT);
}

// 递归遮蔽敏感字段（password / secret / token）
function deepMask(obj: unknown): unknown {
  if (!obj || typeof obj !== "object") return obj;
  if (Array.isArray(obj)) return obj.map(deepMask);
  const result: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(obj as Record<string, unknown>)) {
    const lower = k.toLowerCase();
    if (lower === "password" || lower === "secret" || lower === "token") {
      result[k] = "***";
    } else {
      result[k] = deepMask(v);
    }
  }
  return result;
}

const maskedConfig = computed(() => {
  if (!props.connection) return "{}";
  const raw =
    props.connection.config ||
    props.connection.relationalConfig ||
    props.connection.mqttConfig ||
    {};
  try {
    return JSON.stringify(deepMask(raw), null, 2);
  } catch {
    return "{}";
  }
});

// ── 交互 ──────────────────────────────────────────────────────────────────

function togglePin() {
  uiPrefs.setPinnedDrawer(!uiPrefs.prefs.pinnedDrawer);
}

function handleClose() {
  visible.value = false;
}

// 点击「编辑配置」：先关抽屉再上抛，避免抽屉与编辑对话框视觉重叠
function handleEdit() {
  if (!props.connection) return;
  emit("edit", props.connection);
  visible.value = false;
}

async function handleTest() {
  if (!props.connection || testTesting.value) return;
  testTesting.value = true;
  testResult.value = { status: "testing" };
  const startAt = performance.now();
  try {
    await testAccessSource(props.projectId, {
      type: props.connection.type,
      config: props.connection.config || props.connection.relationalConfig || props.connection.mqttConfig || {},
    });
    const durationMs = Math.round(performance.now() - startAt);
    testResult.value = {
      status: "success",
      durationMs,
      testedAt: dayjs().format(TIME_FORMAT),
    };
  } catch (err) {
    const shortMsg = getApiErrorMessage(err, "测试失败");
    testResult.value = {
      status: "error",
      shortMsg,
      detail: err instanceof Error ? err.message : String(err),
    };
  } finally {
    testTesting.value = false;
  }
}

async function handleDelete() {
  if (!props.connection) return;
  const ok = await confirm(
    `将删除接入源「${props.connection.name || props.connection.id}」。后端引用检查未启用，相关数据点可能受影响。`,
    { title: "删除接入源", confirmText: "删除", type: "error" },
  );
  if (!ok) return;
  deleting.value = true;
  try {
    await deleteAccessSource(props.projectId, props.connection.id);
    ElMessage.success("接入源已删除");
    emit("deleted");
    visible.value = false;
  } catch (err) {
    ElMessage.error(getApiErrorMessage(err, "删除失败"));
  } finally {
    deleting.value = false;
  }
}
</script>

<style scoped>
/* ── 整体容器 ── */
.asd {
  display: flex;
  flex-direction: column;
  gap: 0;
  height: 100%;
  overflow-y: auto;
}

/* ── 头部 ── */
.asd__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 16px 10px;
  border-bottom: 1px solid var(--dc-border);
}

.asd__header-left {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.asd__header-right {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.asd__name {
  font-size: 16px;
  font-weight: 700;
  color: var(--dc-text);
  overflow-wrap: anywhere;
}

/* ── icon 按钮 ── */
.asd__icon-btn {
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

.asd__icon-btn svg {
  width: 15px;
  height: 15px;
}

.asd__icon-btn:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.asd__icon-btn.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

/* ── 区块通用 ── */
.asd__section {
  padding: 16px;
  border-bottom: 1px solid var(--dc-border);
}

.asd__section:last-child {
  border-bottom: 0;
}

.asd__section-title {
  margin: 0 0 10px;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

/* ── 基础属性 grid ── */
.asd__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin: 0;
}

.asd__grid > div {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface);
}

.asd__grid-full {
  grid-column: 1 / -1;
}

.asd__grid dt {
  margin: 0 0 4px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.asd__grid dd {
  margin: 0;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 600;
  overflow-wrap: anywhere;
}

.asd__mono {
  font-family: var(--dc-font-mono);
}

/* ── 操作按钮行 ── */
.asd__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.asd__action-btn {
  height: 32px;
  padding: 0 14px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 600;
  transition: background 0.18s, border-color 0.18s, color 0.18s;
}

.asd__action-btn:hover:not(:disabled) {
  background: var(--dc-primary-soft);
  border-color: rgba(29, 78, 216, 0.24);
  color: var(--dc-primary);
}

.asd__action-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.asd__action-btn--danger {
  color: var(--dc-danger, #dc2626);
  border-color: rgba(220, 38, 38, 0.2);
  background: var(--dc-danger-soft, rgba(220, 38, 38, 0.06));
}

.asd__action-btn--danger:hover:not(:disabled) {
  background: rgba(220, 38, 38, 0.12);
  border-color: rgba(220, 38, 38, 0.4);
  color: var(--dc-danger, #dc2626);
}

/* ── 测试结果 ── */
.asd__test-result {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.asd__test-msg {
  font-size: 13px;
  color: var(--dc-text-secondary);
}

.asd__test-detail {
  margin-top: 4px;
}

.asd__test-detail-summary {
  color: var(--dc-text-muted);
  font-size: 12px;
  cursor: pointer;
}

/* ── 配置折叠 ── */
.asd__collapse-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--dc-text);
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 700;
  margin-bottom: 10px;
}

.asd__collapse-icon {
  width: 14px;
  height: 14px;
  transition: transform 0.2s;
}

.asd__collapse-icon.is-expanded {
  transform: rotate(180deg);
}

/* ── 代码块 ── */
.asd__code {
  max-height: 200px;
  overflow: auto;
  margin: 0;
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

.asd__code--config {
  margin-top: 0;
}
</style>
