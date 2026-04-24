<template>
  <div>
    <!-- 遮罩 -->
    <transition name="fade">
      <div v-if="visible" class="tm-mask backdrop-blur-md" @click.self="closeDrawer"></div>
    </transition>

    <!-- 抽屉 -->
    <transition name="slide">
      <aside v-if="visible" class="tm-drawer bg-white dark:bg-slate-900 shadow-2xl">
        <header
          class="tm-drawer-head px-8 py-6 border-b border-slate-100 dark:border-slate-800/60 flex justify-between items-start bg-slate-50/50 dark:bg-slate-900/50 backdrop-blur-sm sticky top-0 z-20"
        >
          <div>
            <h2 class="text-xl font-extrabold text-slate-900 dark:text-white tracking-tight">
              {{
                isAdd ? t('tenantManagement.addDialogTitle') : t('tenantManagement.editDialogTitle')
              }}
            </h2>
            <p
              class="text-sm font-medium text-slate-500 dark:text-slate-400 mt-1.5 flex items-center gap-2"
            >
              <span>{{ t('tenantManagement.tenantCode') }}</span>
              <span class="w-1 h-1 rounded-full bg-slate-300 dark:bg-slate-600"></span>
              <span>{{ t('tenantManagement.companyInfo') }}</span>
              <span class="w-1 h-1 rounded-full bg-slate-300 dark:bg-slate-600"></span>
              <span>{{ t('tenantManagement.brandAssets') }}</span>
            </p>
          </div>
          <button
            class="text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-800 dark:hover:text-slate-300 transition-all rounded-xl p-2.5 -mr-2"
            type="button"
            @click="closeDrawer"
          >
            <svg
              width="20"
              height="20"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2.5"
            >
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </header>

        <nav
          class="flex border-b border-slate-100 dark:border-slate-800/60 px-8 pt-4 gap-8 bg-white dark:bg-slate-900 sticky top-[93px] z-10"
        >
          <button
            type="button"
            v-for="tab in tabs"
            :key="tab.k"
            :class="[
              'pb-4 text-sm font-bold transition-all duration-200 ease-in-out relative',
              curTab === tab.k
                ? 'text-blue-600 dark:text-blue-500'
                : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300',
            ]"
            @click="curTab = tab.k"
          >
            {{ tab.l }}
            <span
              v-if="curTab === tab.k"
              class="absolute bottom-0 left-0 w-full h-0.5 bg-blue-600 dark:bg-blue-500 rounded-t-full shadow-[0_-2px_8px_rgba(37,99,235,0.4)]"
            ></span>
          </button>
        </nav>

        <div class="flex-1 overflow-y-auto p-8 custom-scrollbar">
          <!-- ====== 基本信息 ====== -->
          <section v-show="curTab === 'basic'" class="space-y-6">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                  {{ t('tenantManagement.tenantName') }}
                  <span class="text-rose-500 ml-1">*</span>
                </label>
                <input
                  v-model="form.name"
                  required
                  class="fm-input"
                  :placeholder="t('tenantManagement.tenantName')"
                />
                <p
                  v-if="isAdd"
                  class="mt-2 flex items-start gap-1.5 text-xs font-medium leading-5 text-slate-500 dark:text-slate-400"
                >
                  <span
                    class="mt-[6px] h-1.5 w-1.5 shrink-0 rounded-full bg-blue-500/70"
                  ></span>
                  <span>{{ t('tenantManagement.tenantNameUsageHint') }}</span>
                </p>
              </div>
              <div>
                <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                  {{ t('tenantManagement.tenantCode') }}
                  <span class="text-rose-500 ml-1">*</span>
                </label>
                <input
                  v-model="form.code"
                  required
                  :disabled="!isAdd"
                  class="fm-input disabled:opacity-50 disabled:bg-slate-50 dark:disabled:bg-slate-800/50"
                  placeholder="e.g. my-tenant-01"
                />
              </div>
            </div>

            <div>
              <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                {{ t('tenantManagement.description') }}
              </label>
              <textarea
                v-model="form.description"
                rows="3"
                class="fm-input fm-ta"
                placeholder="..."
              ></textarea>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                  {{ t('tenantManagement.contactEmail') }}
                </label>
                <input
                  v-model="form.contactEmail"
                  type="email"
                  class="fm-input"
                  placeholder="name@example.com"
                />
              </div>
              <div>
                <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                  {{ t('tenantManagement.contactPhone') }}
                </label>
                <input v-model="form.contactPhone" class="fm-input" placeholder="+86 123456789" />
              </div>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                  {{ t('tenantManagement.maxUsers') }}
                </label>
                <input v-model.number="form.maxUsers" type="number" min="1" class="fm-input" />
              </div>
              <div>
                <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                  {{ t('tenantManagement.maxProjects') }}
                </label>
                <input v-model.number="form.maxProjects" type="number" min="1" class="fm-input" />
              </div>
            </div>

            <div
              v-if="!isAdd"
              class="p-4 rounded-2xl border border-slate-200/60 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-900/30"
            >
              <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                {{ t('tenantManagement.status') }}
              </label>
              <select
                v-model="form.status"
                class="fm-input cursor-pointer bg-white dark:bg-slate-900"
              >
                <option value="active">
                  {{ t('tenantManagement.statusActive') }}
                </option>
                <option value="inactive">
                  {{ t('tenantManagement.statusInactive') }}
                </option>
                <option value="suspended">
                  {{ t('tenantManagement.statusSuspended') }}
                </option>
              </select>
            </div>
          </section>

          <!-- ====== 公司信息（含备案号） ====== -->
          <section v-show="curTab === 'company'" class="space-y-6">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                  {{ t('tenantManagement.companyName') }}
                </label>
                <input v-model="form.companyName" class="fm-input" />
              </div>
              <div>
                <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                  {{ t('tenantManagement.companyPhone') }}
                </label>
                <input v-model="form.companyPhone" class="fm-input" />
              </div>
            </div>

            <div>
              <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                {{ t('tenantManagement.companyAddress') }}
              </label>
              <textarea v-model="form.companyAddress" rows="2" class="fm-input fm-ta"></textarea>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div class="md:col-span-2">
                <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                  {{ t('tenantManagement.companyWebsite') }}
                </label>
                <input
                  v-model="form.companyWebsite"
                  type="url"
                  class="fm-input"
                  placeholder="https://"
                />
              </div>
              <div class="md:col-span-2">
                <label class="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                  {{ t('tenantManagement.icpNumber') }}
                </label>
                <input
                  v-model="form.icpNumber"
                  class="fm-input"
                  :placeholder="t('tenantManagement.icpPlaceholder')"
                />
              </div>
            </div>

            <div class="h-px bg-slate-200/60 dark:bg-slate-800 my-8"></div>

            <div
              class="bg-slate-50/50 dark:bg-slate-900/30 p-5 rounded-2xl border border-slate-200/60 dark:border-slate-800"
            >
              <h3
                class="text-sm font-bold text-slate-800 dark:text-slate-200 mb-4 flex items-center gap-2"
              >
                <svg
                  class="w-4 h-4 text-slate-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                  />
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                  />
                </svg>
                {{ t('tenantManagement.loginDisplaySettings') }}
              </h3>
              <div class="grid grid-cols-1 md:grid-cols-2 gap-y-3 gap-x-6">
                <label
                  class="flex items-center gap-3 text-sm font-medium text-slate-600 dark:text-slate-400 cursor-pointer hover:bg-white dark:hover:bg-slate-800/50 p-2.5 rounded-xl transition-all shadow-sm border border-transparent hover:border-slate-200 dark:hover:border-slate-700"
                >
                  <input
                    v-model="form.showCompanyName"
                    type="checkbox"
                    class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
                  />
                  <span>{{ t('tenantManagement.showCompanyName') }}</span>
                </label>
                <label
                  class="flex items-center gap-3 text-sm font-medium text-slate-600 dark:text-slate-400 cursor-pointer hover:bg-white dark:hover:bg-slate-800/50 p-2.5 rounded-xl transition-all shadow-sm border border-transparent hover:border-slate-200 dark:hover:border-slate-700"
                >
                  <input
                    v-model="form.showCompanyPhone"
                    type="checkbox"
                    class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
                  />
                  <span>{{ t('tenantManagement.showCompanyPhone') }}</span>
                </label>
                <label
                  class="flex items-center gap-3 text-sm font-medium text-slate-600 dark:text-slate-400 cursor-pointer hover:bg-white dark:hover:bg-slate-800/50 p-2.5 rounded-xl transition-all shadow-sm border border-transparent hover:border-slate-200 dark:hover:border-slate-700"
                >
                  <input
                    v-model="form.showCompanyAddress"
                    type="checkbox"
                    class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
                  />
                  <span>{{ t('tenantManagement.showCompanyAddress') }}</span>
                </label>
                <label
                  class="flex items-center gap-3 text-sm font-medium text-slate-600 dark:text-slate-400 cursor-pointer hover:bg-white dark:hover:bg-slate-800/50 p-2.5 rounded-xl transition-all shadow-sm border border-transparent hover:border-slate-200 dark:hover:border-slate-700"
                >
                  <input
                    v-model="form.showCompanyWebsite"
                    type="checkbox"
                    class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
                  />
                  <span>{{ t('tenantManagement.showCompanyWebsite') }}</span>
                </label>
                <label
                  class="flex items-center gap-3 text-sm font-medium text-slate-600 dark:text-slate-400 cursor-pointer hover:bg-white dark:hover:bg-slate-800/50 p-2.5 rounded-xl transition-all shadow-sm border border-transparent hover:border-slate-200 dark:hover:border-slate-700"
                >
                  <input
                    v-model="form.showIcp"
                    type="checkbox"
                    class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
                  />
                  <span>{{ t('tenantManagement.showIcp') }}</span>
                </label>
              </div>
            </div>
          </section>

          <!-- ====== 品牌资产 ====== -->
          <section v-show="curTab === 'brand'" class="space-y-8">
            <div class="flex flex-col gap-3">
              <div>
                <p class="text-sm font-bold text-slate-800 dark:text-slate-200">
                  {{ t('tenantManagement.logo') }}
                </p>
                <p class="text-xs font-medium text-slate-500 dark:text-slate-400 mt-1">
                  推荐 200×200，PNG / JPG / SVG
                </p>
              </div>
              <el-upload
                class="tm-upload-card mt-1"
                list-type="picture-card"
                :auto-upload="true"
                accept="image/*"
                :file-list="logoFileList"
                :before-upload="beforeImageUpload"
                :http-request="uploadLogoRequest"
                :on-preview="handleUploadPreview"
                :on-remove="removeLogo"
              >
                <div
                  class="flex flex-col items-center justify-center text-slate-400 dark:text-slate-500 hover:text-blue-500 transition-colors"
                >
                  <svg class="w-6 h-6 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M12 6v6m0 0v6m0-6h6m-6 0H6"
                    />
                  </svg>
                  <span class="text-xs font-medium">上传 Logo</span>
                </div>
              </el-upload>
            </div>

            <div class="h-px bg-slate-200/60 dark:bg-slate-800"></div>

            <div class="flex flex-col gap-3">
              <div>
                <p class="text-sm font-bold text-slate-800 dark:text-slate-200">
                  {{ t('tenantManagement.loginBackground') }}
                </p>
                <p class="text-xs font-medium text-slate-500 dark:text-slate-400 mt-1">
                  推荐 1920×1080，JPG 或 PNG
                </p>
              </div>
              <el-upload
                class="tm-upload-card tm-upload-card-bg mt-1"
                list-type="picture-card"
                :auto-upload="true"
                accept="image/*"
                :file-list="backgroundFileList"
                :before-upload="beforeImageUpload"
                :http-request="uploadBackgroundRequest"
                :on-preview="handleUploadPreview"
                :on-remove="removeBackground"
              >
                <div
                  class="flex flex-col items-center justify-center text-slate-400 dark:text-slate-500 hover:text-blue-500 transition-colors"
                >
                  <svg class="w-6 h-6 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                    />
                  </svg>
                  <span class="text-xs font-medium">上传背景图</span>
                </div>
              </el-upload>
            </div>

            <div
              class="flex items-center gap-5 mt-8 p-5 border border-dashed border-slate-300 dark:border-slate-700 rounded-2xl bg-slate-50 dark:bg-slate-800/30"
            >
              <div class="relative">
                <img
                  :src="logoPreviewUrl"
                  alt="logo"
                  class="w-14 h-14 rounded-xl object-cover border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 shadow-sm"
                />
                <div
                  class="absolute -bottom-1 -right-1 w-4 h-4 bg-emerald-500 border-2 border-white dark:border-slate-900 rounded-full"
                ></div>
              </div>
              <div>
                <p class="text-sm font-extrabold text-slate-900 dark:text-white">
                  {{ form.name || t('tenantManagement.tenantName') }}
                </p>
                <p class="text-xs font-medium text-slate-500 dark:text-slate-400 mt-1">
                  {{ form.code || 'tenant-code' }}
                </p>
              </div>
            </div>
          </section>
        </div>

        <footer
          class="flex items-center justify-end gap-3 px-8 py-5 border-t border-slate-100 dark:border-slate-800/60 bg-slate-50/50 dark:bg-slate-900/50 backdrop-blur-sm z-20"
        >
          <button
            type="button"
            class="px-5 py-2.5 text-sm font-semibold text-slate-700 bg-white border border-slate-300 rounded-xl hover:bg-slate-50 focus:outline-none focus:ring-4 focus:ring-slate-100 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-600 dark:hover:bg-slate-700 dark:focus:ring-slate-800 transition-all shadow-sm active:scale-[0.98]"
            @click="closeDrawer"
          >
            {{ t('tenantManagement.cancel') }}
          </button>
          <button
            type="button"
            class="px-5 py-2.5 text-sm font-semibold text-white bg-blue-600 border border-transparent rounded-xl shadow-sm hover:shadow-md hover:bg-blue-700 focus:outline-none focus:ring-4 focus:ring-blue-500/20 dark:focus:ring-blue-900/40 disabled:opacity-50 disabled:cursor-not-allowed transition-all active:scale-[0.98]"
            :disabled="saving"
            @click="save"
          >
            <span v-if="saving" class="flex items-center gap-2">
              <svg
                class="animate-spin h-4 w-4 text-white"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle
                  class="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  stroke-width="4"
                ></circle>
                <path
                  class="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                ></path>
              </svg>
              {{ t('tenantManagement.saving') }}
            </span>
            <span v-else>{{ t('tenantManagement.save') }}</span>
          </button>
        </footer>
      </aside>
    </transition>

    <el-dialog
      v-model="previewVisible"
      width="680px"
      :title="t('tenantManagement.preview')"
      append-to-body
    >
      <div class="flex justify-center">
        <img
          :src="previewImageUrl"
          alt="preview"
          class="max-w-full max-h-[70vh] rounded-2xl shadow-xl border border-slate-200 dark:border-slate-700"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { tenantAPI } from '@/api'
