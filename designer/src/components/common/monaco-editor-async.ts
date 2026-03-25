import { defineAsyncComponent } from "vue";

/** 异步加载 Monaco（含 worker），减轻 Designer 首屏与主 chunk */
export default defineAsyncComponent(() => import("./MonacoEditor.vue"));
