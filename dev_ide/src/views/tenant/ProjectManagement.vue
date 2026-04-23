<template>
  <div class="project-management">
    <!-- 顶部操作栏 (Cockpit-style) -->
    <div
      class="flex items-center justify-between bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-100 dark:border-gray-700 p-3 mb-6"
    >
      <!-- 左侧：搜索与视图切换 -->
      <div class="flex items-center space-x-3">
        <!-- 圆角搜索框 -->
        <el-input
          v-model="searchForm.name"
          :placeholder="t('projectManagement.searchPlaceholder')"
          clearable
          class="!w-64 rounded-full-input"
          prefix-icon="Search"
          @input="handleSearch"
        />

        <!-- 视图切换 -->
        <div
          class="flex items-center bg-gray-50 dark:bg-gray-900 rounded-full p-1 border border-gray-100 dark:border-gray-700"
        >
          <button
            @click="viewMode = 'card'"
            :class="[
              'w-8 h-7 rounded-full flex items-center justify-center transition-colors',
              viewMode === 'card'
                ? 'bg-white dark:bg-gray-600 text-blue-600 shadow-sm'
                : 'text-gray-400 hover:text-gray-600',
            ]"
          >
            <el-icon><Grid /></el-icon>
          </button>
          <button
            @click="viewMode = 'list'"
            :class="[
              'w-8 h-7 rounded-full flex items-center justify-center transition-colors',
              viewMode === 'list'
                ? 'bg-white dark:bg-gray-600 text-blue-600 shadow-sm'
                : 'text-gray-400 hover:text-gray-600',
            ]"
          >
            <svg
              class="w-4 h-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <line x1="8" y1="6" x2="21" y2="6"></line>
              <line x1="8" y1="12" x2="21" y2="12"></line>
              <line x1="8" y1="18" x2="21" y2="18"></line>
              <line x1="3" y1="6" x2="3.01" y2="6"></line>
              <line x1="3" y1="12" x2="3.01" y2="12"></line>
              <line x1="3" y1="18" x2="3.01" y2="18"></line>
            </svg>
          </button>
        </div>

        <!-- 批量操作 (有选中时显示) -->
        <div
          v-if="selectedProjects.length > 0"
          class="flex items-center space-x-2 pl-3 border-l border-gray-200 dark:border-gray-700"
        >
          <el-button
            v-if="canExportProjects"
            type="success"
            size="small"
            round
            @click="batchExportProjects"
            :disabled="selectedProjects.length === 0"
            :loading="batchOperationLoading"
          >
            <el-icon class="mr-1"><Download /></el-icon>
            {{ t('projectManagement.batchExport') }} ({{ selectedProjects.length }})
          </el-button>
          <el-button
            v-if="canDeleteProjects"
            type="danger"
            size="small"
            round
            @click="batchDeleteProjects"
            :disabled="selectedProjects.length === 0"
            :loading="batchOperationLoading"
          >
            <el-icon class="mr-1"><Delete /></el-icon>
            {{ t('projectManagement.batchDelete') }} ({{ selectedProjects.length }})
          </el-button>
        </div>
      </div>

      <!-- 右侧：全局操作按钮组 -->
      <div class="flex items-center space-x-3">
        <el-tooltip
          :content="t('projectManagement.addProject')"
          placement="top"
          v-if="canManageProjects"
        >
          <button
            class="w-9 h-9 rounded-full bg-blue-500 hover:bg-blue-600 text-white flex items-center justify-center transition-colors shadow-sm"
            @click="showCreateDialog = true"
          >
            <el-icon><Plus /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip
          :content="t('projectManagement.importProject')"
          placement="top"
          v-if="canManageProjects"
        >
          <button
            class="w-9 h-9 rounded-full bg-gray-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 flex items-center justify-center text-gray-600 dark:text-gray-300 hover:bg-white dark:hover:bg-gray-600 transition-colors shadow-sm"
            @click="handleImportProject"
          >
            <el-icon><Upload /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip :content="t('common.refresh')" placement="top">
          <button
            class="w-9 h-9 rounded-full bg-gray-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 flex items-center justify-center text-gray-600 dark:text-gray-300 hover:bg-white dark:hover:bg-gray-600 transition-colors shadow-sm"
            @click="resetSearch"
          >
            <el-icon><RefreshRight /></el-icon>
          </button>
        </el-tooltip>
      </div>
    </div>

    <!-- 工程列表 -->
    <div
      :class="
        viewMode === 'list'
          ? 'bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden'
          : ''
      "
    >
      <!-- 卡片视图 (Cockpit-style) -->
      <div v-if="viewMode === 'card'" class="py-2">
        <div v-if="projectList.length === 0 && !loading" class="text-center py-12">
          <el-empty :description="t('projectManagement.emptyProjects')" />
        </div>
        <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-6">
          <div
            v-for="project in projectList"
            :key="project.id"
            class="project-card relative bg-white dark:bg-gray-800 rounded-2xl p-5 transition-all duration-300 cursor-pointer flex flex-col border border-gray-200/80 dark:border-gray-700 shadow-[0_2px_12px_rgba(0,0,0,0.04)] hover:shadow-[0_8px_24px_rgba(0,0,0,0.08)] hover:-translate-y-0.5"
            :class="{
              'ring-2 ring-blue-500 border-blue-500': selectedProjects.includes(project.id),
              'opacity-60': selectionMode && !selectedProjects.includes(project.id),
            }"
            @click="handleCardClick(project)"
          >
            <!-- 卡片头部 (标题 + Tag) -->
            <div class="flex items-center justify-between mb-4">
              <div class="flex items-center gap-2.5 overflow-hidden flex-1">
                <el-checkbox
                  :model-value="selectedProjects.includes(project.id)"
                  @change="(val) => toggleProjectSelection(project.id, val)"
                  size="large"
                  @click.stop
                  class="!mr-0"
                />
                <h3
                  class="text-[16px] font-semibold text-gray-800 dark:text-gray-100 truncate flex-1"
                  :title="project.name"
                >
                  {{ project.name }}
                </h3>
              </div>
              <div class="flex gap-2 shrink-0 ml-2">
                <!-- 状态Tag -->
                <div
                  v-if="project.colorTag"
                  class="px-2.5 py-0.5 rounded-full text-[11px] font-bold text-white shadow-sm flex items-center"
                  :style="{ backgroundColor: project.colorTag }"
                >
                  {{ t('projectManagement.color') }}
                </div>
                <div
                  class="px-2.5 py-0.5 rounded-full text-[11px] font-bold bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300 flex items-center tracking-wide"
                >
                  {{ getProjectModeDisplay(project) }}
                </div>
              </div>
            </div>

            <!-- 卡片主体内容 (灰底框) -->
            <div
              class="bg-gray-50/80 dark:bg-gray-900/40 rounded-xl p-4 flex-1 mb-4 border border-gray-100 dark:border-gray-800"
            >
              <p
                class="text-[13px] text-gray-500 dark:text-gray-400 mb-3 line-clamp-2 min-h-[38px] leading-relaxed"
              >
                {{ project.description || t('projectManagement.noDescription') }}
              </p>
              <div
                class="flex justify-between items-center text-[13px] border-t border-gray-200/60 dark:border-gray-700/60 pt-3 mt-3"
              >
                <span class="text-gray-400">{{ t('projectManagement.creator') }}:</span>
                <span class="font-medium text-gray-700 dark:text-gray-300">{{
                  getProjectCreatorDisplay(project)
                }}</span>
              </div>
            </div>

            <!-- 底部操作区 (时间戳 + 圆形图标按钮组) -->
            <div class="flex items-center justify-between pt-1">
              <div
                class="text-[11px] text-gray-400 font-mono bg-gray-50 dark:bg-gray-800/80 px-2 py-1 rounded border border-gray-100 dark:border-gray-700/50"
              >
                {{ formatDate(project.createdAt) }}
              </div>
              <div class="flex items-center gap-1">
                <el-tooltip :content="t('projectManagement.memberAndPermission')" placement="top">
                  <button
                    v-if="canManageProjects"
                    class="w-8 h-8 rounded-full flex items-center justify-center text-gray-400 hover:bg-gray-100 hover:text-gray-800 dark:hover:bg-gray-700 dark:hover:text-gray-200 transition-colors"
                    @click.stop="openRuntimeAccessDialog(project)"
                  >
                    <el-icon class="text-sm"><User /></el-icon>
                  </button>
                </el-tooltip>
                <el-tooltip :content="t('projectManagement.publishAndDeploy')" placement="top">
                  <button
                    v-if="canPerformOps"
                    class="w-8 h-8 rounded-full flex items-center justify-center text-gray-400 hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/30 dark:hover:text-blue-400 transition-colors"
                    @click.stop="openDeployDialog(project)"
                  >
                    <el-icon class="text-sm"><UploadFilled /></el-icon>
                  </button>
                </el-tooltip>
                <el-tooltip
                  :content="t('opsManagement.title')"
                  placement="top"
                  v-if="canPerformOps && isProjectDeployed(project)"
                >
                  <button
                    class="w-8 h-8 rounded-full flex items-center justify-center text-gray-400 hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/30 dark:hover:text-green-400 transition-colors"
                    @click.stop="openOpsManagement(project)"
                  >
                    <el-icon class="text-sm"><Odometer /></el-icon>
                  </button>
                </el-tooltip>
                <el-tooltip :content="t('projectManagement.export')" placement="top">
                  <button
                    v-if="canExportProjects"
                    class="w-8 h-8 rounded-full flex items-center justify-center text-gray-400 hover:bg-gray-100 hover:text-gray-800 dark:hover:bg-gray-700 dark:hover:text-gray-200 transition-colors"
                    @click.stop="handleExportProject(project)"
                  >
                    <el-icon class="text-sm"><Download /></el-icon>
                  </button>
                </el-tooltip>
                <el-tooltip :content="t('projectManagement.delete')" placement="top">
                  <button
                    v-if="canDeleteProjects"
                    class="w-8 h-8 rounded-full flex items-center justify-center text-gray-400 hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/30 dark:hover:text-red-400 transition-colors"
                    @click.stop="deleteProject(project)"
                  >
                    <el-icon class="text-sm"><Delete /></el-icon>
                  </button>
                </el-tooltip>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 列表视图 -->
      <div v-else>
        <el-table
          :data="projectList"
          v-loading="loading"
          style="width: 100%"
          :header-cell-style="{ background: '#f9fafb', color: '#374151' }"
          @selection-change="handleSelectionChange"
        >
          <!-- 选择列 -->
          <el-table-column type="selection" width="55" fixed="left" />
          <el-table-column :label="t('projectManagement.color')" width="80">
            <template #default="scope">
              <div
                class="w-6 h-6 rounded-full border-2 border-white shadow-sm"
                :style="{ backgroundColor: scope.row.colorTag || '#3b82f6' }"
              ></div>
            </template>
          </el-table-column>
          <el-table-column prop="name" :label="t('projectManagement.projectName')" width="200">
            <template #default="scope">
              <span
                class="cursor-pointer text-blue-600 hover:text-blue-800 underline"
                @click="handleProjectNameClick(scope.row)"
              >
                {{ scope.row.name }}
              </span>
            </template>
          </el-table-column>
          <el-table-column
            prop="description"
            :label="t('projectManagement.description')"
            width="200"
          />
          <el-table-column :label="t('projectManagement.createdBy')" width="200">
            <template #default="scope">
              {{ getProjectCreatorDisplay(scope.row) }}
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" :label="t('projectManagement.createdAt')" width="200">
            <template #default="scope">
              {{ formatDateTime(scope.row.createdAt) }}
            </template>
          </el-table-column>
          <el-table-column :label="t('projectManagement.runtimeMode')" width="120" align="center">
            <template #default="scope">
              <el-tag :type="getProjectModeTagType(scope.row)" size="small">
                {{ getProjectModeDisplay(scope.row) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column
            :label="t('projectManagement.actions')"
            min-width="420"
            fixed="right"
            v-if="canManageProjects || canPerformOps"
          >
            <template #default="scope">
              <el-button
                v-if="canManageProjects"
                type="warning"
                size="small"
                @click="openRuntimeAccessDialog(scope.row)"
                class="mr-2"
              >
                {{ t('projectManagement.memberAndPermission') }}
              </el-button>
              <el-button
                v-if="canPerformOps"
                type="primary"
                size="small"
                @click="openDeployDialog(scope.row)"
                class="mr-2"
              >
                {{ t('projectManagement.publishAndDeploy') }}
              </el-button>
              <el-button
                v-if="canPerformOps && isProjectDeployed(scope.row)"
                type="info"
                size="small"
                @click="openOpsManagement(scope.row)"
                class="mr-2"
              >
                {{ t('opsManagement.title') }}
              </el-button>
              <el-button
                v-if="canExportProjects"
                type="success"
                size="small"
                @click="handleExportProject(scope.row)"
                class="mr-2"
              >
                <el-icon class="mr-1">
                  <Download />
                </el-icon>
                {{ t('projectManagement.export') }}
              </el-button>
              <el-button
                v-if="canDeleteProjects"
                type="danger"
                size="small"
                @click="deleteProject(scope.row)"
              >
                {{ t('projectManagement.delete') }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- 分页 -->
      <div
        v-if="pagination.total > 0"
        class="pagination-bar flex justify-between items-center py-2 px-3 border-t border-gray-200 dark:border-gray-700"
      >
        <div class="text-xs text-gray-500 dark:text-gray-400">
          {{
            t('projectManagement.pageSummary', {
              start: (pagination.page - 1) * pagination.limit + 1,
              end: Math.min(pagination.page * pagination.limit, pagination.total),
              total: pagination.total,
            })
          }}
        </div>
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.limit"
          size="small"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </div>

    <!-- 创建工程对话框 -->
    <el-dialog
      v-model="showCreateDialog"
      :title="t('projectManagement.createDialog')"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form ref="createFormRef" :model="createForm" :rules="createFormRules" label-width="100px">
        <el-form-item :label="t('projectManagement.projectName')" prop="name">
          <el-input
            v-model="createForm.name"
            :placeholder="t('projectManagement.inputProjectName')"
          />
        </el-form-item>
        <el-form-item :label="t('projectManagement.description')">
          <el-input
            v-model="createForm.description"
            type="textarea"
            :placeholder="t('projectManagement.inputProjectDescription')"
            :rows="3"
          />
        </el-form-item>
        <el-form-item :label="t('projectManagement.colorTag')">
          <el-select
            v-model="createForm.colorTag"
            :placeholder="t('projectManagement.selectColorTag')"
            style="width: 100%"
          >
            <el-option
              v-for="color in colorTagOptions"
              :key="color.value"
              :label="color.label"
              :value="color.value"
            >
              <div class="flex items-center">
                <div class="w-4 h-4 rounded mr-2" :style="{ backgroundColor: color.value }"></div>
                {{ color.label }}
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="success" @click="importProject">
          <el-icon class="mr-1">
            <Upload />
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
    <el-dialog
      v-model="showEditDialog"
      :title="t('projectManagement.editDialog')"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form ref="editFormRef" :model="editForm" :rules="editFormRules" label-width="100px">
        <el-form-item :label="t('projectManagement.projectName')" prop="name">
          <el-input
            v-model="editForm.name"
            :placeholder="t('projectManagement.inputProjectName')"
          />
        </el-form-item>
        <el-form-item :label="t('projectManagement.description')">
          <el-input
            v-model="editForm.description"
            type="textarea"
            :placeholder="t('projectManagement.inputProjectDescription')"
            :rows="3"
          />
        </el-form-item>
        <el-form-item :label="t('projectManagement.colorTag')">
          <el-select
            v-model="editForm.colorTag"
            :placeholder="t('projectManagement.selectColorTag')"
            style="width: 100%"
          >
            <el-option
              v-for="color in colorTagOptions"
              :key="color.value"
              :label="color.label"
              :value="color.value"
            >
              <div class="flex items-center">
                <div class="w-4 h-4 rounded mr-2" :style="{ backgroundColor: color.value }"></div>
                {{ color.label }}
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">{{ t('projectManagement.cancel') }}</el-button>
        <el-button type="primary" @click="handleUpdateProject" :loading="editLoading">
          {{ t('projectManagement.save') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 运维操作对话框 -->
    <el-dialog
      v-model="showOperationDialog"
      :title="
        t('projectManagement.operationDialog', {
          name: currentProject?.name || '',
        })
      "
      width="400px"
      :close-on-click-modal="false"
    >
      <div class="space-y-3">
        <el-button
          type="success"
          plain
          block
          @click="performOperation('start')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2">
            <VideoPlay />
          </el-icon>
          {{ t('projectManagement.startProject') }}
        </el-button>
        <el-button
          type="warning"
          plain
          block
          @click="performOperation('stop')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2">
            <VideoPause />
          </el-icon>
          {{ t('projectManagement.stopProject') }}
        </el-button>
        <el-button
          type="info"
          plain
          block
          @click="performOperation('restart')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2">
            <RefreshRight />
          </el-icon>
          {{ t('projectManagement.restartProject') }}
        </el-button>
        <el-button
          type="primary"
          plain
          block
          @click="performOperation('deploy')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2">
            <Upload />
          </el-icon>
          {{ t('projectManagement.deployProject') }}
        </el-button>
        <el-button
          type="danger"
          plain
          block
          @click="performOperation('backup')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2">
            <CopyDocument />
          </el-icon>
          {{ t('projectManagement.backupProject') }}
        </el-button>
      </div>
    </el-dialog>

    <!-- 部署对话框 -->
    <el-dialog
      v-model="showDeployDialog"
      :title="
        t('projectManagement.deployDialog', {
          name: deployForm.project?.name || '',
        })
      "
      width="min(860px, 92vw)"
      :close-on-click-modal="false"
      :lock-scroll="true"
      append-to-body
      class="deploy-dialog"
    >
      <!-- 当前模式显示 -->
      <div class="mb-4 p-3 bg-gray-100 dark:bg-gray-800 rounded">
        <span class="text-gray-600 dark:text-gray-400">{{
          t('projectManagement.currentMode')
        }}</span>
        <el-tag :type="getModeTagType(deployForm.currentMode)">
          {{ getModeDisplayLabel(deployForm.currentMode) }}
        </el-tag>
      </div>

      <!-- 部署模式选择 -->
      <el-form :model="deployForm" label-width="100px">
        <el-form-item :label="t('projectManagement.deployMode')">
          <el-radio-group v-model="deployForm.mode">
            <el-radio value="DEV">{{ t('projectManagement.devModeDesc') }}</el-radio>
            <el-radio value="RELEASE">{{ t('projectManagement.releaseModeDesc') }}</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- RELEASE模式：选择版本 -->
        <template v-if="deployForm.mode === 'RELEASE'">
          <el-form-item :label="t('projectManagement.currentVersion')">
            <el-tag type="info">
              {{
                deployForm.currentVersion
                  ? `v${deployForm.currentVersion}`
                  : t('projectManagement.noReleaseVersion')
              }}
            </el-tag>
          </el-form-item>
          <el-form-item :label="t('projectManagement.version')" required>
            <el-select
              v-model="deployForm.version"
              :placeholder="t('projectManagement.selectVersion')"
              filterable
              style="width: 100%"
            >
              <el-option
                v-for="v in releaseVersionOptions"
                :key="v.id || v.version"
                :label="
                  v.isGenerated
                    ? `v${v.version}（${t('projectManagement.newVersion')}）`
                    : `v${v.version} - ${formatDateTime(v.createdAt)}`
                "
                :value="v.version"
              />
            </el-select>
            <el-button
              class="ml-2"
              type="primary"
              plain
              :disabled="!canAddReleaseVersion"
              @click="addReleaseVersionOption"
            >
              {{ t('projectManagement.addVersion') }}
            </el-button>
            <el-button class="ml-2" type="info" plain @click="openVersionManageDialog">
              {{ t('projectManagement.versionManage') }}
            </el-button>
            <div class="text-xs text-gray-500 mt-1">
              {{ t('projectManagement.versionHelpAuto') }}
            </div>
          </el-form-item>
        </template>

        <!-- 选择节点 -->
        <el-form-item :label="t('projectManagement.targetNode')" required>
          <div class="deploy-node-selector w-full">
            <div class="deploy-node-toolbar">
              <el-input
                v-model="nodeKeyword"
                clearable
                :placeholder="t('projectManagement.nodeSearchPlaceholder')"
                class="deploy-node-search"
              />
            </div>

            <div class="deploy-node-list" role="radiogroup">
              <button
                v-for="n in filteredAvailableNodes"
                :key="n.id"
                type="button"
                :class="['deploy-node-item', { 'is-selected': selectedTargetNodeId === n.id }]"
                @click="selectedTargetNodeId = n.id"
              >
                <span
                  class="deploy-node-indicator"
                  :class="{ selected: selectedTargetNodeId === n.id }"
                />
                <div class="deploy-node-option">
                  <div class="deploy-node-row">
                    <span class="deploy-node-name">{{ n.name || '-' }}</span>
                    <span class="deploy-node-ip">{{ n.ipAddress || '-' }}</span>
                    <el-tag size="small" :type="getModeTagType(nodeModes[n.id])">
                      {{ getModeDisplayLabel(nodeModes[n.id]) }}
                    </el-tag>
                  </div>
                </div>
              </button>
              <el-empty
                v-if="filteredAvailableNodes.length === 0"
                :description="t('projectManagement.noMatchedNodes')"
                :image-size="72"
              />
            </div>
          </div>
        </el-form-item>

        <!-- 部署说明 -->
        <el-alert
          v-if="deployForm.mode === 'RELEASE'"
          type="warning"
          :closable="false"
          show-icon
          class="mt-4"
        >
          <template #title>{{ t('projectManagement.releaseGuideTitle') }}</template>
          <ul class="text-sm mt-1">
            <li>{{ t('projectManagement.releaseGuide1') }}</li>
            <li>{{ t('projectManagement.releaseGuide2') }}</li>
            <li>{{ t('projectManagement.releaseGuide3') }}</li>
          </ul>
        </el-alert>

        <el-alert
          v-if="deployForm.mode === 'DEV'"
          type="info"
          :closable="false"
          show-icon
          class="mt-4"
        >
          <template #title>{{ t('projectManagement.devGuideTitle') }}</template>
          <ul class="text-sm mt-1">
            <li>{{ t('projectManagement.devGuide1') }}</li>
            <li>{{ t('projectManagement.devGuide2') }}</li>
            <li>{{ t('projectManagement.devGuide3') }}</li>
          </ul>
        </el-alert>
      </el-form>

      <template #footer>
        <el-button @click="showDeployDialog = false">{{ t('projectManagement.cancel') }}</el-button>
        <el-button type="primary" @click="confirmDeploy" :loading="deployLoading">
          {{
            deployForm.mode === 'RELEASE'
              ? t('projectManagement.publishAndDeploy')
              : t('projectManagement.deployDevMode')
          }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showVersionManageDialog"
      :title="t('projectManagement.versionManageDialog')"
      width="760px"
      append-to-body
    >
      <el-table
        v-loading="versionManageLoading"
        :data="versionManageList"
        size="small"
        style="width: 100%"
      >
        <el-table-column prop="version" :label="t('projectManagement.version')" width="120">
          <template #default="scope"> v{{ scope.row.version }} </template>
        </el-table-column>
        <el-table-column :label="t('projectManagement.versionStatus')" width="130">
          <template #default="scope">
            <el-tag size="small" :type="getVersionStatusTagType(scope.row.status)">
              {{ getVersionStatusLabel(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('projectManagement.versionRefCount')" width="130">
          <template #default="scope">
            {{ scope.row.nodeDeploymentRefCount || 0 }}
          </template>
        </el-table-column>
        <el-table-column :label="t('projectManagement.createdAt')" min-width="180">
          <template #default="scope">
            {{ formatDateTime(scope.row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('projectManagement.actions')" width="150" align="center">
          <template #default="scope">
            <el-button
              type="danger"
              text
              :disabled="!canDeleteVersion(scope.row)"
              @click="deleteVersion(scope.row)"
            >
              {{ t('projectManagement.deleteVersion') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <template #footer>
        <el-button @click="showVersionManageDialog = false">{{
          t('projectManagement.close')
        }}</el-button>
      </template>
    </el-dialog>

    <!-- 工程功能选择弹窗 -->
    <el-dialog
      v-model="projectDialogVisible"
      :title="`${t('projectManagement.chooseFeature')} - ${selectedProject?.name || t('projectManagement.unknownProject')}`"
      width="600px"
      center
      :close-on-click-modal="false"
      append-to-body
    >
      <div class="project-dialog-content">
        <!-- 工程信息展示 -->
        <div class="text-center mb-6">
          <p class="text-sm text-gray-600 dark:text-gray-400">
            {{ selectedProject?.description || t('projectManagement.noDescription') }}
          </p>
        </div>

        <!-- 功能选择卡片 -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
          <!-- 设计中心卡片 -->
          <div
            class="function-card bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-900/20 dark:to-blue-800/20 border-2 border-blue-200 dark:border-blue-700 rounded-xl p-6 cursor-pointer hover:shadow-lg hover:border-blue-300 dark:hover:border-blue-600 transition-all duration-300 hover:scale-105"
            @click="openDesignCenter(selectedProject)"
          >
            <div class="text-center">
              <!-- 图标 -->
              <div
                class="inline-flex items-center justify-center w-16 h-16 bg-blue-500 rounded-full mb-4"
              >
                <svg
                  class="w-8 h-8 text-white"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zM21 5a2 2 0 00-2-2h-4a2 2 0 00-2 2v12a4 4 0 004 4h4a2 2 0 002-2V5z"
                  />
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
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                    />
                  </svg>
                  {{
                    t('projectManagement.pageCount', {
                      count: selectedProject?.pageCount || 0,
                    })
                  }}
                </span>
                <span class="flex items-center">
                  <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M7 4V2a1 1 0 011-1h8a1 1 0 011 1v2m-9 0h10m-9 0V1m10 3V1m0 3l1 1v16a2 2 0 01-2 2H6a2 2 0 01-2-2V5l1-1z"
                    />
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
            class="function-card bg-gradient-to-br from-green-50 to-green-100 dark:from-green-900/20 dark:to-green-800/20 border-2 border-green-200 dark:border-green-700 rounded-xl p-6 cursor-pointer hover:shadow-lg hover:border-green-300 dark:hover:border-green-600 transition-all duration-300 hover:scale-105"
            @click="openDataCenter(selectedProject)"
          >
            <div class="text-center">
              <!-- 图标 -->
              <div
                class="inline-flex items-center justify-center w-16 h-16 bg-green-500 rounded-full mb-4"
              >
                <svg
                  class="w-8 h-8 text-white"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"
                  />
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
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4"
                    />
                  </svg>
                  {{
                    t('projectManagement.dataSourceCount', {
                      count: selectedProject?.dataSourceCount || 0,
                    })
                  }}
                </span>
                <span class="flex items-center">
                  <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
                    />
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
        </div>
      </div>

      <!-- 底部操作区 -->
      <template #footer>
        <div class="flex justify-between items-center">
          <div class="text-sm text-gray-500 dark:text-gray-400">
            <span v-if="selectedProject?.updatedAt">
              {{ t('projectManagement.lastUpdated') }}:
              {{ formatDateTime(selectedProject.updatedAt) }}
            </span>
          </div>
          <div class="space-x-2">
            <el-button
              v-if="canExportProjects"
              type="success"
              @click="exportProject(selectedProject)"
            >
              <el-icon class="mr-1">
                <Download />
              </el-icon>
              {{ t('projectManagement.exportProject') }}
            </el-button>
            <el-button @click="projectDialogVisible = false">{{
              t('projectManagement.cancel')
            }}</el-button>
            <el-button
              v-if="canManageProjects"
              type="primary"
              @click="editProject(selectedProject)"
            >
              {{ t('projectManagement.settings') }}
            </el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <ProjectRuntimeAccessDialog
      v-model:visible="runtimeAccessDialogVisible"
      :project="runtimeAccessProject"
    />
  </div>
</template>

<script lang="ts">
// @ts-nocheck
import { ref, reactive, computed, onMounted, onUnmounted, getCurrentInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import JSZip from 'jszip'
import { useAuthStore, useAppStore } from '@/store'
import { can } from '@/permissions'
import request, { getApiErrorMessage as resolveApiErrorMessage } from '@/utils/request'
import { projectAPI } from '@/api/project.api'
import { ColorTagEnum } from '@/enums'
import { formatDateTime, formatDate, formatCurrency } from '@/utils'
import { initSocket, getSocket } from '@/utils/socket'
import { Storage } from '@/utils/storage'
import ProjectRuntimeAccessDialog from './components/ProjectRuntimeAccessDialog.vue'

export default {
  name: 'ProjectManagement',
  components: {
    ProjectRuntimeAccessDialog,
  },
  setup() {
    const { t } = useI18n()
    const { emit } = getCurrentInstance()
    const authStore = useAuthStore()
    const appStore = useAppStore()
    const isDark = computed(() => appStore.isDark)

    // 当前用户信息
    const currentUser = computed(() => authStore.userInfo)

    // 状态
    const loading = ref(false)
    const createLoading = ref(false)
    const editLoading = ref(false)
    const operationLoading = ref(false)
    const batchOperationLoading = ref(false)

    // 对话框显示状态
    const showCreateDialog = ref(false)
    const showEditDialog = ref(false)
    const showOperationDialog = ref(false)
    const showDeployDialog = ref(false)
    const showVersionManageDialog = ref(false)
    const projectDialogVisible = ref(false)
    const runtimeAccessDialogVisible = ref(false)

    // 视图模式
    const viewMode = ref('card') // 'card' 或 'list'

    // 选择模式状态
    const selectionMode = ref(false)
    const selectedProjects = ref([]) // 选中的工程ID列表
    const deployStateRefreshTimer = ref(null)

    // 当前操作的工程
    const currentProject = ref(null)
    const selectedProject = ref(null)
    const runtimeAccessProject = ref(null)

    // 部署相关状态
    const deployLoading = ref(false)
    const versionManageLoading = ref(false)
    const versionManageList = ref([])
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
    const projectDeployState = reactive({}) // { projectId: { deployed: boolean, mode: 'DEV'|'RELEASE'|null } }
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

    // 工程列表和分页
    const projectList = ref([])
    const pagination = reactive({
      page: 1,
      limit: 10,
      total: 0,
      totalPages: 0,
    })

    // 搜索表单
    const searchForm = reactive({
      name: '',
    })
    const searchTimer = ref(null)

    // 颜色标签选项
    const colorTagOptions = [
      { value: ColorTagEnum.BLUE, label: t('projectManagement.colorBlue') },
      { value: ColorTagEnum.RED, label: t('projectManagement.colorRed') },
      { value: ColorTagEnum.GREEN, label: t('projectManagement.colorGreen') },
      { value: ColorTagEnum.YELLOW, label: t('projectManagement.colorYellow') },
      { value: ColorTagEnum.PURPLE, label: t('projectManagement.colorPurple') },
      { value: ColorTagEnum.PINK, label: t('projectManagement.colorPink') },
      { value: ColorTagEnum.GRAY, label: t('projectManagement.colorGray') },
    ]

    // 创建工程表单
    const createForm = reactive({
      name: '',
      description: '',
      colorTag: ColorTagEnum.BLUE,
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
      colorTag: ColorTagEnum.BLUE,
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

    /**
     * 将 HEX 颜色转为 RGB 对象。
     * @param {string} hex - HEX 颜色值
     * @returns {{r:number,g:number,b:number}|null}
     */
    const hexToRgb = (hex) => {
      if (!hex || typeof hex !== 'string') return null
      const normalized = hex.replace('#', '')
      if (![3, 6].includes(normalized.length)) return null
      const fullHex =
        normalized.length === 3
          ? normalized
              .split('')
              .map((c) => c + c)
              .join('')
          : normalized
      const num = Number.parseInt(fullHex, 16)
      if (Number.isNaN(num)) return null
      return {
        r: (num >> 16) & 255,
        g: (num >> 8) & 255,
        b: num & 255,
      }
    }

    /**
     * 根据主题生成工程卡片背景样式。
     * @param {object} project - 工程对象
     * @returns {Record<string,string>} 行内样式
     */
    const getProjectCardStyle = (project) => {
      const baseColor = project?.colorTag || '#3b82f6'
      const rgb = hexToRgb(baseColor)
      if (!rgb) return { backgroundColor: baseColor }
      if (isDark.value) {
        return {
          background: `linear-gradient(135deg, rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.34) 0%, rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.56) 100%)`,
        }
      }
      return {
        background: `linear-gradient(135deg, rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.88) 0%, rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.96) 100%)`,
      }
    }

    // 获取工程列表
    const fetchProjects = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page,
          limit: pagination.limit,
          ...searchForm,
        }

        // 移除空值
        Object.keys(params).forEach((key) => {
          if (!params[key]) delete params[key]
        })

        const response = await projectAPI.getProjects(params)
        const payload = response?.data || {}
        const listPayload = payload?.list || payload
        const paginationPayload = payload?.pagination || response?.pagination || {}

        projectList.value = listPayload?.projects || []
        pagination.total = paginationPayload?.total || 0
        pagination.totalPages = paginationPayload?.totalPages || 0
        await fetchProjectDeployState()
      } catch (error) {
        ElMessage.error(
            t('projectManagement.fetchFailed', {
            message: getApiErrorMessage(error, t('projectManagement.fetchFailed')),
          }),
        )
      } finally {
        loading.value = false
      }
    }

    // 执行搜索
    const executeSearch = () => {
      pagination.page = 1
      fetchProjects()
    }

    // 搜索处理（防抖）
    const handleSearch = () => {
      if (searchTimer.value) {
        window.clearTimeout(searchTimer.value)
      }
      searchTimer.value = window.setTimeout(() => {
        executeSearch()
      }, 300)
    }

    // 重置搜索
    const resetSearch = () => {
      if (searchTimer.value) {
        window.clearTimeout(searchTimer.value)
      }
      Object.keys(searchForm).forEach((key) => {
        searchForm[key] = ''
      })
      pagination.page = 1
      fetchProjects()
    }

    // 分页大小改变
    const handleSizeChange = (size) => {
      pagination.limit = size
      pagination.page = 1
      fetchProjects()
    }

    // 页码改变
    const handleCurrentChange = (page) => {
      pagination.page = page
      fetchProjects()
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
          colorTag: createForm.colorTag,
        }

        await projectAPI.createProject(projectData)

        ElMessage.success(t('projectManagement.createSuccess'))
        showCreateDialog.value = false
        resetCreateForm()
        fetchProjects()
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
        if (key === 'colorTag') {
          createForm[key] = ColorTagEnum.BLUE
        } else {
          createForm[key] = ''
        }
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
      editForm.colorTag = project.colorTag || ColorTagEnum.BLUE
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
          colorTag: editForm.colorTag,
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

    // 删除工程
    const deleteProject = async (project) => {
      if (!canDeleteProjects.value) {
        ElMessage.warning(t('projectManagement.noPermission'))
        return
      }
      try {
        let impactData = null
        try {
          const impactRes = await projectAPI.getDeleteImpact(project.id)
          impactData = impactRes?.data ?? impactRes
        } catch (impactError) {
          ElMessage.warning(t('projectManagement.deleteImpactLoadFailed'))
        }

        if (impactData?.hasActiveDeployments) {
          if (!canForceDeleteProject.value) {
            ElMessage.warning(
              t('projectManagement.deleteBlockedByActiveDeployments', {
                count: impactData?.activeDeploymentCount || 0,
              }),
            )
            return
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
          ElMessage.success(t('projectManagement.forceDeleteSuccess'))
          fetchProjects()
          return
        }

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

        await projectAPI.deleteProject(project.id)
        ElMessage.success(t('projectManagement.deleteSuccess'))
        fetchProjects()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(
              t('projectManagement.deleteFailed', {
                message: getApiErrorMessage(error, t('projectManagement.deleteFailed')),
            }),
          )
        }
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
          fetchProjects()
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

    // 显示运维操作对话框
    const openOperationDialog = (project) => {
      currentProject.value = project
      showOperationDialog.value = true
    }

    // 执行运维操作
    const performOperation = async (operation) => {
      if (!currentProject.value) return

      operationLoading.value = true
      try {
        await projectAPI.performOperation(currentProject.value.id, operation)

        ElMessage.success(
          t('projectManagement.operationSuccess', {
            action:
              operation === 'start'
                ? t('projectManagement.actionStart')
                : operation === 'stop'
                  ? t('projectManagement.actionStop')
                  : operation === 'restart'
                    ? t('projectManagement.actionRestart')
                    : operation === 'deploy'
                      ? t('projectManagement.actionDeploy')
                      : t('projectManagement.actionBackup'),
          }),
        )

        showOperationDialog.value = false
        currentProject.value = null
      } catch (error) {
        ElMessage.error(
          t('projectManagement.operationFailed', {
            message: getApiErrorMessage(error, t('projectManagement.operationFailed')),
          }),
        )
      } finally {
        operationLoading.value = false
      }
    }

    // 切换选择模式
    const toggleSelectionMode = () => {
      selectionMode.value = !selectionMode.value
      if (!selectionMode.value) {
        // 退出选择模式时清除选择
        selectedProjects.value = []
      }
    }

    // 卡片点击处理
    const handleCardClick = (project) => {
      if (selectionMode.value) {
        // 在选择模式下点击切换选中状态
        toggleProjectSelection(project.id)
      } else {
        if (!canAccessProjectDetail.value) {
          ElMessage.warning(t('projectManagement.noPermission'))
          return
        }
        // 非选择模式下打开工程详情
        openProjectDialog(project)
      }
    }

    // 工程名称点击处理（列表视图）
    const handleProjectNameClick = (project) => {
      if (selectionMode.value) {
        // 在选择模式下点击切换选中状态
        toggleProjectSelection(project.id)
      } else {
        if (!canAccessProjectDetail.value) {
          ElMessage.warning(t('projectManagement.noPermission'))
          return
        }
        // 非选择模式下打开工程详情
        openProjectDialog(project)
      }
    }

    // 切换工程选中状态
    const toggleProjectSelection = (projectId, value = null) => {
      const index = selectedProjects.value.indexOf(projectId)
      if (value === true || (value === null && index === -1)) {
        // 选中
        if (index === -1) {
          selectedProjects.value.push(projectId)
        }
      } else if (value === false || (value === null && index !== -1)) {
        // 取消选中
        if (index !== -1) {
          selectedProjects.value.splice(index, 1)
        }
      }
    }

    // 处理表格选择变化
    const handleSelectionChange = (selection) => {
      selectedProjects.value = selection.map((p) => p.id)
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

        // 逐个删除工程
        let successCount = 0
        let failCount = 0
        for (const projectId of selectedProjects.value) {
          try {
            await projectAPI.deleteProject(projectId)
            successCount++
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
        fetchProjects()
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

    // 打开工程功能选择弹窗
    const openProjectDialog = (project) => {
      if (isOpsAdminRole.value) {
        openDeployDialog(project)
        return
      }
      selectedProject.value = project
      projectDialogVisible.value = true
    }

    // 打开运行态成员与权限对话框。
    const openRuntimeAccessDialog = (project) => {
      runtimeAccessProject.value = project
      runtimeAccessDialogVisible.value = true
    }

    // 打开设计中心
    const openDesignCenter = (project) => {
      projectDialogVisible.value = false
      // 在标签页内打开设计中心（使用 iframe 嵌入）
      import('@/components/EmbeddedApp.vue').then((module) => {
        const EmbeddedApp = module.default
        emit('open-tab', {
          key: `design-center-${project.id}`,
          titlePrefix: project.name,
          titleKey: 'projectManagement.designCenter',
          component: EmbeddedApp,
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
      // 在标签页内打开数据中心（使用 iframe 嵌入）
      import('@/components/EmbeddedApp.vue').then((module) => {
        const EmbeddedApp = module.default
        emit('open-tab', {
          key: `data-center-${project.id}`,
          titlePrefix: project.name,
          titleKey: 'projectManagement.dataCenter',
          component: EmbeddedApp,
          props: {
            appType: 'datacenter',
            project: project,
          },
          icon: 'database',
        })
      })
    }

    // 打开运维管理并按工程过滤，运行态操作统一在运维管理执行。
    const openOpsManagement = (project) => {
      if (!isProjectDeployed(project)) {
        ElMessage.warning(`${t('projectManagement.notDeployed')}，请先完成发布并部署`)
        return
      }
      emit('open-tab', 'ops-management')
      // 从工程列表跳转运维管理时，不再自动按工程过滤；同时清空可能残留的筛选条件。
      window.setTimeout(() => {
        window.dispatchEvent(
          new window.CustomEvent('ops:set-project-filter', {
            detail: { projectId: '', projectName: '' },
          }),
        )
      }, 80)
    }

    // 同步工程部署状态，用于工程列表的运行模式展示与入口控制。
    const fetchProjectDeployState = async () => {
      Object.keys(projectDeployState).forEach((key) => {
        delete projectDeployState[key]
      })

      try {
        const res = await request.get('/nodes', {
          params: {
            page: 1,
            pageSize: 500,
            approvalStatus: 'approved',
          },
        })
        const payload = res?.data ?? res
        const nodes = payload?.data?.items || payload?.items || []
        nodes.forEach((node) => {
          const deployments = Array.isArray(node?.deployments) ? node.deployments : []
          deployments.forEach((deploy) => {
            const projectId = deploy?.projectId
            if (!projectId) return
            if (!projectDeployState[projectId]) {
              projectDeployState[projectId] = {
                deployed: true,
                mode: deploy.mode || null,
              }
            } else if (!projectDeployState[projectId].mode && deploy.mode) {
              projectDeployState[projectId].mode = deploy.mode
            }
          })
        })
      } catch (error) {
        console.error('获取工程部署状态失败:', error)
      }
    }

    const isProjectDeployed = (project) => {
      return Boolean(projectDeployState[project?.id]?.deployed)
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

    // 获取工程运行模式显示
    const getProjectModeDisplay = (project) => {
      const mode = projectDeployState[project?.id]?.mode
      if (mode === 'DEV') return t('projectManagement.modeDisplayDev')
      if (mode === 'RELEASE') return t('projectManagement.modeDisplayRelease')
      return t('projectManagement.notDeployed')
    }

    // 获取工程模式标签类型
    const getProjectModeTagType = (project) => {
      const mode = projectDeployState[project?.id]?.mode || null
      return getModeTagType(mode)
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

    /**
     * 打开版本管理弹窗并拉取当前工程版本。
     * @returns {Promise<void>}
     */
    const openVersionManageDialog = async () => {
      if (!deployForm.project?.id) return
      showVersionManageDialog.value = true
      await fetchVersionManageList()
    }

    /**
     * 获取版本管理列表。
     * @returns {Promise<void>}
     */
    const fetchVersionManageList = async () => {
      if (!deployForm.project?.id) return
      versionManageLoading.value = true
      try {
        const res = await request.get(`/publish/${deployForm.project.id}/versions`, {
          params: { page: 1, pageSize: 200 },
        })
        const payload = res?.data ?? res
        const data = payload?.data || payload || {}
        const rawItems = data.items || []
        versionManageList.value = rawItems.filter(
          (item) => item?.mode !== 'DEV' && item?.version !== '__DEV__',
        )
      } catch (error) {
        ElMessage.error(getApiErrorMessage(error, t('projectManagement.versionManageLoadFailed')))
      } finally {
        versionManageLoading.value = false
      }
    }

    /**
     * 判断版本是否允许删除。
     * @param {object} versionItem - 版本条目
     * @returns {boolean}
     */
    const canDeleteVersion = (versionItem) => {
      if (!versionItem) return false
      if ((versionItem.nodeDeploymentRefCount || 0) > 0) return false
      return ['success', 'failed'].includes(versionItem.status)
    }

    /**
     * 获取版本状态标签类型。
     * @param {string} status - 状态
     * @returns {string}
     */
    const getVersionStatusTagType = (status) => {
      if (status === 'success') return 'success'
      if (status === 'failed') return 'danger'
      if (status === 'building') return 'warning'
      return 'info'
    }

    /**
     * 获取版本状态显示文本。
     * @param {string} status - 状态
     * @returns {string}
     */
    const getVersionStatusLabel = (status) => {
      return t(`projectManagement.versionStatus_${status || 'pending'}`)
    }

    /**
     * 删除发布版本。
     * @param {object} versionItem - 版本条目
     * @returns {Promise<void>}
     */
    const deleteVersion = async (versionItem) => {
      if (!canDeleteVersion(versionItem)) {
        return ElMessage.warning(t('projectManagement.deleteVersionBlocked'))
      }
      try {
        await ElMessageBox.confirm(
          t('projectManagement.deleteVersionConfirm', {
            version: versionItem.version,
          }),
          t('projectManagement.deleteConfirmTitle'),
          { type: 'warning' },
        )
      } catch {
        return
      }

      try {
        await request.delete(`/publish/deployment/${versionItem.id}`)
        ElMessage.success(t('projectManagement.deleteVersionSuccess'))
        await fetchVersionManageList()
        await openDeployDialog(deployForm.project)
      } catch (error) {
        ElMessage.error(getApiErrorMessage(error, t('projectManagement.deleteVersionFailed')))
      }
    }

    // 创建者展示统一使用用户名，避免显示角色/姓名造成歧义。
    const getProjectCreatorDisplay = (project) => {
      const creator = project?.creator || {}
      return creator.username || creator.fullName || t('projectManagement.unknown')
    }

    const scheduleDeployStateRefresh = () => {
      if (deployStateRefreshTimer.value) {
        window.clearTimeout(deployStateRefreshTimer.value)
      }
      deployStateRefreshTimer.value = window.setTimeout(() => {
        fetchProjectDeployState()
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

    // 打开部署对话框
    const openDeployDialog = async (project) => {
      deployForm.project = project
      deployForm.currentMode = null
      deployForm.currentVersion = ''
      deployForm.mode = 'RELEASE'
      deployForm.version = ''
      deployForm.targetNodes = []
      nodeKeyword.value = ''
      generatedReleaseVersions.value = []
      // 切换工程时清空节点模式缓存，避免沿用上一次工程的模式。
      Object.keys(nodeModes).forEach((nodeId) => {
        delete nodeModes[nodeId]
      })

      try {
        const [nodesRes, projectNodeDeployRes, versionsRes] = await Promise.all([
          // 获取可用节点列表
          request.get('/nodes', {
            params: { approvalStatus: 'approved', status: 'online' },
          }),
          // 获取工程在节点上的部署关系（用于回填模式与已部署节点）
          request.get(`/deployments/project/${project.id}/nodes`),
          // 获取版本列表
          request.get(`/publish/${project.id}/versions`),
        ])

        // 兼容不同的响应结构
        const nodesPayload = nodesRes?.data ?? nodesRes
        const nodesData = nodesPayload?.data || nodesPayload || {}
        availableNodes.value = nodesData.items || nodesData || []

        const projectNodeDeployPayload = projectNodeDeployRes?.data ?? projectNodeDeployRes
        const projectNodeDeployData =
          projectNodeDeployPayload?.data || projectNodeDeployPayload || []
        const projectNodeDeployments = Array.isArray(projectNodeDeployData)
          ? projectNodeDeployData
          : projectNodeDeployData.items || []

        // 以工程当前部署关系作为模式来源，保证“已部署但未运行”也能正确回填。
        const deployedModeByNodeId = {}
        const deployedNodeIds = []
        projectNodeDeployments.forEach((item) => {
          const nodeId = item?.nodeId || item?.node?.id
          if (!nodeId) return
          const mode = item?.mode || item?.deployment?.mode || null
          if (mode) {
            deployedModeByNodeId[nodeId] = mode
          }
          deployedNodeIds.push(nodeId)
        })

        // 回填在线节点的模式展示。
        availableNodes.value.forEach((node) => {
          nodeModes[node.id] = deployedModeByNodeId[node.id] || null
        })

        // 仅默认勾选“当前在线且已部署”的节点。
        const defaultSelectedNodes = availableNodes.value
          .map((node) => node.id)
          .filter((nodeId) => deployedNodeIds.includes(nodeId))
        deployForm.targetNodes = defaultSelectedNodes.length > 0 ? [defaultSelectedNodes[0]] : []

        // 当前模式：优先使用已部署节点的模式。
        const selectedModeSet = new Set(
          deployForm.targetNodes.map((nodeId) => nodeModes[nodeId]).filter(Boolean),
        )
        if (selectedModeSet.size === 1) {
          const [singleMode] = Array.from(selectedModeSet)
          deployForm.currentMode = singleMode
          deployForm.mode = singleMode
        } else if (selectedModeSet.size > 1) {
          deployForm.currentMode = Array.from(selectedModeSet)[0]
          // 混合模式时默认保持 RELEASE，用户可手动切换并选择节点。
          deployForm.mode = 'RELEASE'
        }

        // 获取版本列表
        // 兼容不同的响应结构
        const versionsPayload = versionsRes?.data ?? versionsRes
        const versionsData = versionsPayload?.data || versionsPayload || {}
        const allVersions = versionsData.items || versionsData || []
        // 发布版本全量集合（仅生产模式，含非成功状态），用于“是否已存在版本号”判定。
        allReleaseVersions.value = allVersions
          .filter((item) => item?.mode !== 'DEV' && item?.type !== 'SOURCE')
          .sort((a, b) => {
            const aTime = new Date(a?.createdAt || 0).getTime()
            const bTime = new Date(b?.createdAt || 0).getTime()
            return bTime - aTime
          })

        // 可部署版本：仅保留构建成功版本。
        projectVersions.value = allReleaseVersions.value
          .filter((item) => item?.status === 'success')
          .sort((a, b) => {
            const aTime = new Date(a?.createdAt || 0).getTime()
            const bTime = new Date(b?.createdAt || 0).getTime()
            return bTime - aTime
          })

        // 当前版本号：优先展示当前已部署的生产模式版本；否则展示最新生产版本。
        const currentReleaseDeploy = projectNodeDeployments.find(
          (item) =>
            (item?.mode || item?.deployment?.mode) === 'RELEASE' &&
            (item?.version || item?.deployment?.version),
        )
        deployForm.currentVersion =
          currentReleaseDeploy?.version || currentReleaseDeploy?.deployment?.version || ''

        // 生产模式默认选中当前版本；没有则保持空，等待用户新增版本。
        if (deployForm.mode === 'RELEASE') {
          deployForm.version = deployForm.currentVersion || ''
        }

        showDeployDialog.value = true
      } catch (error) {
        ElMessage.error(
          t('projectManagement.loadDataFailed', {
            message: getApiErrorMessage(error, t('opsManagement.operationFailedFallback')),
          }),
        )
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
      operationLoading,
      batchOperationLoading,
      showCreateDialog,
      showEditDialog,
      showOperationDialog,
      showDeployDialog,
      showVersionManageDialog,
      projectDialogVisible,
      runtimeAccessDialogVisible,

      // 视图
      viewMode,

      // 选择模式
      selectionMode,
      selectedProjects,
      currentProject,
      selectedProject,
      runtimeAccessProject,

      // 数据
      projectList,
      pagination,
      searchForm,
      colorTagOptions,
      currentUser,

      // 部署相关
      deployLoading,
      versionManageLoading,
      versionManageList,
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
      handleSearch,
      resetSearch,
      handleSizeChange,
      handleCurrentChange,
      handleCreateProject,
      toggleSelectionMode,
      handleCardClick,
      handleProjectNameClick,
      toggleProjectSelection,
      handleSelectionChange,
      importProject,
      editProject,
      handleUpdateProject,
      deleteProject,
      handleExportProject,
      handleImportProject,
      openOperationDialog,
      performOperation,
      openProjectDialog,
      openRuntimeAccessDialog,
      openDesignCenter,
      openDataCenter,
      isProjectDeployed,
      openOpsManagement,
      openDeployDialog,
      openVersionManageDialog,
      canDeleteVersion,
      getVersionStatusTagType,
      getVersionStatusLabel,
      deleteVersion,
      exportProject,
      batchExportProjects,
      batchDeleteProjects,
      confirmDeploy,
      getNodeMode,
      getModeTagType,
      getModeDisplayLabel,
      getProjectCardStyle,
      getProjectModeDisplay,
      getProjectModeTagType,
      getProjectCreatorDisplay,
      formatDateTime,
      formatDate,
      formatCurrency,
    }
  },
}
</script>

<style scoped>
.project-management {
  padding: 20px;
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
