import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { Storage } from '@/utils/storage'

const STORAGE_KEY = 'ui_prefs_v2'

interface UiPrefs {
  // 抽屉宽度（像素）
  drawerWidth: number
  // 列表每页条数
  pageSize: number
  // 侧边栏是否折叠
  sidebarCollapsed: boolean
  // 详情抽屉是否处于钉住（旁路面板）状态
  pinnedDrawer: boolean
}

const DEFAULT_PREFS: UiPrefs = {
  drawerWidth: 480,
  pageSize: 20,
  sidebarCollapsed: false,
  pinnedDrawer: false,
}

function loadFromStorage(): UiPrefs {
  const stored = Storage.get(STORAGE_KEY, null) as Partial<UiPrefs> | null
  if (!stored) return { ...DEFAULT_PREFS }
  return { ...DEFAULT_PREFS, ...stored }
}

export const useUiPrefsStore = defineStore('uiPrefs', () => {
  const prefs = ref<UiPrefs>(loadFromStorage())

  // 任何变更同步到 localStorage
  watch(
    prefs,
    (val) => {
      Storage.set(STORAGE_KEY, val)
    },
    { deep: true },
  )

  function setDrawerWidth(width: number) {
    prefs.value.drawerWidth = width
  }

  function setPageSize(size: number) {
    prefs.value.pageSize = size
  }

  function setSidebarCollapsed(collapsed: boolean) {
    prefs.value.sidebarCollapsed = collapsed
  }

  function setPinnedDrawer(pinned: boolean) {
    prefs.value.pinnedDrawer = pinned
  }

  function reset() {
    prefs.value = { ...DEFAULT_PREFS }
  }

  return {
    prefs,
    setDrawerWidth,
    setPageSize,
    setSidebarCollapsed,
    setPinnedDrawer,
    reset,
  }
})
