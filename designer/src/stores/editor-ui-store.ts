import { computed, ref, type ComputedRef, type Ref } from "vue";
import enLocale from "element-plus/es/locale/lang/en";
import zhLocale from "element-plus/es/locale/lang/zh-cn";
import { i18n } from "@/i18n";
import { Storage } from "@/utils/storage";

export type EditorTheme = "light" | "dark";
export type EditorLocale = "zh" | "en";

export interface EditorUiRuntimeInput {
  theme?: unknown;
  locale?: unknown;
}

const isEditorTheme = (value: unknown): value is EditorTheme => value === "light" || value === "dark";

const isEditorLocale = (value: unknown): value is EditorLocale => value === "zh" || value === "en";

export interface EditorUiStore {
  theme: Ref<EditorTheme>;
  locale: Ref<EditorLocale>;
  elementLocale: ComputedRef<typeof zhLocale | typeof enLocale>;
  initFromRuntime(input?: EditorUiRuntimeInput): void;
  setTheme(nextTheme: EditorTheme): void;
  setLocale(nextLocale: EditorLocale): void;
  applyThemeToDom(nextTheme?: EditorTheme): void;
}

/**
 * 编辑器 UI 状态单元。
 *
 * 只负责编辑器壳层的主题、语言和本地缓存，不触碰页面 Schema。
 */
let editorUiStore: EditorUiStore | null = null;

const resolveTheme = (input?: unknown): EditorTheme => {
  if (isEditorTheme(input)) {
    return input;
  }

  return Storage.getDesignerTheme();
};

const resolveLocale = (input?: unknown): EditorLocale => {
  if (isEditorLocale(input)) {
    return input;
  }

  return Storage.getDesignerLanguage();
};

const syncLocaleToI18n = (nextLocale: EditorLocale): void => {
  i18n.global.locale.value = nextLocale;
};

const createEditorUiStoreImpl = (): EditorUiStore => {
  const theme = ref<EditorTheme>("light");
  const locale = ref<EditorLocale>("zh");

  const applyThemeToDom = (nextTheme: EditorTheme = theme.value): void => {
    if (typeof document === "undefined") {
      return;
    }

    const { documentElement } = document;
    documentElement.classList.toggle("dark", nextTheme === "dark");
    documentElement.setAttribute("data-theme", nextTheme);
  };

  const initFromRuntime = (input?: EditorUiRuntimeInput): void => {
    theme.value = resolveTheme(input?.theme);
    locale.value = resolveLocale(input?.locale);

    applyThemeToDom(theme.value);
    syncLocaleToI18n(locale.value);
    Storage.setDesignerTheme(theme.value);
    Storage.setDesignerLanguage(locale.value);
  };

  const setTheme = (nextTheme: EditorTheme): void => {
    if (!isEditorTheme(nextTheme)) {
      return;
    }

    theme.value = nextTheme;
    applyThemeToDom(nextTheme);
    Storage.setDesignerTheme(nextTheme);
  };

  const setLocale = (nextLocale: EditorLocale): void => {
    if (!isEditorLocale(nextLocale)) {
      return;
    }

    locale.value = nextLocale;
    syncLocaleToI18n(nextLocale);
    Storage.setDesignerLanguage(nextLocale);
  };

  const elementLocale = computed(() => (locale.value === "en" ? enLocale : zhLocale));

  return {
    theme,
    locale,
    elementLocale,
    initFromRuntime,
    setTheme,
    setLocale,
    applyThemeToDom,
  };
};

export function createEditorUiStore(): EditorUiStore {
  if (!editorUiStore) {
    editorUiStore = createEditorUiStoreImpl();
  }

  return editorUiStore;
}

export const getEditorUiStore = (): EditorUiStore => createEditorUiStore();

export const editorUiState = createEditorUiStore();
