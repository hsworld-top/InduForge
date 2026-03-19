/**
 * 设计器路由配置
 *
 * 路由说明：
 * - / : 设计器主界面（DesignerView）
 * - /preview : 预览界面（PreviewView）
 *
 * 运行态由独立的 RuntimeEngine 处理，不在此路由内。
 * 鉴权、主题、工程 ID 等通过 URL 参数或 Storage 同步。
 */

import { createRouter, createWebHistory } from "vue-router";
import { Storage } from "@/utils/storage";
import { STORAGE_KEYS } from "@/constants";

const routes = [
  {
    path: "/",
    name: "Designer",
    component: () => import("@/ui/Designer/DesignerView.vue"),
    meta: {
      title: "设计器",
      requiresAuth: true,
    },
  },
  {
    path: "/preview",
    name: "Preview",
    component: () => import("@/ui/Preview/PreviewView.vue"),
    meta: {
      title: "预览",
      requiresAuth: true,
    },
  },
];

const router = createRouter({
  history: createWebHistory("/designer/"),
  routes,
});

/**
 * 应用主题到 document.documentElement
 * @param {string} theme - 主题，'dark' 时添加 dark 类
 */
const applyTheme = (theme) => {
  document.documentElement.classList.toggle("dark", theme === "dark");
};

/**
 * 从 URL 同步鉴权与主题配置
 * 支持 token、refreshToken、theme、pid/id、tenant 等参数，
 * 同步后从 URL 中移除敏感参数并 replaceState
 */
const syncRuntimeSettings = () => {
  const url = new URL(window.location.href);
  const urlParams = url.searchParams;
  let shouldReplace = false;

  const tokenFromUrl = urlParams.get("token");
  const refreshTokenFromUrl = urlParams.get("refreshToken");
  const themeFromUrl = urlParams.get("theme");
  const projectIdFromUrl = urlParams.get("pid") || urlParams.get("id");
  const tenantIdFromUrl = urlParams.get("tenant");

  const themeValue = ["light", "dark"].includes(themeFromUrl)
    ? themeFromUrl
    : Storage.get(STORAGE_KEYS.THEME, "light");

  // 同步 Token
  if (tokenFromUrl) {
    Storage.setToken(tokenFromUrl);
    urlParams.delete("token");
    shouldReplace = true;
  }

  if (refreshTokenFromUrl) {
    Storage.setRefreshToken(refreshTokenFromUrl);
    urlParams.delete("refreshToken");
    shouldReplace = true;
  }

  // 同步主题
  if (themeFromUrl && ["light", "dark"].includes(themeFromUrl)) {
    Storage.set(STORAGE_KEYS.THEME, themeFromUrl);
    urlParams.delete("theme");
    shouldReplace = true;
  }

  // 同步工程 ID
  if (projectIdFromUrl) {
    Storage.setProjectId(projectIdFromUrl);
  }

  // 同步租户 ID
  if (tenantIdFromUrl) {
    Storage.setTenantId(tenantIdFromUrl);
  }

  applyTheme(themeValue);

  if (shouldReplace) {
    const nextQuery = urlParams.toString();
    const nextUrl = nextQuery ? `${url.pathname}?${nextQuery}` : url.pathname;
    window.history.replaceState({}, "", nextUrl);
  }
};

/** 路由守卫：鉴权检查、页面标题、工程上下文同步 */
router.beforeEach(async (to, from, next) => {
  // 设置页面标题
  document.title = `${to.meta.title || "设计器"} - InduForge`;

  // 同步外部传入的 token/主题信息
  syncRuntimeSettings();

  const isDev = import.meta.env.DEV;
  const devHost = import.meta.env.VITE_DEV_HOST || "localhost";
  const idePort = import.meta.env.VITE_IDE_PORT || 9091;
  const ideOrigin = isDev ? `http://${devHost}:${idePort}` : "";

  // 检查鉴权 (Token) - 从 LocalStorage 读取
  const token = Storage.getToken();
  if (!token && to.meta.requiresAuth) {
    // 没登录，跳回主应用登录页，并保存当前 URL 以便登录后跳转回来
    const redirectUrl = encodeURIComponent(window.location.href);
    window.location.href = `${ideOrigin}/login?redirect=${redirectUrl}`;
    return;
  }

  // 从 URL 参数获取工程上下文信息
  const urlParams = new URLSearchParams(window.location.search);
  const projectId =
    urlParams.get("pid") || urlParams.get("id") || Storage.getProjectId();
  const tenantId = urlParams.get("tenant") || Storage.getTenantId();

  // 验证必需参数（设计器需要工程 ID）
  if (!projectId && to.name === "Designer") {
    // 如果没有工程 ID，跳转回主应用的工程管理页面
    window.location.href = `${ideOrigin}/`;
    return;
  }

  // 将工程信息存储到路由 meta 中，供组件使用
  to.meta.project = {
    id: projectId,
    tenantId: tenantId,
  };

  next();
});

export default router;
