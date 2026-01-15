<template>
  <div class="event-panel">
    <template v-if="panelState === 'page'">
      <div class="empty-hint">请选择一个按钮组件以配置点击事件</div>
    </template>
    <template v-else-if="!selectedNode">
      <div class="empty-hint">请选择一个按钮组件以配置点击事件</div>
    </template>
    <template v-else-if="selectedNode.type !== 'Button'">
      <div class="empty-hint">当前仅支持按钮点击事件</div>
    </template>
    <template v-else>
      <div class="section-title">按钮点击事件</div>
      <div class="event-controls">
        <div class="event-toggle">
          <span class="event-label">启动</span>
          <el-switch v-model="eventEnabled" />
        </div>
        <el-tooltip content="打开编辑器" placement="top">
          <el-button class="icon-button" size="small" circle @click="openEditor">
            <IconEpEditPen />
          </el-button>
        </el-tooltip>
      </div>
      <div class="hint">
        预览模式点击按钮可执行脚本（支持 $event、$global、customScripts）。
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
      <div class="meta-desc">按钮点击脚本</div>
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
          <div class="sidebar-title">工程变量</div>
          <el-input
            v-model="variableSearch"
            size="small"
            placeholder="搜索变量/分组"
            clearable
          />
          <div class="sidebar-scroll">
            <el-tree
              ref="variableTreeRef"
              :data="variableTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterSidebarNode"
              @node-click="handleVariableInsert"
            >
              <template #default="{ data }">
                <div class="tree-node" :class="`node-${data.type}`">
                  <el-icon class="node-icon icon-variable">
                    <IconEpFolder v-if="data.type === 'group'" />
                    <IconEpLink v-else-if="data.mapped" />
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
      </div>
    </div>
    <template #footer>
      <el-button @click="editorVisible = false">取消</el-button>
      <el-button type="primary" @click="saveScript">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch, computed } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { usePanelState } from "./use-panel-state";
import MonacoEditor from "@/components/common/MonacoEditor.vue";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";
import IconEpLink from "~icons/ep/link";

const editorStore = useEditorStore();
const { panelState, selectedNode } = usePanelState();
const { projectVariables, projectVariableGroups, globalScripts } = storeToRefs(editorStore);

const scriptCode = ref("");
const eventEnabled = ref(true);
const editorVisible = ref(false);
const editorRef = ref(null);
const variableTreeRef = ref(null);
const customTreeRef = ref(null);
const variableSearch = ref("");
const scriptSearch = ref("");

const currentScript = computed(() => {
  if (!selectedNode.value?.events?.click) return "";
  const handler = selectedNode.value.events.click[0];
  if (typeof handler === "string") return handler;
  return handler?.code || "";
});

watch(
  () => selectedNode.value?.id,
  () => {
    scriptCode.value = currentScript.value || "";
    const handler = selectedNode.value?.events?.click?.[0];
    eventEnabled.value = handler?.enabled !== false;
  },
  { immediate: true }
);

watch(eventEnabled, () => {
  if (!selectedNode.value) return;
  const handler = selectedNode.value.events?.click?.[0];
  if (!handler) return;
  const next = { ...handler, enabled: eventEnabled.value !== false };
  const nextEvents = { ...(selectedNode.value.events || {}) };
  nextEvents.click = [next];
  editorStore.updateNode(selectedNode.value.id, { events: nextEvents });
});

const openEditor = () => {
  editorVisible.value = true;
};

const saveScript = () => {
  if (!selectedNode.value) return;
  const code = scriptCode.value || "";
  const nextEvents = { ...(selectedNode.value.events || {}) };
  if (!code.trim()) {
    delete nextEvents.click;
  } else {
    nextEvents.click = [{ type: "script", code, enabled: eventEnabled.value !== false }];
  }
  editorStore.updateNode(selectedNode.value.id, { events: nextEvents });
  void editorStore.saveCurrentPage?.();
  editorVisible.value = false;
};

const editorTitle = computed(() => {
  const label = selectedNode.value?.label || selectedNode.value?.id || "按钮";
  return label;
});

const variableTree = computed(() => {
  const groups = projectVariableGroups.value || [];
  const variables = projectVariables.value || {};
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

  Object.entries(variables).forEach(([name, detail]) => {
    const node = {
      id: `var:${name}`,
      label: name,
      type: "variable",
      mapped: detail?.source?.type === "dataCenter" || detail?.mapped === true,
    };
    const groupIdValue = detail?.groupId;
    if (groupIdValue && groupMap.has(groupIdValue)) {
      groupMap.get(groupIdValue).children.push(node);
    } else {
      roots.push(node);
    }
  });

  return roots;
});

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

const jsCompletions = computed(() => {
  const items = [
    { label: "console.log", insertText: "console.log()", kind: "Function", detail: "Log output" },
    { label: "if", insertText: "if () {\n  \n}", kind: "Snippet", detail: "if statement" },
    { label: "for", insertText: "for (let i = 0; i < ; i++) {\n  \n}", kind: "Snippet" },
    { label: "function", insertText: "function name() {\n  \n}", kind: "Snippet" },
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

  return items;
});

const filterSidebarNode = (value, data) => {
  if (!value) return true;
  return String(data?.label || "").toLowerCase().includes(value.toLowerCase());
};

watch(variableSearch, (value) => {
  variableTreeRef.value?.filter?.(value);
});

watch(scriptSearch, (value) => {
  customTreeRef.value?.filter?.(value);
});

const handleVariableInsert = (data) => {
  if (data?.type === "group") return;
  editorRef.value?.insertText?.(`$global.${data.label}`);
};

const handleCustomScriptInsert = (data) => {
  if (data?.type === "group") return;
  const params = data?.params ? data.params : "";
  const call = params ? `customScripts.${data.label}(${params})` : `customScripts.${data.label}()`;
  editorRef.value?.insertText?.(call);
};
</script>

<style scoped>
.event-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-regular);
}

.event-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.event-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.event-label {
  font-size: 12px;
  color: var(--el-text-color-regular);
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

.hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.empty-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  text-align: center;
  padding: 16px 0;
}
</style>
