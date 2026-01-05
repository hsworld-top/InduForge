<template>
  <div class="global-vars">
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
        @node-click="handleNodeClick"
        @node-contextmenu="handleContextMenu"
        @node-drop="handleNodeDrop"
      >
        <template #default="{ data }">
          <div class="tree-node" :class="`node-${data.type}`">
            <el-icon
              class="node-icon"
              :class="{ 'is-mapped': data.type === 'variable' && data.meta?.mappedTo, 'is-unmapped': data.type === 'variable' && !data.meta?.mappedTo }"
            >
              <Folder v-if="data.type === 'group'" />
              <Link v-else-if="data.meta?.mappedTo" />
              <EditPen v-else />
            </el-icon>
            <span class="node-label" :class="{ 'is-group': data.type === 'group' }">{{ data.label }}</span>
            <span v-if="data.type === 'variable'" class="node-meta">{{ data.meta?.type || 'string' }}</span>
          </div>
        </template>
      </el-tree>
    <div v-if="!variableTree.length" class="tree-empty">
      <el-empty description="暂无工程变量" :image-size="60" />
    </div>
  </div>

    <div v-if="contextMenuVisible" class="context-menu" :style="contextMenuStyle" @click.stop @mousedown.stop>
      <template v-if="contextMenuNode?.type === 'blank'">
        <div class="context-menu-item" @click="openCreateFromMenu">新增变量</div>
        <div class="context-menu-item" @click="openGroupCreateFromMenu">新建分组</div>
        <div class="context-menu-item" @click="openQuickAddFromMenu">快速添加</div>
        <div class="context-menu-item" :class="{ 'is-disabled': !varClipboard }" @click="pasteVarFromMenu">粘贴</div>
      </template>
      <template v-if="contextMenuNode?.type === 'variable'">
        <div class="context-menu-item" @click="openEditFromMenu">编辑变量</div>
        <div class="context-menu-item" @click="copyVar">复制</div>
        <div class="context-menu-item" @click="showMoveToMenu = !showMoveToMenu">
          移动到
        </div>
        <div class="context-menu-item context-menu-item--danger" @click="removeVar">删除</div>
      </template>
      <template v-else-if="contextMenuNode?.type === 'group'">
        <div class="context-menu-item" @click="openGroupEditFromMenu">编辑分组</div>
        <div class="context-menu-item" @click="openGroupCreateFromMenu">新建子分组</div>
        <div class="context-menu-item" @click="showMoveToMenu = !showMoveToMenu">
          移动到
        </div>
        <div class="context-menu-item context-menu-item--danger" @click="removeGroup">删除分组</div>
      </template>
    </div>

    <div v-if="contextMenuVisible && showMoveToMenu" class="context-menu context-submenu" :style="submenuStyle" @click.stop @mousedown.stop>
      <div class="context-menu-item" @click="handleMoveTo(null)">根目录</div>
      <div v-for="group in availableGroups" :key="group.id" class="context-menu-item" @click="handleMoveTo(group.id)">
        {{ group.name }}
      </div>
    </div>

    <el-dialog v-model="editVisible" :title="editMode ? '编辑变量' : '新增变量'" width="520px" :close-on-click-modal="false" :lock-scroll="false">
      <el-form label-width="80px">
        <el-form-item label="变量名">
          <el-input v-model="editName" />
        </el-form-item>
        <el-form-item label="分组">
          <el-select v-model="editGroupId" placeholder="请选择分组">
            <el-option label="根目录" :value="null" />
            <el-option v-for="group in groupOptions" :key="group.id" :label="group.name" :value="group.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="editType" @change="resetEditValue">
            <el-option v-for="t in types" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="初始值">
          <el-input v-if="isTextType" v-model="editValue" type="textarea" :rows="6" />
          <el-input-number v-else-if="editType === 'number'" v-model="editValue" style="width:100%" />
          <el-switch v-else-if="editType === 'boolean'" v-model="editValue" />
          <el-date-picker v-else-if="editType === 'date'" v-model="editValue" type="datetime" style="width:100%" />
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
        <el-button @click="editVisible=false">取消</el-button>
        <el-button type="primary" @click="saveEdit">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="groupVisible" :title="groupEditMode ? '编辑分组' : '新建分组'" width="420px" :close-on-click-modal="false" :lock-scroll="false">
      <el-form label-width="70px">
        <el-form-item label="名称">
          <el-input v-model="groupName" />
        </el-form-item>
        <el-form-item label="父级">
          <el-select v-model="groupParentId" placeholder="根目录">
            <el-option label="根目录" :value="null" />
            <el-option v-for="group in groupParentOptions" :key="group.id" :label="group.name" :value="group.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="groupVisible=false">取消</el-button>
        <el-button type="primary" @click="saveGroup">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="quickVisible" title="快速添加数据源变量" width="900px" :close-on-click-modal="false" :lock-scroll="false">
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

      <el-table
        :data="filteredFields"
        border
        height="420"
        @selection-change="onSelectFields"
      >
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
        <el-button @click="quickVisible=false">取消</el-button>
        <el-button type="primary" @click="confirmQuickAdd">添加</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Folder, Link, EditPen } from '@element-plus/icons-vue';
