import { createApp } from 'vue'
import App from './App.vue'
import router from './router/index.js'
import { installPreviewDevtoolsBridge } from './preview-devtools.js'
import './styles.css'

installPreviewDevtoolsBridge()
createApp(App).use(router).mount('#app')
