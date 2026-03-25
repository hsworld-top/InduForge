<!--
  BindingPanel - 绑定配置面板
  配置页面/组件的生命周期、定时器、变量变更等绑定脚本
-->
<script setup>
import { ElMessage, ElMessageBox } from "element-plus";
import { storeToRefs } from "pinia";
import { computed, ref, watch } from "vue";
import IconEpDelete from "~icons/ep/delete";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";
import IconEpGrid from "~icons/ep/grid";
import IconEpList from "~icons/ep/list";
import IconEpPlus from "~icons/ep/plus";
import MonacoEditor from "@/components/common/monaco-editor-async";
import { useEditorStore } from "@/stores/editor-store";
import { buildComponentMethodCompletions } from "@/ui/shared/utils/component-methods";
import { usePanelState } from "../composables/use-panel-state";

const { forceShow } = defineProps({
  forceShow: {
    type: Boolean,
    default: false,
  },
});

const editorStore = useEditorStore();
const { panelState } = usePanelState();
const {
  currentPage,
  currentPageId,
  pages,
  projectVariables,
  projectVariableGroups,
  globalScripts,
  doc,
  docVersion,
} = storeToRefs(editorStore);

const lifecycleItems = [
  { key: "onMounted", label: "创建时" },
  { key: "onUnmounted", label: "关闭时" },
];

const scriptCode = ref("");
const editorVisible = ref(false);
const editorRef = ref(null);
const customTreeRef = ref(null);
const scriptSearch = ref("");
const componentSearch = ref("");
const componentTreeRef = ref(null);
const variableEnumVisible = ref(false);
const enumTab = ref("project");
const projectVarSearch = ref("");
const pageVarSearch = ref("");
const enumProjectTreeRef = ref(null);
const enumPageTreeRef = ref(null);
const enumSelectedProjectGroupId = ref(null);
const enumSelectedPageGroupId = ref("page-root");
const enumSelectedProjectVar = ref(null);
const enumSelectedPageVar = ref(null);
const activeLifecycleKey = ref("");
const lifecycleToggleState = ref({});
const activeItemType = ref("");
const activeItemId = ref("");
const selectedIds = ref({ timers: [], variableChanges: [] });
const createDialogVisible = ref(false);
const createDialogType = ref("timer");
const createForm = ref({
  name: "",
  variable: "",
  interval: 1000,
  description: "",
});

const pageSnapshot = computed(() => {
  if (currentPageId.value) {
    const page = pages.value.find((item) => item.id === currentPageId.value);
    if (page) return page;
  }
  return currentPage.value;
});
const pageTitle = computed(() => pageSnapshot.value?.name || "页面");
const timerItems = computed(() => {
  const list = pageSnapshot.value?.lifecycle?.timers;
  return Array.isArray(list) ? list : [];
});
const variableChangeItems = computed(() => {
  const list = pageSnapshot.value?.lifecycle?.variableChanges;
  return Array.isArray(list) ? list : [];
});
const createDialogTitle = computed(() => {
  return createDialogType.value === "timer" ? "新建定时器" : "新建变量改变";
});
const pageVariableOptions = computed(() => {
  docVersion.value;
  const pageId = currentPageId.value;
  if (!pageId || !doc.value) return [];
  const vars = doc.value.vars?.pages?.[pageId];
  if (!vars || typeof vars !== "object") return [];
  return Object.keys(vars);
});

/**
 * 获取生命周期脚本处理器
 * @param {string} key - 生命周期 key
 * @returns {Object | string | null} 处理器
 */
function getLifecycleHandler(key) {
  if (!pageSnapshot.value || !key) return null;
  const handlers = pageSnapshot.value.lifecycle?.[key];
  if (!Array.isArray(handlers) || handlers.length === 0) return null;
  return handlers[0] || null;
}

/**
 * 获取脚本内容
 * @param {string} key - 生命周期 key
 * @returns {string} 脚本内容
 */
function getLifecycleScript(key) {
  const handler = getLifecycleHandler(key);
  if (!handler) return "";
  if (typeof handler === "string") return handler;
  return handler?.code || "";
}

/**
 * 同步生命周期开关状态
 */
function syncLifecycleToggleState() {
  const nextState = {};
  lifecycleItems.forEach((item) => {
    const handler = getLifecycleHandler(item.key);
    nextState[item.key] = handler?.enabled !== false;
  });
  lifecycleToggleState.value = nextState;
}

/**
 * 获取生命周期是否启用
 * @param {string} key - 生命周期 key
 * @returns {boolean} 是否启用
 */