import { useDesignStore } from '@/store/design';
import { designAPI } from '@/api/design.api';

const store = useDesignStore();
const maxGroupDepth = 5;

const types = ['string', 'number', 'boolean', 'array', 'object', 'set', 'map', 'date', 'regexp', 'function'];

const treeRef = ref(null);
const selectedNode = ref(null);
const varClipboard = ref(null);
const contextMenuVisible = ref(false);
const contextMenuPosition = ref({ x: 0, y: 0 });
const contextMenuNode = ref(null);
const showMoveToMenu = ref(false);

const editVisible = ref(false);
const editMode = ref(false);
const originalName = ref('');
const editName = ref('');
const editType = ref('string');
const editValue = ref('');
const editGroupId = ref(null);
const mapped = ref(false);
const mappedDs = ref(null);
const mappedField = ref('');
const dataSources = ref([]);

const groupVisible = ref(false);
const groupEditMode = ref(false);
const groupId = ref('');
const groupName = ref('');
const groupParentId = ref(null);

const quickVisible = ref(false);
const quickDs = ref(null);
const fields = ref([]);
const selectedFields = ref([]);
const searchKey = ref('');
const prefix = ref('');
const suffix = ref('');
const replaceFrom = ref('');
const replaceTo = ref('');

const isTextType = computed(() => ['string', 'array', 'object', 'regexp', 'function', 'set', 'map'].includes(editType.value));

const groupOptions = computed(() => store.projectVariableGroups || []);

const groupParentOptions = computed(() => {
  if (!groupEditMode.value) return groupOptions.value;
  return groupOptions.value.filter((group) => group.id !== groupId.value && !isDescendantGroup(group.id, groupId.value));
});

const selectedVariable = computed(() => {
  if (selectedNode.value?.type !== 'variable') return null;
  const name = selectedNode.value?.name;
  const detail = store.projectVariables?.[name];
  if (!detail) return null;
  return { name, detail };
});

const selectedGroup = computed(() => {
  if (selectedNode.value?.type !== 'group') return null;
  return store.projectVariableGroups?.find((group) => group.id === selectedNode.value.id) || null;
});

const variableTree = computed(() => buildTree(store.projectVariableGroups || [], store.projectVariables || {}));

const contextMenuStyle = computed(() => ({
  left: `${contextMenuPosition.value.x}px`,
  top: `${contextMenuPosition.value.y}px`,
}));

const submenuStyle = computed(() => ({
  left: `${contextMenuPosition.value.x + 120}px`,
  top: `${contextMenuPosition.value.y}px`,
}));

const availableGroups = computed(() => {
  const current = contextMenuNode.value;
  const groups = store.projectVariableGroups || [];
  if (!current || current.type !== 'group') return groups;
  return groups.filter((group) => group.id !== current.id && !isDescendantGroup(group.id, current.id));
});

function buildTree(groups, variables) {
  const groupMap = new Map();
  const roots = [];

  groups.forEach((group) => {
    groupMap.set(group.id, { id: group.id, label: group.name, type: 'group', children: [] });
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
      type: 'variable',
      name,
      meta: detail,
    };
    const groupId = detail?.groupId;
    if (groupId && groupMap.has(groupId)) {
      groupMap.get(groupId).children.push(node);
    } else {
      roots.push(node);
    }
  });

  return roots;
}

function handleNodeClick(data) {
  selectedNode.value = data;
}

