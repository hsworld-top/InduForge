<template>
  <el-dialog
    v-model="dialogVisible"
    :title="t('projectManagement.groupManagement')"
    width="520px"
    :close-on-click-modal="false"
    class="project-group-manage-dialog"
  >
    <div class="project-group-manager">
      <div class="project-group-manager__creator">
        <el-input
          v-model="groupName"
          data-testid="project-group-name-input"
          :placeholder="t('projectManagement.inputGroupName')"
          clearable
          @keyup.enter="handleCreate"
        />
        <el-button
          type="primary"
          class="project-group-manager__create-button"
          data-testid="project-group-create-trigger"
          @click="handleCreate"
        >
          {{ t('projectManagement.createGroup') }}
        </el-button>
      </div>

      <div v-if="normalizedGroups.length > 0" class="project-group-manager__list">
        <div
          v-for="group in normalizedGroups"
          :key="group.id"
          class="project-group-manager__item"
        >
          <div class="project-group-manager__item-main">
            <div class="project-group-manager__item-name">{{ group.name }}</div>
            <div class="project-group-manager__item-count">
              {{ t('projectManagement.groupProjectCount', { count: group.projectCount }) }}
            </div>
          </div>
          <div class="project-group-manager__actions">
            <el-button text size="small" @click="emit('edit', group)">
              {{ t('projectManagement.editGroup') }}
            </el-button>
            <el-button text size="small" type="danger" @click="emit('delete', group)">
              {{ t('projectManagement.deleteGroup') }}
            </el-button>
          </div>
        </div>
      </div>

      <div v-else class="project-group-manager__empty">
        {{ t('projectManagement.groupManageEmpty') }}
      </div>
    </div>

    <template #footer>
      <el-button @click="dialogVisible = false">{{ t('projectManagement.close') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ProjectGroupCardViewModel } from './project-overview.types'

const props = withDefaults(
  defineProps<{
    visible: boolean
    groups?: ProjectGroupCardViewModel[]
  }>(),
  {
    groups: () => [],
  },
)

const emit = defineEmits<{
  (event: 'update:visible', value: boolean): void
  (event: 'create', payload: { name: string }): void
  (event: 'edit', group: ProjectGroupCardViewModel): void
  (event: 'delete', group: ProjectGroupCardViewModel): void
}>()

const { t } = useI18n()
const groupName = ref('')

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

const normalizeCount = (value: unknown) => {
  const parsed = Number.parseInt(String(value), 10)
  return Number.isInteger(parsed) && parsed >= 0 ? parsed : 0
}

const normalizedGroups = computed<ProjectGroupCardViewModel[]>(() =>
  [...props.groups]
    .filter((group) => Boolean(group.id))
    .map((group) => ({
      ...group,
      name: group.name?.trim() || t('projectManagement.unnamedGroup'),
      projectCount: normalizeCount(group.projectCount),
      projects: Array.isArray(group.projects) ? group.projects : [],
    }))
    .sort((left, right) => {
      const sortDelta = (left.sortOrder ?? 0) - (right.sortOrder ?? 0)
      if (sortDelta !== 0) {
        return sortDelta
      }
      return left.name.localeCompare(right.name, 'zh-Hans-CN')
    }),
)

const handleCreate = () => {
  const name = groupName.value.trim()
  if (!name) {
    return
  }
  emit('create', { name })
  groupName.value = ''
}

watch(
  () => props.visible,
  (visible) => {
    if (!visible) {
      groupName.value = ''
    }
  },
)
</script>

<style scoped>
.project-group-manager {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.project-group-manager__creator {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  padding: 12px;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  background: #f8fafc;
}

.project-group-manager__creator :deep(.el-input__wrapper) {
  border-radius: 8px;
  background: #f3f4f6;
  box-shadow: none;
}

.project-group-manager__create-button {
  border-radius: 8px;
  background: #2563eb;
  border-color: #2563eb;
}

.project-group-manager__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 340px;
  overflow-y: auto;
}

.project-group-manager__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  background: #ffffff;
}

.project-group-manager__item-main {
  min-width: 0;
}

.project-group-manager__item-name {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-group-manager__item-count {
  margin-top: 4px;
  font-size: 12px;
  color: #6b7280;
}

.project-group-manager__actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  gap: 4px;
}

.project-group-manager__empty {
  padding: 28px 16px;
  border: 1px dashed #d1d5db;
  border-radius: 10px;
  background: #f9fafb;
  color: #6b7280;
  text-align: center;
  font-size: 13px;
}

html.dark .project-group-manager__creator,
[data-theme='dark'] .project-group-manager__creator,
html.dark .project-group-manager__empty,
[data-theme='dark'] .project-group-manager__empty {
  border-color: #374151;
  background: #1f2937;
}

html.dark .project-group-manager__item,
[data-theme='dark'] .project-group-manager__item {
  border-color: #374151;
  background: #111827;
}

html.dark .project-group-manager__item-name,
[data-theme='dark'] .project-group-manager__item-name {
  color: #e5e7eb;
}

html.dark .project-group-manager__item-count,
[data-theme='dark'] .project-group-manager__item-count,
html.dark .project-group-manager__empty,
[data-theme='dark'] .project-group-manager__empty {
  color: #9ca3af;
}

@media (max-width: 640px) {
  .project-group-manager__creator {
    grid-template-columns: 1fr;
  }

  .project-group-manager__item {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
