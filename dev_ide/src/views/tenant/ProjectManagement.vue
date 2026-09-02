<template>
  <div class="project-management">
    <!-- ═══ 区域一：工具栏 ═══ -->
    <ProjectOverviewToolbar data-testid="project-overview-toolbar" :search="filters.search"
      :search-placeholder="t('projectManagement.searchPlaceholder')" :view-mode="viewMode" :sort-by="filters.sortBy"
      :sort-order="filters.sortOrder" :composite-filters="filters.composite" :tag-ids="filters.tagIds"
      :tag-options="projectTagOptions" :tag-max-selected="projectTagLimit" :runtime-mode-options="runtimeModeOptions"
      :deploy-status-options="deployStatusOptions" :sort-field-options="sortFieldOptions"
      :sort-order-options="sortOrderOptions" :selected-count="selectedProjects.length"
      :can-manage-projects="canManageProjects" :can-export-projects="canExportProjects"
      :can-delete-projects="canDeleteProjects" @update:search="handleOverviewSearchInput"
      @update:view-mode="handleViewModeChange" @update:sort-by="handleSortByChange"
      @update:sort-order="handleSortOrderChange" @update:composite-filters="handleCompositeFiltersChange"
      @update:tag-ids="handleTagIdsChange" @delete-tag="handleDeleteProjectTag" @tag-limit="handleProjectTagLimit"
      @create-project="showCreateDialog = true" @refresh="resetSearch"
      @open-group-manager="showGroupManageDialog = true" @import-project="handleImportProject"
      @batch-export="batchExportProjects" @batch-delete="batchDeleteProjects" @open-settings="handleReservedSettings" />

    <!-- ═══ 区域二：内容区域（撑满剩余高度） ═══ -->
    <div class="project-content-area" data-testid="project-overview-shell">
      <!-- 面包屑（进入分组后显示） -->
      <div v-if="groupContext.groupId" class="project-group-breadcrumb" data-testid="project-group-breadcrumb">
        <button type="button" class="project-group-breadcrumb__link" data-testid="project-group-breadcrumb-all"
          @click="handleGroupBreadcrumbAll">
          <el-icon style="margin-right: 4px">
            <FolderOpened />
          </el-icon>
          {{ t('projectManagement.groupBreadcrumbAll') }}
        </button>
        <el-icon class="project-group-breadcrumb__separator">
          <ArrowRight />
        </el-icon>
        <span class="project-group-breadcrumb__current">
          {{ groupContext.groupName }}
          <span class="project-group-breadcrumb__count">({{ pagination.total }})</span>
        </span>
        <el-button v-if="canManageProjects" size="small" type="primary" plain data-testid="project-group-add-project"
          class="project-group-breadcrumb__action" @click="
            openGroupProjectPicker({ id: groupContext.groupId, name: groupContext.groupName })
            ">
          {{ t('projectManagement.addProjectToGroup') }}
        </el-button>
      </div>

      <!-- 可滚动卡片/表格区域 -->
      <div class="project-content-scroll">
        <ProjectOverviewGrid v-if="viewMode === 'card'" :projects="projectList" :loading="loading"
          :empty-description="t('projectManagement.emptyProjects')" :selected-ids="selectedProjects"
          :show-selection="true" :show-member-action="canManageProjects" :show-deploy-action="canPerformOps"
          :show-export-action="canExportProjects" :show-delete-action="canDeleteProjects"
          :can-toggle-visibility="canManageProjects" :current-user-id="currentUser?.id || ''" :group-cards="!groupContext.groupId && viewMode === 'card' ? resolvedProjectGroupCards : []
            " :grouped="!groupContext.groupId" :show-grouped-project-items="false" @open-project="handleOpenProject"
          @selection-change="handleProjectSelectionChange" @open-runtime-access="openRuntimeAccessDialog"
          @deploy="openDeployDialog" @export="handleExportProject" @delete="deleteProject"
          @visibility-toggle="toggleProjectVisibility" @group-select="handleGroupCardSelect"
          @group-add-project="openGroupProjectPicker" @group-edit="openProjectGroupEditDialog"
          @group-delete="deleteProjectGroup" @group-remove-project="removeProjectFromGroup">
          <template #entry-actions="{ project }">
            <div class="project-entry-actions project-entry-actions--card">
              <el-button data-testid="project-ai-workbench-action" size="small"
                class="project-entry-button project-entry-button--designer" @click.stop="openDesignCenter(project)">
                <el-icon>
                  <EditPen />
                </el-icon>
                <span>{{ t('projectManagement.designCenter') }}</span>
              </el-button>
              <el-button data-testid="project-data-center-action" size="small"
                class="project-entry-button project-entry-button--datacenter" @click.stop="openDataCenter(project)">
                <el-icon>
                  <DataAnalysis />
                </el-icon>
                <span>{{ t('projectManagement.dataCenter') }}</span>
              </el-button>
            </div>
          </template>
          <template #actions="{ project }">
            <div class="project-management-actions project-management-actions--card">
              <el-tooltip v-if="canManageProjects" :content="t('projectManagement.editProject')" placement="top">
                <el-button data-testid="project-edit-action" size="small" text circle class="project-action-button"
                  @click.stop="editProject(project)">
                  <el-icon>
                    <Edit />
                  </el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="canManageProjects" :content="t('projectManagement.memberAndPermission')"
                placement="top">
                <el-button data-testid="project-runtime-access-action" size="small" text circle
                  class="project-action-button" @click.stop="openRuntimeAccessDialog(project)">
                  <el-icon>
                    <User />
                  </el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="canManageProjects" :content="t('projectManagement.editTags')" placement="top">
                <el-button data-testid="project-edit-tags-action" size="small" text circle class="project-action-button"
                  @click.stop="openProjectTagDialog(project)">
                  <el-icon>
                    <PriceTag />
                  </el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="canPerformOps" :content="t('projectManagement.openFormalDeployment')" placement="top">
                <el-button data-testid="project-deploy-action" size="small" text circle class="project-action-button"
                  @click.stop="openDeployDialog(project)">
                  <el-icon>
                    <UploadFilled />
                  </el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="canExportProjects" :content="t('projectManagement.export')" placement="top">
                <el-button data-testid="project-export-action" size="small" text circle class="project-action-button"
                  @click.stop="handleExportProject(project)">
                  <el-icon>
                    <Upload />
                  </el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="canDeleteProjects" :content="t('projectManagement.delete')" placement="top">
                <el-button data-testid="project-delete-action" size="small" text circle
                  class="project-action-button project-action-button--danger" @click.stop="deleteProject(project)">
                  <el-icon>
                    <Delete />
                  </el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </ProjectOverviewGrid>

        <ProjectOverviewTable v-else :projects="projectList"
          :group-cards="!groupContext.groupId ? resolvedProjectGroupCards : []" :loading="loading"
          :grouped="!groupContext.groupId" :show-grouped-project-items="false"
          :empty-description="t('projectManagement.emptyProjects')" :selected-ids="selectedProjects"
          :show-selection="true" :show-tag-column="false" :show-member-action="canManageProjects"
          :show-deploy-action="canPerformOps" :show-export-action="canExportProjects"
          :show-delete-action="canDeleteProjects" @open-project="handleOpenProject"
          @selection-change="handleProjectSelectionChange" @open-runtime-access="openRuntimeAccessDialog"
          @deploy="openDeployDialog" @export="handleExportProject" @delete="deleteProject"
          @group-select="handleGroupCardSelect" @group-add-project="
            (row) => openGroupProjectPicker({ id: row.groupId, name: row.groupName })
          " @group-edit="
            (row) => openProjectGroupEditDialog({ id: row.groupId, name: row.groupName })
          " @group-delete="(row) => deleteProjectGroup({ id: row.groupId, name: row.groupName })">
          <template #actions="{ project }">
            <div class="project-entry-actions project-entry-actions--table">
              <el-button data-testid="project-ai-workbench-action" size="small"
                class="project-entry-button project-entry-button--designer" @click.stop="openDesignCenter(project)">
                <el-icon>
                  <EditPen />
                </el-icon>
                <span>{{ t('projectManagement.designCenter') }}</span>
              </el-button>
              <el-button data-testid="project-data-center-action" size="small"
                class="project-entry-button project-entry-button--datacenter" @click.stop="openDataCenter(project)">
                <el-icon>
                  <DataAnalysis />
                </el-icon>
                <span>{{ t('projectManagement.dataCenter') }}</span>
              </el-button>
            </div>
            <div class="project-management-actions project-management-actions--table">
              <el-tooltip v-if="canManageProjects" :content="t('projectManagement.editProject')" placement="top">
                <el-button data-testid="project-edit-action" size="small" text circle class="project-action-button"
                  @click.stop="editProject(project)">
                  <el-icon>
                    <Edit />
                  </el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="canManageProjects" :content="t('projectManagement.memberAndPermission')"
                placement="top">
                <el-button data-testid="project-runtime-access-action" size="small" text circle
                  class="project-action-button" @click.stop="openRuntimeAccessDialog(project)">
                  <el-icon>
                    <User />
                  </el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="canManageProjects" :content="t('projectManagement.editTags')" placement="top">
                <el-button data-testid="project-edit-tags-action" size="small" text circle class="project-action-button"
                  @click.stop="openProjectTagDialog(project)">
                  <el-icon>
                    <PriceTag />
                  </el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="canPerformOps" :content="t('projectManagement.openFormalDeployment')" placement="top">
                <el-button data-testid="project-deploy-action" size="small" text circle class="project-action-button"
                  @click.stop="openDeployDialog(project)">
                  <el-icon>
                    <UploadFilled />
                  </el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="canExportProjects" :content="t('projectManagement.export')" placement="top">
                <el-button data-testid="project-export-action" size="small" text circle class="project-action-button"
                  @click.stop="handleExportProject(project)">
                  <el-icon>
                    <Upload />
                  </el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="canDeleteProjects" :content="t('projectManagement.delete')" placement="top">
                <el-button data-testid="project-delete-action" size="small" text circle
                  class="project-action-button project-action-button--danger" @click.stop="deleteProject(project)">
                  <el-icon>
                    <Delete />
                  </el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </ProjectOverviewTable>
      </div>

      <!-- 分页固定底部 -->
      <div v-if="pagination.total > 0" class="pagination-bar" data-testid="project-overview-pagination">
        <ProjectOverviewPagination :page="pagination.page" :limit="pagination.limit" :total="pagination.total"
          :total-pages="pagination.totalPages" :summary="pagination.summary" @change="handleOverviewPaginationChange" />
      </div>
    </div>

    <!-- 创建工程对话框 -->
    <el-dialog v-model="showCreateDialog" :title="t('projectManagement.createDialog')" width="600px"
      :close-on-click-modal="true">
      <el-form ref="createFormRef" :model="createForm" :rules="createFormRules" label-width="100px">
        <el-form-item :label="t('projectManagement.projectName')" prop="name">
          <el-input v-model="createForm.name" :placeholder="t('projectManagement.inputProjectName')" />
        </el-form-item>
        <el-form-item :label="t('projectManagement.description')">
          <el-input v-model="createForm.description" type="textarea"
            :placeholder="t('projectManagement.inputProjectDescription')" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="success" @click="importProject">
          <el-icon class="mr-1">
            <Download />
          </el-icon>
          {{ t('projectManagement.importAction') }}
        </el-button>
        <el-button @click="showCreateDialog = false">{{ t('projectManagement.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreateProject" :loading="createLoading">
          {{ t('projectManagement.create') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 编辑工程对话框 -->
    <el-dialog v-model="showEditDialog" :title="t('projectManagement.editDialog')" width="600px"
      :close-on-click-modal="true">
      <el-form ref="editFormRef" :model="editForm" :rules="editFormRules" label-width="100px">
        <el-form-item :label="t('projectManagement.projectName')" prop="name">
          <el-input v-model="editForm.name" :placeholder="t('projectManagement.inputProjectName')" />
        </el-form-item>
        <el-form-item :label="t('projectManagement.description')">
          <el-input v-model="editForm.description" type="textarea"
            :placeholder="t('projectManagement.inputProjectDescription')" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">{{ t('projectManagement.cancel') }}</el-button>
        <el-button type="primary" @click="handleUpdateProject" :loading="editLoading">
          {{ t('projectManagement.save') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 编辑工程标签对话框 -->
    <el-dialog v-model="showTagEditDialog" :title="`${t('projectManagement.editTags')} - ${tagEditProject?.name || ''}`"
      width="520px" :close-on-click-modal="true">
      <el-form label-width="96px">
        <el-form-item :label="t('projectManagement.createTag')">
          <div class="project-tag-creator">
            <el-input v-model="tagCreateName" data-testid="project-tag-name-input"
              :placeholder="t('projectManagement.inputTagName')" clearable @keyup.enter="handleCreateProjectTag" />
            <el-tooltip :disabled="projectTagOptions.length < projectTagLimit"
              :content="t('projectManagement.tagLimitReached', { count: projectTagLimit })" placement="top">
              <span class="project-tag-creator__create-wrapper">
                <el-button type="primary" plain data-testid="project-tag-create-trigger" :loading="tagCreateLoading"
                  :disabled="projectTagOptions.length >= projectTagLimit" @click="handleCreateProjectTag">
                  {{ t('projectManagement.createTag') }}
                </el-button>
              </span>
            </el-tooltip>
          </div>
        </el-form-item>
        <el-form-item :label="t('projectManagement.tagFilter')">
          <el-select v-model="tagEditIds" class="project-tag-editor__select" multiple filterable clearable
            :placeholder="t('projectManagement.tagSearchPlaceholder')" :multiple-limit="projectTagLimit"
            style="width: 100%" @change="handleTagEditIdsChange">
            <el-option v-for="tag in projectTagOptions" :key="tag.id" :label="tag.name" :value="tag.id">
              <div class="project-tag-option">
                <span class="project-tag-option__name">{{ tag.name }}</span>
                <button type="button" class="project-tag-option__delete"
                  :aria-label="`${t('projectManagement.deleteTag')} ${tag.name}`"
                  @click.stop.prevent="handleDeleteProjectTag(tag)">
                  <el-icon>
                    <Delete />
                  </el-icon>
                </button>
              </div>
            </el-option>
          </el-select>
          <div v-if="projectTagOptions.length === 0" class="text-xs text-gray-400 mt-2">
            {{ t('projectManagement.noTagOptions') }}
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showTagEditDialog = false">{{
          t('projectManagement.cancel')
          }}</el-button>
        <el-button type="primary" :loading="tagEditLoading" @click="handleUpdateProjectTags">
          {{ t('projectManagement.save') }}
        </el-button>
      </template>
    </el-dialog>

    <ProjectPublishDialog v-model:visible="showDeployDialog" :project="deployForm.project"
      @confirm="handlePublishConfirm" />

    <!-- 设计中心和数据中心通过工程卡片/表格行的直接入口打开。 -->
    <template v-if="false">
      <!-- 设计中心卡片 -->
      <div
        class="function-card bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-900/20 dark:to-blue-800/20 border-2 border-blue-200 dark:border-blue-700 rounded-xl p-6 cursor-pointer hover:shadow-lg hover:border-blue-300 dark:hover:border-blue-600 transition-shadow duration-200"
        @click="openDesignCenter(selectedProject)">
        <div class="text-center">
          <!-- 图标 -->
          <div class="inline-flex items-center justify-center w-16 h-16 bg-blue-500 rounded-full mb-4">
            <svg class="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zM21 5a2 2 0 00-2-2h-4a2 2 0 00-2 2v12a4 4 0 004 4h4a2 2 0 002-2V5z" />
            </svg>
          </div>

          <!-- 标题 -->
          <h4 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
            {{ t('projectManagement.designCenter') }}
          </h4>

          <!-- 描述 -->
          <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
            {{ t('projectManagement.designCenterDesc') }}
          </p>

          <!-- 统计信息 -->
          <div class="flex justify-center space-x-4 text-xs text-gray-500 dark:text-gray-400">
            <span class="flex items-center">
              <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              {{
                t('projectManagement.pageCount', {
                  count: selectedProject?.pageCount || 0,
                })
              }}
            </span>
            <span class="flex items-center">
              <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M7 4V2a1 1 0 011-1h8a1 1 0 011 1v2m-9 0h10m-9 0V1m10 3V1m0 3l1 1v16a2 2 0 01-2 2H6a2 2 0 01-2-2V5l1-1z" />
              </svg>
              {{
                t('projectManagement.componentCount', {
                  count: selectedProject?.componentCount || 0,
                })
              }}
            </span>
          </div>

          <!-- 操作提示 -->
          <div class="mt-4 text-xs text-blue-600 dark:text-blue-400 font-medium">
            {{ t('projectManagement.openDesignCenter') }}
          </div>
        </div>
      </div>

      <!-- 数据中心卡片 -->
      <div
        class="function-card bg-gradient-to-br from-green-50 to-green-100 dark:from-green-900/20 dark:to-green-800/20 border-2 border-green-200 dark:border-green-700 rounded-xl p-6 cursor-pointer hover:shadow-lg hover:border-green-300 dark:hover:border-green-600 transition-shadow duration-200"
        @click="openDataCenter(selectedProject)">
        <div class="text-center">
          <!-- 图标 -->
          <div class="inline-flex items-center justify-center w-16 h-16 bg-green-500 rounded-full mb-4">
            <svg class="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
            </svg>
          </div>

          <!-- 标题 -->
          <h4 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
            {{ t('projectManagement.dataCenter') }}
          </h4>

          <!-- 描述 -->
          <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
            {{ t('projectManagement.dataCenterDesc') }}
          </p>

          <!-- 统计信息 -->
          <div class="flex justify-center space-x-4 text-xs text-gray-500 dark:text-gray-400">
            <span class="flex items-center">
              <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4" />
              </svg>
              {{
                t('projectManagement.dataSourceCount', {
                  count: selectedProject?.dataSourceCount || 0,
                })
              }}
            </span>
            <span class="flex items-center">
              <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
              </svg>
              {{
                t('projectManagement.scriptCount', {
                  count: selectedProject?.scriptCount || 0,
                })
              }}
            </span>
          </div>

          <!-- 操作提示 -->
          <div class="mt-4 text-xs text-green-600 dark:text-green-400 font-medium">
            {{ t('projectManagement.openDataCenter') }}
          </div>
        </div>
      </div>
    </template>

    <ProjectRuntimeAccessDialog v-model:visible="runtimeAccessDialogVisible" :project="runtimeAccessProject" />

    <el-dialog v-model="showGroupEditDialog" :title="t('projectManagement.editGroup')" width="480px"
      :close-on-click-modal="true">
      <el-form :model="groupEditForm" label-width="92px">
        <el-form-item :label="t('projectManagement.groupName')">
          <el-input v-model="groupEditForm.name" data-testid="project-group-edit-dialog-name-input"
            :placeholder="t('projectManagement.inputGroupName')" clearable @keyup.enter="submitProjectGroupEdit" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showGroupEditDialog = false">{{
          t('projectManagement.cancel')
          }}</el-button>
        <el-button type="primary" data-testid="project-group-edit-dialog-save" :loading="groupEditLoading"
          @click="submitProjectGroupEdit">
          {{ t('projectManagement.save') }}
        </el-button>
      </template>
    </el-dialog>

    <ProjectGroupManageDialog v-model:visible="showGroupManageDialog" :groups="resolvedProjectGroupCards"
      @create="createProjectGroup" @edit="openProjectGroupEditDialog" @delete="deleteProjectGroup" />

    <ProjectGroupProjectPickerDialog v-model:visible="groupProjectPickerVisible"
      :group-id="targetGroupForProjectPicker?.id || ''" :group-name="targetGroupForProjectPicker?.name || ''"
      :projects="groupProjectCandidates" :tag-options="projectTagOptions" :runtime-mode-options="runtimeModeOptions"
      :deploy-status-options="deployStatusOptions" :loading="groupProjectCandidatesLoading" @select="addProjectToGroup"
      @query-change="handleGroupProjectCandidateQueryChange" />
  </div>
</template>

<script lang="ts">
// @ts-nocheck
import { ref, reactive, computed, onMounted, onUnmounted, getCurrentInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import JSZip from 'jszip'
import { useAuthStore } from '@/store'
import { can } from '@/permissions'
import request, { getApiErrorMessage as resolveApiErrorMessage } from '@/utils/request'
import { projectAPI } from '@/api/project.api'
import { opsAPI } from '@/api/ops.api'
import { formatDateTime, formatDate, formatCurrency } from '@/utils'
import { initSocket, getSocket } from '@/utils/socket'
import { Storage } from '@/utils/storage'
import ProjectRuntimeAccessDialog from './components/ProjectRuntimeAccessDialog.vue'
import ProjectOverviewToolbar from './project-management/ProjectOverviewToolbar.vue'
import ProjectOverviewGrid from './project-management/ProjectOverviewGrid.vue'
import ProjectOverviewTable from './project-management/ProjectOverviewTable.vue'
import ProjectOverviewPagination from './project-management/ProjectOverviewPagination.vue'
import ProjectPublishDialog from './project-management/ProjectPublishDialog.vue'
import ProjectGroupCards from './project-management/ProjectGroupCards.vue'
import ProjectGroupManageDialog from './project-management/ProjectGroupManageDialog.vue'
import ProjectGroupProjectPickerDialog from './project-management/ProjectGroupProjectPickerDialog.vue'
import { buildProjectGroupCardItems } from './project-management/project-group-utils'
import { buildProjectOverviewQueryParams } from './project-management/use-project-filters'
import {
  applyOpsDeploymentRuntimeSummaries,
  useProjectOverviewState,
} from './project-management/use-project-overview'

const PROJECT_TAG_LIMIT = 10

export default {
  name: 'ProjectManagement',
  components: {
    ProjectRuntimeAccessDialog,
    ProjectOverviewToolbar,
    ProjectOverviewGrid,
    ProjectOverviewTable,
    ProjectOverviewPagination,
    ProjectPublishDialog,
    ProjectGroupCards,
    ProjectGroupManageDialog,
    ProjectGroupProjectPickerDialog,
  },
  setup() {
    const { t } = useI18n()
    const { emit } = getCurrentInstance()
    const authStore = useAuthStore()

    // 当前用户信息
    const currentUser = computed(() => authStore.userInfo)

    const {
      projects: projectList,
      loading,
      filters,
      pagination,
      groupContext,
      viewMode,
      patchFilters,
      setSort,
      setGroupContext,
      clearGroupContext,
      setPage,
      setLimit,
      fetchProjects: fetchProjectOverview,
    } = useProjectOverviewState({
      initialPagination: {
        page: 1,
        limit: 10,
      },
      summaryFormatter: ({ start, end, total }) =>
        t('projectManagement.pageSummary', { start, end, total }),
    })

    // 状态
    const createLoading = ref(false)
    const editLoading = ref(false)
    const batchOperationLoading = ref(false)

    // 对话框显示状态
    const showCreateDialog = ref(false)
    const showEditDialog = ref(false)
    const showDeployDialog = ref(false)
    const showGroupManageDialog = ref(false)
    const showGroupEditDialog = ref(false)
    const groupProjectPickerVisible = ref(false)
    const showTagEditDialog = ref(false)
    const projectDialogVisible = ref(false)
    const runtimeAccessDialogVisible = ref(false)

    // 选择状态
    const selectedProjects = ref([]) // 选中的工程ID列表
    const deployStateRefreshTimer = ref(null)
    const searchTimer = ref(null)
    const projectTagOptions = ref([])
    const projectTagLimit = PROJECT_TAG_LIMIT
    const projectGroupCards = ref([])
    const groupProjectCandidates = ref([])
    const groupProjectCandidatesLoading = ref(false)
    const groupProjectCandidateFilters = reactive({
      search: '',
      tagIds: [],
      runtimeModes: [],
      deployStatuses: [],
      visibility: [],
    })
    const resolvedProjectGroupCards = computed(() =>
      buildProjectGroupCardItems(projectGroupCards.value, projectList.value),
    )
    const hasProjectGroups = computed(() => resolvedProjectGroupCards.value.length > 0)

    // 当前操作的工程
    const selectedProject = ref(null)
    const runtimeAccessProject = ref(null)
    const targetGroupForProjectPicker = ref(null)
    const tagEditProject = ref(null)
    const tagEditIds = ref([])
    const tagEditLoading = ref(false)
    const tagCreateName = ref('')
    const tagCreateLoading = ref(false)
    const groupEditLoading = ref(false)

    // 部署相关状态
    const deployLoading = ref(false)
    const deployForm = reactive({
      project: null,
      currentMode: null,
      currentVersion: '',
      mode: 'RELEASE',
      version: '',
      targetNodes: [],
    })
    const projectVersions = ref([])
    const allReleaseVersions = ref([])
    const generatedReleaseVersions = ref([])
    const releaseVersionOptions = computed(() => {
      return [...generatedReleaseVersions.value, ...projectVersions.value]
    })
    const canAddReleaseVersion = computed(() => generatedReleaseVersions.value.length === 0)
    const availableNodes = ref([])
    const nodeModes = reactive({}) // { nodeId: 'DEV' | 'RELEASE' | null }
    const nodeKeyword = ref('')
    const filteredAvailableNodes = computed(() => {
      const keyword = nodeKeyword.value.trim().toLowerCase()
      return availableNodes.value.filter((node) => {
        if (!keyword) {
          return true
        }
        const nodeName = (node?.name || '').toLowerCase()
        const nodeIp = (node?.ipAddress || '').toLowerCase()
        return nodeName.includes(keyword) || nodeIp.includes(keyword)
      })
    })
    const selectedTargetNodeId = computed({
      get: () => deployForm.targetNodes[0] || null,
      set: (nodeId) => {
        deployForm.targetNodes = nodeId ? [nodeId] : []
      },
    })

    const runtimeModeOptions = computed(() => [
      { label: t('projectManagement.modeDisplayDev'), value: 'DEV' },
      { label: t('projectManagement.modeDisplayRelease'), value: 'RELEASE' },
    ])
    const deployStatusOptions = computed(() => [
      { label: t('projectManagement.deployStatusPending'), value: 'pending' },
      { label: t('projectManagement.deployStatusDeploying'), value: 'deploying' },
      { label: t('projectManagement.deployStatusRunning'), value: 'running' },
      { label: t('projectManagement.deployStatusStopped'), value: 'stopped' },
      { label: t('projectManagement.deployStatusError'), value: 'error' },
      { label: t('projectManagement.deployStatusRollback'), value: 'rollback' },
    ])
    const sortFieldOptions = computed(() => [
      { label: t('projectManagement.sortFieldCreatedAt'), value: 'createdAt' },
      { label: t('projectManagement.sortFieldUpdatedAt'), value: 'updatedAt' },
      { label: t('projectManagement.sortFieldLastDeployedAt'), value: 'lastDeployedAt' },
      { label: t('projectManagement.sortFieldRuntimeStatus'), value: 'runtimeStatus' },
    ])
    const sortOrderOptions = computed(() => [
      { label: t('projectManagement.sortOrderDesc'), value: 'DESC' },
      { label: t('projectManagement.sortOrderAsc'), value: 'ASC' },
    ])

    // 创建工程表单
    const createForm = reactive({
      name: '',
      description: '',
    })

    // 创建表单验证规则
    const createFormRules = {
      name: [
        {
          required: true,
          message: t('projectManagement.nameRequired'),
          trigger: 'blur',
        },
        {
          min: 2,
          max: 100,
          message: t('projectManagement.nameLength'),
          trigger: 'blur',
        },
      ],
    }

    // 编辑工程表单
    const editForm = reactive({
      id: '',
      name: '',
      description: '',
    })

    // 分组编辑独立使用弹窗，避免从卡片/列表入口点击后直接提交旧名称。
    const groupEditForm = reactive({
      id: '',
      name: '',
    })

    // 编辑表单验证规则
    const editFormRules = {
      name: [
        {
          required: true,
          message: t('projectManagement.nameRequired'),
          trigger: 'blur',
        },
        {
          min: 2,
          max: 100,
          message: t('projectManagement.nameLength'),
          trigger: 'blur',
        },
      ],
    }

    // 表单引用
    const createFormRef = ref(null)
    const editFormRef = ref(null)

    // 检查是否可以管理工程
    const canManageProjects = computed(() => {
      return can(currentUser.value?.role, 'project:write')
    })
    const canExportProjects = computed(() => {
      return canManageProjects.value || can(currentUser.value?.role, 'project:export')
    })
    const canDeleteProjects = computed(() => {
      return canManageProjects.value || can(currentUser.value?.role, 'project:delete')
    })
    const canAccessProjectDetail = computed(() => {
      return can(currentUser.value?.role, 'project:read')
    })
    const isOpsAdminRole = computed(() => currentUser.value?.role === 'OPS_ADMIN')

    // 检查是否可以执行运维操作
    const canPerformOps = computed(() => {
      return can(currentUser.value?.role, 'deploy:execute')
    })
    const canForceDeleteProject = computed(() =>
      can(currentUser.value?.role, 'project:forceDelete'),
    )

    /**
     * 统一提取接口错误信息，兼容 message/error 字段。
     * @param {any} error - 异常对象
     * @param {string} fallback - 兜底文案
     * @returns {string}
     */
    const getApiErrorMessage = (error, fallback) => {
      return resolveApiErrorMessage(error, fallback)
    }

    const asRecord = (value) => {
      if (value && typeof value === 'object' && !Array.isArray(value)) {
        return value
      }
      return {}
    }

    const extractCollectionItems = (response, candidateKeys = []) => {
      const root = asRecord(response)
      const payload = asRecord(root.data)
      const business = Object.keys(payload).length > 0 ? payload : root
      const list = asRecord(business.list)

      const candidates = [
        ...candidateKeys.map((key) => list[key]).filter(Array.isArray),
        ...candidateKeys.map((key) => business[key]).filter(Array.isArray),
        list.items,
        business.items,
        Array.isArray(business.list) ? business.list : null,
      ]

      for (const candidate of candidates) {
        if (Array.isArray(candidate)) {
          return candidate
        }
      }

      return []
    }

    const extractNamedItem = (response, candidateKeys = []) => {
      const root = asRecord(response)
      const payload = asRecord(root.data)
      const business = Object.keys(payload).length > 0 ? payload : root

      for (const key of candidateKeys) {
        const candidate = asRecord(business[key])
        if (Object.keys(candidate).length > 0) {
          return candidate
        }
      }

      const item = asRecord(business.item)
      return Object.keys(item).length > 0 ? item : {}
    }

    const normalizeSelectionIds = (values = []) => {
      return [...new Set(values.map((item) => String(item || '').trim()).filter(Boolean))]
    }

    const syncSelectionWithProjectList = () => {
      const currentIdSet = new Set(projectList.value.map((project) => String(project?.id || '')))
      selectedProjects.value = selectedProjects.value.filter((id) => currentIdSet.has(String(id)))
    }

    const fetchProjectTagOptions = async () => {
      const response = await projectAPI.listProjectTags()
      const rawTags = extractCollectionItems(response, ['tags'])
      projectTagOptions.value = rawTags
        .map((item) => {
          const tag = asRecord(item)
          const id = String(tag.id || '').trim()
          if (!id) {
            return null
          }
          return {
            id,
            name: String(tag.name || '').trim() || id,
            description: String(tag.description || '').trim() || null,
            sortOrder: Number.parseInt(String(tag.sortOrder || 0), 10) || 0,
          }
        })
        .filter(Boolean)
    }

    const fetchProjectGroupCards = async () => {
      const response = await projectAPI.listProjectGroups()
      const rawGroups = extractCollectionItems(response, ['groups'])
      projectGroupCards.value = rawGroups
        .map((item) => {
          const group = asRecord(item)
          const id = String(group.id || '').trim()
          if (!id) {
            return null
          }
          const countCandidates = [group.projectCount, group.totalProjects, group.count]
          let projectCount = 0
          for (const candidate of countCandidates) {
            const parsed = Number.parseInt(String(candidate ?? ''), 10)
            if (Number.isInteger(parsed) && parsed >= 0) {
              projectCount = parsed
              break
            }
          }
          return {
            id,
            name: String(group.name || '').trim() || t('projectManagement.unknownGroup'),
            description: String(group.description || '').trim() || '',
            sortOrder: Number.parseInt(String(group.sortOrder || 0), 10) || 0,
            projectCount,
          }
        })
        .filter(Boolean)
    }

    const fetchProjectOverviewMeta = async () => {
      const [tagResult, groupResult] = await Promise.allSettled([
        fetchProjectTagOptions(),
        fetchProjectGroupCards(),
      ])

      if (tagResult.status === 'rejected') {
        console.warn('获取工程标签筛选项失败:', tagResult.reason)
      }
      if (groupResult.status === 'rejected') {
        console.warn('获取工程分组入口失败:', groupResult.reason)
      }
    }

    // 获取工程列表（统一通过状态层构建查询参数）。
    const fetchProjects = async () => {
      try {
        await fetchProjectOverview()
        // 卡片的部署态只接受 ops 单槽摘要，不从工程资料或 K3s 细节推断。
        const deploymentResults = await Promise.all(
          projectList.value.map((project) =>
            opsAPI.listProjectDeployments({ page: 1, pageSize: 1, projectId: String(project.id) }),
          ),
        )
        projectList.value = applyOpsDeploymentRuntimeSummaries(
          projectList.value,
          deploymentResults.flatMap((result) => result.items),
        )
        syncSelectionWithProjectList()
      } catch (error) {
        ElMessage.error(
          t('projectManagement.fetchFailed', {
            message: getApiErrorMessage(error, t('projectManagement.fetchFailed')),
          }),
        )
      }
    }

    const executeSearch = () => {
      setPage(1)
      fetchProjects()
    }

    const handleOverviewSearchInput = (value) => {
      patchFilters({
        search: String(value || ''),
      })
      if (searchTimer.value) {
        window.clearTimeout(searchTimer.value)
      }
      searchTimer.value = window.setTimeout(() => {
        executeSearch()
      }, 300)
    }

    const handleViewModeChange = (mode) => {
      viewMode.value = mode === 'list' ? 'list' : 'card'
    }

    const handleSortByChange = (sortBy) => {
      setSort(sortBy, filters.sortOrder)
      fetchProjects()
    }

    const handleSortOrderChange = (sortOrder) => {
      setSort(filters.sortBy, sortOrder)
      fetchProjects()
    }

    const handleCompositeFiltersChange = (compositeFilters) => {
      patchFilters({ composite: compositeFilters })
      fetchProjects()
    }

    const handleTagIdsChange = (tagIds) => {
      patchFilters({ tagIds: normalizeSelectionIds(tagIds) })
      fetchProjects()
    }

    const handleProjectTagLimit = (limit = projectTagLimit) => {
      ElMessage.warning(t('projectManagement.tagLimitReached', { count: limit }))
    }

    const handleGroupCardSelect = ({ groupId, groupName }) => {
      if (groupId) {
        setGroupContext({
          groupId: String(groupId),
          groupName: String(groupName || ''),
        })
      } else {
        clearGroupContext()
      }
      fetchProjects()
    }

    const handleGroupBreadcrumbAll = () => {
      clearGroupContext()
      fetchProjects()
    }

    const resetGroupProjectCandidateFilters = () => {
      groupProjectCandidateFilters.search = ''
      groupProjectCandidateFilters.tagIds = []
      groupProjectCandidateFilters.runtimeModes = []
      groupProjectCandidateFilters.deployStatuses = []
      groupProjectCandidateFilters.visibility = []
    }

    const fetchGroupProjectCandidates = async () => {
      const query = buildProjectOverviewQueryParams({
        filters: {
          search: groupProjectCandidateFilters.search,
          tagIds: groupProjectCandidateFilters.tagIds,
          composite: {
            runtimeModes: groupProjectCandidateFilters.runtimeModes,
            deployStatuses: groupProjectCandidateFilters.deployStatuses,
            visibility: groupProjectCandidateFilters.visibility,
            createdBy: '',
          },
          sortBy: 'createdAt',
          sortOrder: 'DESC',
        },
        pagination: {
          page: 1,
          limit: 100,
        },
        groupContext: null,
      })
      groupProjectCandidatesLoading.value = true
      try {
        const response = await projectAPI.getProjects(query)
        groupProjectCandidates.value = extractCollectionItems(response, ['projects'])
      } finally {
        groupProjectCandidatesLoading.value = false
      }
    }

    const openGroupProjectPicker = async (group) => {
      const groupId = String(group?.id || '').trim()
      if (!groupId) {
        return
      }
      resetGroupProjectCandidateFilters()
      targetGroupForProjectPicker.value = {
        ...group,
        id: groupId,
        name: String(group?.name || groupId),
      }
      try {
        await fetchGroupProjectCandidates()
      } catch (error) {
        return ElMessage.error(
          getApiErrorMessage(error, t('opsManagement.operationFailedFallback')),
        )
      }
      groupProjectPickerVisible.value = true
    }

    const handleGroupProjectCandidateQueryChange = async (criteria = {}) => {
      groupProjectCandidateFilters.search = String(criteria.search || '').trim()
      groupProjectCandidateFilters.tagIds = normalizeSelectionIds(criteria.tagIds || [])
      groupProjectCandidateFilters.runtimeModes = normalizeSelectionIds(criteria.runtimeModes || [])
      groupProjectCandidateFilters.deployStatuses = normalizeSelectionIds(
        criteria.deployStatuses || [],
      )

      try {
        await fetchGroupProjectCandidates()
      } catch (error) {
        ElMessage.error(getApiErrorMessage(error, t('opsManagement.operationFailedFallback')))
      }
    }

    const addProjectToGroup = async (projectIds) => {
      const normalizedProjectIds = normalizeSelectionIds(
        Array.isArray(projectIds) ? projectIds : [projectIds],
      )
      const targetGroupId = String(targetGroupForProjectPicker.value?.id || '').trim()
      if (normalizedProjectIds.length === 0 || !targetGroupId) {
        return
      }

      try {
        for (const projectId of normalizedProjectIds) {
          await projectAPI.bindProjectGroup(projectId, { groupId: targetGroupId })
        }
      } catch (error) {
        return ElMessage.error(
          getApiErrorMessage(error, t('opsManagement.operationFailedFallback')),
        )
      }

      ElMessage.success(
        normalizedProjectIds.length > 1
          ? t('projectManagement.groupBindBatchSuccess', { count: normalizedProjectIds.length })
          : t('projectManagement.groupBindSuccess'),
      )
      groupProjectPickerVisible.value = false
      await fetchProjects()
      await refreshProjectGroupsAfterMutation()
    }

    const removeProjectFromGroup = async ({ projectId }) => {
      const normalizedProjectId = String(projectId || '').trim()
      if (!normalizedProjectId) {
        return
      }

      try {
        await projectAPI.bindProjectGroup(normalizedProjectId, { groupId: null })
      } catch (error) {
        return ElMessage.error(
          getApiErrorMessage(error, t('opsManagement.operationFailedFallback')),
        )
      }

      ElMessage.success(t('projectManagement.groupUnbindSuccess'))
      await fetchProjects()
      await refreshProjectGroupsAfterMutation()
    }

    const refreshProjectGroupsAfterMutation = async () => {
      try {
        await fetchProjectGroupCards()
      } catch (error) {
        console.warn('刷新工程分组列表失败:', error)
      }
    }

    const createProjectGroup = async ({ name }) => {
      const trimmedName = String(name || '').trim()
      if (!trimmedName) {
        return ElMessage.warning(t('projectManagement.groupNameRequired'))
      }

      try {
        await projectAPI.createProjectGroup({ name: trimmedName })
      } catch (error) {
        return ElMessage.error(
          getApiErrorMessage(error, t('opsManagement.operationFailedFallback')),
        )
      }

      ElMessage.success(t('projectManagement.groupCreateSuccess'))
      await refreshProjectGroupsAfterMutation()
    }

    const openProjectGroupEditDialog = (group) => {
      const groupId = String(group?.id || '').trim()
      if (!groupId) {
        return
      }
      groupEditForm.id = groupId
      groupEditForm.name = String(group?.name || '').trim()
      showGroupEditDialog.value = true
    }

    const submitProjectGroupEdit = async () => {
      const groupId = String(groupEditForm.id || '').trim()
      const trimmedName = String(groupEditForm.name || '').trim()
      if (!groupId) {
        showGroupEditDialog.value = false
        return
      }
      if (!trimmedName) {
        return ElMessage.warning(t('projectManagement.groupNameRequired'))
      }

      groupEditLoading.value = true
      try {
        await projectAPI.updateProjectGroup(groupId, { name: trimmedName })
      } catch (error) {
        return ElMessage.error(
          getApiErrorMessage(error, t('opsManagement.operationFailedFallback')),
        )
      } finally {
        groupEditLoading.value = false
      }

      ElMessage.success(t('projectManagement.groupUpdateSuccess'))
      showGroupEditDialog.value = false
      await refreshProjectGroupsAfterMutation()
    }

    const deleteProjectGroup = async (group) => {
      const groupId = String(group?.id || '').trim()
      if (!groupId) {
        return
      }

      try {
        await ElMessageBox.confirm(
          `${t('projectManagement.deleteGroup')}：${group?.name || groupId}`,
          t('projectManagement.deleteConfirmTitle'),
          {
            type: 'warning',
            confirmButtonText: t('projectManagement.deleteConfirmButton'),
            cancelButtonText: t('projectManagement.cancel'),
          },
        )
        await projectAPI.deleteProjectGroup(groupId)
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(getApiErrorMessage(error, t('opsManagement.operationFailedFallback')))
        }
        return
      }

      ElMessage.success(t('projectManagement.groupDeleteSuccess'))
      if (String(groupContext.groupId || '') === groupId) {
        clearGroupContext()
        try {
          await fetchProjects()
        } catch (error) {
          console.warn('刷新工程列表失败:', error)
        }
      }
      await refreshProjectGroupsAfterMutation()
    }

    const resetSearch = () => {
      if (searchTimer.value) {
        window.clearTimeout(searchTimer.value)
      }
      patchFilters({
        search: '',
        composite: {
          runtimeModes: [],
          deployStatuses: [],
          visibility: [],
          createdBy: '',
        },
        tagIds: [],
      })
      clearGroupContext()
      fetchProjects()
    }

    const handleReservedSettings = () => {
      ElMessage.info(t('projectManagement.settings'))
    }

    const handleOverviewPaginationChange = ({ page, limit }) => {
      if (Number.parseInt(String(limit), 10) !== pagination.limit) {
        setLimit(limit)
        setPage(page)
      } else {
        setPage(page)
      }
      fetchProjects()
    }

    const handleProjectSelectionChange = ({ projectId, selected }) => {
      toggleProjectSelection(projectId, selected)
    }

    // 创建工程
    const handleCreateProject = async () => {
      if (!createFormRef.value) return

      try {
        await createFormRef.value.validate()
      } catch {
        return
      }

      createLoading.value = true
      try {
        const projectData = {
          name: createForm.name,
          description: createForm.description || '',
        }

        await projectAPI.createProject(projectData)

        ElMessage.success(t('projectManagement.createSuccess'))
        showCreateDialog.value = false
        resetCreateForm()
        await fetchProjects()
        await fetchProjectGroupCards()
      } catch (error) {
        ElMessage.error(
          t('projectManagement.createFailed', {
            message: getApiErrorMessage(error, t('projectManagement.createFailed')),
          }),
        )
      } finally {
        createLoading.value = false
      }
    }

    // 重置创建表单
    const resetCreateForm = () => {
      Object.keys(createForm).forEach((key) => {
        createForm[key] = ''
      })
      if (createFormRef.value) {
        createFormRef.value.clearValidate()
      }
    }

    // 导入工程
    const importProject = () => {
      handleImportProject()
    }

    // 导出工程
    const exportProject = (project) => {
      if (!project) return
      handleExportProject(project)
    }

    // 编辑工程
    const editProject = (project) => {
      editForm.id = project.id
      editForm.name = project.name
      editForm.description = project.description
      showEditDialog.value = true
    }

    // 更新工程
    const handleUpdateProject = async () => {
      if (!editFormRef.value) return

      try {
        await editFormRef.value.validate()
      } catch {
        return
      }

      editLoading.value = true
      try {
        const projectData = {
          name: editForm.name,
          description: editForm.description || '',
        }

        await projectAPI.updateProject(editForm.id, projectData)

        ElMessage.success(t('projectManagement.updateSuccess'))
        showEditDialog.value = false
        fetchProjects()
      } catch (error) {
        ElMessage.error(
          t('projectManagement.updateDevFailed', {
            message: getApiErrorMessage(error, t('projectManagement.updateDevFailed')),
          }),
        )
      } finally {
        editLoading.value = false
      }
    }

    // 单删与批删共用同一删除决策路径，保持影响评估、强删保护与成功/失败处理一致。
    const executeProjectDelete = async (
      project,
      options = {
        skipDefaultConfirm: false,
        showSuccessMessage: true,
      },
    ) => {
      if (!project?.id) {
        return { status: 'skipped' }
      }

      let impactData = null
      try {
        const impactRes = await projectAPI.getDeleteImpact(project.id)
        impactData = impactRes?.data ?? impactRes
      } catch {
        ElMessage.warning(t('projectManagement.deleteImpactLoadFailed'))
      }

      try {
        if (impactData?.hasActiveDeployments) {
          if (!canForceDeleteProject.value) {
            ElMessage.warning(
              t('projectManagement.deleteBlockedByActiveDeployments', {
                count: impactData?.activeDeploymentCount || 0,
              }),
            )
            return { status: 'blocked' }
          }

          await ElMessageBox.confirm(
            t('projectManagement.forceDeleteConfirmWithImpact', {
              name: project.name,
              deploymentCount: impactData?.activeDeploymentCount || 0,
              nodeCount: impactData?.activeNodeCount || 0,
            }),
            t('projectManagement.forceDeleteTitle'),
            {
              confirmButtonText: t('projectManagement.forceDeleteButton'),
              cancelButtonText: t('projectManagement.cancel'),
              type: 'warning',
            },
          )

          await projectAPI.deleteProject(project.id, { force: true })
          if (options.showSuccessMessage) {
            ElMessage.success(t('projectManagement.forceDeleteSuccess'))
          }
          return { status: 'success', forced: true }
        }

        if (!options.skipDefaultConfirm) {
          await ElMessageBox.confirm(
            t('projectManagement.deleteConfirmWithImpact', {
              name: project.name,
              deploymentCount: impactData?.totalDeploymentCount || 0,
            }),
            t('projectManagement.deleteConfirmTitle'),
            {
              confirmButtonText: t('projectManagement.deleteConfirmButton'),
              cancelButtonText: t('projectManagement.cancel'),
              type: 'warning',
            },
          )
        }

        await projectAPI.deleteProject(project.id)
        if (options.showSuccessMessage) {
          ElMessage.success(t('projectManagement.deleteSuccess'))
        }
        return { status: 'success', forced: false }
      } catch (error) {
        if (error === 'cancel') {
          return { status: 'cancelled' }
        }
        ElMessage.error(
          t('projectManagement.deleteFailed', {
            message: getApiErrorMessage(error, t('projectManagement.deleteFailed')),
          }),
        )
        return { status: 'failed' }
      }
    }

    // 删除工程
    const deleteProject = async (project) => {
      if (!canDeleteProjects.value) {
        ElMessage.warning(t('projectManagement.noPermission'))
        return
      }

      const result = await executeProjectDelete(project)
      if (result.status === 'success') {
        await fetchProjects()
        await fetchProjectGroupCards()
      }
    }

    // 导出工程
    const handleExportProject = async (project) => {
      if (!project?.id) return
      if (!canExportProjects.value) {
        ElMessage.warning(t('projectManagement.noPermission'))
        return
      }
      try {
        const response = await projectAPI.exportProject(project.id)
        const blob =
          response instanceof window.Blob
            ? response
            : new window.Blob([response], { type: 'application/zip' })
        const url = URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.download = `${project.name || 'project'}.zip`
        link.click()
        URL.revokeObjectURL(url)
        ElMessage.success(t('projectManagement.exportSuccess'))
      } catch (error) {
        ElMessage.error(
          t('projectManagement.exportFailed', {
            message: getApiErrorMessage(error, t('projectManagement.exportFailed')),
          }),
        )
      }
    }

    // 导入工程
    const handleImportProject = () => {
      const input = document.createElement('input')
      input.type = 'file'
      input.accept = '.json,.zip,application/json,application/zip'
      input.onchange = async (event) => {
        const file = event.target.files?.[0]
        if (!file) return
        try {
          let payload = null
          if (file.name.toLowerCase().endsWith('.zip')) {
            const zip = await JSZip.loadAsync(file)
            const readJson = async (path) => {
              const entry = zip.file(path)
              if (!entry) return null
              const content = await entry.async('string')
              return JSON.parse(content)
            }

            const projectJson = await readJson('project.json')
            if (!projectJson?.project) {
              ElMessage.error(t('projectManagement.importMissingManifest'))
              return
            }
            const globalVariables = await readJson('designer/global-variables.json')
            const globalScripts = await readJson('designer/global-scripts.json')
            const projectVariables = await readJson('designer/project-variables.json')
            const pageIndex = await readJson('designer/pages/index.json')
            const pages = []
            if (Array.isArray(pageIndex)) {
              for (const item of pageIndex) {
                if (!item?.file) continue
                const schema = await readJson(`designer/pages/${item.file}`)
                pages.push({
                  page: {
                    id: item.id,
                    name: item.name,
                    type: item.type,
                    parentId: item.parentId ?? null,
                    sortOrder: item.sortOrder ?? 0,
                  },
                  schemaContent: schema,
                })
              }
            }

            const datacenter = {
              connections: await readJson('datacenter/connections.json'),
              relationalConfigs: await readJson('datacenter/relational-configs.json'),
              queries: await readJson('datacenter/queries.json'),
              mqttConfigs: await readJson('datacenter/mqtt-configs.json'),
              mqttSubscriptions: await readJson('datacenter/mqtt-subscriptions.json'),
              mqttTagGroups: await readJson('datacenter/mqtt-tag-groups.json'),
              mqttTags: await readJson('datacenter/mqtt-tags.json'),
              datapoints: await readJson('datacenter/datapoints.json'),
            }

            payload = {
              project: projectJson.project,
              entryConfig: projectJson.entryConfig || {},
              settings: {
                globalVariables: globalVariables || {},
                globalScripts: globalScripts || {},
                projectVariables: projectVariables || {},
              },
              pages,
              datacenter,
            }
          } else {
            const text = await file.text()
            const parsed = JSON.parse(text)
            payload = parsed?.payload || parsed
          }

          if (!payload) {
            ElMessage.error(t('projectManagement.importInvalidFormat'))
            return
          }
          await projectAPI.importProject({ payload })
          ElMessage.success(t('projectManagement.importSuccess'))
          await fetchProjects()
          await fetchProjectGroupCards()
        } catch (error) {
          ElMessage.error(
            t('projectManagement.importFailed', {
              message: getApiErrorMessage(error, t('projectManagement.importFailed')),
            }),
          )
        }
      }
      input.click()
    }

    // Grid/Table 统一 open-project 入口，避免页面层分叉处理。
    const handleOpenProject = (project) => {
      // 卡片主体和工程名称仅用于浏览，不触发业务跳转；入口按钮负责打开子应用。
      return project
    }

    // 切换工程选中状态
    const toggleProjectSelection = (projectId, value = null) => {
      const normalizedId = String(projectId || '')
      const index = selectedProjects.value.indexOf(normalizedId)
      if (value === true || (value === null && index === -1)) {
        // 选中
        if (index === -1) {
          selectedProjects.value.push(normalizedId)
        }
      } else if (value === false || (value === null && index !== -1)) {
        // 取消选中
        if (index !== -1) {
          selectedProjects.value.splice(index, 1)
        }
      }
    }

    // 批量导出工程
    const batchExportProjects = async () => {
      if (selectedProjects.value.length === 0) {
        return ElMessage.warning(t('projectManagement.selectForExport'))
      }

      batchOperationLoading.value = true
      try {
        const zip = new JSZip()
        let successCount = 0
        let failCount = 0

        for (const projectId of selectedProjects.value) {
          const targetProject = projectList.value.find((item) => item.id === projectId)
          if (!targetProject) {
            failCount++
            continue
          }

          try {
            const response = await projectAPI.exportProject(projectId)
            const blob =
              response instanceof window.Blob
                ? response
                : new window.Blob([response], { type: 'application/zip' })
            const arrayBuffer = await blob.arrayBuffer()
            zip.file(`${targetProject.name || projectId}.zip`, arrayBuffer)
            successCount++
          } catch (error) {
            console.error(`导出工程 ${projectId} 失败:`, error)
            failCount++
          }
        }

        if (successCount === 0) {
          ElMessage.error(t('projectManagement.batchExportFailed'))
          return
        }

        const packageBlob = await zip.generateAsync({ type: 'blob' })
        const url = URL.createObjectURL(packageBlob)
        const link = document.createElement('a')
        link.href = url
        link.download = `projects-export-${Date.now()}.zip`
        link.click()
        URL.revokeObjectURL(url)

        if (failCount > 0) {
          ElMessage.warning(
            t('projectManagement.batchExportSummary', {
              success: successCount,
              fail: failCount,
            }),
          )
        } else {
          ElMessage.success(
            t('projectManagement.batchExportAllSuccess', {
              success: successCount,
            }),
          )
        }
      } catch (error) {
        ElMessage.error(
          t('projectManagement.batchExportError', {
            message: getApiErrorMessage(error, t('projectManagement.batchExportError')),
          }),
        )
      } finally {
        batchOperationLoading.value = false
      }
    }

    // 批量删除工程
    const batchDeleteProjects = async () => {
      if (!canDeleteProjects.value) {
        ElMessage.warning(t('projectManagement.noPermission'))
        return
      }
      if (selectedProjects.value.length === 0) {
        return ElMessage.warning(t('projectManagement.selectForDelete'))
      }
      batchOperationLoading.value = true
      try {
        await ElMessageBox.confirm(
          t('projectManagement.batchDeleteConfirm', {
            count: selectedProjects.value.length,
          }),
          t('projectManagement.batchDeleteTitle'),
          {
            confirmButtonText: t('projectManagement.deleteConfirmButton'),
            cancelButtonText: t('projectManagement.cancel'),
            type: 'warning',
          },
        )

        // 批量删除沿用单删同一条决策路径，但批量总确认只保留一次。
        let successCount = 0
        let failCount = 0
        for (const projectId of selectedProjects.value) {
          const targetProject = projectList.value.find((item) => item.id === projectId)
          if (!targetProject) {
            failCount++
            continue
          }
          try {
            const result = await executeProjectDelete(targetProject, {
              skipDefaultConfirm: true,
              showSuccessMessage: false,
            })
            if (result.status === 'success') {
              successCount++
            } else if (result.status !== 'cancelled') {
              failCount++
            }
          } catch (error) {
            console.error(`删除工程 ${projectId} 失败:`, error)
            failCount++
          }
        }

        if (successCount > 0) {
          ElMessage.success(
            t('projectManagement.batchDeleteSuccess', {
              success: successCount,
            }),
          )
        }
        if (failCount > 0) {
          ElMessage.warning(
            t('projectManagement.batchDeleteFail', {
              fail: failCount,
            }),
          )
        }

        // 清除选择并刷新列表
        selectedProjects.value = []
        await fetchProjects()
        await fetchProjectGroupCards()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(
            t('projectManagement.batchDeleteError', {
              message: getApiErrorMessage(error, t('projectManagement.batchDeleteError')),
            }),
          )
        }
      } finally {
        batchOperationLoading.value = false
      }
    }

    // 打开运行态成员与权限对话框。
    const openRuntimeAccessDialog = (project) => {
      runtimeAccessProject.value = project
      runtimeAccessDialogVisible.value = true
    }

    const openProjectTagDialog = (project) => {
      tagEditProject.value = project
      tagEditIds.value = Array.isArray(project?.tags)
        ? project.tags.map((tag) => String(tag?.id || '')).filter(Boolean)
        : []
      tagEditIds.value = normalizeSelectionIds(tagEditIds.value).slice(0, projectTagLimit)
      tagCreateName.value = ''
      showTagEditDialog.value = true
    }

    const handleTagEditIdsChange = (tagIds) => {
      const normalized = normalizeSelectionIds(tagIds)
      if (normalized.length > projectTagLimit) {
        tagEditIds.value = normalized.slice(0, projectTagLimit)
        handleProjectTagLimit(projectTagLimit)
        return
      }
      tagEditIds.value = normalized
    }

    const handleCreateProjectTag = async () => {
      const name = String(tagCreateName.value || '').trim()
      if (!name) {
        return ElMessage.warning(t('projectManagement.tagNameRequired'))
      }
      if (projectTagOptions.value.length >= projectTagLimit) {
        return handleProjectTagLimit(projectTagLimit)
      }

      tagCreateLoading.value = true
      try {
        const response = await projectAPI.createProjectTag({ name })
        const createdTag = extractNamedItem(response, ['tag'])
        const createdId = String(createdTag.id || '').trim()
        const createdName = String(createdTag.name || name).trim()
        if (!createdId) {
          throw new Error(t('projectManagement.createTagMissingId'))
        }

        const existingIndex = projectTagOptions.value.findIndex(
          (tag) => String(tag?.id || '') === createdId,
        )
        const normalizedTag = {
          id: createdId,
          name: createdName || createdId,
          description: String(createdTag.description || '').trim() || null,
          sortOrder: Number.parseInt(String(createdTag.sortOrder || 0), 10) || 0,
        }
        if (existingIndex >= 0) {
          projectTagOptions.value.splice(existingIndex, 1, normalizedTag)
        } else {
          projectTagOptions.value.push(normalizedTag)
        }
        tagEditIds.value = normalizeSelectionIds([...tagEditIds.value, createdId]).slice(
          0,
          projectTagLimit,
        )
        tagCreateName.value = ''
        ElMessage.success(t('projectManagement.tagCreateSuccess'))
      } catch (error) {
        return ElMessage.error(
          getApiErrorMessage(error, t('opsManagement.operationFailedFallback')),
        )
      } finally {
        tagCreateLoading.value = false
      }
    }

    const removeDeletedTagFromProject = (project, tagId) => {
      if (!project || !Array.isArray(project.tags)) {
        return
      }
      project.tags = project.tags.filter((tag) => String(tag?.id || '') !== tagId)
    }

    const handleDeleteProjectTag = async (tag) => {
      const tagId = String(tag?.id || '').trim()
      if (!tagId) {
        return
      }
      const tagName = String(tag?.name || tagId)

      try {
        await ElMessageBox.confirm(
          t('projectManagement.deleteTagConfirm', { name: tagName }),
          t('projectManagement.deleteTag'),
          {
            confirmButtonText: t('projectManagement.delete'),
            cancelButtonText: t('projectManagement.cancel'),
            type: 'warning',
          },
        )
      } catch {
        return
      }

      try {
        await projectAPI.deleteProjectTag(tagId)
        projectTagOptions.value = projectTagOptions.value.filter(
          (item) => String(item?.id || '') !== tagId,
        )
        tagEditIds.value = normalizeSelectionIds(tagEditIds.value).filter((id) => id !== tagId)
        groupProjectCandidateFilters.tagIds = normalizeSelectionIds(
          groupProjectCandidateFilters.tagIds,
        ).filter((id) => id !== tagId)
        projectList.value.forEach((project) => removeDeletedTagFromProject(project, tagId))
        groupProjectCandidates.value.forEach((project) =>
          removeDeletedTagFromProject(project, tagId),
        )
        if (tagEditProject.value) {
          removeDeletedTagFromProject(tagEditProject.value, tagId)
        }

        const currentFilterTagIds = normalizeSelectionIds(filters.tagIds)
        const nextFilterTagIds = currentFilterTagIds.filter((id) => id !== tagId)
        if (nextFilterTagIds.length !== currentFilterTagIds.length) {
          patchFilters({ tagIds: nextFilterTagIds })
          await fetchProjects()
        }

        ElMessage.success(t('projectManagement.tagDeleteSuccess'))
      } catch (error) {
        ElMessage.error(getApiErrorMessage(error, t('opsManagement.operationFailedFallback')))
      }
    }

    const toggleProjectVisibility = async (project) => {
      if (!canManageProjects.value) {
        ElMessage.warning(t('projectManagement.noPermission'))
        return
      }
      const projectId = String(project?.id || '').trim()
      if (!projectId) {
        return
      }
      const currentVisibility = String(project?.visibility || 'private').toLowerCase()
      const nextVisibility = currentVisibility === 'internal' ? 'private' : 'internal'

      try {
        const response = await projectAPI.updateProject(projectId, { visibility: nextVisibility })
        const updatedProject = extractNamedItem(response, ['project'])
        const targetProject = projectList.value.find((item) => String(item?.id || '') === projectId)
        if (targetProject) {
          targetProject.visibility = String(updatedProject.visibility || nextVisibility)
          targetProject.updatedAt = updatedProject.updatedAt || targetProject.updatedAt
          targetProject.updatedBy = updatedProject.updatedBy || targetProject.updatedBy
          targetProject.updatedByName = updatedProject.updatedByName || targetProject.updatedByName
        }
        ElMessage.success(
          t(
            nextVisibility === 'internal'
              ? 'projectManagement.visibilitySetSharedSuccess'
              : 'projectManagement.visibilitySetPrivateSuccess',
          ),
        )
        fetchProjectGroupCards().catch((error) => {
          console.warn('切换工程共享状态后刷新分组计数失败:', error)
        })
      } catch (error) {
        ElMessage.error(getApiErrorMessage(error, t('opsManagement.operationFailedFallback')))
      }
    }

    const handleUpdateProjectTags = async () => {
      const projectId = String(tagEditProject.value?.id || '').trim()
      if (!projectId) {
        return
      }
      const nextTagIds = normalizeSelectionIds(tagEditIds.value)

      tagEditLoading.value = true
      try {
        await projectAPI.bindProjectTags(projectId, {
          tagIds: nextTagIds,
        })
      } catch (error) {
        tagEditLoading.value = false
        return ElMessage.error(
          getApiErrorMessage(error, t('opsManagement.operationFailedFallback')),
        )
      }

      const tagOptionById = new Map(
        projectTagOptions.value.map((tag) => [String(tag?.id || ''), tag]).filter(([id]) => id),
      )
      const nextTags = nextTagIds.map((id) => {
        const tag = tagOptionById.get(id)
        return {
          id,
          name: String(tag?.name || id),
          description: tag?.description ?? null,
          sortOrder: tag?.sortOrder ?? 0,
        }
      })
      const targetProject = projectList.value.find((item) => String(item?.id || '') === projectId)
      if (targetProject) {
        targetProject.tags = nextTags
      }
      if (tagEditProject.value && String(tagEditProject.value?.id || '') === projectId) {
        tagEditProject.value.tags = nextTags
      }

      ElMessage.success(t('projectManagement.updateSuccess'))
      showTagEditDialog.value = false
      const hasActiveTagFilter = normalizeSelectionIds(filters.tagIds).length > 0
      if (hasActiveTagFilter) {
        try {
          await fetchProjects()
        } catch (error) {
          console.warn('工程标签更新后刷新筛选列表失败:', error)
        }
      }
      tagEditLoading.value = false
    }

    // 打开设计中心
    const openDesignCenter = (project) => {
      projectDialogVisible.value = false
      import('@/components/WujieMicroApp.vue').then((module) => {
        const WujieMicroApp = module.default
        emit('open-tab', {
          key: `${project.id}:ai`,
          title: `${project.name} · AI开发`,
          component: WujieMicroApp,
          props: {
            appType: 'designer',
            project: project,
          },
          icon: 'design',
        })
      })
    }

    // 打开数据中心
    const openDataCenter = (project) => {
      projectDialogVisible.value = false
      import('@/components/WujieMicroApp.vue').then((module) => {
        const WujieMicroApp = module.default
        emit('open-tab', {
          key: `data-center-${project.id}`,
          titlePrefix: project.name,
          titleKey: 'projectManagement.dataCenter',
          component: WujieMicroApp,
          props: {
            appType: 'datacenter',
            project: project,
          },
          icon: 'database',
        })
      })
    }

    // 获取节点当前模式
    const getNodeMode = (nodeId) => {
      return nodeModes[nodeId] || null
    }

    // 获取模式标签类型
    const getModeTagType = (mode) => {
      if (mode === 'DEV') return 'warning'
      if (mode === 'RELEASE') return 'success'
      return 'info'
    }

    const getModeDisplayLabel = (mode) => {
      if (mode === 'DEV') return t('projectManagement.modeDisplayDev')
      if (mode === 'RELEASE') return t('projectManagement.modeDisplayRelease')
      return t('projectManagement.notDeployed')
    }

    const addReleaseVersionOption = () => {
      if (!canAddReleaseVersion.value) {
        ElMessage.warning(t('projectManagement.onlyOneGeneratedVersion'))
        return
      }
      const existing = [
        ...allReleaseVersions.value
          .filter((item) => item?.status === 'success')
          .map((item) => item?.version)
          .filter(Boolean),
        ...generatedReleaseVersions.value.map((item) => item?.version).filter(Boolean),
      ]
      const parse = (version) => {
        if (typeof version !== 'string') return null
        const match = version.trim().match(/^(\d+)\.(\d+)$/)
        if (!match) return null
        return { major: Number(match[1]), minor: Number(match[2]) }
      }
      const parsed = existing.map(parse).filter(Boolean)
      let nextVersion = '0.0'
      if (parsed.length > 0) {
        parsed.sort((a, b) => {
          if (a.major !== b.major) return b.major - a.major
          return b.minor - a.minor
        })
        const latest = parsed[0]
        let nextMajor = latest.major
        let nextMinor = latest.minor + 1
        if (nextMinor >= 10) {
          nextMajor += 1
          nextMinor = 0
        }
        nextVersion = `${nextMajor}.${nextMinor}`
      }
      const option = {
        id: `generated-${nextVersion}`,
        version: nextVersion,
        createdAt: null,
        isGenerated: true,
      }
      generatedReleaseVersions.value.unshift(option)
      deployForm.version = nextVersion
    }

    /**
     * 规范化版本号文本，避免空白字符导致匹配失败。
     * @param {string} version - 原始版本号
     * @returns {string}
     */
    const normalizeVersion = (version) => (typeof version === 'string' ? version.trim() : '')

    // 创建者展示统一使用用户名，避免显示角色/姓名造成歧义。
    const getProjectCreatorDisplay = (project) => {
      const creator = project?.creator || {}
      return (
        project?.createdByName ||
        creator.username ||
        creator.fullName ||
        project?.createdBy ||
        t('projectManagement.unknown')
      )
    }

    const scheduleDeployStateRefresh = () => {
      if (deployStateRefreshTimer.value) {
        window.clearTimeout(deployStateRefreshTimer.value)
      }
      deployStateRefreshTimer.value = window.setTimeout(() => {
        fetchProjects()
      }, 120)
    }

    const setupRealtimeUpdates = () => {
      const tenantId = Storage.getTenantId() || 'default'
      const socket = initSocket(tenantId)
      socket.on('ops:deploy:status', (data = {}) => {
        if (!data?.projectId) {
          return
        }
        scheduleDeployStateRefresh()
      })
    }

    // 发布入口留在工程卡片内；运行环境和运行引擎分布在同一弹窗完成选择。
    const openDeployDialog = (project) => {
      deployForm.project = project
      showDeployDialog.value = true
    }

    const handlePublishConfirm = async (payload) => {
      deployLoading.value = true
      try {
        await opsAPI.createProjectDeployment({
          projectId: payload.projectId,
          environmentId: payload.environmentId,
          mode: payload.mode,
          applicationVersionId: payload.applicationVersionId || undefined,
          accessPort: 17800,
          placements: Object.fromEntries(
            Object.entries(payload.placements).filter(([, nodeId]) => Boolean(nodeId)),
          ),
        })
        ElMessage.success(t('projectManagement.deploySuccess', { count: 1 }))
        showDeployDialog.value = false
        await fetchProjectOverview()
      } catch (error) {
        ElMessage.error(getApiErrorMessage(error, t('projectManagement.deployFailed')))
      } finally {
        deployLoading.value = false
      }
    }

    // 确认部署
    const confirmDeploy = async () => {
      const { project, mode, targetNodes, version } = deployForm
      const normalizedVersion = normalizeVersion(version)

      if (targetNodes.length !== 1) {
        return ElMessage.warning(t('projectManagement.selectSingleTargetNode'))
      }

      if (mode === 'RELEASE' && !normalizedVersion) {
        return ElMessage.warning(t('projectManagement.selectVersion'))
      }

      deployLoading.value = true
      try {
        if (mode === 'RELEASE') {
          // RELEASE模式：需要选择或创建版本
          let deploymentId
          let isNewlyPublishedVersion = false

          // 检查是否选择已有版本
          const existingVersion = allReleaseVersions.value.find(
            (v) => normalizeVersion(v?.version) === normalizedVersion,
          )
          if (existingVersion) {
            if (existingVersion.status === 'success') {
              deploymentId = existingVersion.id
            } else if (existingVersion.status === 'failed') {
              // 失败版本允许重试：先删除失败记录，再用相同版本重新发布。
              await request.delete(`/publish/deployment/${existingVersion.id}`)
              const retryPublishRes = await request.post(`/publish/${project.id}`, {
                version: normalizedVersion,
                name: `v${normalizedVersion}`,
                description: t('projectManagement.publishByDeployDialog'),
              })
              const retryPublishPayload = retryPublishRes?.data ?? retryPublishRes
              deploymentId = retryPublishPayload?.data?.id || retryPublishPayload?.id
              if (!deploymentId) {
                throw new Error(t('projectManagement.publishFailed'))
              }
              isNewlyPublishedVersion = true
              ElMessage.info(
                t('projectManagement.versionRetryFromFailed', {
                  version: normalizedVersion,
                }),
              )
            } else {
              // building/pending 状态不可重建，避免并发构建冲突。
              return ElMessage.warning(
                t('projectManagement.versionNotReady', {
                  version: normalizedVersion,
                }),
              )
            }
          } else {
            // 需要先发布新版本
            const publishRes = await request.post(`/publish/${project.id}`, {
              version: normalizedVersion,
              name: `v${normalizedVersion}`,
              description: t('projectManagement.publishByDeployDialog'),
            })
            const publishPayload = publishRes?.data ?? publishRes
            deploymentId = publishPayload?.data?.id || publishPayload?.id
            if (!deploymentId) {
              throw new Error(t('projectManagement.publishFailed'))
            }
            isNewlyPublishedVersion = true
          }

          // 检查是否有DEV实例需要停止
          const devNodes = targetNodes.filter((n) => getNodeMode(n) === 'DEV')
          if (devNodes.length > 0) {
            await ElMessageBox.confirm(
              t('projectManagement.switchConfirm', { count: devNodes.length }),
              t('projectManagement.switchConfirmTitle'),
              { type: 'warning' },
            )
          }

          // 部署到节点
          const deployRes = await request.post(`/deployments/${deploymentId}/deploy`, {
            nodeIds: targetNodes,
            mode: 'RELEASE',
            runtimeConfig: {},
          })
          const deployPayload = deployRes?.data ?? deployRes
          const { success: successCount = 0, failed: failCount = 0 } =
            deployPayload?.data?.summary || deployPayload?.summary || {}

          // 若本次为“新建版本后立即部署”，且全部节点部署失败，则自动回滚该发布记录。
          if (isNewlyPublishedVersion && successCount === 0 && failCount > 0) {
            try {
              await request.delete(`/publish/deployment/${deploymentId}`)
              ElMessage.warning(t('projectManagement.deployFailedAndVersionRolledBack'))
            } catch (cleanupError) {
              console.warn('部署失败后回滚发布记录失败:', cleanupError)
              ElMessage.warning(t('projectManagement.deployFailedVersionKept'))
            }
            return
          }

          if (failCount > 0) {
            ElMessage.warning(
              `${t('projectManagement.deploySuccess', { count: successCount })}，失败 ${failCount} 个节点`,
            )
          } else {
            ElMessage.success(t('projectManagement.deploySuccess', { count: successCount }))
          }
        } else {
          // DEV模式：需要先确保没有RELEASE部署
          const releaseNodes = targetNodes.filter((n) => getNodeMode(n) === 'RELEASE')
          if (releaseNodes.length > 0) {
            return ElMessage.warning(t('projectManagement.releaseConflict'))
          }
          // DEV模式幂等保护：所选节点已全部处于DEV时，不再重复下发部署。
          const alreadyDevNodes = targetNodes.filter((n) => getNodeMode(n) === 'DEV')
          if (alreadyDevNodes.length === targetNodes.length) {
            return ElMessage.warning(t('projectManagement.devAlreadyDeployed'))
          }

          // DEV模式按工程直接部署，无需发布版本
          const deployRes = await request.post(`/deployments/project/${project.id}/deploy-dev`, {
            nodeIds: targetNodes,
          })
          const deployPayload = deployRes?.data ?? deployRes
          const { success: successCount = 0, failed: failCount = 0 } =
            deployPayload?.data?.summary || deployPayload?.summary || {}
          if (failCount > 0) {
            ElMessage.warning(
              `${t('projectManagement.devDeploySuccess', { count: successCount })}，失败 ${failCount} 个节点`,
            )
          } else {
            ElMessage.success(t('projectManagement.devDeploySuccess', { count: successCount }))
          }
        }

        showDeployDialog.value = false
        await fetchProjects()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(getApiErrorMessage(error, t('opsManagement.operationFailedFallback')))
        }
      } finally {
        deployLoading.value = false
      }
    }

    // 组件挂载时获取数据
    onMounted(() => {
      fetchProjects()
      fetchProjectOverviewMeta()
      setupRealtimeUpdates()
    })

    onUnmounted(() => {
      if (searchTimer.value) {
        window.clearTimeout(searchTimer.value)
      }
      if (deployStateRefreshTimer.value) {
        window.clearTimeout(deployStateRefreshTimer.value)
        deployStateRefreshTimer.value = null
      }
      const socket = getSocket()
      if (socket) {
        socket.off('ops:deploy:status')
      }
    })

    return {
      // 状态
      loading,
      createLoading,
      editLoading,
      batchOperationLoading,
      showCreateDialog,
      showEditDialog,
      showDeployDialog,
      showGroupManageDialog,
      showGroupEditDialog,
      groupProjectPickerVisible,
      showTagEditDialog,
      runtimeAccessDialogVisible,

      // 视图与筛选状态
      viewMode,
      filters,
      groupContext,
      runtimeModeOptions,
      deployStatusOptions,
      sortFieldOptions,
      sortOrderOptions,

      // 选择状态
      selectedProjects,
      selectedProject,
      runtimeAccessProject,
      targetGroupForProjectPicker,
      tagEditProject,
      tagEditIds,
      tagEditLoading,
      tagCreateName,
      tagCreateLoading,
      projectTagLimit,
      groupEditLoading,

      // 数据
      projectList,
      pagination,
      projectTagOptions,
      projectGroupCards,
      groupProjectCandidates,
      groupProjectCandidatesLoading,
      resolvedProjectGroupCards,
      hasProjectGroups,
      currentUser,

      // 部署相关
      deployLoading,
      deployForm,
      projectVersions,
      releaseVersionOptions,
      canAddReleaseVersion,
      addReleaseVersionOption,
      availableNodes,
      nodeModes,
      nodeKeyword,
      filteredAvailableNodes,
      selectedTargetNodeId,

      // 表单
      createForm,
      createFormRules,
      editForm,
      editFormRules,
      groupEditForm,

      // 表单引用
      createFormRef,
      editFormRef,

      // 方法
      t,
      canManageProjects,
      canExportProjects,
      canDeleteProjects,
      canAccessProjectDetail,
      canPerformOps,
      canForceDeleteProject,
      fetchProjects,
      handleOverviewSearchInput,
      handleViewModeChange,
      handleSortByChange,
      handleSortOrderChange,
      handleCompositeFiltersChange,
      handleTagIdsChange,
      handleGroupCardSelect,
      handleGroupBreadcrumbAll,
      openGroupProjectPicker,
      handleGroupProjectCandidateQueryChange,
      addProjectToGroup,
      removeProjectFromGroup,
      createProjectGroup,
      openProjectGroupEditDialog,
      submitProjectGroupEdit,
      deleteProjectGroup,
      resetSearch,
      handleReservedSettings,
      handleOverviewPaginationChange,
      handleCreateProject,
      handleOpenProject,
      handleProjectSelectionChange,
      toggleProjectSelection,
      importProject,
      editProject,
      handleUpdateProject,
      toggleProjectVisibility,
      openProjectTagDialog,
      handleTagEditIdsChange,
      handleCreateProjectTag,
      handleDeleteProjectTag,
      handleProjectTagLimit,
      handleUpdateProjectTags,
      deleteProject,
      handleExportProject,
      handleImportProject,
      openRuntimeAccessDialog,
      openDesignCenter,
      openDataCenter,
      openDeployDialog,
      handlePublishConfirm,
      exportProject,
      batchExportProjects,
      batchDeleteProjects,
      confirmDeploy,
      getNodeMode,
      getModeTagType,
      getModeDisplayLabel,
      getProjectCreatorDisplay,
      formatDateTime,
      formatDate,
      formatCurrency,
    }
  },
}
</script>

<style scoped>
/* ═══ 两区域制布局 ═══ */
.project-management {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 32px);
  padding: 16px;
  gap: 12px;
}

