<template>
  <div class="alarm-directory-node">
    <div class="alarm-directory-node__row" :class="{ 'is-active': selectedId === group.id }">
      <button
        type="button"
        class="alarm-directory-node__toggle"
        :disabled="!group.hasChildren"
        :title="expanded ? t('alarm.collapseChildren') : t('alarm.expandChildren')"
        @click="toggle"
      >
        <IconTablerChevronRight :class="{ 'is-expanded': expanded }" />
      </button>
      <button type="button" class="alarm-directory-node__select" @click="emit('select', group.id)">
        <IconTablerFolderOpen v-if="expanded" />
        <IconTablerFolder v-else />
        <span :title="group.fullPath || group.name">{{ group.name }}</span>
      </button>
      <el-dropdown trigger="click" @command="handleCommand">
        <button type="button" class="alarm-directory-node__more" :title="t('alarm.directoryActions')" @click.stop>
          <IconTablerDots />
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="edit">{{ t('alarm.edit') }}</el-dropdown-item>
            <el-dropdown-item command="delete">{{ t('alarm.delete') }}</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
    <div v-if="expanded" class="alarm-directory-node__children">
      <AlarmDirectoryNode
        v-for="child in children"
        :key="child.id"
        :project-id="projectId"
        :group="child"
        :selected-id="selectedId"
        @select="emit('select', $event)"
        @edit="emit('edit', $event)"
        @delete="emit('delete', $event)"
      />
      <span v-if="loading" class="alarm-directory-node__loading">{{ t('common.loading') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerDots from '~icons/tabler/dots'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderOpen from '~icons/tabler/folder-open'
import { listAlarmGroups } from '@/api/alarm.api'
import type { AlarmGroup } from '@/api/schemas/alarm.schema'
import { getApiErrorMessage } from '@/utils/request'
import { t } from '@/i18n/runtime'

defineOptions({ name: 'AlarmDirectoryNode' })
const props = defineProps<{ projectId: string; group: AlarmGroup; selectedId?: string }>()
const emit = defineEmits<{
  select: [id: string]
  edit: [group: AlarmGroup]
  delete: [group: AlarmGroup]
}>()
const expanded = ref(false)
const loading = ref(false)
const children = ref<AlarmGroup[]>([])
const loaded = ref(false)

async function toggle() {
  if (!props.group.hasChildren) return
  expanded.value = !expanded.value
  if (expanded.value && !loaded.value) await loadChildren()
}

async function loadChildren() {
  if (loading.value) return
  loading.value = true
  try {
    const nextChildren: AlarmGroup[] = []
    let page = 1
    let totalPages = 1
    do {
      const result = await listAlarmGroups(props.projectId, {
        parentId: props.group.id,
        page,
        pageSize: 100,
      })
      nextChildren.push(...result.list)
      totalPages = result.pagination.totalPages
      page += 1
    } while (page <= totalPages)
    children.value = Array.from(
      new Map(nextChildren.map((item) => [item.id, item])).values(),
    )
    loaded.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarm.loadChildrenFailed')))
  } finally {
    loading.value = false
  }
}

function handleCommand(command: string) {
  if (command === 'edit') emit('edit', props.group)
  if (command === 'delete') emit('delete', props.group)
}
</script>

<style scoped>
.alarm-directory-node__row {
  min-height: 32px;
  display: grid;
  grid-template-columns: 22px minmax(0, 1fr) 28px;
  align-items: center;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  color: var(--dc-text-secondary);
}
.alarm-directory-node__row:hover,
.alarm-directory-node__row.is-active {
  border-color: rgba(29, 78, 216, 0.24);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.alarm-directory-node__toggle,
.alarm-directory-node__more,
.alarm-directory-node__select {
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
}
.alarm-directory-node__toggle,
.alarm-directory-node__more {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.alarm-directory-node__toggle:disabled {
  visibility: hidden;
}
.alarm-directory-node__toggle svg {
  transition: transform 0.16s ease;
}
.alarm-directory-node__toggle svg.is-expanded {
  transform: rotate(90deg);
}
.alarm-directory-node__select {
  min-width: 0;
  height: 30px;
  display: flex;
  align-items: center;
  gap: 6px;
  text-align: left;
}
.alarm-directory-node__select span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}
.alarm-directory-node__more {
  opacity: 0;
}
.alarm-directory-node__row:hover .alarm-directory-node__more,
.alarm-directory-node__row.is-active .alarm-directory-node__more,
.alarm-directory-node__row:focus-within .alarm-directory-node__more {
  opacity: 1;
}
.alarm-directory-node__children {
  margin-left: 15px;
  padding-left: 6px;
  border-left: 1px solid var(--dc-border);
}
.alarm-directory-node__loading {
  min-height: 30px;
  padding: 0 10px;
  color: var(--dc-primary);
  font-size: 12px;
}
</style>
