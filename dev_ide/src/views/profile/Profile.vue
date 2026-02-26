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
          <p class="tenant">{{ profile.tenant?.name || '未绑定租户' }}</p>
        </div>
      </div>
      <div class="header-actions">
        <el-upload
          :auto-upload="false"
          :show-file-list="false"
          :on-change="handleAvatarChange"
          accept="image/png,image/jpeg,image/webp"
        >
          <el-button>上传头像</el-button>
        </el-upload>
        <el-button @click="loadProfile" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <div class="grid grid-cols-1 xl:grid-cols-2 gap-4">
      <div class="panel">
        <div class="panel-title">账户信息</div>
        <div class="meta-list">
          <div class="meta-item">
            <span class="label">用户ID</span>
            <span class="value">{{ profile.id || '-' }}</span>
          </div>
          <div class="meta-item">
            <span class="label">用户名</span>
            <span class="value">{{ profile.username || '-' }}</span>
          </div>
          <div class="meta-item">
            <span class="label">角色</span>
            <span class="value">{{ getRoleLabel(profile.role) }}</span>
          </div>
          <div class="meta-item">
            <span class="label">租户ID</span>
            <span class="value">{{ profile.tenant?.id || '-' }}</span>
          </div>
        </div>
      </div>

      <div class="panel">
        <div class="panel-title">修改密码</div>
        <el-form ref="passwordFormRef" :model="passwordForm" :rules="passwordRules" label-width="92px">
          <el-form-item label="新密码" prop="newPassword">
            <el-input
              v-model="passwordForm.newPassword"
              type="password"
              show-password
              placeholder="请输入新密码（至少6位）"
            />
          </el-form-item>
          <el-form-item label="确认密码" prop="confirmPassword">
            <el-input
              v-model="passwordForm.confirmPassword"
              type="password"
              show-password
              placeholder="请再次输入新密码"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="savingPassword" @click="handleChangePassword">
              保存新密码
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { authAPI, userAPI } from '@/api'
import { useAuthStore } from '@/store'
import { Storage } from '@/utils/storage'
import { ENUM_LABELS } from '@/enums'

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
    const authStore = useAuthStore()
    const loading = ref(false)
    const savingPassword = ref(false)
    const passwordFormRef = ref(null)

    const profile = ref({
      id: '',
      username: '',
      role: '',
      tenant: null,
      avatarUrl: '',
    })

    const userInitials = computed(() => {
      const username = profile.value.username || 'U'
      return username.charAt(0).toUpperCase()
    })

    const passwordForm = reactive({
      newPassword: '',
      confirmPassword: '',
    })

    const passwordRules = {
      newPassword: [
        { required: true, message: '请输入新密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于 6 个字符', trigger: 'blur' },
      ],
      confirmPassword: [
        { required: true, message: '请再次输入新密码', trigger: 'blur' },
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
        role: profile.value.role,
        tenantId: profile.value.tenant?.id || authStore.userInfo?.tenantId,
        tenant: profile.value.tenant,
        avatarUrl: profile.value.avatarUrl || '',
      }
      Storage.setUserInfo(authStore.userInfo)
    }

    const getRoleLabel = (role) => ENUM_LABELS[role] || role || '-'

    const loadProfile = async () => {
      loading.value = true
      try {
        const response = await authAPI.getCurrentUser()
        const user = response.data?.user || response.data || {}
        const avatarFromCache = localStorage.getItem(getAvatarStorageKey(user.id))
        profile.value = {
          id: user.id || '',
          username: user.username || '',
          role: user.role || '',
          tenant: user.tenant || null,
          avatarUrl: user.avatarUrl || avatarFromCache || authStore.userInfo?.avatarUrl || '',
        }
        syncUserCache()
      } catch {
        const localUser = Storage.getUserInfo()
        const avatarFromCache = localStorage.getItem(getAvatarStorageKey(localUser?.id))
        profile.value = {
          id: localUser?.id || '',
          username: localUser?.username || '',
          role: localUser?.role || '',
          tenant: localUser?.tenant || null,
          avatarUrl: localUser?.avatarUrl || avatarFromCache || '',
        }
        ElMessage.warning('获取个人资料失败，已显示本地缓存信息')
      } finally {
        loading.value = false
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
        ElMessage.error('仅支持图片文件')
        return
      }
      if (!isValidSize) {
        ElMessage.error('头像大小不能超过 2MB')
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
        ElMessage.success('头像已更新')
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

      const userId = profile.value.id || authStore.userInfo?.id
      if (!userId) {
        ElMessage.error('未获取到当前用户信息，请刷新后重试')
        return
      }

      savingPassword.value = true
      try {
        await userAPI.updatePassword(userId, passwordForm.newPassword)
        ElMessage.success('密码修改成功')
        passwordForm.newPassword = ''
        passwordForm.confirmPassword = ''
        passwordFormRef.value?.clearValidate()
      } catch (error) {
        ElMessage.error(`密码修改失败：${error.response?.data?.message || error.message}`)
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
      getRoleLabel,
      loadProfile,
      handleAvatarChange,
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