function handleContextMenu(event, data) {
  event.preventDefault();
  event.stopPropagation();
  selectedNode.value = data;
  contextMenuNode.value = data;
  contextMenuPosition.value = { x: event.clientX, y: event.clientY };
  contextMenuVisible.value = true;
  showMoveToMenu.value = false;
}

function handleBlankContextMenu(event) {
  event.preventDefault();
  selectedNode.value = null;
  contextMenuNode.value = { type: 'blank' };
  contextMenuPosition.value = { x: event.clientX, y: event.clientY };
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
  closeContextMenu();
  openEdit();
}

function openGroupEditFromMenu() {
  closeContextMenu();
  openGroupEdit();
}

function openGroupCreateFromMenu() {
  closeContextMenu();
  openGroupCreate();
}

function handleMoveTo(groupId) {
  if (!contextMenuNode.value) return;
  const node = contextMenuNode.value;
  closeContextMenu();
  if (node.type === 'variable' && node.name) {
    store.projectVariables[node.name] = {
      ...store.projectVariables[node.name],
      groupId,
    };
    store.isDirty = true;
    persistProjectGlobals();
    return;
  }
  if (node.type === 'group') {
    const groups = store.projectVariableGroups || [];
    store.projectVariableGroups = groups.map((group) =>
      group.id === node.id ? { ...group, parentId: groupId } : group,
    );
    store.isDirty = true;
    persistProjectGlobals();
  }
}

function allowDrag() {
  return true;
}

function allowDrop(draggingNode, dropNode, type) {
  const dragData = draggingNode.data;
  const dropData = dropNode.data;

  if (dragData.type === 'variable') {
    if (type === 'inner' && dropData.type !== 'group') return false;
    return true;
  }

  if (dragData.type === 'group') {
    if (type === 'inner' && dropData.type !== 'group') return false;
    if (dropData.type === 'group' && isDescendantGroup(dropData.id, dragData.id)) return false;
    if (type === 'inner') {
      const depth = getGroupDepth(dropData.id) + getGroupSubtreeDepth(dragData.id);
      return depth <= maxGroupDepth;
    }
    return true;
  }

  return false;
}

function handleNodeDrop(draggingNode, dropNode, dropType) {
  const dragData = draggingNode.data;
  const dropData = dropNode.data;

  if (dragData.type === 'variable') {
    const targetGroupId = resolveTargetGroupId(dropNode, dropType);
    if (!dragData.name) return;
    store.projectVariables[dragData.name] = {
      ...store.projectVariables[dragData.name],
      groupId: targetGroupId,
    };
    store.isDirty = true;
    persistProjectGlobals();
    return;
  }

  if (dragData.type === 'group') {
    const targetGroupId = resolveTargetGroupId(dropNode, dropType);
    const groups = store.projectVariableGroups || [];
    const next = groups.map((group) =>
      group.id === dragData.id
        ? { ...group, parentId: targetGroupId }
        : group,
    );
    store.projectVariableGroups = next;
    store.isDirty = true;
    persistProjectGlobals();
  }
}

function resolveTargetGroupId(dropNode, dropType) {
  const dropData = dropNode.data;
  if (dropType === 'inner') {
    return dropData.type === 'group' ? dropData.id : null;
  }
  const parent = dropNode.parent?.data;
  return parent?.type === 'group' ? parent.id : null;
}

function getGroupDepth(groupId) {
  if (!groupId) return 0;
  let depth = 1;
  let currentId = groupId;
  const groups = store.projectVariableGroups || [];
  const map = new Map(groups.map((g) => [g.id, g]));
  while (map.get(currentId)?.parentId) {
    depth += 1;
    currentId = map.get(currentId).parentId;
  }
  return depth;
}

function getGroupSubtreeDepth(groupId) {
  const groups = store.projectVariableGroups || [];
  const children = groups.filter((group) => group.parentId === groupId);
  if (!children.length) return 1;
  const depths = children.map((child) => getGroupSubtreeDepth(child.id));
  return 1 + Math.max(...depths);
}

function isDescendantGroup(targetId, parentId) {
  const groups = store.projectVariableGroups || [];
  let current = groups.find((group) => group.id === targetId);
  while (current?.parentId) {
    if (current.parentId === parentId) return true;
    current = groups.find((group) => group.id === current.parentId);
  }
  return false;
}