import defaultLogoUrl from '@/assets/images/default-logo.svg'
import defaultLoginBgUrl from '@/assets/images/default-login-bg.svg'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  mode: {
    type: String,
    default: 'add',
  },
  initialData: {
    type: Object,
    default: () => ({}),
  },
})

const emit = defineEmits(['update:visible', 'saved', 'updateListBrand'])

const { t } = useI18n()

const saving = ref(false)
const curTab = ref('basic')
const isAdd = computed(() => props.mode === 'add')

const tabs = computed(() => [
  { k: 'basic', l: t('tenantManagement.tenantInfo') },
  { k: 'company', l: t('tenantManagement.companyInfo') },
  { k: 'brand', l: t('tenantManagement.brandAssets') },
])

const form = reactive({
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
  showCompanyName: true,
  showCompanyPhone: false,
  showCompanyAddress: false,
  showCompanyWebsite: false,
  showIcp: false,
  icpNumber: '',
})

const previewVisible = ref(false)
const previewImageUrl = ref('')

const logoPreviewUrl = computed(() => form.logoUrl || defaultLogoUrl)
const backgroundPreviewUrl = computed(() => form.loginBackgroundUrl || defaultLoginBgUrl)

const logoFileList = computed(() =>
  form.logoUrl ? [{ name: 'logo', url: form.logoUrl, status: 'success' }] : [],
)
const backgroundFileList = computed(() =>
  form.loginBackgroundUrl
    ? [{ name: 'background', url: form.loginBackgroundUrl, status: 'success' }]
    : [],
)

watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      curTab.value = 'basic'
      if (isAdd.value) {
        resetForm()
      } else {
        fillForm(props.initialData)
      }
    }
  },
)

const resetForm = () => {
  Object.assign(form, {
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
    showCompanyName: true,
    showCompanyPhone: false,
    showCompanyAddress: false,
    showCompanyWebsite: false,
    showIcp: false,
    icpNumber: '',
  })
}

const fillForm = (tenant: any) => {
  const ld = tenant.settings?.loginDisplay || {}
  Object.assign(form, {
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
    showCompanyName: ld.showCompanyName !== false,
    showCompanyPhone: Boolean(ld.showCompanyPhone),
    showCompanyAddress: Boolean(ld.showCompanyAddress),
    showCompanyWebsite: Boolean(ld.showCompanyWebsite),
    showIcp: Boolean(ld.showIcp),
    icpNumber: ld.icpNumber || '',
  })
}

const closeDrawer = () => {
  emit('update:visible', false)
}

const save = async () => {
  if (!form.name || !form.code) {
    ElMessage.warning(t('common.pleaseFillRequiredFields'))
    return
  }

  saving.value = true
  try {
    const payload = {
      ...form,
      settings: {
        loginDisplay: {
          showCompanyName: Boolean(form.showCompanyName),
          showCompanyPhone: Boolean(form.showCompanyPhone),
          showCompanyAddress: Boolean(form.showCompanyAddress),
          showCompanyWebsite: Boolean(form.showCompanyWebsite),
          showIcp: Boolean(form.showIcp),
          icpNumber: form.icpNumber || '',
        },
      },
    }

    if (isAdd.value) {
      await tenantAPI.createTenant(payload)
      ElMessage.success(t('tenantManagement.createSuccess'))
    } else {
      await tenantAPI.updateTenant(form.code, payload)
      ElMessage.success(t('tenantManagement.updateSuccess'))
    }

    emit('saved')
    closeDrawer()
  } catch (e: any) {
    console.error(e)
    ElMessage.error(e.response?.data?.message || t('tenantManagement.saveFailed'))
  } finally {
    saving.value = false
  }
}

