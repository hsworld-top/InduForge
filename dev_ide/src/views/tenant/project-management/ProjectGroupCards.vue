<template>
  <section class="project-group-cards space-y-3">
    <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3">
      <button
        v-if="showAllEntry"
        type="button"
        class="text-left rounded-xl border p-3 transition-colors"
        :class="isAllEntryActive ? activeClassName : inactiveClassName"
        @click="handleSelectAll"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2 min-w-0">
            <el-icon class="text-sm"><Folder /></el-icon>
            <span class="font-semibold text-sm truncate">{{ resolvedAllProjectsLabel }}</span>
          </div>
          <el-tag size="small" type="info">{{ allProjectsCount }}</el-tag>
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
          {{ t('projectManagement.allProjectsDescription') }}
        </p>
      </button>

      <button
        v-for="group in normalizedGroups"
        :key="group.id"
        type="button"
        class="text-left rounded-xl border p-3 transition-colors"
        :class="group.id === activeGroupId ? activeClassName : inactiveClassName"
        @click="handleSelectGroup(group)"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2 min-w-0">
            <el-icon class="text-sm"><FolderOpened /></el-icon>
            <span class="font-semibold text-sm truncate">{{ group.name }}</span>
          </div>
          <el-tag size="small" type="info">{{ group.projectCount }}</el-tag>
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400 mt-2 line-clamp-2 min-h-[32px]">
          {{ group.description || t('projectManagement.noGroupDescription') }}
        </p>
      </button>
    </div>

    <el-empty v-if="!showAllEntry && normalizedGroups.length === 0" :description="resolvedEmptyText" />
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Folder, FolderOpened } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import type { ProjectOverviewGroup } from './project-overview.types'

export interface ProjectGroupCardItem extends ProjectOverviewGroup {
  projectCount?: number
}

const props = withDefaults(
  defineProps<{
    groups?: ProjectGroupCardItem[]
    activeGroupId?: string
    allProjectsCount?: number
    allProjectsLabel?: string
    showAllEntry?: boolean
    emptyText?: string
  }>(),
  {
    groups: () => [],
    activeGroupId: '',
    allProjectsCount: 0,
    allProjectsLabel: '',
    showAllEntry: true,
    emptyText: '',
  },
)

const { t } = useI18n()

const resolvedAllProjectsLabel = computed(
  () => props.allProjectsLabel || t('projectManagement.allProjects'),
)
const resolvedEmptyText = computed(() => props.emptyText || t('projectManagement.emptyProjects'))

const emit = defineEmits<{
  (event: 'select', payload: { groupId: string; groupName: string }): void
}>()

const normalizeCount = (value: unknown) => {
  const parsed = Number.parseInt(String(value), 10)
  return Number.isInteger(parsed) && parsed >= 0 ? parsed : 0
}

const normalizedGroups = computed<ProjectGroupCardItem[]>(() =>
  [...props.groups]
    .filter((group) => Boolean(group.id))
    .map((group) => ({
      ...group,
      name: group.name?.trim() || t('projectManagement.unnamedGroup'),
      description: group.description?.trim() || '',
      projectCount: normalizeCount(group.projectCount),
    }))
    .sort((left, right) => {
      const sortDelta = (left.sortOrder ?? 0) - (right.sortOrder ?? 0)
      if (sortDelta !== 0) {
        return sortDelta
      }
      return left.name.localeCompare(right.name, 'zh-Hans-CN')
    }),
)

const isAllEntryActive = computed(() => !props.activeGroupId)

const activeClassName =
  'border-blue-500 bg-blue-50/80 dark:bg-blue-900/20 shadow-sm text-blue-700 dark:text-blue-200'
const inactiveClassName =
  'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 hover:border-blue-300 dark:hover:border-blue-600 text-gray-700 dark:text-gray-200'

const handleSelectAll = () => {
  emit('select', {
    groupId: '',
    groupName: resolvedAllProjectsLabel.value,
  })
}

const handleSelectGroup = (group: ProjectGroupCardItem) => {
  emit('select', {
    groupId: group.id,
    groupName: group.name,
  })
}
</script>