function getLifecycleEnabled(key) {
  if (!key) return true;
  return lifecycleToggleState.value[key] !== false;
}

/**
 * 切换生命周期启用状态
 * @param {string} key - 生命周期 key
 * @param {boolean} enabled - 是否启用
 */
function handleToggleLifecycle(key, enabled) {
  if (!currentPage.value || !key) return;
  lifecycleToggleState.value = {
    ...lifecycleToggleState.value,
    [key]: enabled !== false,
  };
  const handler = getLifecycleHandler(key);
  if (!handler) return;
  const nextHandler =
    typeof handler === "string"
      ? { type: "script", code: handler, enabled: enabled !== false }
      : { ...handler, enabled: enabled !== false };
  const nextLifecycle = { ...(pageSnapshot.value?.lifecycle || {}) };
  nextLifecycle[key] = [nextHandler];
  editorStore.updateCurrentPage({ lifecycle: nextLifecycle });
}

/**
 * 打开脚本编辑器
 * @param {{ key: string }} item - 生命周期项
 */
function openEditor(item) {
  activeLifecycleKey.value = item?.key || "";
  activeItemType.value = "lifecycle";
  activeItemId.value = "";
  scriptCode.value = getLifecycleScript(activeLifecycleKey.value) || "";
  editorVisible.value = true;
}

/**
 * 保存脚本
 */
function saveScript() {
  if (!currentPage.value) return;
  if (activeItemType.value === "timer" || activeItemType.value === "variableChanges") {
    const code = scriptCode.value || "";
    const items = activeItemType.value === "timer" ? timerItems.value : variableChangeItems.value;
    const nextItems = items.map((item) =>
      item.id === activeItemId.value ? { ...item, code } : item,
    );
    const nextLifecycle = { ...(pageSnapshot.value?.lifecycle || {}) };
    if (activeItemType.value === "timer") {
      nextLifecycle.timers = nextItems;
    } else {
      nextLifecycle.variableChanges = nextItems;
    }
    editorStore.updateCurrentPage({ lifecycle: nextLifecycle });
    void editorStore.saveCurrentPage?.();
    editorVisible.value = false;
    return;
  }
  if (!activeLifecycleKey.value) return;
  const code = scriptCode.value || "";
  const nextLifecycle = { ...(pageSnapshot.value?.lifecycle || {}) };
  if (!code.trim()) {
    delete nextLifecycle[activeLifecycleKey.value];
  } else {
    nextLifecycle[activeLifecycleKey.value] = [
      {
        type: "script",
        code,
        enabled: getLifecycleEnabled(activeLifecycleKey.value),
      },
    ];
  }
  editorStore.updateCurrentPage({ lifecycle: nextLifecycle });
  void editorStore.saveCurrentPage?.();
  editorVisible.value = false;
}

const editorTitle = computed(() => pageTitle.value);

const editorDescription = computed(() => {
  if (activeItemType.value === "timer") {
    const item = timerItems.value.find((entry) => entry.id === activeItemId.value);
    const label = item?.name || "定时器";
    return `${pageTitle.value}${label}脚本`;
  }
  if (activeItemType.value === "variableChanges") {
    const item = variableChangeItems.value.find((entry) => entry.id === activeItemId.value);
    const label = item?.name || "变量改变";
    return `${pageTitle.value}${label}脚本`;
  }
  const item = lifecycleItems.find((entry) => entry.key === activeLifecycleKey.value);
  const label = item?.label || activeLifecycleKey.value || "事件";
  return `${pageTitle.value}${label}脚本`;
});

const pageVars = computed(() => {
  docVersion.value;
  const pageId = currentPageId.value;
  if (!pageId || !doc.value) return {};
  const vars = doc.value.vars?.pages?.[pageId];
  return vars && typeof vars === "object" ? vars : {};
});

const projectGroupTree = computed(() => [
  {
    id: "all",
    label: "全部",
    type: "group",
    children: buildGroupTree(projectVariableGroups.value || []),
  },
]);

const projectVariableRows = computed(() => {
  const keyword = String(projectVarSearch.value || "").toLowerCase();
  const items = Object.entries(projectVariables.value || {}).map(([name, detail]) => ({
    name,
    groupId: detail?.groupId || null,
    type: detail?.type || "string",
    description: detail?.description || "",
    mapped: detail?.source?.type === "dataCenter" || detail?.mapped === true,
  }));
  return items
    .filter((item) => {
      if (enumSelectedProjectGroupId.value) {
        return item.groupId === enumSelectedProjectGroupId.value;
      }
      return true;
    })
    .filter((item) => {
      if (!keyword) return true;
      return String(item.name || "")
        .toLowerCase()
        .includes(keyword);
    });
});

