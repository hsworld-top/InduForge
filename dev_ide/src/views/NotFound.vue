<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
    <div class="text-center">
      <div class="text-6xl font-bold text-gray-300 dark:text-gray-600 mb-4">404</div>
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white mb-4">
        {{ t('notFound.title') }}
      </h1>
      <p class="text-gray-600 dark:text-gray-400 mb-8">
        {{ t('notFound.description') }}
      </p>
      <router-link to="/dashboard" class="btn-primary inline-block">
        {{ t('notFound.backDashboard') }}
      </router-link>
    </div>
  </div>
</template>

<script lang="ts">
import { useI18n } from 'vue-i18n'

export default {
  name: 'NotFound',
  setup() {
    const { t } = useI18n()
    return { t }
  },
}
</script>
