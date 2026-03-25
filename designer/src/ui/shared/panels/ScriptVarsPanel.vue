<!--
  ScriptVarsPanel - 脚本与变量面板
  管理系统脚本（启动/关闭）、定时器、变量变更、自定义脚本
-->
<script setup>
import { ElMessage, ElMessageBox } from "element-plus";
import { storeToRefs } from "pinia";
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import IconEpDocument from "~icons/ep/document";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";
import IconEpList from "~icons/ep/list";
import MonacoEditor from "@/components/common/monaco-editor-async";
import { useEditorStore } from "@/stores/editor-store";
import { buildComponentMethodCompletions } from "@/ui/shared/utils/component-methods";
import ScriptVarsCustomSection from "./ScriptVarsCustomSection.vue";
import ScriptVarsSystemSection from "./ScriptVarsSystemSection.vue";
import ScriptVarsTimersSection from "./ScriptVarsTimersSection.vue";
import ScriptVarsVariableChangesSection from "./ScriptVarsVariableChangesSection.vue";
import VariableGroupFormDialog from "./VariableGroupFormDialog.vue";

const editorStore = useEditorStore();
const {
  projectId,
  globalScripts,
  projectVariables,
  projectVariableGroups,
  pages,
  doc,
  docVersion,
  currentPage,
} = storeToRefs(editorStore);
const maxGroupDepth = 5;

const activeSections = ref("system");
const selectedSystemKey = ref("startup");

const selectedTimerId = ref("");
const selectedVariableId = ref("");
const selectedCustomId = ref("");
const selectedNodes = ref({
  timers: [],
  variableChanges: [],
  custom: [],
});

const systemCode = ref("");
const editorCode = ref("");
const systemOriginalCode = ref("");
const editorOriginalCode = ref("");
const editorOriginalInterval = ref(1000);
const editorInterval = ref(1000);
const editorParams = ref("");

const systemEditorRef = ref(null);
const activeEditorRef = ref(null);

const scriptClipboard = ref(null);

const groupDialogVisible = ref(false);
const groupEditMode = ref(false);
const groupDialogModule = ref("");
const groupId = ref("");
const groupName = ref("");
const groupParentId = ref(null);

const metaDialogVisible = ref(false);
const metaDialogMode = ref("create");
const metaDialogModule = ref("");
const metaForm = ref({
  id: "",
  name: "",
  interval: 1000,
  description: "",
  variable: "",
  params: "",
  groupId: null,
});

const systemEditorVisible = ref(false);
const scriptEditorVisible = ref(false);
const scriptEditorModule = ref("");

const contextMenuVisible = ref(false);
const contextMenuPosition = ref({ x: 0, y: 0 });
const contextMenuNode = ref(null);
const contextMenuModule = ref("");
const showMoveToMenu = ref(false);

const projectVariableNames = computed(() => Object.keys(projectVariables.value || {}).sort());
const projectVariablesList = computed(() =>
  Object.entries(projectVariables.value || {}).map(([name, detail]) => ({
    name,
    groupId: detail?.groupId || null,
    meta: {
      ...detail,
      mapped: detail?.source?.type === "dataCenter" || detail?.mapped === true,
    },
  })),
);
const customScripts = computed(() => globalScripts.value?.custom?.items || []);
const customScriptGroups = computed(() => globalScripts.value?.custom?.groups || []);
const variableGroups = computed(() => projectVariableGroups.value || []);

const scriptSearch = ref("");
const customScriptTreeRef = ref(null);
const pageSearch = ref("");
const pageTreeRef = ref(null);
const variableEnumVisible = ref(false);
const enumVariableSearch = ref("");
const enumGroupTreeRef = ref(null);
const enumSelectedGroupId = ref(null);
const enumSelectedVar = ref(null);

const selectedSystemLabel = computed(() =>
  selectedSystemKey.value === "startup" ? "系统启动" : "系统关闭",
);
const editorMetaTitle = computed(
  () => getSelectedItem(scriptEditorModule.value)?.name || "未命名脚本",
);
const editorMetaDescription = computed(
  () => getSelectedItem(scriptEditorModule.value)?.description || "无描述",
);

const variableSidebarTree = computed(() =>
  buildVariableTree(variableGroups.value, projectVariablesList.value),
);
const customScriptSidebarTree = computed(() =>
  buildScriptTree(customScriptGroups.value, customScripts.value),
);
const pageSidebarTree = computed(() => buildPageTree(pages.value || []));
const enumGroupTree = computed(() => [
  {
    id: "all",
    label: "全部",
    type: "group",
    children: buildGroupTree(variableGroups.value),
  },
]);
const enumVariableRows = computed(() => {
  const keyword = String(enumVariableSearch.value || "").toLowerCase();
  return projectVariablesList.value
    .filter((item) => {
      if (enumSelectedGroupId.value) {
        return item.groupId === enumSelectedGroupId.value;
      }
      return true;
    })
    .filter((item) => {
      if (!keyword) return true;
      return String(item.name || "")
        .toLowerCase()
        .includes(keyword);
    })
    .map((item) => ({
      name: item.name,
      type: item.meta?.type || "string",
      description: item.meta?.description || "",
      mapped: !!item.meta?.mapped,
    }));
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
      insertText: "if () {\n  \n}",
      kind: "Snippet",
      detail: "if statement",
    },
    {
      label: "for",
      insertText: "for (let i = 0; i < ; i++) {\n  \n}",
      kind: "Snippet",
      detail: "for loop",
    },
    {
      label: "function",
      insertText: "function name() {\n  \n}",
      kind: "Snippet",
      detail: "function declaration",
    },
    { label: "const", insertText: "const ", kind: "Keyword" },
    { label: "let", insertText: "let ", kind: "Keyword" },
    { label: "return", insertText: "return ", kind: "Keyword" },
  ];

  projectVariableNames.value.forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: "Variable",
      detail: "工程变量",
      prefix: "$global.",
    });
  });

  customScripts.value.forEach((script) => {
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

  items.push(...buildComponentMethodCompletions(pageComponentTree.value));

  return items;
});

const timerGroups = computed(() => globalScripts.value?.timers?.groups || []);
const variableChangeGroups = computed(() => globalScripts.value?.variableChanges?.groups || []);
const customGroups = computed(() => globalScripts.value?.custom?.groups || []);

const timerItems = computed(() => globalScripts.value?.timers?.items || []);
const variableChangeItems = computed(() => globalScripts.value?.variableChanges?.items || []);
const customItems = computed(() => globalScripts.value?.custom?.items || []);

const selectedTimer = computed(() =>
  timerItems.value.find((item) => item.id === selectedTimerId.value),
);
const selectedVariableChange = computed(() =>
  variableChangeItems.value.find((item) => item.id === selectedVariableId.value),
);
const selectedCustom = computed(() =>
  customItems.value.find((item) => item.id === selectedCustomId.value),
);

