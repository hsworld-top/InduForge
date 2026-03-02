<template>
  <div class="settings-page">
    <el-card class="panel-card">
      <el-form label-width="130px">
        <el-form-item label="主题">
          <el-segmented v-model="theme" :options="themeOptions" />
        </el-form-item>
        <el-form-item label="国际化">
          <el-segmented v-model="locale" :options="localeOptions" />
        </el-form-item>
        <el-form-item label="开机自启动">
          <el-switch v-model="autostart" :loading="savingAutostart" />
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { nodeApi } from '@/api/nodeApi'
import { usePreferencesStore } from '@/store/preferencesStore'

const preferences = usePreferencesStore()
const autostart = ref(false)
const savingAutostart = ref(false)
const autostartReady = ref(false)

const theme = ref(preferences.theme)
const locale = ref(preferences.locale)

const themeOptions = [
  { label: '浅色', value: 'light' },
  { label: '深色', value: 'dark' },
]
const localeOptions = [
  { label: '中文', value: 'zh-CN' },
  { label: 'English', value: 'en-US' },
]

watch(theme, (value) => preferences.setTheme(value))
watch(locale, (value) => preferences.setLocale(value))
watch(autostart, async (value) => {
  if (!autostartReady.value) return
  savingAutostart.value = true
  try {
    await nodeApi.saveBootstrapAutostart(value)
  } catch (error) {
    ElMessage.error(error?.message || '更新开机自启动失败')
  } finally {
    savingAutostart.value = false
  }
})

const loadCurrent = async () => {
  try {
    const bootstrap = await nodeApi.getBootstrapStatus()
    autostart.value = !!bootstrap?.autoStart
    autostartReady.value = true
  } catch (error) {
    console.error('加载基础配置失败:', error)
  }
}

onMounted(() => {
  loadCurrent()
})
</script>

<style scoped>
.settings-page {
  display: grid;
  gap: 16px;
}
</style>
