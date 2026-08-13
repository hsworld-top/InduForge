<template>
  <main class="workspace-micro-app">
    <WujieMicroApp v-if="microAppType" :app-type="microAppType" :project="project" />
  </main>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import WujieMicroApp from '@/components/WujieMicroApp.vue'
import { Storage } from '@/utils/storage'
import type { MicroAppType } from '@/types/micro-app'

const route = useRoute()

const microAppType = computed<MicroAppType | null>(() => {
  const value = String(route.params.appType || '')
  return value === 'designer' || value === 'datacenter' ? value : null
})

const project = computed(() => ({
  id: String(route.params.projectId || ''),
  tenantId: Storage.getTenantId() ?? undefined,
}))
</script>

<style scoped>
.workspace-micro-app {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}
</style>