const pageGroupTree = computed(() => [
  { id: "page-root", label: "页面变量", type: "group", children: [] },
]);

const pageVariableRows = computed(() => {
  const keyword = String(pageVarSearch.value || "").toLowerCase();
  return Object.entries(pageVars.value || {})
    .map(([name, detail]) => ({
      name,
      type: detail?.type || "string",
      defaultValue: formatDefaultValue(detail?.default),
      description: detail?.description || "",
    }))
    .filter((item) => {
      if (!keyword) return true;
      return String(item.name || "")
        .toLowerCase()
        .includes(keyword);
    });
});

const pageComponentTree = computed(() => {
  docVersion.value;
  const rootId = currentPage.value?.rootNodeId;
  if (!rootId || !doc.value) return [];
  const buildNode = (nodeId) => {
    const node = doc.value.getNode(nodeId);
    if (!node) return null;
    const children = (node.children || []).map((childId) => buildNode(childId)).filter(Boolean);
    const label = node.label || node.type || "组件";
    return {
      id: node.id,
      label,
      type: children.length ? "group" : "component",
      componentName: node.label || "",
      componentType: node.type || "",
      children,
    };
  };
  const root = buildNode(rootId);
  if (!root) return [];
  return root.children?.length ? root.children : [root];
});

const customScriptTree = computed(() => {
  const groups = globalScripts.value?.custom?.groups || [];
  const items = globalScripts.value?.custom?.items || [];
  const groupMap = new Map();
  const roots = [];

  groups.forEach((group) => {
    groupMap.set(group.id, {
      id: group.id,
      label: group.name,
      type: "group",
      children: [],
    });
  });

  groupMap.forEach((node, id) => {
    const group = groups.find((item) => item.id === id);
    if (group?.parentId && groupMap.has(group.parentId)) {
      groupMap.get(group.parentId).children.push(node);
    } else {
      roots.push(node);
    }
  });

  items.forEach((item) => {
    if (!item?.id) return;
    const node = {
      id: item.id,
      label: item.name || "未命名",
      type: "item",
      params: item.params || item.args || "",
    };
    if (item.groupId && groupMap.has(item.groupId)) {
      groupMap.get(item.groupId).children.push(node);
    } else {
      roots.push(node);
    }
  });

  return roots;
});

const jsCompletions = computed(() => {
  const items = [
    {
      label: "console.log",
      insertText: "console.log()",
      kind: "Function",
      detail: "Log output",
    },
    {
      label: "if",
      insertText: "if () {\\n  \\n}",
      kind: "Snippet",
      detail: "if statement",
    },
    {
      label: "for",
      insertText: "for (let i = 0; i < ; i++) {\\n  \\n}",
      kind: "Snippet",
    },
    {
      label: "function",
      insertText: "function name() {\\n  \\n}",
      kind: "Snippet",
    },
    { label: "const", insertText: "const ", kind: "Keyword" },
    { label: "let", insertText: "let ", kind: "Keyword" },
    { label: "return", insertText: "return ", kind: "Keyword" },
  ];

  Object.keys(projectVariables.value || {}).forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: "Variable",
      detail: "工程变量",
      prefix: "$global.",
    });
  });

  (globalScripts.value?.custom?.items || []).forEach((script) => {
    if (!script?.name) return;
    const params =
      typeof script.params === "string" && script.params.trim()
        ? script.params.trim()
        : typeof script.args === "string"
          ? script.args.trim()
          : "";
    const call = params ? `${script.name}(${params})` : `${script.name}()`;
    items.push({
      label: script.name,
      insertText: call,
      kind: "Function",
      detail: "自定义脚本",
      prefix: "customScripts.",
    });
  });

  Object.keys(pageVars.value || {}).forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: "Variable",
      detail: "页面变量",
      prefix: "$vars.",
    });
  });

  items.push(...buildComponentMethodCompletions(pageComponentTree.value));
  return items;
});

function buildGroupTree(groups) {
  const groupMap = new Map();
  const roots = [];
  const normalized = Array.isArray(groups) ? groups : [];
  normalized.forEach((group) => {
    groupMap.set(group.id, {
      id: group.id,
      label: group.name,
      type: "group",
      children: [],
    });
  });
  groupMap.forEach((node, id) => {
    const group = normalized.find((item) => item.id === id);
    if (group?.parentId && groupMap.has(group.parentId)) {
      groupMap.get(group.parentId).children.push(node);
    } else {
      roots.push(node);
    }
  });
  return roots;
}

