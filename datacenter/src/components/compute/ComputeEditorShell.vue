<template>
  <section class="compute-editor">
    <div v-if="loading && !activeDraft" class="compute-editor__loading">
      <el-skeleton :rows="10" animated />
    </div>

    <EmptyState
      v-else-if="error && !activeDraft"
      icon-name="warning"
      title="计算单元详情不可用"
      :description="error"
    />

    <EmptyState
      v-else-if="!activeDraft"
      icon-name="compute"
      title="选择一个计算单元"
      description="从左侧资源树选择计算单元，右侧会打开编辑标签。"
    />

    <template v-else>
      <header class="compute-editor__tabs">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="compute-editor__file-tab"
          :class="{ 'is-active': tab.id === activeId, 'is-dirty': tab.dirty }"
          @click="$emit('activate-tab', tab.id)"
        >
          <span>{{ tab.name }}</span>
          <em v-if="tab.dirty">*</em>
          <button
            type="button"
            class="compute-editor__close"
            aria-label="关闭标签"
            @click.stop="$emit('close-tab', tab.id)"
          >
            <IconTablerX />
          </button>
        </button>
      </header>

      <div class="compute-editor__toolbar">
        <div class="compute-editor__title-wrap">
          <div class="compute-editor__meta">
            <StatusBadge
              :tone="statusTone(activeDraft.status)"
              :text="statusText(activeDraft.status)"
            />
          </div>
          <input
            v-model="activeDraft.name"
            class="compute-editor__name-input"
            aria-label="计算单元名称"
            @input="markDirty"
          />
          <div class="compute-editor__subline">
            <span class="compute-editor__lang-pill">{{ langText(activeDraft.lang) }}</span>
            <span class="compute-editor__output-inline">{{ previewOutputPath }}</span>
          </div>
        </div>

        <div class="compute-editor__actions">
          <button
            type="button"
            class="compute-editor__icon-action"
            :title="activeDraft.isEnabled === false ? '启用' : '停用'"
            :aria-label="activeDraft.isEnabled === false ? '启用' : '停用'"
            :disabled="saving"
            @click="$emit('toggle-enabled', activeDraft.id, activeDraft.isEnabled === false)"
          >
            <IconTablerPlayerPlay
              v-if="activeDraft.isEnabled === false"
              class="compute-editor__action-icon"
            />
            <IconTablerPlayerPause v-else class="compute-editor__action-icon" />
          </button>
          <button
            type="button"
            class="compute-editor__icon-action"
            :class="{ 'is-active': inspectorOpen }"
            title="输出配置"
            aria-label="输出配置"
            @click="inspectorOpen = !inspectorOpen"
          >
            <IconTablerSettings class="compute-editor__action-icon" />
          </button>
          <button
            type="button"
            class="compute-editor__icon-action is-danger"
            title="删除"
            aria-label="删除"
            :disabled="deleting"
            @click="$emit('delete-unit', activeDraft.id)"
          >
            <IconTablerTrash class="compute-editor__action-icon" />
          </button>
          <button
            type="button"
            class="compute-editor__save"
            title="保存"
            aria-label="保存"
            :disabled="saving || !activeDraft.dirty"
            @click="$emit('save', activeDraft.id)"
          >
            <IconTablerDeviceFloppy class="compute-editor__action-icon" />
            <span>保存</span>
          </button>
        </div>
      </div>

      <main class="compute-editor__main">
        <section class="compute-editor__code">
          <div class="compute-editor__code-head">
            <span class="compute-editor__code-title">
              <IconTablerCode class="compute-editor__head-icon" />
              代码
            </span>
            <div class="compute-editor__code-actions">
              <button
                type="button"
                class="compute-editor__tool-btn"
                title="插入数据点"
                aria-label="插入数据点"
                @click="openDatapointPicker('script')"
              >
                <IconTablerDatabaseImport class="compute-editor__action-icon" />
              </button>
              <button
                type="button"
                class="compute-editor__tool-btn"
                title="Dry-run"
                aria-label="Dry-run"
                :disabled="debugRunning || activeDraft.dirty"
                @click="quickDryRun"
              >
                <IconTablerPlayerPlay class="compute-editor__action-icon" />
              </button>
              <em>{{ langText(activeDraft.lang) }}</em>
            </div>
          </div>
          <MonacoEditor
            ref="monacoEditorRef"
            v-model="activeDraft.code"
            class="compute-editor__monaco"
            :language="monacoLanguage"
            height="100%"
            @change="markDirty"
          />
        </section>

        <section v-if="inspectorOpen" class="compute-editor__inspector">
          <div class="compute-editor__inspector-head">
            <div class="compute-editor__inspector-title">
              <IconTablerSettings class="compute-editor__head-icon" />
              <h3>输出</h3>
            </div>
            <button
              type="button"
              class="compute-editor__ghost-icon"
              title="关闭输出配置"
              aria-label="关闭输出配置"
              @click="inspectorOpen = false"
            >
              <IconTablerX class="compute-editor__action-icon" />
            </button>
            <div class="compute-editor__output-path">{{ previewOutputPath }}</div>
          </div>
          <label class="compute-editor__field">
            <span>超时</span>
            <input
              v-model.number="activeDraft.timeoutMs"
              type="number"
              min="1"
              max="120000"
              @input="markDirty"
            />
          </label>
          <label class="compute-editor__field">
            <span>输出名</span>
            <input
              v-model="outputName"
              placeholder="result"
              @input="updateOutputBindings"
            />
          </label>
        </section>
      </main>

      <footer
        class="compute-editor__panel-tabs"
        :class="{ 'is-collapsed': panelCollapsed }"
      >
        <div v-if="panelCollapsed" class="compute-editor__panel-summary">
          <span>输入 {{ inputCount }}</span>
          <span>触发 {{ triggerText }}</span>
          <span>依赖 {{ dependencyCount }}</span>
          <span>调试 {{ debugStateText }}</span>
        </div>
        <div v-else class="compute-editor__panel-tablist">
          <button
            v-for="tab in panelTabs"
            :key="tab.id"
            type="button"
            class="compute-editor__panel-tab"
            :class="{ 'is-active': activePanel === tab.id }"
            :title="tab.label"
            @click="activePanel = tab.id"
          >
            <component :is="tab.icon" class="compute-editor__panel-tab-icon" />
            <span>{{ tab.label }}</span>
          </button>
        </div>
        <button
          type="button"
          class="compute-editor__panel-toggle"
          :title="panelCollapsed ? '展开' : '收起'"
          :aria-label="panelCollapsed ? '展开底部面板' : '收起底部面板'"
          @click="panelCollapsed = !panelCollapsed"
        >
          <IconTablerChevronUp
            class="compute-editor__panel-toggle-icon"
            :class="{ 'is-collapsed': panelCollapsed }"
          />
        </button>
      </footer>

      <section v-show="!panelCollapsed" class="compute-editor__panel">
        <template v-if="activePanel === 'inputs'">
          <div class="compute-editor__panel-head">
            <h3>脚本参数</h3>
            <div class="compute-editor__panel-actions">
              <button
                type="button"
                class="compute-editor__tool-btn"
                title="添加参数"
                aria-label="添加参数"
                @click="addInput"
              >
                <IconTablerPlus class="compute-editor__action-icon" />
              </button>
              <button
                type="button"
                class="compute-editor__tool-btn"
                title="从数据点选择"
                aria-label="从数据点选择"
                @click="openDatapointPicker('input')"
              >
                <IconTablerDatabaseImport class="compute-editor__action-icon" />
              </button>
            </div>
          </div>
          <div class="compute-editor__input-search">
            <input
              v-model="datapointSearch"
              placeholder="搜索数据点路径或名称"
              @keyup.enter="searchDatapoints"
            />
            <button
              type="button"
              title="搜索"
              aria-label="搜索"
              @click="searchDatapoints"
            >
              <IconTablerSearch class="compute-editor__action-icon" />
            </button>
          </div>
          <div v-if="datapointOptions.length" class="compute-editor__datapoints">
            <button
              v-for="point in datapointOptions"
              :key="point.id"
              type="button"
              @click="addInputFromDatapoint(point)"
            >
              <strong>{{ point.name }}</strong>
              <span>{{ point.path }}</span>
            </button>
          </div>
          <div
            v-if="!activeDraft.inputRows.length"
            class="compute-editor__empty compute-editor__empty-action"
          >
            <strong>还没有脚本参数</strong>
            <span>脚本参数用于把数据点值传入代码里的 input 对象。</span>
            <div>
              <button type="button" @click="addInput">添加参数</button>
              <button type="button" @click="openDatapointPicker('input')">
                从数据点选择
              </button>
            </div>
          </div>
          <div v-else class="compute-editor__mapping-head">
            <span>参数名</span>
            <span>数据点路径</span>
            <span></span>
          </div>
          <div
            v-for="(row, index) in activeDraft.inputRows"
            :key="row.uid"
            class="compute-editor__mapping-row"
          >
            <input v-model="row.name" placeholder="入参名" @input="markDirty" />
            <input
              v-model="row.path"
              placeholder="数据点 path"
              @input="markDirty"
            />
            <button type="button" @click="removeInput(index)">删除</button>
          </div>
        </template>

        <template v-else-if="activePanel === 'trigger'">
          <div class="compute-editor__panel-head">
            <h3>触发方式</h3>
            <div class="compute-editor__segmented">
              <button
                v-for="item in triggerTypes"
                :key="item.id"
                type="button"
                :class="{ 'is-active': activeDraft.triggerType === item.id }"
                @click="setTriggerType(item.id)"
              >
                {{ item.label }}
              </button>
            </div>
          </div>
          <div v-if="activeDraft.triggerType === 'timer'" class="compute-editor__form-grid">
            <label class="compute-editor__field">
              <span>间隔秒数</span>
              <input
                v-model.number="activeDraft.triggerConfig.intervalSeconds"
                type="number"
                min="1"
                @input="markDirty"
              />
            </label>
          </div>
          <div
            v-else-if="activeDraft.triggerType === 'datapoint_change'"
            class="compute-editor__form-grid"
          >
            <label class="compute-editor__field">
              <span>数据点路径</span>
              <input
                v-model="activeDraft.triggerConfig.path"
                placeholder="db.localhost.temperature"
                @input="markDirty"
              />
            </label>
          </div>
          <div v-else class="compute-editor__empty">无配置项</div>
        </template>

        <template v-else-if="activePanel === 'dependencies'">
          <div class="compute-editor__panel-head">
            <h3>依赖库</h3>
            <button type="button" class="compute-editor__small" @click="$emit('refresh-dependencies')">
              刷新
            </button>
          </div>
          <div v-if="dependenciesError" class="compute-editor__empty">
            {{ dependenciesError }}
          </div>
          <div v-else class="compute-editor__dependency-list">
            <label
              v-for="dep in filteredDependencies"
              :key="dep.id"
              class="compute-editor__dependency"
            >
              <input
                type="checkbox"
                :checked="isDependencyChecked(dep.id)"
                @change="toggleDependency(dep.id)"
              />
              <span>
                <strong>{{ dep.name }}</strong>
                <em>{{ dep.runtime }} · {{ dep.importName || dep.name }}</em>
                <small>{{ dep.description || "内置依赖" }}</small>
              </span>
            </label>
          </div>
        </template>

        <template v-else-if="activePanel === 'debug'">
          <div class="compute-editor__panel-head">
            <h3>调试</h3>
            <div class="compute-editor__debug-actions">
              <button
                type="button"
                class="compute-editor__small"
                :disabled="debugRunning || activeDraft.dirty"
                @click="executeDebug(true)"
              >
                Dry-run
              </button>
              <button
                type="button"
                class="compute-editor__danger"
                :disabled="debugRunning || activeDraft.dirty"
                @click="executeDebug(false)"
              >
                真实执行
              </button>
            </div>
          </div>
          <div class="compute-editor__debug-layout">
            <label class="compute-editor__debug-input">
              <span>输入 JSON</span>
              <textarea v-model="debugInputText" spellcheck="false" />
            </label>
            <div class="compute-editor__debug-results">
              <section class="compute-editor__debug-card">
                <strong>返回值</strong>
                <pre>{{ debugOutputText }}</pre>
              </section>
              <section class="compute-editor__debug-card">
                <strong>日志</strong>
                <pre>{{ debugLogsText }}</pre>
              </section>
              <section class="compute-editor__debug-card">
                <strong>副作用</strong>
                <pre>{{ debugSideEffectsText }}</pre>
              </section>
              <section
                class="compute-editor__debug-card"
                :class="{ 'has-error': Boolean(debugErrorText) }"
              >
                <strong>错误</strong>
                <pre>{{ debugErrorText || "-" }}</pre>
              </section>
            </div>
          </div>
        </template>
      </section>
    </template>

    <DcDialog
      v-model="datapointPickerVisible"
      title="插入数据点"
      width="860px"
      body-max-height="620px"
    >
      <div class="compute-editor__picker">
        <div class="compute-editor__picker-toolbar">
          <el-input
            v-model="datapointPickerKeyword"
            size="small"
            clearable
            placeholder="搜索名称或路径"
            @keyup.enter="reloadPickerDatapoints"
          />
          <select
            v-model="datapointPickerSource"
            class="compute-editor__picker-select"
            aria-label="来源类型"
            @change="reloadPickerDatapoints"
          >
            <option value="">全部来源</option>
            <option value="mqtt.subscription">MQTT</option>
            <option value="db.query">数据库</option>
            <option value="http">HTTP</option>
            <option value="manual">手动</option>
          </select>
          <select
            v-model="datapointPickerDataType"
            class="compute-editor__picker-select"
            aria-label="数据类型"
            @change="reloadPickerDatapoints"
          >
            <option value="">全部类型</option>
            <option value="object">object</option>
            <option value="number">number</option>
            <option value="string">string</option>
            <option value="boolean">boolean</option>
          </select>
          <select
            v-model="datapointPickerStatus"
            class="compute-editor__picker-select"
            aria-label="状态"
            @change="reloadPickerDatapoints"
          >
            <option value="">全部状态</option>
            <option value="active">正常</option>
            <option value="inactive">停用</option>
            <option value="error">异常</option>
            <option value="unknown">未知</option>
          </select>
          <button
            type="button"
            class="compute-editor__small"
            @click="reloadPickerDatapoints"
          >
            搜索
          </button>
        </div>
        <div v-if="datapointPickerLoading" class="compute-editor__picker-loading">
          <el-skeleton :rows="5" animated />
        </div>
        <div v-else class="compute-editor__picker-table-wrap">
          <table
            v-if="datapointPickerOptions.length"
            class="compute-editor__picker-table"
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
                v-for="point in datapointPickerOptions"
                :key="point.id"
                :class="{ 'is-selected': selectedDatapointId === String(point.id) }"
                @click="selectedDatapointId = String(point.id)"
                @dblclick="confirmSelectedDatapoint(point)"
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
                    class="compute-editor__picker-status"
                    :class="`is-${point.status || 'unknown'}`"
                  >
                    {{ datapointStatusText(point.status) }}
                  </span>
                </td>
                <td>
                  <span
                    class="compute-editor__picker-row-action"
                    role="button"
                    tabindex="0"
                    @click.stop="confirmSelectedDatapoint(point)"
                    @keydown.enter.stop.prevent="confirmSelectedDatapoint(point)"
                  >
                    插入
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
          <div
            v-if="datapointPickerOptions.length === 0"
            class="compute-editor__empty compute-editor__picker-empty"
          >
            <strong>没有匹配的数据点</strong>
            <span>换个关键词，或放宽来源、类型、状态筛选。</span>
          </div>
        </div>
        <div class="compute-editor__picker-footer">
          <div class="compute-editor__picker-count">
            共 {{ datapointPickerTotal }} 条
            <span v-if="selectedDatapoint">已选 {{ selectedDatapoint.name }}</span>
          </div>
          <div class="compute-editor__picker-pager">
            <button
              type="button"
              :disabled="datapointPickerPage <= 1 || datapointPickerLoading"
              @click="changePickerPage(datapointPickerPage - 1)"
            >
              上一页
            </button>
            <span>{{ datapointPickerPage }} / {{ datapointPickerTotalPages }}</span>
            <button
              type="button"
              :disabled="
                datapointPickerPage >= datapointPickerTotalPages ||
                datapointPickerLoading
              "
              @click="changePickerPage(datapointPickerPage + 1)"
            >
              下一页
            </button>
            <button
              type="button"
              class="compute-editor__picker-confirm"
              :disabled="!selectedDatapoint"
              @click="confirmSelectedDatapoint()"
              @mousedown.prevent="confirmSelectedDatapoint()"
            >
              插入
            </button>
          </div>
        </div>
      </div>
    </DcDialog>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import IconTablerChevronUp from "~icons/tabler/chevron-up";