/* ─── 区域二：内容面板 ─── */
.project-content-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-radius: var(--ck-radius-lg);
  border: 1px solid var(--ck-border);
  background: var(--ck-bg-card);
  box-shadow: var(--ck-shadow-sm);
  overflow: hidden;
}

.project-content-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

/* ─── 分页底部栏 ─── */
.pagination-bar {
  flex-shrink: 0;
  border-top: 1px solid var(--ck-border-light);
  background: var(--ck-bg-card);
  padding: 10px 16px;
}

/* ─── 面包屑：cockpit 风格 ─── */
.project-group-breadcrumb {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--ck-text-secondary);
  font-size: 13px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--ck-border-light);
  flex-shrink: 0;
}

.project-group-breadcrumb__link {
  display: inline-flex;
  align-items: center;
  border: 0;
  background: transparent;
  padding: 4px 10px;
  border-radius: var(--ck-radius-sm);
  color: var(--ck-primary);
  cursor: pointer;
  font: inherit;
  transition: all 0.2s;
}

.project-group-breadcrumb__link:hover {
  background: var(--ck-primary-light);
  color: var(--ck-primary-hover);
}

.project-group-breadcrumb__separator {
  color: var(--ck-text-muted);
}

.project-group-breadcrumb__current {
  color: var(--ck-text-primary);
  font-weight: 600;
}

