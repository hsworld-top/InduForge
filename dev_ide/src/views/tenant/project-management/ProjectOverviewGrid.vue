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
      <!-- 分组文件夹卡片（混排在网格头部） -->
      <div
        v-for="group in groupCards"
        :key="`folder-${group.id}`"
        class="project-overview-grid__card folder-inline-card"
        @click="emit('group-select', { groupId: group.id, groupName: group.name })"
      >
        <div class="folder-inline-header">
          <div class="folder-inline-icon">
            <el-icon :size="24"><FolderOpened /></el-icon>
          </div>
          <div class="folder-inline-info">
            <span class="folder-inline-name">{{ group.name }}</span>
            <span class="folder-inline-count">
              {{ t('projectManagement.groupProjectCount', { count: group.projectCount }) }}
            </span>
          </div>
          <el-tooltip :content="t('projectManagement.addProjectToGroup')" placement="top">
            <button
              type="button"
              class="folder-icon-btn"
              @click.stop="emit('group-add-project', group)"
            >
              <el-icon :size="14"><Plus /></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip :content="t('projectManagement.editGroup')" placement="top">
            <button
              type="button"
              class="folder-icon-btn"
              @click.stop="emit('group-edit', group)"
            >
              <el-icon :size="14"><Edit /></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip :content="t('projectManagement.deleteGroup')" placement="top">
            <button
              type="button"
              class="folder-icon-btn folder-delete-btn"
              @click.stop="emit('group-delete', group)"
            >
              <el-icon :size="14"><Delete /></el-icon>
            </button>
          </el-tooltip>
        </div>
        <div v-if="group.projects && group.projects.length > 0" class="folder-inline-preview">
          <div
            v-for="project in group.projects.slice(0, 5)"
            :key="project.id"
            class="folder-preview-item"
          >
            <span class="folder-preview-name">{{ project.name }}</span>
            <button
              type="button"
              class="folder-preview-remove-btn"
              :title="t('projectManagement.removeProjectFromGroup')"
              @click.stop="emit('group-remove-project', { groupId: group.id, projectId: project.id })"
            >
              <el-icon :size="12"><Close /></el-icon>
            </button>
          </div>
          <div v-if="group.projects.length > 5" class="folder-preview-more">
            +{{ group.projects.length - 5 }} ...
          </div>
        </div>
        <div v-else class="folder-inline-empty">
          {{ t('projectManagement.emptyProjects') }}
        </div>
      </div>

      <!-- 工程卡片 -->
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
import { Close, Delete, Download, Edit, FolderOpened, Plus, UploadFilled, User } from '@element-plus/icons-vue'
import type { TagProps } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { ProjectOverviewItem, ProjectGroupCardViewModel } from './project-overview.types'

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
    /** 分组卡片数据，混排在工程卡片前面 */
    groupCards?: ProjectGroupCardViewModel[]
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
    groupCards: () => [],
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
  (event: 'group-select', payload: { groupId: string; groupName: string }): void
  (event: 'group-add-project', group: ProjectGroupCardViewModel): void
  (event: 'group-edit', group: ProjectGroupCardViewModel): void
  (event: 'group-delete', group: ProjectGroupCardViewModel): void
  (event: 'group-remove-project', payload: { groupId: string; projectId: string }): void
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
/* ─── 网格容器 ─── */
.project-overview-grid {
  padding: 4px 0;
}

.project-overview-grid__list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 280px), 1fr));
  gap: 16px;
  animation: ck-fadeUp 0.5s cubic-bezier(0.2, 0.8, 0.2, 1) both;
}

/* ─── 卡片：轻量紧凑风格 ─── */
.project-overview-grid__card {
  position: relative;
  display: flex;
  min-height: 200px;
  flex-direction: column;
  border: 1px solid rgba(0, 0, 0, 0.04);
  border-radius: var(--ck-radius-md);
  background: #ffffff;
  padding: 14px;
  box-shadow:
    0 1px 2px rgba(0, 0, 0, 0.02),
    0 4px 16px rgba(0, 0, 0, 0.02);
  transition: all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
  overflow: visible;
}

.project-overview-grid__card:hover {
  transform: translateY(-2px);
  box-shadow:
    0 4px 12px rgba(0, 0, 0, 0.06),
    0 12px 24px rgba(0, 0, 0, 0.04);
  border-color: rgba(0, 0, 0, 0.08);
}

.project-overview-grid__card.is-selected {
  border-color: var(--ck-primary);
  box-shadow: 0 4px 12px rgba(29, 78, 216, 0.08);
}

/* ─── 卡片头部 ─── */
.project-overview-grid__card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.project-overview-grid__title-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
}

.project-overview-grid__title {
  min-width: 0;
  border: 0;
  background: transparent;
  padding: 0;
  color: var(--ck-text-primary);
  cursor: pointer;
  overflow: hidden;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 600;
}

.project-overview-grid__title:hover {
  color: var(--ck-primary);
}

/* ─── 描述区：降低视觉权重 ─── */
.project-overview-grid__description {
  min-height: 44px;
  margin: 10px 0 0;
  border-radius: 8px;
  background: #f8fafc;
  border: 1px solid rgba(0, 0, 0, 0.03);
  padding: 10px;
  color: var(--ck-text-secondary);
  font-size: 12px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* ─── 标签行 ─── */
.project-overview-grid__tags {
  display: flex;
  min-height: 24px;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}

/* ─── 底部：时间 + 操作始终保持一行 ─── */
.project-overview-grid__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  margin-top: auto;
  padding-top: 10px;
  border-top: 1px solid var(--ck-border-light);
  min-width: 0;
}

.project-overview-grid__time {
  flex-shrink: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--ck-text-muted);
  font-size: 10px;
  opacity: 0.7;
}

