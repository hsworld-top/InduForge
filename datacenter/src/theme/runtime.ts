import { ref } from 'vue'

export type DatacenterTheme = 'light' | 'dark'

export const datacenterTheme = ref<DatacenterTheme>('light')

export const normalizeDatacenterTheme = (theme: unknown): DatacenterTheme =>
  theme === 'dark' ? 'dark' : 'light'

/**
 * 统一应用宿主或独立运行时传入的主题，确保组件变量、Element Plus 和浏览器原生控件同步。
 */
export const applyDatacenterTheme = (theme: unknown): DatacenterTheme => {
  const normalized = normalizeDatacenterTheme(theme)
  datacenterTheme.value = normalized
  if (typeof document !== 'undefined') {
    document.documentElement.classList.toggle('dark', normalized === 'dark')
    document.documentElement.style.colorScheme = normalized
  }
  return normalized
}
