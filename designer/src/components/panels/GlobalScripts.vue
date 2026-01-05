<template>
  <div class="global-scripts">
    <el-tabs v-model="activeTab" class="scripts-tabs">
      <el-tab-pane label="系统脚本" name="system">
        <div class="scripts-layout">
          <div class="scripts-list is-full">
            <div class="list-header">系统事件</div>
            <el-menu :default-active="selectedSystemKey" class="list-menu" @select="selectSystem">
              <el-menu-item index="startup" @dblclick="openSystemEditor('startup')">
                <el-icon class="node-icon icon-system"><Pointer /></el-icon>
                系统启动
              </el-menu-item>
              <el-menu-item index="shutdown" @dblclick="openSystemEditor('shutdown')">
                <el-icon class="node-icon icon-system"><Close /></el-icon>
                系统关闭
              </el-menu-item>
            </el-menu>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="定时器" name="timers">
        <div class="scripts-layout">
          <div class="scripts-list is-full" @contextmenu="(event) => handleBlankContextMenu('timers', event)">
            <el-tree
              :data="timerTree"
              node-key="id"
              :default-expand-all="true"
              highlight-current
              :expand-on-click-node="false"
              draggable
              :allow-drop="allowScriptDrop"
              :allow-drag="allowScriptDrag"
              @node-click="(data) => handleScriptNodeClick('timers', data)"
              @node-dblclick="(data) => openScriptEditor('timers', data)"
              @node-contextmenu="(event, data) => handleTreeContextMenu('timers', event, data)"
              @node-drop="(draggingNode, dropNode, dropType) => handleScriptDrop('timers', draggingNode, dropNode, dropType)">
              <template #default="{ data }">
                <div class="tree-node" :class="`node-${data.type}`" @dblclick.stop="openScriptEditor('timers', data)">
                  <el-icon class="node-icon icon-timer">
                    <Folder v-if="data.type === 'group'" />
                    <Timer v-else />
                  </el-icon>
                  <span class="node-label" :class="{ 'is-group': data.type === 'group' }">{{ data.label }}</span>
                </div>
              </template>
            </el-tree>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="变量改变" name="variableChanges">
        <div class="scripts-layout">
          <div class="scripts-list is-full" @contextmenu="(event) => handleBlankContextMenu('variableChanges', event)">
            <el-tree
              :data="variableChangeTree"
              node-key="id"
              :default-expand-all="true"
              highlight-current
              :expand-on-click-node="false"
              draggable
              :allow-drop="allowScriptDrop"
              :allow-drag="allowScriptDrag"
              @node-click="(data) => handleScriptNodeClick('variableChanges', data)"
              @node-dblclick="(data) => openScriptEditor('variableChanges', data)"
              @node-contextmenu="(event, data) => handleTreeContextMenu('variableChanges', event, data)"
              @node-drop="(draggingNode, dropNode, dropType) => handleScriptDrop('variableChanges', draggingNode, dropNode, dropType)">
              <template #default="{ data }">
                <div class="tree-node" :class="`node-${data.type}`" @dblclick.stop="openScriptEditor('variableChanges', data)">
                  <el-icon class="node-icon icon-change">
                    <Folder v-if="data.type === 'group'" />
                    <Refresh v-else />
                  </el-icon>
                  <span class="node-label" :class="{ 'is-group': data.type === 'group' }">{{ data.label }}</span>
                </div>
              </template>
            </el-tree>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="自定义脚本" name="custom">
        <div class="scripts-layout">
          <div class="scripts-list is-full" @contextmenu="(event) => handleBlankContextMenu('custom', event)">
            <el-tree
              :data="customTree"
              node-key="id"
              :default-expand-all="true"
              highlight-current
              :expand-on-click-node="false"
              draggable
              :allow-drop="allowScriptDrop"
              :allow-drag="allowScriptDrag"
              @node-click="(data) => handleScriptNodeClick('custom', data)"
              @node-dblclick="(data) => openScriptEditor('custom', data)"
              @node-contextmenu="(event, data) => handleTreeContextMenu('custom', event, data)"
              @node-drop="(draggingNode, dropNode, dropType) => handleScriptDrop('custom', draggingNode, dropNode, dropType)">
              <template #default="{ data }">
                <div class="tree-node" :class="`node-${data.type}`" @dblclick.stop="openScriptEditor('custom', data)">
                  <el-icon class="node-icon icon-custom">
                    <Folder v-if="data.type === 'group'" />
                    <EditPen v-else />
                  </el-icon>
                  <span class="node-label" :class="{ 'is-group': data.type === 'group' }">{{ data.label }}</span>
                </div>
              </template>
            </el-tree>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="groupDialogVisible" :title="groupEditMode ? '编辑分组' : '新建分组'" width="420px" :close-on-click-modal="false" :lock-scroll="false">
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
        <el-button @click="groupDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveGroup">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="metaDialogVisible" :title="metaDialogTitle" width="420px" :close-on-click-modal="false" :lock-scroll="false">
      <el-form label-width="90px">
        <template v-if="metaDialogModule === 'timers'">
          <el-form-item label="定时器名称">
            <el-input v-model="metaForm.name" />
          </el-form-item>
          <el-form-item label="时间(ms)">
            <el-input-number v-model="metaForm.interval" :min="100" :step="100" style="width: 100%" />
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="metaForm.description" />
          </el-form-item>
        </template>
        <template v-else-if="metaDialogModule === 'variableChanges'">
          <el-form-item label="变量">
            <el-select v-model="metaForm.variable" placeholder="请选择变量">
              <el-option v-for="name in projectVariableNames" :key="name" :label="name" :value="name" />
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
      :close-on-click-modal="false"
      :lock-scroll="false"
      :before-close="handleSystemBeforeClose"
    >
      <div class="editor-meta">
        <div class="meta-title">{{ selectedSystemLabel }}</div>
        <div class="meta-desc">系统脚本</div>
      </div>
      <div class="editor-body">
        <div class="editor-main">
          <MonacoEditor ref="systemEditorRef" v-model="systemCode" language="javascript" height="360px" :completions="jsCompletions" />
        </div>
          <div class="editor-sidebar">
            <div class="sidebar-section">
              <div class="sidebar-title">工程变量</div>
              <el-input v-model="variableSearch" size="small" placeholder="搜索变量/分组" clearable />
              <div class="sidebar-scroll">
                <el-tree
                  ref="variableTreeRef"
                  :data="variableSidebarTree"
                  node-key="id"
                  :default-expand-all="true"
                  :expand-on-click-node="false"
                  :filter-node-method="filterSidebarNode"
                  @node-click="handleVariableInsert"
                >
                  <template #default="{ data }">
                    <div class="tree-node" :class="`node-${data.type}`">
                      <el-icon class="node-icon icon-variable">
                        <Folder v-if="data.type === 'group'" />
                        <Link v-else-if="data.mapped" />
                        <EditPen v-else />
                      </el-icon>
                      <span class="node-label" :class="{ 'is-group': data.type === 'group' }">{{ data.label }}</span>
                    </div>
                  </template>
                </el-tree>
              </div>
            </div>
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
                        <Folder v-if="data.type === 'group'" />
                        <EditPen v-else />
                      </el-icon>
                      <span class="node-label" :class="{ 'is-group': data.type === 'group' }">{{ data.label }}</span>
                    </div>
                  </template>
                </el-tree>
              </div>
            </div>
          </div>
      </div>
      <template #footer>
        <el-button @click="systemEditorVisible = false">取消</el-button>
        <el-button @click="formatSystemCode">格式化 (Shift+Alt+F)</el-button>
        <el-button type="primary" @click="saveSystemScript">保存 (Ctrl+S)</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="scriptEditorVisible"
      :title="scriptEditorTitle"
      width="980px"
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
          <span class="meta-value">{{ editorParams || '无' }}</span>
        </div>
      </div>
      <div class="editor-body">
        <div class="editor-main">
          <MonacoEditor ref="activeEditorRef" v-model="editorCode" language="javascript" height="360px" :completions="jsCompletions" />
        </div>
        <div class="editor-sidebar">
          <div class="sidebar-section">
            <div class="sidebar-title">工程变量</div>
            <el-input v-model="variableSearch" size="small" placeholder="搜索变量/分组" clearable />
            <div class="sidebar-scroll">
              <el-tree
                ref="variableTreeRef"
                :data="variableSidebarTree"
                node-key="id"
                :default-expand-all="true"
                :expand-on-click-node="false"
                :filter-node-method="filterSidebarNode"
                @node-click="handleVariableInsert"
              >
                <template #default="{ data }">
                  <div class="tree-node" :class="`node-${data.type}`">
                    <el-icon class="node-icon icon-variable">
                      <Folder v-if="data.type === 'group'" />
                      <Link v-else-if="data.mapped" />
                      <EditPen v-else />
                    </el-icon>
                    <span class="node-label" :class="{ 'is-group': data.type === 'group' }">{{ data.label }}</span>
                  </div>
                </template>
              </el-tree>
            </div>
          </div>
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
                      <Folder v-if="data.type === 'group'" />
                      <EditPen v-else />
                    </el-icon>
                    <span class="node-label" :class="{ 'is-group': data.type === 'group' }">{{ data.label }}</span>
                  </div>
                </template>
              </el-tree>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="scriptEditorVisible = false">取消</el-button>
        <el-button @click="formatActiveCode">格式化 (Shift+Alt+F)</el-button>
        <el-button type="primary" @click="saveActiveScript">保存 (Ctrl+S)</el-button>
      </template>
    </el-dialog>

    <div v-if="contextMenuVisible" class="context-menu" :style="contextMenuStyle" @click.stop @mousedown.stop>
      <template v-if="contextMenuNode?.type === 'blank'">
        <div class="context-menu-item" @click="openMetaDialog(contextMenuModule, 'create')">新建脚本</div>
        <div class="context-menu-item" @click="openGroupCreateFromMenu">新建分组</div>
        <div class="context-menu-item" :class="{ 'is-disabled': !scriptClipboard }" @click="pasteScript(contextMenuModule)">粘贴</div>
      </template>
      <template v-else-if="contextMenuNode?.type === 'item'">
        <div class="context-menu-item" @click="openScriptEditor(contextMenuModule)">打开</div>
        <div class="context-menu-item" @click="openMetaDialog(contextMenuModule, 'edit')">编辑</div>
        <div class="context-menu-item" @click="copyScript(contextMenuModule)">复制</div>
        <div class="context-menu-item" @click="showMoveToMenu = !showMoveToMenu">移动到</div>
        <div class="context-menu-item context-menu-item--danger" @click="removeScript(contextMenuModule)">删除</div>
      </template>
      <template v-else-if="contextMenuNode?.type === 'group'">
        <div class="context-menu-item" @click="openGroupEditFromMenu">编辑分组</div>
        <div class="context-menu-item" @click="openGroupCreateFromMenu">新建子分组</div>
        <div class="context-menu-item" @click="showMoveToMenu = !showMoveToMenu">移动到</div>
        <div class="context-menu-item context-menu-item--danger" @click="removeGroup(contextMenuModule)">删除分组</div>
      </template>
    </div>

    <div v-if="contextMenuVisible && showMoveToMenu" class="context-menu context-submenu" :style="submenuStyle" @click.stop @mousedown.stop>
      <div class="context-menu-item" @click="handleMoveTo(null)">根目录</div>
      <div v-for="group in availableGroups" :key="group.id" class="context-menu-item" @click="handleMoveTo(group.id)">
        {{ group.name }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch, onMounted, onUnmounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Folder, Timer, Refresh, EditPen, Pointer, Close, Link } from '@element-plus/icons-vue';