import IconTablerCode from "~icons/tabler/code";
import IconTablerDatabaseImport from "~icons/tabler/database-import";
import IconTablerDeviceFloppy from "~icons/tabler/device-floppy";
import IconTablerGitFork from "~icons/tabler/git-fork";
import IconTablerPlugConnected from "~icons/tabler/plug-connected";
import IconTablerPlayerPause from "~icons/tabler/player-pause";
import IconTablerPlayerPlay from "~icons/tabler/player-play";
import IconTablerPlus from "~icons/tabler/plus";
import IconTablerSearch from "~icons/tabler/search";
import IconTablerSettings from "~icons/tabler/settings";
import IconTablerTerminal2 from "~icons/tabler/terminal-2";
import IconTablerTrash from "~icons/tabler/trash";
import IconTablerX from "~icons/tabler/x";
import type { Datapoint } from "@/api/schemas/datapoint.schema";
import type {
  ComputeDependency,
  ComputeLang,
  ComputeRunResult,
} from "@/api/schemas/compute.schema";
import { debugComputeUnit, runComputeUnit } from "@/api/compute.api";
import { getDatapoints } from "@/api/datapoint.api";
import { getApiErrorMessage } from "@/utils/request";
import MonacoEditor from "@/components/MonacoEditor.vue";
import DcDialog from "@/components/shared/DcDialog.vue";
import EmptyState from "@/components/shared/EmptyState.vue";
import StatusBadge from "@/components/shared/StatusBadge.vue";
import type { ComputeDraft, ComputeEditorTab } from "./computeEditorModel";