function defaultEditValue(type) {
  switch (type) {
    case 'string':
      return '';
    case 'number':
      return 0;
    case 'boolean':
      return false;
    case 'array':
      return '[]';
    case 'object':
      return '{}';
    case 'set':
      return '[]';
    case 'map':
      return '[]';
    case 'date':
      return null;
    case 'regexp':
      return '/pattern/g';
    case 'function':
      return 'function(){}';
    default:
      return '';
  }
}

function resetEditValue() {
  editValue.value = defaultEditValue(editType.value);
}

async function loadDataSourcesForMapping() {
  dataSources.value = store.dataCenterConfig || [];
}

async function persistProjectGlobals() {
  try {
    await store.saveProjectVariables();
  } catch (error) {
    const message = error?.message ? String(error.message) : 'Unknown error';
    ElMessage.error(`保存工程变量失败: ${message}`);
  }
}

async function openCreate() {
  editMode.value = false;
  editName.value = '';
  editType.value = 'string';
  resetEditValue();
  editGroupId.value = selectedGroup.value?.id || null;
  mapped.value = false;
  mappedDs.value = null;
  mappedField.value = '';
  await loadDataSourcesForMapping();
  editVisible.value = true;
}

async function openEdit() {
  if (!selectedVariable.value) return;
  editMode.value = true;
  originalName.value = selectedVariable.value.name;
  editName.value = selectedVariable.value.name;
  editType.value = selectedVariable.value.detail?.type || 'string';
  editValue.value = selectedVariable.value.detail?.value;
  editGroupId.value = selectedVariable.value.detail?.groupId || null;
  mapped.value = !!selectedVariable.value.detail?.mappedTo;
  await loadDataSourcesForMapping();
  if (selectedVariable.value.detail?.mappedTo) {
    mappedDs.value = dataSources.value.find((ds) => ds.id === selectedVariable.value.detail.mappedTo.dsId) || null;
    mappedField.value = selectedVariable.value.detail.mappedTo.field || '';
  } else {
    mappedDs.value = null;
    mappedField.value = '';
  }
  editVisible.value = true;
}

async function saveEdit() {
  const name = editName.value.trim();
  if (!name) return ElMessage.warning('变量名不能为空');
  if (editMode.value && name !== originalName.value) {
    delete store.projectVariables[originalName.value];
  }
  store.projectVariables[name] = {
    type: editType.value,
    value: editValue.value,
    mappedTo: mapped.value && mappedDs.value ? { dsId: mappedDs.value.id, dsName: mappedDs.value.name, field: mappedField.value } : null,
    groupId: editGroupId.value || null,
  };
  store.isDirty = true;
  editVisible.value = false;
  await persistProjectGlobals();
  nextTick(() => {
    treeRef.value?.setCurrentKey(`var:${name}`);
  });
}

async function removeVar() {
  if (!selectedVariable.value) return;
  try {
    await ElMessageBox.confirm(`确定删除变量 "${selectedVariable.value.name}" 吗？`, '确认删除', { type: 'warning', lockScroll: false });
  } catch (error) {
    return;
  }
  delete store.projectVariables[selectedVariable.value.name];
  store.isDirty = true;
  await persistProjectGlobals();
  selectedNode.value = null;
}

function copyVar() {
  if (!selectedVariable.value) return;
  varClipboard.value = {
    name: selectedVariable.value.name,
    detail: JSON.parse(JSON.stringify(selectedVariable.value.detail || {})),
  };
  ElMessage.success('已复制变量');
}

async function pasteVar() {
  if (!varClipboard.value) return;
  const baseName = varClipboard.value.name || '变量';
  let name = baseName;
  let index = 1;
  while (store.projectVariables[name]) {
    name = `${baseName}_copy${index}`;
    index += 1;
  }
  const targetGroupId = selectedGroup.value?.id || selectedVariable.value?.detail?.groupId || null;
  store.projectVariables[name] = {
    ...varClipboard.value.detail,
    groupId: targetGroupId,
  };
  store.isDirty = true;
  await persistProjectGlobals();
  nextTick(() => treeRef.value?.setCurrentKey(`var:${name}`));
}

function openGroupCreate() {
  groupEditMode.value = false;
  groupId.value = '';
  groupName.value = '';
  groupParentId.value = selectedGroup.value?.id || null;
  if (groupParentId.value && getGroupDepth(groupParentId.value) >= maxGroupDepth) {
    ElMessage.warning(`分组最多支持 ${maxGroupDepth} 层`);
    groupParentId.value = null;
  }
  groupVisible.value = true;
}