.project-group-breadcrumb__count {
  color: var(--ck-text-muted);
  font-weight: 400;
  margin-left: 4px;
}

.project-group-breadcrumb__action {
  margin-left: 6px;
}

.project-action-button {
  width: 28px !important;
  height: 28px !important;
  min-width: 28px !important;
  color: var(--ck-text-muted);
  border-radius: 6px;
  background: transparent;
  margin: 0 !important;
  padding: 0 !important;
  transition: all 0.2s;
}

.project-entry-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 0 0 auto;
}

.project-management-actions {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  flex: 0 0 auto;
}

.project-entry-actions--card {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  width: 100%;
  gap: 8px;
  margin-top: 8px;
}

.project-entry-actions--card .project-entry-button {
  width: 100%;
  min-width: 0;
  height: 36px;
  margin-left: 0 !important;
  padding: 0 10px !important;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
}

.project-entry-actions--card .project-entry-button :deep(.el-icon) {
  font-size: 16px;
}

.project-management-actions--card {
  width: 100%;
  justify-content: flex-end;
}

.project-entry-actions--table {
  padding-right: 8px;
  margin-right: 6px;
  border-right: 1px solid var(--ck-border-light);
}

.project-management-actions--table {
  justify-content: flex-start;
}

.project-entry-button {
  height: 28px;
  padding: 0 8px !important;
  border: 1px solid transparent !important;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  gap: 4px;
  transition: all 0.2s;
}

