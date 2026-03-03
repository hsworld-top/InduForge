<template>
  <div class="project-management">
    <!-- 页面标题和操作栏 -->
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
        {{ t('projectManagement.title') }}
      </h1>
      <div class="flex items-center space-x-4">
        <!-- 视图切换 -->
        <div
          class="flex items-center bg-gray-100 dark:bg-gray-700 rounded-lg p-1"
        >
          <button
            @click="viewMode = 'card'"
            :class="[
              'px-3 py-2 rounded-md text-sm font-medium transition-colors',
              viewMode === 'card'
                ? 'bg-white dark:bg-gray-600 text-blue-600 dark:text-blue-400 shadow-sm'
                : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200',
            ]"
          >
            <el-icon class="mr-1"><Grid /></el-icon>
            {{ t('projectManagement.cardView') }}
          </button>
          <button
            @click="viewMode = 'list'"
            :class="[
              'px-3 py-2 rounded-md text-sm font-medium transition-colors',
              viewMode === 'list'
                ? 'bg-white dark:bg-gray-600 text-blue-600 dark:text-blue-400 shadow-sm'
                : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200',
            ]"
          >
            <el-icon class="mr-1"><List /></el-icon>
            {{ t('projectManagement.listView') }}
          </button>
        </div>

        <!-- 添加工程按钮 -->
        <el-button
          v-if="canManageProjects"
          type="primary"
          @click="showCreateDialog = true"
          class="bg-blue-600 hover:bg-blue-700"
        >
          <el-icon class="mr-2"><Plus /></el-icon>
          {{ t('projectManagement.addProject') }}
        </el-button>
        <el-button
          v-if="canManageProjects"
          type="default"
          @click="handleImportProject"
        >
          <el-icon class="mr-2"><Upload /></el-icon>
          {{ t('projectManagement.importProject') }}
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选栏 -->
    <div
      class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-4 mb-6"
    >
      <div class="flex items-center justify-between">
        <!-- 搜索表单 -->
        <div class="flex items-center space-x-3">
          <el-input
            v-model="searchForm.name"
            :placeholder="t('projectManagement.searchPlaceholder')"
            clearable
            style="width: 200px"
            @input="handleSearch"
          />
          <el-button @click="resetSearch" type="default" size="small">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>

        <!-- 操作按钮组 -->
        <div class="flex items-center space-x-3">
          <template v-if="selectionMode">
            <el-button
              type="success"
              size="small"
              @click="batchExportProjects"
              :disabled="selectedProjects.length === 0"
              :loading="batchOperationLoading"
            >
              <el-icon class="mr-1"><Download /></el-icon>
              {{ t('projectManagement.batchExport') }} ({{ selectedProjects.length }})
            </el-button>
            <el-button
              v-if="canManageProjects"
              type="danger"
              size="small"
              @click="batchDeleteProjects"
              :disabled="selectedProjects.length === 0"
              :loading="batchOperationLoading"
            >
              <el-icon class="mr-1"><Delete /></el-icon>
              {{ t('projectManagement.batchDelete') }} ({{ selectedProjects.length }})
            </el-button>
            <el-divider direction="vertical" />
          </template>

          <!-- 多选切换按钮 -->
          <el-button
            :type="selectionMode ? 'primary' : 'default'"
            size="small"
            @click="toggleSelectionMode"
          >
            <el-icon class="mr-1"><Select /></el-icon>
            {{ selectionMode ? t("projectManagement.cancelSelection") : t("projectManagement.multiSelect") }}
          </el-button>
        </div>
      </div>
    </div>

    <!-- 工程列表 -->
    <div
      class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700"
    >
      <!-- 卡片视图 -->
      <div v-if="viewMode === 'card'" class="p-6">
        <div
          v-if="projectList.length === 0 && !loading"
          class="text-center py-12"
        >
          <el-empty :description="t('projectManagement.emptyProjects')" />
        </div>
        <div
          v-else
          class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"
        >
          <div
            v-for="project in projectList"
            :key="project.id"
            class="project-card relative border border-gray-200 dark:border-gray-600 rounded-lg p-6 hover:shadow-lg transition-all duration-200 cursor-pointer"
            :class="{
              'ring-4 ring-blue-500': selectedProjects.includes(project.id),
              'opacity-60': selectionMode,
            }"
            :style="getProjectCardStyle(project)"
            @click="handleCardClick(project)"
          >
            <!-- 选择模式下的复选框 -->
            <div
              v-if="selectionMode"
              class="absolute top-2 right-2 z-10"
              @click.stop
            >
              <el-checkbox
                :model-value="selectedProjects.includes(project.id)"
                @change="(val) => toggleProjectSelection(project.id, val)"
                size="large"
                :style="{
                  '--el-checkbox-checked-bg-color': '#10b981',
                  '--el-checkbox-checked-input-border-color': '#10b981',
                }"
              />
            </div>

            <!-- 卡片头部 -->
            <div class="flex items-start justify-between mb-4">
              <div class="flex-1">
                <h3 class="text-lg font-semibold text-white mb-1">
                  {{ project.name }}
                </h3>
              </div>
              <div
                class="w-6 h-6 rounded-full border-2 border-white shadow-sm bg-white opacity-20"
              ></div>
            </div>

            <!-- 工程描述 -->
            <p class="text-sm text-white opacity-90 mb-4 line-clamp-2">
              {{ project.description || t("projectManagement.noDescription") }}
            </p>

            <!-- 工程信息 -->
            <div class="space-y-2 mb-4">
              <div class="flex justify-between text-sm">
                <span class="text-white opacity-75">{{ t('projectManagement.creator') }}:</span>
                <span class="text-white font-medium">{{
                  project.creator?.fullName || t("projectManagement.unknown")
                }}</span>
              </div>
              <div class="flex justify-between text-sm">
                <span class="text-white opacity-75">{{ t('projectManagement.runtimeMode') }}:</span>
                <el-tag
                  :type="getProjectModeTagType(project)"
                  size="small"
                  effect="dark"
                >
                  {{ getProjectModeDisplay(project) }}
                </el-tag>
              </div>
            </div>

            <!-- 操作按钮 -->
            <div class="flex justify-end space-x-2">
              <el-button
                v-if="canPerformOps"
                type="primary"
                size="small"
                @click.stop="openDeployDialog(project)"
              >
                {{ t('projectManagement.publishAndDeploy') }}
              </el-button>
              <el-button
                v-if="canPerformOps && isProjectDeployed(project)"
                type="info"
                size="small"
                @click.stop="openOpsManagement(project)"
              >
                {{ t('opsManagement.title') }}
              </el-button>
              <el-button
                v-if="canManageProjects"
                type="success"
                size="small"
                @click.stop="handleExportProject(project)"
              >
                <el-icon class="mr-1"><Download /></el-icon>
                {{ t('projectManagement.export') }}
              </el-button>
              <el-button
                v-if="canManageProjects"
                type="danger"
                size="small"
                @click.stop="deleteProject(project)"
              >
                {{ t('projectManagement.delete') }}
              </el-button>
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
          <!-- 选择列（仅在选择模式显示） -->
          <el-table-column
            v-if="selectionMode"
            type="selection"
            width="55"
            fixed="left"
          />
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
          <el-table-column prop="description" :label="t('projectManagement.description')" width="200" />
          <el-table-column prop="creator.fullName" :label="t('projectManagement.createdBy')" width="200" />
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
            min-width="320"
            fixed="right"
            v-if="canManageProjects || canPerformOps"
          >
            <template #default="scope">
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
                v-if="canManageProjects"
                type="success"
                size="small"
                @click="handleExportProject(scope.row)"
                class="mr-2"
              >
                <el-icon class="mr-1"><Download /></el-icon>
                {{ t('projectManagement.export') }}
              </el-button>
              <el-button
                v-if="canManageProjects"
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
        class="flex justify-between items-center p-4 border-t border-gray-200 dark:border-gray-700"
      >
        <div class="text-sm text-gray-500 dark:text-gray-400">
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
      <el-form
        ref="createFormRef"
        :model="createForm"
        :rules="createFormRules"
        label-width="100px"
      >
        <el-form-item :label="t('projectManagement.projectName')" prop="name">
          <el-input v-model="createForm.name" :placeholder="t('projectManagement.inputProjectName')" />
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
                <div
                  class="w-4 h-4 rounded mr-2"
                  :style="{ backgroundColor: color.value }"
                ></div>
                {{ color.label }}
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="success" @click="importProject">
          <el-icon class="mr-1"><Upload /></el-icon>
          {{ t('projectManagement.importAction') }}
        </el-button>
        <el-button @click="showCreateDialog = false">{{ t('projectManagement.cancel') }}</el-button>
        <el-button
          type="primary"
          @click="handleCreateProject"
          :loading="createLoading"
        >
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
      <el-form
        ref="editFormRef"
        :model="editForm"
        :rules="editFormRules"
        label-width="100px"
      >
        <el-form-item :label="t('projectManagement.projectName')" prop="name">
          <el-input v-model="editForm.name" :placeholder="t('projectManagement.inputProjectName')" />
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
                <div
                  class="w-4 h-4 rounded mr-2"
                  :style="{ backgroundColor: color.value }"
                ></div>
                {{ color.label }}
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">{{ t('projectManagement.cancel') }}</el-button>
        <el-button
          type="primary"
          @click="handleUpdateProject"
          :loading="editLoading"
        >
          {{ t('projectManagement.save') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 运维操作对话框 -->
    <el-dialog
      v-model="showOperationDialog"
      :title="t('projectManagement.operationDialog', { name: currentProject?.name || '' })"
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
          <el-icon class="mr-2"><VideoPlay /></el-icon>
          {{ t('projectManagement.startProject') }}
        </el-button>
        <el-button
          type="warning"
          plain
          block
          @click="performOperation('stop')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2"><VideoPause /></el-icon>
          {{ t('projectManagement.stopProject') }}
        </el-button>
        <el-button
          type="info"
          plain
          block
          @click="performOperation('restart')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2"><RefreshRight /></el-icon>
          {{ t('projectManagement.restartProject') }}
        </el-button>
        <el-button
          type="primary"
          plain
          block
          @click="performOperation('deploy')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2"><Upload /></el-icon>
          {{ t('projectManagement.deployProject') }}
        </el-button>
        <el-button
          type="danger"
          plain
          block
          @click="performOperation('backup')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2"><CopyDocument /></el-icon>
          {{ t('projectManagement.backupProject') }}
        </el-button>
      </div>
    </el-dialog>

    <!-- 部署对话框 -->
    <el-dialog
      v-model="showDeployDialog"
      :title="t('projectManagement.deployDialog', { name: deployForm.project?.name || '' })"
      width="600px"
      :close-on-click-modal="false"
    >
      <!-- 当前模式显示 -->
      <div class="mb-4 p-3 bg-gray-100 dark:bg-gray-800 rounded">
        <span class="text-gray-600 dark:text-gray-400">{{ t('projectManagement.currentMode') }}</span>
        <el-tag :type="getModeTagType(deployForm.currentMode)">
          {{ deployForm.currentMode || t('projectManagement.notDeployed') }}
        </el-tag>
      </div>

      <!-- 部署模式选择 -->
      <el-form :model="deployForm" label-width="100px">
        <el-form-item :label="t('projectManagement.deployMode')">
          <el-radio-group v-model="deployForm.mode">
            <el-radio value="RELEASE">{{ t('projectManagement.releaseModeDesc') }}</el-radio>
            <el-radio value="DEV">{{ t('projectManagement.devModeDesc') }}</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- RELEASE模式：选择版本 -->
        <template v-if="deployForm.mode === 'RELEASE'">
          <el-form-item :label="t('projectManagement.version')" required>
            <el-select
              v-model="deployForm.version"
              :placeholder="t('projectManagement.selectOrInputVersion')"
              filterable
              allow-create
              style="width: 100%"
            >
              <el-option
                v-for="v in projectVersions"
                :key="v.id"
                :label="`v${v.version} - ${formatDateTime(v.createdAt)}`"
                :value="v.version"
              />
            </el-select>
            <div class="text-xs text-gray-500 mt-1">
              {{ t('projectManagement.versionHelp') }}
            </div>
          </el-form-item>
        </template>

        <!-- 选择节点 -->
        <el-form-item :label="t('projectManagement.targetNode')" required>
          <el-checkbox-group v-model="deployForm.targetNodes">
            <el-checkbox v-for="n in availableNodes" :key="n.id" :value="n.id">
              {{ n.name }} ({{ n.ipAddress }})
              <el-tag
                v-if="getNodeMode(n.id) === 'DEV'"
                type="warning"
                size="small"
                class="ml-1"
              >
                DEV
              </el-tag>
              <el-tag
                v-if="getNodeMode(n.id) === 'RELEASE'"
                type="success"
                size="small"
                class="ml-1"
              >
                RELEASE
              </el-tag>
            </el-checkbox>
          </el-checkbox-group>
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
        <el-button
          type="primary"
          @click="confirmDeploy"
          :loading="deployLoading"
        >
          {{ deployForm.mode === "RELEASE" ? t('projectManagement.publishAndDeploy') : t('projectManagement.deployDevMode') }}
        </el-button>
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
              <h4
                class="text-xl font-semibold text-gray-900 dark:text-white mb-2"
              >
                {{ t('projectManagement.designCenter') }}
              </h4>

              <!-- 描述 -->
              <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                {{ t('projectManagement.designCenterDesc') }}
              </p>

              <!-- 统计信息 -->
              <div
                class="flex justify-center space-x-4 text-xs text-gray-500 dark:text-gray-400"
              >
                <span class="flex items-center">
                  <svg
                    class="w-4 h-4 mr-1"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                    />
                  </svg>
                  {{ t('projectManagement.pageCount', { count: selectedProject?.pageCount || 0 }) }}
                </span>
                <span class="flex items-center">
                  <svg
                    class="w-4 h-4 mr-1"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M7 4V2a1 1 0 011-1h8a1 1 0 011 1v2m-9 0h10m-9 0V1m10 3V1m0 3l1 1v16a2 2 0 01-2 2H6a2 2 0 01-2-2V5l1-1z"
                    />
                  </svg>
                  {{ t('projectManagement.componentCount', { count: selectedProject?.componentCount || 0 }) }}
                </span>
              </div>

              <!-- 操作提示 -->
              <div
                class="mt-4 text-xs text-blue-600 dark:text-blue-400 font-medium"
              >
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
              <h4
                class="text-xl font-semibold text-gray-900 dark:text-white mb-2"
              >
                {{ t('projectManagement.dataCenter') }}
              </h4>

              <!-- 描述 -->
              <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                {{ t('projectManagement.dataCenterDesc') }}
              </p>

              <!-- 统计信息 -->
              <div
                class="flex justify-center space-x-4 text-xs text-gray-500 dark:text-gray-400"
              >
                <span class="flex items-center">
                  <svg
                    class="w-4 h-4 mr-1"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4"
                    />
                  </svg>
                  {{ t('projectManagement.dataSourceCount', { count: selectedProject?.dataSourceCount || 0 }) }}
                </span>
                <span class="flex items-center">
                  <svg
                    class="w-4 h-4 mr-1"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
                    />
                  </svg>
                  {{ t('projectManagement.scriptCount', { count: selectedProject?.scriptCount || 0 }) }}
                </span>
              </div>

              <!-- 操作提示 -->
              <div
                class="mt-4 text-xs text-green-600 dark:text-green-400 font-medium"
              >
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
              {{ t('projectManagement.lastUpdated') }}: {{ formatDateTime(selectedProject.updatedAt) }}
            </span>
          </div>
          <div class="space-x-2">
            <el-button type="success" @click="exportProject(selectedProject)">
              <el-icon class="mr-1"><Download /></el-icon>
              {{ t('projectManagement.exportProject') }}
            </el-button>
            <el-button @click="projectDialogVisible = false">{{ t('projectManagement.cancel') }}</el-button>
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
  </div>
</template>

<script>
import {
  ref,
  reactive,
  computed,
  onMounted,
  onUnmounted,
  getCurrentInstance,
} from "vue";
import { useI18n } from "vue-i18n";
import { ElMessage, ElMessageBox } from "element-plus";
import JSZip from "jszip";
import { useAuthStore, useAppStore } from "@/store";
import request from "@/utils/request";
import { projectAPI } from "@/api/project.api";
import { ColorTagEnum } from "@/enums";
import { formatDateTime, formatDate, formatCurrency } from "@/utils";
import { initSocket, getSocket } from "@/utils/socket";
import { Storage } from "@/utils/storage";

export default {
  name: "ProjectManagement",
  setup() {
    const { t } = useI18n();
    const { emit } = getCurrentInstance();
    const authStore = useAuthStore();
    const appStore = useAppStore();
    const isDark = computed(() => appStore.isDark);

    // 当前用户信息
    const currentUser = computed(() => authStore.userInfo);

    // 状态
    const loading = ref(false);
    const createLoading = ref(false);
    const editLoading = ref(false);
    const operationLoading = ref(false);
    const batchOperationLoading = ref(false);

    // 对话框显示状态
    const showCreateDialog = ref(false);
    const showEditDialog = ref(false);
    const showOperationDialog = ref(false);
    const showDeployDialog = ref(false);
    const projectDialogVisible = ref(false);

    // 视图模式
    const viewMode = ref("card"); // 'card' 或 'list'

    // 选择模式状态
    const selectionMode = ref(false);
    const selectedProjects = ref([]); // 选中的工程ID列表
    const deployStateRefreshTimer = ref(null);

    // 当前操作的工程
    const currentProject = ref(null);
    const selectedProject = ref(null);

    // 部署相关状态
    const deployLoading = ref(false);
    const deployForm = reactive({
      project: null,
      currentMode: null,
      mode: "RELEASE",
      version: "",
      targetNodes: [],
    });
    const projectVersions = ref([]);
    const availableNodes = ref([]);
    const nodeModes = reactive({}); // { nodeId: 'DEV' | 'RELEASE' | null }
    const projectDeployState = reactive({}); // { projectId: { deployed: boolean, mode: 'DEV'|'RELEASE'|null } }

    // 工程列表和分页
    const projectList = ref([]);
    const pagination = reactive({
      page: 1,
      limit: 10,
      total: 0,
      totalPages: 0,
    });

    // 搜索表单
    const searchForm = reactive({
      name: "",
    });
    const searchTimer = ref(null);

    // 颜色标签选项
    const colorTagOptions = [
      { value: ColorTagEnum.BLUE, label: t("projectManagement.colorBlue") },
      { value: ColorTagEnum.RED, label: t("projectManagement.colorRed") },
      { value: ColorTagEnum.GREEN, label: t("projectManagement.colorGreen") },
      { value: ColorTagEnum.YELLOW, label: t("projectManagement.colorYellow") },
      { value: ColorTagEnum.PURPLE, label: t("projectManagement.colorPurple") },
      { value: ColorTagEnum.PINK, label: t("projectManagement.colorPink") },
      { value: ColorTagEnum.GRAY, label: t("projectManagement.colorGray") },
    ];

    // 创建工程表单
    const createForm = reactive({
      name: "",
      description: "",
      colorTag: ColorTagEnum.BLUE,
    });

    // 创建表单验证规则
    const createFormRules = {
      name: [
        {
          required: true,
          message: t("projectManagement.nameRequired"),
          trigger: "blur",
        },
        {
          min: 2,
          max: 100,
          message: t("projectManagement.nameLength"),
          trigger: "blur",
        },
      ],
    };

    // 编辑工程表单
    const editForm = reactive({
      id: "",
      name: "",
      description: "",
      colorTag: ColorTagEnum.BLUE,
    });

    // 编辑表单验证规则
    const editFormRules = {
      name: [
        {
          required: true,
          message: t("projectManagement.nameRequired"),
          trigger: "blur",
        },
        {
          min: 2,
          max: 100,
          message: t("projectManagement.nameLength"),
          trigger: "blur",
        },
      ],
    };

    // 表单引用
    const createFormRef = ref(null);
    const editFormRef = ref(null);

    // 检查是否可以管理工程
    const canManageProjects = computed(() => {
      return ["SYSTEM_ADMIN", "PROJECT_ADMIN"].includes(
        currentUser.value?.role,
      );
    });

    // 检查是否可以执行运维操作
    const canPerformOps = computed(() => {
      return ["SYSTEM_ADMIN", "OPS_ADMIN"].includes(currentUser.value?.role);
    });

    /**
     * 统一提取接口错误信息，兼容 message/error 字段。
     * @param {any} error - 异常对象
     * @param {string} fallback - 兜底文案
     * @returns {string}
     */
    const getApiErrorMessage = (error, fallback) => {
      return (
        error?.response?.data?.message ||
        error?.response?.data?.error ||
        error?.message ||
        fallback
      );
    };

    /**
     * 将 HEX 颜色转为 RGB 对象。
     * @param {string} hex - HEX 颜色值
     * @returns {{r:number,g:number,b:number}|null}
     */
    const hexToRgb = (hex) => {
      if (!hex || typeof hex !== "string") return null;
      const normalized = hex.replace("#", "");
      if (![3, 6].includes(normalized.length)) return null;
      const fullHex =
        normalized.length === 3
          ? normalized
              .split("")
              .map((c) => c + c)
              .join("")
          : normalized;
      const num = Number.parseInt(fullHex, 16);
      if (Number.isNaN(num)) return null;
      return {
        r: (num >> 16) & 255,
        g: (num >> 8) & 255,
        b: num & 255,
      };
    };

    /**
     * 根据主题生成工程卡片背景样式。
     * @param {object} project - 工程对象
     * @returns {Record<string,string>} 行内样式
     */
    const getProjectCardStyle = (project) => {
      const baseColor = project?.colorTag || "#3b82f6";
      const rgb = hexToRgb(baseColor);
      if (!rgb) return { backgroundColor: baseColor };
      if (isDark.value) {
        return {
          background: `linear-gradient(135deg, rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.34) 0%, rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.56) 100%)`,
        };
      }
      return {
        background: `linear-gradient(135deg, rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.88) 0%, rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.96) 100%)`,
      };
    };

    // 获取工程列表
    const fetchProjects = async () => {
      loading.value = true;
      try {
        const params = {
          page: pagination.page,
          limit: pagination.limit,
          ...searchForm,
        };

        // 移除空值
        Object.keys(params).forEach((key) => {
          if (!params[key]) delete params[key];
        });

        const response = await projectAPI.getProjects(params);

        projectList.value = response.data.projects || [];
        pagination.total = response.pagination?.total || 0;
        pagination.totalPages = response.pagination?.totalPages || 0;
        await fetchProjectDeployState();
      } catch (error) {
        ElMessage.error(
          t("projectManagement.fetchFailed", {
            message: error.response?.data?.message || error.message,
          }),
        );
      } finally {
        loading.value = false;
      }
    };

    // 执行搜索
    const executeSearch = () => {
      pagination.page = 1;
      fetchProjects();
    };

    // 搜索处理（防抖）
    const handleSearch = () => {
      if (searchTimer.value) {
        window.clearTimeout(searchTimer.value);
      }
      searchTimer.value = window.setTimeout(() => {
        executeSearch();
      }, 300);
    };

    // 重置搜索
    const resetSearch = () => {
      if (searchTimer.value) {
        window.clearTimeout(searchTimer.value);
      }
      Object.keys(searchForm).forEach((key) => {
        searchForm[key] = "";
      });
      pagination.page = 1;
      fetchProjects();
    };

    // 分页大小改变
    const handleSizeChange = (size) => {
      pagination.limit = size;
      pagination.page = 1;
      fetchProjects();
    };

    // 页码改变
    const handleCurrentChange = (page) => {
      pagination.page = page;
      fetchProjects();
    };

    // 创建工程
    const handleCreateProject = async () => {
      if (!createFormRef.value) return;

      try {
        await createFormRef.value.validate();
      } catch {
        return;
      }

      createLoading.value = true;
      try {
        const projectData = {
          name: createForm.name,
          description: createForm.description || "",
          colorTag: createForm.colorTag,
        };

        await projectAPI.createProject(projectData);

        ElMessage.success(t("projectManagement.createSuccess"));
        showCreateDialog.value = false;
        resetCreateForm();
        fetchProjects();
      } catch (error) {
        ElMessage.error(
          t("projectManagement.createFailed", {
            message: error.response?.data?.message || error.message,
          }),
        );
      } finally {
        createLoading.value = false;
      }
    };

    // 重置创建表单
    const resetCreateForm = () => {
      Object.keys(createForm).forEach((key) => {
        if (key === "colorTag") {
          createForm[key] = ColorTagEnum.BLUE;
        } else {
          createForm[key] = "";
        }
      });
      if (createFormRef.value) {
        createFormRef.value.clearValidate();
      }
    };

    // 导入工程
    const importProject = () => {
      handleImportProject();
    };

    // 导出工程
    const exportProject = (project) => {
      if (!project) return;
      handleExportProject(project);
    };

    // 编辑工程
    const editProject = (project) => {
      editForm.id = project.id;
      editForm.name = project.name;
      editForm.description = project.description;
      editForm.colorTag = project.colorTag || ColorTagEnum.BLUE;
      showEditDialog.value = true;
    };

    // 更新工程
    const handleUpdateProject = async () => {
      if (!editFormRef.value) return;

      try {
        await editFormRef.value.validate();
      } catch {
        return;
      }

      editLoading.value = true;
      try {
        const projectData = {
          name: editForm.name,
          description: editForm.description || "",
          colorTag: editForm.colorTag,
        };

        await projectAPI.updateProject(editForm.id, projectData);

        ElMessage.success(t("projectManagement.updateSuccess"));
        showEditDialog.value = false;
        fetchProjects();
      } catch (error) {
        ElMessage.error(
          t("projectManagement.updateDevFailed", {
            message: error.response?.data?.message || error.message,
          }),
        );
      } finally {
        editLoading.value = false;
      }
    };

    // 删除工程
    const deleteProject = async (project) => {
      try {
        await ElMessageBox.confirm(
          t("projectManagement.deleteConfirm", { name: project.name }),
          t("projectManagement.deleteConfirmTitle"),
          {
            confirmButtonText: t("projectManagement.deleteConfirmButton"),
            cancelButtonText: t("projectManagement.cancel"),
            type: "warning",
          },
        );

        await projectAPI.deleteProject(project.id);
        ElMessage.success(t("projectManagement.deleteSuccess"));
        fetchProjects();
      } catch (error) {
        if (error !== "cancel") {
          ElMessage.error(
            t("projectManagement.deleteFailed", {
              message: error.response?.data?.message || error.message,
            }),
          );
        }
      }
    };

    // 导出工程
    const handleExportProject = async (project) => {
      if (!project?.id) return;
      try {
        const response = await projectAPI.exportProject(project.id);
        const blob =
          response instanceof window.Blob
            ? response
            : new window.Blob([response], { type: "application/zip" });
        const url = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = url;
        link.download = `${project.name || "project"}.zip`;
        link.click();
        URL.revokeObjectURL(url);
        ElMessage.success(t("projectManagement.exportSuccess"));
      } catch (error) {
        ElMessage.error(
          t("projectManagement.exportFailed", {
            message: error.response?.data?.message || error.message,
          }),
        );
      }
    };

    // 导入工程
    const handleImportProject = () => {
      const input = document.createElement("input");
      input.type = "file";
      input.accept = ".json,.zip,application/json,application/zip";
      input.onchange = async (event) => {
        const file = event.target.files?.[0];
        if (!file) return;
        try {
          let payload = null;
          if (file.name.toLowerCase().endsWith(".zip")) {
            const zip = await JSZip.loadAsync(file);
            const readJson = async (path) => {
              const entry = zip.file(path);
              if (!entry) return null;
              const content = await entry.async("string");
              return JSON.parse(content);
            };

            const projectJson = await readJson("project.json");
            if (!projectJson?.project) {
              ElMessage.error(t("projectManagement.importMissingManifest"));
              return;
            }
            const globalVariables = await readJson(
              "designer/global-variables.json",
            );
            const globalScripts = await readJson(
              "designer/global-scripts.json",
            );
            const projectVariables = await readJson(
              "designer/project-variables.json",
            );
            const pageIndex = await readJson("designer/pages/index.json");
            const pages = [];
            if (Array.isArray(pageIndex)) {
              for (const item of pageIndex) {
                if (!item?.file) continue;
                const schema = await readJson(`designer/pages/${item.file}`);
                pages.push({
                  page: {
                    id: item.id,
                    name: item.name,
                    type: item.type,
                    parentId: item.parentId ?? null,
                    sortOrder: item.sortOrder ?? 0,
                  },
                  schemaContent: schema,
                });
              }
            }

            const datacenter = {
              connections: await readJson("datacenter/connections.json"),
              relationalConfigs: await readJson(
                "datacenter/relational-configs.json",
              ),
              queries: await readJson("datacenter/queries.json"),
              mqttConfigs: await readJson("datacenter/mqtt-configs.json"),
              mqttSubscriptions: await readJson(
                "datacenter/mqtt-subscriptions.json",
              ),
              mqttTagGroups: await readJson("datacenter/mqtt-tag-groups.json"),
              mqttTags: await readJson("datacenter/mqtt-tags.json"),
              datapoints: await readJson("datacenter/datapoints.json"),
            };

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
            };
          } else {
            const text = await file.text();
            const parsed = JSON.parse(text);
            payload = parsed?.payload || parsed;
          }

          if (!payload) {
            ElMessage.error(t("projectManagement.importInvalidFormat"));
            return;
          }
          await projectAPI.importProject({ payload });
          ElMessage.success(t("projectManagement.importSuccess"));
          fetchProjects();
        } catch (error) {
          ElMessage.error(
            t("projectManagement.importFailed", {
              message: error.response?.data?.message || error.message,
            }),
          );
        }
      };
      input.click();
    };

    // 显示运维操作对话框
    const openOperationDialog = (project) => {
      currentProject.value = project;
      showOperationDialog.value = true;
    };

    // 执行运维操作
    const performOperation = async (operation) => {
      if (!currentProject.value) return;

      operationLoading.value = true;
      try {
        await projectAPI.performOperation(currentProject.value.id, operation);

        ElMessage.success(
          t("projectManagement.operationSuccess", {
            action:
              operation === "start"
                ? t("projectManagement.actionStart")
                : operation === "stop"
                  ? t("projectManagement.actionStop")
                  : operation === "restart"
                    ? t("projectManagement.actionRestart")
                    : operation === "deploy"
                      ? t("projectManagement.actionDeploy")
                      : t("projectManagement.actionBackup"),
          }),
        );

        showOperationDialog.value = false;
        currentProject.value = null;
      } catch (error) {
        ElMessage.error(
          t("projectManagement.operationFailed", {
            message: error.response?.data?.message || error.message,
          }),
        );
      } finally {
        operationLoading.value = false;
      }
    };

    // 切换选择模式
    const toggleSelectionMode = () => {
      selectionMode.value = !selectionMode.value;
      if (!selectionMode.value) {
        // 退出选择模式时清除选择
        selectedProjects.value = [];
      }
    };

    // 卡片点击处理
    const handleCardClick = (project) => {
      if (selectionMode.value) {
        // 在选择模式下点击切换选中状态
        toggleProjectSelection(project.id);
      } else {
        // 非选择模式下打开工程详情
        openProjectDialog(project);
      }
    };

    // 工程名称点击处理（列表视图）
    const handleProjectNameClick = (project) => {
      if (selectionMode.value) {
        // 在选择模式下点击切换选中状态
        toggleProjectSelection(project.id);
      } else {
        // 非选择模式下打开工程详情
        openProjectDialog(project);
      }
    };

    // 切换工程选中状态
    const toggleProjectSelection = (projectId, value = null) => {
      const index = selectedProjects.value.indexOf(projectId);
      if (value === true || (value === null && index === -1)) {
        // 选中
        if (index === -1) {
          selectedProjects.value.push(projectId);
        }
      } else if (value === false || (value === null && index !== -1)) {
        // 取消选中
        if (index !== -1) {
          selectedProjects.value.splice(index, 1);
        }
      }
    };

    // 处理表格选择变化
    const handleSelectionChange = (selection) => {
      selectedProjects.value = selection.map((p) => p.id);
    };

    // 批量导出工程
    const batchExportProjects = async () => {
      if (selectedProjects.value.length === 0) {
        return ElMessage.warning(t("projectManagement.selectForExport"));
      }

      batchOperationLoading.value = true;
      try {
        const zip = new JSZip();
        let successCount = 0;
        let failCount = 0;

        for (const projectId of selectedProjects.value) {
          const targetProject = projectList.value.find((item) => item.id === projectId);
          if (!targetProject) {
            failCount++;
            continue;
          }

          try {
            const response = await projectAPI.exportProject(projectId);
            const blob =
              response instanceof window.Blob
                ? response
                : new window.Blob([response], { type: "application/zip" });
            const arrayBuffer = await blob.arrayBuffer();
            zip.file(`${targetProject.name || projectId}.zip`, arrayBuffer);
            successCount++;
          } catch (error) {
            console.error(`导出工程 ${projectId} 失败:`, error);
            failCount++;
          }
        }

        if (successCount === 0) {
          ElMessage.error(t("projectManagement.batchExportFailed"));
          return;
        }

        const packageBlob = await zip.generateAsync({ type: "blob" });
        const url = URL.createObjectURL(packageBlob);
        const link = document.createElement("a");
        link.href = url;
        link.download = `projects-export-${Date.now()}.zip`;
        link.click();
        URL.revokeObjectURL(url);

        if (failCount > 0) {
          ElMessage.warning(
            t("projectManagement.batchExportSummary", {
              success: successCount,
              fail: failCount,
            }),
          );
        } else {
          ElMessage.success(
            t("projectManagement.batchExportAllSuccess", {
              success: successCount,
            }),
          );
        }
      } catch (error) {
        ElMessage.error(
          t("projectManagement.batchExportError", {
            message: error.response?.data?.message || error.message,
          }),
        );
      } finally {
        batchOperationLoading.value = false;
      }
    };

    // 批量删除工程
    const batchDeleteProjects = async () => {
      if (selectedProjects.value.length === 0) {
        return ElMessage.warning(t("projectManagement.selectForDelete"));
      }
      batchOperationLoading.value = true;
      try {
        await ElMessageBox.confirm(
          t("projectManagement.batchDeleteConfirm", {
            count: selectedProjects.value.length,
          }),
          t("projectManagement.batchDeleteTitle"),
          {
            confirmButtonText: t("projectManagement.deleteConfirmButton"),
            cancelButtonText: t("projectManagement.cancel"),
            type: "warning",
          },
        );

        // 逐个删除工程
        let successCount = 0;
        let failCount = 0;
        for (const projectId of selectedProjects.value) {
          try {
            await projectAPI.deleteProject(projectId);
            successCount++;
          } catch (error) {
            console.error(`删除工程 ${projectId} 失败:`, error);
            failCount++;
          }
        }

        if (successCount > 0) {
          ElMessage.success(
            t("projectManagement.batchDeleteSuccess", {
              success: successCount,
            }),
          );
        }
        if (failCount > 0) {
          ElMessage.warning(
            t("projectManagement.batchDeleteFail", {
              fail: failCount,
            }),
          );
        }

        // 清除选择并刷新列表
        selectedProjects.value = [];
        fetchProjects();
      } catch (error) {
        if (error !== "cancel") {
          ElMessage.error(
            t("projectManagement.batchDeleteError", {
              message: error.response?.data?.message || error.message,
            }),
          );
        }
      } finally {
        batchOperationLoading.value = false;
      }
    };

    // 打开工程功能选择弹窗
    const openProjectDialog = (project) => {
      selectedProject.value = project;
      projectDialogVisible.value = true;
    };

    // 打开设计中心
    const openDesignCenter = (project) => {
      projectDialogVisible.value = false;
      // 在标签页内打开设计中心（使用 iframe 嵌入）
      import("@/components/EmbeddedApp.vue").then((module) => {
        const EmbeddedApp = module.default;
        emit("open-tab", {
          key: `design-center-${project.id}`,
          title: `${project.name} - ${t("projectManagement.designCenter")}`,
          component: EmbeddedApp,
          props: {
            appType: "designer",
            project: project,
          },
          icon: "design",
        });
      });
    };

    // 打开数据中心
    const openDataCenter = (project) => {
      projectDialogVisible.value = false;
      // 在标签页内打开数据中心（使用 iframe 嵌入）
      import("@/components/EmbeddedApp.vue").then((module) => {
        const EmbeddedApp = module.default;
        emit("open-tab", {
          key: `data-center-${project.id}`,
          title: `${project.name} - ${t("projectManagement.dataCenter")}`,
          component: EmbeddedApp,
          props: {
            appType: "datacenter",
            project: project,
          },
          icon: "database",
        });
      });
    };

    // 打开运维管理并按工程过滤，运行态操作统一在运维管理执行。
    const openOpsManagement = (project) => {
      if (!isProjectDeployed(project)) {
        ElMessage.warning(`${t("projectManagement.notDeployed")}，请先完成发布并部署`);
        return;
      }
      emit("open-tab", "ops-management");
      [80, 220, 420].forEach((delay) => {
        window.setTimeout(() => {
          window.dispatchEvent(
            new window.CustomEvent("ops:set-project-filter", {
              detail: { projectId: project?.id || "", projectName: project?.name || "" },
            }),
          );
        }, delay);
      });
    };

    // 同步工程部署状态，用于工程列表的运行模式展示与入口控制。
    const fetchProjectDeployState = async () => {
      Object.keys(projectDeployState).forEach((key) => {
        delete projectDeployState[key];
      });

      try {
        const res = await request.get("/nodes", {
          params: {
            page: 1,
            pageSize: 500,
            approvalStatus: "approved",
          },
        });
        const payload = res?.data ?? res;
        if (payload?.success === false) {
          return;
        }
        const nodes = payload?.data?.items || payload?.items || [];
        nodes.forEach((node) => {
          const deployments = Array.isArray(node?.deployments) ? node.deployments : [];
          deployments.forEach((deploy) => {
            const projectId = deploy?.projectId;
            if (!projectId) return;
            if (!projectDeployState[projectId]) {
              projectDeployState[projectId] = {
                deployed: true,
                mode: deploy.mode || null,
              };
            } else if (!projectDeployState[projectId].mode && deploy.mode) {
              projectDeployState[projectId].mode = deploy.mode;
            }
          });
        });
      } catch (error) {
        console.error("获取工程部署状态失败:", error);
      }
    };

    const isProjectDeployed = (project) => {
      return Boolean(projectDeployState[project?.id]?.deployed);
    };

    // 获取节点当前模式
    const getNodeMode = (nodeId) => {
      return nodeModes[nodeId] || null;
    };

    // 获取模式标签类型
    const getModeTagType = (mode) => {
      if (mode === "DEV") return "warning";
      if (mode === "RELEASE") return "success";
      return "info";
    };

    // 获取工程运行模式显示
    const getProjectModeDisplay = (project) => {
      const mode = projectDeployState[project?.id]?.mode;
      if (mode === "DEV") return t("projectManagement.modeDisplayDev");
      if (mode === "RELEASE") return t("projectManagement.modeDisplayRelease");
      return t("projectManagement.notDeployed");
    };

    // 获取工程模式标签类型
    const getProjectModeTagType = (project) => {
      const mode = projectDeployState[project?.id]?.mode || null;
      return getModeTagType(mode);
    };

    const scheduleDeployStateRefresh = () => {
      if (deployStateRefreshTimer.value) {
        window.clearTimeout(deployStateRefreshTimer.value);
      }
      deployStateRefreshTimer.value = window.setTimeout(() => {
        fetchProjectDeployState();
      }, 120);
    };

    const setupRealtimeUpdates = () => {
      const tenantId = Storage.getTenantId() || "default";
      const socket = initSocket(tenantId);
      socket.on("ops:deploy:status", (data = {}) => {
        if (!data?.projectId) {
          return;
        }
        scheduleDeployStateRefresh();
      });
    };

    // 打开部署对话框
    const openDeployDialog = async (project) => {
      deployForm.project = project;
      deployForm.currentMode = null;
      deployForm.mode = "RELEASE";
      deployForm.version = "";
      deployForm.targetNodes = [];

      try {
        // 获取可用节点列表
        const nodesRes = await request.get("/nodes", {
          params: { approvalStatus: "approved", status: "online" },
        });
        // 兼容不同的响应结构
        const nodesPayload = nodesRes?.data ?? nodesRes;
        const nodesData = nodesPayload?.data || nodesPayload || {};
        availableNodes.value = nodesData.items || nodesData || [];

        // 并行获取每个节点的当前部署模式，避免串行请求导致弹窗打开慢。
        await Promise.all(availableNodes.value.map(async (node) => {
          try {
            const modeRes = await request.get(
              `/deployments/project/${project.id}/node/${node.id}/mode`,
            );
            const modePayload = modeRes?.data ?? modeRes;
            nodeModes[node.id] = modePayload?.data?.mode || null;
          } catch {
            nodeModes[node.id] = null;
          }
        }));

        // 获取版本列表
        const versionsRes = await request.get(
          `/publish/${project.id}/versions`,
        );
        // 兼容不同的响应结构
        const versionsPayload = versionsRes?.data ?? versionsRes;
        const versionsData = versionsPayload?.data || versionsPayload || {};
        const allVersions = versionsData.items || versionsData || [];
        // 仅保留构建成功的版本用于部署选择。
        projectVersions.value = allVersions.filter((item) => item?.status === "success");

        // 设置当前模式（如果有节点部署的话）
        if (availableNodes.value.length > 0) {
          deployForm.currentMode = nodeModes[availableNodes.value[0].id];
        }

        showDeployDialog.value = true;
      } catch (error) {
        ElMessage.error(
          t("projectManagement.loadDataFailed", {
            message: getApiErrorMessage(error, t("opsManagement.operationFailedFallback")),
          }),
        );
      }
    };

    // 确认部署
    const confirmDeploy = async () => {
      const { project, mode, targetNodes, version } = deployForm;

      if (targetNodes.length === 0) {
        return ElMessage.warning(t("projectManagement.selectTargetNodes"));
      }

      if (mode === "RELEASE" && !version) {
        return ElMessage.warning(t("projectManagement.selectVersion"));
      }

      deployLoading.value = true;
      try {
        if (mode === "RELEASE") {
          // RELEASE模式：需要选择或创建版本
          let deploymentId;

          // 检查是否选择已有版本
          const existingVersion = projectVersions.value.find(
            (v) => v.version === version,
          );
          if (existingVersion) {
            deploymentId = existingVersion.id;
          } else {
            // 需要先发布新版本
            const publishRes = await request.post(`/publish/${project.id}`, {
              version,
              name: `v${version}`,
              description: t("projectManagement.publishByDeployDialog"),
            });
            const publishPayload = publishRes?.data ?? publishRes;
            if (publishPayload?.success === false) {
              throw new Error(
                publishPayload?.message || t("projectManagement.publishFailed"),
              );
            }
            deploymentId = publishPayload?.data?.id || publishPayload?.id;
            if (!deploymentId) {
              throw new Error(t("projectManagement.publishFailed"));
            }
          }

          // 检查是否有DEV实例需要停止
          const devNodes = targetNodes.filter((n) => getNodeMode(n) === "DEV");
          if (devNodes.length > 0) {
            await ElMessageBox.confirm(
              t("projectManagement.switchConfirm", { count: devNodes.length }),
              t("projectManagement.switchConfirmTitle"),
              { type: "warning" },
            );
          }

          // 部署到节点
          const deployRes = await request.post(
            `/deployments/${deploymentId}/deploy`,
            {
              nodeIds: targetNodes,
              mode: "RELEASE",
              runtimeConfig: {},
            },
          );
          const deployPayload = deployRes?.data ?? deployRes;
          if (deployPayload?.success === false) {
            throw new Error(
              deployPayload?.message || t("opsManagement.operationFailedFallback"),
            );
          }
          const { success: successCount = 0, failed: failCount = 0 } =
            deployPayload?.data?.summary || deployPayload?.summary || {};
          if (failCount > 0) {
            ElMessage.warning(
              `${t("projectManagement.deploySuccess", { count: successCount })}，失败 ${failCount} 个节点`,
            );
          } else {
            ElMessage.success(
              t("projectManagement.deploySuccess", { count: successCount }),
            );
          }
        } else {
          // DEV模式：需要先确保没有RELEASE部署
          const releaseNodes = targetNodes.filter(
            (n) => getNodeMode(n) === "RELEASE",
          );
          if (releaseNodes.length > 0) {
            return ElMessage.warning(
              t("projectManagement.releaseConflict"),
            );
          }

          // DEV模式按工程直接部署，无需发布版本
          const deployRes = await request.post(
            `/deployments/project/${project.id}/deploy-dev`,
            { nodeIds: targetNodes },
          );
          const deployPayload = deployRes?.data ?? deployRes;
          if (deployPayload?.success === false) {
            throw new Error(
              deployPayload?.message || t("opsManagement.operationFailedFallback"),
            );
          }
          const { success: successCount = 0, failed: failCount = 0 } =
            deployPayload?.data?.summary || deployPayload?.summary || {};
          if (failCount > 0) {
            ElMessage.warning(
              `${t("projectManagement.devDeploySuccess", { count: successCount })}，失败 ${failCount} 个节点`,
            );
          } else {
            ElMessage.success(
              t("projectManagement.devDeploySuccess", { count: successCount }),
            );
          }
        }

        showDeployDialog.value = false;
        await fetchProjects();
      } catch (error) {
        if (error !== "cancel") {
          ElMessage.error(
            getApiErrorMessage(error, t("opsManagement.operationFailedFallback")),
          );
        }
      } finally {
        deployLoading.value = false;
      }
    };

    // 组件挂载时获取数据
    onMounted(() => {
      fetchProjects();
      setupRealtimeUpdates();
    });

    onUnmounted(() => {
      if (searchTimer.value) {
        window.clearTimeout(searchTimer.value);
      }
      if (deployStateRefreshTimer.value) {
        window.clearTimeout(deployStateRefreshTimer.value);
        deployStateRefreshTimer.value = null;
      }
      const socket = getSocket();
      if (socket) {
        socket.off("ops:deploy:status");
      }
    });

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
      projectDialogVisible,

      // 视图
      viewMode,

      // 选择模式
      selectionMode,
      selectedProjects,
      currentProject,
      selectedProject,

      // 数据
      projectList,
      pagination,
      searchForm,
      colorTagOptions,
      currentUser,

      // 部署相关
      deployLoading,
      deployForm,
      projectVersions,
      availableNodes,
      nodeModes,

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
      canPerformOps,
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
      openDesignCenter,
      openDataCenter,
      isProjectDeployed,
      openOpsManagement,
      openDeployDialog,
      exportProject,
      batchExportProjects,
      batchDeleteProjects,
      confirmDeploy,
      getNodeMode,
      getModeTagType,
      getProjectCardStyle,
      getProjectModeDisplay,
      getProjectModeTagType,
      formatDateTime,
      formatDate,
      formatCurrency,
    };
  },
};
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
</style>