const selectedTimerGroup = computed(() =>
  timerGroups.value.find((group) => group.id === selectedTimerId.value),
);
const selectedVariableGroup = computed(() =>
  variableChangeGroups.value.find((group) => group.id === selectedVariableId.value),
);
const selectedCustomGroup = computed(() =>
  customGroups.value.find((group) => group.id === selectedCustomId.value),
);

const timerTree = computed(() => buildScriptTree(timerGroups.value, timerItems.value));
const variableChangeTree = computed(() =>
  buildScriptTree(variableChangeGroups.value, variableChangeItems.value),
);
const customTree = computed(() => buildScriptTree(customGroups.value, customItems.value));

const contextMenuStyle = computed(() => ({
  left: `${contextMenuPosition.value.x}px`,
  top: `${contextMenuPosition.value.y}px`,
}));

const submenuStyle = computed(() => {
  const subWidth = 180;
  const leftCandidate = contextMenuPosition.value.x + 180;
  const left =
    leftCandidate + subWidth > window.innerWidth
      ? contextMenuPosition.value.x - subWidth
      : leftCandidate;
  return {
    left: `${Math.max(8, left)}px`,
    top: `${contextMenuPosition.value.y}px`,
  };
});

const availableGroups = computed(() => {
  const module = contextMenuModule.value;
  const current = contextMenuNode.value;
  const groups = getGroupsByModule(module);
  if (!current || current.type !== "group") return groups;
  return groups.filter(
    (group) => group.id !== current.id && !isDescendantGroup(group.id, current.id, module),
  );
});

const groupParentOptions = computed(() => {
  const groups = getGroupsByModule(groupDialogModule.value);
  if (!groupEditMode.value) return groups;
  return groups.filter(
    (group) =>
      group.id !== groupId.value &&
      !isDescendantGroup(group.id, groupId.value, groupDialogModule.value),
  );
});

const scriptEditorTitle = computed(() => {
  const module = scriptEditorModule.value || contextMenuModule.value || "system";
  const selected = getSelectedItem(module);
  if (selected?.name) return `${selected.name}脚本`;
  if (module === "timers") return "定时器脚本";
  if (module === "variableChanges") return "变量监听脚本";
  if (module === "custom") return "自定义脚本";
  return "脚本";
});

const metaDialogTitle = computed(() => {
  const action = metaDialogMode.value === "edit" ? "编辑" : "新建";
  if (metaDialogModule.value === "timers") return `${action}定时器`;
  if (metaDialogModule.value === "variableChanges") return `${action}变量监听`;
  if (metaDialogModule.value === "custom") return `${action}自定义脚本`;
  return `${action}脚本`;
});

function syncSystemCode() {
  const system = globalScripts.value?.system || {};
  systemCode.value =
    selectedSystemKey.value === "startup"
      ? system.startup?.code || ""
      : system.shutdown?.code || "";
}

watch(selectedSystemKey, syncSystemCode, { immediate: true });
watch(() => globalScripts.value?.system, syncSystemCode, { deep: true });

watch(selectedTimer, (item) => {
  if (!item) return;
  editorCode.value = item.code || "";
});

watch(selectedVariableChange, (item) => {
  if (!item) return;
  editorCode.value = item.code || "";
});

watch(selectedCustom, (item) => {
  if (!item) return;
  editorCode.value = item.code || "";
  editorParams.value = item.params || item.args || "";
});

watch(scriptSearch, (value) => {
  customScriptTreeRef.value?.filter?.(value);
});

watch(pageSearch, (value) => {
  pageTreeRef.value?.filter?.(value);
});

function buildScriptTree(groups, items) {
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
    const label = item.name || item.variable || "未命名";
    const node = { id: item.id, label, type: "item", itemId: item.id };
    if (item.groupId && groupMap.has(item.groupId)) {
      groupMap.get(item.groupId).children.push(node);
    } else {
      roots.push(node);
    }
  });

  return roots;
}

function buildVariableTree(groups, variables) {
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

  variables.forEach((item) => {
    const node = {
      id: `var:${item.name}`,
      label: item.name,
      type: "variable",
      name: item.name,
      mapped: !!item.meta?.mapped,
    };
    if (item.groupId && groupMap.has(item.groupId)) {
      groupMap.get(item.groupId).children.push(node);
    } else {
      roots.push(node);
    }
  });

  return roots;
}

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

function buildPageTree(pageList) {
  const nodeMap = new Map();
  const roots = [];
  const normalized = Array.isArray(pageList) ? pageList : [];
  normalized
    .filter((page) => page.type !== "dialog")
    .forEach((page) => {
      nodeMap.set(page.id, {
        id: page.id,
        label: page.name || page.title || "未命名页面",
        name: page.name || page.title || "未命名页面",
        type: page.type || "page",
        parentId: page.parentId || null,
        children: [],
      });
    });

  nodeMap.forEach((node) => {
    if (node.parentId && nodeMap.has(node.parentId)) {
      nodeMap.get(node.parentId).children.push(node);
    } else {
      roots.push(node);
    }
  });
  return roots;
}

function selectSystem(key) {
  selectedSystemKey.value = key;
}

const getSelectedNodes = (module) => selectedNodes.value?.[module] || [];

function setSelectedNodes(module, nodes) {
  selectedNodes.value = {
    ...selectedNodes.value,
    [module]: nodes,
  };
}

function isScriptSelected(module, data) {
  return getSelectedNodes(module).some((node) => node.id === data.id);
}

function getMenuItemCount(nodeType) {
  if (nodeType === "blank") return 3;
  if (nodeType === "item") return 5;
  if (nodeType === "group") return 4;
  return 4;
}

function setContextMenuPosition(event, nodeType) {
  const width = 180;
  const itemHeight = 38;
  const height = getMenuItemCount(nodeType) * itemHeight + 12;
  const maxX = window.innerWidth - width - 8;
  const maxY = window.innerHeight - height - 8;
  const x = Math.max(8, Math.min(event.clientX, maxX));
  const y = Math.max(8, Math.min(event.clientY, maxY));
  contextMenuPosition.value = { x, y };
}

const isMixedSelection = computed(() => {
  const module = contextMenuModule.value;
  const nodes = getSelectedNodes(module);
  if (nodes.length <= 1) return false;
  const types = new Set(nodes.map((node) => node.type));
  return types.size > 1;
});

const canOpenEditScript = computed(() => {
  const module = contextMenuModule.value;
  const nodes = getSelectedNodes(module);
  if (nodes.length !== 1) return false;
  return nodes[0]?.type === "item";
});

const canEditGroup = computed(() => {
  const module = contextMenuModule.value;
  const nodes = getSelectedNodes(module);
  if (nodes.length !== 1) return false;
  return nodes[0]?.type === "group";
});

const canDeleteSelection = computed(() => !isMixedSelection.value);
function insertText(text) {
  if (systemEditorVisible.value) {
    systemEditorRef.value?.insertText?.(text);
    return;
  }
  if (scriptEditorVisible.value) {
    activeEditorRef.value?.insertText?.(text);
  }
}