function formatDefaultValue(value) {
  if (value === null || value === undefined) return "";
  if (value instanceof Set) return JSON.stringify(Array.from(value));
  if (value instanceof Map) return JSON.stringify(Array.from(value.entries()));
  if (typeof value === "object") {
    try {
      return JSON.stringify(value);
    } catch (error) {
      return String(value);
    }
  }
  return String(value);
}

function filterSidebarNode(value, data) {
  if (!value) return true;
  return String(data?.label || "")
    .toLowerCase()
    .includes(value.toLowerCase());
}

watch(scriptSearch, (value) => {
  customTreeRef.value?.filter?.(value);
});

watch(componentSearch, (value) => {
  componentTreeRef.value?.filter?.(value);
});

function handleCustomScriptInsert(data) {
  if (data?.type === "group") return;
  const params = data?.params ? data.params : "";
  const call = params ? `customScripts.${data.label}(${params})` : `customScripts.${data.label}()`;
  editorRef.value?.insertText?.(call);
}

function handleComponentInsert(data) {
  if (!data?.componentName) return;
  editorRef.value?.insertText?.(`components.${data.componentName}`);
}

function openVariableEnum() {
  projectVarSearch.value = "";
  pageVarSearch.value = "";
  enumTab.value = "project";
  enumSelectedProjectGroupId.value = null;
  enumSelectedPageGroupId.value = "page-root";
  enumSelectedProjectVar.value = null;
  enumSelectedPageVar.value = null;
  variableEnumVisible.value = true;
}

function handleProjectGroupSelect(data) {
  if (!data) {
    enumSelectedProjectGroupId.value = null;
    return;
  }
  enumSelectedProjectGroupId.value = data.id === "all" ? null : data.id;
}

function handlePageGroupSelect(data) {
  enumSelectedPageGroupId.value = data?.id || "page-root";
}

function handleProjectRowClick(row) {
  enumSelectedProjectVar.value = row || null;
}

function handlePageRowClick(row) {
  enumSelectedPageVar.value = row || null;
}

function handleProjectRowDblClick(row) {
  enumSelectedProjectVar.value = row || null;
  confirmEnumInsert();
}

function handlePageRowDblClick(row) {
  enumSelectedPageVar.value = row || null;
  confirmEnumInsert();
}

function enumProjectRowClass({ row }) {
  if (enumSelectedProjectVar.value?.name === row.name) return "is-selected";
  return "";
}

function enumPageRowClass({ row }) {
  if (enumSelectedPageVar.value?.name === row.name) return "is-selected";
  return "";
}

function confirmEnumInsert() {
  if (enumTab.value === "page" && enumSelectedPageVar.value?.name) {
    editorRef.value?.insertText?.(`$vars.${enumSelectedPageVar.value.name}`);
    variableEnumVisible.value = false;
    return;
  }
  if (enumSelectedProjectVar.value?.name) {
    editorRef.value?.insertText?.(`$global.${enumSelectedProjectVar.value.name}`);
    variableEnumVisible.value = false;
  }
}

/**
 * 处理创建命令
 * @param {string} command - 创建类型
 */
function handleCreateCommand(command) {
  createDialogType.value = command === "variable" ? "variable" : "timer";
  createForm.value = {
    name: "",
    variable: "",
    interval: 1000,
    description: "",
  };
  createDialogVisible.value = true;
}

/**
 * 处理创建确认
 */
function handleCreateConfirm() {
  if (!currentPage.value) return;
  const nextLifecycle = { ...(pageSnapshot.value?.lifecycle || {}) };
  if (createDialogType.value === "timer") {
    const name = createForm.value.name.trim();
    if (!name) {
      ElMessage.warning("请输入定时器名称");
      return;
    }
    const interval = Number(createForm.value.interval) || 1000;
    const nextItems = [
      ...timerItems.value,
      {
        id: createId("timer"),
        name,
        code: "",
        enabled: true,
        interval,
        description: createForm.value.description || "",
      },
    ];
    nextLifecycle.timers = nextItems;
    selectedIds.value = {
      ...selectedIds.value,
      timers: [nextItems.at(-1).id],
    };
  } else {
    const variable = createForm.value.variable.trim();
    if (!variable) {
      ElMessage.warning("请选择变量");
      return;
    }
    const nextItems = [
      ...variableChangeItems.value,
      {
        id: createId("var"),
        name: variable,
        code: "",
        enabled: true,
        description: createForm.value.description || "",
      },
    ];
    nextLifecycle.variableChanges = nextItems;
    selectedIds.value = {
      ...selectedIds.value,
      variableChanges: [nextItems.at(-1).id],
    };
  }
  editorStore.updateCurrentPage({ lifecycle: nextLifecycle });
  void editorStore.saveCurrentPage?.();
  createDialogVisible.value = false;
}