.project-entry-button--designer {
  color: var(--ck-primary) !important;
  background: var(--ck-primary-light) !important;
  border-color: rgba(29, 78, 216, 0.18) !important;
}

.project-entry-button--designer:hover {
  color: var(--ck-primary-hover) !important;
  border-color: rgba(29, 78, 216, 0.32) !important;
  background: rgba(29, 78, 216, 0.14) !important;
}

.project-entry-button--datacenter {
  color: var(--ck-accent) !important;
  background: rgba(14, 165, 165, 0.1) !important;
  border-color: rgba(14, 165, 165, 0.2) !important;
}

.project-entry-button--datacenter:hover {
  color: #0f766e !important;
  border-color: rgba(14, 165, 165, 0.34) !important;
  background: rgba(14, 165, 165, 0.16) !important;
}

html.dark .project-entry-actions--table,
[data-theme='dark'] .project-entry-actions--table {
  border-right-color: rgba(255, 255, 255, 0.1);
}

html.dark .project-entry-button--designer,
[data-theme='dark'] .project-entry-button--designer {
  color: #93c5fd !important;
  background: rgba(29, 78, 216, 0.2) !important;
  border-color: rgba(147, 197, 253, 0.22) !important;
}

html.dark .project-entry-button--datacenter,
[data-theme='dark'] .project-entry-button--datacenter {
  color: #5eead4 !important;
  background: rgba(14, 165, 165, 0.18) !important;
  border-color: rgba(94, 234, 212, 0.22) !important;
}

