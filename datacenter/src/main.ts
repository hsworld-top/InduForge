// @ts-nocheck
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
// import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import router from './router'
import App from './App.vue'
import './assets/styles/main.css'
import { configureElementPlusPopper } from './config/element-plus'
import { setDatacenterLocale } from './i18n/runtime'
import { Storage } from './utils/storage'
import { initializeWujieContext } from './runtime/wujie-context'

configureElementPlusPopper()

const microAppContext = initializeWujieContext()
setDatacenterLocale(microAppContext?.locale ?? Storage.getLanguage())

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(ElementPlus)

// 注册所有Element Plus图标组件
// 已迁移到 unplugin-icons，不再需要全局注册
// for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
//   app.component(key, component)
// }

app.mount('#app')

const initialLoading = document.getElementById('app-loading')
if (initialLoading) {
  window.requestAnimationFrame(() => {
    initialLoading.remove()
  })
}
