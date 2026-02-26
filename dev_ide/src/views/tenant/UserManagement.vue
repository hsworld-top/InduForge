<template>
  <div class="user-management">
    <!-- 页面标题和操作栏 -->
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">用户管理</h1>
      <el-button
        v-if="canManageUsers"
        type="primary"
        @click="showCreateDialog = true"
        class="bg-blue-600 hover:bg-blue-700"
      >
        <el-icon class="mr-2"><Plus /></el-icon>
        添加用户
      </el-button>
    </div>

    <!-- 搜索和筛选栏 -->
    <div
      class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6 mb-6"
    >
      <el-form :inline="true" :model="searchForm" class="flex flex-wrap gap-4">
        <el-form-item label="用户名">
          <el-input
            v-model="searchForm.username"
            placeholder="输入用户名搜索"
            clearable
            style="width: 200px"
            @input="handleSearch"
          />
        </el-form-item>
        <el-form-item label="角色">
          <el-select
            v-model="searchForm.role"
            placeholder="选择角色"
            clearable
            style="width: 150px"
            @change="handleSearch"
          >
            <el-option
              v-for="role in roleOptions"
              :key="role.value"
              :label="role.label"
              :value="role.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select
            v-model="searchForm.status"
            placeholder="选择状态"
            clearable
            style="width: 150px"
            @change="handleSearch"
          >
            <el-option
              v-for="status in statusOptions"
              :key="status.value"
              :label="status.label"
              :value="status.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button @click="resetSearch" type="default">
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 用户列表 -->
    <div
      class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700"
    >
      <el-table
        :data="userList"
        v-loading="loading"
        style="width: 100%"
        :header-cell-style="{ background: '#f9fafb', color: '#374151' }"
      >
        <el-table-column prop="username" label="用户名" width="120" />
        <el-table-column prop="fullName" label="真实姓名" width="120" />
        <el-table-column prop="email" label="邮箱" width="300" />
        <el-table-column prop="role" label="角色" min-width="200">
          <template #default="scope">
            <el-tag :type="getRoleTagType(scope.row.role)">
              {{ getRoleLabel(scope.row.role) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="scope">
            <el-tag :type="getStatusTagType(scope.row.status)">
              {{ getStatusLabel(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="tenant.name" label="所属租户" width="120" />
        <el-table-column prop="lastLoginAt" label="最后登录" width="200">
          <template #default="scope">
            {{ formatDateTime(scope.row.lastLoginAt) }}
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="200">
          <template #default="scope">
            {{ formatDateTime(scope.row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="220" fixed="right" v-if="canManageUsers">
          <template #default="scope">
            <el-button type="primary" size="small" @click="editUser(scope.row)" class="mr-2">
              编辑
            </el-button>
            <el-button type="warning" size="small" @click="resetPassword(scope.row)" class="mr-2">
              重置密码
            </el-button>
            <el-button
              type="danger"
              size="small"
              @click="deleteUser(scope.row)"
              v-if="scope.row.id !== currentUser?.id && scope.row.role !== superAdminRole"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div
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

    <!-- 创建用户对话框 -->
    <el-dialog
      v-model="showCreateDialog"
      title="创建用户"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form ref="createFormRef" :model="createForm" :rules="createFormRules" label-width="100px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="createForm.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="createForm.password"
            type="password"
            placeholder="请输入密码"
            show-password
          />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input
            v-model="createForm.confirmPassword"
            type="password"
            placeholder="请再次输入密码"
            show-password
          />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="createForm.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="真实姓名" prop="fullName">
          <el-input v-model="createForm.fullName" placeholder="请输入真实姓名" />
        </el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="createForm.role" placeholder="请选择角色" style="width: 100%">
            <el-option
              v-for="role in roleOptions"
              :key="role.value"
              :label="role.label"
              :value="role.value"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreateUser" :loading="createLoading">
          创建
        </el-button>
      </template>
    </el-dialog>

    <!-- 编辑用户对话框 -->
    <el-dialog
      v-model="showEditDialog"
      title="编辑用户"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form ref="editFormRef" :model="editForm" :rules="editFormRules" label-width="100px">
        <el-form-item label="用户名">
          <el-input v-model="editForm.username" disabled />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="editForm.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="真实姓名" prop="fullName">
          <el-input v-model="editForm.fullName" placeholder="请输入真实姓名" />
        </el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select
            v-model="editForm.role"
            placeholder="请选择角色"
            style="width: 100%"
            :disabled="!canEditRole"
          >
            <el-option
              v-for="role in roleOptions"
              :key="role.value"
              :label="role.label"
              :value="role.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-select v-model="editForm.status" placeholder="请选择状态" style="width: 100%">
            <el-option
              v-for="status in statusOptions"
              :key="status.value"
              :label="status.label"
              :value="status.value"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" @click="handleUpdateUser" :loading="editLoading">
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog
      v-model="showPasswordDialog"
      title="重置密码"
      width="400px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="passwordFormRef"
        :model="passwordForm"
        :rules="passwordFormRules"
        label-width="100px"
      >
        <el-form-item label="新密码" prop="newPassword">
          <el-input
            v-model="passwordForm.newPassword"
            type="password"
            placeholder="请输入新密码"
            show-password
          />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input
            v-model="passwordForm.confirmPassword"
            type="password"
            placeholder="请再次输入密码"
            show-password
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPasswordDialog = false">取消</el-button>
        <el-button type="primary" @click="handleResetPassword" :loading="passwordLoading">
          重置
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/store'
import { userAPI } from '@/api/user.api'
import { RoleEnum, UserStatusEnum, ENUM_LABELS } from '@/enums'
import { formatDateTime } from '@/utils/date'
import { canManageUsers } from '@/permissions'
import { ROLES } from '@/constants'

export default {
  name: 'TenantUserManagement',
  setup() {
    const authStore = useAuthStore()

    // 当前用户信息
    const currentUser = computed(() => authStore.userInfo)

    // 状态
    const loading = ref(false)
    const createLoading = ref(false)
    const editLoading = ref(false)
    const passwordLoading = ref(false)

    // 对话框显示状态
    const showCreateDialog = ref(false)
    const showEditDialog = ref(false)
    const showPasswordDialog = ref(false)

    // 用户列表和分页
    const userList = ref([])
    const pagination = reactive({
      page: 1,
      limit: 10,
      total: 0,
      totalPages: 0,
    })

    // 搜索表单
    const searchForm = reactive({
      username: '',
      role: '',
      status: '',
    })
    const searchTimer = ref(null)

    // 角色选项
    const roleOptions = [
      { value: RoleEnum.SYSTEM_ADMIN, label: ENUM_LABELS[RoleEnum.SYSTEM_ADMIN] },
      { value: RoleEnum.PROJECT_ADMIN, label: ENUM_LABELS[RoleEnum.PROJECT_ADMIN] },
      { value: RoleEnum.OPS_ADMIN, label: ENUM_LABELS[RoleEnum.OPS_ADMIN] },
      { value: RoleEnum.USER_ADMIN, label: ENUM_LABELS[RoleEnum.USER_ADMIN] },
    ]

    // 状态选项
    const statusOptions = [
      { value: UserStatusEnum.ACTIVE, label: ENUM_LABELS[UserStatusEnum.ACTIVE] },
      { value: UserStatusEnum.INACTIVE, label: ENUM_LABELS[UserStatusEnum.INACTIVE] },
      { value: UserStatusEnum.SUSPENDED, label: ENUM_LABELS[UserStatusEnum.SUSPENDED] },
    ]

    // 创建用户表单
    const createForm = reactive({
      username: '',
      password: '',
      confirmPassword: '',
      email: '',
      fullName: '',
      role: '',
    })

    // 创建表单验证规则
    const createFormRules = {
      username: [
        { required: true, message: '请输入用户名', trigger: 'blur' },
        { min: 3, max: 50, message: '用户名长度在 3 到 50 个字符', trigger: 'blur' },
      ],
      password: [
        { required: true, message: '请输入密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于 6 个字符', trigger: 'blur' },
      ],
      confirmPassword: [
        { required: true, message: '请再次输入密码', trigger: 'blur' },
        {
          validator: (rule, value, callback) => {
            if (value !== createForm.password) {
              callback(new Error('两次输入密码不一致'))
            } else {
              callback()
            }
          },
          trigger: 'blur',
        },
      ],
      email: [{ type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }],
      fullName: [{ required: true, message: '请输入真实姓名', trigger: 'blur' }],
      role: [{ required: true, message: '请选择角色', trigger: 'change' }],
    }

    // 编辑用户表单
    const editForm = reactive({
      id: '',
      username: '',
      email: '',
      fullName: '',
      role: '',
      status: '',
    })

    // 编辑表单验证规则
    const editFormRules = {
      email: [{ type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }],
      fullName: [{ required: true, message: '请输入真实姓名', trigger: 'blur' }],
      role: [{ required: true, message: '请选择角色', trigger: 'change' }],
      status: [{ required: true, message: '请选择状态', trigger: 'change' }],
    }

    // 密码表单
    const passwordForm = reactive({
      userId: '',
      newPassword: '',
      confirmPassword: '',
    })

    // 密码表单验证规则
    const passwordFormRules = {
      newPassword: [
        { required: true, message: '请输入新密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于 6 个字符', trigger: 'blur' },
      ],
      confirmPassword: [
        { required: true, message: '请再次输入密码', trigger: 'blur' },
        {
          validator: (rule, value, callback) => {
            if (value !== passwordForm.newPassword) {
              callback(new Error('两次输入密码不一致'))
            } else {
              callback()
            }
          },
          trigger: 'blur',
        },
      ],
    }

    // 表单引用
    const createFormRef = ref(null)
    const editFormRef = ref(null)
    const passwordFormRef = ref(null)

    // 获取角色标签
    const getRoleLabel = (role) => {
      return ENUM_LABELS[role] || role
    }

    // 获取状态标签
    const getStatusLabel = (status) => {
      return ENUM_LABELS[status] || status
    }

    // 获取角色标签类型
    const getRoleTagType = (role) => {
      const typeMap = {
        [RoleEnum.SYSTEM_ADMIN]: 'danger',
        [RoleEnum.PROJECT_ADMIN]: 'warning',
        [RoleEnum.OPS_ADMIN]: 'info',
        [RoleEnum.USER_ADMIN]: 'success',
      }
      return typeMap[role] || ''
    }

    // 获取状态标签类型
    const getStatusTagType = (status) => {
      const typeMap = {
        [UserStatusEnum.ACTIVE]: 'success',
        [UserStatusEnum.INACTIVE]: 'warning',
        [UserStatusEnum.SUSPENDED]: 'danger',
      }
      return typeMap[status] || ''
    }

    // 检查是否可以编辑角色
    const canEditRole = computed(() => canManageUsers(currentUser.value?.role))

    // 检查是否可以管理用户（创建、编辑、删除）
    const canManageUsersAction = computed(() => canManageUsers(currentUser.value?.role))

    // 获取用户列表
    const fetchUsers = async () => {
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

        const response = await userAPI.getUsers(params)

        userList.value = response.data.users || []
        pagination.total = response.pagination?.total || 0
        pagination.totalPages = response.pagination?.totalPages || 0
      } catch (error) {
        ElMessage.error('获取用户列表失败：' + (error.response?.data?.message || error.message))
      } finally {
        loading.value = false
      }
    }

    // 执行搜索
    const executeSearch = () => {
      pagination.page = 1
      fetchUsers()
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
      fetchUsers()
    }

    // 分页大小改变
    const handleSizeChange = (size) => {
      pagination.limit = size
      pagination.page = 1
      fetchUsers()
    }

    // 页码改变
    const handleCurrentChange = (page) => {
      pagination.page = page
      fetchUsers()
    }

    // 创建用户
    const handleCreateUser = async () => {
      if (!createFormRef.value) return

      try {
        await createFormRef.value.validate()
      } catch {
        return
      }

      createLoading.value = true
      try {
        const userData = {
          username: createForm.username,
          password: createForm.password,
          email: createForm.email || '',
          fullName: createForm.fullName,
          role: createForm.role,
        }

        await userAPI.createUser(userData)

        ElMessage.success('用户创建成功')
        showCreateDialog.value = false
        resetCreateForm()
        fetchUsers()
      } catch (error) {
        ElMessage.error('创建用户失败：' + (error.response?.data?.message || error.message))
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

    // 编辑用户
    const editUser = (user) => {
      editForm.id = user.id
      editForm.username = user.username
      editForm.email = user.email
      editForm.fullName = user.fullName
      editForm.role = user.role
      editForm.status = user.status
      showEditDialog.value = true
    }

    // 更新用户
    const handleUpdateUser = async () => {
      if (!editFormRef.value) return

      try {
        await editFormRef.value.validate()
      } catch {
        return
      }

      editLoading.value = true
      try {
        const userData = {}
        if (editForm.email) {
          userData.email = editForm.email
        }
        if (editForm.fullName) {
          userData.fullName = editForm.fullName
        }
        if (editForm.role) {
          userData.role = editForm.role
        }
        if (editForm.status) {
          userData.status = editForm.status
        }

        await userAPI.updateUser(editForm.id, userData)

        ElMessage.success('用户更新成功')
        showEditDialog.value = false
        fetchUsers()
      } catch (error) {
        ElMessage.error('更新用户失败：' + (error.response?.data?.message || error.message))
      } finally {
        editLoading.value = false
      }
    }

    // 重置密码
    const resetPassword = (user) => {
      passwordForm.userId = user.id
      passwordForm.newPassword = ''
      passwordForm.confirmPassword = ''
      showPasswordDialog.value = true
    }

    // 处理重置密码
    const handleResetPassword = async () => {
      if (!passwordFormRef.value) return

      try {
        await passwordFormRef.value.validate()
      } catch {
        return
      }

      passwordLoading.value = true
      try {
        await userAPI.updatePassword(passwordForm.userId, passwordForm.newPassword)

        ElMessage.success('密码重置成功')
        showPasswordDialog.value = false
      } catch (error) {
        ElMessage.error('重置密码失败：' + (error.response?.data?.message || error.message))
      } finally {
        passwordLoading.value = false
      }
    }

    // 删除用户
    const deleteUser = async (user) => {
      try {
        await ElMessageBox.confirm(
          `确定要删除用户 "${user.username}" 吗？此操作不可恢复。`,
          '确认删除',
          {
            confirmButtonText: '确定删除',
            cancelButtonText: '取消',
            type: 'warning',
          }
        )

        await userAPI.deleteUser(user.id)
        ElMessage.success('用户删除成功')
        fetchUsers()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('删除用户失败：' + (error.response?.data?.message || error.message))
        }
      }
    }

    // 组件挂载时获取数据
    onMounted(() => {
      fetchUsers()
    })

    onUnmounted(() => {
      if (searchTimer.value) {
        window.clearTimeout(searchTimer.value)
      }
    })

    return {
      // 状态
      loading,
      createLoading,
      editLoading,
      passwordLoading,
      showCreateDialog,
      showEditDialog,
      showPasswordDialog,

      // 数据
      userList,
      pagination,
      searchForm,
      roleOptions,
      statusOptions,
      currentUser,
      superAdminRole: ROLES.SUPER_ADMIN,

      // 表单
      createForm,
      createFormRules,
      editForm,
      editFormRules,
      passwordForm,
      passwordFormRules,

      // 表单引用
      createFormRef,
      editFormRef,
      passwordFormRef,

      // 方法
      getRoleLabel,
      getStatusLabel,
      getRoleTagType,
      getStatusTagType,
      canEditRole,
      canManageUsers: canManageUsersAction,
      fetchUsers,
      handleSearch,
      resetSearch,
      handleSizeChange,
      handleCurrentChange,
      handleCreateUser,
      editUser,
      handleUpdateUser,
      resetPassword,
      handleResetPassword,
      deleteUser,
      formatDateTime,
    }
  },
}
</script>

<style scoped>
.user-management {
  padding: 20px;
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
</style>
