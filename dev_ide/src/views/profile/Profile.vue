<template>
  <div class="profile">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">个人资料</h1>
      <el-button @click="loadProfile" :loading="loading">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <div class="grid grid-cols-1 xl:grid-cols-2 gap-6">
      <div class="panel">
        <div class="panel-title">账户信息</div>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="用户名">
            {{ profile.username || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="用户ID">
            {{ profile.id || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="角色">
            {{ getRoleLabel(profile.role) }}
          </el-descriptions-item>
          <el-descriptions-item label="租户">
            {{ profile.tenant?.name || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="租户ID">
            {{ profile.tenant?.id || '-' }}
          </el-descriptions-item>
        </el-descriptions>
      </div>

      <div class="panel">
        <div class="panel-title">修改密码</div>
        <el-form ref="passwordFormRef" :model="passwordForm" :rules="passwordRules" label-width="96px">
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
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { authAPI, userAPI } from '@/api'
import { useAuthStore } from '@/store'
import { Storage } from '@/utils/storage'
import { ENUM_LABELS } from '@/enums'

export default {
  name: 'Profile',
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

    const getRoleLabel = (role) => {
      return ENUM_LABELS[role] || role || '-'
    }

    const loadProfile = async () => {
      loading.value = true
      try {
        const response = await authAPI.getCurrentUser()
        const user = response.data?.user || response.data || {}
        profile.value = {
          id: user.id || '',
          username: user.username || '',
          role: user.role || '',
          tenant: user.tenant || null,
        }

        if (profile.value.id) {
          authStore.userInfo = {
            ...authStore.userInfo,
            id: profile.value.id,
            username: profile.value.username,
            role: profile.value.role,
            tenantId: profile.value.tenant?.id || authStore.userInfo?.tenantId,
            tenant: profile.value.tenant,
          }
          Storage.setUserInfo(authStore.userInfo)
        }
      } catch {
        const localUser = Storage.getUserInfo()
        profile.value = {
          id: localUser?.id || '',
          username: localUser?.username || '',
          role: localUser?.role || '',
          tenant: localUser?.tenant || null,
        }
        ElMessage.warning('获取个人资料失败，已显示本地缓存信息')
      } finally {
        loading.value = false
      }
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
        ElMessage.error('密码修改失败：' + (error.response?.data?.message || error.message))
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
      passwordForm,
      passwordRules,
      passwordFormRef,
      getRoleLabel,
      loadProfile,
      handleChangePassword,
    }
  },
}
</script>

<style scoped>
.profile {
  padding: 20px;
}

.panel {
  @apply bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6;
}

.panel-title {
  @apply text-lg font-semibold text-gray-900 dark:text-white mb-4;
}
</style>
