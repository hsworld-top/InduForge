<template>
  <div
    class="p-6 max-w-[1200px] mx-auto w-full font-sans antialiased"
    :class="{
      'px-2 py-2 max-h-[70vh] overflow-y-auto custom-scrollbar': embedded,
    }"
  >
    <!-- 头部资料卡片 -->
    <div
      class="relative overflow-hidden bg-white dark:bg-slate-900 border border-slate-200/60 dark:border-slate-800 rounded-3xl p-6 md:p-8 shadow-sm mb-6 flex flex-col md:flex-row md:items-center justify-between gap-6 transition-all"
    >
      <!-- 装饰性背景光晕 -->
      <div
        class="absolute top-0 right-0 w-64 h-64 bg-blue-500/5 rounded-full blur-3xl -mr-20 -mt-20 pointer-events-none"
      ></div>

      <div class="flex items-center gap-6 relative z-10">
        <el-avatar
          :size="80"
          :src="profile.avatarUrl"
          class="border-4 border-white dark:border-slate-800 shadow-md bg-gradient-to-br from-blue-100 to-indigo-50 dark:from-blue-900/40 dark:to-indigo-900/40 text-blue-600 dark:text-blue-400 font-extrabold text-2xl"
        >
          {{ userInitials }}
        </el-avatar>
        <div>
          <h2 class="text-2xl font-extrabold text-slate-900 dark:text-white tracking-tight">
            {{ profile.username || '-' }}
          </h2>
          <div class="flex flex-wrap items-center gap-3 mt-2.5">
            <span
              class="px-2.5 py-1 text-xs font-bold text-blue-700 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 rounded-lg"
              >{{ getRoleLabel(profile.role) }}</span
            >
            <span
              class="text-sm font-medium text-slate-500 dark:text-slate-400 flex items-center gap-1.5"
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
                />
              </svg>
              {{ getTenantLabel(profile.tenant) }}
            </span>
          </div>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-3 relative z-10">
        <el-upload
          :auto-upload="false"
          :show-file-list="false"
          :on-change="handleAvatarChange"
          accept="image/png,image/jpeg,image/webp"
        >
          <button
            type="button"
            class="px-5 py-2.5 text-sm font-semibold text-slate-700 bg-white border border-slate-300 rounded-xl hover:bg-slate-50 focus:ring-4 focus:ring-slate-100 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-600 dark:hover:bg-slate-700 transition-all shadow-sm active:scale-[0.98]"
          >
            {{ t('profile.uploadAvatar') }}
          </button>
        </el-upload>
        <button
          type="button"
          class="flex items-center gap-2 px-5 py-2.5 text-sm font-semibold text-slate-700 bg-white border border-slate-300 rounded-xl hover:bg-slate-50 focus:ring-4 focus:ring-slate-100 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-600 dark:hover:bg-slate-700 transition-all shadow-sm active:scale-[0.98]"
          @click="loadProfile"
          :disabled="loading"
        >
          <svg v-if="loading" class="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
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
          <svg
            v-else
            class="w-4 h-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
          {{ t('common.refresh') }}
        </button>
      </div>
    </div>

    <!-- 底部表单卡片 -->
    <div class="grid grid-cols-1 xl:grid-cols-2 gap-6">
      <!-- 账号信息 -->
      <div
        class="bg-white dark:bg-slate-900 border border-slate-200/60 dark:border-slate-800 rounded-3xl p-6 md:p-8 shadow-sm"
      >
        <h3
          class="text-lg font-extrabold text-slate-900 dark:text-white mb-6 flex items-center gap-2"
        >
          <div class="p-2 bg-slate-100 dark:bg-slate-800 rounded-lg">
            <svg
              class="w-5 h-5 text-slate-500 dark:text-slate-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
              />
            </svg>
          </div>
          {{ t('profile.accountInfo') }}
        </h3>

        <div class="space-y-4">
          <div
            class="flex items-center justify-between p-4 rounded-2xl bg-slate-50/50 dark:bg-slate-800/30 border border-slate-100 dark:border-slate-800/60"
          >
            <span class="text-sm font-semibold text-slate-500 dark:text-slate-400">{{
              t('profile.username')
            }}</span>
            <span class="text-sm font-bold text-slate-900 dark:text-white">{{
              profile.username || '-'
            }}</span>
          </div>
          <div
            class="flex items-center justify-between p-4 rounded-2xl bg-slate-50/50 dark:bg-slate-800/30 border border-slate-100 dark:border-slate-800/60"
          >
            <span class="text-sm font-semibold text-slate-500 dark:text-slate-400">{{
              t('profile.role')
            }}</span>
            <span class="text-sm font-bold text-slate-900 dark:text-white">{{
              getRoleLabel(profile.role)
            }}</span>
          </div>
          <div
            class="flex items-center justify-between p-4 rounded-2xl bg-slate-50/50 dark:bg-slate-800/30 border border-slate-100 dark:border-slate-800/60 cursor-pointer hover:border-blue-300 dark:hover:border-blue-700 transition-colors group"
            @click="startEmailEdit"
          >
            <span
              class="text-sm font-semibold text-slate-500 dark:text-slate-400 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors"
              >{{ t('profile.email') }}</span
            >
            <div class="flex items-center gap-2">
              <template v-if="isEditingEmail">
                <div ref="emailRowRef" class="w-[240px]" @click.stop>
                  <el-input
                    ref="emailInputRef"
                    v-model="accountForm.email"
                    :placeholder="t('profile.emailPlaceholder')"
                    clearable
                    style="width: 100%"
                    @click.stop
                    @mousedown.stop
                    @keydown.enter.stop.prevent="exitEmailEdit(true)"
                    @keydown.esc.stop.prevent="cancelEmailEdit"
                    @blur="handleEmailBlur"
                  />
                </div>
              </template>
              <div v-else class="flex items-center gap-2">
                <span class="text-sm font-bold text-slate-900 dark:text-white">{{
                  profile.email || '-'
                }}</span>
                <svg
                  class="w-4 h-4 text-slate-400 opacity-0 group-hover:opacity-100 transition-opacity"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
                  />
                </svg>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 偏好设置 -->
      <div
        class="bg-white dark:bg-slate-900 border border-slate-200/60 dark:border-slate-800 rounded-3xl p-6 md:p-8 shadow-sm"
      >
        <h3
          class="text-lg font-extrabold text-slate-900 dark:text-white mb-6 flex items-center gap-2"
        >
          <div class="p-2 bg-slate-100 dark:bg-slate-800 rounded-lg">
            <svg
              class="w-5 h-5 text-slate-500 dark:text-slate-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
              />
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
              />
            </svg>
          </div>
          {{ t('profile.preferences') }}
        </h3>
        <el-form
          ref="preferencesFormRef"
          :model="preferencesForm"
          :rules="preferencesRules"
          label-position="top"
        >
          <el-form-item :label="t('profile.language')" prop="language">
            <el-select
              v-model="preferencesForm.language"
              style="width: 100%"
              size="large"
              class="modern-select"
            >
              <el-option
                v-for="lang in languageOptions"
                :key="lang.value"
                :label="lang.label"
                :value="lang.value"
              />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('profile.theme')" prop="theme">
            <el-select
              v-model="preferencesForm.theme"
              style="width: 100%"
              size="large"
              class="modern-select"
            >
              <el-option
                v-for="theme in themeOptions"
                :key="theme.value"
                :label="theme.label"
                :value="theme.value"
              />
            </el-select>
          </el-form-item>
          <div class="pt-4">
            <button
              type="button"
              class="w-full px-5 py-3 text-sm font-semibold text-white bg-blue-600 rounded-xl hover:bg-blue-700 focus:ring-4 focus:ring-blue-500/20 dark:focus:ring-blue-900/40 transition-all shadow-md active:scale-[0.98] disabled:opacity-70 flex justify-center items-center gap-2"
              :disabled="savingPreferences"
              @click="handleSavePreferences"
            >
              <svg
                v-if="savingPreferences"
                class="animate-spin h-4 w-4"
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
              {{ t('profile.savePreferences') }}
            </button>
          </div>
        </el-form>
      </div>

      <!-- 修改密码 -->
      <div
        class="bg-white dark:bg-slate-900 border border-slate-200/60 dark:border-slate-800 rounded-3xl p-6 md:p-8 shadow-sm xl:col-span-2"
      >
        <h3
          class="text-lg font-extrabold text-slate-900 dark:text-white mb-6 flex items-center gap-2"
        >
          <div class="p-2 bg-slate-100 dark:bg-slate-800 rounded-lg">
            <svg
              class="w-5 h-5 text-slate-500 dark:text-slate-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
              />
            </svg>
          </div>
          {{ t('profile.changePassword') }}
        </h3>
        <el-form
          ref="passwordFormRef"
          :model="passwordForm"
          :rules="passwordRules"
          label-position="top"
        >
          <div class="grid grid-cols-1 md:grid-cols-3 gap-x-6">
            <el-form-item :label="t('profile.oldPassword')" prop="oldPassword">
              <el-input
                v-model="passwordForm.oldPassword"
                type="password"
                show-password
                size="large"
                :placeholder="t('profile.oldPasswordPlaceholder')"
                class="modern-input"
              />
            </el-form-item>
            <el-form-item :label="t('profile.newPassword')" prop="newPassword">
              <el-input
                v-model="passwordForm.newPassword"
                type="password"
                show-password
                size="large"
                :placeholder="t('profile.newPasswordPlaceholder')"
                class="modern-input"
              />
            </el-form-item>
            <el-form-item :label="t('profile.confirmPassword')" prop="confirmPassword">
              <el-input
                v-model="passwordForm.confirmPassword"
                type="password"
                show-password
                size="large"
                :placeholder="t('profile.confirmPasswordPlaceholder')"
                class="modern-input"
              />
            </el-form-item>
          </div>
          <div class="pt-4 flex justify-end">
            <button
              type="button"
              class="px-6 py-3 text-sm font-semibold text-white bg-slate-800 dark:bg-slate-700 rounded-xl hover:bg-slate-900 dark:hover:bg-slate-600 focus:ring-4 focus:ring-slate-500/20 transition-all shadow-md active:scale-[0.98] disabled:opacity-70 flex justify-center items-center gap-2"
              :disabled="savingPassword"
              @click="handleChangePassword"
            >
              <svg
                v-if="savingPassword"
                class="animate-spin h-4 w-4"
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
              {{ t('profile.savePassword') }}
            </button>
          </div>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