type MonacoEditorExpose = InstanceType<typeof MonacoEditor> & {
  insertText?: (text: string) => void;
};

const props = withDefaults(
  defineProps<{
    projectId: string;
    tabs: ComputeEditorTab[];
    activeId: string | null;
    activeDraft: ComputeDraft | null;
    loading?: boolean;
    saving?: boolean;
    deleting?: boolean;
    error?: string;
    dependencies?: ComputeDependency[];
    dependenciesLoading?: boolean;
    dependenciesError?: string;
  }>(),
  {
    loading: false,
    saving: false,
    deleting: false,
    error: "",
    dependencies: () => [],
    dependenciesLoading: false,
    dependenciesError: "",
  },
);

const emit = defineEmits<{
  (event: "activate-tab", id: string): void;
  (event: "close-tab", id: string): void;
  (event: "save", id: string): void;
  (event: "toggle-enabled", id: string, enabled: boolean): void;
  (event: "delete-unit", id: string): void;
  (event: "mark-dirty", id: string): void;
  (event: "refresh-dependencies"): void;
}>();

const activePanel = ref("inputs");
const panelCollapsed = ref(true);
const inspectorOpen = ref(false);
const datapointSearch = ref("");
const datapointOptions = ref<Datapoint[]>([]);
const monacoEditorRef = ref<MonacoEditorExpose | null>(null);
const datapointPickerVisible = ref(false);
const datapointPickerMode = ref<"script" | "input">("script");
const datapointPickerKeyword = ref("");
const datapointPickerSource = ref("");
const datapointPickerDataType = ref("");
const datapointPickerStatus = ref("");
const datapointPickerPage = ref(1);
const datapointPickerPageSize = 12;
const datapointPickerTotal = ref(0);
const datapointPickerOptions = ref<Datapoint[]>([]);
const datapointPickerLoading = ref(false);
const selectedDatapointId = ref<string | null>(null);
const debugInputText = ref("{}");
const debugRunning = ref(false);
const debugResult = ref<ComputeRunResult | null>(null);
const debugError = ref("");

