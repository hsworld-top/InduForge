<template>
  <div ref="editorContainerRef" class="monaco-editor-container" :style="{ height }"></div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from "vue";
import * as monaco from "monaco-editor";
import EditorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";
import CssWorker from "monaco-editor/esm/vs/language/css/css.worker?worker";
import HtmlWorker from "monaco-editor/esm/vs/language/html/html.worker?worker";
import JsonWorker from "monaco-editor/esm/vs/language/json/json.worker?worker";
import TsWorker from "monaco-editor/esm/vs/language/typescript/ts.worker?worker";
import "monaco-editor/min/vs/editor/editor.main.css";
import "monaco-editor/esm/vs/basic-languages/css/css.contribution";
import "monaco-editor/esm/vs/basic-languages/javascript/javascript.contribution";
import "monaco-editor/esm/vs/basic-languages/html/html.contribution";
import "monaco-editor/esm/vs/language/css/monaco.contribution";
import "monaco-editor/esm/vs/language/html/monaco.contribution";
import "monaco-editor/esm/vs/language/json/monaco.contribution";
import "monaco-editor/esm/vs/language/typescript/monaco.contribution";

const props = defineProps({
  modelValue: { type: String, default: "" },
  language: { type: String, default: "css" },
  theme: { type: String, default: "vs" },
  height: { type: String, default: "300px" },
  options: { type: Object, default: () => ({}) },
  completions: { type: Array, default: () => [] },
});

const emit = defineEmits(["update:modelValue", "change", "markers"]);

const editorContainerRef = ref(null);
let editorInstance = null;
let isInternalUpdate = false;
const completionItems = ref(props.completions || []);
let completionProvider = null;
let markerTooltipEl = null;
let latestMarkers = [];
let extraLibDisposable = null;

const runEditorAction = async (id) => {
  const action = editorInstance?.getAction(id);
  if (!action) return false;
  try {
    await action.run();
    return true;
  } catch (error) {
    return false;
  }
};

const mediaQuery =
  typeof window !== "undefined" && window.matchMedia
    ? window.matchMedia("(prefers-color-scheme: dark)")
    : null;
const cleanupFns = [];

const normalizeTheme = (theme) => {
  if (theme === "dark") return "vs-dark";
  if (theme === "light") return "vs";
  return theme || "vs";
};

const detectTheme = () => {
  if (typeof document !== "undefined") {
    const html = document.documentElement;
    const body = document.body;
    const isDark = (el) =>
      el && (el.classList?.contains("dark") || el.dataset?.theme === "dark");
    if (isDark(html) || isDark(body)) return "vs-dark";
  }
  if (mediaQuery) return mediaQuery.matches ? "vs-dark" : "vs";
  return "vs";
};

function registerMonacoEnvironment() {
  if (
    typeof self === "undefined" ||
    (self.MonacoEnvironment && self.MonacoEnvironment.getWorker)
  ) {
    return;
  }

  self.MonacoEnvironment = {
    getWorker(_, label) {
      if (label === "css" || label === "scss" || label === "less") {
        return new CssWorker();
      }
      if (label === "html" || label === "handlebars" || label === "razor") {
        return new HtmlWorker();
      }
      if (label === "json") {
        return new JsonWorker();
      }
      if (label === "javascript" || label === "typescript") {
        return new TsWorker();
      }
      return new EditorWorker();
    },
  };
}

registerMonacoEnvironment();

const baseOptions = () => ({
  value: props.modelValue,
  language: props.language,
  theme: normalizeTheme(props.theme || detectTheme()),
  automaticLayout: true,
  fontSize: 14,
  minimap: { enabled: false },
  scrollBeyondLastLine: false,
  wordWrap: "on",
  formatOnPaste: true,
  formatOnType: true,
  folding: true,
  glyphMargin: true,
  lineNumbers: "on",
  tabSize: 2,
  suggestOnTriggerCharacters: true,
  quickSuggestions: { other: true, comments: true, strings: true },
  quickSuggestionsDelay: 50,
  wordBasedSuggestions: "allDocuments",
  inlineSuggest: { enabled: true },
  acceptSuggestionOnEnter: "on",
  acceptSuggestionOnCommitCharacter: true,
  tabCompletion: "on",
  snippetSuggestions: "inline",
  suggestSelection: "first",
  parameterHints: { enabled: true },
  lightbulb: { enabled: true },
  autoClosingBrackets: "always",
  autoClosingQuotes: "always",
  hover: { enabled: true, delay: 300 },
  ...props.options,
});

