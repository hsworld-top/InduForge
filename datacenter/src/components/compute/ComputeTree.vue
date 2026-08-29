<template>
  <TreePanel
    class="compute-tree"
    :title="ui('计算单元', 'Compute Units')"
    :search-text="keyword"
    :search-placeholder="ui('搜索目录或计算单元', 'Search folders or compute units')"
    :all-label="ui('根目录', 'Root')"
    :all-active="!selectedFolderId"
    :loading="loading && hasTreeData"
    @update:search-text="updateKeyword"
    @select-all="$emit('selectFolder', null)"
  >
    <template #actions>
      <button
        type="button"
        class="is-primary"
        :title="ui('新建单元', 'New Unit')"
        :aria-label="ui('新建单元', 'New Unit')"
        @click="$emit('createUnit')"
      >
        <IconTablerPlus class="compute-tree__button-icon" />
      </button>
      <button
        type="button"
        :title="ui('新建文件夹', 'New Folder')"
        :aria-label="ui('新建文件夹', 'New Folder')"
        @click="$emit('createFolder')"
      >
        <IconTablerFolderPlus class="compute-tree__button-icon" />
      </button>
      <button
        type="button"
        :title="ui('依赖管理', 'Dependency Manager')"
        :aria-label="ui('依赖管理', 'Dependency Manager')"
        @click="$emit('manageDependencies')"
      >
        <IconTablerPackage class="compute-tree__button-icon" />
      </button>
      <button type="button" :title="ui('刷新', 'Refresh')" :aria-label="ui('刷新', 'Refresh')" @click="$emit('refresh')">
        <IconTablerRefresh class="compute-tree__button-icon" />
      </button>
    </template>

    <template #notice>
      <div v-if="listError" class="compute-tree__error">
        <IconTablerAlertCircle class="compute-tree__error-icon" />
        <span>{{ listError }}</span>
      </div>
      <div v-else-if="foldersError" class="compute-tree__warn">
        <IconTablerAlertCircle class="compute-tree__error-icon" />
        <span>{{ foldersError }}</span>
      </div>
    </template>

    <div v-if="loading && !hasTreeData" class="compute-tree__loading">
      <el-skeleton :rows="8" animated />
    </div>

    <template v-else>
      <span v-if="loading" class="compute-tree__updating">{{ ui('更新中…', 'Updating…') }}</span>
      <div class="compute-tree__root" role="tree" :aria-label="ui('计算单元资源树', 'Compute unit resource tree')">
        <ComputeTreeBranch
          v-for="folder in filteredFolders"
          :key="folder.id"
          :node="folder"
          :selected-unit-id="selectedUnitId"
          :dirty-unit-ids="dirtyUnitIds"
          :loading-folder-ids="loadingFolderIds"
          :folder-has-more-ids="folderHasMoreIds"
          :selected-folder-id="selectedFolderId"
          :search-active="Boolean(keyword)"
          @select-unit="$emit('selectUnit', $event)"
          @load-folder="$emit('loadFolder', $event)"
          @load-more-folder="$emit('loadMoreFolder', $event)"
          @select-folder="$emit('selectFolder', $event)"
          @folder-contextmenu="openFolderMenu"
          @unit-contextmenu="openUnitMenu"
        />
        <button
          v-if="rootFoldersHasMore"
          type="button"
          class="compute-tree__load-more"
          :disabled="rootFoldersLoading"
          @click="$emit('loadMoreFolder', null)"
        >
          {{ rootFoldersLoading ? ui('加载中…', 'Loading…') : ui('加载更多目录', 'Load More Folders') }}
        </button>
        <div
          v-for="unit in filteredRootUnits"
          :key="String(unit.id)"
          class="compute-tree__unit"
          :class="{ 'is-active': String(unit.id) === selectedUnitId }"
          role="treeitem"
          @contextmenu.prevent.stop="openUnitMenu($event, unit)"
        >
          <button
            type="button"
            class="compute-tree__unit-main"
            :title="unit.name"
            @click="$emit('selectUnit', String(unit.id))"
          >
            <IconTablerFileCode class="compute-tree__unit-icon" />
            <span class="compute-tree__unit-name">{{ unit.name }}</span>
          </button>
          <span
            class="compute-tree__unit-status"
            :class="`is-${unitStatusTone(unit)}`"
            :title="unitStatusText(unit)"
            :aria-label="unitStatusText(unit)"
          ></span>
          <button
            type="button"
            class="compute-tree__more"
            :aria-label="ui(`${unit.name}操作`, `Actions for ${unit.name}`)"
            :title="ui('计算单元操作', 'Compute Unit Actions')"
            @click.stop="openUnitMenu($event, unit)"
          >
            <IconTablerDots />
          </button>
        </div>
      </div>

      <EmptyState
        v-if="filteredRootUnits.length === 0 && filteredFolders.length === 0"
        icon-name="compute"
        :title="keyword ? ui('没有匹配的计算单元', 'No Matching Compute Units') : ui('暂无计算单元', 'No Compute Units')"
        :description="keyword ? ui('换个关键词再试。', 'Try another search term.') : ui('新建计算单元后会显示在列表中。', 'New compute units will appear here.')"
      />
    </template>

    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="compute-tree__menu-mask"
        @click="closeContextMenu"
        @contextmenu.prevent="closeContextMenu"
      >
        <div
          class="compute-tree__context-menu"
          :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
          @click.stop
        >
          <button type="button" @click="emitContextAction('rename')">
            <IconTablerPencil class="compute-tree__menu-icon" />
            <span>{{ ui('重命名', 'Rename') }}</span>
          </button>
          <button type="button" @click="emitContextAction('move')">
            <IconTablerFolderSymlink class="compute-tree__menu-icon" />
            <span>{{ ui('移动到分组', 'Move to Folder') }}</span>
          </button>
          <button type="button" class="is-danger" @click="emitContextAction('delete')">
            <IconTablerTrash class="compute-tree__menu-icon" />
            <span>{{ ui('删除', 'Delete') }}</span>
          </button>
        </div>
      </div>
    </Teleport>
  </TreePanel>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import IconTablerAlertCircle from '~icons/tabler/alert-circle'
