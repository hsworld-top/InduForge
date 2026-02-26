<template>
  <div class="system-settings">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">系统设置</h1>
      <el-button @click="loadSystemConfig" :loading="loading">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <el-alert
      title="该页面仅用于 SYSTEM_ADMIN 的系统级配置，修改后会立即在当前浏览器生效。"
      type="info"
      :closable="false"
      class="mb-6"
    />

    <div class="grid grid-cols-1 xl:grid-cols-2 gap-6">
      <div class="panel">
        <div class="panel-title">系统信息</div>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="系统名称">
            {{ systemInfo.title || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="版本">
            {{ systemInfo.version || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="描述">
            {{ systemInfo.description || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="作者">
            {{ systemInfo.author || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="多租户模式">
            <el-tag :type="systemInfo.multiTenant ? 'success' : 'info'" size="small">
              {{ systemInfo.multiTenant ? '启用' : '关闭' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="活跃租户数">
            {{ systemInfo.activeTenantsCount ?? '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="配置更新时间">
            {{ formattedBuildTime }}
          </el-descriptions-item>
        </el-descriptions>
      </div>

      <div class="panel">
        <div class="panel-title">界面偏好</div>
        <el-form :model="settingsForm" label-width="110px">
          <el-form-item label="系统主题">
            <el-select v-model="settingsForm.theme" style="width: 220px">
              <el-option label="浅色" value="light" />
              <el-option label="深色" value="dark" />
            </el-select>
          </el-form-item>
          <el-form-item label="系统语言">
            <el-select v-model="settingsForm.language" style="width: 220px">
              <el-option label="中文" value="zh" />
              <el-option label="English" value="en" />
            </el-select>
          </el-form-item>
          <el-form-item label="日志默认页长">
            <el-select v-model="settingsForm.logPageSize" style="width: 220px">
              <el-option label="10 条" :value="10" />
              <el-option label="20 条" :value="20" />
              <el-option label="50 条" :value="50" />
              <el-option label="100 条" :value="100" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-space wrap>
              <el-button type="primary" @click="saveSettings" :loading="saving" :disabled="!isDirty">
                保存设置
              </el-button>
              <el-button @click="resetToLastSaved" :disabled="!isDirty">撤销修改</el-button>
              <el-button @click="resetToDefault">恢复默认</el-button>
            </el-space>
          </el-form-item>
          <el-form-item v-if="isDirty">
            <span class="text-sm text-orange-600 dark:text-orange-300">当前存在未保存修改</span>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, reactive, onMounted, computed, onUnmounted } from 'vue'
import dayjs from 'dayjs'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { onBeforeRouteLeave } from 'vue-router'
import { authAPI } from '@/api'
import { useAppStore } from '@/store'
import { Storage } from '@/utils/storage'

export default {
  name: 'SystemSettings',
  components: {
    Refresh,
  },
  setup() {
    const { locale } = useI18n()
    const appStore = useAppStore()
    const loading = ref(false)
    const saving = ref(false)
    const lastSavedSettings = ref({
      theme: appStore.theme || 'light',
      language: appStore.language || 'zh',
      logPageSize: Storage.get('system_log_page_size', 20),
    })

    const systemInfo = ref({
      title: '',
      version: '',
      description: '',
      author: '',
      buildTime: '',
      multiTenant: false,
      activeTenantsCount: 0,
    })

    const settingsForm = reactive({
      theme: lastSavedSettings.value.theme,
      language: lastSavedSettings.value.language,
      logPageSize: lastSavedSettings.value.logPageSize,
    })

    const isDirty = computed(
      () =>
        settingsForm.theme !== lastSavedSettings.value.theme ||
        settingsForm.language !== lastSavedSettings.value.language ||
        settingsForm.logPageSize !== lastSavedSettings.value.logPageSize
    )

    const formattedBuildTime = computed(() => {
      const raw = systemInfo.value.buildTime
      if (!raw) return '-'
      const parsed = dayjs(raw)
      return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm:ss') : raw
    })

    const loadSystemConfig = async () => {
      loading.value = true
      try {
        const response = await authAPI.getConfig()
        systemInfo.value = {
          title: response.data?.title || '',
          version: response.data?.version || '',
          description: response.data?.description || '',
          author: response.data?.author || '',
          buildTime: response.data?.buildTime || '',
          multiTenant: !!response.data?.multiTenant,
          activeTenantsCount: response.data?.activeTenantsCount ?? 0,
        }
      } catch (error) {
        ElMessage.error('加载系统设置失败：' + (error.response?.data?.message || error.message))
      } finally {
        loading.value = false
      }
    }

    const resetToLastSaved = () => {
      settingsForm.theme = lastSavedSettings.value.theme
      settingsForm.language = lastSavedSettings.value.language
      settingsForm.logPageSize = lastSavedSettings.value.logPageSize
    }

    const resetToDefault = () => {
      settingsForm.theme = 'light'
      settingsForm.language = 'zh'
      settingsForm.logPageSize = 20
    }

    const saveSettings = async () => {
      if (!isDirty.value) {
        ElMessage.info('当前没有需要保存的修改')
        return
      }
      saving.value = true
      try {
        appStore.setTheme(settingsForm.theme)
        appStore.setLanguage(settingsForm.language)
        locale.value = settingsForm.language
        Storage.set('system_log_page_size', settingsForm.logPageSize)
        lastSavedSettings.value = {
          theme: settingsForm.theme,
          language: settingsForm.language,
          logPageSize: settingsForm.logPageSize,
        }
        ElMessage.success('设置已保存')
        ElMessage.info('日志默认页长将在系统日志页刷新后生效')
      } finally {
        saving.value = false
      }
    }

    const handleBeforeUnload = (event) => {
      if (!isDirty.value) return
      event.preventDefault()
      event.returnValue = ''
    }

    onMounted(() => {
      loadSystemConfig()
      window.addEventListener('beforeunload', handleBeforeUnload)
    })

    onUnmounted(() => {
      window.removeEventListener('beforeunload', handleBeforeUnload)
    })

    onBeforeRouteLeave(() => {
      if (!isDirty.value) return true
      return window.confirm('当前设置尚未保存，确定要离开吗？')
    })

    return {
      loading,
      saving,
      isDirty,
      formattedBuildTime,
      systemInfo,
      settingsForm,
      loadSystemConfig,
      saveSettings,
      resetToLastSaved,
      resetToDefault,
    }
  },
}
</script>

<style scoped>
.system-settings {
  padding: 20px;
}

.panel {
  @apply bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6;
}

.panel-title {
  @apply text-lg font-semibold text-gray-900 dark:text-white mb-4;
}
</style>
