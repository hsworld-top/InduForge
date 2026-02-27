<template>
  <div class="profile" :class="{ 'profile-embedded': embedded }">
    <div class="profile-header">
      <div class="header-main">
        <el-avatar :size="72" :src="profile.avatarUrl" class="avatar">
          {{ userInitials }}
        </el-avatar>
        <div class="header-meta">
          <h2 class="name">{{ profile.username || '-' }}</h2>
          <p class="role">{{ getRoleLabel(profile.role) }}</p>
          <p class="tenant">{{ getTenantLabel(profile.tenant) }}</p>
        </div>
      </div>
      <div class="header-actions">
        <el-upload
          :auto-upload="false"
          :show-file-list="false"
          :on-change="handleAvatarChange"
          accept="image/png,image/jpeg,image/webp"
        >
          <el-button>{{ t('profile.uploadAvatar') }}</el-button>
        </el-upload>
        <el-button @click="loadProfile" :loading="loading">
          <el-icon><Refresh /></el-icon>
          {{ t('common.refresh') }}
        </el-button>
      </div>
    </div>

    <div class="grid grid-cols-1 xl:grid-cols-2 gap-4">
      <div class="panel">
        <div class="panel-title">{{ t('profile.accountInfo') }}</div>
        <div class="meta-list">
          <div class="meta-item">
            <span class="label">{{ t('profile.username') }}</span>
            <span class="value">{{ profile.username || '-' }}</span>
          </div>
          <div class="meta-item">
            <span class="label">{{ t('profile.role') }}</span>
            <span class="value">{{ getRoleLabel(profile.role) }}</span>
          </div>
          <div class="meta-item email-row" @click="startEmailEdit">
            <span class="label">{{ t('profile.email') }}</span>
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
              <span v-else class="value">{{ profile.email || '-' }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="panel">
        <div class="panel-title">{{ t('profile.preferences') }}</div>
        <el-form ref="preferencesFormRef" :model="preferencesForm" :rules="preferencesRules" label-width="92px">
          <el-form-item :label="t('profile.language')" prop="language">
            <el-select v-model="preferencesForm.language" style="width: 100%">
              <el-option
                v-for="lang in languageOptions"
                :key="lang.value"
                :label="lang.label"
                :value="lang.value"
              />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('profile.theme')" prop="theme">
            <el-select v-model="preferencesForm.theme" style="width: 100%">
              <el-option
                v-for="theme in themeOptions"
                :key="theme.value"
                :label="theme.label"
                :value="theme.value"
              />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="savingPreferences" @click="handleSavePreferences">
              {{ t('profile.savePreferences') }}
            </el-button>
          </el-form-item>
        </el-form>
      </div>

      <div class="panel">
        <div class="panel-title">{{ t('profile.changePassword') }}</div>
        <el-form ref="passwordFormRef" :model="passwordForm" :rules="passwordRules" label-width="92px">
          <el-form-item :label="t('profile.oldPassword')" prop="oldPassword">
            <el-input
              v-model="passwordForm.oldPassword"
              type="password"
              show-password
              :placeholder="t('profile.oldPasswordPlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="t('profile.newPassword')" prop="newPassword">
            <el-input
              v-model="passwordForm.newPassword"
              type="password"
              show-password
              :placeholder="t('profile.newPasswordPlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="t('profile.confirmPassword')" prop="confirmPassword">
            <el-input
              v-model="passwordForm.confirmPassword"
              type="password"
              show-password
              :placeholder="t('profile.confirmPasswordPlaceholder')"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="savingPassword" @click="handleChangePassword">
              {{ t('profile.savePassword') }}
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { authAPI, userAPI } from '@/api'
import { useAuthStore, useAppStore } from '@/store'
import { Storage } from '@/utils/storage'
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
      oldPassword: [{ required: true, message: t('profile.requireOldPassword'), trigger: 'blur' }],
      newPassword: [
        { required: true, message: t('profile.requireNewPassword'), trigger: 'blur' },
        { min: 6, message: t('profile.passwordMinLength'), trigger: 'blur' },
      ],
      confirmPassword: [
        { required: true, message: t('profile.requireConfirmPassword'), trigger: 'blur' },
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
            message: error.response?.data?.message || error.message,
          })
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
            message: error.response?.data?.message || error.message,
          })
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
            message: error.response?.data?.message || error.message,
          })
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
.profile {
  padding: 20px;
}

.profile-embedded {
  padding: 4px;
}

.profile-header {
  @apply flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4 mb-5 p-5 rounded-xl border border-gray-200 dark:border-gray-700 bg-gradient-to-r from-teal-50 to-cyan-50 dark:from-gray-900 dark:to-gray-800;
}

.header-main {
  @apply flex items-center gap-4;
}

.avatar {
  border: 2px solid rgba(20, 184, 166, 0.3);
}

.header-meta .name {
  @apply text-xl font-semibold text-gray-900 dark:text-white;
}

.header-meta .role {
  @apply text-sm text-teal-700 dark:text-teal-300;
}

.header-meta .tenant {
  @apply text-xs text-gray-500 dark:text-gray-400 mt-1;
}

.header-actions {
  @apply flex flex-wrap items-center gap-2;
}

.panel {
  @apply bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-5;
}

.panel-title {
  @apply text-base font-semibold text-gray-900 dark:text-white mb-4;
}

.meta-list {
  @apply space-y-3;
}

.meta-item {
  @apply flex items-center justify-between py-2 border-b border-dashed border-gray-200 dark:border-gray-700;
}

.email-row {
  cursor: pointer;
}

.meta-item:last-child {
  border-bottom: none;
}

.label {
  @apply text-sm text-gray-500 dark:text-gray-400;
}

.value {
  @apply text-sm text-gray-900 dark:text-gray-100 font-medium text-right break-all;
}
</style>