/**
 * 生成 ID
 * @param {string} prefix - 前缀
 * @returns {string} ID
 */
function createId(prefix) {
  const random =
    typeof crypto !== "undefined" && crypto.randomUUID
      ? crypto.randomUUID().replace(/-/g, "").slice(0, 8)
      : Math.random().toString(16).slice(2, 10);
  return `${prefix}_${Date.now().toString(16)}_${random}`;
}

/**
 * 判断是否选中条目
 * @param {'timers' | 'variableChanges'} type - 类型
 * @param {string} id - 条目 ID
 * @returns {boolean} 是否选中
 */
function isSelectedItem(type, id) {
  return selectedIds.value[type].includes(id);
}

/**
 * 切换条目选中状态
 * @param {'timers' | 'variableChanges'} type - 类型
 * @param {string} id - 条目 ID
 * @param {boolean} checked - 是否选中
 */
function toggleSelection(type, id, checked) {
  const list = selectedIds.value[type] || [];
  const nextList = checked ? [...new Set([...list, id])] : list.filter((item) => item !== id);
  selectedIds.value = { ...selectedIds.value, [type]: nextList };
}

/**
 * 切换条目启用状态
 * @param {'timers' | 'variableChanges'} type - 类型
 * @param {string} id - 条目 ID
 * @param {boolean} enabled - 是否启用
 */
function handleToggleItem(type, id, enabled) {
  if (!currentPage.value) return;
  const items = type === "timers" ? timerItems.value : variableChangeItems.value;
  const nextItems = items.map((item) =>
    item.id === id ? { ...item, enabled: enabled !== false } : item,
  );
  const nextLifecycle = { ...(pageSnapshot.value?.lifecycle || {}) };
  if (type === "timers") {
    nextLifecycle.timers = nextItems;
  } else {
    nextLifecycle.variableChanges = nextItems;
  }
  editorStore.updateCurrentPage({ lifecycle: nextLifecycle });
}

/**
 * 打开条目脚本编辑器
 * @param {'timers' | 'variableChanges'} type - 类型
 * @param {{ id: string, code?: string, name?: string }} item - 条目
 */
function openItemEditor(type, item) {
  if (!item?.id) return;
  activeLifecycleKey.value = "";
  activeItemType.value = type === "timers" ? "timer" : "variableChanges";
  activeItemId.value = item.id;
  scriptCode.value = item.code || "";
  editorVisible.value = true;
}

/**
 * 删除条目
 */
function handleDelete() {
  if (!currentPage.value) return;
  const timerIds = selectedIds.value.timers || [];
  const variableIds = selectedIds.value.variableChanges || [];
  const total = timerIds.length + variableIds.length;
  if (total === 0) {
    ElMessage.info("请选择需要删除的条目");
    return;
  }
  ElMessageBox.confirm(`确认删除已选的${total}条记录吗？`, "删除确认", {
    confirmButtonText: "删除",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      const nextLifecycle = { ...(pageSnapshot.value?.lifecycle || {}) };
      if (timerIds.length > 0) {
        nextLifecycle.timers = timerItems.value.filter((item) => !timerIds.includes(item.id));
      }
      if (variableIds.length > 0) {
        nextLifecycle.variableChanges = variableChangeItems.value.filter(
          (item) => !variableIds.includes(item.id),
        );
      }
      editorStore.updateCurrentPage({ lifecycle: nextLifecycle });
      void editorStore.saveCurrentPage?.();
      selectedIds.value = { timers: [], variableChanges: [] };
    })
    .catch(() => {});
}

watch(
  () => [currentPage.value?.id],
  () => {
    syncLifecycleToggleState();
    selectedIds.value = { timers: [], variableChanges: [] };
  },
  { immediate: true },
);

watch(
  () => [currentPage.value?.id, activeLifecycleKey.value],
  () => {
    if (activeItemType.value === "lifecycle") {
      if (!activeLifecycleKey.value) return;
      scriptCode.value = getLifecycleScript(activeLifecycleKey.value) || "";
    }
  },
  { immediate: true },
);

