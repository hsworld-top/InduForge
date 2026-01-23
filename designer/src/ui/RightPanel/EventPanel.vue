<template>
  <div class="event-panel">
    <template v-if="isPageContext">
      <BindingPanel :force-show="true" />
    </template>
    <template v-else-if="eventDefinitions.length === 0">
      <div class="empty-hint">当前组件暂无可配置事件</div>
    </template>
    <template v-else>
      <div class="event-list">
        <div v-for="eventItem in eventDefinitions" :key="eventItem.name" class="event-item">
          <div class="event-row">
            <div class="section-title">{{ getEventTitle(eventItem) }}</div>
            <div class="event-controls">
              <div class="event-toggle">
                <span class="event-label">启用</span>
                <el-switch
                  :model-value="getEventEnabled(eventItem.name)"
                  @change="(value) => handleToggleEvent(eventItem.name, value)"
                />
              </div>
              <el-tooltip content="打开编辑器" placement="top">
                <el-button class="icon-button" size="small" circle @click="openEditor(eventItem)">
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
          <el-input
            v-model="scriptSearch"
            size="small"
            placeholder="搜索脚本/分组"
            clearable
          />
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
          <el-input
            v-model="componentSearch"
            size="small"
            placeholder="搜索组件/分组"
            clearable
          />
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
            <el-input v-model="projectVarSearch" size="small" placeholder="搜索工程变量" clearable />
            <el-table
              :data="projectVariableRows"
              size="small"
              height="320"
              highlight-current-row
              @row-click="handleProjectRowClick"
              @row-dblclick="handleProjectRowDblClick"
              :row-class-name="enumProjectRowClass"
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
              @row-click="handlePageRowClick"
              @row-dblclick="handlePageRowDblClick"
              :row-class-name="enumPageRowClass"
            >
              <el-table-column prop="name" label="变量名" min-width="160" />
              <el-table-column prop="type" label="类型" width="90" />
              <el-table-column prop="description" label="描述" min-width="200" />
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
</template>

<script setup>
import { ref, watch, computed } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { buildComponentMethodCompletions } from "@/ui/utils/component-methods";
import { usePanelState } from "./use-panel-state";
import { componentRegistry } from "@/editor-core";
import { normalizeEventDefinitions } from "@/editor-core/registry/componentEvents.js";
import MonacoEditor from "@/components/common/MonacoEditor.vue";
import BindingPanel from "./BindingPanel.vue";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";
import IconEpList from "~icons/ep/list";
import IconEpGrid from "~icons/ep/grid";

const editorStore = useEditorStore();
const { panelState, selectedNode, selectedGraphic } = usePanelState();
const {
  projectVariables,
  projectVariableGroups,
  globalScripts,
  currentPage,
  doc,
  docVersion,
} = storeToRefs(editorStore);

const scriptCode = ref("");
const editorVisible = ref(false);
const editorRef = ref(null);
const customTreeRef = ref(null);
const scriptSearch = ref("");
const componentSearch = ref("");
const componentTreeRef = ref(null);
const variableEnumVisible = ref(false);
const enumProjectTreeRef = ref(null);
const enumPageTreeRef = ref(null);
const enumTab = ref("project");
const projectVarSearch = ref("");
const pageVarSearch = ref("");
const enumSelectedProjectGroupId = ref(null);
const enumSelectedPageGroupId = ref("page-root");
const enumSelectedProjectVar = ref(null);
const enumSelectedPageVar = ref(null);
const activeEventName = ref("");
const eventToggleState = ref({});

const componentManifest = computed(() => {
  if (!selectedNode.value) return null;
  return componentRegistry.get(selectedNode.value.type) || null;
});

const componentLabel = computed(() => {
  return selectedNode.value?.label || componentManifest.value?.name || "组件";
});

const eventDefinitions = computed(() => {
  const events = componentManifest.value?.events || [];
  return normalizeEventDefinitions(events);
});

const isPageContext = computed(() => {
  if (panelState.value === "page") return true;
  if (selectedGraphic.value) return true;
  if (!selectedNode.value) return true;
  const pageRootId = currentPage.value?.rootNodeId;
  return Boolean(pageRootId && selectedNode.value.id === pageRootId);
});

/**
 * 获取事件处理器
 * @param {string} eventName - 事件名称
 * @returns {Object | string | null} 事件处理器
 */
const getEventHandler = (eventName) => {
  if (!selectedNode.value || !eventName) return null;
  const handlers = selectedNode.value.events?.[eventName];
  if (!Array.isArray(handlers) || handlers.length === 0) return null;
  return handlers[0] || null;
};