.project-action-button :deep(.el-icon) {
  font-size: 16px;
}

.project-action-button:hover {
  color: var(--ck-text-primary);
  background: rgba(0, 0, 0, 0.04);
}

.project-action-button--danger:hover {
  color: var(--ck-danger) !important;
  background: rgba(239, 68, 68, 0.1);
}

.project-tag-creator {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  width: 100%;
}

.project-tag-creator__create-wrapper {
  display: inline-flex;
}

.project-tag-editor__select :deep(.el-select__wrapper) {
  height: auto;
  min-height: 34px;
  align-items: flex-start;
  padding-top: 4px;
  padding-bottom: 4px;
}

.project-tag-editor__select :deep(.el-select__selection) {
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  max-height: 78px;
  overflow-y: auto;
}

.project-tag-editor__select :deep(.el-select__selected-item) {
  max-width: 100%;
}

.project-tag-option {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
}

.project-tag-option__name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-tag-option__delete {
  display: inline-flex;
  width: 24px;
  height: 24px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
}

.project-tag-option__delete:hover {
  background: #fef2f2;
  color: #ef4444;
}

html.dark .project-group-breadcrumb__current,
[data-theme='dark'] .project-group-breadcrumb__current {
  color: var(--ck-text-primary);
}

html.dark .pagination-bar,
[data-theme='dark'] .pagination-bar {
  border-top-color: var(--ck-border);
  background: var(--ck-bg-card);
}

