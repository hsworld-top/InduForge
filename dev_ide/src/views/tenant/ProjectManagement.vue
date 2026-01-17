<template>
  <div class="project-management">
    <!-- 页面标题和操作栏 -->
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">工程管理</h1>
      <div class="flex items-center space-x-4">
        <!-- 视图切换 -->
        <div class="flex items-center bg-gray-100 dark:bg-gray-700 rounded-lg p-1">
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
            卡片
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
            列表
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
          添加工程
        </el-button>
        <el-button
          v-if="canManageProjects"
          type="default"
          @click="handleImportProject"
        >
          <el-icon class="mr-2"><Upload /></el-icon>
          导入工程
        </el-button>

      </div>
    </div>

    <!-- 搜索和筛选栏 -->
    <div
      class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6 mb-6"
    >
      <el-form :inline="true" :model="searchForm" class="flex flex-wrap gap-4">
        <el-form-item label="工程名称">
          <el-input
            v-model="searchForm.name"
            placeholder="输入工程名称搜索"
            clearable
            style="width: 200px"
            @input="handleSearch"
          />
        </el-form-item>
        <el-form-item>
          <el-button @click="resetSearch" type="default">
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 工程列表 -->
    <div
      class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 "  
    >
      <!-- 卡片视图 -->
      <div v-if="viewMode === 'card'" class="p-6 ">
        <div v-if="projectList.length === 0 && !loading" class="text-center py-12">
          <el-empty description="暂无工程数据" />
        </div>
        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <div
            v-for="project in projectList"
            :key="project.id"
            class="project-card border border-gray-200 dark:border-gray-600 rounded-lg p-6 hover:shadow-lg transition-all duration-200 cursor-pointer"
            :style="{ backgroundColor: project.colorTag || '#3b82f6' }"
            @click="openProjectDialog(project)"
          >
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
              {{ project.description || '暂无描述' }}
            </p>

            <!-- 工程信息 -->
            <div class="space-y-2 mb-4">
              <div class="flex justify-between text-sm">
                <span class="text-white opacity-75">创建者:</span>
                <span class="text-white font-medium">{{
                  project.creator?.fullName || '未知'
                }}</span>
              </div>
            </div>

            <!-- 操作按钮 -->
            <div class="flex justify-end space-x-2">
              <el-button
                v-if="canManageProjects"
                type="primary"
                size="small"
                @click.stop="editProject(project)"
              >
                编辑
              </el-button>
              <el-button
                v-if="canPerformOps"
                type="warning"
                size="small"
                @click.stop="openOperationDialog(project)"
              >
                运维
              </el-button>
              <el-button
                v-if="canManageProjects"
                type="success"
                size="small"
                @click.stop="handleExportProject(project)"
              >
                <el-icon class="mr-1"><Download /></el-icon>
                导出
              </el-button>
              <el-button
                v-if="canManageProjects"
                type="danger"
                size="small"
                @click.stop="deleteProject(project)"
              >
                删除
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
        >
          <el-table-column label="颜色" width="80">
            <template #default="scope">
              <div
                class="w-6 h-6 rounded-full border-2 border-white shadow-sm"
                :style="{ backgroundColor: scope.row.colorTag || '#3b82f6' }"
              ></div>
            </template>
          </el-table-column>
          <el-table-column prop="name" label="工程名称" width="200">
            <template #default="scope">
              <span
                class="cursor-pointer text-blue-600 hover:text-blue-800 underline"
                @click="openProjectDialog(scope.row)"
              >
                {{ scope.row.name }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="描述" width="200" />
          <el-table-column prop="creator.fullName" label="创建者" width="200" />
          <el-table-column prop="createdAt" label="创建时间" width="200">
            <template #default="scope">
              {{ formatDateTime(scope.row.createdAt) }}
            </template>
          </el-table-column>
          <el-table-column
            label="操作"
            width="min-200"
            fixed="right"
            v-if="canManageProjects || canPerformOps"
          >
            <template #default="scope">
              <el-button
                v-if="canManageProjects"
                type="primary"
                size="small"
                @click="editProject(scope.row)"
                class="mr-2"
              >
                编辑
              </el-button>
              <el-button
                v-if="canPerformOps"
                type="warning"
                size="small"
                @click="openOperationDialog(scope.row)"
                class="mr-2"
              >
                运维
              </el-button>
              <el-button
                v-if="canManageProjects"
                type="success"
                size="small"
                @click="handleExportProject(scope.row)"
                class="mr-2"
              >
                <el-icon class="mr-1"><Download /></el-icon>
                导出
              </el-button>
              <el-button
                v-if="canManageProjects"
                type="danger"
                size="small"
                @click="deleteProject(scope.row)"
              >
                删除
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
          显示第 {{ (pagination.page - 1) * pagination.limit + 1 }} 到
          {{ Math.min(pagination.page * pagination.limit, pagination.total) }} 条， 共
          {{ pagination.total }} 条记录
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
      title="创建工程"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form ref="createFormRef" :model="createForm" :rules="createFormRules" label-width="100px">
        <el-form-item label="工程名称" prop="name">
          <el-input v-model="createForm.name" placeholder="请输入工程名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="createForm.description"
            type="textarea"
            placeholder="请输入工程描述"
            :rows="3"
          />
        </el-form-item>
        <el-form-item label="颜色标签">
          <el-select v-model="createForm.colorTag" placeholder="请选择颜色标签" style="width: 100%">
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
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreateProject" :loading="createLoading">
          创建
        </el-button>
      </template>
    </el-dialog>

    <!-- 编辑工程对话框 -->
    <el-dialog
      v-model="showEditDialog"
      title="编辑工程"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form ref="editFormRef" :model="editForm" :rules="editFormRules" label-width="100px">
        <el-form-item label="工程名称" prop="name">
          <el-input v-model="editForm.name" placeholder="请输入工程名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="editForm.description"
            type="textarea"
            placeholder="请输入工程描述"
            :rows="3"
          />
        </el-form-item>
        <el-form-item label="颜色标签">
          <el-select v-model="editForm.colorTag" placeholder="请选择颜色标签" style="width: 100%">
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
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" @click="handleUpdateProject" :loading="editLoading">
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 运维操作对话框 -->
    <el-dialog
      v-model="showOperationDialog"
      :title="`运维操作 - ${currentProject?.name || ''}`"
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
          启动工程
        </el-button>
        <el-button
          type="warning"
          plain
          block
          @click="performOperation('stop')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2"><VideoPause /></el-icon>
          停止工程
        </el-button>
        <el-button
          type="info"
          plain
          block
          @click="performOperation('restart')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2"><RefreshRight /></el-icon>
          重启工程
        </el-button>
        <el-button
          type="primary"
          plain
          block
          @click="performOperation('deploy')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2"><Upload /></el-icon>
          部署工程
        </el-button>
        <el-button
          type="danger"
          plain
          block
          @click="performOperation('backup')"
          :loading="operationLoading"
        >
          <el-icon class="mr-2"><CopyDocument /></el-icon>
          备份工程
        </el-button>
      </div>
    </el-dialog>

    <!-- 工程功能选择弹窗 -->
    <el-dialog
      v-model="projectDialogVisible"
      :title="'选择功能 - ' + (selectedProject?.name || '未知工程')"
      width="600px"
      center
      :close-on-click-modal="false"
      append-to-body
    >
      <div class="project-dialog-content">
        <!-- 工程信息展示 -->
        <div class="text-center mb-6">
          <div class="inline-flex items-center justify-center w-16 h-16 rounded-full mb-4"
               :style="{ backgroundColor: selectedProject?.colorTag || '#3b82f6' }">
            <svg class="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"/>
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">
            {{ selectedProject?.name }}
          </h3>
          <p class="text-sm text-gray-600 dark:text-gray-400">
            {{ selectedProject?.description || '暂无描述' }}
          </p>
        </div>

        <!-- 功能选择卡片 -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
          <!-- 设计中心卡片 -->
          <div
            class="function-card bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-900/20 dark:to-blue-800/20
                 border-2 border-blue-200 dark:border-blue-700 rounded-xl p-6 cursor-pointer
                 hover:shadow-lg hover:border-blue-300 dark:hover:border-blue-600 transition-all duration-300
                 hover:scale-105"
            @click="openDesignCenter(selectedProject)"
          >
            <div class="text-center">
              <!-- 图标 -->
              <div class="inline-flex items-center justify-center w-16 h-16 bg-blue-500 rounded-full mb-4">
                <svg class="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zM21 5a2 2 0 00-2-2h-4a2 2 0 00-2 2v12a4 4 0 004 4h4a2 2 0 002-2V5z"/>
                </svg>
              </div>

              <!-- 标题 -->
              <h4 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
                设计中心
              </h4>

              <!-- 描述 -->
              <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                拖拽式页面设计，组件配置，样式编辑，数据绑定
              </p>

              <!-- 统计信息 -->
              <div class="flex justify-center space-x-4 text-xs text-gray-500 dark:text-gray-400">
                <span class="flex items-center">
                  <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                  </svg>
                  {{ selectedProject?.pageCount || 0 }} 个页面
                </span>
                <span class="flex items-center">
                  <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4V2a1 1 0 011-1h8a1 1 0 011 1v2m-9 0h10m-9 0V1m10 3V1m0 3l1 1v16a2 2 0 01-2 2H6a2 2 0 01-2-2V5l1-1z"/>
                  </svg>
                  {{ selectedProject?.componentCount || 0 }} 个组件
                </span>
              </div>

              <!-- 操作提示 -->
              <div class="mt-4 text-xs text-blue-600 dark:text-blue-400 font-medium">
                点击进入设计中心 →
              </div>
            </div>
          </div>

          <!-- 数据中心卡片 -->
          <div
            class="function-card bg-gradient-to-br from-green-50 to-green-100 dark:from-green-900/20 dark:to-green-800/20
                 border-2 border-green-200 dark:border-green-700 rounded-xl p-6 cursor-pointer
                 hover:shadow-lg hover:border-green-300 dark:hover:border-green-600 transition-all duration-300
                 hover:scale-105"
            @click="openDataCenter(selectedProject)"
          >
            <div class="text-center">
              <!-- 图标 -->
              <div class="inline-flex items-center justify-center w-16 h-16 bg-green-500 rounded-full mb-4">
                <svg class="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"/>
                </svg>
              </div>

              <!-- 标题 -->
              <h4 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
                数据中心
              </h4>

              <!-- 描述 -->
              <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                数据库设计，脚本编辑，数据连接，多源数据管理
              </p>

              <!-- 统计信息 -->
              <div class="flex justify-center space-x-4 text-xs text-gray-500 dark:text-gray-400">
                <span class="flex items-center">
                  <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4"/>
                  </svg>
                  {{ selectedProject?.dataSourceCount || 0 }} 个数据源
                </span>
                <span class="flex items-center">
                  <svg class="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"/>
                  </svg>
                  {{ selectedProject?.scriptCount || 0 }} 个脚本
                </span>
              </div>

              <!-- 操作提示 -->
              <div class="mt-4 text-xs text-green-600 dark:text-green-400 font-medium">
                点击进入数据中心 →
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
              最后更新: {{ formatDateTime(selectedProject.updatedAt) }}
            </span>
          </div>
          <div class="space-x-2">
            <el-button @click="projectDialogVisible = false">取消</el-button>
            <el-button
              v-if="canManageProjects"
              type="primary"
              @click="editProject(selectedProject)"
            >
              工程设置
            </el-button>
          </div>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, computed, onMounted, getCurrentInstance } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Refresh,
  Grid,
  List,
  Upload,
  Download,
} from '@element-plus/icons-vue'
import JSZip from 'jszip'
import { useAuthStore } from '@/store'
import { projectAPI } from '@/api/project.api'
import { ColorTagEnum, ENUM_LABELS } from '@/enums'
import { formatDateTime, formatDate, formatCurrency } from '@/utils'