import IconTablerDots from '~icons/tabler/dots'
import IconTablerFileCode from '~icons/tabler/file-code'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerFolderSymlink from '~icons/tabler/folder-symlink'
import IconTablerPackage from '~icons/tabler/package'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerTrash from '~icons/tabler/trash'
import type { ComputeFolder, ComputeUnit } from '@/api/schemas/compute.schema'
import { datacenterLocale } from '@/i18n/runtime'
import EmptyState from '@/components/shared/EmptyState.vue'
import TreePanel from '@/components/shared/TreePanel.vue'
import ComputeTreeBranch from './ComputeTreeBranch.vue'
import {
  buildComputeFolderTree,
  filterComputeTree,
  type ComputeFolderTreeNode,
} from './computeTreeModel'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = withDefaults(
  defineProps<{
    units: ComputeUnit[]
    folders: ComputeFolder[]
    selectedUnitId?: string | null
    dirtyUnitIds?: string[]
    loading?: boolean
    listError?: string
    foldersError?: string
    selectedFolderId?: string | null
    loadingFolderIds?: string[]
    folderHasMoreIds?: string[]
    rootFoldersHasMore?: boolean
    rootFoldersLoading?: boolean
  }>(),
  {
    selectedUnitId: null,
    dirtyUnitIds: () => [],
    loading: false,
    listError: '',
    foldersError: '',
    loadingFolderIds: () => [],
    folderHasMoreIds: () => [],
    rootFoldersHasMore: false,
    rootFoldersLoading: false,
    selectedFolderId: null,
  },
)

