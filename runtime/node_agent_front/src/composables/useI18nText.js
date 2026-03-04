import { computed } from 'vue'
import { usePreferencesStore } from '@/store/preferencesStore'
import { messages } from '@/i18n/messages'

/**
 * 轻量国际化工具，基于偏好设置返回当前语言文案。
 * @returns {{ locale: import('vue').ComputedRef<string>, t: (key: string) => string }}
 */
export const useI18nText = () => {
  const preferences = usePreferencesStore()

  const locale = computed(() => (preferences.locale === 'en-US' ? 'en-US' : 'zh-CN'))

  /**
   * 根据 key 读取文案，key 使用 a.b.c 的路径格式。
   * @param {string} key 文案路径
   * @returns {string}
   */
  const t = (key) => {
    const source = messages[locale.value] || messages['zh-CN']
    const value = key.split('.').reduce((acc, part) => acc?.[part], source)
    return value || key
  }

  return {
    locale,
    t,
  }
}
