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

<script setup>
import { computed, onBeforeUnmount, ref } from 'vue'
import { buildAppEntry } from '@/utils/appUrl'

const props = defineProps({
  appType: {
    type: String,
    required: true,
    validator: (value) => ['datacenter', 'designer'].includes(value),
  },
  project: {
    type: Object,
    required: true,
  },
  tabKey: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['embedded-register', 'embedded-unregister'])
const iframeRef = ref(null)

const appEntry = computed(() => buildAppEntry(props.appType, props.project))
const appUrl = computed(() => appEntry.value.url)

/**
 * iframe 完成加载后，把宿主后续匹配所需的节点与上下文一起上报。
 * Dashboard 会据此登记 source/origin/appType/project 等信息，不能再从相对 src 猜 origin。
 */
const handleIframeLoad = () => {
  if (!iframeRef.value) return

  emit('embedded-register', {
    iframe: iframeRef.value,
    origin: appEntry.value.origin,
    appType: props.appType,
    project: props.project,
    tabKey: props.tabKey,
  })
}

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
