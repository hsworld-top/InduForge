/**
 * 设计器主入口。
 *
 * 关键职责：
 * - 初始化 Vue 应用、Pinia、Router、Element Plus；
 * - 为正式入口建立宿主 bootstrap 握手；
 * - 统一处理来自宿主的认证、主题、语言同步消息。
 */

import ElementPlus from 'element-plus'
import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { getEditorUiStore } from './stores/editor-ui-store'
import { initializeWujieContext } from './runtime/wujie-context'
import 'element-plus/dist/index.css'
import './assets/styles/main.css'

export function bootstrapDesignerApp(): void {
  const app = createApp(App)
  const pinia = createPinia()

  app.use(pinia)

  const editorUi = getEditorUiStore()
  editorUi.initFromRuntime(initializeWujieContext() ?? undefined)

  app.use(router)
  app.use(ElementPlus)
  app.provide('editorUi', editorUi)

  app.mount('#app')

  const initialLoading = document.getElementById('app-loading')
  if (initialLoading) {
    if (window.location.pathname.includes('/preview')) {
      initialLoading.remove()
    } else {
      requestAnimationFrame(() => {
        initialLoading.remove()
      })
    }
  }
}

if (!import.meta.env.VITEST) {
  bootstrapDesignerApp()
}
