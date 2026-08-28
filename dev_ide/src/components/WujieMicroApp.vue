<template>
  <WujieVue
    class="wujie-micro-app"
    :name="instanceName"
    :url="appUrl"
    :props="contextProps"
    :alive="true"
    :sync="false"
    :after-mount="syncContext"
    :activated="syncContext"
    width="100%"
    height="100%"
  />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, shallowReactive, watch } from 'vue'
import WujieVue from 'wujie-vue3'
import { useAppStore, useAuthStore } from '@/store'
import { refreshSession } from '@/utils/request'
import { Storage } from '@/utils/storage'
import type { MicroAppType, MicroAppProjectContext } from '@/types/micro-app'
import type { WorkspaceCloseRequest, WorkspaceOpenRequest } from '@/types/workspace-tool'

const props = defineProps<{
  appType: MicroAppType
  project: MicroAppProjectContext
}>()

const emit = defineEmits<{
  stateChange: [payload: { instanceName: string; title?: string; dirty?: boolean }]
  openWorkspace: [request: WorkspaceOpenRequest]
  closeWorkspace: [request: WorkspaceCloseRequest]
}>()

const appStore = useAppStore()
const authStore = useAuthStore()

const instanceName = computed(() => `${props.appType}-${props.project.id}`)
const appUrl = computed(() => `/${props.appType}/`)
const contextEventName = computed(() => `micro-app:${instanceName.value}:context`)
const contextReadyEventName = computed(() => `micro-app:${instanceName.value}:context-ready`)

const handleAuthExpired = () => {
  authStore.clearAuthData()
  if (window.location.pathname !== '/login') window.location.assign('/login')
}

const buildContext = () => ({
  appType: props.appType,
  instanceName: instanceName.value,
  project: { ...props.project, id: String(props.project.id) },
  projectId: String(props.project.id),
  projectName: props.project.name ? String(props.project.name) : undefined,
  tenantId: props.project.tenantId ? String(props.project.tenantId) : Storage.getTenantId(),
  theme: appStore.theme === 'dark' ? 'dark' : 'light',
  locale: appStore.language === 'en' ? 'en' : 'zh',
  onRefreshAuth: refreshSession,
  onAuthExpired: handleAuthExpired,
  onStateChange: (payload: { title?: string; dirty?: boolean } = {}) => {
    emit('stateChange', { instanceName: instanceName.value, ...payload })
  },
  onOpenWorkspace: (request: WorkspaceOpenRequest) => emit('openWorkspace', request),
  onCloseWorkspace: (request: WorkspaceCloseRequest) => emit('closeWorkspace', request),
  onRegisterWindowMessageListener: (listener: (event: MessageEvent) => void) => {
    window.addEventListener('message', listener)
    return () => window.removeEventListener('message', listener)
  },
  onPostWindowMessage: (target: Window, data: unknown, targetOrigin: string) => {
    target.postMessage(data, targetOrigin)
  },
})

const contextProps = shallowReactive(buildContext())

const syncContext = () => {
  const nextContext = buildContext()
  Object.assign(contextProps, nextContext)
  WujieVue.bus.$emit(contextEventName.value, nextContext)
}

watch([() => appStore.theme, () => appStore.language], syncContext, { flush: 'post' })

onMounted(() => {
  WujieVue.bus.$on(contextReadyEventName.value, syncContext)
  syncContext()
})

onBeforeUnmount(() => {
  WujieVue.bus.$off(contextReadyEventName.value, syncContext)
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