import MonacoEditor from '@/components/common/MonacoEditor.vue';
import { useDesignStore } from '@/store/design';

const store = useDesignStore();
const maxGroupDepth = 5;

const activeTab = ref('system');
const selectedSystemKey = ref('startup');

const selectedTimerId = ref('');
const selectedVariableId = ref('');
const selectedCustomId = ref('');

const systemCode = ref('');
const editorCode = ref('');
const systemOriginalCode = ref('');
const editorOriginalCode = ref('');
const editorOriginalInterval = ref(1000);
const editorInterval = ref(1000);
const editorParams = ref('');

const systemEditorRef = ref(null);

const scriptClipboard = ref(null);

const groupDialogVisible = ref(false);
const groupEditMode = ref(false);
const groupDialogModule = ref('');
const groupId = ref('');
const groupName = ref('');
const groupParentId = ref(null);
const metaDialogVisible = ref(false);
const metaDialogMode = ref('create');
const metaDialogModule = ref('');
const metaForm = ref({ id: '', name: '', interval: 1000, description: '', variable: '', params: '', groupId: null });
const systemEditorVisible = ref(false);
const scriptEditorVisible = ref(false);
const scriptEditorModule = ref('');
const activeEditorRef = ref(null);
const contextMenuVisible = ref(false);
const contextMenuPosition = ref({ x: 0, y: 0 });
const contextMenuNode = ref(null);
const contextMenuModule = ref('');
const showMoveToMenu = ref(false);

