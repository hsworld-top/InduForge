<template>
  <div class="embedded-app">
    <iframe
      ref="iframeRef"
      :src="appUrl"
      frameborder="0"
      class="embedded-iframe"
      allow="cookies"
      sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-modals allow-top-navigation allow-downloads"
      @load="handleIframeLoad"
    >
      {{ $t('common.iframeNotSupported') }}
    </iframe>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { buildAppEntry } from '@/utils/appUrl'
import type { EmbeddedAppType, EmbeddedProjectContext } from '@/types/embedded'

const props = withDefaults(
  defineProps<{
    appType: EmbeddedAppType
    project: EmbeddedProjectContext
    tabKey?: string
  }>(),
  {
    tabKey: '',
  },
)

const emit = defineEmits<{
  (
    event: 'embedded-register',
    payload: {
      iframe: HTMLIFrameElement
      origin: string
      appType: EmbeddedAppType
      project: EmbeddedProjectContext
      tabKey: string
    },
  ): void
  (event: 'embedded-unregister', payload: { iframe: HTMLIFrameElement }): void
}>()
const iframeRef = ref<HTMLIFrameElement | null>(null)

const appEntry = computed(() => buildAppEntry(props.appType, props.project))
const appUrl = computed(() => appEntry.value.url)

const registerEmbeddedApp = async () => {
  await nextTick()

  const iframe = iframeRef.value
  if (!iframe) return

  emit('embedded-register', {
    iframe,
    origin: appEntry.value.origin || window.location.origin,
    appType: props.appType,
    project: props.project,
    tabKey: props.tabKey,
  })
}

/**
 * iframe src is known before the child app finishes loading. Register as soon as
 * Vue attaches the iframe ref so early bootstrap messages are not dropped; the
 * load event refreshes the same registry entry after navigations/reloads.
 */
const handleIframeLoad = () => {
  void registerEmbeddedApp()
}

watch(
  appEntry,
  () => {
    void registerEmbeddedApp()
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  if (!iframeRef.value) return

  // 标签页关闭或组件卸载时同步清理注册表，避免旧窗口继续命中宿主消息匹配。
  emit('embedded-unregister', {
    iframe: iframeRef.value,
  })
})
</script>

<style scoped>
.embedded-app {
  width: 100%;
  height: 100%;
  display: flex;
  overflow: hidden;
}

.embedded-iframe {
  flex: 1;
  width: 100%;
  height: 100%;
  border: none;
}
</style>