// @ts-nocheck
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { authAPI, userAPI } from '@/api'
import { useAuthStore, useAppStore } from '@/store'
import { Storage } from '@/utils/storage'
import { getApiErrorMessage } from '@/utils/request'
import { RoleEnum } from '@/enums'

const AVATAR_KEY_PREFIX = 'profile_avatar_'

export default {
  name: 'Profile',
  props: {
    embedded: {
      type: Boolean,
      default: false,
    },
  },
  setup() {
    const { t, locale } = useI18n()
    const authStore = useAuthStore()
    const appStore = useAppStore()
    const loading = ref(false)
    const savingPassword = ref(false)
    const savingPreferences = ref(false)
    const isEditingEmail = ref(false)
    const emailInputRef = ref(null)
    const emailRowRef = ref(null)
    const passwordFormRef = ref(null)
    const preferencesFormRef = ref(null)

    const profile = ref({
      id: '',
      username: '',
      email: '',
      role: '',
      tenant: null,
      avatarUrl: '',
    })

    const userInitials = computed(() => {
      const username = profile.value.username || 'U'
      return username.charAt(0).toUpperCase()
    })

    const passwordForm = reactive({
      oldPassword: '',
      newPassword: '',
      confirmPassword: '',
    })

    const accountForm = reactive({
      email: '',
    })

    const preferencesForm = reactive({
      language: appStore.language || 'zh',
      theme: appStore.theme || 'light',
    })

    const languageOptions = computed(() => [
      { value: 'zh', label: t('system.languageZh') },
      { value: 'en', label: t('system.languageEn') },
    ])

    const themeOptions = computed(() => [
      { value: 'light', label: t('system.light') },
      { value: 'dark', label: t('system.dark') },
    ])

    const preferencesRules = {}

    const passwordRules = {
      oldPassword: [
        {
          required: true,
          message: t('profile.requireOldPassword'),
          trigger: 'blur',
        },
      ],
      newPassword: [
        {
          required: true,
          message: t('profile.requireNewPassword'),
          trigger: 'blur',
        },
        { min: 6, message: t('profile.passwordMinLength'), trigger: 'blur' },
      ],
      confirmPassword: [
        {
          required: true,
          message: t('profile.requireConfirmPassword'),
          trigger: 'blur',
        },
        {
          validator: (rule, value, callback) => {
            if (value !== passwordForm.newPassword) {
              callback(new Error(t('profile.passwordMismatch')))
            } else {
              callback()
            }
          },
          trigger: 'blur',
        },
      ],
    }

    /**
     * 生成头像缓存键。
     * @param {string} userId - 用户 ID
     * @returns {string} 本地缓存键
     */
    const getAvatarStorageKey = (userId) => `${AVATAR_KEY_PREFIX}${userId || 'anonymous'}`

    /**
     * 同步用户信息到全局状态和本地缓存。
     */
    const syncUserCache = () => {
      if (!profile.value.id) return
      authStore.userInfo = {
        ...authStore.userInfo,
        id: profile.value.id,
        username: profile.value.username,
        email: profile.value.email,
        role: profile.value.role,
        tenantId: profile.value.tenant?.id || authStore.userInfo?.tenantId,
        tenant: profile.value.tenant,
        avatarUrl: profile.value.avatarUrl || '',
      }
      Storage.setUserInfo(authStore.userInfo)
    }

    const getRoleLabel = (role) => {
      const roleLabelMap = {
        [RoleEnum.SUPER_ADMIN]: t('profile.roleSuperAdmin'),
        [RoleEnum.SYSTEM_ADMIN]: t('profile.roleSystemAdmin'),
        [RoleEnum.PROJECT_ADMIN]: t('profile.roleProjectAdmin'),
        [RoleEnum.OPS_ADMIN]: t('profile.roleOpsAdmin'),
        [RoleEnum.USER_ADMIN]: t('profile.roleUserAdmin'),
        [RoleEnum.USER]: t('profile.roleUser'),
      }
      return roleLabelMap[role] || role || '-'
    }

    const getTenantLabel = (tenant) => {
      const tenantName = tenant?.name?.trim()
      if (!tenantName) return t('profile.unboundTenant')
      if (tenantName === '默认租户' || tenantName === 'Default Tenant') {
        return t('profile.defaultTenant')
      }
      return tenantName
    }

    const loadProfile = async () => {
      loading.value = true
      try {
        const response = await authAPI.getCurrentUser()
        const user = response.data?.user || response.data || {}
        const avatarFromCache = localStorage.getItem(getAvatarStorageKey(user.id))
        profile.value = {
          id: user.id || '',
          username: user.username || '',
          email: user.email || authStore.userInfo?.email || '',
          role: user.role || '',
          tenant: user.tenant || null,
          avatarUrl: user.avatarUrl || avatarFromCache || authStore.userInfo?.avatarUrl || '',
        }
        accountForm.email = profile.value.email
        isEditingEmail.value = false
        preferencesForm.language = appStore.language || 'zh'
        preferencesForm.theme = appStore.theme || 'light'
        syncUserCache()
      } catch {
        const localUser = Storage.getUserInfo()
        const avatarFromCache = localStorage.getItem(getAvatarStorageKey(localUser?.id))
        profile.value = {
          id: localUser?.id || '',
          username: localUser?.username || '',
          email: localUser?.email || '',
          role: localUser?.role || '',
          tenant: localUser?.tenant || null,
          avatarUrl: localUser?.avatarUrl || avatarFromCache || '',
        }
        accountForm.email = profile.value.email
        isEditingEmail.value = false
        preferencesForm.language = appStore.language || 'zh'
        preferencesForm.theme = appStore.theme || 'light'
        ElMessage.warning(t('profile.profileLoadFailed'))
      } finally {
        loading.value = false
      }
    }

    const startEmailEdit = async () => {
      if (isEditingEmail.value) return
      accountForm.email = profile.value.email || ''
      isEditingEmail.value = true
      await nextTick()
      emailInputRef.value?.focus?.()
    }

    const handleSaveAccountInfo = async () => {
      const userId = profile.value.id || authStore.userInfo?.id
      if (!userId) {
        ElMessage.error(t('profile.noUserInfo'))
        return false
      }

      try {
        const nextEmail = (accountForm.email || '').trim()

        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
        if (nextEmail && !emailRegex.test(nextEmail)) {
          ElMessage.warning(t('profile.invalidEmail'))
          return false
        }

        if (nextEmail !== (profile.value.email || '')) {
          await userAPI.updateUser(userId, { email: nextEmail })
          profile.value.email = nextEmail
          syncUserCache()
          ElMessage.success(t('profile.accountUpdated'))
        }
        return true
      } catch (error) {
        ElMessage.error(
          t('profile.accountUpdateFailed', {
            message: getApiErrorMessage(error, t('auth.retry')),
          }),
        )
        return false
      }
    }

    const exitEmailEdit = async (autoSave = false) => {
      if (!isEditingEmail.value) return
      if (autoSave) {
        const saved = await handleSaveAccountInfo()
        if (!saved) return
      } else {
        accountForm.email = profile.value.email || ''
      }
      isEditingEmail.value = false
    }

    const cancelEmailEdit = () => {
      if (!isEditingEmail.value) return
      accountForm.email = profile.value.email || ''
      isEditingEmail.value = false
    }

    const handleEmailBlur = () => {
      window.setTimeout(() => {
        void exitEmailEdit(true)
      }, 0)
    }

    const handleDocumentClick = (event) => {
      if (!isEditingEmail.value) return
      const rowEl = emailRowRef.value
      if (rowEl && !rowEl.contains(event.target)) {
        void exitEmailEdit(true)
      }
    }

    const handleSavePreferences = async () => {
      if (!preferencesFormRef.value) return

      savingPreferences.value = true
      try {
        if (preferencesForm.language !== appStore.language) {
          appStore.setLanguage(preferencesForm.language)
          locale.value = preferencesForm.language
        }

        if (preferencesForm.theme !== appStore.theme) {
          appStore.setTheme(preferencesForm.theme)
        }

        ElMessage.success(t('profile.preferencesUpdated'))
      } catch (error) {
        ElMessage.error(
          t('profile.preferencesUpdateFailed', {
            message: getApiErrorMessage(error, t('auth.retry')),
          }),
        )
      } finally {
        savingPreferences.value = false
      }
    }

    /**
     * 处理头像上传，当前版本以前端缓存方式保存。
     * @param {import('element-plus').UploadFile} uploadFile - 上传文件对象
     */
    const handleAvatarChange = (uploadFile) => {
      const rawFile = uploadFile.raw
      if (!rawFile) return
      const isImage = rawFile.type.startsWith('image/')
      const isValidSize = rawFile.size / 1024 / 1024 < 2

      if (!isImage) {
        ElMessage.error(t('profile.avatarOnlyImage'))
        return
      }
      if (!isValidSize) {
        ElMessage.error(t('profile.avatarMaxSize'))
        return
      }

      const reader = new window.FileReader()
      reader.onload = () => {
        const avatarUrl = typeof reader.result === 'string' ? reader.result : ''
        profile.value.avatarUrl = avatarUrl
        if (profile.value.id) {
          localStorage.setItem(getAvatarStorageKey(profile.value.id), avatarUrl)
        }
        syncUserCache()
        ElMessage.success(t('profile.avatarUpdated'))
      }
      reader.readAsDataURL(rawFile)
    }

    const handleChangePassword = async () => {
      if (!passwordFormRef.value) return
      try {
        await passwordFormRef.value.validate()
      } catch {
        return
      }

      savingPassword.value = true
      try {
        await authAPI.changePassword({
          oldPassword: passwordForm.oldPassword,
          newPassword: passwordForm.newPassword,
        })
        ElMessage.success(t('profile.passwordUpdated'))
        passwordForm.oldPassword = ''
        passwordForm.newPassword = ''
        passwordForm.confirmPassword = ''
        passwordFormRef.value?.clearValidate()
      } catch (error) {
        ElMessage.error(
          t('profile.passwordUpdateFailed', {
            message: getApiErrorMessage(error, t('auth.retry')),
          }),
        )
      } finally {
        savingPassword.value = false
      }
    }

    onMounted(() => {
      loadProfile()
      document.addEventListener('mousedown', handleDocumentClick)
    })

    onUnmounted(() => {
      document.removeEventListener('mousedown', handleDocumentClick)
    })

    return {
      loading,
      savingPassword,
      savingPreferences,
      isEditingEmail,
      emailInputRef,
      emailRowRef,
      profile,
      userInitials,
      passwordForm,
      passwordRules,
      passwordFormRef,
      accountForm,
      preferencesFormRef,
      preferencesForm,
      preferencesRules,
      languageOptions,
      themeOptions,
      getRoleLabel,
      getTenantLabel,
      t,
      loadProfile,
      startEmailEdit,
      exitEmailEdit,
      cancelEmailEdit,
      handleEmailBlur,
      handleAvatarChange,
      handleSaveAccountInfo,
      handleSavePreferences,
      handleChangePassword,
    }
  },
}
</script>

