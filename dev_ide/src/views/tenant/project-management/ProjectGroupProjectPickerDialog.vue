<template>
  <el-dialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="520px"
    :close-on-click-modal="false"
    class="project-group-project-picker"
  >
    <div v-if="availableProjects.length > 0" class="project-group-project-picker__list">
      <button
        v-for="project in availableProjects"
        :key="project.id"
        type="button"
        class="project-group-project-picker__item"
        data-testid="project-group-picker-item"
        @click="emit('select', project.id)"
      >
        {{ project.name }}
      </button>
    </div>
    <el-empty v-else :description="t('projectManagement.emptyProjects')" />
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ProjectOverviewItem } from './project-overview.types'

const props = withDefaults(
  defineProps<{
    visible: boolean
    groupName?: string
    projects?: ProjectOverviewItem[]
    groupId?: string
  }>(),
  {
    groupName: '',
    projects: () => [],
    groupId: '',
  },
)

const emit = defineEmits<{
  (event: 'update:visible', value: boolean): void
  (event: 'select', projectId: string): void
}>()

const { t } = useI18n()

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

const dialogTitle = computed(() => `${t('projectManagement.addProject')} - ${props.groupName}`)

const availableProjects = computed(() =>
  [...props.projects]
    .filter((project) => {
      const projectId = String(project?.id || '').trim()
      const projectGroupId = String(project?.group?.id || '').trim()
      return projectId && projectGroupId !== props.groupId
    })
    .map((project) => ({
      id: String(project.id),
      name: String(project.name || project.id),
    })),
)
</script>

<style scoped>
.project-group-project-picker__list {
  display: flex;
  max-height: 360px;
  flex-direction: column;
  gap: 8px;
  overflow-y: auto;
}

.project-group-project-picker__item {
  width: 100%;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
  padding: 10px 12px;
  color: #1f2937;
  cursor: pointer;
  text-align: left;
  transition:
    border-color 0.2s ease,
    background 0.2s ease;
}

.project-group-project-picker__item:hover {
  border-color: #93c5fd;
  background: #f8fbff;
}

html.dark .project-group-project-picker__item,
[data-theme='dark'] .project-group-project-picker__item {
  border-color: #374151;
  background: #111827;
  color: #e5e7eb;
}
</style>
