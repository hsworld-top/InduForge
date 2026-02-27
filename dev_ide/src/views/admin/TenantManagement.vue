<template>
  <div class="tenant-management">
    <!-- 页面头部 -->
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-3xl font-bold text-gray-900 dark:text-white">{{ t('tenantManagement.title') }}</h1>
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
        {{ t('tenantManagement.addTenant') }}
      </button>
    </div>

    <!-- 租户列表 -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('tenantManagement.list') }}</h2>
          <div class="flex items-center space-x-4">
            <!-- 状态筛选 -->
            <select
              v-model="filters.status"
              @change="loadTenants"
              class="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded-md text-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
            >
              <option value="">{{ t('tenantManagement.allStatus') }}</option>
              <option value="active">{{ t('tenantManagement.statusActive') }}</option>
              <option value="inactive">{{ t('tenantManagement.statusInactive') }}</option>
              <option value="suspended">{{ t('tenantManagement.statusSuspended') }}</option>
            </select>

            <!-- 搜索框 -->
            <input
              v-model="filters.search"
              @input="debouncedSearch"
              type="text"
              :placeholder="t('tenantManagement.searchPlaceholder')"
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
                {{ t('tenantManagement.tenantInfo') }}
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ t('tenantManagement.contact') }}
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ t('tenantManagement.status') }}
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ t('tenantManagement.createdAt') }}
              </th>
              <th
                class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
              >
                {{ t('tenantManagement.actions') }}
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
                  <div class="w-10 h-10 rounded-lg overflow-hidden border border-gray-200 dark:border-gray-600 mr-3">
                    <img
                      :src="tenant.logoUrl || defaultLogoUrl"
                      :alt="tenant.name"
                      class="w-full h-full object-cover"
                    />
                  </div>
                  <div>
                    <div class="text-sm font-medium text-gray-900 dark:text-white">
                      {{ tenant.name }}
                    </div>
                    <div class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t('tenantManagement.codePrefix') }}: {{ tenant.code }}
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
                      ? t('tenantManagement.statusActive')
                      : tenant.status === 'inactive'
                        ? t('tenantManagement.statusInactive')
                        : t('tenantManagement.statusSuspended')
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
                    {{ t('tenantManagement.edit') }}
                  </button>
                  <button
                    @click="deleteTenant(tenant)"
                    class="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"
                  >
                    {{ t('tenantManagement.delete') }}
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
            {{
              t('tenantManagement.pageSummary', {
                start: (pagination.page - 1) * pagination.limit + 1,
                end: Math.min(pagination.page * pagination.limit, pagination.total),
                total: pagination.total,
              })
            }}
          </div>
          <div class="flex items-center space-x-2">
            <button
              @click="changePage(pagination.page - 1)"
              :disabled="pagination.page <= 1"
              class="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded-md text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              {{ t('tenantManagement.prevPage') }}
            </button>
            <span class="text-sm text-gray-700 dark:text-gray-300">
              {{ t('tenantManagement.pageInfo', { page: pagination.page, totalPages: pagination.totalPages }) }}
            </span>
            <button
              @click="changePage(pagination.page + 1)"
              :disabled="pagination.page >= pagination.totalPages"
              class="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded-md text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              {{ t('tenantManagement.nextPage') }}
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
            {{ showAddDialog ? t('tenantManagement.addDialogTitle') : t('tenantManagement.editDialogTitle') }}
          </h3>

          <form @submit.prevent="saveTenant" class="space-y-4">
            <!-- 基本信息 -->
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {{ t('tenantManagement.tenantName') }} *
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
                  {{ t('tenantManagement.tenantCode') }} *
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
                {{ t('tenantManagement.description') }}
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
                  {{ t('tenantManagement.contactEmail') }}
                </label>
                <input
                  v-model="tenantForm.contactEmail"
                  type="email"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {{ t('tenantManagement.contactPhone') }}
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
                  {{ t('tenantManagement.maxUsers') }}
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
                  {{ t('tenantManagement.maxProjects') }}
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
                {{ t('tenantManagement.status') }}
              </label>
              <select
                v-model="tenantForm.status"
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              >
                <option value="active">{{ t('tenantManagement.statusActive') }}</option>
                <option value="inactive">{{ t('tenantManagement.statusInactive') }}</option>
                <option value="suspended">{{ t('tenantManagement.statusSuspended') }}</option>
              </select>
            </div>

            <!-- 公司信息 -->
            <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
              <h4 class="text-md font-medium text-gray-900 dark:text-white mb-3">{{ t('tenantManagement.companyInfo') }}</h4>

              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    {{ t('tenantManagement.companyName') }}
                  </label>
                  <input
                    v-model="tenantForm.companyName"
                    type="text"
                    class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  />
                </div>
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    {{ t('tenantManagement.companyPhone') }}
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
                  {{ t('tenantManagement.companyAddress') }}
                </label>
                <textarea
                  v-model="tenantForm.companyAddress"
                  rows="2"
                  class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                ></textarea>
              </div>

              <div class="mt-4">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {{ t('tenantManagement.companyWebsite') }}
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
              <h4 class="text-md font-medium text-gray-900 dark:text-white mb-3">{{ t('tenantManagement.brandAssets') }}</h4>

              <!-- Logo上传 -->
              <div class="mb-4">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {{ t('tenantManagement.logo') }}
                </label>
                <div class="flex items-center space-x-4">
                  <div class="w-16 h-16 rounded-lg overflow-hidden border border-gray-300 dark:border-gray-600">
                    <img :src="logoPreviewUrl" alt="Logo" class="w-full h-full object-cover" />
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
                      {{ tenantForm.logoUrl ? t('tenantManagement.replaceLogo') : t('tenantManagement.uploadLogo') }}
                    </button>
                  </div>
                </div>
              </div>

              <!-- 背景图上传 -->
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {{ t('tenantManagement.loginBackground') }}
                </label>
                <div class="flex items-center space-x-4">
                  <div class="w-16 h-16 rounded-lg overflow-hidden border border-gray-300 dark:border-gray-600">
                    <img
                      :src="backgroundPreviewUrl"
                      :alt="t('tenantManagement.backgroundImageAlt')"
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
                      {{ tenantForm.loginBackgroundUrl ? t('tenantManagement.replaceBackground') : t('tenantManagement.uploadBackground') }}
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
                {{ t('tenantManagement.cancel') }}
              </button>
              <button
                type="submit"
                :disabled="saving"
                class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {{ saving ? t('tenantManagement.saving') : t('tenantManagement.save') }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, reactive, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import dayjs from 'dayjs'
import { tenantAPI } from '@/api'
import { TIME_FORMAT } from '@/constants'
import defaultLogoUrl from '@/assets/images/default-logo.svg'
import defaultLoginBgUrl from '@/assets/images/default-login-bg.svg'

export default {
  name: 'TenantManagement',
  setup() {
    const { t } = useI18n()
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
    const logoPreviewUrl = computed(() => tenantForm.logoUrl || defaultLogoUrl)
    const backgroundPreviewUrl = computed(
      () => tenantForm.loginBackgroundUrl || defaultLoginBgUrl
    )

    // 防抖搜索
    let searchTimeout = null
    const debouncedSearch = () => {
      window.clearTimeout(searchTimeout)
      searchTimeout = window.setTimeout(() => {
        loadTenants()
      }, 500)
    }

    // 加载租户列表
    const loadTenants = async () => {
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
        ElMessage.error(t('tenantManagement.loadFailed'))
      } finally {
        loading.value = false
      }
    }

    // 格式化日期
    const formatDate = (date) => {
      const parsed = dayjs(date)
      return parsed.isValid() ? parsed.format(TIME_FORMAT) : ''
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
          t('tenantManagement.deleteConfirmText', { name: tenant.name }),
          t('tenantManagement.deleteConfirmTitle'),
          {
            confirmButtonText: t('tenantManagement.deleteConfirmButton'),
            cancelButtonText: t('tenantManagement.cancel'),
            type: 'warning',
          }
        )

        await tenantAPI.deleteTenant(tenant.id)
        ElMessage.success(t('tenantManagement.deleteSuccess'))
        loadTenants()
      } catch (error) {
        if (error !== 'cancel') {
          console.error('删除租户失败:', error)
          ElMessage.error(t('tenantManagement.deleteFailed'))
        }
      }
    }

    // 保存租户
    const saveTenant = async () => {
      saving.value = true
      try {
        const payload = {
          ...tenantForm,
          logoUrl: tenantForm.logoUrl || '',
          loginBackgroundUrl: tenantForm.loginBackgroundUrl || '',
        }

        if (showAddDialog.value) {
          await tenantAPI.createTenant(payload)
          ElMessage.success(t('tenantManagement.createSuccess'))
        } else {
          await tenantAPI.updateTenant(tenantForm.code, payload)
          ElMessage.success(t('tenantManagement.updateSuccess'))
        }

        closeDialog()
        loadTenants()
      } catch (error) {
        console.error('保存租户失败:', error)
        ElMessage.error(error.response?.data?.message || t('tenantManagement.saveFailed'))
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
      if (logoInput.value) logoInput.value.value = ''
      if (backgroundInput.value) backgroundInput.value.value = ''
    }

    /**
     * 将图片文件转换为 base64 DataURL。
     * @param {File} file - 图片文件
     * @returns {Promise<string>} base64 数据
     */
    const fileToDataUrl = (file) =>
      new Promise((resolve, reject) => {
        const reader = new window.FileReader()
        reader.onload = () => resolve(reader.result)
        reader.onerror = () => reject(new Error('文件读取失败'))
        reader.readAsDataURL(file)
      })

    // 处理Logo上传
    const handleLogoUpload = async (event) => {
      const file = event.target.files[0]
      if (!file) return

      try {
        // 新增租户时尚未有租户实体，使用本地base64并在创建接口中一并提交
        if (showAddDialog.value) {
          tenantForm.logoUrl = await fileToDataUrl(file)
          ElMessage.success(t('tenantManagement.logoUploadSuccess'))
          return
        }

        const formData = new window.FormData()
        formData.append('file', file)

        const response = await tenantAPI.uploadFile(
          tenantForm.code,
          'logo',
          formData
        )

        tenantForm.logoUrl = response.data.fileUrl
        ElMessage.success(t('tenantManagement.logoUploadSuccess'))
      } catch (error) {
        console.error('Logo上传失败:', error)
        ElMessage.error(error.response?.data?.message || t('tenantManagement.logoUploadFailed'))
      } finally {
        event.target.value = ''
      }
    }

    // 处理背景图上传
    const handleBackgroundUpload = async (event) => {
      const file = event.target.files[0]
      if (!file) return

      try {
        // 新增租户时尚未有租户实体，使用本地base64并在创建接口中一并提交
        if (showAddDialog.value) {
          tenantForm.loginBackgroundUrl = await fileToDataUrl(file)
          ElMessage.success(t('tenantManagement.backgroundUploadSuccess'))
          return
        }

        const formData = new window.FormData()
        formData.append('file', file)

        const response = await tenantAPI.uploadFile(
          tenantForm.code,
          'background',
          formData
        )

        tenantForm.loginBackgroundUrl = response.data.fileUrl
        ElMessage.success(t('tenantManagement.backgroundUploadSuccess'))
      } catch (error) {
        console.error('背景图上传失败:', error)
        ElMessage.error(error.response?.data?.message || t('tenantManagement.backgroundUploadFailed'))
      } finally {
        event.target.value = ''
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
      defaultLogoUrl,
      logoPreviewUrl,
      backgroundPreviewUrl,
      t,
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
