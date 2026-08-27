<template>
  <section
    class="compute-tree-branch"
    role="treeitem"
    :aria-expanded="canExpand ? expanded : undefined"
  >
    <div
      class="compute-tree-branch__folder"
      :class="{ 'is-active': selectedFolderId === node.id }"
      @contextmenu.prevent.stop="$emit('folderContextmenu', $event, node)"
    >
      <button
        v-if="canExpand"
        type="button"
        class="compute-tree-branch__toggle"
        :aria-label="expanded ? `收起${node.name}` : `展开${node.name}`"
        @click.stop="toggleExpanded"
      >
        <IconTablerChevronRight
          class="compute-tree-branch__chevron"
          :class="{ 'is-open': expanded }"
        />
      </button>
      <span v-else class="compute-tree-branch__toggle-placeholder" aria-hidden="true"></span>
      <button
        type="button"
        class="compute-tree-branch__folder-main"
        :title="node.path || node.name"
        @click="$emit('selectFolder', node.id)"
        @dblclick="expandFromLabel"
      >
        <IconTablerFolderOpen v-if="expanded" class="compute-tree-branch__icon" />
        <IconTablerFolder v-else class="compute-tree-branch__icon" />
        <span class="compute-tree-branch__folder-name">{{ node.name }}</span>
        <span class="compute-tree-branch__count" :title="`${node.unitCount} 个计算单元`">{{
          node.unitCount
        }}</span>
      </button>
      <button
        type="button"
        class="compute-tree-branch__more"
        :aria-label="`${node.name}目录操作`"
        title="目录操作"
        @click.stop="$emit('folderContextmenu', $event, node)"
      >
        <IconTablerDots />
      </button>
    </div>

    <div v-if="expanded" class="compute-tree-branch__children" role="group">
      <div
        v-for="unit in node.units"
        :key="String(unit.id)"
        class="compute-tree-branch__unit"
        :class="{ 'is-active': String(unit.id) === selectedUnitId }"
        role="treeitem"
        @contextmenu.prevent.stop="$emit('unitContextmenu', $event, unit)"
      >
        <button
          type="button"
          class="compute-tree-branch__unit-main"
          :title="unit.name"
          @click="$emit('selectUnit', String(unit.id))"
        >
          <IconTablerFileCode class="compute-tree-branch__unit-icon" />
          <span class="compute-tree-branch__unit-name">{{ unit.name }}</span>
        </button>
        <span
          class="compute-tree-branch__status"
          :class="`is-${unitStatusTone(unit)}`"
          :title="unitStatusText(unit)"
          :aria-label="unitStatusText(unit)"
        ></span>
        <button
          type="button"
          class="compute-tree-branch__more"
          :aria-label="`${unit.name}操作`"
          title="计算单元操作"
          @click.stop="$emit('unitContextmenu', $event, unit)"
        >
          <IconTablerDots />
        </button>
      </div>

      <ComputeTreeBranch
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :selected-unit-id="selectedUnitId"
        :dirty-unit-ids="dirtyUnitIds"
        :loading-folder-ids="loadingFolderIds"
        :folder-has-more-ids="folderHasMoreIds"
        :selected-folder-id="selectedFolderId"
        :search-active="searchActive"
        @select-unit="$emit('selectUnit', $event)"
        @load-folder="$emit('loadFolder', $event)"
        @load-more-folder="$emit('loadMoreFolder', $event)"
        @select-folder="$emit('selectFolder', $event)"
        @folder-contextmenu="(mouseEvent, folder) => $emit('folderContextmenu', mouseEvent, folder)"
        @unit-contextmenu="(mouseEvent, unit) => $emit('unitContextmenu', mouseEvent, unit)"
      />
      <button
        v-if="folderHasMoreIds?.includes(node.id)"
        type="button"
        class="compute-tree-branch__load-more"
        :disabled="loadingFolderIds?.includes(node.id)"
        @click.stop="$emit('loadMoreFolder', node.id)"
      >
        {{ loadingFolderIds?.includes(node.id) ? '加载中…' : '加载更多子目录' }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerDots from '~icons/tabler/dots'
import IconTablerFileCode from '~icons/tabler/file-code'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderOpen from '~icons/tabler/folder-open'
import type { ComputeUnit } from '@/api/schemas/compute.schema'
import type { ComputeFolderTreeNode } from './computeTreeModel'

defineOptions({ name: 'ComputeTreeBranch' })

const props = defineProps<{
  node: ComputeFolderTreeNode
  selectedUnitId?: string | null
  dirtyUnitIds?: string[]
  loadingFolderIds?: string[]
  folderHasMoreIds?: string[]
  selectedFolderId?: string | null
  searchActive?: boolean
}>()

const emit = defineEmits<{
  (event: 'selectUnit', id: string): void
  (event: 'unitContextmenu', mouseEvent: MouseEvent, unit: ComputeUnit): void
  (event: 'folderContextmenu', mouseEvent: MouseEvent, folder: ComputeFolderTreeNode): void
  (event: 'loadFolder', folderId: string): void
  (event: 'loadMoreFolder', folderId: string): void
  (event: 'selectFolder', folderId: string): void
}>()

const expanded = ref(false)
const dirtyUnitIdSet = computed(() => new Set((props.dirtyUnitIds || []).map(String)))
const canExpand = computed(
  () => props.node.hasChildren || props.node.unitCount > 0 || props.node.children.length > 0,
)
const containsSelectedUnit = (node: ComputeFolderTreeNode): boolean =>
  node.units.some((unit) => String(unit.id) === props.selectedUnitId) ||
  node.children.some(containsSelectedUnit)

function toggleExpanded() {
  if (!canExpand.value) return
  expanded.value = !expanded.value
  if (
    expanded.value &&
    props.node.hasChildren &&
    props.node.children.length === 0 &&
    !props.loadingFolderIds?.includes(props.node.id)
  ) {
    emit('loadFolder', props.node.id)
  }
}

function expandFromLabel() {
  if (!canExpand.value) return
  expanded.value = true
}

watch(
  () => [props.selectedUnitId, props.searchActive, props.node] as const,
  () => {
    if (props.searchActive || containsSelectedUnit(props.node)) expanded.value = true
  },
  { immediate: true },
)

const statusText = (status?: string) => {
  const map: Record<string, string> = {
    enabled: '启用',
    idle: '空闲',
    running: '运行中',
    error: '异常',
    disabled: '停用',
  }
  return map[status || ''] || '未知'
}

const statusTone = (status?: string) => {
  if (status === 'running' || status === 'enabled' || status === 'idle') return 'success'
  if (status === 'error') return 'danger'
  if (status === 'disabled') return 'muted'
  return 'info'
}

const isUnitDirty = (unit: ComputeUnit) => dirtyUnitIdSet.value.has(String(unit.id))

const unitStatusText = (unit: ComputeUnit) =>
  isUnitDirty(unit) ? '未保存' : statusText(unit.status)

const unitStatusTone = (unit: ComputeUnit) =>
  isUnitDirty(unit) ? 'warning' : statusTone(unit.status)
</script>

<style scoped>
.compute-tree-branch {
  display: grid;
  gap: 2px;
}

.compute-tree-branch__folder,
.compute-tree-branch__unit {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}

.compute-tree-branch__load-more {
  min-height: 28px;
  margin-left: 20px;
  border: 0;
  background: transparent;
  color: var(--dc-primary);
  font-size: 12px;
  text-align: left;
}

.compute-tree-branch__folder {
  min-height: 32px;
  display: grid;
  grid-template-columns: 20px minmax(0, 1fr) 26px;
  align-items: center;
  gap: 2px;
  padding: 0 3px;
  font-size: 13px;
  font-weight: 700;
  text-align: left;
}

.compute-tree-branch__folder:hover,
.compute-tree-branch__unit:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.compute-tree-branch__folder.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.compute-tree-branch__toggle,
.compute-tree-branch__more,
.compute-tree-branch__folder-main,
.compute-tree-branch__unit-main {
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
}

.compute-tree-branch__toggle,
.compute-tree-branch__more {
  width: 24px;
  height: 26px;
  display: inline-grid;
  place-items: center;
  padding: 0;
  border-radius: 5px;
}

.compute-tree-branch__toggle:hover,
.compute-tree-branch__more:hover {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
}

.compute-tree-branch__toggle-placeholder {
  width: 20px;
}

.compute-tree-branch__folder-main {
  min-width: 0;
  height: 30px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 5px;
  padding: 0 2px;
  text-align: left;
}

.compute-tree-branch__more {
  opacity: 0;
  color: var(--dc-text-muted);
}

.compute-tree-branch__folder:hover .compute-tree-branch__more,
.compute-tree-branch__unit:hover .compute-tree-branch__more,
.compute-tree-branch__more:focus-visible {
  opacity: 1;
}

.compute-tree-branch__more svg {
  width: 16px;
  height: 16px;
}

.compute-tree-branch__chevron {
  width: 14px;
  height: 14px;
  transform: rotate(0deg);
  transition: transform 0.16s ease;
}

.compute-tree-branch__chevron.is-open {
  transform: rotate(90deg);
}

.compute-tree-branch__icon,
.compute-tree-branch__unit-icon {
  width: 16px;
  height: 16px;
  color: var(--dc-primary);
}

.compute-tree-branch__folder-name,
.compute-tree-branch__unit-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-tree-branch__count {
  min-width: 20px;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  text-align: center;
}

.compute-tree-branch__children {
  display: grid;
  gap: 2px;
  margin-left: 10px;
  padding-left: 6px;
}

.compute-tree-branch__unit {
  min-height: 32px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 10px 26px;
  align-items: center;
  gap: 4px;
  padding: 2px 3px 2px 6px;
  text-align: left;
}

.compute-tree-branch__unit.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.compute-tree-branch__unit-main {
  min-width: 0;
  height: 28px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  align-items: center;
  gap: 6px;
  padding: 0;
  text-align: left;
}

.compute-tree-branch__unit-name {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.compute-tree-branch__status {
  width: 8px;
  height: 8px;
  justify-self: end;
  border-radius: 999px;
  background: var(--dc-text-muted);
}

.compute-tree-branch__status.is-success {
  background: var(--dc-success);
}

.compute-tree-branch__status.is-danger {
  background: var(--dc-danger);
}

.compute-tree-branch__status.is-warning {
  background: var(--dc-warning);
}

.compute-tree-branch__status.is-muted {
  background: var(--dc-text-muted);
}

.compute-tree-branch__status.is-info {
  background: var(--dc-primary);
}
</style>
