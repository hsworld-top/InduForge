<template>
  <div class="collector-group-tree" @contextmenu.prevent>
    <div class="collector-group-tree__head">
      <strong>变量组织</strong>
      <el-tooltip content="新建变量分组" placement="top">
        <el-button
          text
          circle
          class="collector-group-tree__create"
          aria-label="新建变量分组"
          @click="createGroup(null)"
        >
          <IconTablerFolderPlus />
        </el-button>
      </el-tooltip>
    </div>
    <button
      type="button"
      class="collector-group-tree__all"
      @click="emit('select', null)"
      @contextmenu.prevent="openMenu($event, null)"
    >
      全部变量
    </button>
    <el-tree
      :key="treeVersion"
      lazy
      :load="loadNode"
      node-key="id"
      :props="{ label: 'name', children: 'children', isLeaf: 'leaf' }"
      highlight-current
      @current-change="emit('select', $event?.id || null)"
      @node-contextmenu="openMenu"
    />

    <div
      v-if="menu.visible"
      class="collector-group-tree__menu"
      :style="{ left: `${menu.x}px`, top: `${menu.y}px` }"
      @click.stop
    >
      <button type="button" @click="createGroup(menu.group?.id || null)">
        {{ menu.group ? '新建子分组' : '新建顶级分组' }}
      </button>
      <button type="button" @click="createPoint(menu.group?.id || null)">新建变量</button>
      <template v-if="menu.group">
        <span />
        <button type="button" @click="renameGroup(menu.group)">重命名</button>
        <button type="button" class="is-danger" @click="removeGroup(menu.group)">删除分组</button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import {
  createCollectorPointGroup,
  deleteCollectorPointGroup,
  listCollectorPointGroups,
  updateCollectorPointGroup,
} from '@/api/collector.api'
import type { CollectorPointGroup } from '@/api/schemas/collector.schema'

const props = defineProps<{ projectId: string; connectionId: string }>()
const emit = defineEmits<{
  select: [groupId: string | null]
  createPoint: [groupId: string | null]
  changed: []
}>()
const treeVersion = ref(0)
const menu = reactive({
  visible: false,
  x: 0,
  y: 0,
  group: null as CollectorPointGroup | null,
})

async function loadNode(
  node: { level: number; data?: { id: string } },
  resolve: (data: unknown[]) => void,
) {
  const groups = await listCollectorPointGroups(
    props.projectId,
    props.connectionId,
    node.level === 0 ? null : node.data?.id,
  )
  resolve(groups.map((group) => ({ ...group, leaf: false })))
}
function openMenu(event: MouseEvent, data: CollectorPointGroup | null) {
  event.preventDefault()
  menu.visible = true
  menu.x = Math.min(event.clientX, window.innerWidth - 170)
  menu.y = Math.min(event.clientY, window.innerHeight - 190)
  menu.group = data?.id ? data : null
}
function closeMenu() {
  menu.visible = false
}
async function createGroup(parentId: string | null) {
  closeMenu()
  try {
    const result = await ElMessageBox.prompt(
      '请输入分组名称',
      parentId ? '新增子分组' : '新增变量分组',
      {
        inputPattern: /^.{1,100}$/,
        inputErrorMessage: '分组名称长度必须为 1 到 100 个字符',
      },
    )
    await createCollectorPointGroup(props.projectId, props.connectionId, {
      parentId,
      name: result.value.trim(),
      sortOrder: 0,
      metadata: {},
    })
    refreshTree()
    ElMessage.success('分组已创建')
  } catch (error) {
    if (error !== 'cancel') throw error
  }
}
async function renameGroup(group: CollectorPointGroup) {
  closeMenu()
  try {
    const result = await ElMessageBox.prompt('请输入新的分组名称', '重命名变量分组', {
      inputValue: group.name,
      inputPattern: /^.{1,100}$/,
      inputErrorMessage: '分组名称长度必须为 1 到 100 个字符',
    })
    await updateCollectorPointGroup(props.projectId, props.connectionId, group.id, {
      name: result.value.trim(),
    })
    refreshTree()
    ElMessage.success('分组已重命名')
  } catch (error) {
    if (error !== 'cancel') throw error
  }
}
async function removeGroup(group: CollectorPointGroup) {
  closeMenu()
  try {
    await ElMessageBox.confirm(
      `确认删除分组“${group.name}”及其子分组？组内变量会移动到上级分组。`,
      '删除变量分组',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
    )
    await deleteCollectorPointGroup(props.projectId, props.connectionId, group.id)
    emit('select', group.parentId)
    refreshTree()
    ElMessage.success('分组已删除，变量已移动到上级分组')
  } catch (error) {
    if (error !== 'cancel') throw error
  }
}
function createPoint(groupId: string | null) {
  closeMenu()
  emit('createPoint', groupId)
}
function refreshTree() {
  treeVersion.value += 1
  emit('changed')
}
onMounted(() => document.addEventListener('click', closeMenu))
onBeforeUnmount(() => document.removeEventListener('click', closeMenu))
</script>

<style scoped>
.collector-group-tree {
  width: 220px;
  min-width: 220px;
  padding-right: 14px;
  border-right: 1px solid var(--dc-border);
}
.collector-group-tree__head {
  display: flex;
  min-height: 48px;
  align-items: center;
  justify-content: space-between;
}
.collector-group-tree__head strong {
  color: var(--dc-text-secondary);
  font-size: 14px;
  font-weight: 700;
}
.collector-group-tree__create {
  color: var(--dc-text-secondary);
}
.collector-group-tree__create:hover {
  color: var(--dc-primary);
  background: var(--dc-primary-soft);
}
.collector-group-tree__all {
  width: 100%;
  height: 34px;
  margin-bottom: 5px;
  padding: 0 10px;
  border: none;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
  text-align: left;
}
.collector-group-tree :deep(.el-tree) {
  --el-tree-node-hover-bg-color: var(--dc-surface-subtle);
  background: transparent;
  color: var(--dc-text-secondary);
}
.collector-group-tree :deep(.el-tree-node__content) {
  height: 34px;
  border-radius: var(--dc-radius-sm);
}
.collector-group-tree__menu {
  position: fixed;
  z-index: 4000;
  width: 164px;
  padding: 6px;
  border: 1px solid var(--dc-border);
  border-radius: 10px;
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-popover);
}
.collector-group-tree__menu button {
  width: 100%;
  height: 32px;
  padding: 0 10px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  text-align: left;
}
.collector-group-tree__menu button:hover {
  background: var(--dc-surface-subtle);
  color: var(--dc-text);
}
.collector-group-tree__menu button.is-danger {
  color: var(--dc-danger);
}
.collector-group-tree__menu span {
  display: block;
  height: 1px;
  margin: 5px 4px;
  background: var(--dc-border);
}
</style>