function handleVariableInsert(data) {
  if (data?.type !== "variable") return;
  insertText(`$global.${data.name}`);
}

function handleCustomScriptInsert(data) {
  if (data?.type !== "item") return;
  const script = customScripts.value.find((item) => item.id === data.itemId);
  if (!script?.name) return;
  const params =
    typeof script.params === "string" && script.params.trim()
      ? script.params.trim()
      : typeof script.args === "string"
        ? script.args.trim()
        : "";
  const call = params ? `${script.name}(${params})` : `${script.name}()`;
  insertText(`customScripts.${call}`);
}

function handlePageInsert(data) {
  if (!data || data.type !== "page") return;
  const name = data.name || data.label;
  if (!name) return;
  insertText(`components.pages["${name}"]`);
}

function handleEnumGroupSelect(data) {
  if (!data) {
    enumSelectedGroupId.value = null;
    return;
  }
  enumSelectedGroupId.value = data.id === "all" ? null : data.id;
}

function handleEnumRowClick(row) {
  enumSelectedVar.value = row || null;
}

function handleEnumRowDblClick(row) {
  enumSelectedVar.value = row || null;
  confirmEnumInsert();
}

function enumRowClass({ row }) {
  if (enumSelectedVar.value?.name === row.name) return "is-selected";
  return "";
}

function filterSidebarNode(value, data) {
  if (!value) return true;
  const keyword = String(value).toLowerCase();
  return String(data?.label || "")
    .toLowerCase()
    .includes(keyword);
}

function openVariableEnum() {
  enumVariableSearch.value = "";
  enumSelectedVar.value = null;
  enumSelectedGroupId.value = null;
  variableEnumVisible.value = true;
}

function confirmEnumInsert() {
  if (!enumSelectedVar.value?.name) return;
  insertText(`$global.${enumSelectedVar.value.name}`);
  variableEnumVisible.value = false;
}

function formatSystemCode() {
  systemEditorRef.value?.format?.();
}

function formatActiveCode() {
  activeEditorRef.value?.format?.();
}

function openSystemEditor(key) {
  if (contextMenuVisible.value) closeContextMenu();
  if (key) selectedSystemKey.value = key;
  systemOriginalCode.value = systemCode.value || "";
  systemEditorVisible.value = true;
}

function openScriptEditor(module, data) {
  if (contextMenuVisible.value) closeContextMenu();
  if (!canOpenEditScript.value && !data) return;
  if (data?.type === "item") {
    handleScriptNodeClick(module, data);
  }
  const selected = getSelectedItem(module);
  if (!selected) return;
  editorOriginalCode.value = selected.code || "";
  editorCode.value = selected.code || "";
  const interval = Number(selected.interval ?? selected.time ?? selected.schedule ?? 1000);
  editorInterval.value = Number.isFinite(interval) ? interval : 1000;
  editorOriginalInterval.value = editorInterval.value;
  editorParams.value = selected.params || selected.args || "";
  scriptEditorModule.value = module;
  scriptEditorVisible.value = true;
}

async function persistGlobals() {
  if (!projectId.value) {
    ElMessage.error("缺少工程信息，无法保存");
    return;
  }
  const result = await editorStore.saveProjectSettings();
  if (!result.ok) {
    ElMessage.error(result.error?.message || "保存失败");
  }
}

async function saveSystemScript() {
  const scripts = globalScripts.value || {};
  const system = scripts.system || {
    startup: { code: "" },
    shutdown: { code: "" },
  };
  const next = {
    ...system,
    [selectedSystemKey.value]: {
      ...system[selectedSystemKey.value],
      code: systemCode.value || "",
    },
  };
  globalScripts.value = { ...scripts, system: next };
  systemOriginalCode.value = systemCode.value || "";
  await persistGlobals();
  ElMessage.success("已保存脚本");
}

function handleScriptNodeClick(module, data, event) {
  const isCtrl = Boolean(event?.ctrlKey || event?.metaKey);
  const current = getSelectedNodes(module);
  if (isCtrl) {
    if (current.some((node) => node.id === data.id)) {
      setSelectedNodes(
        module,
        current.filter((node) => node.id !== data.id),
      );
    } else {
      setSelectedNodes(module, [...current, data]);
    }
  } else {
    setSelectedNodes(module, [data]);
  }

  if (data.type === "item") {
    if (module === "timers") selectedTimerId.value = data.itemId;
    if (module === "variableChanges") selectedVariableId.value = data.itemId;
    if (module === "custom") selectedCustomId.value = data.itemId;
  } else if (data.type === "group") {
    if (module === "timers") selectedTimerId.value = data.id;
    if (module === "variableChanges") selectedVariableId.value = data.id;
    if (module === "custom") selectedCustomId.value = data.id;
  }
  if (contextMenuVisible.value) closeContextMenu();
}

function handleTreeContextMenu(module, event, data) {
  event.preventDefault();
  event.stopPropagation();
  if (!isScriptSelected(module, data)) {
    setSelectedNodes(module, [data]);
  }
  contextMenuModule.value = module;
  contextMenuNode.value = data;
  setContextMenuPosition(event, data.type);
  contextMenuVisible.value = true;
  showMoveToMenu.value = false;
  if (data.type === "item") {
    if (module === "timers") selectedTimerId.value = data.itemId;
    if (module === "variableChanges") selectedVariableId.value = data.itemId;
    if (module === "custom") selectedCustomId.value = data.itemId;
  } else if (data.type === "group") {
    if (module === "timers") selectedTimerId.value = data.id;
    if (module === "variableChanges") selectedVariableId.value = data.id;
    if (module === "custom") selectedCustomId.value = data.id;
  }
}

function handleBlankContextMenu(module, event) {
  event.preventDefault();
  event.stopPropagation();
  setSelectedNodes(module, []);
  contextMenuModule.value = module;
  contextMenuNode.value = { type: "blank" };
  setContextMenuPosition(event, "blank");
  contextMenuVisible.value = true;
  showMoveToMenu.value = false;
}

function closeContextMenu() {
  contextMenuVisible.value = false;
  contextMenuNode.value = null;
  showMoveToMenu.value = false;
}

function openGroupEditFromMenu() {
  if (!canEditGroup.value) return;
  const module = contextMenuModule.value;
  closeContextMenu();
  openGroupEdit(module);
}

function openGroupCreateFromMenu() {
  const module = contextMenuModule.value;
  const groupIdValue = contextMenuNode.value?.type === "group" ? contextMenuNode.value?.id : null;
  closeContextMenu();
  groupDialogModule.value = module;
  groupEditMode.value = false;
  groupId.value = "";
  groupName.value = "";
  groupParentId.value = groupIdValue;
  groupDialogVisible.value = true;
}

