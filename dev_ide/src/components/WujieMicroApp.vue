<template>
  <WujieVue
    class="wujie-micro-app"
    :name="instanceName"
    :url="appUrl"
    :props="initialProps"
    :alive="true"
    :sync="false"
    width="100%"
    height="100%"
  />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue'
import WujieVue from 'wujie-vue3'
import { useAppStore } from '@/store'
import { Storage } from '@/utils/storage'
import type { MicroAppType, MicroAppProjectContext } from '@/types/micro-app'
import type { WorkspaceOpenRequest } from '@/types/workspace-tool'

const props = defineProps<{
  appType: MicroAppType
  project: MicroAppProjectContext
}>()

const emit = defineEmits<{
  stateChange: [payload: { instanceName: string; title?: string; dirty?: boolean }]
  openWorkspace: [request: WorkspaceOpenRequest]
}>()

const appStore = useAppStore()

const instanceName = computed(() => `${props.appType}-${props.project.id}`)
const appUrl = computed(() => `/${props.appType}/`)
const contextEventName = computed(() => `micro-app:${instanceName.value}:context`)

const buildContext = () => ({
  appType: props.appType,
  instanceName: instanceName.value,
  project: { ...props.project, id: String(props.project.id) },
  projectId: String(props.project.id),
  projectName: props.project.name ? String(props.project.name) : undefined,
  tenantId: props.project.tenantId ? String(props.project.tenantId) : Storage.getTenantId(),
  theme: appStore.theme === 'dark' ? 'dark' : 'light',
  locale: appStore.language === 'en' ? 'en' : 'zh',
  onStateChange: (payload: { title?: string; dirty?: boolean } = {}) => {
    emit('stateChange', { instanceName: instanceName.value, ...payload })
  },
  onOpenWorkspace: (request: WorkspaceOpenRequest) => emit('openWorkspace', request),
})

const initialProps = computed(buildContext)

const syncContext = () => {
  WujieVue.bus.$emit(contextEventName.value, buildContext())
}

watch(() => [appStore.theme, appStore.language], syncContext)

onBeforeUnmount(() => {
  WujieVue.bus.$emit(contextEventName.value, null)
})
</script>

<style scoped>
.wujie-micro-app {
  display: block;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.wujie-micro-app :deep(wujie-app),
.wujie-micro-app :deep(iframe) {
  display: block;
  width: 100%;
  height: 100%;
  min-height: 0;
}
</style>
