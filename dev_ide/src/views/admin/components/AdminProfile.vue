<template>
  <div class="p-2 w-full font-sans antialiased text-slate-700 dark:text-slate-300">
    <div class="flex flex-col gap-6">
      <!-- 头部资料区 (极简风) -->
      <div
        class="flex items-center gap-5 p-5 bg-slate-50 dark:bg-slate-800/40 rounded-2xl border border-slate-200/60 dark:border-slate-700/60"
      >
        <el-avatar
          :size="64"
          :src="profile.avatarUrl"
          class="border-2 border-white dark:border-slate-700 shadow-sm bg-gradient-to-br from-slate-200 to-slate-100 dark:from-slate-700 dark:to-slate-800 text-slate-600 dark:text-slate-300 font-bold text-xl"
        >
          {{ userInitials }}
        </el-avatar>
        <div class="flex-1">
          <h2 class="text-xl font-bold text-slate-900 dark:text-white">
            {{ profile.username || '-' }}
          </h2>
          <div class="flex items-center gap-2 mt-1.5">
            <span
              class="px-2 py-0.5 text-xs font-semibold text-slate-700 bg-slate-200/70 dark:bg-slate-700 dark:text-slate-300 rounded-md"
              >{{ getRoleLabel(profile.role) }}</span
            >
            <span class="text-xs text-slate-500 dark:text-slate-400">{{
              getTenantLabel(profile.tenant)
            }}</span>
          </div>
        </div>
        <div class="flex gap-2">
          <el-upload
            :auto-upload="false"
            :show-file-list="false"
            :on-change="handleAvatarChange"
            accept="image/png,image/jpeg,image/webp"
          >
            <button
              type="button"
              class="px-3 py-1.5 text-xs font-medium text-slate-600 bg-white border border-slate-200 rounded-lg hover:bg-slate-50 hover:text-slate-900 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-600 dark:hover:bg-slate-700 transition-colors shadow-sm"
            >
              {{ t('profile.uploadAvatar') }}
            </button>
          </el-upload>
        </div>
      </div>

      <!-- 账号信息区 -->
      <div class="space-y-4">
        <h3 class="text-sm font-bold text-slate-900 dark:text-white px-1">账号信息</h3>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div class="space-y-1.5">
            <label class="text-xs font-semibold text-slate-500 dark:text-slate-400 px-1">{{
              t('profile.username')
            }}</label>
            <div
              class="w-full px-4 py-2.5 bg-slate-100/50 dark:bg-slate-800/30 border border-slate-200/60 dark:border-slate-700/50 rounded-xl text-sm text-slate-600 dark:text-slate-400 cursor-not-allowed"
            >
              {{ profile.username || '-' }}
            </div>
          </div>
          <div class="space-y-1.5">
            <label class="text-xs font-semibold text-slate-500 dark:text-slate-400 px-1">{{
              t('profile.role')
            }}</label>
            <div
              class="w-full px-4 py-2.5 bg-slate-100/50 dark:bg-slate-800/30 border border-slate-200/60 dark:border-slate-700/50 rounded-xl text-sm text-slate-600 dark:text-slate-400 cursor-not-allowed"
            >
              {{ getRoleLabel(profile.role) }}
            </div>
          </div>
          <div class="space-y-1.5 sm:col-span-2">
            <label class="text-xs font-semibold text-slate-500 dark:text-slate-400 px-1">{{
              t('profile.email')
            }}</label>
            <div class="flex items-center gap-2">
              <input
                v-model="accountForm.email"
                type="email"
                class="flex-1 px-4 py-2.5 bg-white dark:bg-slate-900/50 border border-slate-300 dark:border-slate-600 rounded-xl text-sm focus:bg-white focus:border-slate-500 focus:ring-4 focus:ring-slate-500/10 transition-all outline-none text-slate-800 dark:text-slate-200"
                :placeholder="t('profile.emailPlaceholder')"
              />
              <button
                type="button"
                class="px-4 py-2.5 text-sm font-medium text-white bg-slate-800 rounded-xl hover:bg-slate-900 dark:bg-slate-700 dark:hover:bg-slate-600 transition-colors shadow-sm disabled:opacity-50"
                @click="handleSaveEmail"
                :disabled="accountForm.email === profile.email"
              >
                保存邮箱
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="h-px bg-slate-200/60 dark:bg-slate-700/60 my-2"></div>

      <!-- 修改密码区 -->
      <div class="space-y-4">
        <h3 class="text-sm font-bold text-slate-900 dark:text-white px-1">
          {{ t('profile.changePassword') }}
        </h3>
        <el-form
          ref="passwordFormRef"
          :model="passwordForm"
          :rules="passwordRules"
          label-position="top"
        >
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <el-form-item prop="oldPassword" class="mb-0">
              <el-input
                v-model="passwordForm.oldPassword"
                type="password"
                show-password
                size="large"
                :placeholder="t('profile.oldPasswordPlaceholder')"
                class="admin-input"
              />
            </el-form-item>
            <el-form-item prop="newPassword" class="mb-0">
              <el-input
                v-model="passwordForm.newPassword"
                type="password"
                show-password
                size="large"
                :placeholder="t('profile.newPasswordPlaceholder')"
                class="admin-input"
              />
            </el-form-item>
            <el-form-item prop="confirmPassword" class="mb-0">
              <el-input
                v-model="passwordForm.confirmPassword"
                type="password"
                show-password
                size="large"
                :placeholder="t('profile.confirmPasswordPlaceholder')"
                class="admin-input"
              />
            </el-form-item>
          </div>
          <div class="mt-4 flex justify-end">
            <button
              type="button"
              class="px-5 py-2.5 text-sm font-medium text-white bg-slate-800 rounded-xl hover:bg-slate-900 dark:bg-slate-700 dark:hover:bg-slate-600 transition-all shadow-sm flex items-center gap-2 disabled:opacity-70"
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
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { authAPI, userAPI } from '@/api'
import { useAuthStore } from '@/store'
import { Storage } from '@/utils/storage'
import { RoleEnum } from '@/enums'