const fileToDataUrl = (file: File) =>
  new Promise((resolve, reject) => {
    const r = new window.FileReader()
    r.onload = () => resolve(r.result)
    r.onerror = () => reject(new Error('读取失败'))
    r.readAsDataURL(file)
  })

const beforeImageUpload = (file: File) => {
  const isImage = file.type.startsWith('image/')
  if (!isImage) {
    ElMessage.error(t('tenantManagement.uploadImageOnly'))
    return false
  }
  const isLt5MB = file.size / 1024 / 1024 < 5
  if (!isLt5MB) {
    ElMessage.error(t('tenantManagement.uploadSizeLimit'))
    return false
  }
  return true
}

const uploadLogoRequest = async (options: any) => {
  const f = options.file
  try {
    if (isAdd.value) {
      form.logoUrl = (await fileToDataUrl(f)) as string
      ElMessage.success(t('tenantManagement.logoUploadSuccess'))
      options.onSuccess?.({}, f)
      return true
    }
    const fd = new window.FormData()
    fd.append('file', f)
    const res = await tenantAPI.uploadFile(form.code, 'logo', fd)
    form.logoUrl = res.data.fileUrl

    emit('updateListBrand', { code: form.code, logoUrl: res.data.fileUrl })

    ElMessage.success(t('tenantManagement.logoUploadSuccess'))
    options.onSuccess?.(res, f)
    return true
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message || t('tenantManagement.logoUploadFailed'))
    options.onError?.(err)
    return false
  }
}