const projectVariableNames = computed(() => Object.keys(store.projectVariables || {}).sort());
const projectVariablesList = computed(() =>
  Object.entries(store.projectVariables || {}).map(([name, detail]) => ({
    name,
    groupId: detail?.groupId || null,
    meta: detail,
  })),
);
const customScripts = computed(() => store.projectGlobalScripts?.custom?.items || []);
const customScriptGroups = computed(() => store.projectGlobalScripts?.custom?.groups || []);
const variableGroups = computed(() => store.projectVariableGroups || []);

const variableSearch = ref('');
const scriptSearch = ref('');
const variableTreeRef = ref(null);
const customScriptTreeRef = ref(null);

const selectedSystemLabel = computed(() => (selectedSystemKey.value === 'startup' ? '系统启动' : '系统关闭'));
const editorMetaTitle = computed(() => getSelectedItem(scriptEditorModule.value)?.name || '未命名脚本');
const editorMetaDescription = computed(() => getSelectedItem(scriptEditorModule.value)?.description || '无描述');

const variableSidebarTree = computed(() => buildVariableTree(variableGroups.value, projectVariablesList.value));
const customScriptSidebarTree = computed(() => buildScriptTree(customScriptGroups.value, customScripts.value));

const jsCompletions = computed(() => {
  const items = [
    { label: 'console.log', insertText: 'console.log()', kind: 'Function', detail: 'Log output' },
    { label: 'if', insertText: 'if () {\\n  \\n}', kind: 'Snippet', detail: 'if statement' },
    { label: 'for', insertText: 'for (let i = 0; i < ; i++) {\\n  \\n}', kind: 'Snippet', detail: 'for loop' },
    { label: 'function', insertText: 'function name() {\\n  \\n}', kind: 'Snippet', detail: 'function declaration' },
    { label: 'const', insertText: 'const ', kind: 'Keyword' },
    { label: 'let', insertText: 'let ', kind: 'Keyword' },
    { label: 'return', insertText: 'return ', kind: 'Keyword' },
  ];

  projectVariableNames.value.forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: 'Variable',
      detail: '工程变量',
      prefix: '$global.',
    });
  });

  customScripts.value.forEach((script) => {
    if (!script?.name) return;
    const params = typeof script.params === 'string' ? script.params.trim() : '';
    const call = params ? `${script.name}(${params})` : `${script.name}()`;
    items.push({
      label: script.name,
      insertText: call,
      kind: 'Function',
      detail: '自定义脚本',
      prefix: 'customScripts.',
    });
  });

  return items;
});