const panelTabs = [
  { id: "inputs", label: "输入", icon: IconTablerPlugConnected },
  { id: "trigger", label: "触发", icon: IconTablerPlayerPlay },
  { id: "dependencies", label: "依赖", icon: IconTablerGitFork },
  { id: "debug", label: "调试", icon: IconTablerTerminal2 },
];

const triggerTypes = [
  { id: "manual", label: "手动" },
  { id: "timer", label: "定时" },
  { id: "datapoint_change", label: "数据点变化" },
];

const monacoLanguage = computed(() => {
  if (props.activeDraft?.lang === "python") return "python";
  return "javascript";
});

const previewOutputPath = computed(() => {
  const name = props.activeDraft?.name?.trim() || "unnamed";
  return `calc.${name.replace(/\s+/g, "_")}.${outputName.value || "result"}`;
});

const outputName = computed({
  get() {
    const outputs = props.activeDraft?.outputBindings.outputs;
    if (Array.isArray(outputs) && outputs[0]) {
      const first = outputs[0] as Record<string, unknown>;
      return String(first.name || "result");
    }
    return "result";
  },
  set(value: string) {
    if (!props.activeDraft) return;
    props.activeDraft.outputBindings = {
      ...props.activeDraft.outputBindings,
      outputs: [{ name: value || "result", dataType: "object" }],
    };
  },
});

const filteredDependencies = computed(() => {
  const runtime = props.activeDraft?.lang === "python" ? "python" : "javascript";
  return props.dependencies.filter((item) => item.runtime === runtime);
});