const uploadBackgroundRequest = async (options: any) => {
  const f = options.file
  try {
    if (isAdd.value) {
      form.loginBackgroundUrl = (await fileToDataUrl(f)) as string
      ElMessage.success(t('tenantManagement.backgroundUploadSuccess'))
      options.onSuccess?.({}, f)
      return true
    }
    const fd = new window.FormData()
    fd.append('file', f)
    const res = await tenantAPI.uploadFile(form.code, 'background', fd)
    form.loginBackgroundUrl = res.data.fileUrl

    emit('updateListBrand', {
      code: form.code,
      loginBackgroundUrl: res.data.fileUrl,
    })

    ElMessage.success(t('tenantManagement.backgroundUploadSuccess'))
    options.onSuccess?.(res, f)
    return true
  } catch (err: any) {
    ElMessage.error(err.response?.data?.message || t('tenantManagement.backgroundUploadFailed'))
    options.onError?.(err)
    return false
  }
}

const removeLogo = () => {
  form.logoUrl = ''
}

const removeBackground = () => {
  form.loginBackgroundUrl = ''
}

const handleUploadPreview = (file: any) => {
  previewImageUrl.value = file.url || file.response?.data?.fileUrl || ''
  if (!previewImageUrl.value) return
  previewVisible.value = true
}
</script>

