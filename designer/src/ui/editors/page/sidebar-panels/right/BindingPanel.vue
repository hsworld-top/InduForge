<!--
  BindingPanel - 绑定配置面板
  配置页面/组件的生命周期、定时器、变量变更等绑定脚本
-->
<script setup lang="ts">
import { ElMessage, ElMessageBox } from "element-plus";
import { storeToRefs } from "pinia";
import { computed, ref, watch } from "vue";
import IconEpDelete from "~icons/ep/delete";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";
import IconEpGrid from "~icons/ep/grid";
import IconEpList from "~icons/ep/list";
import IconEpPlus from "~icons/ep/plus";
import MonacoEditor from "@/ui/shared/widgets/base/monaco-editor-async";
import {
  resolvePageVariableSnippet,
  resolveProjectVariableSnippet,
  resolveProjectVariableSourceLabel,
} from "@/services/data-variable-mapping";
import { useEditorStore } from "@/stores/editor-store";
import { buildComponentMethodCompletions } from "@/ui/shared/helpers/component-methods";
import { usePanelState } from "../composables/use-panel-state";

interface LifecycleItem {
  key: string;
  label: string;
}

interface ScriptHandlerLike {
  type?: string;
  code?: string;
  enabled?: boolean;
  [key: string]: unknown;
}

interface TimerItemLike {
  id: string;
  name: string;
  code?: string;
  enabled?: boolean;
  interval?: number;
  description?: string;
  [key: string]: unknown;
}

interface VariableChangeItemLike {
  id: string;
  name: string;
  code?: string;
  enabled?: boolean;
  description?: string;
  [key: string]: unknown;
}

interface SidebarTreeNodeLike {
  id: string;
  label: string;
  type: "group" | "item" | "component";
  children?: SidebarTreeNodeLike[];
  params?: string;
  componentName?: string;
  componentType?: string;
}

interface VariableGroupLike {
  id: string;
  name: string;
  parentId?: string | null;
}

interface ProjectVariableLike {
  groupId?: string | null;
  type?: string;
  description?: string;
  mapped?: boolean;
  source?: {
    type?: string;
    [key: string]: unknown;
  };
}

interface VariableRowLike {
  name: string;
  type: string;
  description: string;
  groupId?: string | null;
  mapped?: boolean;
  sourceLabel?: string;
  defaultValue?: string;
}

interface CreateFormState {
  name: string;
  variable: string;
  interval: number;
  description: string;
}

interface SelectionState {
  timers: string[];
  variableChanges: string[];
}

interface MonacoEditorExposeLike {
  insertText?: (value: string) => void;
}

const { forceShow } = defineProps({
  forceShow: {
    type: Boolean,
    default: false,
  },
});

const UUID_DASH_PATTERN = /-/g;

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

const lifecycleItems: LifecycleItem[] = [
  { key: "onMounted", label: "创建时" },
  { key: "onUnmounted", label: "关闭时" },
];