const emit = defineEmits<{
  (event: 'selectUnit', id: string): void
  (event: 'createUnit'): void
  (event: 'createFolder'): void
  (event: 'manageDependencies'): void
  (event: 'refresh'): void
  (event: 'search', keyword: string): void
  (event: 'loadFolder', folderId: string): void
  (event: 'loadMoreFolder', folderId: string | null): void
  (event: 'selectFolder', folderId: string | null): void
  (event: 'renameUnit', unit: ComputeUnit): void
  (event: 'moveUnit', unit: ComputeUnit): void
  (event: 'deleteUnit', unit: ComputeUnit): void
  (event: 'renameFolder', folder: ComputeFolderTreeNode): void
  (event: 'moveFolder', folder: ComputeFolderTreeNode): void
  (event: 'deleteFolder', folder: ComputeFolderTreeNode): void
}>()

const keyword = ref('')
const contextMenu = ref<{
  visible: boolean
  type: 'unit' | 'folder' | null
  x: number
  y: number
  unit: ComputeUnit | null
  folder: ComputeFolderTreeNode | null
}>({
  visible: false,
  type: null,
  x: 0,
  y: 0,
  unit: null,
  folder: null,
})

let searchTimer: number | undefined
const tree = computed(() => buildComputeFolderTree(props.folders, props.units))
const filteredTree = computed(() =>
  filterComputeTree(tree.value.rootFolders, tree.value.rootUnits, keyword.value),
)
const filteredFolders = computed(() => filteredTree.value.folders)
const filteredRootUnits = computed(() => filteredTree.value.units)
// 目录展开会触发按目录加载；保留现有树 DOM，避免加载态卸载分支后丢失展开状态。
const hasTreeData = computed(() => props.folders.length > 0 || props.units.length > 0)
const dirtyUnitIdSet = computed(() => new Set(props.dirtyUnitIds.map(String)))

function scheduleSearch(value: string) {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => emit('search', value.trim()), 250)
}

function updateKeyword(value: string) {
  keyword.value = value
  scheduleSearch(value)
}

onBeforeUnmount(() => window.clearTimeout(searchTimer))

function openUnitMenu(event: MouseEvent, unit: ComputeUnit) {
  const point = contextMenuPoint(event)
  contextMenu.value = {
    visible: true,
    type: 'unit',
    x: Math.min(point.x, window.innerWidth - 180),
    y: Math.min(point.y, window.innerHeight - 126),
    unit,
    folder: null,
  }
}

function openFolderMenu(event: MouseEvent, folder: ComputeFolderTreeNode) {
  const point = contextMenuPoint(event)
  contextMenu.value = {
    visible: true,
    type: 'folder',
    x: Math.min(point.x, window.innerWidth - 180),
    y: Math.min(point.y, window.innerHeight - 126),
    unit: null,
    folder,
  }
}

function contextMenuPoint(event: MouseEvent) {
  if (event.clientX || event.clientY) return { x: event.clientX, y: event.clientY }
  const rect = (event.currentTarget as HTMLElement | null)?.getBoundingClientRect()
  return { x: rect?.right || 0, y: rect?.bottom || 0 }
}

function closeContextMenu() {
  contextMenu.value.visible = false
}

function emitContextAction(action: 'rename' | 'move' | 'delete') {
  const { type, unit, folder } = contextMenu.value
  closeContextMenu()
  if (type === 'unit' && unit) {
    if (action === 'rename') {
      emit('renameUnit', unit)
      return
    }
    if (action === 'delete') {
      emit('deleteUnit', unit)
      return
    }
    emit('moveUnit', unit)
    return
  }
  if (type === 'folder' && folder) {
    if (action === 'rename') {
      emit('renameFolder', folder)
      return
    }
    if (action === 'delete') {
      emit('deleteFolder', folder)
      return
    }
    emit('moveFolder', folder)
    return
  }
}