function applyTheme(theme) {
  monaco.editor.setTheme(normalizeTheme(theme || detectTheme()));
}

const ensureMarkerTooltip = () => {
  if (markerTooltipEl || !editorContainerRef.value) return;
  const el = document.createElement("div");
  el.style.position = "absolute";
  el.style.zIndex = "10";
  el.style.display = "none";
  el.style.maxWidth = "320px";
  el.style.padding = "6px 8px";
  el.style.fontSize = "12px";
  el.style.lineHeight = "1.4";
  el.style.color = "#303133";
  el.style.background = "#fff";
  el.style.border = "1px solid #e4e7ed";
  el.style.borderRadius = "4px";
  el.style.boxShadow = "0 2px 8px rgba(0,0,0,0.12)";
  el.style.pointerEvents = "none";
  el.style.whiteSpace = "pre-wrap";
  editorContainerRef.value.appendChild(el);
  markerTooltipEl = el;
};

const hideMarkerTooltip = () => {
  if (markerTooltipEl) {
    markerTooltipEl.style.display = "none";
    markerTooltipEl.textContent = "";
  }
};

const markerContainsPosition = (marker, position) => {
  if (!marker || !position) return false;
  if (position.lineNumber < marker.startLineNumber) return false;
  if (position.lineNumber > marker.endLineNumber) return false;
  if (
    position.lineNumber === marker.startLineNumber &&
    position.column < marker.startColumn
  ) {
    return false;
  }
  if (
    position.lineNumber === marker.endLineNumber &&
    position.column > marker.endColumn
  ) {
    return false;
  }
  return true;
};

const pickMarkerAtPosition = (position) => {
  if (!position) return null;
  const markers = latestMarkers || [];
  const matches = markers.filter((marker) =>
    markerContainsPosition(marker, position)
  );
  if (!matches.length) return null;
  return matches.sort((a, b) => b.severity - a.severity)[0];
};

const showMarkerTooltip = (position) => {
  if (!editorInstance || !position) {
    hideMarkerTooltip();
    return;
  }
  const marker = pickMarkerAtPosition(position);
  if (!marker) {
    hideMarkerTooltip();
    return;
  }
  const coords = editorInstance.getScrolledVisiblePosition(position);
  if (!coords) {
    hideMarkerTooltip();
    return;
  }
  ensureMarkerTooltip();
  if (!markerTooltipEl) return;
  markerTooltipEl.textContent = marker.message || "";
  markerTooltipEl.style.left = `${coords.left + 8}px`;
  markerTooltipEl.style.top = `${coords.top + coords.height + 6}px`;
  markerTooltipEl.style.display = "block";
};

const resolveCompletionKind = (kind) => {
  if (typeof kind === "number") return kind;
  if (
    typeof kind === "string" &&
    monaco.languages.CompletionItemKind[kind]
  ) {
    return monaco.languages.CompletionItemKind[kind];
  }
  return monaco.languages.CompletionItemKind.Text;
};

