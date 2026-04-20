import { createApp } from "vue";
import { createPinia } from "pinia";
import ElementPlus from "element-plus";
import "element-plus/dist/index.css";
// import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import router from "./router";
import App from "./App.vue";
import "./assets/styles/main.css";
import {
  applyThemeToDocument,
  getTrustedHostOriginSet,
  getTrustedHostSources,
  handleAuthRefreshedMessage,
  handleBootstrapResponseMessage,
  initializeHostBootstrap,
  isTrustedHostMessage,
  postAppBootstrapRequest,
} from "./runtime/host-bootstrap.js";
import { initMessageHandler } from "./utils/messageHandler";
import { Storage } from "./utils/storage.js";
import "./utils/socketTest";

export function createRuntimeMessageHandler({
  getTrustedOriginSet = () => getTrustedHostOriginSet(),
  getTrustedSources = () => getTrustedHostSources(),
} = {}) {
  return (event) => {
    const data = event.data;
    if (!data || typeof data !== "object") {
      return;
    }

    if (
      ![
        "APP_BOOTSTRAP_RESPONSE",
        "AUTH_REFRESHED",
        "THEME_UPDATE",
        "LOCALE_UPDATE",
      ].includes(data.type)
    ) {
      return;
    }

    if (
      !isTrustedHostMessage(event, {
        trustedOrigins: getTrustedOriginSet(),
        trustedSources: getTrustedSources(),
      })
    ) {
      return;
    }

    if (handleBootstrapResponseMessage(data)) {
      return;
    }

    if (handleAuthRefreshedMessage(data)) {
      return;
    }

    if (
      data.type === "THEME_UPDATE" &&
      ["light", "dark"].includes(data.theme)
    ) {
      Storage.setTheme(data.theme);
      applyThemeToDocument(data.theme);
      return;
    }

    if (data.type === "LOCALE_UPDATE" && typeof data.locale === "string") {
      const locale = data.locale.trim();
      if (locale) {
        Storage.setLanguage(locale);
      }
    }
  };
}

const hostBootstrap = initializeHostBootstrap({
  currentUrl: window.location.href,
  isTopLevelWindow: window.parent === window,
  referrer: document.referrer,
});
const runtimeMessageHandler = createRuntimeMessageHandler();

applyThemeToDocument(Storage.getTheme());
window.addEventListener("message", runtimeMessageHandler);

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(ElementPlus);

// 注册所有Element Plus图标组件
// 已迁移到 unplugin-icons，不再需要全局注册
// for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
//   app.component(key, component)
// }

app.mount("#app");

// Designer 与宿主都依赖这条桥接链路，因此初始化放在挂载后立即执行。
initMessageHandler();

if (hostBootstrap.shouldWaitForBootstrap) {
  postAppBootstrapRequest();
}

const initialLoading = document.getElementById("app-loading");
if (initialLoading) {
  window.requestAnimationFrame(() => {
    initialLoading.remove();
  });
}
