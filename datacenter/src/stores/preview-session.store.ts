import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

// 局部作用域：仅在接入源工作台预览激活时持有
// session 生命周期由外部（usePreviewSession composable）管理
interface PreviewSession {
  sessionId: string
  projectId: string
  // 会话创建时间（dayjs 格式字符串）
  createdAt: string
}

export const usePreviewSessionStore = defineStore('previewSession', () => {
  const session = ref<PreviewSession | null>(null)
  // 预览激活状态
  const active = computed(() => session.value !== null)

  function open(payload: PreviewSession) {
    session.value = payload
  }

  function close() {
    session.value = null
  }

  return {
    session,
    active,
    open,
    close,
  }
})