function openGroupEdit() {
  if (!selectedGroup.value) return;
  groupEditMode.value = true;
  groupId.value = selectedGroup.value.id;
  groupName.value = selectedGroup.value.name;
  groupParentId.value = selectedGroup.value.parentId || null;
  groupVisible.value = true;
}

async function saveGroup() {
  const name = groupName.value.trim();
  if (!name) return ElMessage.warning('分组名不能为空');

  const parentId = groupParentId.value || null;
  const depth = parentId ? getGroupDepth(parentId) + 1 : 1;
  if (depth > maxGroupDepth) {
    return ElMessage.warning(`分组最多支持 ${maxGroupDepth} 层`);
  }

  const groups = store.projectVariableGroups || [];
  if (groupEditMode.value) {
    store.projectVariableGroups = groups.map((group) =>
      group.id === groupId.value ? { ...group, name, parentId } : group,
    );
  } else {
    store.projectVariableGroups = [
      ...groups,
      { id: crypto.randomUUID(), name, parentId, sortOrder: 0 },
    ];
  }
  store.isDirty = true;
  groupVisible.value = false;
  await persistProjectGlobals();
}

async function removeGroup() {
  if (!selectedGroup.value) return;
  try {
    await ElMessageBox.confirm(`确定删除分组 "${selectedGroup.value.name}" 吗？分组内成员会移动到父级。`, '确认删除', {
      type: 'warning',
      lockScroll: false,
    });
  } catch (error) {
    return;
  }
  const groups = store.projectVariableGroups || [];
  const parentId = selectedGroup.value.parentId || null;
  const nextGroups = groups
    .filter((group) => group.id !== selectedGroup.value.id)
    .map((group) => (group.parentId === selectedGroup.value.id ? { ...group, parentId } : group));
  store.projectVariableGroups = nextGroups;

  Object.entries(store.projectVariables || {}).forEach(([name, detail]) => {
    if (detail?.groupId === selectedGroup.value.id) {
      store.projectVariables[name] = { ...detail, groupId: parentId };
    }
  });

  store.isDirty = true;
  await persistProjectGlobals();
  selectedNode.value = null;
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
  if (!ds) {
    fields.value = [];
    return;
  }
  if (ds.type === 'relational') {
    const res = await designAPI.getQueries(store.projectId, { connectionId: ds.id });
    const variables = res.data?.queries || [];
    const isMysql = ds.relationalConfig?.dbType === 'mysql';
    fields.value = variables.map((v) => ({
      name: v.name,
      type: isMysql ? 'object' : 'string',
    }));
  } else {
    fields.value = [];
  }
}

const filteredFields = computed(() =>
  fields.value.filter((f) => f.name.toLowerCase().includes(searchKey.value.toLowerCase())),
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
  if (!quickDs.value) return '';
  return `{{${quickDs.value.name}.${field}}}`;
}

async function confirmQuickAdd() {
  const targetGroupId = selectedGroup.value?.id || null;
  selectedFields.value.forEach((field) => {
    const name = buildVarName(field.name);
    if (store.projectVariables[name]) return;
    store.projectVariables[name] = {
      type: field.type || 'string',
      value: defaultEditValue(field.type),
      mappedTo: { dsId: quickDs.value.id, dsName: quickDs.value.name, field: field.name },
      groupId: targetGroupId,
    };
  });
  store.isDirty = true;
  quickVisible.value = false;
  await persistProjectGlobals();
}

function handleClickOutside() {
  if (contextMenuVisible.value) closeContextMenu();
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
});
</script>

<style scoped>
.global-vars {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.context-menu-item.is-disabled {
  color: #c0c4cc;
  pointer-events: none;
}

.tree-wrap {
  flex: 1;
  overflow: hidden;
  border: 1px solid #e4e7ed;
  border-radius: 10px;
  position: relative;
  background: #fff;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
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
  padding: 4px 6px;
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
}

:deep(.tree-wrap .el-tree-node__content) {
  height: 34px;
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
  border-radius: 6px;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.12);
  z-index: 4000;
  min-width: 120px;
  padding: 4px 0;
}

.context-menu-item {
  padding: 8px 14px;
  cursor: pointer;
  font-size: 13px;
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
</style>