export default {
  name: 'ProjectManagement',
  setup() {
    const { emit } = getCurrentInstance()
    const authStore = useAuthStore()

    // 当前用户信息
    const currentUser = computed(() => authStore.userInfo)

    // 状态
    const loading = ref(false)
    const createLoading = ref(false)
    const editLoading = ref(false)
    const operationLoading = ref(false)

    // 对话框显示状态
    const showCreateDialog = ref(false)
    const showEditDialog = ref(false)
    const showOperationDialog = ref(false)
    const projectDialogVisible = ref(false)

    // 视图模式
    const viewMode = ref('card') // 'card' 或 'list'

    // 当前操作的工程
    const currentProject = ref(null)
    const selectedProject = ref(null)

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

    // 颜色标签选项
    const colorTagOptions = [
      { value: ColorTagEnum.BLUE, label: ENUM_LABELS[ColorTagEnum.BLUE] },
      { value: ColorTagEnum.RED, label: ENUM_LABELS[ColorTagEnum.RED] },
      { value: ColorTagEnum.GREEN, label: ENUM_LABELS[ColorTagEnum.GREEN] },
      { value: ColorTagEnum.YELLOW, label: ENUM_LABELS[ColorTagEnum.YELLOW] },
      { value: ColorTagEnum.PURPLE, label: ENUM_LABELS[ColorTagEnum.PURPLE] },
      { value: ColorTagEnum.PINK, label: ENUM_LABELS[ColorTagEnum.PINK] },
      { value: ColorTagEnum.GRAY, label: ENUM_LABELS[ColorTagEnum.GRAY] },
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
        { required: true, message: '请输入工程名称', trigger: 'blur' },
        { min: 2, max: 100, message: '工程名称长度在 2 到 100 个字符', trigger: 'blur' },
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
        { required: true, message: '请输入工程名称', trigger: 'blur' },
        { min: 2, max: 100, message: '工程名称长度在 2 到 100 个字符', trigger: 'blur' },
      ],
    }

    // 表单引用
    const createFormRef = ref(null)
    const editFormRef = ref(null)

    // 检查是否可以管理工程
    const canManageProjects = computed(() => {
      return ['SYSTEM_ADMIN', 'PROJECT_ADMIN'].includes(currentUser.value?.role)
    })

    // 检查是否可以执行运维操作
    const canPerformOps = computed(() => {
      return ['SYSTEM_ADMIN', 'OPS_ADMIN'].includes(currentUser.value?.role)
    })

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

        projectList.value = response.data.projects || []
        pagination.total = response.pagination?.total || 0
        pagination.totalPages = response.pagination?.totalPages || 0
      } catch (error) {
        ElMessage.error('获取工程列表失败：' + (error.response?.data?.message || error.message))
      } finally {
        loading.value = false
      }
    }

    // 搜索处理
    const handleSearch = () => {
      pagination.page = 1
      fetchProjects()
    }

    // 重置搜索
    const resetSearch = () => {
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
      } catch (error) {
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

        ElMessage.success('工程创建成功')
        showCreateDialog.value = false
        resetCreateForm()
        fetchProjects()
      } catch (error) {
        ElMessage.error('创建工程失败：' + (error.response?.data?.message || error.message))
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
      } catch (error) {
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

        ElMessage.success('工程更新成功')
        showEditDialog.value = false
        fetchProjects()
      } catch (error) {
        ElMessage.error('更新工程失败：' + (error.response?.data?.message || error.message))
      } finally {
        editLoading.value = false
      }
    }

    // 删除工程
    const deleteProject = async (project) => {
      try {
        await ElMessageBox.confirm(
          `确定要删除工程 "${project.name}" 吗？此操作不可恢复。`,
          '确认删除',
          {
            confirmButtonText: '确定删除',
            cancelButtonText: '取消',
            type: 'warning',
          }
        )

        await projectAPI.deleteProject(project.id)
        ElMessage.success('工程删除成功')
        fetchProjects()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('删除工程失败：' + (error.response?.data?.message || error.message))
        }
      }
    }

    // 导出工程
    const handleExportProject = async (project) => {
      if (!project?.id) return
      try {
        const response = await projectAPI.exportProject(project.id)
        const blob =
          response instanceof Blob
            ? response
            : new Blob([response], { type: 'application/zip' })
        const url = URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.download = `${project.name || 'project'}.zip`
        link.click()
        URL.revokeObjectURL(url)
        ElMessage.success('工程已导出')
      } catch (error) {
        ElMessage.error('导出工程失败：' + (error.response?.data?.message || error.message))
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
              ElMessage.error('未找到project.json')
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
            ElMessage.error('工程数据格式不正确')
            return
          }
          await projectAPI.importProject({ payload })
          ElMessage.success('工程已导入')
          fetchProjects()
        } catch (error) {
          ElMessage.error('导入工程失败：' + (error.response?.data?.message || error.message))
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
          `工程${
            operation === 'start'
              ? '启动'
              : operation === 'stop'
                ? '停止'
                : operation === 'restart'
                  ? '重启'
                  : operation === 'deploy'
                    ? '部署'
                    : '备份'
          }操作成功`
        )

        showOperationDialog.value = false
        currentProject.value = null
      } catch (error) {
        ElMessage.error(`工程操作失败：${error.response?.data?.message || error.message}`)
      } finally {
        operationLoading.value = false
      }
    }

    // 打开工程功能选择弹窗
    const openProjectDialog = (project) => {
      selectedProject.value = project
      projectDialogVisible.value = true
    }

    // 打开设计中心
    const openDesignCenter = (project) => {
      projectDialogVisible.value = false
      // 在标签页内打开设计中心（使用 iframe 嵌入）
      import('@/components/EmbeddedApp.vue').then((module) => {
        const EmbeddedApp = module.default
        emit('open-tab', {
          key: `design-center-${project.id}`,
          title: `${project.name} - 设计中心`,
          component: EmbeddedApp,
          props: {
            appType: 'designer',
            project: project
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
          title: `${project.name} - 数据中心`,
          component: EmbeddedApp,
          props: {
            appType: 'datacenter',
            project: project
          },
          icon: 'database',
        })
      })
    }

    // 组件挂载时获取数据
    onMounted(() => {
      fetchProjects()
    })

    return {
      // 状态
      loading,
      createLoading,
      editLoading,
      operationLoading,
      showCreateDialog,
      showEditDialog,
      showOperationDialog,
      projectDialogVisible,

      // 视图
      viewMode,
      currentProject,
      selectedProject,

      // 数据
      projectList,
      pagination,
      searchForm,
      colorTagOptions,
      currentUser,

      // 表单
      createForm,
      createFormRules,
      editForm,
      editFormRules,

      // 表单引用
      createFormRef,
      editFormRef,

      // 方法
      canManageProjects,
      canPerformOps,
      fetchProjects,
      handleSearch,
      resetSearch,
      handleSizeChange,
      handleCurrentChange,
      handleCreateProject,
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
