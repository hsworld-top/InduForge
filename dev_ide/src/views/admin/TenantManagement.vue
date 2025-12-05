<template>
  <div class="tenant-management">
    <!-- 页面头部 -->
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-3xl font-bold text-gray-900 dark:text-white">租户管理</h1>
      <button
        @click="showAddDialog = true"
        class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
      >
        <svg class="w-5 h-5 mr-2 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 6v6m0 0v6m0-6h6m-6 0H6"
          />
        </svg>
        添加租户
      </button>
    </div>

    <!-- 租户列表 -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">租户列表</h2>
          <div class="flex items-center space-x-4">
            <!-- 状态筛选 -->
            <select
              v-model="filters.status"
              @change="loadTenants"
              class="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
            >
              <option value="">全部状态</option>
              <option value="active">活跃</option>
              <option value="inactive">未激活</option>
              <option value="suspended">暂停</option>
            </select>

            <!-- 搜索框 -->
            <input
              v-model="filters.search"
              @input="debouncedSearch"
              type="text"
              placeholder="搜索租户名称或代码..."
              class="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
            />
          </div>
        </div>
      </div>

      <!-- 表格 -->
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                租户信息
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                联系方式
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                状态
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                创建时间
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                操作
              </th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
            <tr
              v-for="tenant in tenants"
              :key="tenant.id"
              class="hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="flex items-center">
                  <div v-if="tenant.logoUrl" class="w-10 h-10 rounded-lg overflow-hidden mr-3">
                    <img
                      :src="tenant.logoUrl"
                      :alt="tenant.name"
                      class="w-full h-full object-cover"
                    />
                  </div>
                  <div
                    v-else
                    class="w-10 h-10 bg-gray-200 dark:bg-gray-600 rounded-lg flex items-center justify-center mr-3"
                  >
                    <span class="text-gray-500 dark:text-gray-400 text-sm font-medium">{{
                      tenant.name.charAt(0)
                    }}</span>
                  </div>
                  <div>
                    <div class="text-sm font-medium text-gray-900 dark:text-white">
                      {{ tenant.name }}
                    </div>
                    <div class="text-sm text-gray-500 dark:text-gray-400">
                      代码: {{ tenant.code }}
                    </div>
                  </div>
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900 dark:text-white">
                  {{ tenant.contactEmail || '-' }}
                </div>
                <div class="text-sm text-gray-500 dark:text-gray-400">
                  {{ tenant.contactPhone || '-' }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span
                  :class="[
                    'inline-flex px-2 py-1 text-xs font-semibold rounded-full',
                    tenant.status === 'active'
                      ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                      : tenant.status === 'inactive'
                        ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
                        : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
                  ]"
                >
                  {{
                    tenant.status === 'active'
                      ? '活跃'
                      : tenant.status === 'inactive'
                        ? '未激活'
                        : '暂停'
                  }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                {{ formatDate(tenant.createdAt) }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                <div class="flex items-center space-x-2">
                  <button
                    @click="editTenant(tenant)"
                    class="text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-300"
                  >
                    编辑
                  </button>
                  <button
                    @click="deleteTenant(tenant)"
                    class="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"
                  >
                    删除
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 分页 -->
      <div class="px-6 py-4 border-t border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <div class="text-sm text-gray-700 dark:text-gray-300">
            显示 {{ (pagination.page - 1) * pagination.limit + 1 }} -
            {{ Math.min(pagination.page * pagination.limit, pagination.total) }} 条， 共
            {{ pagination.total }} 条
          </div>
          <div class="flex items-center space-x-2">
            <button
              @click="changePage(pagination.page - 1)"
              :disabled="pagination.page <= 1"
              class="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded-md text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              上一页
            </button>
            <span class="text-sm text-gray-700 dark:text-gray-300">
              第 {{ pagination.page }} 页，共 {{ pagination.totalPages }} 页
            </span>
            <button
              @click="changePage(pagination.page + 1)"
              :disabled="pagination.page >= pagination.totalPages"
              class="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded-md text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              下一页
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加/编辑租户对话框 -->
    <div
      v-if="showAddDialog || showEditDialog"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      @click.self="closeDialog"
    >
      <div
        class="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto"
      >
        <div class="p-6">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
            {{ showAddDialog ? '添加租户' : '编辑租户' }}
          </h3>

          <form @submit.prevent="saveTenant" class="space-y-4">
            <!-- 基本信息 -->
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  租户名称 *
                </label>
                <input
                  v-model="tenantForm.name"
                  type="text"
                  required
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  租户代码 *
                </label>
                <input
                  v-model="tenantForm.code"
                  type="text"
                  required
                  :disabled="showEditDialog"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white disabled:bg-gray-100 dark:disabled:bg-gray-600"
                />
              </div>
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                描述
              </label>
              <textarea
                v-model="tenantForm.description"
                rows="3"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              ></textarea>
            </div>

            <!-- 联系信息 -->
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  联系邮箱
                </label>
                <input
                  v-model="tenantForm.contactEmail"
                  type="email"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  联系电话
                </label>
                <input
                  v-model="tenantForm.contactPhone"
                  type="text"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
              </div>
            </div>

            <!-- 资源限制 -->
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  最大用户数
                </label>
                <input
                  v-model.number="tenantForm.maxUsers"
                  type="number"
                  min="1"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  最大工程数
                </label>
                <input
                  v-model.number="tenantForm.maxProjects"
                  type="number"
                  min="1"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
              </div>
            </div>

            <!-- 状态 -->
            <div v-if="showEditDialog">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                状态
              </label>
              <select
                v-model="tenantForm.status"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              >
                <option value="active">活跃</option>
                <option value="inactive">未激活</option>
                <option value="suspended">暂停</option>
              </select>
            </div>

            <!-- 公司信息 -->
            <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
              <h4 class="text-md font-medium text-gray-900 dark:text-white mb-3">公司信息</h4>

              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    公司名称
                  </label>
                  <input
                    v-model="tenantForm.companyName"
                    type="text"
                    class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  />
                </div>
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    公司电话
                  </label>
                  <input
                    v-model="tenantForm.companyPhone"
                    type="text"
                    class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  />
                </div>
              </div>

              <div class="mt-4">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  公司地址
                </label>
                <textarea
                  v-model="tenantForm.companyAddress"
                  rows="2"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                ></textarea>
              </div>

              <div class="mt-4">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  公司网站
                </label>
                <input
                  v-model="tenantForm.companyWebsite"
                  type="url"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
              </div>
            </div>

            <!-- 文件上传 -->
            <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
              <h4 class="text-md font-medium text-gray-900 dark:text-white mb-3">品牌资产</h4>

              <!-- Logo上传 -->
              <div class="mb-4">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Logo
                </label>
                <div class="flex items-center space-x-4">
                  <div
                    v-if="tenantForm.logoUrl"
                    class="w-16 h-16 rounded-lg overflow-hidden border border-gray-300 dark:border-gray-600"
                  >
                    <img :src="tenantForm.logoUrl" alt="Logo" class="w-full h-full object-cover" />
                  </div>
                  <div class="flex-1">
                    <input
                      ref="logoInput"
                      type="file"
                      accept="image/*"
                      @change="handleLogoUpload"
                      class="hidden"
                    />
                    <button
                      type="button"
                      @click="$refs.logoInput.click()"
                      class="px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-md hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
                    >
                      {{ tenantForm.logoUrl ? '更换Logo' : '上传Logo' }}
                    </button>
                  </div>
                </div>
              </div>

              <!-- 背景图上传 -->
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  登录页面背景图
                </label>
                <div class="flex items-center space-x-4">
                  <div
                    v-if="tenantForm.loginBackgroundUrl"
                    class="w-16 h-16 rounded-lg overflow-hidden border border-gray-300 dark:border-gray-600"
                  >
                    <img
                      :src="tenantForm.loginBackgroundUrl"
                      alt="背景图"
                      class="w-full h-full object-cover"
                    />
                  </div>
                  <div class="flex-1">
                    <input
                      ref="backgroundInput"
                      type="file"
                      accept="image/*"
                      @change="handleBackgroundUpload"
                      class="hidden"
                    />
                    <button
                      type="button"
                      @click="$refs.backgroundInput.click()"
                      class="px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-md hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
                    >
                      {{ tenantForm.loginBackgroundUrl ? '更换背景图' : '上传背景图' }}
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <!-- 操作按钮 -->
            <div
              class="flex justify-end space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700"
            >
              <button
                type="button"
                @click="closeDialog"
                class="px-4 py-2 text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
              >
                取消
              </button>
              <button
                type="submit"
                :disabled="saving"
                class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {{ saving ? '保存中...' : '保存' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { tenantAPI } from '@/api'

export default {
  name: 'TenantManagement',
  setup() {
    const tenants = ref([])
    const loading = ref(false)
    const saving = ref(false)

    // 对话框状态
    const showAddDialog = ref(false)
    const showEditDialog = ref(false)

    // 分页信息
    const pagination = reactive({
      page: 1,
      limit: 10,
      total: 0,
      totalPages: 0,
    })

    // 筛选条件
    const filters = reactive({
      status: '',
      search: '',
    })

    // 租户表单
    const tenantForm = reactive({
      name: '',
      code: '',
      description: '',
      contactEmail: '',
      contactPhone: '',
      maxUsers: 100,
      maxProjects: 50,
      status: 'active',
      logoUrl: '',
      loginBackgroundUrl: '',
      companyName: '',
      companyAddress: '',
      companyPhone: '',
      companyWebsite: '',
    })

    // 文件输入引用
    const logoInput = ref(null)
    const backgroundInput = ref(null)

    // 防抖搜索
    let searchTimeout = null
    const debouncedSearch = () => {
      clearTimeout(searchTimeout)
      searchTimeout = setTimeout(() => {
        loadTenants()
      }, 500)
    }

    // 加载租户列表
    const loadTenants = async () => {
      debugger
      loading.value = true
      try {
        const params = {
          page: pagination.page,
          limit: Number(pagination.limit),
          status: filters.status || undefined,
        }

        // 注意：这里需要扩展API以支持搜索功能
        // 暂时只使用状态筛选
        const response = await tenantAPI.getTenants(params)

        tenants.value = response.data.tenants
        pagination.total = response.pagination.total
        pagination.totalPages = response.pagination.totalPages
      } catch (error) {
        console.error('加载租户列表失败:', error)
        ElMessage.error('加载租户列表失败')
      } finally {
        loading.value = false
      }
    }

    // 格式化日期
    const formatDate = (date) => {
      return new Date(date).toLocaleDateString('zh-CN')
    }

    // 换页
    const changePage = (page) => {
      if (page >= 1 && page <= pagination.totalPages) {
        pagination.page = page
        loadTenants()
      }
    }

    // 编辑租户
    const editTenant = (tenant) => {
      Object.assign(tenantForm, {
        name: tenant.name,
        code: tenant.code,
        description: tenant.description || '',
        contactEmail: tenant.contactEmail || '',
        contactPhone: tenant.contactPhone || '',
        maxUsers: tenant.maxUsers,
        maxProjects: tenant.maxProjects,
        status: tenant.status,
        logoUrl: tenant.logoUrl || '',
        loginBackgroundUrl: tenant.loginBackgroundUrl || '',
        companyName: tenant.companyName || '',
        companyAddress: tenant.companyAddress || '',
        companyPhone: tenant.companyPhone || '',
        companyWebsite: tenant.companyWebsite || '',
      })
      showEditDialog.value = true
    }

    // 删除租户
    const deleteTenant = async (tenant) => {
      try {
        await ElMessageBox.confirm(
          `确定要删除租户 "${tenant.name}" 吗？此操作不可恢复。`,
          '确认删除',
          {
            confirmButtonText: '确定删除',
            cancelButtonText: '取消',
            type: 'warning',
          }
        )

        await tenantAPI.deleteTenant(tenant.id)
        ElMessage.success('租户删除成功')
        loadTenants()
      } catch (error) {
        if (error !== 'cancel') {
          console.error('删除租户失败:', error)
          ElMessage.error('删除租户失败')
        }
      }
    }

    // 保存租户
    const saveTenant = async () => {
      saving.value = true
      try {
        if (showAddDialog.value) {
          await tenantAPI.createTenant(tenantForm)
          ElMessage.success('租户创建成功')
        } else {
          await tenantAPI.updateTenant(tenantForm.code, tenantForm)
          ElMessage.success('租户更新成功')
        }

        closeDialog()
        loadTenants()
      } catch (error) {
        console.error('保存租户失败:', error)
        ElMessage.error(error.response?.data?.message || '保存租户失败')
      } finally {
        saving.value = false
      }
    }

    // 关闭对话框
    const closeDialog = () => {
      showAddDialog.value = false
      showEditDialog.value = false

      // 重置表单
      Object.assign(tenantForm, {
        name: '',
        code: '',
        description: '',
        contactEmail: '',
        contactPhone: '',
        maxUsers: 100,
        maxProjects: 50,
        status: 'active',
        logoUrl: '',
        loginBackgroundUrl: '',
        companyName: '',
        companyAddress: '',
        companyPhone: '',
        companyWebsite: '',
      })
    }

    // 处理Logo上传
    const handleLogoUpload = async (event) => {
      const file = event.target.files[0]
      if (!file) return

      try {
        const formData = new FormData()
        formData.append('file', file)

        const response = await tenantAPI.uploadFile(
          showEditDialog.value ? tenantForm.code : 'temp',
          'logo',
          formData
        )

        tenantForm.logoUrl = response.data.fileUrl
        ElMessage.success('Logo上传成功')
      } catch (error) {
        console.error('Logo上传失败:', error)
        ElMessage.error('Logo上传失败')
      }
    }

    // 处理背景图上传
    const handleBackgroundUpload = async (event) => {
      const file = event.target.files[0]
      if (!file) return

      try {
        const formData = new FormData()
        formData.append('file', file)

        const response = await tenantAPI.uploadFile(
          showEditDialog.value ? tenantForm.code : 'temp',
          'background',
          formData
        )

        tenantForm.loginBackgroundUrl = response.data.fileUrl
        ElMessage.success('背景图上传成功')
      } catch (error) {
        console.error('背景图上传失败:', error)
        ElMessage.error('背景图上传失败')
      }
    }

    onMounted(() => {
      loadTenants()
    })

    return {
      tenants,
      loading,
      saving,
      pagination,
      filters,
      tenantForm,
      showAddDialog,
      showEditDialog,
      logoInput,
      backgroundInput,
      loadTenants,
      formatDate,
      changePage,
      editTenant,
      deleteTenant,
      saveTenant,
      closeDialog,
      handleLogoUpload,
      handleBackgroundUpload,
      debouncedSearch,
    }
  },
}
</script>

<style scoped>
.tenant-management {
  padding: 24px;
}
</style>