<style scoped>
/* Override Element Plus inputs for modern look */
:deep(.modern-input .el-input__wrapper),
:deep(.modern-select .el-input__wrapper) {
  border-radius: 12px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  padding: 8px 16px;
  transition: all 0.2s;
  background-color: #f8fafc;
}

:deep(.modern-input .el-input__wrapper.is-focus),
:deep(.modern-select .el-input__wrapper.is-focus) {
  box-shadow:
    0 0 0 1px #3b82f6 inset,
    0 0 0 4px rgba(59, 130, 246, 0.1);
  background-color: #ffffff;
}

html.dark :deep(.modern-input .el-input__wrapper),
html.dark :deep(.modern-select .el-input__wrapper) {
  background-color: #0f172a;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}

html.dark :deep(.modern-input .el-input__wrapper.is-focus),
html.dark :deep(.modern-select .el-input__wrapper.is-focus) {
  box-shadow:
    0 0 0 1px #3b82f6 inset,
    0 0 0 4px rgba(59, 130, 246, 0.2);
}

:deep(.el-form-item__label) {
  font-weight: 600;
  color: #475569;
  padding-bottom: 8px;
}

html.dark :deep(.el-form-item__label) {
  color: #cbd5e1;
}

/* Custom scrollbar for embedded mode */
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
  background-color: #475569;
}
</style>