html.dark .project-content-area,
[data-theme='dark'] .project-content-area {
  border-color: var(--ck-border);
  background: var(--ck-bg-card);
}

/* 卡片视图样式 */
.project-card {
  transition: all 0.2s ease-in-out;
}

.project-card:hover {
  transform: translateY(-2px);
}

/* 行截断 */
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 表格样式 */
:deep(.el-table) {
  border-radius: 8px;
}

:deep(.el-table th) {
  background-color: #f9fafb !important;
  color: #374151 !important;
  font-weight: 600;
}

:deep(.el-table td) {
  border-bottom: 1px solid #e5e7eb;
}

/* 对话框样式 */
:deep(.el-dialog) {
  border-radius: 12px;
}

:deep(.el-dialog__header) {
  background-color: #f9fafb;
  border-radius: 12px 12px 0 0;
  margin: 0;
  padding: 20px;
}

html.dark :deep(.el-dialog__header),
[data-theme='dark'] :deep(.el-dialog__header) {
  background-color: #1f2937 !important;
  border-bottom: 1px solid #374151;
}

:deep(.el-dialog__body) {
  padding: 20px;
}

:deep(.el-dialog__footer) {
  padding: 20px;
  border-top: 1px solid #e5e7eb;
}

/* 分页样式 */
:deep(.el-pagination) {
  justify-content: center;
}