const statusText = (status?: string) => {
  const map: Record<string, string> = {
    enabled: ui('启用', 'Enabled'),
    idle: ui('空闲', 'Idle'),
    running: ui('运行中', 'Running'),
    error: ui('异常', 'Error'),
    disabled: ui('停用', 'Disabled'),
  }
  return map[status || ''] || ui('未知', 'Unknown')
}

const statusTone = (status?: string) => {
  if (status === 'running' || status === 'enabled' || status === 'idle') return 'success'
  if (status === 'error') return 'danger'
  if (status === 'disabled') return 'muted'
  return 'info'
}

const isUnitDirty = (unit: ComputeUnit) => dirtyUnitIdSet.value.has(String(unit.id))

const unitStatusText = (unit: ComputeUnit) =>
  isUnitDirty(unit) ? ui('未保存', 'Unsaved') : statusText(unit.status)

const unitStatusTone = (unit: ComputeUnit) =>
  isUnitDirty(unit) ? 'warning' : statusTone(unit.status)
</script>

<style scoped>
.compute-tree__button-icon,
.compute-tree__error-icon,
.compute-tree__unit-icon {
  width: 16px;
  height: 16px;
}

.compute-tree__updating {
  padding: 3px 6px;
  color: var(--dc-text-muted);
  font-size: 11px;
}

.compute-tree__loading {
  padding: 12px;
}

.compute-tree__root {
  display: grid;
  gap: 2px;
}

.compute-tree__error,
.compute-tree__warn {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  margin: 10px 12px 0;
  padding: 9px 10px;
  border-radius: var(--dc-radius-sm);
  font-size: 12px;
  line-height: 1.5;
}

.compute-tree__error {
  border: 1px solid rgba(220, 38, 38, 0.2);
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.compute-tree__warn {
  border: 1px solid rgba(217, 119, 6, 0.22);
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
}

.compute-tree__unit {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  text-align: left;
}

.compute-tree__load-more {
  min-height: 30px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  font-size: 12px;
  font-weight: 700;
}

.compute-tree__load-more:disabled {
  cursor: wait;
  color: var(--dc-text-muted);
}

.compute-tree__unit:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.compute-tree__unit {
  min-height: 32px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 10px 26px;
  align-items: center;
  gap: 4px;
  padding: 2px 3px 2px 6px;
}

.compute-tree__unit.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.compute-tree__unit-main {
  min-width: 0;
  height: 28px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  align-items: center;
  gap: 6px;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
}

.compute-tree__unit-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-tree__unit-name {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.compute-tree__unit-status {
  width: 8px;
  height: 8px;
  justify-self: end;
  border-radius: 999px;
  background: var(--dc-text-muted);
}

.compute-tree__unit-status.is-success {
  background: var(--dc-success);
}

.compute-tree__unit-status.is-danger {
  background: var(--dc-danger);
}

.compute-tree__unit-status.is-warning {
  background: var(--dc-warning);
}

.compute-tree__unit-status.is-muted {
  background: var(--dc-text-muted);
}

.compute-tree__unit-status.is-info {
  background: var(--dc-primary);
}

.compute-tree__more {
  width: 24px;
  height: 26px;
  display: inline-grid;
  place-items: center;
  padding: 0;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--dc-text-muted);
  opacity: 0;
}

.compute-tree__unit:hover .compute-tree__more,
.compute-tree__more:focus-visible {
  opacity: 1;
}

.compute-tree__more:hover {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
}

.compute-tree__more svg {
  width: 16px;
  height: 16px;
}

.compute-tree__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}

.compute-tree__context-menu {
  position: fixed;
  min-width: 148px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.compute-tree__context-menu button {
  width: 100%;
  height: 30px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  text-align: left;
}

.compute-tree__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.compute-tree__context-menu button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.compute-tree__menu-icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

@media (max-width: 900px) {
  .compute-tree {
    border-right: none;
    border-bottom: 1px solid var(--dc-border);
  }
}
</style>
