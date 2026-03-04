<template>
  <div class="settings-page">
    <el-card class="panel-card">
      <el-form label-width="130px">
        <el-form-item :label="t('init.theme')">
          <el-segmented v-model="theme" :options="themeOptions" />
        </el-form-item>
        <el-form-item :label="t('init.locale')">
          <el-segmented v-model="locale" :options="localeOptions" />
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { usePreferencesStore } from '@/store/preferencesStore'
import { useI18nText } from '@/composables/useI18nText'

const preferences = usePreferencesStore()
const { t } = useI18nText()

const theme = computed({
  get: () => preferences.theme,
  set: (value) => preferences.setTheme(value),
})

const locale = computed({
  get: () => preferences.locale,
  set: (value) => preferences.setLocale(value),
})

const themeOptions = computed(() => [
  { label: t('init.light'), value: 'light' },
  { label: t('init.dark'), value: 'dark' },
])
const localeOptions = computed(() => [
  { label: t('common.zh'), value: 'zh-CN' },
  { label: t('common.en'), value: 'en-US' },
])
</script>

<style scoped>
.settings-page {
  display: grid;
  gap: 16px;
}
</style>