/* ─── 操作按钮组：极简透明风格，不换行不压缩 ─── */
.project-overview-grid__actions {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  gap: 1px;
  background: transparent;
  padding: 0;
  border: none;
  box-shadow: none;
}

/* ─── 暗色模式 ─── */
html.dark .project-overview-grid__card,
[data-theme='dark'] .project-overview-grid__card {
  background: #1e293b;
  border-color: rgba(255, 255, 255, 0.05);
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.2);
}

html.dark .project-overview-grid__card:hover,
[data-theme='dark'] .project-overview-grid__card:hover {
  border-color: rgba(255, 255, 255, 0.12);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.3);
}

html.dark .project-overview-grid__title,
[data-theme='dark'] .project-overview-grid__title {
  color: var(--ck-text-primary);
}

html.dark .project-overview-grid__description,
[data-theme='dark'] .project-overview-grid__description {
  background: rgba(0, 0, 0, 0.2);
  border-color: rgba(255, 255, 255, 0.05);
  color: var(--ck-text-secondary);
}

html.dark .project-overview-grid__footer,
[data-theme='dark'] .project-overview-grid__footer {
  border-top-color: rgba(255, 255, 255, 0.06);
}

/* ─── stagger 入场动画 ─── */
.project-overview-grid__card:nth-child(1) { animation: ck-fadeUp 0.5s cubic-bezier(0.2, 0.8, 0.2, 1) 40ms both; }
.project-overview-grid__card:nth-child(2) { animation: ck-fadeUp 0.5s cubic-bezier(0.2, 0.8, 0.2, 1) 80ms both; }
.project-overview-grid__card:nth-child(3) { animation: ck-fadeUp 0.5s cubic-bezier(0.2, 0.8, 0.2, 1) 120ms both; }
.project-overview-grid__card:nth-child(4) { animation: ck-fadeUp 0.5s cubic-bezier(0.2, 0.8, 0.2, 1) 160ms both; }
.project-overview-grid__card:nth-child(5) { animation: ck-fadeUp 0.5s cubic-bezier(0.2, 0.8, 0.2, 1) 200ms both; }
.project-overview-grid__card:nth-child(6) { animation: ck-fadeUp 0.5s cubic-bezier(0.2, 0.8, 0.2, 1) 240ms both; }
.project-overview-grid__card:nth-child(7) { animation: ck-fadeUp 0.5s cubic-bezier(0.2, 0.8, 0.2, 1) 280ms both; }
.project-overview-grid__card:nth-child(8) { animation: ck-fadeUp 0.5s cubic-bezier(0.2, 0.8, 0.2, 1) 320ms both; }

@media (max-width: 720px) {
  .project-overview-grid__list {
    grid-template-columns: 1fr;
  }

  .project-overview-grid__footer {
    align-items: flex-start;
    flex-direction: column;
  }
}

/* ═══ 分组文件夹卡片（混排在网格中） ═══ */
.folder-inline-card {
  cursor: pointer;
  background: linear-gradient(135deg, #f0f7ff 0%, #e8f4f8 100%) !important;
  border: 1.5px dashed rgba(29, 78, 216, 0.25) !important;
  box-shadow: none !important;
}

.folder-inline-card:hover {
  border-color: var(--ck-primary) !important;
  background: linear-gradient(135deg, #e8f0fe 0%, #dbeafe 100%) !important;
  box-shadow: 0 8px 24px rgba(29, 78, 216, 0.12) !important;
}

.folder-inline-header {
  display: flex;
  align-items: center;
  gap: 10px;
}

.folder-inline-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: var(--ck-gradient-primary);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.folder-inline-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.folder-inline-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--ck-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.folder-inline-count {
  font-size: 11px;
  color: var(--ck-text-muted);
}

.folder-icon-btn {
  background: none;
  border: none;
  color: var(--ck-text-muted);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.folder-icon-btn:hover {
  color: var(--ck-primary);
  background: rgba(59, 130, 246, 0.1);
}

.folder-delete-btn:hover {
  color: var(--ck-danger);
  background: rgba(239, 68, 68, 0.1);
}

.folder-inline-preview {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
  background: rgba(255, 255, 255, 0.6);
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.04);
  margin-top: 10px;
  max-height: 140px;
  overflow-y: auto;
}

.folder-preview-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  padding: 3px 0;
}

.folder-preview-name {
  font-size: 12px;
  color: var(--ck-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}

.folder-preview-remove-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--ck-text-muted);
  cursor: pointer;
  transition: all 0.2s;
}

.folder-preview-remove-btn:hover {
  color: var(--ck-danger);
  background: rgba(239, 68, 68, 0.1);
}

.folder-preview-more {
  text-align: center;
  color: var(--ck-text-muted);
  font-size: 11px;
  padding: 2px 0;
}

.folder-inline-empty {
  margin-top: 8px;
  color: var(--ck-text-muted);
  font-size: 12px;
}

/* ─── 暗色模式分组卡片 ─── */
html.dark .folder-inline-card,
[data-theme='dark'] .folder-inline-card {
  background: linear-gradient(135deg, #1a2744 0%, #1e293b 100%) !important;
  border-color: rgba(96, 165, 250, 0.2) !important;
}

html.dark .folder-inline-card:hover,
[data-theme='dark'] .folder-inline-card:hover {
  background: linear-gradient(135deg, #1e3a5f 0%, #1e3050 100%) !important;
  border-color: rgba(96, 165, 250, 0.4) !important;
}

html.dark .folder-inline-preview,
[data-theme='dark'] .folder-inline-preview {
  background: rgba(0, 0, 0, 0.2);
  border-color: rgba(255, 255, 255, 0.05);
}
</style>
