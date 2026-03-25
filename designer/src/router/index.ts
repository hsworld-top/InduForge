/**
 * 设计器路由配置
 *
 * 路由说明：
 * - / : 设计器主界面（DesignerView）
 * - /preview : 预览界面（PreviewView）
 */

import type { NavigationGuardNext, RouteLocationNormalized } from "vue-router";
import { createRouter, createWebHistory } from "vue-router";
import { STORAGE_KEYS } from "@/constants";
import { Storage } from "@/utils/storage";

declare module "vue-router" {
  interface RouteMeta {
    title?: string;
    requiresAuth?: boolean;
    project?: {
      id: string | null | undefined;
      tenantId: string | null | undefined;
    };
  }
}

const routes = [
  {
    path: "/",
    name: "Designer",
    component: () => import("@/ui/shell/DesignerView.vue"),
    meta: {
      title: "设计器",
      requiresAuth: true,
    },
  },
  {
    path: "/preview",
    name: "Preview",
    component: () => import("@/ui/editors/page/preview/PreviewView.vue"),
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

function applyTheme(theme: string): void {
  document.documentElement.classList.toggle("dark", theme === "dark");
}

function syncRuntimeSettings(): void {
  const url = new URL(window.location.href);
  const urlParams = url.searchParams;
  let shouldReplace = false;

  const tokenFromUrl = urlParams.get("token");
  const refreshTokenFromUrl = urlParams.get("refreshToken");
  const themeFromUrl = urlParams.get("theme");
  const projectIdFromUrl = urlParams.get("pid") || urlParams.get("id");
  const tenantIdFromUrl = urlParams.get("tenant");

  const themeValue = ["light", "dark"].includes(themeFromUrl || "")
    ? (themeFromUrl as string)
    : (Storage.get(STORAGE_KEYS.THEME, "light") as string);

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

  if (themeFromUrl && ["light", "dark"].includes(themeFromUrl)) {
    Storage.set(STORAGE_KEYS.THEME, themeFromUrl);
    urlParams.delete("theme");
    shouldReplace = true;
  }

  if (projectIdFromUrl) {
    Storage.setProjectId(projectIdFromUrl);
  }

  if (tenantIdFromUrl) {
    Storage.setTenantId(tenantIdFromUrl);
  }

  applyTheme(themeValue);

  if (shouldReplace) {
    const nextQuery = urlParams.toString();
    const nextUrl = nextQuery ? `${url.pathname}?${nextQuery}` : url.pathname;
    window.history.replaceState({}, "", nextUrl);
  }
}

router.beforeEach(async (to: RouteLocationNormalized, _from, next: NavigationGuardNext) => {
  document.title = `${to.meta.title || "设计器"} - InduForge`;

  syncRuntimeSettings();

  const isDev = import.meta.env.DEV;
  const devHost = import.meta.env.VITE_DEV_HOST || "localhost";
  const idePort = import.meta.env.VITE_IDE_PORT || 9091;
  const ideOrigin = isDev ? `http://${devHost}:${idePort}` : "";

  const token = Storage.getToken();
  if (!token && to.meta.requiresAuth) {
    const redirectUrl = encodeURIComponent(window.location.href);
    window.location.href = `${ideOrigin}/login?redirect=${redirectUrl}`;
    return;
  }

  const urlParams = new URLSearchParams(window.location.search);
  const projectId = urlParams.get("pid") || urlParams.get("id") || Storage.getProjectId();
  const tenantId = urlParams.get("tenant") || Storage.getTenantId();

  if (!projectId && to.name === "Designer") {
    window.location.href = `${ideOrigin}/`;
    return;
  }

  to.meta.project = {
    id: projectId,
    tenantId,
  };

  next();
});

export default router;