const AVATAR_KEY_PREFIX = 'profile_avatar_'

export default {
  name: 'AdminProfile',
  setup() {
    const { t } = useI18n()
    const authStore = useAuthStore()

    const loading = ref(false)
    const savingPassword = ref(false)
    const passwordFormRef = ref<any>(null)

    const profile = ref({
      id: '',
      username: '',
      email: '',
      role: '',
      tenant: null,
      avatarUrl: '',
    })

    const userInitials = computed(() => {
      const username = profile.value.username || 'A'
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

    const getAvatarStorageKey = (userId) => `${AVATAR_KEY_PREFIX}${userId || 'anonymous'}`

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
      } finally {
        loading.value = false
      }
    }

    const handleSaveEmail = async () => {
      const userId = profile.value.id || authStore.userInfo?.id
      if (!userId) {
        ElMessage.error(t('profile.noUserInfo'))
        return
      }

      try {
        const nextEmail = (accountForm.email || '').trim()

        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
        if (nextEmail && !emailRegex.test(nextEmail)) {
          ElMessage.warning(t('profile.invalidEmail'))
          return
        }

        if (nextEmail !== (profile.value.email || '')) {
          await userAPI.updateUser(userId, { email: nextEmail })
          profile.value.email = nextEmail
          syncUserCache()
          ElMessage.success(t('profile.accountUpdated'))
        }
      } catch (error) {
        ElMessage.error(
          t('profile.accountUpdateFailed', {
            message: error.response?.data?.message || error.message,
          }),
        )
      }
    }

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
          }),
        )
      } finally {
        savingPassword.value = false
      }
    }

    onMounted(() => {
      loadProfile()
    })

    return {
      loading,
      savingPassword,
      profile,
      userInitials,
      passwordForm,
      passwordRules,
      passwordFormRef,
      accountForm,
      getRoleLabel,
      getTenantLabel,
      t,
      handleAvatarChange,
      handleSaveEmail,
      handleChangePassword,
    }
  },
}
</script>

<style scoped>
:deep(.admin-input .el-input__wrapper) {
  border-radius: 0.75rem; /* xl */
  box-shadow: none;
  border: 1px solid #cbd5e1;
  padding: 8px 16px;
  background-color: #fff;
  transition: all 0.2s;
}

html.dark :deep(.admin-input .el-input__wrapper) {
  background-color: rgba(15, 23, 42, 0.5);
  border-color: #475569;
}

:deep(.admin-input .el-input__wrapper.is-focus) {
  border-color: #64748b;
  box-shadow: 0 0 0 4px rgba(100, 116, 139, 0.1);
}

html.dark :deep(.admin-input .el-input__wrapper.is-focus) {
  border-color: #94a3b8;
  box-shadow: 0 0 0 4px rgba(148, 163, 184, 0.15);
}
</style>