/**
 * 获取事件脚本内容
 * @param {string} eventName - 事件名称
 * @returns {string} 事件脚本
 */
const getEventScript = (eventName) => {
  const handler = getEventHandler(eventName);
  if (!handler) return "";
  if (typeof handler === "string") return handler;
  return handler?.code || "";
};

/**
 * 同步事件开关状态
 * @returns {void}
 */
const syncEventToggleState = () => {
  const nextState = {};
  eventDefinitions.value.forEach((eventItem) => {
    const handler = getEventHandler(eventItem.name);
    nextState[eventItem.name] = handler?.enabled !== false;
  });
  eventToggleState.value = nextState;
};

/**
 * 获取事件启用状态
 * @param {string} eventName - 事件名称
 * @returns {boolean} 是否启用
 */
const getEventEnabled = (eventName) => {
  if (!eventName) return true;
  return eventToggleState.value[eventName] !== false;
};

/**
 * 处理事件开关切换
 * @param {string} eventName - 事件名称
 * @param {boolean} enabled - 是否启用
 * @returns {void}
 */
const handleToggleEvent = (eventName, enabled) => {
  if (!eventName) return;
  eventToggleState.value = {
    ...eventToggleState.value,
    [eventName]: enabled !== false,
  };
  if (!selectedNode.value) return;
  const handler = getEventHandler(eventName);
  if (!handler) return;
  const nextHandler =
    typeof handler === "string"
      ? { type: "script", code: handler, enabled: enabled !== false }
      : { ...handler, enabled: enabled !== false };
  const nextEvents = { ...(selectedNode.value.events || {}) };
  nextEvents[eventName] = [nextHandler];
  editorStore.updateNode(selectedNode.value.id, { events: nextEvents });
};

/**
 * 打开脚本编辑器
 * @param {{ name: string }} eventItem - 事件定义
 * @returns {void}
 */
const openEditor = (eventItem) => {
  activeEventName.value = eventItem?.name || "";
  scriptCode.value = getEventScript(activeEventName.value) || "";
  editorVisible.value = true;
};

/**
 * 保存脚本配置
 * @returns {void}
 */
const saveScript = () => {
  if (!selectedNode.value || !activeEventName.value) return;
  const code = scriptCode.value || "";
  const nextEvents = { ...(selectedNode.value.events || {}) };
  if (!code.trim()) {
    delete nextEvents[activeEventName.value];
  } else {
    nextEvents[activeEventName.value] = [
      {
        type: "script",
        code,
        enabled: getEventEnabled(activeEventName.value),
      },
    ];
  }
  editorStore.updateNode(selectedNode.value.id, { events: nextEvents });
  void editorStore.saveCurrentPage?.();
  editorVisible.value = false;
};

/**
 * 获取事件标题
 * @param {{ name: string, label?: string }} eventItem - 事件定义
 * @returns {string} 标题文本
 */
const getEventTitle = (eventItem) => {
  return eventItem?.name || "";
};

watch(
  () => [selectedNode.value?.id, eventDefinitions.value.length],
  () => {
    syncEventToggleState();
  },
  { immediate: true }
);

watch(
  () => [selectedNode.value?.id, activeEventName.value],
  () => {
    if (!activeEventName.value) return;
    scriptCode.value = getEventScript(activeEventName.value) || "";
  },
  { immediate: true }
);

const editorTitle = computed(() => componentLabel.value);

const editorDescription = computed(() => {
  const eventItem = eventDefinitions.value.find(
    (item) => item.name === activeEventName.value
  );
  const label = eventItem?.label || activeEventName.value || "事件";
  return `${componentLabel.value}${label}脚本`;
});

const projectGroupTree = computed(() => [
  { id: "all", label: "全部", type: "group", children: buildGroupTree(projectVariableGroups.value || []) },
]);

const customScriptTree = computed(() => {
  const groups = globalScripts.value?.custom?.groups || [];
  const items = globalScripts.value?.custom?.items || [];
  const groupMap = new Map();
  const roots = [];

  groups.forEach((group) => {
    groupMap.set(group.id, { id: group.id, label: group.name, type: "group", children: [] });
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

const pageVars = computed(() => {
  docVersion.value;
  const pageId = currentPage.value?.id;
  if (!pageId || !doc.value) return {};
  const vars = doc.value.vars?.pages?.[pageId];
  return vars && typeof vars === "object" ? vars : {};
});

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
      return String(item.name || "").toLowerCase().includes(keyword);
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
      description: detail?.description || "",
    }))
    .filter((item) => {
      if (!enumSelectedPageGroupId.value) return true;
      return true;
    })
    .filter((item) => {
      if (!keyword) return true;
      return String(item.name || "").toLowerCase().includes(keyword);
    });
});

