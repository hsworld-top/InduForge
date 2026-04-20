/**
 * 璁捐鍣ㄤ富鍏ュ彛
 *
 * 鑱岃矗锛?
 * - 鍒涘缓 Vue 搴旂敤骞舵寕杞?
 * - 娉ㄥ唽 Pinia銆乂ue Router銆丒lement Plus
 * - 娉ㄥ唽鍐呯疆缁勪欢锛坆uiltin-manifests锛?
 * - 绉婚櫎 HTML 涓殑鍒濆鍔犺浇鍗犱綅
 * - 鐩戝惉鐖剁獥鍙ｄ富棰樻洿鏂版秷鎭紙iframe 宓屽叆鍦烘櫙锛?
 */

import ElementPlus from "element-plus";
import { createPinia } from "pinia";
import { createApp } from "vue";
import { i18n } from "./i18n";
import App from "./App.vue";
import { registerAllDescriptors } from "./materials";
import * as descriptorRegistry from "./editor-core/descriptors/registry";
import { initDescriptorRegistry } from "./editor-core/document/factory";
import { registerBuiltinComponents } from "./editor-core/registry/builtin-manifests";
import router from "./router";
import { getEditorUiStore } from "./stores/editor-ui-store";
import { shouldSyncEditorUiForPath } from "./router/runtime-settings";
import "element-plus/dist/index.css";
import "./assets/styles/main.css";

type TrustedMessageSource = MessageEventSource | null;

function isThemeUpdatePayload(data: unknown): data is {
  type: string;
  theme: "light" | "dark";
} {
  if (data === null || typeof data !== "object") return false;
  const o = data as Record<string, unknown>;
  if (o.type !== "THEME_UPDATE") return false;
  const theme = o.theme;
  return theme === "light" || theme === "dark";
}

function isLocaleUpdatePayload(data: unknown): data is {
  type: string;
  locale: "zh" | "en";
} {
  if (data === null || typeof data !== "object") return false;
  const o = data as Record<string, unknown>;
  if (o.type !== "LOCALE_UPDATE") return false;
  const locale = o.locale;
  return locale === "zh" || locale === "en";
}

type RuntimeMessageHandlerDependencies = {
  editorUi: ReturnType<typeof getEditorUiStore>;
  i18n: typeof i18n;
  getPathname?: () => string;
  getTrustedOriginSet?: () => Set<string>;
  getTrustedSources?: () => TrustedMessageSource[];
};

export function resolveTrustedMessageSources({
  locationOrigin,
  referrer,
}: {
  locationOrigin: string;
  referrer: string;
}): Set<string> {
  const trustedOrigins = new Set<string>([locationOrigin]);

  if (!referrer) {
    return trustedOrigins;
  }

  try {
    trustedOrigins.add(new URL(referrer).origin);
  } catch {
    // 忽略无法解析的 referrer，保持当前页面 origin 为最低信任边界。
  }

  return trustedOrigins;
}

export function createRuntimeMessageHandler({
  editorUi,
  i18n,
  getPathname = () => window.location.pathname,
  getTrustedOriginSet = () =>
    resolveTrustedMessageSources({
      locationOrigin: window.location.origin,
      referrer: document.referrer,
    }),
  getTrustedSources = () => {
    const sources: TrustedMessageSource[] = [window];
    if (window.parent !== window) {
      sources.push(window.parent);
    }
    return sources;
  },
}: RuntimeMessageHandlerDependencies): (event: MessageEvent) => void {
  return (event: MessageEvent) => {
    if (!shouldSyncEditorUiForPath(getPathname())) {
      return;
    }

    const trustedOrigins = getTrustedOriginSet();
    if (!trustedOrigins.has(event.origin)) {
      return;
    }

    const trustedSources = getTrustedSources();
    if (!trustedSources.some((source) => source === event.source)) {
      return;
    }

    const data = event.data;

    if (isThemeUpdatePayload(data)) {
      editorUi.setTheme(data.theme);
      return;
    }

    if (isLocaleUpdatePayload(data)) {
      editorUi.setLocale(data.locale);
      i18n.global.locale.value = data.locale;
    }
  };
}

export function bootstrapDesignerApp(): void {
  const app = createApp(App);
  const editorUi = getEditorUiStore();

  registerBuiltinComponents();
  registerAllDescriptors();
  initDescriptorRegistry(descriptorRegistry);

  app.use(createPinia());
  app.use(router);
  app.use(i18n);
  app.use(ElementPlus);
  app.provide("editorUi", editorUi);

  app.mount("#app");

  const initialLoading = document.getElementById("app-loading");
  if (initialLoading) {
    if (window.location.pathname.includes("/preview")) {
      initialLoading.remove();
    } else {
      requestAnimationFrame(() => {
        initialLoading.remove();
      });
    }
  }

  window.addEventListener("message", createRuntimeMessageHandler({
    editorUi,
    i18n,
    getPathname: () => window.location.pathname,
    getTrustedOriginSet: () =>
      resolveTrustedMessageSources({
        locationOrigin: window.location.origin,
        referrer: document.referrer,
      }),
    getTrustedSources: () => {
      const sources: TrustedMessageSource[] = [window];
      if (window.parent !== window) {
        sources.push(window.parent);
      }
      return sources;
    },
  }));
}

if (!import.meta.env.VITEST) {
  bootstrapDesignerApp();
}

