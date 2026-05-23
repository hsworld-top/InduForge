<template>
  <section class="project-group-cards">
    <div class="project-group-cards__grid">
      <button
        v-if="showAllEntry"
        type="button"
        class="project-group-card project-group-card--button"
        :class="{ 'is-active': !activeGroupId }"
        @click="handleSelectAll"
      >
        <div class="project-group-card__header">
          <span class="project-group-card__title">
            <el-icon><Folder /></el-icon>
            <span>{{ t('projectManagement.allProjects') }}</span>
          </span>
        </div>
        <div class="project-group-card__description">
          {{ t('projectManagement.allProjectsDescription') }}
        </div>
      </button>

      <article
        v-for="group in normalizedGroups"
        :key="group.id"
        class="project-group-card"
        :class="{ 'is-active': group.id === activeGroupId }"
      >
        <button type="button" class="project-group-card__select" @click="handleSelectGroup(group)">
          <div class="project-group-card__header">
            <span class="project-group-card__title">
              <el-icon><FolderOpened /></el-icon>
              <span>{{ group.name }}</span>
            </span>
            <el-tag size="small" type="info">
              {{ t('projectManagement.groupProjectCount', { count: group.projectCount }) }}
            </el-tag>
          </div>
        </button>

        <div class="project-group-card__projects">
          <div
            v-for="project in group.projects"
            :key="project.id"
            class="project-group-card__project"
          >
            <span class="project-group-card__project-name">{{ project.name }}</span>
            <el-button
              text
              size="small"
              type="danger"
              data-testid="project-group-remove-project"
              :title="t('projectManagement.removeProjectFromGroup')"
              @click.stop="emitRemoveProject(group.id, project.id)"
            >
              {{ t('projectManagement.removeProjectFromGroup') }}
            </el-button>
          </div>
          <div v-if="group.projects.length === 0" class="project-group-card__empty">
            {{ t('projectManagement.emptyProjects') }}
          </div>
        </div>

        <div class="project-group-card__actions">
          <el-button
            size="small"
            data-testid="project-group-add-project"
            @click.stop="emit('add-project', group)"
          >
            {{ t('projectManagement.addProject') }}
          </el-button>
          <el-button
            size="small"
            data-testid="project-group-edit"
            @click.stop="emit('edit', group)"
          >
            {{ t('projectManagement.editGroup') }}
          </el-button>
          <el-button
            size="small"
            type="danger"
            data-testid="project-group-delete"
            @click.stop="emit('delete', group)"
          >
            {{ t('projectManagement.deleteGroup') }}
          </el-button>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Folder, FolderOpened } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import type { ProjectGroupCardViewModel } from './project-overview.types'

const props = withDefaults(
  defineProps<{
    groups?: ProjectGroupCardViewModel[]
    activeGroupId?: string
    showAllEntry?: boolean
  }>(),
  {
    groups: () => [],
    activeGroupId: '',
    showAllEntry: false,
  },
)

const emit = defineEmits<{
  (event: 'select', payload: { groupId: string; groupName: string }): void
  (event: 'add-project', group: ProjectGroupCardViewModel): void
  (event: 'edit', group: ProjectGroupCardViewModel): void
  (event: 'delete', group: ProjectGroupCardViewModel): void
  (event: 'remove-project', payload: { groupId: string; projectId: string }): void
}>()

const { t } = useI18n()

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
      description: group.description?.trim() || '',
      projectCount: normalizeCount(group.projectCount),
      projects: Array.isArray(group.projects)
        ? group.projects
            .filter((project) => project?.id)
            .map((project) => ({
              id: String(project.id),
              name: String(project.name || project.id),
            }))
        : [],
    }))
    .sort((left, right) => {
      const sortDelta = (left.sortOrder ?? 0) - (right.sortOrder ?? 0)
      if (sortDelta !== 0) {
        return sortDelta
      }
      return left.name.localeCompare(right.name, 'zh-Hans-CN')
    }),
)

const handleSelectAll = () => {
  emit('select', {
    groupId: '',
    groupName: t('projectManagement.allProjects'),
  })
}

const handleSelectGroup = (group: ProjectGroupCardViewModel) => {
  emit('select', {
    groupId: group.id,
    groupName: group.name,
  })
}

const emitRemoveProject = (groupId: string, projectId: string) => {
  emit('remove-project', { groupId, projectId })
}
</script>

<style scoped>
.project-group-cards__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 12px;
}

.project-group-card {
  display: flex;
  min-height: 180px;
  flex-direction: column;
  gap: 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
  padding: 14px;
  text-align: left;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.project-group-card.is-active {
  border-color: #2563eb;
  box-shadow: 0 0 0 1px rgba(37, 99, 235, 0.18);
}

.project-group-card--button {
  cursor: pointer;
}

.project-group-card--button:hover,
.project-group-card:hover {
  border-color: #93c5fd;
}

.project-group-card__select {
  display: block;
  width: 100%;
  border: 0;
  background: transparent;
  padding: 0;
  color: inherit;
  cursor: pointer;
  text-align: left;
}

.project-group-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
}

.project-group-card__title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: #111827;
  font-size: 14px;
  font-weight: 600;
}

.project-group-card__title span:last-child {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-group-card__description,
.project-group-card__empty {
  color: #6b7280;
  font-size: 12px;
  line-height: 1.5;
}

.project-group-card__projects {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 6px;
  min-height: 54px;
}

.project-group-card__project {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  border-radius: 6px;
  background: #f8fafc;
  padding: 6px 8px;
}

.project-group-card__project-name {
  overflow: hidden;
  color: #374151;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-group-card__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
}

html.dark .project-group-card,
[data-theme='dark'] .project-group-card {
  border-color: #374151;
  background: #111827;
}

html.dark .project-group-card__title,
[data-theme='dark'] .project-group-card__title {
  color: #e5e7eb;
}

html.dark .project-group-card__project,
[data-theme='dark'] .project-group-card__project {
  background: #1f2937;
}

html.dark .project-group-card__project-name,
[data-theme='dark'] .project-group-card__project-name {
  color: #d1d5db;
}
</style>
