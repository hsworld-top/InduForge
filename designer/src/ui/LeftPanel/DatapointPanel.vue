
<template>
  <div class="global-vars">
    <div class="toolbar">
      <el-button
        class="toolbar-button toolbar-button--ghost"
        size="small"
        @click="openQuickAdd"
      >
        <IconEpLink class="toolbar-icon" />
        快速添加数据中心变量
      </el-button>
      <el-dropdown @command="handleExport">
        <el-button class="toolbar-button" size="small">
          <IconEpUpload class="toolbar-icon" />
          导出变量
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="csv">导出 CSV</el-dropdown-item>
            <el-dropdown-item command="xlsx">导出 XLSX</el-dropdown-item>
            <el-dropdown-item command="json">导出 JSON</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-dropdown @command="handleImport">
        <el-button class="toolbar-button" size="small">
          <IconEpDownload class="toolbar-icon" />
          导入变量
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
    <div class="tree-wrap" @contextmenu="handleBlankContextMenu">
      <el-tree
        ref="treeRef"
        :data="variableTree"
        node-key="id"
        :default-expand-all="true"
        highlight-current
        :expand-on-click-node="false"
        draggable
        :allow-drop="allowDrop"
        :allow-drag="allowDrag"
        @node-contextmenu="handleContextMenu"
        @node-dblclick="() => contextMenuVisible && closeContextMenu()"
        @node-drop="handleNodeDrop"
      >
        <template #default="{ data }">
          <div
            class="tree-node"
            :class="[{ 'is-selected': isNodeSelected(data) }, `node-${data.type}`]"
            @click.stop="(event) => handleNodeClick(data, event)"
          >
            <el-icon
              class="node-icon"
              :class="{
                'is-mapped': data.type === 'variable' && data.meta?.mapped,
                'is-unmapped': data.type === 'variable' && !data.meta?.mapped,
              }"
            >
              <IconEpFolder v-if="data.type === 'group'" />
              <IconEpLink v-else-if="data.meta?.mapped" />
              <IconEpEditPen v-else />
            </el-icon>
            <span class="node-label" :class="{ 'is-group': data.type === 'group' }">
              {{ data.label }}
            </span>
            <span v-if="data.type === 'variable'" class="node-meta">
              {{ data.meta?.type || "string" }}
            </span>
          </div>
        </template>
      </el-tree>
      <div v-if="!variableTree.length" class="tree-empty">
        <el-empty description="暂无工程变量" :image-size="60" />
      </div>
    </div>
    <div
      v-if="contextMenuVisible"
      class="context-menu"
      :style="contextMenuStyle"
      @click.stop
      @mousedown.stop
    >
      <template v-if="contextMenuNode?.type === 'blank'">
        <div class="context-menu-item" @click="openGroupCreateFromMenu">新建分组</div>
        <div class="context-menu-item" @click="openCreateFromMenu">新增变量</div>
        <div class="context-menu-item" @click="openQuickAddFromMenu">快速添加</div>
        <div
          class="context-menu-item"
          :class="{ 'is-disabled': !varClipboard }"
          @click="pasteVarFromMenu"
        >
          粘贴
        </div>
      </template>
      <template v-if="contextMenuNode?.type === 'variable'">
        <div
          class="context-menu-item"
          :class="{ 'is-disabled': !canEditSelection }"
          @click="openEditFromMenu"
        >
          编辑变量
        </div>
        <div class="context-menu-item" @click="copyVar">复制</div>
        <div class="context-menu-item" @click="showMoveToMenu = !showMoveToMenu">
          移动到
        </div>
        <div
          class="context-menu-item context-menu-item--danger"
          :class="{ 'is-disabled': !canDeleteSelection }"
          @click="removeVar"
        >
          删除
        </div>
      </template>
      <template v-else-if="contextMenuNode?.type === 'group'">
        <div
          class="context-menu-item"
          :class="{ 'is-disabled': !canEditSelection }"
          @click="openGroupEditFromMenu"
        >
          编辑分组
        </div>
        <div class="context-menu-item" @click="openGroupCreateFromMenu">
          新建子分组
        </div>
        <div class="context-menu-item" @click="showMoveToMenu = !showMoveToMenu">
          移动到
        </div>
        <div
          class="context-menu-item context-menu-item--danger"
          :class="{ 'is-disabled': !canDeleteSelection }"
          @click="removeGroup"
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

    <el-dialog
      v-model="editVisible"
      :title="editMode ? '编辑变量' : '新增变量'"
      width="520px"
      :close-on-click-modal="false"
      :lock-scroll="false"
    >
      <el-form label-width="80px">
        <el-form-item label="变量名">
          <el-input v-model="editName" />
        </el-form-item>
        <el-form-item label="分组">
          <el-select v-model="editGroupId" placeholder="请选择分组">
            <el-option label="根目录" :value="ROOT_GROUP_ID" />
            <el-option
              v-for="group in groupOptions"
              :key="group.id"
              :label="group.name"
              :value="group.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="editType" @change="resetEditValue">
            <el-option v-for="t in types" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="初始值">
          <div v-if="isEditorType" class="edit-value-block">
            <MonacoEditor
              ref="editValueEditorRef"
              v-model="editValue"
              :language="editorLanguage"
              height="220px"
              @markers="handleEditValueMarkers"
            />
          </div>
          <el-input v-else-if="isTextType" v-model="editValue" type="textarea" :rows="6" />
          <el-input-number
            v-else-if="editType === 'number'"
            v-model="editValue"
            style="width: 100%"
          />
          <el-switch v-else-if="editType === 'boolean'" v-model="editValue" />
          <el-date-picker
            v-else-if="editType === 'date'"
            v-model="editValue"
            type="datetime"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="editDescription" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="映射">
          <el-switch v-model="mapped" />
        </el-form-item>
        <template v-if="mapped">
          <el-form-item label="数据源">
            <el-select v-model="mappedDs">
              <el-option v-for="ds in dataSources" :key="ds.id" :label="ds.name" :value="ds" />
            </el-select>
          </el-form-item>
          <el-form-item label="字段">
            <el-input v-model="mappedField" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" @click="saveEdit">确定</el-button>
      </template>
    </el-dialog>
    <el-dialog
      v-model="groupVisible"
      :title="groupEditMode ? '编辑分组' : '新建分组'"
      width="420px"
      :close-on-click-modal="false"
      :lock-scroll="false"
    >
      <el-form label-width="70px">
        <el-form-item label="名称">
          <el-input v-model="groupName" />
        </el-form-item>
        <el-form-item label="父级">
          <el-select v-model="groupParentId" placeholder="根目录">
            <el-option label="根目录" :value="null" />
            <el-option
              v-for="group in groupParentOptions"
              :key="group.id"
              :label="group.name"
              :value="group.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="groupVisible = false">取消</el-button>
        <el-button type="primary" @click="saveGroup">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="quickVisible"
      title="快速添加数据源变量"
      width="900px"
      :close-on-click-modal="false"
      :lock-scroll="false"
    >
      <el-form :inline="true" class="quick-form" label-width="60px" size="small">
        <el-row :gutter="12" class="quick-form-row">
          <el-col :span="6">
            <el-form-item label="数据源">
              <el-select v-model="quickDs" placeholder="请选择数据源" @change="loadFields">
                <el-option v-for="ds in dataSources" :key="ds.id" :label="ds.name" :value="ds" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="搜索">
              <el-input v-model="searchKey" placeholder="字段名搜索" />
            </el-form-item>
          </el-col>
          <el-col :span="3">
            <el-form-item label="前缀">
              <el-input v-model="prefix" placeholder="前缀" />
            </el-form-item>
          </el-col>
          <el-col :span="3">
            <el-form-item label="后缀">
              <el-input v-model="suffix" placeholder="后缀" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="替换" class="quick-replace">
              <el-input v-model="replaceFrom" placeholder="替换" />
              <span class="quick-arrow">→</span>
              <el-input v-model="replaceTo" placeholder="为" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <el-table :data="filteredFields" border height="420" @selection-change="onSelectFields">
        <el-table-column type="selection" width="50" />
        <el-table-column prop="name" label="字段名" sortable width="150" />
        <el-table-column prop="type" label="类型" width="120" sortable />
        <el-table-column label="变量名">
          <template #default="{ row }">
            {{ buildVarName(row.name) }}
          </template>
        </el-table-column>
        <el-table-column label="映射">
          <template #default="{ row }">
            {{ buildMappedExpression(row.name) }}
          </template>
        </el-table-column>
      </el-table>

      <template #footer>
        <el-button @click="quickVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmQuickAdd">添加</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, onUnmounted } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { datacenterApi } from "@/services";
