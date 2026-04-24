<template>
  <section class="project-overview-grid">
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-4">
      <el-skeleton v-for="index in 4" :key="index" animated class="rounded-xl border border-gray-100 p-4">
        <template #template>
          <el-skeleton-item variant="h3" class="!w-2/3 !h-5" />
          <el-skeleton-item variant="text" class="!w-full !h-3 mt-3" />
          <el-skeleton-item variant="text" class="!w-5/6 !h-3 mt-2" />
          <div class="mt-6 flex justify-between items-center">
            <el-skeleton-item variant="text" class="!w-1/3 !h-3" />
            <div class="flex gap-2">
              <el-skeleton-item variant="circle" class="!w-6 !h-6" />
              <el-skeleton-item variant="circle" class="!w-6 !h-6" />
              <el-skeleton-item variant="circle" class="!w-6 !h-6" />
            </div>
          </div>
        </template>
      </el-skeleton>
    </div>

    <el-empty v-else-if="projects.length === 0" :description="resolvedEmptyDescription" />

    <div v-else class="project-overview-grid__list">
      <article
        v-for="project in projects"
        :key="project.id"
        class="project-overview-grid__card"
        :class="{ 'is-selected': isSelected(project.id) }"
      >
        <header class="project-overview-grid__card-header">
          <div class="project-overview-grid__title-row">
            <el-checkbox
              v-if="showSelection"
              :model-value="isSelected(project.id)"
              @change="(value) => handleSelectionChange(project.id, value)"
            />
            <button
              type="button"
              class="project-overview-grid__title"
              :title="project.name"
              @click="emit('open-project', project)"
            >
              {{ project.name }}
            </button>
          </div>
          <el-tag size="small" :type="resolveRuntimeTag(project.runtimeSummary.runtimeStatus)">
            {{ resolveRuntimeText(project.runtimeSummary.runtimeStatus) }}
          </el-tag>
        </header>

        <p class="project-overview-grid__description">
          {{ project.description || t('projectManagement.noDescription') }}
        </p>

        <div class="project-overview-grid__tags">
          <el-tag
            v-for="tag in project.tags"
            :key="tag.id"
            size="small"
            effect="plain"
            class="!mr-0"
            type="info"
          >
            {{ tag.name }}
          </el-tag>
          <span
            v-if="project.tags.length === 0"
            class="text-[11px] text-gray-400 dark:text-gray-500 leading-5"
          >
            {{ t('projectManagement.noTags') }}
          </span>
        </div>

        <div
          class="project-overview-grid__footer"
        >
          <span class="project-overview-grid__time">
            {{ resolveDisplayTime(project) }}
          </span>

          <div class="project-overview-grid__actions">
            <slot name="actions" :project="project">
              <el-tooltip v-if="showMemberAction" :content="t('projectManagement.memberAndPermission')" placement="top">
                <el-button
                  size="small"
                  text
                  circle
                  class="!w-7 !h-7"
                  @click.stop="emit('open-runtime-access', project)"
                >
                  <el-icon><User /></el-icon>
                </el-button>
              </el-tooltip>

              <el-tooltip v-if="showDeployAction" :content="t('projectManagement.publishAndDeploy')" placement="top">
                <el-button
                  size="small"
                  text
                  circle
                  class="!w-7 !h-7"
                  @click.stop="emit('deploy', project)"
                >
                  <el-icon><UploadFilled /></el-icon>
                </el-button>
              </el-tooltip>

              <el-tooltip v-if="showExportAction" :content="t('projectManagement.exportProject')" placement="top">
                <el-button
                  size="small"
                  text
                  circle
                  class="!w-7 !h-7"
                  @click.stop="emit('export', project)"
                >
                  <el-icon><Download /></el-icon>
                </el-button>
              </el-tooltip>

              <el-tooltip v-if="showDeleteAction" :content="t('projectManagement.delete')" placement="top">
                <el-button
                  size="small"
                  text
                  circle
                  class="!w-7 !h-7 !text-red-500"
                  @click.stop="emit('delete', project)"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </slot>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import dayjs from 'dayjs'
import { Delete, Download, UploadFilled, User } from '@element-plus/icons-vue'
import type { TagProps } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { ProjectOverviewItem } from './project-overview.types'

const props = withDefaults(
  defineProps<{
    projects?: ProjectOverviewItem[]
    loading?: boolean
    emptyDescription?: string
    selectedIds?: string[]
    showSelection?: boolean
    showMemberAction?: boolean
    showDeployAction?: boolean
    showExportAction?: boolean
    showDeleteAction?: boolean
  }>(),
  {
    projects: () => [],
    loading: false,
    emptyDescription: '',
    selectedIds: () => [],
    showSelection: false,
    showMemberAction: true,
    showDeployAction: true,
    showExportAction: true,
    showDeleteAction: true,
  },
)

const { t } = useI18n()

const resolvedEmptyDescription = computed(
  () => props.emptyDescription || t('projectManagement.emptyProjects'),
)