const scriptCode = ref("");
const editorVisible = ref(false);
const editorRef = ref<MonacoEditorExposeLike | null>(null);
const customTreeRef = ref<any>(null);
const scriptSearch = ref("");
const componentSearch = ref("");
const componentTreeRef = ref<any>(null);
const variableEnumVisible = ref(false);
const projectVarSearch = ref("");
const pageVarSearch = ref("");
const enumProjectTreeRef = ref<any>(null);
const enumSelectedProjectGroupId = ref<string | null>(null);
const enumSelectedProjectVar = ref<VariableRowLike | null>(null);
const activeLifecycleKey = ref("");
const lifecycleToggleState = ref<Record<string, boolean>>({});
const activeItemType = ref<"" | "lifecycle" | "timer" | "variableChanges">("");
const activeItemId = ref("");
const selectedIds = ref<SelectionState>({ timers: [], variableChanges: [] });
const createDialogVisible = ref(false);
const createDialogType = ref<"timer" | "variable">("timer");
const createForm = ref<CreateFormState>({
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
const timerItems = computed<TimerItemLike[]>(() => {
  const list = (pageSnapshot.value?.lifecycle as Record<string, unknown> | undefined)?.timers;
  return Array.isArray(list) ? list : [];
});
const variableChangeItems = computed<VariableChangeItemLike[]>(() => {
  const list = (pageSnapshot.value?.lifecycle as Record<string, unknown> | undefined)
    ?.variableChanges;
  return Array.isArray(list) ? list : [];
});
const createDialogTitle = computed<string>(() => {
  return createDialogType.value === "timer" ? "新建定时器" : "新建变量改变";
});
const pageVariableOptions = computed<string[]>(() => {
  void docVersion.value;
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
function getLifecycleHandler(key: string): ScriptHandlerLike | string | null {
  if (!pageSnapshot.value || !key) return null;
  const handlers = (pageSnapshot.value.lifecycle as Record<string, unknown> | undefined)?.[key];
  if (!Array.isArray(handlers) || handlers.length === 0) return null;
  return (handlers[0] as ScriptHandlerLike | string | null) || null;
}

/**
 * 获取脚本内容
 * @param {string} key - 生命周期 key
 * @returns {string} 脚本内容
 */
function getLifecycleScript(key: string): string {
  const handler = getLifecycleHandler(key);
  if (!handler) return "";
  if (typeof handler === "string") return handler;
  return handler?.code || "";
}

/**
 * 同步生命周期开关状态
 */
function syncLifecycleToggleState() {
  const nextState: Record<string, boolean> = {};
  lifecycleItems.forEach((item) => {
    const handler = getLifecycleHandler(item.key) as any;
    nextState[item.key] = typeof handler === "string" ? true : handler?.enabled !== false;
  });
  lifecycleToggleState.value = nextState;
}

/**
 * 获取生命周期是否启用
 * @param {string} key - 生命周期 key
 * @returns {boolean} 是否启用
 */
function getLifecycleEnabled(key: string): boolean {
  if (!key) return true;
  return lifecycleToggleState.value[key] !== false;
}

/**
 * 切换生命周期启用状态
 * @param {string} key - 生命周期 key
 * @param {boolean} enabled - 是否启用
 */
function handleToggleLifecycle(key: string, enabled: boolean): void {
  if (!currentPage.value || !key) return;
  lifecycleToggleState.value = {
    ...lifecycleToggleState.value,
    [key]: enabled !== false,
  };
  const handler = getLifecycleHandler(key);
  if (!handler) return;
  const nextHandler: ScriptHandlerLike =
    typeof handler === "string"
      ? { type: "script", code: handler, enabled: enabled !== false }
      : { ...handler, enabled: enabled !== false };
  const nextLifecycle: Record<string, unknown> = { ...(pageSnapshot.value?.lifecycle || {}) };
  nextLifecycle[key] = [nextHandler] as ScriptHandlerLike[];
  editorStore.updateCurrentPage({ lifecycle: nextLifecycle });
}

/**
 * 打开脚本编辑器
 * @param {{ key: string }} item - 生命周期项
 */
function openEditor(item: LifecycleItem | null | undefined): void {
  activeLifecycleKey.value = item?.key || "";
  activeItemType.value = "lifecycle";
  activeItemId.value = "";
  scriptCode.value = getLifecycleScript(activeLifecycleKey.value) || "";
  editorVisible.value = true;
}

/**
 * 保存脚本
 */
function saveScript(): void {
  if (!currentPage.value) return;
  if (activeItemType.value === "timer" || activeItemType.value === "variableChanges") {
    const code = scriptCode.value || "";
    const items = activeItemType.value === "timer" ? timerItems.value : variableChangeItems.value;
    const nextItems = items.map((item) =>
      item.id === activeItemId.value ? { ...item, code } : item,
    );
    const nextLifecycle: Record<string, unknown> = { ...(pageSnapshot.value?.lifecycle || {}) };
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
  const nextLifecycle: Record<string, unknown> = { ...(pageSnapshot.value?.lifecycle || {}) };
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

const pageVars = computed<Record<string, any>>(() => {
  void docVersion.value;
  const pageId = currentPageId.value;
  if (!pageId || !doc.value) return {};
  const vars = doc.value.vars?.pages?.[pageId];
  return vars && typeof vars === "object" ? (vars as Record<string, any>) : {};
});

const projectGroupTree = computed<SidebarTreeNodeLike[]>(() => [
  {
    id: "all",
    label: "全部",
    type: "group",
    children: buildGroupTree(projectVariableGroups.value || []),
  },
]);

const projectVariableRows = computed<VariableRowLike[]>(() => {
  const keyword = String(projectVarSearch.value || "").toLowerCase();
  const items = Object.entries(
    (projectVariables.value || {}) as Record<string, ProjectVariableLike>,
  ).map(([name, detail]) => ({
    name,
    groupId: detail?.groupId || null,
    type: detail?.type || "string",
    description: detail?.description || "",
    mapped: detail?.source?.type === "dataCenter" || detail?.mapped === true,
    sourceLabel: resolveProjectVariableSourceLabel(detail)
      ? `来自数据点：${resolveProjectVariableSourceLabel(detail)}`
      : "",
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

const pageVariableRows = computed<VariableRowLike[]>(() => {
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

const pageComponentTree = computed<any[]>(() => {
  void docVersion.value;
  const rootId = currentPage.value?.rootNodeId;
  if (!rootId || !doc.value) return [];
  const currentDoc = doc.value;
  const buildNode = (nodeId: string): any => {
    const node = currentDoc.getNode(nodeId) as any;
    if (!node) return null;
    const children = (node.children || []).reduce((result: any[], childId: any) => {
      const child = buildNode(childId);
      if (child) result.push(child);
      return result;
    }, [] as any[]);
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

const customScriptTree = computed<any[]>(() => {
  const groups = (globalScripts.value?.custom?.groups || []) as any[];
  const items = (globalScripts.value?.custom?.items || []) as any[];
  const groupMap = new Map<string, any>();
  const roots: any[] = [];

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
      groupMap.get(group.parentId)?.children?.push(node);
    } else {
      roots.push(node);
    }
  });

  items.forEach((item) => {
    if (!item?.id) return;
    const node: any = {
      id: item.id,
      label: item.name || "未命名",
      type: "item",
      params: item.params || item.args || "",
    };
    if (item.groupId && groupMap.has(item.groupId)) {
      groupMap.get(item.groupId)?.children?.push(node);
    } else {
      roots.push(node);
    }
  });

  return roots;
});

const jsCompletions = computed<any[]>(() => {
  const items: any[] = [
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

  Object.keys((projectVariables.value || {}) as Record<string, unknown>).forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: "Variable",
      detail: "工程变量",
      prefix: "$global.",
    });
  });

  ((globalScripts.value?.custom?.items || []) as any[]).forEach((script: any) => {
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

  items.push(...(buildComponentMethodCompletions(pageComponentTree.value) as any[]));
  return items;
});

function buildGroupTree(groups: unknown): SidebarTreeNodeLike[] {
  const groupMap = new Map<string, SidebarTreeNodeLike>();
  const roots: SidebarTreeNodeLike[] = [];
  const normalized = (Array.isArray(groups) ? groups : []) as VariableGroupLike[];
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
      groupMap.get(group.parentId)?.children?.push(node);
    } else {
      roots.push(node);
    }
  });
  return roots;
}

function formatDefaultValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (value instanceof Set) return JSON.stringify(Array.from(value));
  if (value instanceof Map) return JSON.stringify(Array.from(value.entries()));
  if (typeof value === "object") {
    try {
      return JSON.stringify(value);
    } catch {
      return String(value);
    }
  }
  return String(value);
}

function filterSidebarNode(value: string, data: SidebarTreeNodeLike): boolean {
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

function handleCustomScriptInsert(data: SidebarTreeNodeLike): void {
  if (data?.type === "group") return;
  const params = data?.params ? data.params : "";
  const call = params ? `customScripts.${data.label}(${params})` : `customScripts.${data.label}()`;
  editorRef.value?.insertText?.(call);
}

function handleComponentInsert(data: SidebarTreeNodeLike): void {
  if (!data?.componentName) return;
  editorRef.value?.insertText?.(`components.${data.componentName}`);
}

function handlePageVariableInsert(row: VariableRowLike): void {
  if (!row?.name) return;
  editorRef.value?.insertText?.(resolvePageVariableSnippet(row.name));
}

function openVariableEnum() {
  projectVarSearch.value = "";
  enumSelectedProjectGroupId.value = null;
  enumSelectedProjectVar.value = null;
  variableEnumVisible.value = true;
}

function handleProjectGroupSelect(data: { id?: string } | null | undefined): void {
  if (!data) {
    enumSelectedProjectGroupId.value = null;
    return;
  }
  enumSelectedProjectGroupId.value =
    data.id === "all" ? null : ((data.id as string | null) ?? null);
}

function handleProjectRowClick(row: VariableRowLike | null | undefined): void {
  enumSelectedProjectVar.value = row || null;
}

function handleProjectRowDblClick(row: VariableRowLike | null | undefined): void {
  enumSelectedProjectVar.value = row || null;
  confirmEnumInsert();
}

function enumProjectRowClass({ row }: { row: VariableRowLike }): string {
  if (enumSelectedProjectVar.value?.name === row.name) return "is-selected";
  return "";
}

function confirmEnumInsert(): void {
  if (enumSelectedProjectVar.value?.name) {
    editorRef.value?.insertText?.(resolveProjectVariableSnippet(enumSelectedProjectVar.value.name));
    variableEnumVisible.value = false;
  }
}

/**
 * 处理创建命令
 * @param {string} command - 创建类型
 */
function handleCreateCommand(command: string): void {
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
function handleCreateConfirm(): void {
  if (!currentPage.value) return;
  const nextLifecycle: Record<string, unknown> = { ...(pageSnapshot.value?.lifecycle || {}) };
  if (createDialogType.value === "timer") {
    const name = createForm.value.name.trim();
    if (!name) {
      ElMessage.warning({ message: "请输入定时器名称" } as any);
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
      timers: [nextItems.at(-1)?.id || ""],
    };
  } else {
    const variable = createForm.value.variable.trim();
    if (!variable) {
      ElMessage.warning({ message: "请选择变量" } as any);
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
      variableChanges: [nextItems.at(-1)?.id || ""],
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
function createId(prefix: string): string {
  const random =
    typeof crypto !== "undefined" && crypto.randomUUID
      ? crypto.randomUUID().replace(UUID_DASH_PATTERN, "").slice(0, 8)
      : Math.random().toString(16).slice(2, 10);
  return `${prefix}_${Date.now().toString(16)}_${random}`;
}

/**
 * 判断是否选中条目
 * @param {'timers' | 'variableChanges'} type - 类型
 * @param {string} id - 条目 ID
 * @returns {boolean} 是否选中
 */
function isSelectedItem(type: keyof SelectionState, id: string): boolean {
  return selectedIds.value[type].includes(id);
}

/**
 * 切换条目选中状态
 * @param {'timers' | 'variableChanges'} type - 类型
 * @param {string} id - 条目 ID
 * @param {boolean} checked - 是否选中
 */
function toggleSelection(type: keyof SelectionState, id: string, checked: boolean): void {
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
function handleToggleItem(type: keyof SelectionState, id: string, enabled: boolean): void {
  if (!currentPage.value) return;
  const items = type === "timers" ? timerItems.value : variableChangeItems.value;
  const nextItems = items.map((item) =>
    item.id === id ? { ...item, enabled: enabled !== false } : item,
  );
  const nextLifecycle: Record<string, unknown> = { ...(pageSnapshot.value?.lifecycle || {}) };
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
function openItemEditor(
  type: keyof SelectionState,
  item: TimerItemLike | VariableChangeItemLike | null | undefined,
): void {
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
function handleDelete(): void {
  if (!currentPage.value) return;
  const timerIds = selectedIds.value.timers || [];
  const variableIds = selectedIds.value.variableChanges || [];
  const total = timerIds.length + variableIds.length;
  if (total === 0) {
    ElMessage.info({ message: "请选择需要删除的条目" } as any);
    return;
  }
  ElMessageBox.confirm(`确认删除已选的${total}条记录吗？`, "删除确认", {
    confirmButtonText: "删除",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      const nextLifecycle: Record<string, unknown> = { ...(pageSnapshot.value?.lifecycle || {}) };
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
                  @change="(value: any) => handleToggleLifecycle(item.key, value)"
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
              @change="(value: any) => toggleSelection('timers', item.id, value)"
            />
            <div class="binding-name">{{ item.name }}</div>
            <div class="binding-actions">
              <div class="binding-toggle">
                <span class="binding-toggle-label">启动</span>
                <el-switch
                  :model-value="item.enabled !== false"
                  @change="(value: any) => handleToggleItem('timers', item.id, value)"
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
              @change="(value: any) => toggleSelection('variableChanges', item.id, value)"
            />
            <div class="binding-name">{{ item.name }}</div>
            <div class="binding-actions">
              <div class="binding-toggle">
                <span class="binding-toggle-label">启动</span>
                <el-switch
                  :model-value="item.enabled !== false"
                  @change="(value: any) => handleToggleItem('variableChanges', item.id, value)"
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
          <div class="sidebar-title">页面变量</div>
          <el-input v-model="pageVarSearch" size="small" placeholder="搜索页面变量" clearable />
          <div class="sidebar-scroll page-var-list">
            <div
              v-for="row in pageVariableRows"
              :key="row.name"
              class="page-var-item"
              @click="handlePageVariableInsert(row)"
            >
              <el-icon class="node-icon icon-variable">
                <IconEpList />
              </el-icon>
              <span class="node-label">{{ row.name }}</span>
              <span class="page-var-type">{{ row.type }}</span>
            </div>
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
        <el-input v-model="projectVarSearch" size="small" placeholder="搜索工程变量" clearable />
        <el-table
          :data="projectVariableRows"
          size="small"
          height="320"
          highlight-current-row
          :row-class-name="enumProjectRowClass"
          @row-click="handleProjectRowClick"
          @row-dblclick="handleProjectRowDblClick"
        >
          <el-table-column prop="name" label="变量名" min-width="150" />
          <el-table-column prop="type" label="类型" width="90" />
          <el-table-column prop="description" label="描述" min-width="140" />
          <el-table-column prop="sourceLabel" label="来源" min-width="150" />
        </el-table>
      </div>
    </div>
    <template #footer>
      <el-button @click="variableEnumVisible = false">取消</el-button>
      <el-button type="primary" :disabled="!enumSelectedProjectVar" @click="confirmEnumInsert">
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
  background: var(--designer-shell-surface);
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

.page-var-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-var-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 28px;
  padding: 4px 6px;
  border-radius: 6px;
  cursor: pointer;
}

.page-var-item:hover {
  background: #f5f7fa;
}

.page-var-type {
  margin-left: auto;
  color: #909399;
  font-size: 12px;
}

.node-icon {
  color: var(--designer-text-muted);
  flex-shrink: 0;
}

.node-item .node-icon.icon-custom {
  color: var(--designer-success-text);
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