import MonacoEditor from "@/components/common/MonacoEditor.vue";
import * as XLSX from "xlsx";
import IconEpFolder from "~icons/ep/folder";
import IconEpLink from "~icons/ep/link";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpUpload from "~icons/ep/upload";
import IconEpDownload from "~icons/ep/download";

const editorStore = useEditorStore();
const { projectId, projectVariables, projectVariableGroups } = storeToRefs(editorStore);
const maxGroupDepth = 5;

const types = [
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

const unwrapApiData = (payload) => {
  if (payload && typeof payload === "object" && "data" in payload) {
    return payload.data;
  }
  return payload;
};

const treeRef = ref(null);
const selectedNode = ref(null);
const selectedNodes = ref([]);
const varClipboard = ref(null);
const contextMenuVisible = ref(false);
const contextMenuPosition = ref({ x: 0, y: 0 });
const contextMenuNode = ref(null);
const showMoveToMenu = ref(false);

const editVisible = ref(false);
const editMode = ref(false);
const originalName = ref("");
const editName = ref("");
const editType = ref("string");
const editValue = ref("");
const editDescription = ref("");
const editValueEditorRef = ref(null);
const ROOT_GROUP_ID = "__root__";
const editGroupId = ref(ROOT_GROUP_ID);
const mapped = ref(false);
const mappedDs = ref(null);
const mappedField = ref("");
const dataSources = ref([]);
const importInputRef = ref(null);
const importType = ref("json");
const editValueHasErrors = ref(false);

const groupVisible = ref(false);
const groupEditMode = ref(false);
const groupId = ref("");
const groupName = ref("");
const groupParentId = ref(null);

const quickVisible = ref(false);
const quickDs = ref(null);
const fields = ref([]);
const selectedFields = ref([]);
const searchKey = ref("");
const prefix = ref("");
const suffix = ref("");
const replaceFrom = ref("");
const replaceTo = ref("");

const isEditorType = computed(() =>
  ["function", "array", "object", "set", "map"].includes(editType.value)
);
const isStructuredType = computed(() =>
  ["array", "object", "set", "map"].includes(editType.value)
);
const editorLanguage = computed(() =>
  isStructuredType.value ? "json" : "javascript"
);
const isTextType = computed(() =>
  ["string", "regexp"].includes(editType.value)
);

const groupOptions = computed(() => projectVariableGroups.value || []);

const groupParentOptions = computed(() => {
  if (!groupEditMode.value) return groupOptions.value;
  return groupOptions.value.filter(
    (group) => group.id !== groupId.value && !isDescendantGroup(group.id, groupId.value)
  );
});

const selectedVariable = computed(() => {
  if (selectedNode.value?.type !== "variable") return null;
  const name = selectedNode.value?.name;
  const detail = projectVariables.value?.[name];
  if (!detail) return null;
  return { name, detail };
});

const selectedGroup = computed(() => {
  if (selectedNode.value?.type !== "group") return null;
  return (
    projectVariableGroups.value?.find((group) => group.id === selectedNode.value.id) || null
  );
});

const selectedGroupId = computed(() => {
  const groupNode = selectedNodes.value.find((node) => node.type === "group");
  if (groupNode?.id) return groupNode.id;
  if (selectedGroup.value?.id) return selectedGroup.value.id;
  if (selectedVariable.value?.detail?.groupId) return selectedVariable.value.detail.groupId;
  return null;
});

const variableTree = computed(() =>
  buildTree(projectVariableGroups.value || [], projectVariables.value || {})
);

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
  const current = contextMenuNode.value;
  const groups = projectVariableGroups.value || [];
  if (!current || current.type !== "group") return groups;
  return groups.filter(
    (group) => group.id !== current.id && !isDescendantGroup(group.id, current.id)
  );
});