watch(
  () => [currentPage.value?.id, activeItemId.value, activeItemType.value],
  () => {
    if (!activeItemId.value) return;
    if (activeItemType.value === "timer") {
      const item = timerItems.value.find((entry) => entry.id === activeItemId.value);
      scriptCode.value = item?.code || "";
    } else if (activeItemType.value === "variableChanges") {
      const item = variableChangeItems.value.find((entry) => entry.id === activeItemId.value);
      scriptCode.value = item?.code || "";
    }
  },
  { immediate: true },
);
</script>

<template>
  <div class="binding-panel">
    <template v-if="!forceShow && panelState !== 'page'">
      <div class="empty-hint">请选择页面以配置绑定</div>
    </template>
    <template v-else>
      <div class="binding-toolbar">
        <el-dropdown trigger="click" @command="handleCreateCommand">
          <el-button class="toolbar-button" size="small" circle>
            <IconEpPlus />
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="timer">创建定时器</el-dropdown-item>
              <el-dropdown-item command="variable">变量改变</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-tooltip content="删除" placement="top">
          <el-button class="toolbar-button" size="small" circle @click="handleDelete">
            <IconEpDelete />
          </el-button>
        </el-tooltip>
      </div>
      <div class="binding-section">
        <div class="section-header">基本</div>
        <div class="section-body">
          <div v-for="item in lifecycleItems" :key="item.key" class="binding-row">
            <div class="binding-name">{{ item.label }}</div>
            <div class="binding-actions">
              <div class="binding-toggle">
                <span class="binding-toggle-label">启动</span>
                <el-switch
                  :model-value="getLifecycleEnabled(item.key)"
                  @change="(value) => handleToggleLifecycle(item.key, value)"
                />
              </div>
              <el-tooltip content="打开编辑器" placement="top">
                <el-button class="icon-button" size="small" circle @click="openEditor(item)">
                  <IconEpEditPen />
                </el-button>
              </el-tooltip>
            </div>
          </div>
        </div>
      </div>

      <div class="binding-section">
        <div class="section-header">定时器</div>
        <div v-if="timerItems.length === 0" class="empty-block">暂无数据</div>
        <div v-else class="section-body">
          <div
            v-for="item in timerItems"
            :key="item.id"
            class="binding-row is-shifted"
            :class="{ 'is-active': isSelectedItem('timers', item.id) }"
          >
            <el-checkbox
              class="select-check"
              :model-value="isSelectedItem('timers', item.id)"
              @change="(value) => toggleSelection('timers', item.id, value)"
            />
            <div class="binding-name">{{ item.name }}</div>
            <div class="binding-actions">
              <div class="binding-toggle">
                <span class="binding-toggle-label">启动</span>
                <el-switch
                  :model-value="item.enabled !== false"
                  @change="(value) => handleToggleItem('timers', item.id, value)"
                />
              </div>
              <el-tooltip content="打开编辑器" placement="top">
                <el-button
                  class="icon-button"
                  size="small"
                  circle
                  @click.stop="openItemEditor('timers', item)"
                >
                  <IconEpEditPen />
                </el-button>
              </el-tooltip>
            </div>
          </div>
        </div>
      </div>

      <div class="binding-section">
        <div class="section-header">变量改变</div>
        <div v-if="variableChangeItems.length === 0" class="empty-block">暂无数据</div>
        <div v-else class="section-body">
          <div
            v-for="item in variableChangeItems"
            :key="item.id"
            class="binding-row is-shifted"
            :class="{ 'is-active': isSelectedItem('variableChanges', item.id) }"
          >
            <el-checkbox
              class="select-check"
              :model-value="isSelectedItem('variableChanges', item.id)"
              @change="(value) => toggleSelection('variableChanges', item.id, value)"
            />
            <div class="binding-name">{{ item.name }}</div>
            <div class="binding-actions">
              <div class="binding-toggle">
                <span class="binding-toggle-label">启动</span>
                <el-switch
                  :model-value="item.enabled !== false"
                  @change="(value) => handleToggleItem('variableChanges', item.id, value)"
                />
              </div>
              <el-tooltip content="打开编辑器" placement="top">
                <el-button
                  class="icon-button"
                  size="small"
                  circle
                  @click.stop="openItemEditor('variableChanges', item)"
                >
                  <IconEpEditPen />
                </el-button>
              </el-tooltip>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>

  <el-dialog
    v-model="editorVisible"
    :title="editorTitle"
    width="980px"
    top="3vh"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <div class="editor-meta">
      <div class="meta-title">{{ editorTitle }}</div>
      <div class="meta-desc">{{ editorDescription }}</div>
      <div class="meta-actions">
        <el-tooltip content="枚举变量" placement="top">
          <el-button class="icon-button" size="small" circle @click="openVariableEnum">
            <IconEpList />
          </el-button>
        </el-tooltip>
      </div>
    </div>
    <div class="editor-body">
      <div class="editor-main">
        <MonacoEditor
          ref="editorRef"
          v-model="scriptCode"
          language="javascript"
          height="520px"
          :completions="jsCompletions"
        />
      </div>
      <div class="editor-sidebar">
        <div class="sidebar-section">
          <div class="sidebar-title">自定义脚本</div>
          <el-input v-model="scriptSearch" size="small" placeholder="搜索脚本/分组" clearable />
          <div class="sidebar-scroll">
            <el-tree
              ref="customTreeRef"
              :data="customScriptTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterSidebarNode"
              @node-click="handleCustomScriptInsert"
            >
              <template #default="{ data }">
                <div class="tree-node" :class="`node-${data.type}`">
                  <el-icon class="node-icon icon-custom">
                    <IconEpFolder v-if="data.type === 'group'" />
                    <IconEpEditPen v-else />
                  </el-icon>
                  <span class="node-label" :class="{ 'is-group': data.type === 'group' }">
                    {{ data.label }}
                  </span>
                </div>
              </template>
            </el-tree>
          </div>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-title">页面组件</div>
          <el-input v-model="componentSearch" size="small" placeholder="搜索组件/分组" clearable />
          <div class="sidebar-scroll">
            <el-tree
              ref="componentTreeRef"
              :data="pageComponentTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterSidebarNode"
              @node-click="handleComponentInsert"
            >
              <template #default="{ data }">
                <div class="tree-node" :class="`node-${data.type}`">
                  <el-icon class="node-icon icon-component">
                    <IconEpFolder v-if="data.type === 'group'" />
                    <IconEpGrid v-else />
                  </el-icon>
                  <span class="node-label" :class="{ 'is-group': data.type === 'group' }">
                    {{ data.label }}
                  </span>
                </div>
              </template>
            </el-tree>
          </div>
        </div>
      </div>
    </div>
    <template #footer>
      <el-button @click="editorVisible = false">取消</el-button>
      <el-button type="primary" @click="saveScript">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog
    v-model="variableEnumVisible"
    title="变量枚举"
    width="760px"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <el-tabs v-model="enumTab">
      <el-tab-pane label="工程变量" name="project">
        <div class="enum-layout">
          <div class="enum-left">
            <div class="sidebar-title">分组</div>
            <el-tree
              ref="enumProjectTreeRef"
              :data="projectGroupTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterSidebarNode"
              @node-click="handleProjectGroupSelect"
            >
              <template #default="{ data }">
                <div class="tree-node node-group">
                  <el-icon class="node-icon icon-variable">
                    <IconEpFolder />
                  </el-icon>
                  <span class="node-label is-group">{{ data.label }}</span>
                </div>
              </template>
            </el-tree>
          </div>
          <div class="enum-right">
            <el-input
              v-model="projectVarSearch"
              size="small"
              placeholder="搜索工程变量"
              clearable
            />
            <el-table
              :data="projectVariableRows"
              size="small"
              height="320"
              highlight-current-row
              :row-class-name="enumProjectRowClass"
              @row-click="handleProjectRowClick"
              @row-dblclick="handleProjectRowDblClick"
            >
              <el-table-column prop="name" label="变量名" min-width="160" />
              <el-table-column prop="type" label="类型" width="90" />
              <el-table-column prop="description" label="描述" min-width="160" />
              <el-table-column prop="mapped" label="映射" width="70">
                <template #default="{ row }">
                  {{ row.mapped ? "是" : "" }}
                </template>
              </el-table-column>
            </el-table>
          </div>
        </div>
      </el-tab-pane>
      <el-tab-pane label="页面变量" name="page">
        <div class="enum-layout">
          <div class="enum-left">
            <div class="sidebar-title">分组</div>
            <el-tree
              ref="enumPageTreeRef"
              :data="pageGroupTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              @node-click="handlePageGroupSelect"
            >
              <template #default="{ data }">
                <div class="tree-node node-group">
                  <el-icon class="node-icon icon-variable">
                    <IconEpFolder />
                  </el-icon>
                  <span class="node-label is-group">{{ data.label }}</span>
                </div>
              </template>
            </el-tree>
          </div>
          <div class="enum-right">
            <el-input v-model="pageVarSearch" size="small" placeholder="搜索页面变量" clearable />
            <el-table
              :data="pageVariableRows"
              size="small"
              height="320"
              highlight-current-row
              :row-class-name="enumPageRowClass"
              @row-click="handlePageRowClick"
              @row-dblclick="handlePageRowDblClick"
            >
              <el-table-column prop="name" label="变量名" min-width="160" />
              <el-table-column prop="type" label="类型" width="90" />
              <el-table-column prop="defaultValue" label="初始值" min-width="160" />
              <el-table-column prop="description" label="描述" min-width="160" />
            </el-table>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
    <template #footer>
      <el-button @click="variableEnumVisible = false">取消</el-button>
      <el-button
        type="primary"
        :disabled="!enumSelectedProjectVar && !enumSelectedPageVar"
        @click="confirmEnumInsert"
      >
        插入
      </el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="createDialogVisible" :title="createDialogTitle" width="420px">
    <el-form label-width="90px">
      <template v-if="createDialogType === 'timer'">
        <el-form-item label="定时器名称">
          <el-input v-model="createForm.name" placeholder="请输入定时器名称" />
        </el-form-item>
        <el-form-item label="时间(ms)">
          <el-input-number
            v-model="createForm.interval"
            :min="100"
            :step="100"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="createForm.description" placeholder="请输入描述" />
        </el-form-item>
      </template>
      <template v-else>
        <el-form-item label="变量">
          <el-select v-model="createForm.variable" placeholder="请选择变量">
            <el-option
              v-for="option in pageVariableOptions"
              :key="option"
              :label="option"
              :value="option"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="createForm.description" placeholder="请输入描述" />
        </el-form-item>
      </template>
    </el-form>
    <template #footer>
      <el-button @click="createDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="handleCreateConfirm">确定</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.binding-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.binding-toolbar {
  display: flex;
  justify-content: flex-start;
  gap: 6px;
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

