import { createRouter, createWebHistory } from "vue-router";
import { STORAGE_KEYS } from "@/constants";
import { shouldRedirectTopLevelToIde, shouldUseDebugMode, waitForHostBootstrap, buildIdeRestoreUrl, resolveIdeOriginFromRuntime } from "@/runtime/host-bootstrap";
import { Storage } from "@/utils/storage";
// import DataCenter from "../views/DataCenter.vue"; // 原版本
import DataCenter from "../views/DataCenterNew.vue"; // 重构版本

const routes = [
  {
    path: "/",
    name: "datacenter",
    component: DataCenter,
    meta: {
      title: "数据中心",
      requiresAuth: true,
    },
  },
];

const router = createRouter({
  history: createWebHistory("/datacenter/"),
  routes,
});

/**
 * 应用主题到文档根节点。
 * @param {string} theme - 主题
 */
const applyTheme = (theme) => {
  document.documentElement.classList.toggle("dark", theme === "dark");
};

/**
 * 用当前导航目标重建绝对地址，避免依赖 window.location.href 误判切路由场景。
 * @param {RouteLocationNormalized} to - 目标路由
 * @returns {URL} 目标地址
 */
const resolveTargetUrl = (to) => {
  const resolvedHref = router.resolve(to).href;
  return new URL(resolvedHref, window.location.origin);
};

/**
 * 从目标地址里提取 handoff 标识。
 * @param {URL} url - 目标地址
 * @returns {{handoffId?: string}|null} handoff 信息
 */
const resolveHandoff = (url) => {
  const handoffId = url.searchParams.get("handoffId");
  return handoffId ? { handoffId } : null;
};

/**
 * 构建 IDE 登录地址。
 * iframe 场景里 bootstrap 超时后，仍然保留现有的登录兜底逻辑。
 * @param {string} currentUrl - 当前访问地址
 * @param {string} ideOrigin - IDE origin
 * @returns {string} 登录地址
 */
const buildIdeLoginUrl = (currentUrl, ideOrigin) => {
  const loginUrl = new URL("/login", ideOrigin);
  loginUrl.searchParams.set("redirect", currentUrl);
  return loginUrl.toString();
};

// 路由守卫 - 鉴权检查和状态恢复
router.beforeEach(async (to, from, next) => {
  // 设置页面标题
  document.title = `${to.meta.title || "数据中心"} - ProjectIDE`;

  const targetUrl = resolveTargetUrl(to);
  const isDebugRoute = shouldUseDebugMode(targetUrl.pathname);
  const isTopLevelWindow = window.parent === window;
  const ideOrigin = resolveIdeOriginFromRuntime({
    currentUrl: targetUrl.toString(),
    referrer: document.referrer,
  });

  if (shouldRedirectTopLevelToIde(targetUrl.pathname, isTopLevelWindow) && !isDebugRoute) {
    next(false);
    window.location.replace(buildIdeRestoreUrl(resolveHandoff(targetUrl), ideOrigin));
    return;
  }

  // 正式入口只接受宿主 bootstrap 注入的运行态，不再从 URL 消费 token / refreshToken / theme / pid / tenant。
  let token = Storage.getToken();
  if (!token && !isDebugRoute) {
    await waitForHostBootstrap();
    token = Storage.getToken();
  }

  const theme = Storage.get(STORAGE_KEYS.THEME, "light");
  applyTheme(theme);

  if (!token && !isDebugRoute) {
    next(false);
    window.location.replace(buildIdeLoginUrl(targetUrl.toString(), ideOrigin));
    return;
  }

  const projectId = Storage.getProjectId();
  const tenantId = Storage.getTenantId();

  if (!projectId && !isDebugRoute) {
    next(false);
    window.location.replace(buildIdeRestoreUrl(resolveHandoff(targetUrl), ideOrigin));
    return;
  }

  to.meta.project = {
    id: projectId,
    tenantId,
  };

  next();
});

export default router;