/* 标签样式 */
:deep(.el-tag) {
  font-weight: 500;
}

/* 按钮样式 */
:deep(.el-button) {
  border-radius: 6px;
}

/* 表单样式 */
:deep(.el-form-item__label) {
  font-weight: 500;
}

:deep(.el-input) {
  border-radius: 6px;
}

:deep(.el-select) {
  width: 100%;
}

.deploy-node-selector {
  border: 1px solid #dfe6ef;
  border-radius: 10px;
  padding: 12px;
  background: #f8fafc;
  width: 100%;
}

.deploy-node-toolbar {
  display: flex;
  margin-bottom: 8px;
}

.deploy-node-search {
  width: 100%;
}

.deploy-node-list {
  display: block;
  width: 100%;
  max-height: clamp(220px, 38vh, 360px);
  overflow-y: auto !important;
  overflow-x: hidden;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 6px;
  background: #ffffff;
  box-sizing: border-box;
}

.deploy-node-item {
  width: 100%;
  margin: 0 0 6px 0;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: all 0.2s ease;
  box-sizing: border-box;
  border: none;
  padding: 0;
  text-align: left;
  background: transparent;
  cursor: pointer;
}

.deploy-node-item:last-child {
  margin-bottom: 0;
}

.deploy-node-indicator {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid #93c5fd;
  background: #ffffff;
  flex-shrink: 0;
  position: relative;
}