const canEditSelection = computed(
  () => selectedNodes.value.length === 0 || selectedNodes.value.length === 1
);

const canDeleteSelection = computed(() => {
  if (selectedNodes.value.length <= 1) return true;
  const types = new Set(selectedNodes.value.map((node) => node.type));
  return types.size <= 1;
});

const importAccept = computed(() => {
  if (importType.value === "csv") return ".csv";
  if (importType.value === "xlsx") return ".xlsx,.xls";
  return ".json";
});

function buildTree(groups, variables) {
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
      name,
      meta: {
        ...detail,
        mapped: detail?.source?.type === "dataCenter" || detail?.mapped === true,
      },
    };
    const groupIdValue = detail?.groupId;
    if (groupIdValue && groupMap.has(groupIdValue)) {
      groupMap.get(groupIdValue).children.push(node);
    } else {
      roots.push(node);
    }
  });

  return roots;
}

const isNodeSelected = (data) =>
  selectedNodes.value.some((node) => node.id === data.id);

function handleNodeClick(data, event) {
  const isCtrl = Boolean(event?.ctrlKey || event?.metaKey);
  if (isCtrl) {
    if (isNodeSelected(data)) {
      selectedNodes.value = selectedNodes.value.filter((node) => node.id !== data.id);
    } else {
      selectedNodes.value = [...selectedNodes.value, data];
    }
  } else {
    selectedNodes.value = [data];
  }
  selectedNode.value = data;
  treeRef.value?.setCurrentKey?.(data.id);
  if (contextMenuVisible.value) closeContextMenu();
}

