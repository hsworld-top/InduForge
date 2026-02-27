<template>
  <div class="system-settings">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('systemSettings.title') }}</h1>
      <el-button @click="loadSystemConfig" :loading="loading">
        <el-icon><Refresh /></el-icon>
        {{ t('systemSettings.refresh') }}
      </el-button>
    </div>

    <el-alert
      :title="t('systemSettings.infoAlert')"
      type="info"
      :closable="false"
      class="mb-6"
    />

    <div class="grid grid-cols-1 xl:grid-cols-2 gap-6">
      <div class="panel">
        <div class="panel-title">{{ t('systemSettings.systemInfo') }}</div>
        <el-descriptions :column="1" border>
          <el-descriptions-item :label="t('systemSettings.systemName')">
            {{ systemInfo.title || '-' }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('systemSettings.version')">
            {{ systemInfo.version || '-' }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('systemSettings.description')">
            {{ systemInfo.description || '-' }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('systemSettings.author')">
            {{ systemInfo.author || '-' }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('systemSettings.multiTenantMode')">
            <el-tag :type="systemInfo.multiTenant ? 'success' : 'info'" size="small">
              {{ systemInfo.multiTenant ? t('systemSettings.enabled') : t('systemSettings.disabled') }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="t('systemSettings.activeTenantsCount')">
            {{ systemInfo.activeTenantsCount ?? '-' }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('systemSettings.configUpdatedAt')">
            {{ formattedBuildTime }}
          </el-descriptions-item>
        </el-descriptions>
      </div>

      <div class="panel">
        <div class="panel-title">{{ t('systemSettings.uiPreference') }}</div>
        <el-form :model="settingsForm" label-width="110px">
          <el-form-item :label="t('systemSettings.theme')">
            <el-select v-model="settingsForm.theme" style="width: 220px">
              <el-option :label="t('systemSettings.light')" value="light" />
              <el-option :label="t('systemSettings.dark')" value="dark" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('systemSettings.language')">
            <el-select v-model="settingsForm.language" style="width: 220px">
              <el-option :label="t('system.languageZh')" value="zh" />
              <el-option :label="t('system.languageEn')" value="en" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('systemSettings.logPageSize')">
            <el-select v-model="settingsForm.logPageSize" style="width: 220px">
              <el-option
                v-for="size in systemLogPageSizeOptions"
                :key="size"
                :label="`${size} ${t('systemSettings.itemsUnit')}`"
                :value="size"
              />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-space wrap>
              <el-button type="primary" @click="saveSettings" :loading="saving" :disabled="!isDirty">
                {{ t('systemSettings.saveSettings') }}
              </el-button>
              <el-button @click="resetToLastSaved" :disabled="!isDirty">{{ t('systemSettings.undoChanges') }}</el-button>
              <el-button @click="resetToDefault">{{ t('systemSettings.resetDefault') }}</el-button>
            </el-space>
          </el-form-item>
          <el-form-item v-if="isDirty">
            <span class="text-sm text-orange-600 dark:text-orange-300">{{ t('systemSettings.unsavedHint') }}</span>
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
import { SYSTEM_LOG_PAGE_SIZE_OPTIONS } from '@/constants'

export default {
  name: 'SystemSettings',
  components: {
    Refresh,
  },
  setup() {
    const { locale, t } = useI18n()
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
        ElMessage.error(
          t('systemSettings.loadFailed', {
            message: error.response?.data?.message || error.message,
          })
        )
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
        ElMessage.info(t('systemSettings.noChanges'))
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
        ElMessage.success(t('systemSettings.saved'))
        ElMessage.info(t('systemSettings.logPageSizeTip'))
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
      return window.confirm(t('systemSettings.leaveConfirm'))
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
      t,
      systemLogPageSizeOptions: SYSTEM_LOG_PAGE_SIZE_OPTIONS,
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
