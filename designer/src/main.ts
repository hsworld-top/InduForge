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
import App from "./App.vue";
import { registerAllDescriptors } from "./components";
import * as descriptorRegistry from "./editor-core/descriptors/registry";
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