const resolveCompletionItems = (model, position) => {
  const word = model.getWordUntilPosition(position);
  const range = new monaco.Range(
    position.lineNumber,
    word.startColumn,
    position.lineNumber,
    word.endColumn
  );
  const lineText = model.getLineContent(position.lineNumber);
  const prefixText = lineText.slice(0, Math.max(0, word.startColumn - 1));

  return (completionItems.value || [])
    .filter((item) => {
      if (!item?.prefix) return true;
      const prefixes = Array.isArray(item.prefix) ? item.prefix : [item.prefix];
      return prefixes.some((prefix) => prefixText.endsWith(prefix));
    })
    .map((item) => ({
      label: item.label ?? item.insertText ?? "",
      kind: resolveCompletionKind(item.kind),
      detail: item.detail,
      documentation: item.documentation,
      insertText: item.insertText ?? item.label ?? "",
      range,
    }))
    .filter((item) => item.label && item.insertText);
};

const registerCompletionProvider = () => {
  if (completionProvider) {
    completionProvider.dispose();
    completionProvider = null;
  }
  if (!completionItems.value || completionItems.value.length === 0) return;
  completionProvider = monaco.languages.registerCompletionItemProvider(
    props.language,
    {
      triggerCharacters: [".", "$"],
      provideCompletionItems(model, position) {
        return { suggestions: resolveCompletionItems(model, position) };
      },
    }
  );
};

function initEditor() {
  if (!editorContainerRef.value) return;
  if (editorInstance) editorInstance.dispose();

  if (props.language === "css") {
    monaco.languages.css.cssDefaults.setOptions({
      validate: true,
      lint: {
        important: "warning",
        duplicateProperties: "warning",
        emptyRules: "warning",
        unknownProperties: "error",
      },
    });
  }
  if (props.language === "javascript" || props.language === "typescript") {
    const ignoreDiagnostics = [1003, 1108, 1308, 1375, 1378, 1379];
    monaco.languages.typescript.javascriptDefaults.setCompilerOptions({
      allowJs: true,
      allowNonTsExtensions: true,
      target: monaco.languages.typescript.ScriptTarget.ES2020,
      module: monaco.languages.typescript.ModuleKind.ESNext,
      lib: ["es2020", "dom"],
      moduleResolution: monaco.languages.typescript.ModuleResolutionKind.NodeJs,
      allowSyntheticDefaultImports: true,
      esModuleInterop: true,
      noEmit: true,
      checkJs: true,
    });
    monaco.languages.typescript.javascriptDefaults.setDiagnosticsOptions({
      noSemanticValidation: false,
      noSyntaxValidation: false,
      onlyVisible: false,
      diagnosticCodesToIgnore: ignoreDiagnostics,
    });
    monaco.languages.typescript.javascriptDefaults.setEagerModelSync(true);
    if (extraLibDisposable) {
      extraLibDisposable.dispose();
      extraLibDisposable = null;
    }
    extraLibDisposable = monaco.languages.typescript.javascriptDefaults.addExtraLib(
      "declare const $global: Record<string, any>;\n" +
        "declare const customScripts: Record<string, (...args: any[]) => any>;\n" +
        "declare const components: Record<string, any>;\n" +
        "declare const $event: any;\n",
      "ts:global-scripts.d.ts"
    );
  }

  editorInstance = monaco.editor.create(editorContainerRef.value, baseOptions());
  applyTheme(props.theme);
  registerCompletionProvider();
  const markerListener = monaco.editor.onDidChangeMarkers((uris) => {
    const model = editorInstance?.getModel();
    if (!model) return;
    const match = uris.some((uri) => uri.toString() === model.uri.toString());
    if (!match) return;
    const markers = monaco.editor.getModelMarkers({ resource: model.uri });
    latestMarkers = markers;
    emit("markers", markers);
  });
  cleanupFns.push(() => markerListener.dispose());
  requestAnimationFrame(() => {
    const model = editorInstance?.getModel();
    if (!model) return;
    const markers = monaco.editor.getModelMarkers({ resource: model.uri });
    latestMarkers = markers;
    emit("markers", markers);
  });

  editorInstance.onDidChangeModelContent(() => {
    if (isInternalUpdate) return;
    const val = editorInstance.getValue();
    isInternalUpdate = true;
    emit("update:modelValue", val);
    emit("change", val);
    requestAnimationFrame(() => (isInternalUpdate = false));
  });

  const mouseMoveListener = editorInstance.onMouseMove((event) => {
    if (!event?.target?.position) {
      hideMarkerTooltip();
      return;
    }
    showMarkerTooltip(event.target.position);
  });
  const mouseLeaveListener = editorInstance.onMouseLeave(() =>
    hideMarkerTooltip()
  );
  const scrollListener = editorInstance.onDidScrollChange(() =>
    hideMarkerTooltip()
  );
  cleanupFns.push(() => mouseMoveListener.dispose());
  cleanupFns.push(() => mouseLeaveListener.dispose());
  cleanupFns.push(() => scrollListener.dispose());
}