<style scoped>
.tm-mask {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.4);
  z-index: 1000;
  transition: all 0.4s cubic-bezier(0.16, 1, 0.3, 1);
}

html.dark .tm-mask {
  background: rgba(0, 0, 0, 0.6);
}

.tm-drawer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  width: min(600px, 100vw);
  z-index: 1001;
  display: flex;
  flex-direction: column;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.slide-enter-active,
.slide-leave-active {
  transition: transform 0.5s cubic-bezier(0.16, 1, 0.3, 1);
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
}

.fm-input {
  width: 100%;
  height: 42px;
  padding: 0 16px;
  border: 1px solid #cbd5e1;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 500;
  color: #0f172a;
  background: #ffffff;
  outline: none;
  transition: all 0.2s ease;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  text-overflow: ellipsis;
}

.fm-input::placeholder {
  color: #94a3b8;
  text-overflow: ellipsis;
}

select.fm-input {
  appearance: none;
  padding-right: 36px;
  background-image: url("data:image/svg+xml;charset=UTF-8,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 20 20' fill='%2364748b'%3e%3cpath fill-rule='evenodd' d='M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z' clip-rule='evenodd'/%3e%3c/svg%3e");
  background-repeat: no-repeat;
  background-position: right 12px center;
  background-size: 16px 16px;
}

.fm-input:focus {
  border-color: #3b82f6;
  box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.1);
}

.fm-ta {
  height: auto;
  min-height: 100px;
  padding: 12px 16px;
  resize: vertical;
}

html.dark .fm-input {
  background: #0f172a;
  border-color: #334155;
  color: #f8fafc;
}

html.dark select.fm-input {
  background-image: url("data:image/svg+xml;charset=UTF-8,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 20 20' fill='%2394a3b8'%3e%3cpath fill-rule='evenodd' d='M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z' clip-rule='evenodd'/%3e%3c/svg%3e");
}

html.dark .fm-input:focus {
  border-color: #3b82f6;
  box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.2);
}

/* Upload styles */
:deep(.tm-upload-card .el-upload--picture-card),
:deep(.tm-upload-card .el-upload-list__item) {
  width: 120px;
  height: 120px;
  border-radius: 16px;
  border: 2px dashed #cbd5e1;
  background: #f8fafc;
  transition: all 0.2s;
}

:deep(.tm-upload-card-bg .el-upload--picture-card),
:deep(.tm-upload-card-bg .el-upload-list__item) {
  width: 200px;
  height: 120px;
}

html.dark :deep(.tm-upload-card .el-upload--picture-card) {
  background: #0f172a;
  border-color: #334155;
}

:deep(.tm-upload-card .el-upload--picture-card:hover) {
  border-color: #3b82f6;
  background: #eff6ff;
}

html.dark :deep(.tm-upload-card .el-upload--picture-card:hover) {
  border-color: #3b82f6;
  background: #1e3a8a;
}

/* Custom scrollbar */
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: #cbd5e1;
  border-radius: 10px;
}
html.dark .custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: #334155;
}
</style>
