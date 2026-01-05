import { createRouter, createWebHistory } from 'vue-router';
import { Storage } from '@/utils/storage';
import { STORAGE_KEYS } from '@/constants';
import DesignCenter from '../views/DesignCenter.vue';

const routes = [
    {
        path: '/',
        name: 'designer',
        component: DesignCenter,
        meta: {
            title: '设计中心',
            requiresAuth: true,
        },
    },
];

const router = createRouter({
    history: createWebHistory('/designer/'),
    routes,
});

/**
 * 应用主题到文档根节点。
 * @param {string} theme - 主题
 */
const applyTheme = (theme) => {
    document.documentElement.classList.toggle('dark', theme === 'dark');
};

/**
 * 从 URL 同步鉴权与主题配置。
 */
const syncRuntimeSettings = () => {
    const url = new URL(window.location.href);
    const urlParams = url.searchParams;
    let shouldReplace = false;

    const tokenFromUrl = urlParams.get('token');
    const refreshTokenFromUrl = urlParams.get('refreshToken');
    const themeFromUrl = urlParams.get('theme');
    const themeValue = ['light', 'dark'].includes(themeFromUrl)
        ? themeFromUrl
        : Storage.get(STORAGE_KEYS.THEME, 'light');

    if (tokenFromUrl) {
        Storage.setToken(tokenFromUrl);
        urlParams.delete('token');
        shouldReplace = true;
    }

    if (refreshTokenFromUrl) {
        Storage.setRefreshToken(refreshTokenFromUrl);
        urlParams.delete('refreshToken');
        shouldReplace = true;
    }

    if (themeFromUrl && ['light', 'dark'].includes(themeFromUrl)) {
        Storage.set(STORAGE_KEYS.THEME, themeFromUrl);
        urlParams.delete('theme');
        shouldReplace = true;
    }

    applyTheme(themeValue);

    if (shouldReplace) {
        const nextQuery = urlParams.toString();
        const nextUrl = nextQuery ? `${url.pathname}?${nextQuery}` : url.pathname;
        window.history.replaceState({}, '', nextUrl);
    }
};

// 路由守卫 - 鉴权检查和状态恢复
router.beforeEach(async (to, from, next) => {
    // 设置页面标题
    document.title = `${to.meta.title || '设计中心'} - ProjectIDE`;

    // 同步外部传入的 token/主题信息
    syncRuntimeSettings();

    const isDev = import.meta.env.DEV;
    const devHost = import.meta.env.VITE_DEV_HOST || 'localhost';
    const idePort = import.meta.env.VITE_IDE_PORT || 9091;
    const ideOrigin = isDev ? `http://${devHost}:${idePort}` : '';

    // 1. 检查鉴权 (Token) - 从 LocalStorage 读取
    const token = Storage.getToken();
    if (!token) {
        // 没登录，跳回主应用登录页，并保存当前 URL 以便登录后跳转回来
        const redirectUrl = encodeURIComponent(window.location.href);
        window.location.href = `${ideOrigin}/login?redirect=${redirectUrl}`;
        return;
    }

    // 2. 从 URL 参数获取工程上下文信息
    const urlParams = new URLSearchParams(window.location.search);
    const projectId = urlParams.get('pid') || urlParams.get('id'); // 兼容旧的 id 参数
    const tenantId = urlParams.get('tenant');
    const pageId = urlParams.get('pageid') || '1';
    const type = urlParams.get('type') || 'app';

    // 3. 验证必需参数
    if (!projectId) {
        // 如果没有工程 ID，跳转回主应用的工程管理页面
        window.location.href = `${ideOrigin}/`;
        return;
    }

    // 4. 将工程信息存储到路由 meta 中，供组件使用
    to.meta.project = {
        id: projectId,
        tenantId: tenantId,
        pageId: pageId,
        type: type,
    };

    // 5. 确保租户 ID 已设置（如果 URL 中有）
    if (tenantId) {
        const currentTenantId = Storage.getTenantId();
        if (currentTenantId !== tenantId) {
            Storage.setTenantId(tenantId);
        }
    }

    next();
});

export default router;