.binding-section {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: #ffffff;
}

.section-header {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-regular);
  padding: 8px 10px;
  border-bottom: 1px solid #e4e7ed;
  background: #f7f8fa;
}

.section-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
}

.binding-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 6px;
  border-radius: 6px;
  cursor: pointer;
}

.binding-name {
  font-size: 12px;
  color: var(--el-text-color-regular);
  min-width: 120px;
}

.binding-row.is-active {
  background: #eef2ff;
}

.binding-row.is-shifted .binding-actions {
  transform: translateX(-18px);
}

.select-check {
  margin-right: 2px;
}

.binding-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.binding-check {
  font-size: 12px;
}

.binding-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 6px;
  border-radius: 6px;
  background: var(--el-fill-color-lighter);
}

.binding-toggle-label {
  font-size: 12px;
  color: var(--el-text-color-regular);
  white-space: nowrap;
}

.icon-button {
  background: #eef2ff;
  border: none;
  color: #4f46e5;
}

.icon-button:hover {
  background: #e0e7ff;
}

.empty-block {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  text-align: center;
  padding: 18px 0;
}

.empty-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  text-align: center;
  padding: 16px 0;
}

.editor-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 16px;
  padding: 10px 12px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  background: #fafafa;
  margin-bottom: 10px;
  align-items: center;
}