const inputCount = computed(() => props.activeDraft?.inputRows.length || 0);
const dependencyCount = computed(() => props.activeDraft?.dependencies.length || 0);
const triggerText = computed(() => {
  const triggerType = props.activeDraft?.triggerType || "manual";
  return triggerTypes.find((item) => item.id === triggerType)?.label || "手动";
});

const debugStateText = computed(() => {
  if (debugRunning.value) return "运行中";
  if (debugErrorText.value) return "异常";
  if (debugResult.value) return "完成";
  return "未运行";
});

const debugOutputText = computed(() =>
  formatDebugValue(debugResult.value?.output),
);

const debugLogsText = computed(() => {
  const logs = debugResult.value?.logs || [];
  return logs.length ? logs.join("\n") : "-";
});

const debugSideEffectsText = computed(() =>
  formatDebugValue(debugResult.value?.sideEffects || []),
);

const debugErrorText = computed(
  () =>
    debugError.value ||
    debugResult.value?.errorMessage ||
    debugResult.value?.error ||
    "",
);

const selectedDatapoint = computed(() =>
  datapointPickerOptions.value.find(
    (point) => String(point.id) === selectedDatapointId.value,
  ),
);

const datapointPickerTotalPages = computed(() =>
  Math.max(1, Math.ceil(datapointPickerTotal.value / datapointPickerPageSize)),
);

watch(
  () => props.activeId,
  () => {
    activePanel.value = "inputs";
    panelCollapsed.value = true;
    inspectorOpen.value = false;
    debugResult.value = null;
    debugError.value = "";
    debugInputText.value = buildDefaultDebugInput();
  },
);

watch(
  () => props.activeDraft?.inputRows,
  () => {
    if (!debugResult.value && !debugError.value) {
      debugInputText.value = buildDefaultDebugInput();
    }
  },
  { deep: true },
);

function markDirty() {
  if (props.activeDraft) {
    emit("mark-dirty", props.activeDraft.id);
  }
}

function updateOutputBindings() {
  markDirty();
}

function addInput() {
  if (!props.activeDraft) return;
  props.activeDraft.inputRows.push({
    uid: crypto.randomUUID(),
    name: `input${props.activeDraft.inputRows.length + 1}`,
    path: "",
  });
  markDirty();
}

function addInputFromDatapoint(point: Datapoint) {
  if (!props.activeDraft) return;
  const nextName = point.name || `input${props.activeDraft.inputRows.length + 1}`;
  props.activeDraft.inputRows.push({
    uid: crypto.randomUUID(),
    name: nextName.replace(/[^\w]/g, "_"),
    path: point.path,
    datapointId: point.id,
  });
  markDirty();
}

function removeInput(index: number) {
  if (!props.activeDraft) return;
  props.activeDraft.inputRows.splice(index, 1);
  markDirty();
}

async function searchDatapoints() {
  if (!props.projectId) return;
  const result = await getDatapoints(props.projectId, {
    search: datapointSearch.value,
    page: 1,
    pageSize: 8,
  });
  datapointOptions.value = result.list;
}

async function openDatapointPicker(mode: "script" | "input") {
  datapointPickerMode.value = mode;
  datapointPickerVisible.value = true;
  selectedDatapointId.value = null;
  await reloadPickerDatapoints();
}

async function reloadPickerDatapoints() {
  datapointPickerPage.value = 1;
  await loadPickerDatapoints();
}

async function loadPickerDatapoints() {
  if (!props.projectId) return;
  datapointPickerLoading.value = true;
  try {
    const params: Record<string, unknown> = {
      search: datapointPickerKeyword.value,
      page: datapointPickerPage.value,
      pageSize: datapointPickerPageSize,
    };
    if (datapointPickerSource.value) {
      params.type = datapointPickerSource.value;
      params.sourceType = datapointPickerSource.value;
    }
    if (datapointPickerDataType.value) {
      params.dataType = datapointPickerDataType.value;
    }
    if (datapointPickerStatus.value) {
      params.status = datapointPickerStatus.value;
    }
    const result = await getDatapoints(props.projectId, params);
    datapointPickerOptions.value = result.list;
    datapointPickerTotal.value = Number(
      result.pagination?.total ?? result.list.length,
    );
    if (
      selectedDatapointId.value &&
      !result.list.some((point) => String(point.id) === selectedDatapointId.value)
    ) {
      selectedDatapointId.value = null;
    }
  } finally {
    datapointPickerLoading.value = false;
  }
}

async function changePickerPage(page: number) {
  const nextPage = Math.min(Math.max(1, page), datapointPickerTotalPages.value);
  if (nextPage === datapointPickerPage.value) return;
  datapointPickerPage.value = nextPage;
  await loadPickerDatapoints();
}

function confirmSelectedDatapoint(point?: Datapoint) {
  const target = point || selectedDatapoint.value;
  if (!target) return;
  datapointPickerVisible.value = false;
  try {
    if (datapointPickerMode.value === "input") {
      addInputFromDatapoint(target);
    } else {
      insertDatapointIntoScript(target);
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "插入数据点失败"));
  }
}

function insertDatapointIntoScript(point: Datapoint) {
  const expression = `input[${JSON.stringify(point.path)}]`;
  monacoEditorRef.value?.insertText?.(expression);
  markDirty();
}