const emit = defineEmits<{
  (event: 'open-project', payload: ProjectOverviewItem): void
  (event: 'selection-change', payload: { projectId: string; selected: boolean }): void
  (event: 'open-runtime-access', payload: ProjectOverviewItem): void
  (event: 'deploy', payload: ProjectOverviewItem): void
  (event: 'export', payload: ProjectOverviewItem): void
  (event: 'delete', payload: ProjectOverviewItem): void
}>()

const selectedIdSet = computed(() => new Set(props.selectedIds))

const resolveRuntimeTag = (status: string): TagProps['type'] => {
  const normalized = status.trim().toLowerCase()
  if (normalized === 'running') {
    return 'success'
  }
  if (normalized === 'deploying' || normalized === 'pending') {
    return 'warning'
  }
  if (normalized === 'error' || normalized === 'rollback') {
    return 'danger'
  }
  return 'info'
}

const resolveRuntimeText = (status: string) => {
  const normalized = status.trim().toLowerCase()
  if (normalized === 'running') {
    return t('projectManagement.deployStatusRunning')
  }
  if (normalized === 'deploying') {
    return t('projectManagement.deployStatusDeploying')
  }
  if (normalized === 'pending') {
    return t('projectManagement.deployStatusPending')
  }
  if (normalized === 'stopped') {
    return t('projectManagement.deployStatusStopped')
  }
  if (normalized === 'error') {
    return t('projectManagement.deployStatusError')
  }
  if (normalized === 'rollback') {
    return t('projectManagement.deployStatusRollback')
  }
  return t('projectManagement.deployStatusUnknown')
}

const resolveDisplayTime = (project: ProjectOverviewItem) => {
  const candidate =
    project.lastDeployedAt || project.updatedAt || project.createdAt || project.runtimeSummary.lastDeployedAt
  if (!candidate) {
    return '--'
  }
  const parsed = dayjs(candidate)
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm:ss') : '--'
}

const isSelected = (projectId: string) => selectedIdSet.value.has(projectId)

const handleSelectionChange = (projectId: string, value: string | number | boolean) => {
  emit('selection-change', {
    projectId,
    selected: Boolean(value),
  })
}
</script>

<style scoped>
.project-overview-grid {
  padding: 18px;
}

.project-overview-grid__list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 360px), 1fr));
  gap: 18px;
}

.project-overview-grid__card {
  display: flex;
  min-height: 238px;
  flex-direction: column;
  border: 1px solid #dbe3ed;
  border-radius: 18px;
  background: #ffffff;
  padding: 18px;
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.08);
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;
}

.project-overview-grid__card:hover {
  border-color: #9bbcff;
  box-shadow: 0 18px 34px rgba(15, 23, 42, 0.12);
  transform: translateY(-1px);
}

.project-overview-grid__card.is-selected {
  border-color: #2f67ff;
  box-shadow:
    0 0 0 1px rgba(47, 103, 255, 0.2),
    0 16px 32px rgba(37, 99, 235, 0.18);
}

.project-overview-grid__card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.project-overview-grid__title-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
}

.project-overview-grid__title {
  min-width: 0;
  border: 0;
  background: transparent;
  padding: 0;
  color: #111827;
  cursor: pointer;
  overflow: hidden;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
  font-weight: 700;
}

.project-overview-grid__title:hover {
  color: #1d4ed8;
}

.project-overview-grid__description {
  min-height: 56px;
  margin: 18px 0 0;
  border-radius: 12px;
  background: linear-gradient(135deg, #f7f9fc 0%, #eef2f7 100%);
  padding: 14px;
  color: #526173;
  font-size: 13px;
  line-height: 1.55;
}

.project-overview-grid__tags {
  display: flex;
  min-height: 30px;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 12px;
}

.project-overview-grid__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: auto;
  padding-top: 16px;
  border-top: 1px dashed #d9e0ea;
}

.project-overview-grid__time {
  flex-shrink: 0;
  border-radius: 999px;
  background: #f3f5f8;
  padding: 7px 12px;
  color: #7b8798;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
}

.project-overview-grid__actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 4px;
  max-width: 100%;
  min-height: 36px;
  border: 1px solid #e2e8f0;
  border-radius: 999px;
  background: #ffffff;
  padding: 4px 8px;
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.06);
}

html.dark .project-overview-grid__card,
[data-theme='dark'] .project-overview-grid__card {
  border-color: #374151;
  background: #111827;
  box-shadow: none;
}

html.dark .project-overview-grid__title,
[data-theme='dark'] .project-overview-grid__title {
  color: #e5e7eb;
}

html.dark .project-overview-grid__description,
[data-theme='dark'] .project-overview-grid__description {
  background: #1f2937;
  color: #cbd5e1;
}

html.dark .project-overview-grid__footer,
[data-theme='dark'] .project-overview-grid__footer {
  border-top-color: #374151;
}

html.dark .project-overview-grid__time,
html.dark .project-overview-grid__actions,
[data-theme='dark'] .project-overview-grid__time,
[data-theme='dark'] .project-overview-grid__actions {
  border-color: #374151;
  background: #1f2937;
}

@media (max-width: 720px) {
  .project-overview-grid {
    padding: 12px;
  }

  .project-overview-grid__list {
    grid-template-columns: 1fr;
  }

  .project-overview-grid__footer {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