const timerGroups = computed(() => store.projectGlobalScripts?.timers?.groups || []);
const variableChangeGroups = computed(() => store.projectGlobalScripts?.variableChanges?.groups || []);
const customGroups = computed(() => store.projectGlobalScripts?.custom?.groups || []);

const timerItems = computed(() => store.projectGlobalScripts?.timers?.items || []);
const variableChangeItems = computed(() => store.projectGlobalScripts?.variableChanges?.items || []);
const customItems = computed(() => store.projectGlobalScripts?.custom?.items || []);

const selectedTimer = computed(() => timerItems.value.find((item) => item.id === selectedTimerId.value));
const selectedVariableChange = computed(() => variableChangeItems.value.find((item) => item.id === selectedVariableId.value));
const selectedCustom = computed(() => customItems.value.find((item) => item.id === selectedCustomId.value));

const selectedTimerGroup = computed(() => timerGroups.value.find((group) => group.id === selectedTimerId.value));
const selectedVariableGroup = computed(() => variableChangeGroups.value.find((group) => group.id === selectedVariableId.value));
const selectedCustomGroup = computed(() => customGroups.value.find((group) => group.id === selectedCustomId.value));

const timerTree = computed(() => buildScriptTree(timerGroups.value, timerItems.value));
const variableChangeTree = computed(() => buildScriptTree(variableChangeGroups.value, variableChangeItems.value));
const customTree = computed(() => buildScriptTree(customGroups.value, customItems.value));

const contextMenuStyle = computed(() => ({
  left: `${contextMenuPosition.value.x}px`,
  top: `${contextMenuPosition.value.y}px`,
}));

const submenuStyle = computed(() => ({
  left: `${contextMenuPosition.value.x + 120}px`,
  top: `${contextMenuPosition.value.y}px`,
}));

const availableGroups = computed(() => {
  const module = contextMenuModule.value;
  const current = contextMenuNode.value;
  const groups = getGroupsByModule(module);
  if (!current || current.type !== 'group') return groups;
  return groups.filter((group) => group.id !== current.id && !isDescendantGroup(group.id, current.id, module));
});

const groupParentOptions = computed(() => {
  const groups = getGroupsByModule(groupDialogModule.value);
  if (!groupEditMode.value) return groups;
  return groups.filter((group) => group.id !== groupId.value && !isDescendantGroup(group.id, groupId.value, groupDialogModule.value));
});

const scriptEditorTitle = computed(() => {
  const module = scriptEditorModule.value || activeTab.value;
  const selected = getSelectedItem(module);
  if (selected?.name) return `${selected.name}脚本`;
  if (module === 'timers') return '定时器脚本';
  if (module === 'variableChanges') return '变量监听脚本';
  if (module === 'custom') return '自定义脚本';
  return '脚本';
});

const metaDialogTitle = computed(() => {
  const action = metaDialogMode.value === 'edit' ? '编辑' : '新建';
  if (metaDialogModule.value === 'timers') return `${action}定时器`;
  if (metaDialogModule.value === 'variableChanges') return `${action}变量监听`;
  if (metaDialogModule.value === 'custom') return `${action}自定义脚本`;
  return `${action}脚本`;
});

const syncSystemCode = () => {
  const system = store.projectGlobalScripts?.system || {};
  systemCode.value = selectedSystemKey.value === 'startup' ? system.startup?.code || '' : system.shutdown?.code || '';
};

watch(selectedSystemKey, syncSystemCode, { immediate: true });
watch(() => store.projectGlobalScripts?.system, syncSystemCode, { deep: true });

watch(selectedTimer, (item) => {
  if (!item) return;
  editorCode.value = item.code || '';
}, { immediate: true });

watch(selectedVariableChange, (item) => {
  if (!item) return;
  editorCode.value = item.code || '';
}, { immediate: true });

watch(selectedCustom, (item) => {
  if (!item) return;
  editorCode.value = item.code || '';
}, { immediate: true });

watch(variableSearch, (value) => {
  variableTreeRef.value?.filter?.(value);
});

watch(scriptSearch, (value) => {
  customScriptTreeRef.value?.filter?.(value);
});