const pageComponentTree = computed(() => {
  docVersion.value;
  const rootId = currentPage.value?.rootNodeId;
  if (!rootId || !doc.value) return [];
  const buildNode = (nodeId) => {
    const node = doc.value.getNode(nodeId);
    if (!node) return null;
    const children = (node.children || [])
      .map((childId) => buildNode(childId))
      .filter(Boolean);
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
    { label: "console.log", insertText: "console.log()", kind: "Function", detail: "Log output" },
    { label: "if", insertText: "if () {\\n  \\n}", kind: "Snippet", detail: "if statement" },
    { label: "for", insertText: "for (let i = 0; i < ; i++) {\\n  \\n}", kind: "Snippet" },
    { label: "function", insertText: "function name() {\\n  \\n}", kind: "Snippet" },
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

const filterSidebarNode = (value, data) => {
  if (!value) return true;
  return String(data?.label || "").toLowerCase().includes(value.toLowerCase());
};

const buildGroupTree = (groups) => {
  const groupMap = new Map();
  const roots = [];
  const normalized = Array.isArray(groups) ? groups : [];
  normalized.forEach((group) => {
    groupMap.set(group.id, { id: group.id, label: group.name, type: "group", children: [] });
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
};

watch(scriptSearch, (value) => {
  customTreeRef.value?.filter?.(value);
});

watch(componentSearch, (value) => {
  componentTreeRef.value?.filter?.(value);
});

const handleCustomScriptInsert = (data) => {
  if (data?.type === "group") return;
  const params = data?.params ? data.params : "";
  const call = params ? `customScripts.${data.label}(${params})` : `customScripts.${data.label}()`;
  editorRef.value?.insertText?.(call);
};

const handleComponentInsert = (data) => {
  if (!data?.componentName) return;
  editorRef.value?.insertText?.(`components.${data.componentName}`);
};

const openVariableEnum = () => {
  projectVarSearch.value = "";
  pageVarSearch.value = "";
  enumTab.value = "project";
  enumSelectedProjectGroupId.value = null;
  enumSelectedPageGroupId.value = "page-root";
  enumSelectedProjectVar.value = null;
  enumSelectedPageVar.value = null;
  variableEnumVisible.value = true;
};

const handleProjectGroupSelect = (data) => {
  if (!data) {
    enumSelectedProjectGroupId.value = null;
    return;
  }
  enumSelectedProjectGroupId.value = data.id === "all" ? null : data.id;
};

const handlePageGroupSelect = (data) => {
  enumSelectedPageGroupId.value = data?.id || "page-root";
};

const handleProjectRowClick = (row) => {
  enumSelectedProjectVar.value = row || null;
};

const handlePageRowClick = (row) => {
  enumSelectedPageVar.value = row || null;
};

const handleProjectRowDblClick = (row) => {
  enumSelectedProjectVar.value = row || null;
  confirmEnumInsert();
};

const handlePageRowDblClick = (row) => {
  enumSelectedPageVar.value = row || null;
  confirmEnumInsert();
};

const enumProjectRowClass = ({ row }) => {
  if (enumSelectedProjectVar.value?.name === row.name) return "is-selected";
  return "";
};

const enumPageRowClass = ({ row }) => {
  if (enumSelectedPageVar.value?.name === row.name) return "is-selected";
  return "";
};

const confirmEnumInsert = () => {
  if (enumTab.value === "page" && enumSelectedPageVar.value?.name) {
    editorRef.value?.insertText?.(`$vars.${enumSelectedPageVar.value.name}`);
    variableEnumVisible.value = false;
    return;
  }
  if (enumSelectedProjectVar.value?.name) {
    editorRef.value?.insertText?.(`$global.${enumSelectedProjectVar.value.name}`);
    variableEnumVisible.value = false;
  }
};
</script>

<style scoped>
.event-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.event-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.event-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 6px 8px;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
  background: #ffffff;
}

.event-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-regular);
  min-width: 120px;
}

.event-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.event-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.event-label {
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

.icon-button {
  background: #eef2ff;
  border: none;
  color: #4f46e5;
}

.icon-button:hover {
  background: #e0e7ff;
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

.node-component .node-icon.icon-component {
  color: #6366f1;
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

.empty-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  text-align: center;
  padding: 16px 0;
}

.enum-body {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 420px;
  overflow: auto;
}

.enum-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
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