function handleMoveTo(groupIdValue) {
  const module = contextMenuModule.value;
  const node = contextMenuNode.value;
  closeContextMenu();
  if (!module || !node) return;
  const nodesToMove = getSelectedNodes(module).length ? getSelectedNodes(module) : [node];
  let nextItems = getItemsByModule(module);
  let nextGroups = getGroupsByModule(module);
  let blocked = false;

  nodesToMove.forEach((item) => {
    if (item.type === "item") {
      nextItems = nextItems.map((row) =>
        row.id === item.itemId ? { ...row, groupId: groupIdValue } : row,
      );
    } else if (item.type === "group") {
      if (groupIdValue && isDescendantGroup(groupIdValue, item.id, module)) {
        blocked = true;
        return;
      }
      nextGroups = nextGroups.map((row) =>
        row.id === item.id ? { ...row, parentId: groupIdValue } : row,
      );
    }
  });

  if (blocked) {
    ElMessage.warning("无法移动到子分组");
  }

  updateModuleGroups(module, nextGroups);
  updateModuleItems(module, nextItems);
}

function allowScriptDrag() {
  return true;
}

function allowScriptDrop(module, draggingNode, dropNode, type) {
  const dragData = draggingNode.data;
  const dropData = dropNode.data;

  if (dragData.type === "item") {
    if (type === "inner" && dropData.type !== "group") return false;
    return true;
  }

  if (dragData.type === "group") {
    if (type === "inner" && dropData.type !== "group") return false;
    if (dropData.type === "group" && isDescendantGroup(dropData.id, dragData.id, module)) {
      return false;
    }
    if (type === "inner" && dropData.type === "group") {
      const depth = getGroupDepth(dropData.id, module) + getGroupSubtreeDepth(dragData.id, module);
      if (depth > maxGroupDepth) return false;
    }
    return true;
  }

  return false;
}

function handleScriptDrop(module, draggingNode, dropNode, dropType) {
  const dragData = draggingNode.data;
  const targetGroupId = resolveTargetGroupId(dropNode, dropType);

  if (dragData.type === "item") {
    const items = getItemsByModule(module).map((item) =>
      item.id === dragData.itemId ? { ...item, groupId: targetGroupId } : item,
    );
    updateModuleItems(module, items);
    return;
  }

  if (dragData.type === "group") {
    const groups = getGroupsByModule(module).map((group) =>
      group.id === dragData.id ? { ...group, parentId: targetGroupId } : group,
    );
    updateModuleGroups(module, groups);
  }
}

function resolveTargetGroupId(dropNode, dropType) {
  const dropData = dropNode.data;
  if (dropType === "inner") {
    return dropData.type === "group" ? dropData.id : null;
  }
  const parent = dropNode.parent?.data;
  return parent?.type === "group" ? parent.id : null;
}

async function removeScript(module) {
  if (contextMenuVisible.value) closeContextMenu();
  if (!canDeleteSelection.value) {
    ElMessage.warning("分组与成员混选时不能删除");
    return;
  }
  const selected = getSelectedItem(module);
  const nodes = getSelectedNodes(module);
  const itemsToRemove = nodes.filter((node) => node.type === "item").map((node) => node.itemId);
  const groupsToRemove = nodes.filter((node) => node.type === "group").map((node) => node.id);
  if (!itemsToRemove.length && !groupsToRemove.length && !selected) return;
  try {
    const count = itemsToRemove.length + groupsToRemove.length || 1;
    let message = `确定删除 "${selected?.name || "未命名"}" 吗？`;
    if (count > 1) {
      message = `确定删除选中的 ${count} 项吗？`;
    } else if (groupsToRemove.length === 1 && itemsToRemove.length === 0) {
      const name = getSelectedNodes(module).find((node) => node.type === "group")?.label || "";
      message = `确定删除分组 "${name}" 吗？组内成员会移动到父级。`;
    }
    await ElMessageBox.confirm(message, "确认删除", {
      type: "warning",
      lockScroll: false,
    });
  } catch (error) {
    return;
  }
  const items = getItemsByModule(module);
  const groups = getGroupsByModule(module);
  const nextItems = items.filter((item) => !itemsToRemove.includes(item.id));

  if (groupsToRemove.length) {
    const parentMap = new Map();
    groups.forEach((group) => {
      if (groupsToRemove.includes(group.id)) {
        parentMap.set(group.id, group.parentId || null);
      }
    });

    const nextGroups = groups
      .filter((group) => !groupsToRemove.includes(group.id))
      .map((group) =>
        groupsToRemove.includes(group.parentId)
          ? { ...group, parentId: parentMap.get(group.parentId) || null }
          : group,
      );

    const reboundItems = nextItems.map((item) =>
      groupsToRemove.includes(item.groupId)
        ? { ...item, groupId: parentMap.get(item.groupId) || null }
        : item,
    );

    updateModuleGroups(module, nextGroups);
    updateModuleItems(module, reboundItems);
  } else {
    updateModuleItems(module, nextItems);
  }
}

function copyScript(module) {
  if (contextMenuVisible.value) closeContextMenu();
  const nodes = getSelectedNodes(module);
  const items = nodes
    .filter((node) => node.type === "item")
    .map((node) => getItemsByModule(module).find((item) => item.id === node.itemId))
    .filter(Boolean);
  if (!items.length) {
    const selected = getSelectedItem(module);
    if (!selected) return;
    items.push(selected);
  }
  scriptClipboard.value = {
    module,
    items: items.map((item) => JSON.parse(JSON.stringify(item))),
  };
  ElMessage.success(`已复制 ${scriptClipboard.value.items.length} 个脚本`);
}

async function pasteScript(module) {
  if (contextMenuVisible.value) closeContextMenu();
  if (!scriptClipboard.value?.items?.length) return;
  const items = getItemsByModule(module);
  const targetGroupId = getSelectedGroupId(module);
  const nextItems = [...items];

  scriptClipboard.value.items.forEach((source) => {
    const nameBase = source.name || "脚本";
    let name = nameBase;
    let index = 1;
    while (nextItems.some((item) => item.name === name)) {
      name = `${nameBase}_copy${index}`;
      index += 1;
    }
    nextItems.push({
      ...source,
      id: createId(),
      name,
      groupId: targetGroupId ?? source.groupId ?? null,
    });
  });

  updateModuleItems(module, nextItems);
}

function getSelectedItem(module) {
  if (module === "timers") return selectedTimer.value;
  if (module === "variableChanges") return selectedVariableChange.value;
  if (module === "custom") return selectedCustom.value;
  return null;
}

function getGroupsByModule(module) {
  if (module === "timers") return timerGroups.value;
  if (module === "variableChanges") return variableChangeGroups.value;
  if (module === "custom") return customGroups.value;
  return [];
}

function getItemsByModule(module) {
  if (module === "timers") return timerItems.value;
  if (module === "variableChanges") return variableChangeItems.value;
  if (module === "custom") return customItems.value;
  return [];
}