async function executeDebug(dryRun: boolean) {
  if (!props.activeDraft) return;
  if (props.activeDraft.dirty) {
    ElMessage.warning("请先保存后再调试");
    return;
  }
  let input: Record<string, unknown>;
  try {
    input = parseDebugInput();
  } catch (error) {
    debugError.value =
      error instanceof Error ? error.message : "输入 JSON 格式无效";
    return;
  }

  if (!dryRun) {
    try {
      await ElMessageBox.confirm(
        "真实执行可能写入输出数据点或触发外部副作用。",
        "确认真实执行",
        {
          confirmButtonText: "真实执行",
          cancelButtonText: "取消",
          type: "warning",
        },
      );
    } catch {
      return;
    }
  }

  debugRunning.value = true;
  debugResult.value = null;
  debugError.value = "";
  try {
    debugResult.value = dryRun
      ? await debugComputeUnit(props.projectId, props.activeDraft.id, input, true)
      : await runComputeUnit(props.projectId, props.activeDraft.id, input);
    ElMessage.success(dryRun ? "Dry-run 完成" : "真实执行完成");
  } catch (error) {
    debugError.value = getApiErrorMessage(error, "调试执行失败");
  } finally {
    debugRunning.value = false;
  }
}

async function quickDryRun() {
  activePanel.value = "debug";
  panelCollapsed.value = false;
  await executeDebug(true);
}

function setTriggerType(type: string) {
  if (!props.activeDraft) return;
  props.activeDraft.triggerType = type;
  if (type === "timer") {
    props.activeDraft.triggerConfig = {
      intervalSeconds: props.activeDraft.triggerConfig.intervalSeconds || 60,
    };
  } else if (type === "datapoint_change") {
    props.activeDraft.triggerConfig = {
      path: props.activeDraft.triggerConfig.path || "",
    };
  } else {
    props.activeDraft.triggerConfig = {};
  }
  markDirty();
}

function isDependencyChecked(id: string) {
  return props.activeDraft?.dependencies.some((item) => item.id === id) || false;
}

function toggleDependency(id: string) {
  if (!props.activeDraft) return;
  if (isDependencyChecked(id)) {
    props.activeDraft.dependencies = props.activeDraft.dependencies.filter(
      (item) => item.id !== id,
    );
  } else {
    props.activeDraft.dependencies.push({ id });
  }
  markDirty();
}

function parseDebugInput(): Record<string, unknown> {
  let parsed: unknown;
  try {
    parsed = JSON.parse(debugInputText.value || "{}");
  } catch {
    throw new Error("输入 JSON 格式无效");
  }
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("输入 JSON 必须是对象");
  }
  return parsed as Record<string, unknown>;
}

function buildDefaultDebugInput() {
  const input = Object.fromEntries(
    (props.activeDraft?.inputRows || [])
      .filter((row) => row.name.trim())
      .map((row) => [row.name.trim(), ""]),
  );
  return JSON.stringify(input, null, 2);
}

function formatDebugValue(value: unknown) {
  if (value === undefined || value === null) return "-";
  if (typeof value === "string") return value;
  return JSON.stringify(value, null, 2);
}

const langText = (lang?: ComputeLang | string) => {
  const map: Record<string, string> = {
    js: "JavaScript",
    javascript: "JavaScript",
    python: "Python",
  };
  return map[lang || ""] || "未知语言";
};

const statusText = (status?: string) => {
  const map: Record<string, string> = {
    enabled: "启用",
    idle: "空闲",
    running: "运行中",
    error: "异常",
    disabled: "停用",
  };
  return map[status || ""] || "未知";
};

const datapointStatusText = (status?: string) => {
  const map: Record<string, string> = {
    active: "正常",
    inactive: "停用",
    error: "异常",
    unknown: "未知",
  };
  return map[status || ""] || "未知";
};

const sourceTypeText = (sourceType?: string) => {
  const map: Record<string, string> = {
    "mqtt.subscription": "MQTT",
    "db.query": "数据库",
    http: "HTTP",
    manual: "手动",
  };
  return map[sourceType || ""] || sourceType || "-";
};

const statusTone = (status?: string) => {
  if (status === "running" || status === "enabled" || status === "idle")
    return "success";
  if (status === "error") return "danger";
  if (status === "disabled") return "muted";
  return "info";
};
</script>

<style scoped>
.compute-editor {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--dc-surface);
}

.compute-editor__loading {
  padding: 20px;
}

.compute-editor__tabs {
  height: 38px;
  display: flex;
  align-items: flex-end;
  gap: 2px;
  padding: 0 8px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  overflow-x: auto;
}

.compute-editor__file-tab {
  height: 32px;
  max-width: 220px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 6px 0 10px;
  border: 1px solid transparent;
  border-bottom: none;
  border-radius: var(--dc-radius-sm) var(--dc-radius-sm) 0 0;
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  font-weight: 700;
}

.compute-editor__file-tab.is-active {
  border-color: var(--dc-border);
  background: var(--dc-surface);
  color: var(--dc-primary);
}

.compute-editor__file-tab span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__file-tab em {
  color: var(--dc-warning);
  font-style: normal;
}

.compute-editor__close {
  width: 20px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: inherit;
}

.compute-editor__close:hover {
  background: var(--dc-surface-muted);
}

.compute-editor__toolbar {
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface);
}

.compute-editor__title-wrap {
  min-width: 0;
  display: grid;
  grid-template-columns: auto minmax(150px, 340px) minmax(0, 1fr);
  align-items: center;
  gap: 8px;
}