.meta-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.meta-desc {
  font-size: 12px;
  color: #606266;
}

.meta-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
}

.editor-body {
  display: flex;
  gap: 12px;
  flex: 1;
  align-items: stretch;
  height: 520px;
}

.editor-main {
  flex: 1;
  min-width: 0;
}

.editor-sidebar {
  width: 220px;
  height: 520px;
  border-left: 1px solid #e4e7ed;
  padding-left: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.sidebar-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
  min-height: 0;
}

.sidebar-title {
  font-size: 12px;
  font-weight: 600;
  color: #606266;
}

.sidebar-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding-right: 4px;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  width: 100%;
  padding: 6px 8px;
  border-radius: 6px;
  transition: background-color 0.2s;
}

.tree-node:hover {
  background: #f5f7fa;
}

.node-icon {
  color: #94a3b8;
  flex-shrink: 0;
}

.node-item .node-icon.icon-custom {
  color: #10b981;
}

.node-variable .node-icon.icon-variable {
  color: #0ea5e9;
}

.node-label {
  font-size: 13px;
  color: #303133;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.node-label.is-group {
  font-weight: 600;
}

.node-component .node-icon.icon-component {
  color: #6366f1;
}

.enum-layout {
  display: flex;
  gap: 12px;
  padding-top: 8px;
}

.enum-left {
  width: 200px;
  border-right: 1px solid #e4e7ed;
  padding-right: 8px;
  max-height: 360px;
  overflow: auto;
}

.enum-right {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.enum-right :deep(.el-table__row.is-selected) {
  background: #eef2ff;
}
</style>