function getSelectedGroupId(module) {
  if (module === "timers") {
    if (selectedTimerGroup.value) return selectedTimerGroup.value.id;
    if (selectedTimer.value?.groupId) return selectedTimer.value.groupId;
  }
  if (module === "variableChanges") {
    if (selectedVariableGroup.value) return selectedVariableGroup.value.id;
    if (selectedVariableChange.value?.groupId) return selectedVariableChange.value.groupId;
  }
  if (module === "custom") {
    if (selectedCustomGroup.value) return selectedCustomGroup.value.id;
    if (selectedCustom.value?.groupId) return selectedCustom.value.groupId;
  }
  return null;
}

function updateModuleGroups(module, groups) {
  globalScripts.value = {
    ...globalScripts.value,
    [module]: {
      ...(globalScripts.value?.[module] || {}),
      groups,
    },
  };
  persistGlobals();
}

function updateModuleItems(module, items) {
  globalScripts.value = {
    ...globalScripts.value,
    [module]: {
      ...(globalScripts.value?.[module] || {}),
      items,
    },
  };
  persistGlobals();
}

function openGroupCreate(module) {
  groupEditMode.value = false;
  groupDialogModule.value = module;
  groupId.value = "";
  groupName.value = "";
  groupParentId.value = null;
  groupDialogVisible.value = true;
}

function openGroupEdit(module) {
  const groups = getGroupsByModule(module);
  const selectedId =
    module === "timers"
      ? selectedTimerId.value
      : module === "variableChanges"
        ? selectedVariableId.value
        : selectedCustomId.value;
  const group = groups.find((item) => item.id === selectedId);
  if (!group) return;
  groupEditMode.value = true;
  groupDialogModule.value = module;
  groupId.value = group.id;
  groupName.value = group.name;
  groupParentId.value = group.parentId || null;
  groupDialogVisible.value = true;
}

function createId() {
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return `id_${Date.now()}_${Math.floor(Math.random() * 1000)}`;
}

async function saveGroup() {
  const name = groupName.value.trim();
  if (!name) return ElMessage.warning("分组名不能为空");

  const parentId = groupParentId.value || null;
  const depth = parentId ? getGroupDepth(parentId, groupDialogModule.value) + 1 : 1;
  if (depth > maxGroupDepth) {
    return ElMessage.warning(`分组最多支持 ${maxGroupDepth} 层`);
  }

  const groups = getGroupsByModule(groupDialogModule.value);
  if (groupEditMode.value) {
    updateModuleGroups(
      groupDialogModule.value,
      groups.map((group) => (group.id === groupId.value ? { ...group, name, parentId } : group)),
    );
  } else {
    updateModuleGroups(groupDialogModule.value, [
      ...groups,
      { id: createId(), name, parentId, sortOrder: 0 },
    ]);
  }
  groupDialogVisible.value = false;
}

async function removeGroup(module) {
  if (!canDeleteSelection.value) {
    ElMessage.warning("分组与成员混选时不能删除");
    return;
  }
  const groups = getGroupsByModule(module);
  const selectedId =
    module === "timers"
      ? selectedTimerId.value
      : module === "variableChanges"
        ? selectedVariableId.value
        : selectedCustomId.value;
  const group = groups.find((item) => item.id === selectedId);
  if (!group) return;

  try {
    await ElMessageBox.confirm(
      `确定删除分组 "${group.name}" 吗？组内成员会移动到父级。`,
      "确认删除",
      {
        type: "warning",
        lockScroll: false,
      },
    );
  } catch (error) {
    return;
  }

  const parentId = group.parentId || null;
  const nextGroups = groups
    .filter((item) => item.id !== group.id)
    .map((item) => (item.parentId === group.id ? { ...item, parentId } : item));

  const items = getItemsByModule(module).map((item) =>
    item.groupId === group.id ? { ...item, groupId: parentId } : item,
  );

  updateModuleGroups(module, nextGroups);
  updateModuleItems(module, items);
}

function getGroupDepth(groupIdValue, module) {
  if (!groupIdValue) return 0;
  let depth = 1;
  let currentId = groupIdValue;
  const groups = getGroupsByModule(module);
  const map = new Map(groups.map((group) => [group.id, group]));
  while (map.get(currentId)?.parentId) {
    depth += 1;
    currentId = map.get(currentId).parentId;
  }
  return depth;
}

function getGroupSubtreeDepth(groupIdValue, module) {
  const groups = getGroupsByModule(module);
  const children = groups.filter((group) => group.parentId === groupIdValue);
  if (!children.length) return 1;
  const depths = children.map((child) => getGroupSubtreeDepth(child.id, module));
  return 1 + Math.max(...depths);
}

function isDescendantGroup(targetId, parentId, module) {
  const groups = getGroupsByModule(module);
  let current = groups.find((group) => group.id === targetId);
  while (current?.parentId) {
    if (current.parentId === parentId) return true;
    current = groups.find((group) => group.id === current.parentId);
  }
  return false;
}

function saveScriptCode(module) {
  const selected = getSelectedItem(module);
  if (!selected) return;
  const items = getItemsByModule(module).map((item) =>
    item.id === selected.id
      ? {
          ...item,
          code: editorCode.value || "",
          interval: module === "timers" ? Number(editorInterval.value) || 1000 : item.interval,
        }
      : item,
  );
  updateModuleItems(module, items);
  editorOriginalCode.value = editorCode.value || "";
  editorOriginalInterval.value = editorInterval.value;
  ElMessage.success("已保存脚本");
}

function saveActiveScript() {
  if (scriptEditorModule.value === "timers") {
    saveScriptCode("timers");
  } else if (scriptEditorModule.value === "variableChanges") {
    saveScriptCode("variableChanges");
  } else if (scriptEditorModule.value === "custom") {
    saveScriptCode("custom");
  }
}
async function handleSystemBeforeClose(done) {
  const isDirty = (systemCode.value || "") !== (systemOriginalCode.value || "");
  if (!isDirty) {
    done();
    return;
  }
  try {
    await ElMessageBox.confirm("脚本已修改，是否保存？", "提示", {
      confirmButtonText: "保存",
      cancelButtonText: "不保存",
      type: "warning",
      lockScroll: false,
    });
    await saveSystemScript();
    done();
  } catch (error) {
    done();
  }
}

async function handleScriptBeforeClose(done) {
  const codeDirty = (editorCode.value || "") !== (editorOriginalCode.value || "");
  const intervalDirty =
    scriptEditorModule.value === "timers" && editorInterval.value !== editorOriginalInterval.value;
  const isDirty = codeDirty || intervalDirty;
  if (!isDirty) {
    done();
    return;
  }
  try {
    await ElMessageBox.confirm("脚本已修改，是否保存？", "提示", {
      confirmButtonText: "保存",
      cancelButtonText: "不保存",
      type: "warning",
      lockScroll: false,
    });
    saveActiveScript();
    done();
  } catch (error) {
    done();
  }
}

