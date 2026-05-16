import { defineStore } from "pinia";
import { ref, computed } from "vue";

// IDE bootstrap 注入的项目上下文
export const useProjectStore = defineStore("project", () => {
  const projectId = ref<string>("");
  const tenantId = ref<string>("");
  // 界面语言，如 zh-CN / en-US
  const locale = ref<string>("zh-CN");
  // 主题：light / dark
  const theme = ref<string>("light");
  // 工程能力标志集合（功能开关，由宿主 bootstrap 注入）
  const capability = ref<Record<string, boolean>>({});

  const hasProject = computed(() => projectId.value !== "");

  /** 由 IDE bootstrap 调用，初始化项目上下文 */
  function bootstrap(payload: {
    projectId: string;
    tenantId?: string;
    locale?: string;
    theme?: string;
    capability?: Record<string, boolean>;
  }) {
    projectId.value = payload.projectId;
    if (payload.tenantId) tenantId.value = payload.tenantId;
    if (payload.locale) locale.value = payload.locale;
    if (payload.theme) theme.value = payload.theme;
    if (payload.capability) capability.value = payload.capability;
  }

  function setLocale(val: string) {
    locale.value = val;
  }

  function setTheme(val: string) {
    theme.value = val;
  }

  return {
    projectId,
    tenantId,
    locale,
    theme,
    capability,
    hasProject,
    bootstrap,
    setLocale,
    setTheme,
  };
});