watch(
  () => props.modelValue,
  (val) => {
    if (!editorInstance) return;
    if (val === editorInstance.getValue()) return;
    isInternalUpdate = true;
    editorInstance.setValue(val || "");
    requestAnimationFrame(() => (isInternalUpdate = false));
  }
);

watch(
  () => props.language,
  (lang) => {
    if (editorInstance) {
      monaco.editor.setModelLanguage(editorInstance.getModel(), lang);
    }
  }
);

watch(
  () => props.theme,
  (theme) => {
    applyTheme(theme);
  }
);

watch(
  () => props.completions,
  (items) => {
    completionItems.value = items || [];
    registerCompletionProvider();
  },
  { deep: true }
);

watch(
  () => props.language,
  () => {
    registerCompletionProvider();
  }
);

function setupThemeListeners() {
  if (mediaQuery) {
    const handler = (event) => applyTheme(event.matches ? "vs-dark" : "vs");
    mediaQuery.addEventListener("change", handler);
    cleanupFns.push(() => mediaQuery.removeEventListener("change", handler));
  }

  if (typeof document !== "undefined") {
    const observer = new MutationObserver(() => applyTheme());
    const targetOptions = { attributes: true, attributeFilter: ["class", "data-theme"] };
    observer.observe(document.documentElement, targetOptions);
    if (document.body) observer.observe(document.body, targetOptions);
    cleanupFns.push(() => observer.disconnect());
  }
}

onMounted(() => {
  nextTick(() => {
    initEditor();
    setupThemeListeners();
  });
});

onBeforeUnmount(() => {
  if (editorInstance) {
    editorInstance.dispose();
    editorInstance = null;
  }
  if (completionProvider) {
    completionProvider.dispose();
    completionProvider = null;
  }
  if (markerTooltipEl) {
    markerTooltipEl.remove();
    markerTooltipEl = null;
  }
  cleanupFns.forEach((fn) => fn());
});

defineExpose({
  focus: () => editorInstance?.focus(),
  format: async () => {
    if (!editorInstance) return false;
    if (await runEditorAction("editor.action.formatDocument")) return true;
    if (await runEditorAction("editor.action.formatSelection")) return true;
    try {
      editorInstance.trigger("format", "editor.action.formatDocument");
      return true;
    } catch (error) {
      return false;
    }
  },
  getValue: () => editorInstance?.getValue() ?? "",
  setValue: (value) => {
    if (!editorInstance) return;
    isInternalUpdate = true;
    editorInstance.setValue(value || "");
    requestAnimationFrame(() => (isInternalUpdate = false));
  },
  insertText: (text) => {
    if (!editorInstance || !text) return;
    const selection = editorInstance.getSelection();
    const position = editorInstance.getPosition();
    const range =
      selection ||
      new monaco.Range(
        position.lineNumber,
        position.column,
        position.lineNumber,
        position.column
      );
    editorInstance.pushUndoStop();
    editorInstance.executeEdits("insert", [{ range, text, forceMoveMarkers: true }]);
    editorInstance.pushUndoStop();
    editorInstance.focus();
  },
});
</script>

<style scoped>
.monaco-editor-container {
  width: 100%;
  min-height: 200px;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  overflow: hidden;
}

.monaco-editor-container :deep(.monaco-editor) {
  width: 100% !important;
  height: 100% !important;
}
</style>