function handleEditorShortcut(event) {
  if (!systemEditorVisible.value && !scriptEditorVisible.value) return;
  if (!(event.ctrlKey || event.metaKey)) return;
  const key = event.key.toLowerCase();
  if (key === "s") {
    event.preventDefault();
    if (systemEditorVisible.value) saveSystemScript();
    if (scriptEditorVisible.value) saveActiveScript();
  }
  if (key === "f" && event.shiftKey && event.altKey) {
    event.preventDefault();
    if (systemEditorVisible.value) formatSystemCode();
    if (scriptEditorVisible.value) formatActiveCode();
  }
}

function openMetaDialog(module, mode) {
  if (contextMenuVisible.value) closeContextMenu();
  if (mode === "edit" && getSelectedNodes(module).length > 1) {
    ElMessage.warning("多选时不能编辑");
    return;
  }
  metaDialogMode.value = mode;
  metaDialogModule.value = module;
  const selected = getSelectedItem(module);
  if (mode === "edit" && selected) {
    metaForm.value = {
      id: selected.id,
      name: selected.name || selected.variable || "",
      interval: Number(selected.interval ?? selected.time ?? selected.schedule ?? 1000),
      description: selected.description || "",
      variable: selected.variable || "",
      params: selected.params || selected.args || "",
      groupId: selected.groupId || null,
    };
  } else {
    const groupIdValue =
      contextMenuNode.value?.type === "group"
        ? contextMenuNode.value?.id
        : getSelectedGroupId(module);
    metaForm.value = {
      id: "",
      name: "",
      interval: 1000,
      description: "",
      variable: "",
      params: "",
      groupId: groupIdValue || null,
    };
  }
  metaDialogVisible.value = true;
}

function saveMetaDialog() {
  const module = metaDialogModule.value;
  if (!module) return;
  const items = getItemsByModule(module);

  if (module === "timers") {
    const name = metaForm.value.name.trim();
    if (!name) return ElMessage.warning("请输入定时器名称");
    const exists = items.some((item) => item.name === name && item.id !== metaForm.value.id);
    if (exists) return ElMessage.warning("定时器名称已存在");
    if (!Number.isFinite(Number(metaForm.value.interval))) {
      return ElMessage.warning("请输入正确的时间");
    }
  }

  if (module === "variableChanges") {
    if (!metaForm.value.variable) return ElMessage.warning("请选择变量");
  }

  if (module === "custom") {
    const name = metaForm.value.name.trim();
    if (!name) return ElMessage.warning("请输入函数名称");
    const exists = items.some((item) => item.name === name && item.id !== metaForm.value.id);
    if (exists) return ElMessage.warning("函数名称已存在");
  }

  if (metaDialogMode.value === "create") {
    const groupIdValue = metaForm.value.groupId || null;
    const newItem = {
      id: createId(),
      name: module === "variableChanges" ? metaForm.value.variable : metaForm.value.name.trim(),
      description: metaForm.value.description || "",
      variable: metaForm.value.variable || "",
      params: metaForm.value.params || "",
      interval: Number(metaForm.value.interval) || 1000,
      groupId: groupIdValue,
      code: "",
    };
    updateModuleItems(module, [...items, newItem]);
  } else {
    const next = items.map((item) => {
      if (item.id !== metaForm.value.id) return item;
      return {
        ...item,
        name: module === "variableChanges" ? metaForm.value.variable : metaForm.value.name.trim(),
        description: metaForm.value.description || "",
        variable: metaForm.value.variable || item.variable || "",
        params: metaForm.value.params || "",
        interval: Number(metaForm.value.interval) || item.interval || 1000,
      };
    });
    updateModuleItems(module, next);
  }

  metaDialogVisible.value = false;
}

function handleClickOutside() {
  if (contextMenuVisible.value) closeContextMenu();
}

onMounted(() => {
  document.addEventListener("click", handleClickOutside);
  window.addEventListener("keydown", handleEditorShortcut);
});

onUnmounted(() => {
  document.removeEventListener("click", handleClickOutside);
  window.removeEventListener("keydown", handleEditorShortcut);
});
</script>

