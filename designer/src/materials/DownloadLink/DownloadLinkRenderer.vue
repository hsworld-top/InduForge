<script setup lang="ts">
import { computed } from 'vue'
import {
  buildDownloadLinkStyle,
  resolveDownloadLinkConfig,
  shouldTriggerByEvent,
} from './download-link-utils'

interface RendererProps {
  node?: Record<string, unknown> | null
  resolvedProps?: Record<string, unknown> | null
}

const props = defineProps<RendererProps>()

const config = computed(() => resolveDownloadLinkConfig(props.resolvedProps))

const renderStyle = computed(() => buildDownloadLinkStyle(config.value))

const linkHref = computed(() => {
  if (config.value.disabled || !config.value.href) return 'javascript:void(0)'
  return config.value.href
})

const downloadAttr = computed<string | undefined>(() => {
  if (config.value.actionMode !== 'download') return undefined
  return config.value.downloadFileName || ''
})

function triggerAction(): void {
  const current = config.value
  if (current.disabled || current.actionMode === 'none' || !current.href) return

  const anchor = document.createElement('a')
  anchor.href = current.href
  anchor.target = current.target
  if (current.actionMode === 'download') {
    anchor.download = current.downloadFileName || ''
  }
  if (current.target === '_blank') {
    anchor.rel = 'noopener noreferrer'
  }
  anchor.style.display = 'none'
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
}

function handleClick(event: MouseEvent): void {
  event.preventDefault()
  event.stopPropagation()
  if (!shouldTriggerByEvent(config.value.triggerMode, 'click')) return
  triggerAction()
}

function handleDoubleClick(event: MouseEvent): void {
  event.preventDefault()
  event.stopPropagation()
  if (!shouldTriggerByEvent(config.value.triggerMode, 'dblclick')) return
  triggerAction()
}
</script>

<template>
  <span class="designer-download-link-renderer">
    <a
      class="download-link-trigger"
      :class="[
        `is-${config.displayMode}`,
        {
          'is-disabled': config.disabled,
        },
      ]"
      :href="linkHref"
      :target="config.target"
      :download="downloadAttr"
      :style="renderStyle"
      @click="handleClick"
      @dblclick="handleDoubleClick"
    >
      {{ config.text }}
    </a>
  </span>
</template>

<style scoped>
.designer-download-link-renderer {
  display: block;
  width: 100%;
  height: 100%;
}

.download-link-trigger {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.download-link-trigger.is-disabled {
  pointer-events: none;
}
</style>