function buildScriptTree(groups, items) {
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

  items.forEach((item) => {
    const node = { id: item.id, label: item.name || '未命名', type: 'item', itemId: item.id };
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

  variables.forEach((item) => {
    const node = {
      id: `var:${item.name}`,
      label: item.name,
      type: 'variable',
      name: item.name,
      mapped: !!item.meta?.mappedTo,
    };
    if (item.groupId && groupMap.has(item.groupId)) {
      groupMap.get(item.groupId).children.push(node);
    } else {
      roots.push(node);
    }
  });

  return roots;
}

function selectSystem(key) {
  selectedSystemKey.value = key;
}

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
  if (data?.type !== 'variable') return;
  insertText(`$global.${data.name}`);
}

function handleCustomScriptInsert(data) {
  if (data?.type !== 'item') return;
  const script = customScripts.value.find((item) => item.id === data.itemId);
  if (!script?.name) return;
  const params = typeof script.params === 'string' ? script.params.trim() : '';
  const call = params ? `${script.name}(${params})` : `${script.name}()`;
  insertText(`customScripts.${call}`);
}

function filterSidebarNode(value, data) {
  if (!value) return true;
  const keyword = String(value).toLowerCase();
  return String(data?.label || '').toLowerCase().includes(keyword);
}

function formatSystemCode() {
  systemEditorRef.value?.format?.();
}

function formatActiveCode() {
  activeEditorRef.value?.format?.();
}

function openSystemEditor(key) {
  if (key) selectedSystemKey.value = key;
  systemOriginalCode.value = systemCode.value || '';
  systemEditorVisible.value = true;
}

function openScriptEditor(module, data) {
  if (data?.type === 'item') {
    handleScriptNodeClick(module, data);
  }
  const selected = getSelectedItem(module);
  if (!selected) return;
  editorOriginalCode.value = selected.code || '';
  editorCode.value = selected.code || '';
  editorInterval.value = Number.isFinite(selected.interval) ? selected.interval : 1000;
  editorOriginalInterval.value = editorInterval.value;
  editorParams.value = selected.params || '';
  scriptEditorModule.value = module;
  scriptEditorVisible.value = true;
  closeContextMenu();
}

function openMetaDialog(module, mode) {
  metaDialogModule.value = module;
  metaDialogMode.value = mode;
  const selected = mode === 'edit' ? getSelectedItem(module) : null;
  if (mode === 'edit' && !selected) return;
  const targetGroupId = mode === 'create' ? getSelectedGroupId(module) : selected?.groupId || null;

  if (module === 'timers') {
    metaForm.value = {
      id: selected?.id || '',
      name: selected?.name || '',
      interval: Number.isFinite(selected?.interval) ? selected.interval : 1000,
      description: selected?.description || '',
      variable: '',
      params: '',
      groupId: targetGroupId,
    };
  } else if (module === 'variableChanges') {
    metaForm.value = {
      id: selected?.id || '',
      name: selected?.name || '',
      interval: 1000,
      description: selected?.description || '',
      variable: selected?.variable || '',
      params: '',
      groupId: targetGroupId,
    };
  } else if (module === 'custom') {
    metaForm.value = {
      id: selected?.id || '',
      name: selected?.name || '',
      interval: 1000,
      description: selected?.description || '',
      variable: '',
      params: selected?.params || '',
      groupId: targetGroupId,
    };
  }

  metaDialogVisible.value = true;
  closeContextMenu();
}

function saveMetaDialog() {
  const module = metaDialogModule.value;
  if (!module) return;

  if (module === 'timers') {
    const name = metaForm.value.name.trim();
    if (!name) return ElMessage.warning('定时器名称不能为空');
    const conflict = timerItems.value.some((item) => item.name === name && item.id !== metaForm.value.id);
    if (conflict) return ElMessage.warning('定时器名称不能重复');
    const interval = Number(metaForm.value.interval) || 1000;
    const description = metaForm.value.description.trim();

    if (metaDialogMode.value === 'edit') {
      const next = timerItems.value.map((item) =>
        item.id === metaForm.value.id ? { ...item, name, interval, description } : item,
      );
      updateModuleItems('timers', next);
    } else {
      const item = {
        id: crypto.randomUUID(),
        name,
        groupId: metaForm.value.groupId ?? null,
        interval,
        enabled: true,
        description,
        code: '',
      };
      updateModuleItems('timers', [...timerItems.value, item]);
      selectedTimerId.value = item.id;
    }
  }

  if (module === 'variableChanges') {
    const variable = metaForm.value.variable.trim();
    if (!variable) return ElMessage.warning('请选择变量');
    const description = metaForm.value.description.trim();
    const name = variable;

    if (metaDialogMode.value === 'edit') {
      const next = variableChangeItems.value.map((item) =>
        item.id === metaForm.value.id ? { ...item, name, variable, description } : item,
      );
      updateModuleItems('variableChanges', next);
    } else {
      const item = {
        id: crypto.randomUUID(),
        name,
        groupId: metaForm.value.groupId ?? null,
        variable,
        description,
        code: '',
      };
      updateModuleItems('variableChanges', [...variableChangeItems.value, item]);
      selectedVariableId.value = item.id;
    }
  }

  if (module === 'custom') {
    const name = metaForm.value.name.trim();
    if (!name) return ElMessage.warning('函数名称不能为空');
    const conflict = customItems.value.some((item) => item.name === name && item.id !== metaForm.value.id);
    if (conflict) return ElMessage.warning('函数名称不能重复');
    const description = metaForm.value.description.trim();
    const params = metaForm.value.params.trim();

    if (metaDialogMode.value === 'edit') {
      const next = customItems.value.map((item) =>
        item.id === metaForm.value.id ? { ...item, name, params, description } : item,
      );
      updateModuleItems('custom', next);
    } else {
      const item = {
        id: crypto.randomUUID(),
        name,
        groupId: metaForm.value.groupId ?? null,
        params,
        description,
        code: '',
      };
      updateModuleItems('custom', [...customItems.value, item]);
      selectedCustomId.value = item.id;
    }
  }

  metaDialogVisible.value = false;
}

async function persistGlobals() {
  try {
    await store.saveProjectVariables();
  } catch (error) {
    const message = error?.message ? String(error.message) : 'Unknown error';
    ElMessage.error(`保存脚本失败: ${message}`);
  }
}

async function saveSystemScript() {
  const scripts = store.projectGlobalScripts || {};
  const system = scripts.system || { startup: { code: '' }, shutdown: { code: '' } };
  const next = {
    ...system,
    [selectedSystemKey.value]: { code: systemCode.value || '' },
  };
  store.projectGlobalScripts = { ...scripts, system: next };
  store.isDirty = true;
  systemOriginalCode.value = systemCode.value || '';
  await persistGlobals();
  ElMessage.success('已保存脚本');
}

function handleScriptNodeClick(module, data) {
  if (data.type === 'item') {
    if (module === 'timers') selectedTimerId.value = data.itemId;
    if (module === 'variableChanges') selectedVariableId.value = data.itemId;
    if (module === 'custom') selectedCustomId.value = data.itemId;
  } else if (data.type === 'group') {
    if (module === 'timers') selectedTimerId.value = data.id;
    if (module === 'variableChanges') selectedVariableId.value = data.id;
    if (module === 'custom') selectedCustomId.value = data.id;
  }
}

function handleTreeContextMenu(module, event, data) {
  event.preventDefault();
  event.stopPropagation();
  contextMenuModule.value = module;
  contextMenuNode.value = data;
  contextMenuPosition.value = { x: event.clientX, y: event.clientY };
  contextMenuVisible.value = true;
  showMoveToMenu.value = false;
  if (data.type === 'item') {
    if (module === 'timers') selectedTimerId.value = data.itemId;
    if (module === 'variableChanges') selectedVariableId.value = data.itemId;
    if (module === 'custom') selectedCustomId.value = data.itemId;
  } else if (data.type === 'group') {
    if (module === 'timers') selectedTimerId.value = data.id;
    if (module === 'variableChanges') selectedVariableId.value = data.id;
    if (module === 'custom') selectedCustomId.value = data.id;
  }
}

function handleBlankContextMenu(module, event) {
  event.preventDefault();
  event.stopPropagation();
  contextMenuModule.value = module;
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

function openGroupEditFromMenu() {
  const module = contextMenuModule.value;
  closeContextMenu();
  openGroupEdit(module);
}

function openGroupCreateFromMenu() {
  const module = contextMenuModule.value;
  const groupIdValue = contextMenuNode.value?.type === 'group' ? contextMenuNode.value?.id : null;
  closeContextMenu();
  groupDialogModule.value = module;
  groupEditMode.value = false;
  groupId.value = '';
  groupName.value = '';
  groupParentId.value = groupIdValue;
  groupDialogVisible.value = true;
}

function handleMoveTo(groupIdValue) {
  const module = contextMenuModule.value;
  const node = contextMenuNode.value;
  closeContextMenu();
  if (!module || !node) return;
  if (node.type === 'item') {
    const items = getItemsByModule(module).map((item) =>
      item.id === node.itemId ? { ...item, groupId: groupIdValue } : item,
    );
    updateModuleItems(module, items);
    return;
  }
  if (node.type === 'group') {
    const groups = getGroupsByModule(module).map((group) =>
      group.id === node.id ? { ...group, parentId: groupIdValue } : group,
    );
    updateModuleGroups(module, groups);
  }
}

function allowScriptDrag() {
  return true;
}

function allowScriptDrop(draggingNode, dropNode, type) {
  const dragData = draggingNode.data;
  const dropData = dropNode.data;

  if (dragData.type === 'item') {
    if (type === 'inner' && dropData.type !== 'group') return false;
    return true;
  }

  if (dragData.type === 'group') {
    if (type === 'inner' && dropData.type !== 'group') return false;
    if (dropData.type === 'group' && isDescendantGroup(dropData.id, dragData.id, activeTab.value)) return false;
    if (type === 'inner' && dropData.type === 'group') {
      const depth = getGroupDepth(dropData.id, activeTab.value) + getGroupSubtreeDepth(dragData.id, activeTab.value);
      if (depth > maxGroupDepth) return false;
    }
    return true;
  }

  return false;
}

function handleScriptDrop(module, draggingNode, dropNode, dropType) {
  const dragData = draggingNode.data;
  const targetGroupId = resolveTargetGroupId(dropNode, dropType);

  if (dragData.type === 'item') {
    const items = getItemsByModule(module).map((item) =>
      item.id === dragData.itemId ? { ...item, groupId: targetGroupId } : item,
    );
    updateModuleItems(module, items);
    return;
  }

  if (dragData.type === 'group') {
    const groups = getGroupsByModule(module).map((group) =>
      group.id === dragData.id ? { ...group, parentId: targetGroupId } : group,
    );
    updateModuleGroups(module, groups);
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

async function removeScript(module) {
  const selected = getSelectedItem(module);
  if (!selected) return;
  try {
    await ElMessageBox.confirm(`确定删除 "${selected.name || '未命名'}" 吗？`, '确认删除', { type: 'warning', lockScroll: false });
  } catch (error) {
    return;
  }
  const items = getItemsByModule(module).filter((item) => item.id !== selected.id);
  updateModuleItems(module, items);
}

function copyScript(module) {
  const selected = getSelectedItem(module);
  if (!selected) return;
  scriptClipboard.value = { module, data: JSON.parse(JSON.stringify(selected)) };
  ElMessage.success('已复制脚本');
}

async function pasteScript(module) {
  if (!scriptClipboard.value) return;
  const source = scriptClipboard.value.data;
  const items = getItemsByModule(module);
  const nameBase = source.name || '脚本';
  let name = nameBase;
  let index = 1;
  while (items.some((item) => item.name === name)) {
    name = `${nameBase}_copy${index}`;
    index += 1;
  }
  const targetGroupId = getSelectedGroupId(module);
  const next = { ...source, id: crypto.randomUUID(), name, groupId: targetGroupId ?? source.groupId ?? null };
  updateModuleItems(module, [...items, next]);
}

function getSelectedItem(module) {
  if (module === 'timers') return selectedTimer.value;
  if (module === 'variableChanges') return selectedVariableChange.value;
  if (module === 'custom') return selectedCustom.value;
  return null;
}

function getGroupsByModule(module) {
  if (module === 'timers') return timerGroups.value;
  if (module === 'variableChanges') return variableChangeGroups.value;
  if (module === 'custom') return customGroups.value;
  return [];
}

function getItemsByModule(module) {
  if (module === 'timers') return timerItems.value;
  if (module === 'variableChanges') return variableChangeItems.value;
  if (module === 'custom') return customItems.value;
  return [];
}

function getSelectedGroupId(module) {
  if (module === 'timers') {
    if (selectedTimerGroup.value) return selectedTimerGroup.value.id;
    if (selectedTimer.value?.groupId) return selectedTimer.value.groupId;
  }
  if (module === 'variableChanges') {
    if (selectedVariableGroup.value) return selectedVariableGroup.value.id;
    if (selectedVariableChange.value?.groupId) return selectedVariableChange.value.groupId;
  }
  if (module === 'custom') {
    if (selectedCustomGroup.value) return selectedCustomGroup.value.id;
    if (selectedCustom.value?.groupId) return selectedCustom.value.groupId;
  }
  return null;
}

function updateModuleGroups(module, groups) {
  store.projectGlobalScripts = {
    ...store.projectGlobalScripts,
    [module]: {
      ...store.projectGlobalScripts[module],
      groups,
    },
  };
  store.isDirty = true;
  persistGlobals();
}

function updateModuleItems(module, items) {
  store.projectGlobalScripts = {
    ...store.projectGlobalScripts,
    [module]: {
      ...store.projectGlobalScripts[module],
      items,
    },
  };
  store.isDirty = true;
  persistGlobals();
}

function openGroupCreate(module) {
  groupEditMode.value = false;
  groupDialogModule.value = module;
  groupId.value = '';
  groupName.value = '';
  groupParentId.value = null;
  groupDialogVisible.value = true;
}

function openGroupEdit(module) {
  const groups = getGroupsByModule(module);
  const selectedId = module === 'timers' ? selectedTimerId.value : module === 'variableChanges' ? selectedVariableId.value : selectedCustomId.value;
  const group = groups.find((item) => item.id === selectedId);
  if (!group) return;
  groupEditMode.value = true;
  groupDialogModule.value = module;
  groupId.value = group.id;
  groupName.value = group.name;
  groupParentId.value = group.parentId || null;
  groupDialogVisible.value = true;
}

async function saveGroup() {
  const name = groupName.value.trim();
  if (!name) return ElMessage.warning('分组名不能为空');

  const parentId = groupParentId.value || null;
  const depth = parentId ? getGroupDepth(parentId, groupDialogModule.value) + 1 : 1;
  if (depth > maxGroupDepth) {
    return ElMessage.warning(`分组最多支持 ${maxGroupDepth} 层`);
  }

  const groups = getGroupsByModule(groupDialogModule.value);
  if (groupEditMode.value) {
    updateModuleGroups(groupDialogModule.value, groups.map((group) =>
      group.id === groupId.value ? { ...group, name, parentId } : group,
    ));
  } else {
    updateModuleGroups(groupDialogModule.value, [...groups, { id: crypto.randomUUID(), name, parentId, sortOrder: 0 }]);
  }
  groupDialogVisible.value = false;
}

async function removeGroup(module) {
  const groups = getGroupsByModule(module);
  const selectedId = module === 'timers' ? selectedTimerId.value : module === 'variableChanges' ? selectedVariableId.value : selectedCustomId.value;
  const group = groups.find((item) => item.id === selectedId);
  if (!group) return;

  try {
    await ElMessageBox.confirm(`确定删除分组 "${group.name}" 吗？组内成员会移动到父级。`, '确认删除', { type: 'warning', lockScroll: false });
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

function getGroupDepth(groupId, module) {
  if (!groupId) return 0;
  let depth = 1;
  let currentId = groupId;
  const groups = getGroupsByModule(module);
  const map = new Map(groups.map((g) => [g.id, g]));
  while (map.get(currentId)?.parentId) {
    depth += 1;
    currentId = map.get(currentId).parentId;
  }
  return depth;
}

function getGroupSubtreeDepth(groupId, module) {
  const groups = getGroupsByModule(module);
  const children = groups.filter((group) => group.parentId === groupId);
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
        code: editorCode.value || '',
        interval: module === 'timers' ? Number(editorInterval.value) || 1000 : item.interval,
      }
      : item,
  );
  updateModuleItems(module, items);
  editorOriginalCode.value = editorCode.value || '';
  editorOriginalInterval.value = editorInterval.value;
  ElMessage.success('已保存脚本');
}

function saveActiveScript() {
  if (scriptEditorModule.value === 'timers') {
    saveScriptCode('timers');
  } else if (scriptEditorModule.value === 'variableChanges') {
    saveScriptCode('variableChanges');
  } else if (scriptEditorModule.value === 'custom') {
    saveScriptCode('custom');
  }
}

async function handleSystemBeforeClose(done) {
  const isDirty = (systemCode.value || '') !== (systemOriginalCode.value || '');
  if (!isDirty) {
    done();
    return;
  }
  try {
    await ElMessageBox.confirm('脚本已修改，是否保存？', '提示', {
      confirmButtonText: '保存',
      cancelButtonText: '不保存',
      type: 'warning',
      lockScroll: false,
    });
    await saveSystemScript();
    done();
  } catch (error) {
    done();
  }
}

async function handleScriptBeforeClose(done) {
  const codeDirty = (editorCode.value || '') !== (editorOriginalCode.value || '');
  const intervalDirty = scriptEditorModule.value === 'timers' && editorInterval.value !== editorOriginalInterval.value;
  const isDirty = codeDirty || intervalDirty;
  if (!isDirty) {
    done();
    return;
  }
  try {
    await ElMessageBox.confirm('脚本已修改，是否保存？', '提示', {
      confirmButtonText: '保存',
      cancelButtonText: '不保存',
      type: 'warning',
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
  if (key === 's') {
    event.preventDefault();
    if (systemEditorVisible.value) saveSystemScript();
    if (scriptEditorVisible.value) saveActiveScript();
  }
  if (key === 'f' && event.shiftKey && event.altKey) {
    event.preventDefault();
    if (systemEditorVisible.value) formatSystemCode();
    if (scriptEditorVisible.value) formatActiveCode();
  }
}

function handleClickOutside() {
  if (contextMenuVisible.value) closeContextMenu();
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside);
  window.addEventListener('keydown', handleEditorShortcut);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
  window.removeEventListener('keydown', handleEditorShortcut);
});
</script>

<style scoped>
.global-scripts {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.scripts-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
}

:deep(.scripts-tabs .el-tabs__content) {
  flex: 1;
}

.scripts-layout {
  display: flex;
  gap: 12px;
  height: 100%;
  padding: 12px;
  box-sizing: border-box;
}

.scripts-list {
  width: 270px;
  border: 1px solid #e4e7ed;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
}

.scripts-list.is-full {
  width: 100%;
  flex: 1;
}

.list-header {
  padding: 12px 14px;
  font-size: 13px;
  font-weight: 600;
  border-bottom: 1px solid #e4e7ed;
  background: linear-gradient(180deg, #fafafa 0%, #f5f7fa 100%);
  letter-spacing: 0.3px;
}

.list-menu {
  border-right: 0;
}

.list-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 10px 10px 6px;
  border-bottom: 1px solid #e4e7ed;
  background: linear-gradient(180deg, #fbfbfd 0%, #f5f7fa 100%);
}

.scripts-editor {
  flex: 1;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.editor-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.editor-actions {
  display: flex;
  gap: 8px;
}

.editor-body {
  display: flex;
  gap: 12px;
  flex: 1;
  align-items: stretch;
  height: 360px;
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

.editor-main {
  flex: 1;
  min-width: 0;
}

.editor-sidebar {
  width: 200px;
  height: 360px;
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

.sidebar-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sidebar-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding-right: 4px;
}

.sidebar-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sidebar-group-title {
  font-size: 12px;
  color: #909399;
  padding-top: 6px;
}

.sidebar-item {
  justify-content: flex-start;
  padding: 2px 0;
  font-size: 12px;
}

.script-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
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

.list-menu .node-icon.icon-system {
  color: #6366f1;
  margin-right: 6px;
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

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

:deep(.scripts-list .el-tree) {
  flex: 1;
  overflow: auto;
  padding: 10px;
}

:deep(.scripts-list .el-tree-node__content) {
  height: 34px;
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

.context-menu-item.is-disabled {
  color: #c0c4cc;
  pointer-events: none;
}
</style>