.compute-editor__meta,
.compute-editor__subline {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.compute-editor__meta > span {
  color: var(--dc-primary);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__name-input {
  width: min(340px, 46vw);
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text);
  font-size: 14px;
  font-weight: 800;
  line-height: 1.3;
}

.compute-editor__name-input:focus {
  border-color: var(--dc-primary);
  background: var(--dc-surface-raised);
  outline: none;
}

.compute-editor__subline span {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__lang-pill {
  padding: 2px 6px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  font-weight: 800;
}

.compute-editor__output-inline {
  font-family: Consolas, "Courier New", monospace;
}

.compute-editor__actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.compute-editor__icon-action,
.compute-editor__save,
.compute-editor__danger,
.compute-editor__small,
.compute-editor__tool-btn,
.compute-editor__ghost-icon {
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--dc-radius-sm);
  font-weight: 700;
}

.compute-editor__icon-action {
  width: 30px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.compute-editor__icon-action.is-active {
  border-color: rgba(29, 78, 216, 0.32);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.compute-editor__icon-action.is-danger {
  color: var(--dc-danger);
}

.compute-editor__save {
  gap: 6px;
  min-width: 68px;
  padding: 0 10px;
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
}

.compute-editor__danger {
  min-width: 64px;
  border: 1px solid var(--dc-danger);
  background: transparent;
  color: var(--dc-danger);
}

.compute-editor__small {
  min-width: 64px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.compute-editor__icon-action:disabled,
.compute-editor__save:disabled,
.compute-editor__danger:disabled,
.compute-editor__tool-btn:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.compute-editor__action-icon {
  width: 16px;
  height: 16px;
}

.compute-editor__main {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 6px;
  padding: 6px;
  overflow: hidden;
}

.compute-editor__code,
.compute-editor__inspector,
.compute-editor__panel {
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.compute-editor__code {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.compute-editor__code-head {
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 0 8px;
  border-bottom: 1px solid var(--dc-border);
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 800;
}

.compute-editor__code-title,
.compute-editor__inspector-title {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.compute-editor__code-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.compute-editor__code-head em {
  color: var(--dc-text-muted);
  font-size: 12px;
  font-style: normal;
  font-weight: 600;
}

.compute-editor__monaco {
  min-height: 0;
  flex: 1;
}

.compute-editor__inspector {
  width: 238px;
  min-width: 0;
  padding: 8px;
}

.compute-editor__inspector h3,
.compute-editor__panel h3 {
  margin: 0;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 800;
}

.compute-editor__inspector-head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 6px;
  margin-bottom: 8px;
}

.compute-editor__inspector-title {
  grid-column: 1;
}

.compute-editor__ghost-icon,
.compute-editor__tool-btn {
  width: 30px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
}

.compute-editor__output-path {
  grid-column: 1 / -1;
  min-width: 0;
  padding: 5px 7px;
  overflow: hidden;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-family: Consolas, "Courier New", monospace;
  font-size: 11px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__field {
  display: grid;
  gap: 3px;
  margin-bottom: 7px;
}

.compute-editor__field span {
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.compute-editor__field input,
.compute-editor__input-search input,
.compute-editor__mapping-row input {
  height: 28px;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  padding: 0 10px;
}

.compute-editor__inspector p,
.compute-editor__panel p {
  margin: 0;
  color: var(--dc-text-muted);
  font-size: 13px;
  line-height: 1.6;
}

.compute-editor__panel-tabs {
  min-height: 34px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px 0;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.compute-editor__panel-tabs.is-collapsed {
  padding-bottom: 4px;
}

.compute-editor__panel-tablist {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.compute-editor__panel-tab {
  height: 28px;
  min-width: 58px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  padding: 0 8px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.compute-editor__panel-tab.is-active {
  border-color: var(--dc-border);
  background: var(--dc-surface);
  color: var(--dc-primary);
}

.compute-editor__panel-tab-icon,
.compute-editor__head-icon {
  width: 15px;
  height: 15px;
}

.compute-editor__panel {
  min-height: 152px;
  max-height: 270px;
  margin: 0 6px 6px;
  padding: 8px;
  overflow: auto;
  box-shadow: none;
}

.compute-editor__panel-toggle {
  width: 28px;
  height: 28px;
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}

.compute-editor__panel-summary {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-left: 4px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.compute-editor__panel-summary span {
  padding: 2px 7px;
  border-radius: 999px;
  background: var(--dc-surface);
  white-space: nowrap;
}

.compute-editor__panel-toggle:hover {
  color: var(--dc-primary);
  background: var(--dc-surface);
}

.compute-editor__panel-toggle-icon {
  width: 16px;
  height: 16px;
  transition: transform 0.16s ease;
}

.compute-editor__panel-toggle-icon.is-collapsed {
  transform: rotate(180deg);
}

.compute-editor__panel-head,
.compute-editor__input-search,
.compute-editor__mapping-row,
.compute-editor__mapping-head,
.compute-editor__debug-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.compute-editor__panel-head {
  justify-content: space-between;
  margin-bottom: 8px;
}

.compute-editor__panel-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.compute-editor__input-search {
  margin-bottom: 8px;
}

.compute-editor__input-search input {
  flex: 1;
}

.compute-editor__input-search button,
.compute-editor__mapping-row button {
  height: 28px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
}

.compute-editor__input-search button {
  width: 32px;
}

.compute-editor__datapoints {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 8px;
  margin-bottom: 10px;
}

.compute-editor__datapoints button {
  min-width: 0;
  display: grid;
  gap: 4px;
  padding: 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  text-align: left;
}

.compute-editor__datapoints strong,
.compute-editor__datapoints span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__datapoints span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.compute-editor__empty {
  padding: 10px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 13px;
}

.compute-editor__empty-action {
  display: grid;
  gap: 6px;
  padding: 12px;
}

.compute-editor__empty-action strong {
  color: var(--dc-text);
  font-size: 13px;
}

.compute-editor__empty-action span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.compute-editor__empty-action div {
  display: inline-flex;
  gap: 8px;
}

.compute-editor__empty-action button {
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.compute-editor__empty-action button:first-child {
  border-color: rgba(29, 78, 216, 0.3);
  color: var(--dc-primary);
}

.compute-editor__mapping-row {
  margin-top: 8px;
}

.compute-editor__mapping-head {
  display: grid;
  grid-template-columns: minmax(120px, 0.5fr) minmax(160px, 1fr) 56px;
  margin-top: 6px;
  padding: 0 2px;
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 700;
}

.compute-editor__mapping-row {
  display: grid;
  grid-template-columns: minmax(120px, 0.5fr) minmax(160px, 1fr) 56px;
}

.compute-editor__mapping-row input {
  width: 100%;
}

.compute-editor__segmented {
  display: inline-flex;
  gap: 4px;
  padding: 3px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}

.compute-editor__segmented button {
  height: 26px;
  padding: 0 10px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.compute-editor__segmented button.is-active {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
}

.compute-editor__form-grid {
  max-width: 360px;
  margin-top: 12px;
}

.compute-editor__dependency-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 8px;
}

.compute-editor__dependency {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.compute-editor__dependency span {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.compute-editor__dependency strong {
  color: var(--dc-text);
  font-size: 13px;
}

.compute-editor__dependency em,
.compute-editor__dependency small {
  color: var(--dc-text-muted);
  font-size: 12px;
  font-style: normal;
}

.compute-editor__debug-layout {
  min-width: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 10px;
}

.compute-editor__debug-input {
  min-width: 0;
  display: grid;
  gap: 6px;
}

.compute-editor__debug-input span,
.compute-editor__debug-card strong {
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__debug-input textarea {
  min-height: 138px;
  resize: vertical;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  padding: 10px;
  font-family: Consolas, "Courier New", monospace;
  font-size: 12px;
  line-height: 1.5;
}

.compute-editor__debug-results {
  min-width: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.compute-editor__debug-card {
  min-width: 0;
  display: grid;
  gap: 6px;
  padding: 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.compute-editor__debug-card.has-error {
  border-color: var(--dc-danger);
}

.compute-editor__debug-card pre {
  min-height: 54px;
  max-height: 110px;
  margin: 0;
  overflow: auto;
  color: var(--dc-text);
  font-family: Consolas, "Courier New", monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}

.compute-editor__picker {
  display: grid;
  gap: 10px;
}

.compute-editor__picker-toolbar {
  display: grid;
  grid-template-columns: minmax(200px, 1fr) 120px 112px 112px auto;
  gap: 8px;
  align-items: center;
}

.compute-editor__picker-select {
  height: 30px;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
  padding: 0 8px;
  font-size: 12px;
  font-weight: 700;
}

.compute-editor__picker-loading {
  padding: 10px;
}

.compute-editor__picker-table-wrap {
  min-height: 300px;
  max-height: 390px;
  overflow: auto;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.compute-editor__picker-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.compute-editor__picker-table th,
.compute-editor__picker-table td {
  min-width: 0;
  padding: 8px 10px;
  border-bottom: 1px solid var(--dc-border);
  text-align: left;
  vertical-align: middle;
}

.compute-editor__picker-table th {
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--dc-surface-raised);
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__picker-table th:nth-child(1) {
  width: 20%;
}

.compute-editor__picker-table th:nth-child(2) {
  width: 32%;
}

.compute-editor__picker-table th:nth-child(3) {
  width: 12%;
}

.compute-editor__picker-table th:nth-child(4) {
  width: 14%;
}

.compute-editor__picker-table th:nth-child(5) {
  width: 12%;
}

.compute-editor__picker-table th:nth-child(6) {
  width: 10%;
}

.compute-editor__picker-table tbody tr {
  cursor: pointer;
}

.compute-editor__picker-table tbody tr:hover,
.compute-editor__picker-table tbody tr.is-selected {
  background: var(--dc-primary-soft);
}

.compute-editor__picker-table strong,
.compute-editor__picker-table code,
.compute-editor__picker-table td {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__picker-table strong {
  display: block;
  color: var(--dc-text);
  font-size: 13px;
}

.compute-editor__picker-table code {
  display: block;
  color: var(--dc-text-muted);
  font-family: Consolas, "Courier New", monospace;
  font-size: 12px;
}

.compute-editor__picker-status {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__picker-status.is-active {
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.compute-editor__picker-status.is-error {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.compute-editor__picker-status.is-inactive {
  color: var(--dc-text-muted);
}

.compute-editor__picker-row-action {
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

.compute-editor__picker-empty {
  margin: 10px;
  display: grid;
  gap: 5px;
}

.compute-editor__picker-empty strong {
  color: var(--dc-text);
}

.compute-editor__picker-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.compute-editor__picker-count {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.compute-editor__picker-count span {
  max-width: 240px;
  overflow: hidden;
  color: var(--dc-primary);
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-editor__picker-pager {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.compute-editor__picker-pager button {
  height: 30px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 800;
}

.compute-editor__picker-pager button:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.compute-editor__picker-pager span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.compute-editor__picker-pager .compute-editor__picker-confirm {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
}

@media (max-width: 1100px) {
  .compute-editor__main {
    grid-template-columns: 1fr;
    overflow-y: auto;
  }

  .compute-editor__debug-layout {
    grid-template-columns: 1fr;
  }

  .compute-editor__debug-results {
    grid-template-columns: 1fr;
  }

  .compute-editor__picker-toolbar {
    grid-template-columns: 1fr 1fr;
  }

  .compute-editor__picker-footer {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