function getMenuItemCount(nodeType) {
  if (nodeType === "blank") return 4;
  if (nodeType === "variable") return 4;
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

function handleContextMenu(event, data) {
  event.preventDefault();
  event.stopPropagation();
  if (!isNodeSelected(data)) {
    selectedNodes.value = [data];
  }
  selectedNode.value = data;
  contextMenuNode.value = data;
  setContextMenuPosition(event, data.type);
  contextMenuVisible.value = true;
  showMoveToMenu.value = false;
}

function handleBlankContextMenu(event) {
  event.preventDefault();
  selectedNode.value = null;
  selectedNodes.value = [];
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

function openCreateFromMenu() {
  closeContextMenu();
  openCreate();
}

function openQuickAddFromMenu() {
  closeContextMenu();
  openQuickAdd();
}

function pasteVarFromMenu() {
  if (!varClipboard.value) return;
  closeContextMenu();
  pasteVar();
}

function openEditFromMenu() {
  if (!canEditSelection.value) return;
  closeContextMenu();
  openEdit();
}

function openGroupEditFromMenu() {
  if (!canEditSelection.value) return;
  closeContextMenu();
  openGroupEdit();
}

function openGroupCreateFromMenu() {
  closeContextMenu();
  openGroupCreate();
}

function handleMoveTo(groupIdValue) {
  if (!contextMenuNode.value) return;
  const node = contextMenuNode.value;
  const nodesToMove = selectedNodes.value.length ? selectedNodes.value : [node];
  closeContextMenu();
  const nextVariables = { ...(projectVariables.value || {}) };
  let nextGroups = projectVariableGroups.value || [];
  let blocked = false;

  nodesToMove.forEach((item) => {
    if (item.type === "variable" && item.name) {
      nextVariables[item.name] = {
        ...nextVariables[item.name],
        groupId: groupIdValue,
      };
    } else if (item.type === "group") {
      if (groupIdValue && isDescendantGroup(groupIdValue, item.id)) {
        blocked = true;
        return;
      }
      nextGroups = nextGroups.map((group) =>
        group.id === item.id ? { ...group, parentId: groupIdValue } : group
      );
    }
  });

  if (blocked) {
    ElMessage.warning("无法移动到子分组");
  }

  projectVariables.value = nextVariables;
  projectVariableGroups.value = nextGroups;
  persistProjectGlobals();
}

function allowDrag() {
  return true;
}

function allowDrop(draggingNode, dropNode, type) {
  const dragData = draggingNode.data;
  const dropData = dropNode.data;

  if (dragData.type === "variable") {
    if (type === "inner" && dropData.type !== "group") return false;
    return true;
  }

  if (dragData.type === "group") {
    if (type === "inner" && dropData.type !== "group") return false;
    if (dropData.type === "group" && isDescendantGroup(dropData.id, dragData.id)) return false;
    if (type === "inner") {
      const depth = getGroupDepth(dropData.id) + getGroupSubtreeDepth(dragData.id);
      return depth <= maxGroupDepth;
    }
    return true;
  }

  return false;
}

function handleNodeDrop(draggingNode, dropNode, dropType) {
  const dragData = draggingNode.data;
  const targetGroupId = resolveTargetGroupId(dropNode, dropType);

  if (dragData.type === "variable") {
    if (!dragData.name) return;
    const nextVariables = { ...(projectVariables.value || {}) };
    nextVariables[dragData.name] = {
      ...nextVariables[dragData.name],
      groupId: targetGroupId,
    };
    projectVariables.value = nextVariables;
    persistProjectGlobals();
    return;
  }

  if (dragData.type === "group") {
    const groups = projectVariableGroups.value || [];
    projectVariableGroups.value = groups.map((group) =>
      group.id === dragData.id ? { ...group, parentId: targetGroupId } : group
    );
    persistProjectGlobals();
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

function getGroupDepth(groupIdValue) {
  if (!groupIdValue) return 0;
  let depth = 1;
  let currentId = groupIdValue;
  const groups = projectVariableGroups.value || [];
  const map = new Map(groups.map((group) => [group.id, group]));
  while (map.get(currentId)?.parentId) {
    depth += 1;
    currentId = map.get(currentId).parentId;
  }
  return depth;
}

function getGroupSubtreeDepth(groupIdValue) {
  const groups = projectVariableGroups.value || [];
  const children = groups.filter((group) => group.parentId === groupIdValue);
  if (!children.length) return 1;
  const depths = children.map((child) => getGroupSubtreeDepth(child.id));
  return 1 + Math.max(...depths);
}

function isDescendantGroup(targetId, parentId) {
  const groups = projectVariableGroups.value || [];
  let current = groups.find((group) => group.id === targetId);
  while (current?.parentId) {
    if (current.parentId === parentId) return true;
    current = groups.find((group) => group.id === current.parentId);
  }
  return false;
}

function defaultEditValue(type) {
  switch (type) {
    case "string":
      return "";
    case "number":
      return 0;
    case "boolean":
      return false;
    case "array":
      return "[]";
    case "object":
      return "{}";
    case "set":
      return "[]";
    case "map":
      return "[]";
    case "date":
      return null;
    case "regexp":
      return "/pattern/g";
    case "function":
      return "function(){}";
    default:
      return "";
  }
}

function parseEditValue(type, value) {
  if (type === "number") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  if (type === "boolean") {
    return Boolean(value);
  }
  if (type === "date") {
    return value || null;
  }
  if (["array", "object", "set", "map"].includes(type)) {
    if (value instanceof Set) return Array.from(value);
    if (value instanceof Map) return Array.from(value.entries());
    if (Array.isArray(value)) return value;
    if (value && typeof value === "object") {
      if (type === "map") return Object.entries(value);
      if (type === "set") return Object.values(value);
      if (type === "object") return value;
    }
    if (value && typeof value === "string") {
      try {
        const parsed = JSON.parse(value);
        if (type === "array") return Array.isArray(parsed) ? parsed : [];
        if (type === "set") {
          if (Array.isArray(parsed)) return parsed;
          if (parsed && typeof parsed === "object") return Object.values(parsed);
          return [];
        }
        if (type === "map") {
          if (Array.isArray(parsed)) return parsed;
          if (parsed && typeof parsed === "object") return Object.entries(parsed);
          return [];
        }
        if (parsed && typeof parsed === "object") return parsed;
      } catch (error) {
        if (type === "array" || type === "set" || type === "map") return [];
        return {};
      }
    }
    if (type === "array" || type === "set" || type === "map") return [];
    return {};
  }
  return value ?? "";
}

function resetEditValue() {
  editValue.value = defaultEditValue(editType.value);
  editValueHasErrors.value = false;
}

const parseStructuredJson = (value, type) => {
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
    if (type === "set" || type === "map") {
      if (!Array.isArray(parsed) && (!parsed || typeof parsed !== "object")) {
        return { ok: false, error: "Set/Map 需要 JSON 数组或对象" };
      }
    }
    return { ok: true, parsed };
  } catch (error) {
    return { ok: false, error: "JSON 格式不正确" };
  }
};

const validateStructuredValue = () => {
  if (!isStructuredType.value) return true;
  const result = parseStructuredJson(editValue.value, editType.value);
  if (!result.ok) {
    ElMessage.error(result.error || "校验失败");
    return false;
  }
  return true;
};


const formatValue = (value) => {
  if (value === null || value === undefined) return "";
  if (value instanceof Set) {
    return JSON.stringify(Array.from(value));
  }
  if (value instanceof Map) {
    return JSON.stringify(Array.from(value.entries()));
  }
  if (typeof value === "object") {
    try {
      return JSON.stringify(value);
    } catch (error) {
      return "";
    }
  }
  return String(value);
};

async function loadDataSourcesForMapping() {
  if (!projectId.value) return;
  const result = await datacenterApi.getConnections(projectId.value, {
    page: 1,
    limit: 200,
  });
  const data = unwrapApiData(result) || {};
  dataSources.value = data.connections || data.items || data.list || [];
}

async function persistProjectGlobals() {
  if (!projectId.value) {
    ElMessage.error("缺少工程信息，无法保存");
    return;
  }
  const result = await editorStore.saveProjectSettings();
  if (!result.ok) {
    ElMessage.error(result.error?.message || "保存失败");
  } else {
    ElMessage.success("已保存");
  }
}

async function openCreate() {
  editMode.value = false;
  originalName.value = "";
  editName.value = "";
  editType.value = "string";
  resetEditValue();
  editDescription.value = "";
  editGroupId.value = selectedGroupId.value || ROOT_GROUP_ID;
  mapped.value = false;
  mappedDs.value = null;
  mappedField.value = "";
  await loadDataSourcesForMapping();
  editVisible.value = true;
}

async function openEdit() {
  if (selectedNodes.value.length > 1) {
    ElMessage.warning("多选时不能编辑");
    return;
  }
  if (!selectedVariable.value) return;
  editMode.value = true;
  originalName.value = selectedVariable.value.name;
  editName.value = selectedVariable.value.name;
  editType.value = selectedVariable.value.detail?.type || "string";
  editValue.value = formatValue(selectedVariable.value.detail?.default);
  editDescription.value = selectedVariable.value.detail?.description || "";
  editGroupId.value = selectedVariable.value.detail?.groupId || ROOT_GROUP_ID;
  mapped.value = selectedVariable.value.detail?.source?.type === "dataCenter";
  await loadDataSourcesForMapping();

  if (mapped.value) {
    const path = selectedVariable.value.detail?.source?.path || "";
    const [dsName, ...rest] = String(path).split(".");
    mappedDs.value = dataSources.value.find((ds) => ds.name === dsName) || null;
    mappedField.value = rest.join(".") || path;
  } else {
    mappedDs.value = null;
    mappedField.value = "";
  }
  editVisible.value = true;
}

async function saveEdit() {
  const name = editName.value.trim();
  if (!name) return ElMessage.warning("变量名不能为空");

  const current = projectVariables.value || {};
  if ((!editMode.value || name !== originalName.value) && current[name]) {
    return ElMessage.warning("变量名已存在");
  }

  if (mapped.value && !mappedField.value.trim()) {
    return ElMessage.warning("请输入映射字段");
  }
  if (editValueHasErrors.value) {
    return ElMessage.error("初始值存在语法错误，请先修正");
  }
  if (isStructuredType.value && !validateStructuredValue()) {
    return;
  }

  const nextVariables = { ...(projectVariables.value || {}) };
  if (editMode.value && name !== originalName.value) {
    delete nextVariables[originalName.value];
  }

  const value = parseEditValue(editType.value, editValue.value);
  const groupIdValue = editGroupId.value === ROOT_GROUP_ID ? null : editGroupId.value;
  const next = {
    type: editType.value,
    default: value,
    groupId: groupIdValue,
  };

  if (editDescription.value) {
    next.description = editDescription.value;
  }

  if (mapped.value && (mappedField.value || mappedDs.value)) {
    const sourcePath = mappedDs.value
      ? `${mappedDs.value.name}.${mappedField.value}`
      : mappedField.value;
    next.mapped = true;
    next.source = {
      type: "dataCenter",
      path: sourcePath,
    };
  }

  nextVariables[name] = next;
  projectVariables.value = nextVariables;
  editVisible.value = false;
  await persistProjectGlobals();
  nextTick(() => {
    treeRef.value?.setCurrentKey?.(`var:${name}`);
  });
}

async function removeVar() {
  if (contextMenuVisible.value) closeContextMenu();
  if (!canDeleteSelection.value) {
    ElMessage.warning("分组与成员混选时不能删除");
    return;
  }
  if (!selectedNodes.value.length && !selectedVariable.value) return;
  const variablesToRemove = selectedNodes.value
    .filter((node) => node.type === "variable")
    .map((node) => node.name);
  const groupsToRemove = selectedNodes.value
    .filter((node) => node.type === "group")
    .map((node) => node.id);
  try {
    const count = variablesToRemove.length + groupsToRemove.length;
    let message = `确定删除变量 "${selectedVariable.value?.name || ""}" 吗？`;
    if (count > 1) {
      message = `确定删除选中的 ${count} 项吗？`;
    } else if (groupsToRemove.length === 1 && variablesToRemove.length === 0) {
      const name = selectedNodes.value.find((node) => node.type === "group")?.label || "";
      message = `确定删除分组 "${name}" 吗？分组内成员会移动到父级。`;
    }
    await ElMessageBox.confirm(message, "确认删除", {
      type: "warning",
      lockScroll: false,
    });
  } catch (error) {
    return;
  }
  const nextVariables = { ...(projectVariables.value || {}) };
  variablesToRemove.forEach((name) => {
    delete nextVariables[name];
  });

  if (groupsToRemove.length) {
    const parentMap = new Map();
    (projectVariableGroups.value || []).forEach((group) => {
      if (groupsToRemove.includes(group.id)) {
        parentMap.set(group.id, group.parentId || null);
      }
    });

    const nextGroups = (projectVariableGroups.value || [])
      .filter((group) => !groupsToRemove.includes(group.id))
      .map((group) =>
        groupsToRemove.includes(group.parentId)
          ? { ...group, parentId: parentMap.get(group.parentId) || null }
          : group
      );

    Object.entries(nextVariables).forEach(([name, detail]) => {
      if (groupsToRemove.includes(detail?.groupId)) {
        nextVariables[name] = {
          ...detail,
          groupId: parentMap.get(detail.groupId) || null,
        };
      }
    });

    projectVariableGroups.value = nextGroups;
  }

  projectVariables.value = nextVariables;
  selectedNodes.value = [];
  selectedNode.value = null;
  await persistProjectGlobals();
}

function copyVar() {
  if (contextMenuVisible.value) closeContextMenu();
  const selectedVars = selectedNodes.value
    .filter((node) => node.type === "variable")
    .map((node) => ({ name: node.name, detail: projectVariables.value?.[node.name] }));
  if (!selectedVars.length && selectedVariable.value) {
    selectedVars.push({
      name: selectedVariable.value.name,
      detail: selectedVariable.value.detail,
    });
  }
  if (!selectedVars.length) {
    ElMessage.warning("请选择变量后再复制");
    return;
  }
  varClipboard.value = {
    items: selectedVars.map((item) => ({
      name: item.name,
      detail: JSON.parse(JSON.stringify(item.detail || {})),
    })),
  };
  ElMessage.success(`已复制 ${varClipboard.value.items.length} 个变量`);
}

async function pasteVar() {
  if (contextMenuVisible.value) closeContextMenu();
  if (!varClipboard.value?.items?.length) return;
  const targetGroupId = selectedGroupId.value || null;
  const nextVariables = { ...(projectVariables.value || {}) };
  let lastName = "";

  varClipboard.value.items.forEach((item) => {
    const baseName = item.name || "变量";
    let name = baseName;
    let index = 1;
    while (nextVariables[name]) {
      name = `${baseName}_copy${index}`;
      index += 1;
    }
    nextVariables[name] = {
      ...item.detail,
      groupId: targetGroupId ?? item.detail?.groupId ?? null,
    };
    lastName = name;
  });

  projectVariables.value = nextVariables;
  await persistProjectGlobals();
  if (lastName) {
    nextTick(() => treeRef.value?.setCurrentKey?.(`var:${lastName}`));
  }
}

function openGroupCreate() {
  groupEditMode.value = false;
  groupId.value = "";
  groupName.value = "";
  groupParentId.value = selectedGroup.value?.id || null;
  if (groupParentId.value && getGroupDepth(groupParentId.value) >= maxGroupDepth) {
    ElMessage.warning(`分组最多支持 ${maxGroupDepth} 层`);
    groupParentId.value = null;
  }
  groupVisible.value = true;
}

function openGroupEdit() {
  if (selectedNodes.value.filter((node) => node.type === "group").length > 1) {
    ElMessage.warning("多选时不能编辑");
    return;
  }
  if (!selectedGroup.value) return;
  groupEditMode.value = true;
  groupId.value = selectedGroup.value.id;
  groupName.value = selectedGroup.value.name;
  groupParentId.value = selectedGroup.value.parentId || null;
  groupVisible.value = true;
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
  const depth = parentId ? getGroupDepth(parentId) + 1 : 1;
  if (depth > maxGroupDepth) {
    return ElMessage.warning(`分组最多支持 ${maxGroupDepth} 层`);
  }

  const groups = projectVariableGroups.value || [];
  if (groupEditMode.value) {
    projectVariableGroups.value = groups.map((group) =>
      group.id === groupId.value ? { ...group, name, parentId } : group
    );
  } else {
    projectVariableGroups.value = [
      ...groups,
      { id: createId(), name, parentId, sortOrder: 0 },
    ];
  }
  groupVisible.value = false;
  await persistProjectGlobals();
}

async function removeGroup() {
  await removeVar();
}

async function openQuickAdd() {
  quickVisible.value = true;
  await loadDataSourcesForMapping();
  quickDs.value = null;
  fields.value = [];
  selectedFields.value = [];
}

async function loadFields(ds) {
  quickDs.value = ds;
  if (!ds || !projectId.value) {
    fields.value = [];
    return;
  }

  if (ds.type === "relational") {
    const result = await datacenterApi.getQueries(projectId.value, {
      connectionId: ds.id,
      page: 1,
      limit: 200,
    });
    const data = unwrapApiData(result) || {};
    const queries = data.queries || data.items || data.list || [];
    if (!Array.isArray(queries) || queries.length === 0) {
      ElMessage.warning("未获取到数据查询，请检查数据源或权限");
    }
    fields.value = queries.map((item) => ({
      name: item.name || item.id,
      type: "object",
    }));
    return;
  }

  const result = await datacenterApi.getDataPoints(projectId.value, {
    sourceId: ds.id,
    pageSize: 200,
    page: 1,
  });
  const data = unwrapApiData(result) || {};
  const datapoints = data.datapoints || data.items || data.list || [];
  if (!Array.isArray(datapoints) || datapoints.length === 0) {
    ElMessage.warning("未获取到数据点，请检查数据源或权限");
  }

  fields.value = datapoints.map((item) => ({
    name: item.path || item.name || item.id,
    type: item.dataType || item.type || "string",
  }));
}

const filteredFields = computed(() =>
  fields.value.filter((field) =>
    field.name.toLowerCase().includes(searchKey.value.toLowerCase())
  )
);

function onSelectFields(rows) {
  selectedFields.value = rows;
}

function buildVarName(field) {
  let name = field;
  if (replaceFrom.value) name = name.replace(replaceFrom.value, replaceTo.value);
  return `${prefix.value}${name}${suffix.value}`;
}

function buildMappedExpression(field) {
  if (!quickDs.value) return "";
  return `${quickDs.value.name}.${field}`;
}

async function confirmQuickAdd() {
  if (!selectedFields.value.length || !quickDs.value) {
    quickVisible.value = false;
    return;
  }

  const targetGroupId = selectedGroup.value?.id || null;
  const nextVariables = { ...(projectVariables.value || {}) };

  selectedFields.value.forEach((field) => {
    const name = buildVarName(field.name);
    if (nextVariables[name]) return;

    const type = field.type || "string";
    nextVariables[name] = {
      type,
      default: parseEditValue(type, defaultEditValue(type)),
      mapped: true,
      source: {
        type: "dataCenter",
        path: buildMappedExpression(field.name),
      },
      groupId: targetGroupId,
    };
  });

  projectVariables.value = nextVariables;
  quickVisible.value = false;
  await persistProjectGlobals();
}

function handleClickOutside() {
  if (contextMenuVisible.value) closeContextMenu();
}

const downloadBlob = (content, name, type) => {
  const blob = new Blob([content], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  link.click();
  URL.revokeObjectURL(url);
};

const buildGroupPathMap = (groups) => {
  const map = new Map();
  const groupMap = new Map((groups || []).map((group) => [group.id, group]));

  const buildPath = (groupIdValue) => {
    if (!groupIdValue || !groupMap.has(groupIdValue)) return "";
    if (map.has(groupIdValue)) return map.get(groupIdValue);
    const group = groupMap.get(groupIdValue);
    const parentPath = buildPath(group.parentId);
    const path = parentPath ? `${parentPath} / ${group.name}` : group.name;
    map.set(groupIdValue, path);
    return path;
  };

  (groups || []).forEach((group) => buildPath(group.id));
  return map;
};

const ensureGroupPath = (groups, path) => {
  if (!path) return null;
  const segments = String(path)
    .split("/")
    .map((segment) => segment.trim())
    .filter(Boolean);
  if (!segments.length) return null;
  let parentId = null;
  segments.forEach((segment) => {
    let match = groups.find(
      (group) => group.name === segment && (group.parentId || null) === parentId
    );
    if (!match) {
      match = { id: createId(), name: segment, parentId, sortOrder: 0 };
      groups.push(match);
    }
    parentId = match.id;
  });
  return parentId;
};

const buildExportRows = () => {
  const groupPathMap = buildGroupPathMap(projectVariableGroups.value || []);
  return Object.entries(projectVariables.value || {}).map(([name, detail]) => ({
    name,
    type: detail?.type || "string",
    default: formatValue(detail?.default ?? detail?.value),
    description: detail?.description || "",
    groupPath: detail?.groupId ? groupPathMap.get(detail.groupId) || "" : "",
    mappedPath: detail?.source?.path || "",
  }));
};

const normalizeRowKey = (row, key) => {
  const lowerKey = key.toLowerCase();
  const hit = Object.keys(row).find((k) => k.toLowerCase() === lowerKey);
  return hit ? row[hit] : "";
};

const mergeImportedRows = async (rows) => {
  const nextGroups = [...(projectVariableGroups.value || [])];
  const nextVariables = { ...(projectVariables.value || {}) };
  let added = 0;
  let skipped = 0;

  rows.forEach((row) => {
    const name = String(normalizeRowKey(row, "name") || "").trim();
    if (!name) return;
    if (nextVariables[name]) {
      skipped += 1;
      return;
    }
    const type = String(normalizeRowKey(row, "type") || "string").trim();
    const defaultRaw = normalizeRowKey(row, "default");
    const description = String(normalizeRowKey(row, "description") || "");
    const groupPath = String(normalizeRowKey(row, "groupPath") || "");
    const mappedPath = String(normalizeRowKey(row, "mappedPath") || "");
    const groupIdValue = ensureGroupPath(nextGroups, groupPath);
    const parsedDefault = parseEditValue(type, defaultRaw);
    nextVariables[name] = {
      type,
      default: parsedDefault,
      description,
      groupId: groupIdValue,
    };
    if (mappedPath) {
      nextVariables[name].mapped = true;
      nextVariables[name].source = { type: "dataCenter", path: mappedPath };
    }
    added += 1;
  });

  projectVariableGroups.value = nextGroups;
  projectVariables.value = nextVariables;
  await persistProjectGlobals();
  ElMessage.success(`导入完成，新增 ${added} 项，跳过 ${skipped} 项`);
};

const mergeImportedDefinitions = async (definitions, groups) => {
  const nextGroups = [...(projectVariableGroups.value || [])];
  const nextVariables = { ...(projectVariables.value || {}) };
  const importedGroups = Array.isArray(groups) ? groups : [];
  const importedPathMap = buildGroupPathMap(importedGroups);
  const idToNew = new Map();

  importedGroups.forEach((group) => {
    const path = importedPathMap.get(group.id) || group.name;
    const newId = ensureGroupPath(nextGroups, path);
    idToNew.set(group.id, newId);
  });

  Object.entries(definitions || {}).forEach(([name, detail]) => {
    if (nextVariables[name]) return;
    const groupPath = detail?.groupId ? importedPathMap.get(detail.groupId) || "" : "";
    const groupIdValue = ensureGroupPath(nextGroups, groupPath);
    nextVariables[name] = {
      ...detail,
      groupId: groupIdValue,
    };
  });

  projectVariableGroups.value = nextGroups;
  projectVariables.value = nextVariables;
  await persistProjectGlobals();
  ElMessage.success("导入完成");
};

const handleExport = async (format) => {
  const rows = buildExportRows();
  if (format === "json") {
    const payload = {
      definitions: projectVariables.value || {},
      groups: projectVariableGroups.value || [],
    };
    downloadBlob(
      JSON.stringify(payload, null, 2),
      "project-variables.json",
      "application/json"
    );
    return;
  }

  const worksheet = XLSX.utils.json_to_sheet(rows);
  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, "variables");
  if (format === "csv") {
    const csv = XLSX.utils.sheet_to_csv(worksheet);
    downloadBlob(csv, "project-variables.csv", "text/csv");
    return;
  }
  const buffer = XLSX.write(workbook, { bookType: "xlsx", type: "array" });
  downloadBlob(buffer, "project-variables.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet");
};

const handleImport = (format) => {
  importType.value = format;
  nextTick(() => {
    if (importInputRef.value) {
      importInputRef.value.value = "";
      importInputRef.value.click();
    }
  });
};

const handleFileChange = async (event) => {
  const file = event.target.files?.[0];
  if (!file) return;
  if (importType.value === "json") {
    const text = await file.text();
    try {
      const data = JSON.parse(text);
      if (data && typeof data === "object" && (data.definitions || data.groups)) {
        await mergeImportedDefinitions(data.definitions || {}, data.groups || []);
        return;
      }
      if (Array.isArray(data)) {
        await mergeImportedRows(data);
        return;
      }
      ElMessage.error("JSON 格式不支持");
    } catch (error) {
      ElMessage.error("JSON 解析失败");
    }
    return;
  }

  const buffer = await file.arrayBuffer();
  const workbook = XLSX.read(buffer, { type: "array" });
  const sheetName = workbook.SheetNames[0];
  if (!sheetName) {
    ElMessage.error("文件中没有数据表");
    return;
  }
  const rows = XLSX.utils.sheet_to_json(workbook.Sheets[sheetName], { defval: "" });
  await mergeImportedRows(rows);
};

const handleEditValueMarkers = (markers) => {
  if (!isEditorType.value) {
    editValueHasErrors.value = false;
    return;
  }
  editValueHasErrors.value = (markers || []).some((marker) => marker.severity === 8);
};

onMounted(() => {
  document.addEventListener("click", handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener("click", handleClickOutside);
});
</script>

<style scoped>
.global-vars {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
  overflow: hidden;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.toolbar-button {
  height: 32px;
  padding: 0 14px;
  border-radius: 8px;
  font-weight: 600;
  letter-spacing: 0.2px;
  box-shadow: 0 6px 14px rgba(64, 158, 255, 0.18);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 1 1 220px;
  justify-content: center;
}

.toolbar-button--ghost {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary-light-5);
  background-color: var(--el-color-primary-light-9);
  box-shadow: 0 6px 12px rgba(64, 158, 255, 0.12);
}

.toolbar-icon {
  font-size: 14px;
}

.hidden-file-input {
  display: none;
}

.toolbar-button--ghost:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary);
  background-color: var(--el-color-primary-light-8);
}

.context-menu-item.is-disabled {
  color: #c0c4cc;
  pointer-events: none;
}

.tree-wrap {
  flex: 1;
  overflow: auto;
  border: 1px solid #e4e7ed;
  border-radius: 10px;
  position: relative;
  background: #fff;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
  min-height: 0;
}

.tree-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
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

.node-variable .node-icon {
  color: #10b981;
}

.node-icon.is-mapped {
  color: #2563eb;
}

.node-icon.is-unmapped {
  color: #94a3b8;
}

.tree-node:hover {
  background: #f5f7fa;
}

.tree-node.is-selected {
  background: linear-gradient(90deg, rgba(59, 130, 246, 0.14), rgba(59, 130, 246, 0.06));
  color: #1d4ed8;
  border: 1px solid rgba(59, 130, 246, 0.18);
  box-shadow: inset 0 0 0 1px rgba(59, 130, 246, 0.12);
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

.node-meta {
  font-size: 12px;
  color: #909399;
}

:deep(.tree-wrap .el-tree) {
  padding: 10px;
  min-height: 100%;
}

:deep(.tree-wrap .el-tree-node__content) {
  height: 38px;
}

.quick-form {
  --quick-control-height: var(--el-component-size-small, 28px);
}

.quick-form-row {
  margin-bottom: 10px;
}

.quick-form :deep(.el-form-item) {
  width: 100%;
  margin-bottom: 8px;
  align-items: center;
}

.quick-form :deep(.el-form-item__label) {
  line-height: var(--quick-control-height);
}

.quick-form :deep(.el-form-item__content) {
  flex: 1;
  min-width: 0;
}

.quick-form :deep(.el-input),
.quick-form :deep(.el-select) {
  width: 100%;
}

.quick-form :deep(.el-input__wrapper),
.quick-form :deep(.el-select .el-input__wrapper) {
  height: var(--quick-control-height);
  min-height: var(--quick-control-height);
}

.quick-form :deep(.el-input__inner),
.quick-form :deep(.el-select .el-input__inner) {
  height: var(--quick-control-height);
  line-height: var(--quick-control-height);
}

.quick-replace :deep(.el-form-item__content) {
  display: flex;
  align-items: center;
  gap: 6px;
}

.quick-replace :deep(.el-input) {
  flex: 1;
}

.quick-arrow {
  color: #909399;
  flex: 0 0 auto;
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
