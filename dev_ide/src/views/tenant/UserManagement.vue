<template>
  <div class="user-management">
    <!-- 页面标题和操作栏 -->
    <div class="flex justify-between items-center mb-3">
      <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('userManagement.title') }}</h1>
      <el-button v-if="canManageUsers" type="primary" @click="showCreateDialog = true" size="small"
        class="bg-blue-600 hover:bg-blue-700">
        <el-icon class="mr-2">
          <Plus />
        </el-icon>
        {{ t('userManagement.addUser') }}
      </el-button>
    </div>

    <!-- 搜索和筛选栏 -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6 mb-6">
      <el-form :inline="true" :model="searchForm" class="flex flex-wrap gap-4">
        <el-form-item :label="t('userManagement.username')">
          <el-input v-model="searchForm.username" :placeholder="t('userManagement.searchUsername')" clearable
            style="width: 200px" @input="handleSearch" />
        </el-form-item>
        <el-form-item :label="t('userManagement.role')">
          <el-select v-model="searchForm.role" :placeholder="t('userManagement.selectRole')" clearable
            style="width: 150px" @change="handleSearch">
            <el-option v-for="role in roleOptions" :key="role.value" :label="role.label" :value="role.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('userManagement.status')">
          <el-select v-model="searchForm.status" :placeholder="t('userManagement.selectStatus')" clearable
            style="width: 150px" @change="handleSearch">
            <el-option v-for="status in statusOptions" :key="status.value" :label="status.label"
              :value="status.value" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button @click="resetSearch" type="default">
            <el-icon>
              <Refresh />
            </el-icon>
            {{ t('userManagement.reset') }}
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 用户列表 -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
      <el-table :data="userList" v-loading="loading" style="width: 100%"
        :header-cell-style="{ background: '#f9fafb', color: '#374151' }">
        <el-table-column prop="username" :label="t('userManagement.username')" width="120" />
        <el-table-column prop="fullName" :label="t('userManagement.fullName')" width="120" />
        <el-table-column prop="email" :label="t('userManagement.email')" width="300" />
        <el-table-column prop="role" :label="t('userManagement.role')" min-width="200">
          <template #default="scope">
            <el-tag :type="getRoleTagType(scope.row.role)">
              {{ getRoleLabel(scope.row.role) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" :label="t('userManagement.status')" width="100">
          <template #default="scope">
            <el-tag :type="getStatusTagType(scope.row.status)">
              {{ getStatusLabel(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('userManagement.tenant')" width="140">
          <template #default="scope">
            {{ getTenantLabel(scope.row.tenant) }}
          </template>
        </el-table-column>
        <el-table-column prop="lastLoginAt" :label="t('userManagement.lastLogin')" width="200">
          <template #default="scope">
            {{ formatDateTime(scope.row.lastLoginAt) }}
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" :label="t('userManagement.createdAt')" width="200">
          <template #default="scope">
            {{ formatDateTime(scope.row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('userManagement.actions')" width="300" fixed="right" v-if="canManageUsers">
          <template #default="scope">
            <div class="user-actions">
              <el-button type="primary" size="small" @click="editUser(scope.row)">
                {{ t('userManagement.edit') }}
              </el-button>
              <el-button type="warning" size="small" @click="resetPassword(scope.row)">
                {{ t('userManagement.resetPassword') }}
              </el-button>
              <el-button type="danger" size="small" @click="deleteUser(scope.row)"
                v-if="scope.row.id !== currentUser?.id && scope.row.role !== superAdminRole">
                {{ t('userManagement.delete') }}
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-bar flex justify-between items-center py-2 px-3 border-t border-gray-200 dark:border-gray-700">
        <div class="text-xs text-gray-500 dark:text-gray-400">
          {{
            t('userManagement.totalRange', {
              start: (pagination.page - 1) * pagination.limit + 1,
              end: Math.min(pagination.page * pagination.limit, pagination.total),
              total: pagination.total,
            })
          }}
        </div>
        <el-pagination size="small" v-model:current-page="pagination.page" v-model:page-size="pagination.limit"
          :page-sizes="[10, 20, 50, 100]" :total="pagination.total" layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange" @current-change="handleCurrentChange" />
      </div>
    </div>

    <!-- 创建用户对话框 -->
    <el-dialog v-model="showCreateDialog" :title="t('userManagement.createUser')" width="500px"
      :close-on-click-modal="false">
      <el-form ref="createFormRef" :model="createForm" :rules="createFormRules" label-width="100px">
        <el-form-item :label="t('userManagement.username')" prop="username">
          <el-input v-model="createForm.username" :placeholder="t('userManagement.inputUsername')" />
        </el-form-item>
        <el-form-item :label="t('auth.password')" prop="password">
          <el-input v-model="createForm.password" type="password" :placeholder="t('userManagement.inputPassword')"
            show-password />
        </el-form-item>
        <el-form-item :label="t('profile.confirmPassword')" prop="confirmPassword">
          <el-input v-model="createForm.confirmPassword" type="password"
            :placeholder="t('userManagement.inputConfirmPassword')" show-password />
        </el-form-item>
        <el-form-item :label="t('userManagement.email')" prop="email">
          <el-input v-model="createForm.email" :placeholder="t('userManagement.inputEmail')" />
        </el-form-item>
        <el-form-item :label="t('userManagement.fullName')" prop="fullName">
          <el-input v-model="createForm.fullName" :placeholder="t('userManagement.inputFullName')" />
        </el-form-item>
        <el-form-item :label="t('userManagement.role')" prop="role">
          <el-select v-model="createForm.role" :placeholder="t('userManagement.chooseRole')" style="width: 100%">
            <el-option v-for="role in roleOptions" :key="role.value" :label="role.label" :value="role.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">{{ t('userManagement.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreateUser" :loading="createLoading">
          {{ t('userManagement.create') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 编辑用户对话框 -->
    <el-dialog v-model="showEditDialog" :title="t('userManagement.editUser')" width="500px"
      :close-on-click-modal="false">
      <el-form ref="editFormRef" :model="editForm" :rules="editFormRules" label-width="100px">
        <el-form-item :label="t('userManagement.username')">
          <el-input v-model="editForm.username" disabled />
        </el-form-item>
        <el-form-item :label="t('userManagement.email')" prop="email">
          <el-input v-model="editForm.email" :placeholder="t('userManagement.inputEmail')" />
        </el-form-item>
        <el-form-item :label="t('userManagement.fullName')" prop="fullName">
          <el-input v-model="editForm.fullName" :placeholder="t('userManagement.inputFullName')" />
        </el-form-item>
        <el-form-item :label="t('userManagement.role')" prop="role">
          <el-select v-model="editForm.role" :placeholder="t('userManagement.chooseRole')" style="width: 100%"
            :disabled="!canEditRole">
            <el-option v-for="role in roleOptions" :key="role.value" :label="role.label" :value="role.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('userManagement.status')" prop="status">
          <el-select v-model="editForm.status" :placeholder="t('userManagement.chooseStatus')" style="width: 100%">
            <el-option v-for="status in statusOptions" :key="status.value" :label="status.label"
              :value="status.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">{{ t('userManagement.cancel') }}</el-button>
        <el-button type="primary" @click="handleUpdateUser" :loading="editLoading">
          {{ t('userManagement.save') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog v-model="showPasswordDialog" :title="t('userManagement.resetPassword')" width="400px"
      :close-on-click-modal="false">
      <el-form ref="passwordFormRef" :model="passwordForm" :rules="passwordFormRules" label-width="100px">
        <el-form-item :label="t('profile.newPassword')" prop="newPassword">
          <el-input v-model="passwordForm.newPassword" type="password"
            :placeholder="t('userManagement.inputNewPassword')" show-password />
        </el-form-item>
        <el-form-item :label="t('profile.confirmPassword')" prop="confirmPassword">
          <el-input v-model="passwordForm.confirmPassword" type="password"
            :placeholder="t('userManagement.inputConfirmPassword')" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPasswordDialog = false">{{ t('userManagement.cancel') }}</el-button>
        <el-button type="primary" @click="handleResetPassword" :loading="passwordLoading">
          {{ t('userManagement.resetAction') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script lang="ts">
// @ts-nocheck
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/store'
import { userAPI } from '@/api/user.api'
import { RoleEnum, UserStatusEnum } from '@/enums'
import { formatDateTime } from '@/utils/date'
import { canManageUsers } from '@/permissions'
import { ROLES } from '@/constants'

export default {
  name: 'TenantUserManagement',
  setup() {
    const { t } = useI18n()
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
      { value: RoleEnum.SYSTEM_ADMIN, label: t('userManagement.roleSystemAdmin') },
      { value: RoleEnum.PROJECT_ADMIN, label: t('userManagement.roleProjectAdmin') },
      { value: RoleEnum.OPS_ADMIN, label: t('userManagement.roleOpsAdmin') },
      { value: RoleEnum.USER_ADMIN, label: t('userManagement.roleUserAdmin') },
    ]

    // 状态选项
    const statusOptions = [
      { value: UserStatusEnum.ACTIVE, label: t('userManagement.statusActive') },
      { value: UserStatusEnum.INACTIVE, label: t('userManagement.statusInactive') },
      { value: UserStatusEnum.SUSPENDED, label: t('userManagement.statusSuspended') },
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
        { required: true, message: t('userManagement.inputUsername'), trigger: 'blur' },
        { min: 3, max: 50, message: t('userManagement.usernameLength'), trigger: 'blur' },
      ],
      password: [
        { required: true, message: t('userManagement.inputPassword'), trigger: 'blur' },
        { min: 6, message: t('userManagement.passwordMinLength'), trigger: 'blur' },
      ],
      confirmPassword: [
        { required: true, message: t('userManagement.inputConfirmPassword'), trigger: 'blur' },
        {
          validator: (rule, value, callback) => {
            if (value !== createForm.password) {
              callback(new Error(t('userManagement.passwordMismatch')))
            } else {
              callback()
            }
          },
          trigger: 'blur',
        },
      ],
      email: [{ type: 'email', message: t('userManagement.invalidEmail'), trigger: 'blur' }],
      fullName: [{ required: true, message: t('userManagement.inputFullName'), trigger: 'blur' }],
      role: [{ required: true, message: t('userManagement.chooseRole'), trigger: 'change' }],
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
      email: [{ type: 'email', message: t('userManagement.invalidEmail'), trigger: 'blur' }],
      fullName: [{ required: true, message: t('userManagement.inputFullName'), trigger: 'blur' }],
      role: [{ required: true, message: t('userManagement.chooseRole'), trigger: 'change' }],
      status: [{ required: true, message: t('userManagement.chooseStatus'), trigger: 'change' }],
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
        { required: true, message: t('userManagement.inputNewPassword'), trigger: 'blur' },
        { min: 6, message: t('userManagement.passwordMinLength'), trigger: 'blur' },
      ],
      confirmPassword: [
        { required: true, message: t('userManagement.inputConfirmPassword'), trigger: 'blur' },
        {
          validator: (rule, value, callback) => {
            if (value !== passwordForm.newPassword) {
              callback(new Error(t('userManagement.passwordMismatch')))
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
      const map = {
        [RoleEnum.SYSTEM_ADMIN]: t('userManagement.roleSystemAdmin'),
        [RoleEnum.PROJECT_ADMIN]: t('userManagement.roleProjectAdmin'),
        [RoleEnum.OPS_ADMIN]: t('userManagement.roleOpsAdmin'),
        [RoleEnum.USER_ADMIN]: t('userManagement.roleUserAdmin'),
      }
      return map[role] || role
    }

    // 获取状态标签
    const getStatusLabel = (status) => {
      const map = {
        [UserStatusEnum.ACTIVE]: t('userManagement.statusActive'),
        [UserStatusEnum.INACTIVE]: t('userManagement.statusInactive'),
        [UserStatusEnum.SUSPENDED]: t('userManagement.statusSuspended'),
      }
      return map[status] || status
    }

    // 获取租户展示名称
    const getTenantLabel = (tenant) => {
      const tenantName = tenant?.name?.trim()
      if (!tenantName) return '-'
      if (tenantName === '默认租户' || tenantName === 'Default Tenant') {
        return t('profile.defaultTenant')
      }
      return tenantName
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
        ElMessage.error(
          t('userManagement.fetchUsersFailed', {
            message: error.response?.data?.message || error.message,
          })
        )
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

        ElMessage.success(t('userManagement.createUserSuccess'))
        showCreateDialog.value = false
        resetCreateForm()
        fetchUsers()
      } catch (error) {
        ElMessage.error(
          t('userManagement.createUserFailed', {
            message: error.response?.data?.message || error.message,
          })
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

        ElMessage.success(t('userManagement.updateUserSuccess'))
        showEditDialog.value = false
        fetchUsers()
      } catch (error) {
        ElMessage.error(
          t('userManagement.updateUserFailed', {
            message: error.response?.data?.message || error.message,
          })
        )
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

        ElMessage.success(t('userManagement.resetPasswordSuccess'))
        showPasswordDialog.value = false
      } catch (error) {
        ElMessage.error(
          t('userManagement.resetPasswordFailed', {
            message: error.response?.data?.message || error.message,
          })
        )
      } finally {
        passwordLoading.value = false
      }
    }

    // 删除用户
    const deleteUser = async (user) => {
      try {
        await ElMessageBox.confirm(
          t('userManagement.deleteConfirmText', { username: user.username }),
          t('userManagement.deleteConfirmTitle'),
          {
            confirmButtonText: t('userManagement.deleteConfirmButton'),
            cancelButtonText: t('userManagement.cancel'),
            type: 'warning',
          }
        )

        await userAPI.deleteUser(user.id)
        ElMessage.success(t('userManagement.deleteUserSuccess'))
        fetchUsers()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(
            t('userManagement.deleteUserFailed', {
              message: error.response?.data?.message || error.message,
            })
          )
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
      t,
      getTenantLabel,
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

.user-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
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
</style>