<template>
  <div class="global-scripts">
    <el-collapse v-model="activeSections" class="scripts-collapse" :accordion="true">
      <ScriptVarsSystemSection
        :selected-system-key="selectedSystemKey"
        @select-system="selectSystem"
        @open-system-editor="openSystemEditor"
      />

      <ScriptVarsTimersSection
        :tree="timerTree"
        :allow-drop="
          (draggingNode, dropNode, dropType) =>
            allowScriptDrop('timers', draggingNode, dropNode, dropType)
        "
        :allow-drag="allowScriptDrag"
        :is-selected="(data) => isScriptSelected('timers', data)"
        @blank-contextmenu="(event) => handleBlankContextMenu('timers', event)"
        @node-dblclick="(data) => openScriptEditor('timers', data)"
        @node-contextmenu="(event, data) => handleTreeContextMenu('timers', event, data)"
        @node-drop="
          (draggingNode, dropNode, dropType) =>
            handleScriptDrop('timers', draggingNode, dropNode, dropType)
        "
        @node-click="(data, event) => handleScriptNodeClick('timers', data, event)"
      />
      <ScriptVarsVariableChangesSection
        :tree="variableChangeTree"
        :allow-drop="
          (draggingNode, dropNode, dropType) =>
            allowScriptDrop('variableChanges', draggingNode, dropNode, dropType)
        "
        :allow-drag="allowScriptDrag"
        :is-selected="(data) => isScriptSelected('variableChanges', data)"
        @blank-contextmenu="(event) => handleBlankContextMenu('variableChanges', event)"
        @node-dblclick="(data) => openScriptEditor('variableChanges', data)"
        @node-contextmenu="(event, data) => handleTreeContextMenu('variableChanges', event, data)"
        @node-drop="
          (draggingNode, dropNode, dropType) =>
            handleScriptDrop('variableChanges', draggingNode, dropNode, dropType)
        "
        @node-click="(data, event) => handleScriptNodeClick('variableChanges', data, event)"
      />
      <ScriptVarsCustomSection
        :tree="customTree"
        :allow-drop="
          (draggingNode, dropNode, dropType) =>
            allowScriptDrop('custom', draggingNode, dropNode, dropType)
        "
        :allow-drag="allowScriptDrag"
        :is-selected="(data) => isScriptSelected('custom', data)"
        @blank-contextmenu="(event) => handleBlankContextMenu('custom', event)"
        @node-dblclick="(data) => openScriptEditor('custom', data)"
        @node-contextmenu="(event, data) => handleTreeContextMenu('custom', event, data)"
        @node-drop="
          (draggingNode, dropNode, dropType) =>
            handleScriptDrop('custom', draggingNode, dropNode, dropType)
        "
        @node-click="(data, event) => handleScriptNodeClick('custom', data, event)"
      />
    </el-collapse>
    <VariableGroupFormDialog
      v-model="groupDialogVisible"
      v-model:name="groupName"
      v-model:parent-id="groupParentId"
      :edit-mode="groupEditMode"
      :parent-options="groupParentOptions"
      @confirm="saveGroup"
    />

    <el-dialog
      v-model="metaDialogVisible"
      :title="metaDialogTitle"
      width="420px"
      :close-on-click-modal="false"
      :lock-scroll="false"
    >
      <el-form label-width="90px">
        <template v-if="metaDialogModule === 'timers'">
          <el-form-item label="定时器名称">
            <el-input v-model="metaForm.name" />
          </el-form-item>
          <el-form-item label="时间(ms)">
            <el-input-number
              v-model="metaForm.interval"
              :min="100"
              :step="100"
              style="width: 100%"
            />
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="metaForm.description" />
          </el-form-item>
        </template>
        <template v-else-if="metaDialogModule === 'variableChanges'">
          <el-form-item label="变量">
            <el-select v-model="metaForm.variable" placeholder="请选择变量">
              <el-option
                v-for="name in projectVariableNames"
                :key="name"
                :label="name"
                :value="name"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="metaForm.description" />
          </el-form-item>
        </template>
        <template v-else-if="metaDialogModule === 'custom'">
          <el-form-item label="函数名称">
            <el-input v-model="metaForm.name" />
          </el-form-item>
          <el-form-item label="入参">
            <el-input v-model="metaForm.params" placeholder="例如: id, value" />
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="metaForm.description" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="metaDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveMetaDialog">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="systemEditorVisible"
      :title="`${selectedSystemLabel}脚本`"
      width="980px"
      top="3vh"
      :close-on-click-modal="false"
      :lock-scroll="false"
      :before-close="handleSystemBeforeClose"
    >
      <div class="editor-meta">
        <div class="meta-title">{{ selectedSystemLabel }}</div>
        <div class="meta-desc">系统脚本</div>
        <div class="meta-actions">
          <el-tooltip content="枚举工程变量" placement="top">
            <el-button class="icon-button" size="small" circle @click="openVariableEnum">
              <IconEpList />
            </el-button>
          </el-tooltip>
        </div>
      </div>
      <div class="editor-body">
        <div class="editor-main">
          <MonacoEditor
            ref="systemEditorRef"
            v-model="systemCode"
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
                ref="customScriptTreeRef"
                :data="customScriptSidebarTree"
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
                    <span
                      class="node-label"
                      :class="{
                        'is-group': data.type === 'group' || data.type === 'folder',
                      }"
                      >{{ data.label }}</span
                    >
                  </div>
                </template>
              </el-tree>
            </div>
          </div>
          <div class="sidebar-section">
            <div class="sidebar-title">页面</div>
            <el-input v-model="pageSearch" size="small" placeholder="搜索页面/分组" clearable />
            <div class="sidebar-scroll">
              <el-tree
                ref="pageTreeRef"
                :data="pageSidebarTree"
                node-key="id"
                :default-expand-all="true"
                :expand-on-click-node="false"
                :filter-node-method="filterSidebarNode"
                @node-click="handlePageInsert"
              >
                <template #default="{ data }">
                  <div class="tree-node" :class="`node-${data.type}`">
                    <el-icon class="node-icon icon-page">
                      <IconEpFolder v-if="data.type === 'folder'" />
                      <IconEpDocument v-else />
                    </el-icon>
                    <span
                      class="node-label"
                      :class="{
                        'is-group': data.type === 'group' || data.type === 'folder',
                      }"
                      >{{ data.label }}</span
                    >
                  </div>
                </template>
              </el-tree>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="systemEditorVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSystemScript">保存 (Ctrl+S)</el-button>
      </template>
    </el-dialog>
    <el-dialog
      v-model="scriptEditorVisible"
      :title="scriptEditorTitle"
      width="980px"
      top="3vh"
      :close-on-click-modal="false"
      :lock-scroll="false"
      :before-close="handleScriptBeforeClose"
    >
      <div class="editor-meta">
        <div class="meta-title">{{ editorMetaTitle }}</div>
        <div class="meta-desc">{{ editorMetaDescription }}</div>
        <div v-if="scriptEditorModule === 'timers'" class="meta-inline">
          <span class="meta-label">时间(ms)</span>
          <el-input-number v-model="editorInterval" :min="100" :step="100" size="small" />
        </div>
        <div v-if="scriptEditorModule === 'custom'" class="meta-inline">
          <span class="meta-label">入参</span>
          <span class="meta-value">{{ editorParams || "无" }}</span>
        </div>
        <div class="meta-actions">
          <el-tooltip content="枚举工程变量" placement="top">
            <el-button class="icon-button" size="small" circle @click="openVariableEnum">
              <IconEpList />
            </el-button>
          </el-tooltip>
        </div>
      </div>
      <div class="editor-body">
        <div class="editor-main">
          <MonacoEditor
            ref="activeEditorRef"
            v-model="editorCode"
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
                ref="customScriptTreeRef"
                :data="customScriptSidebarTree"
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
                    <span class="node-label" :class="{ 'is-group': data.type === 'group' }">{{
                      data.label
                    }}</span>
                  </div>
                </template>
              </el-tree>
            </div>
          </div>
          <div class="sidebar-section">
            <div class="sidebar-title">页面</div>
            <el-input v-model="pageSearch" size="small" placeholder="搜索页面/分组" clearable />
            <div class="sidebar-scroll">
              <el-tree
                ref="pageTreeRef"
                :data="pageSidebarTree"
                node-key="id"
                :default-expand-all="true"
                :expand-on-click-node="false"
                :filter-node-method="filterSidebarNode"
                @node-click="handlePageInsert"
              >
                <template #default="{ data }">
                  <div class="tree-node" :class="`node-${data.type}`">
                    <el-icon class="node-icon icon-page">
                      <IconEpFolder v-if="data.type === 'folder'" />
                      <IconEpDocument v-else />
                    </el-icon>
                    <span class="node-label" :class="{ 'is-group': data.type === 'group' }">{{
                      data.label
                    }}</span>
                  </div>
                </template>
              </el-tree>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="scriptEditorVisible = false">取消</el-button>
        <el-button type="primary" @click="saveActiveScript">保存 (Ctrl+S)</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="variableEnumVisible"
      title="工程变量"
      width="760px"
      :close-on-click-modal="false"
      :lock-scroll="false"
    >
      <div class="enum-layout">
        <div class="enum-left">
          <div class="sidebar-title">分组</div>
          <el-tree
            ref="enumGroupTreeRef"
            :data="enumGroupTree"
            node-key="id"
            :default-expand-all="true"
            :expand-on-click-node="false"
            :filter-node-method="filterSidebarNode"
            @node-click="handleEnumGroupSelect"
          >
            <template #default="{ data }">
              <div class="tree-node" :class="`node-${data.type}`">
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
            v-model="enumVariableSearch"
            size="small"
            placeholder="搜索工程变量"
            clearable
          />
          <el-table
            :data="enumVariableRows"
            size="small"
            height="360"
            highlight-current-row
            :row-class-name="enumRowClass"
            @row-click="handleEnumRowClick"
            @row-dblclick="handleEnumRowDblClick"
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
      <template #footer>
        <el-button @click="variableEnumVisible = false">取消</el-button>
        <el-button type="primary" :disabled="!enumSelectedVar" @click="confirmEnumInsert"
          >插入</el-button
        >
      </template>
    </el-dialog>

    <div
      v-if="contextMenuVisible"
      class="context-menu"
      :style="contextMenuStyle"
      @click.stop
      @mousedown.stop
    >
      <template v-if="contextMenuNode?.type === 'blank'">
        <div class="context-menu-item" @click="openGroupCreateFromMenu">新建分组</div>
        <div class="context-menu-item" @click="openMetaDialog(contextMenuModule, 'create')">
          新建脚本
        </div>
        <div
          class="context-menu-item"
          :class="{ 'is-disabled': !scriptClipboard }"
          @click="pasteScript(contextMenuModule)"
        >
          粘贴
        </div>
      </template>
      <template v-else-if="contextMenuNode?.type === 'item'">
        <div
          class="context-menu-item"
          :class="{ 'is-disabled': !canOpenEditScript }"
          @click="openScriptEditor(contextMenuModule)"
        >
          打开
        </div>
        <div
          class="context-menu-item"
          :class="{ 'is-disabled': !canOpenEditScript }"
          @click="openMetaDialog(contextMenuModule, 'edit')"
        >
          编辑
        </div>
        <div class="context-menu-item" @click="copyScript(contextMenuModule)">复制</div>
        <div class="context-menu-item" @click="showMoveToMenu = !showMoveToMenu">移动到</div>
        <div
          class="context-menu-item context-menu-item--danger"
          :class="{ 'is-disabled': !canDeleteSelection }"
          @click="removeScript(contextMenuModule)"
        >
          删除
        </div>
      </template>
      <template v-else-if="contextMenuNode?.type === 'group'">
        <div
          class="context-menu-item"
          :class="{ 'is-disabled': !canEditGroup }"
          @click="openGroupEditFromMenu"
        >
          编辑分组
        </div>
        <div class="context-menu-item" @click="openGroupCreateFromMenu">新建子分组</div>
        <div class="context-menu-item" @click="showMoveToMenu = !showMoveToMenu">移动到</div>
        <div
          class="context-menu-item context-menu-item--danger"
          :class="{ 'is-disabled': !canDeleteSelection }"
          @click="removeGroup(contextMenuModule)"
        >
          删除分组
        </div>
      </template>
    </div>

    <div
      v-if="contextMenuVisible && showMoveToMenu"
      class="context-menu context-submenu"
      :style="submenuStyle"
      @click.stop
      @mousedown.stop
    >
      <div class="context-menu-item" @click="handleMoveTo(null)">根目录</div>
      <div
        v-for="group in availableGroups"
        :key="group.id"
        class="context-menu-item"
        @click="handleMoveTo(group.id)"
      >
        {{ group.name }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.global-scripts {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.scripts-collapse {
  flex: 1;
  overflow: auto;
  padding-bottom: 8px;
  display: flex;
  flex-direction: column;
}

:deep(.scripts-collapse .el-collapse-item__header) {
  font-weight: 600;
}

:deep(.scripts-collapse .el-collapse-item) {
  display: flex;
  flex-direction: column;
}

:deep(.scripts-collapse .el-collapse-item.is-active) {
  flex: 1;
}

:deep(.scripts-collapse .el-collapse-item__wrap) {
  flex: 1;
  display: flex;
  flex-direction: column;
}

:deep(.scripts-collapse .el-collapse-item__content) {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 0;
}

.scripts-layout {
  display: flex;
  gap: 12px;
  flex: 1;
  padding: 6px 4px;
  box-sizing: border-box;
  min-height: 320px;
}

.scripts-list {
  width: 100%;
  border: 1px solid #e4e7ed;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
  min-height: 320px;
  flex: 1;
}

.scripts-list.is-full {
  flex: 1;
}

.list-menu {
  border-right: 0;
}

.editor-body {
  display: flex;
  gap: 12px;
  flex: 1;
  align-items: stretch;
  height: 520px;
}

.editor-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 16px;
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

.meta-inline {
  display: flex;
  align-items: center;
  gap: 8px;
}

.meta-label {
  font-size: 12px;
  color: #909399;
}

.meta-value {
  font-size: 12px;
  color: #303133;
}

.meta-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
}

.icon-button {
  background: #eef2ff;
  border: none;
  color: #4f46e5;
}

.icon-button:hover {
  background: #e0e7ff;
}

.editor-main {
  flex: 1;
  min-width: 0;
}

.editor-sidebar {
  width: 200px;
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

.node-icon {
  color: #94a3b8;
  flex-shrink: 0;
}

.node-group .node-icon {
  color: #3b82f6;
}

.node-item .node-icon {
  color: #10b981;
}

.node-item .node-icon.icon-timer {
  color: #f59e0b;
}

.node-item .node-icon.icon-change {
  color: #8b5cf6;
}

.node-item .node-icon.icon-custom {
  color: #10b981;
}

.node-variable .node-icon.icon-variable {
  color: #0ea5e9;
}

.node-page .node-icon.icon-page {
  color: #6366f1;
}

.list-menu .node-icon.icon-system {
  color: #6366f1;
  margin-right: 6px;
}

.tree-node:hover {
  background: #f5f7fa;
}

.tree-node.is-selected {
  background: #e8f3ff;
  color: #303133;
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

.enum-body {
  margin-top: 8px;
  max-height: 420px;
  overflow: auto;
}

.enum-layout {
  display: flex;
  gap: 12px;
}

.enum-left {
  width: 200px;
  border-right: 1px solid #e4e7ed;
  padding-right: 8px;
  max-height: 420px;
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

:deep(.scripts-list .el-tree) {
  flex: 1;
  overflow: auto;
  padding: 10px 8px;
}

:deep(.scripts-list .el-tree-node__content) {
  height: 38px;
}

.context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.12);
  z-index: 4000;
  min-width: 160px;
  padding: 6px 0;
}

.context-menu-item {
  padding: 10px 18px;
  cursor: pointer;
  font-size: 14px;
  color: #606266;
  white-space: nowrap;
}

.context-menu-item:hover {
  background-color: #f5f7fa;
}

.context-menu-item--danger {
  color: #f56c6c;
}

.context-menu-item--danger:hover {
  background-color: #fef0f0;
}

.context-submenu {
  max-height: 300px;
  overflow-y: auto;
}

.context-menu-item.is-disabled {
  color: #c0c4cc;
  pointer-events: none;
}
</style>
