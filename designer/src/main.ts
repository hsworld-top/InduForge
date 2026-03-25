/**
 * 设计器主入口
 *
 * 职责：
 * - 创建 Vue 应用并挂载
 * - 注册 Pinia、Vue Router、Element Plus
 * - 注册内置组件（builtin-manifests）
 * - 移除 HTML 中的初始加载占位
 * - 监听父窗口主题更新消息（iframe 嵌入场景）
 */

import ElementPlus from "element-plus";
import { createPinia } from "pinia";
import { createApp } from "vue";
import App from "./App.vue";
import { registerAllDescriptors } from "./components";
import * as descriptorRegistry from "./components/descriptors/registry";
import { STORAGE_KEYS } from "./constants";
import { initDescriptorRegistry } from "./editor-core/document/factory";
import { registerBuiltinComponents } from "./editor-core/registry/builtin-manifests";
import router from "./router";
import { Storage } from "./utils/storage";
import "element-plus/dist/index.css";
import "./assets/styles/main.css";

const app = createApp(App);

registerBuiltinComponents();
registerAllDescriptors();
initDescriptorRegistry(descriptorRegistry);

app.use(createPinia());
app.use(router);
app.use(ElementPlus);

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

function handleThemeMessage(event: MessageEvent) {
  const data = event.data;
  if (!isThemeUpdatePayload(data)) return;
  Storage.set(STORAGE_KEYS.THEME, data.theme);
  document.documentElement.classList.toggle("dark", data.theme === "dark");
}

window.addEventListener("message", handleThemeMessage);
