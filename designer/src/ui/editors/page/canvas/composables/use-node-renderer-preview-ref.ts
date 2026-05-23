import type { ComputedRef, Ref } from 'vue'
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { getPreviewRuntime } from '@/ui/editors/page/preview/previewRuntime'
import { NODE_RENDERER_MAX_REGISTER_ATTEMPTS } from '../node-renderer-constants'

interface PreviewRefPage {
  id?: string
  name?: string
}

/**
 * 预览运行时：按页面 id/name 注册组件 ref，供脚本调用
 */
export function useNodeRendererPreviewRef(options: {
  readonly: ComputedRef<boolean>
  node: ComputedRef<{ label?: string } | null>
  currentPage: Ref<PreviewRefPage | null | undefined>
  buildRefInfo: () => unknown
}) {
  const { readonly, node, currentPage, buildRefInfo } = options
  const previewPageId = computed(() => currentPage.value?.name || currentPage.value?.id || '')

  let registerTimer: ReturnType<typeof setTimeout> | null = null
  let registerAttempts = 0
  const maxRegisterAttempts = NODE_RENDERER_MAX_REGISTER_ATTEMPTS

  function tryRegisterPreviewRef(pageIdValue: string): boolean {
    if (!readonly.value || !node.value?.label) return false
    const runtime = getPreviewRuntime()
    if (!runtime?.registerComponentRef) return false
    if (!pageIdValue) return false
    const refInfo = buildRefInfo()
    if (!refInfo) return false
    runtime.registerComponentRef(pageIdValue, node.value.label, refInfo)
    const pageId = currentPage.value?.id
    const pageName = currentPage.value?.name
    if (pageId && pageId !== pageIdValue) {
      runtime.registerComponentRef(pageId, node.value.label, refInfo)
    }
    if (pageName && pageName !== pageIdValue) {
      runtime.registerComponentRef(pageName, node.value.label, refInfo)
    }
    return true
  }

  function scheduleRegisterPreviewRef(pageIdValue: string) {
    if (tryRegisterPreviewRef(pageIdValue)) return
    if (registerAttempts >= maxRegisterAttempts) return
    registerAttempts += 1
    if (registerTimer) clearTimeout(registerTimer)
    registerTimer = setTimeout(() => {
      scheduleRegisterPreviewRef(pageIdValue)
    }, 120)
  }

  function unregisterPreviewRef(label: string, pageIdValue = previewPageId.value) {
    if (!readonly.value || !label) return
    const runtime = getPreviewRuntime()
    if (!runtime?.unregisterComponentRef) return
    if (!pageIdValue) return
    const refInfo = buildRefInfo()
    runtime.unregisterComponentRef(pageIdValue, label, refInfo)
    const pageId = currentPage.value?.id
    const pageName = currentPage.value?.name
    if (pageId && pageId !== pageIdValue) {
      runtime.unregisterComponentRef(pageId, label, refInfo)
    }
    if (pageName && pageName !== pageIdValue) {
      runtime.unregisterComponentRef(pageName, label, refInfo)
    }
  }

  watch(
    () => node.value?.label,
    (next, prev) => {
      if (!readonly.value) return
      if (prev && prev !== next) {
        unregisterPreviewRef(prev)
      }
      if (next) {
        registerAttempts = 0
        scheduleRegisterPreviewRef(previewPageId.value)
      }
    },
  )

  watch(
    () => previewPageId.value,
    (next, prev) => {
      if (!readonly.value) return
      if (prev && node.value?.label) {
        unregisterPreviewRef(node.value.label, prev)
      }
      if (next && node.value?.label) {
        registerAttempts = 0
        scheduleRegisterPreviewRef(next)
      }
    },
  )

  onMounted(() => {
    registerAttempts = 0
    scheduleRegisterPreviewRef(previewPageId.value)
  })

  onBeforeUnmount(() => {
    if (registerTimer) {
      clearTimeout(registerTimer)
      registerTimer = null
    }
    if (node.value?.label) unregisterPreviewRef(node.value.label)
  })
}