.deploy-node-indicator.selected {
  border-color: #3b82f6;
}

.deploy-node-indicator.selected::after {
  content: '';
  position: absolute;
  left: 50%;
  top: 50%;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #3b82f6;
  transform: translate(-50%, -50%);
}

.deploy-node-option {
  flex: 1;
  width: auto;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
  padding: 8px 10px;
  box-sizing: border-box;
}

.deploy-node-item:hover .deploy-node-option {
  border-color: #93c5fd;
  background: #f8fbff;
}

.deploy-node-item.is-selected .deploy-node-option {
  border-color: #3b82f6;
  background: #eff6ff;
}

.deploy-node-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(96px, 160px) auto;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.deploy-node-name {
  font-weight: 500;
  color: #1f2937;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.deploy-node-ip {
  min-width: 96px;
  font-size: 12px;
  color: #6b7280;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  text-align: right;
}

html.dark .deploy-node-selector,
[data-theme='dark'] .deploy-node-selector {
  border-color: #3b4454;
  background: #1f2937;
}

html.dark .deploy-node-list,
[data-theme='dark'] .deploy-node-list {
  border-color: #374151;
  background: #111827;
}

html.dark .deploy-node-item,
[data-theme='dark'] .deploy-node-item {
  background: transparent;
}

html.dark .deploy-node-option,
[data-theme='dark'] .deploy-node-option {
  border-color: #374151;
  background: #111827;
}

html.dark .deploy-node-indicator,
[data-theme='dark'] .deploy-node-indicator {
  background: #111827;
  border-color: #60a5fa;
}

html.dark .deploy-node-item:hover .deploy-node-option,
[data-theme='dark'] .deploy-node-item:hover .deploy-node-option {
  border-color: #60a5fa;
  background: #1e293b;
}

html.dark .deploy-node-item.is-selected .deploy-node-option,
[data-theme='dark'] .deploy-node-item.is-selected .deploy-node-option {
  border-color: #60a5fa;
  background: #1d4f91;
}

html.dark .deploy-node-name,
[data-theme='dark'] .deploy-node-name {
  color: #e5e7eb;
}

html.dark .deploy-node-ip,
[data-theme='dark'] .deploy-node-ip {
  color: #9ca3af;
}

:deep(.deploy-dialog .el-dialog) {
  margin-top: 6vh !important;
  max-height: 84vh;
  display: flex;
  flex-direction: column;
}

:deep(.deploy-dialog .el-dialog__body) {
  overflow: hidden;
}

:deep(.deploy-dialog .el-overlay-dialog) {
  overflow: hidden;
}

/* 视图切换按钮样式 */
:deep(.el-button-group .el-button) {
  border-radius: 6px;
}

/* 卡片网格布局 */
.grid {
  display: grid;
  gap: 1.5rem;
}

@media (min-width: 768px) {
  .grid-cols-2 {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1024px) {
  .grid-cols-3 {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .project-management {
    padding: 10px;
    gap: 8px;
  }

  .project-content-scroll {
    padding: 10px;
  }

  .project-tag-creator {
    grid-template-columns: 1fr;
  }

  .deploy-node-toolbar {
    flex-direction: column;
  }

  .deploy-node-list {
    max-height: clamp(180px, 32vh, 260px);
  }

  .deploy-node-row {
    grid-template-columns: minmax(0, 1fr) minmax(80px, 120px) auto;
    gap: 8px;
  }
}
</style>
