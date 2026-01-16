import { createApp } from 'vue';
import { createPinia } from 'pinia';
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
// import * as ElementPlusIconsVue from '@element-plus/icons-vue';
import router from './router';
import { Storage } from './utils/storage';
import { STORAGE_KEYS } from './constants';
import App from './App.vue';
import { registerBuiltinComponents } from "./editor-core/registry/builtinManifests.js";
import './assets/styles/main.css';

const app = createApp(App);

registerBuiltinComponents();

app.use(createPinia());
app.use(router);
app.use(ElementPlus);

// 注册所有Element Plus图标组件
// 已迁移到 unplugin-icons，不再需要全局注册
// for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
//     app.component(key, component);
// }

app.mount('#app');

const initialLoading = document.getElementById('app-loading');
if (initialLoading) {
    if (window.location.pathname.includes('/preview')) {
        initialLoading.remove();
    } else {
        requestAnimationFrame(() => {
            initialLoading.remove();
        });
    }
}

/**
 * 处理来自父窗口的主题更新消息。
 * @param {MessageEvent} event - 消息事件
 */
const handleThemeMessage = (event) => {
    const data = event.data;
    if (!data || typeof data !== 'object') {
        return;
    }
    if (data.type !== 'THEME_UPDATE' || !['light', 'dark'].includes(data.theme)) {
        return;
    }
    Storage.set(STORAGE_KEYS.THEME, data.theme);
    document.documentElement.classList.toggle('dark', data.theme === 'dark');
};

window.addEventListener('message', handleThemeMessage);
