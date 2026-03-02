import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

const THEME_KEY = 'node_agent_theme'
const LOCALE_KEY = 'node_agent_locale'

export const usePreferencesStore = defineStore('preferences', () => {
  const theme = ref(localStorage.getItem(THEME_KEY) || 'light')
  const locale = ref(localStorage.getItem(LOCALE_KEY) || 'zh-CN')

  const applyTheme = (value) => {
    document.documentElement.setAttribute('data-theme', value)
  }

  const setTheme = (value) => {
    theme.value = value
    applyTheme(value)
  }

  const setLocale = (value) => {
    locale.value = value
  }

  watch(theme, (value) => {
    localStorage.setItem(THEME_KEY, value)
  })

  watch(locale, (value) => {
    localStorage.setItem(LOCALE_KEY, value)
  })

  applyTheme(theme.value)

  return {
    theme,
    locale,
    setTheme,
    setLocale,
  }
})

