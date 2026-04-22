import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import router from './router'
import i18n from './lang'
import App from './App.vue'
import './assets/styles/main.css'
import { Storage } from './utils/storage'

const initialTheme = Storage.getTheme()
document.documentElement.classList.toggle('dark', initialTheme === 'dark')
document.documentElement.setAttribute('data-theme', initialTheme)

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(ElementPlus)
app.use(i18n)

// 注册所有Element Plus图标组件
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.mount('#app')
