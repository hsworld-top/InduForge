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

import { createApp } from "vue";
import { createPinia } from "pinia";
import ElementPlus from "element-plus";
import "element-plus/dist/index.css";
import router from "./router";
import { Storage } from "./utils/storage";
import { STORAGE_KEYS } from "./constants";
import App from "./App.vue";
import { registerBuiltinComponents } from "./editor-core/registry/builtin-manifests";
import { registerAllDescriptors } from "./components/index.js";
import * as descriptorRegistry from "./components/descriptors/registry.js";
import { initDescriptorRegistry } from "./editor-core/document/factory.ts";
import "./assets/styles/main.css";

const app = createApp(App);

/** 注册设计器内置组件（按钮、输入框、布局等） */
registerBuiltinComponents();
/** 注册组件描述符（渲染标签、子项策略等元数据） */
registerAllDescriptors();
/** 将 descriptor 注册中心注入 factory，使 inferPositioning 可读取 descriptor */
initDescriptorRegistry(descriptorRegistry);

app.use(createPinia());
app.use(router);
app.use(ElementPlus);

app.mount("#app");

/** 移除 index.html 中的初始加载占位元素，预览页立即移除，设计页等待首帧渲染后移除 */
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

/**
 * 处理来自父窗口的主题更新消息（postMessage）。
 * 当设计器以 iframe 嵌入 IDE 时，父窗口可发送 THEME_UPDATE 消息切换明暗主题。
 * @param {MessageEvent} event - 消息事件
 */
const handleThemeMessage = (event) => {
  const data = event.data;
  if (!data || typeof data !== "object") {
    return;
  }
  if (data.type !== "THEME_UPDATE" || !["light", "dark"].includes(data.theme)) {
    return;
  }
  Storage.set(STORAGE_KEYS.THEME, data.theme);
  document.documentElement.classList.toggle("dark", data.theme === "dark");
};

window.addEventListener("message", handleThemeMessage);
