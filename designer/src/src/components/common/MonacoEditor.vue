<template>
    <div ref="editorContainerRef" class="monaco-editor-container" :style="{ height }"></div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue';
import * as monaco from 'monaco-editor';
import 'monaco-editor/min/vs/editor/editor.main.css';
import 'monaco-editor/esm/vs/basic-languages/css/css.contribution';
import 'monaco-editor/esm/vs/basic-languages/javascript/javascript.contribution';
import 'monaco-editor/esm/vs/basic-languages/html/html.contribution';
import 'monaco-editor/esm/vs/language/css/monaco.contribution';
import 'monaco-editor/esm/vs/language/html/monaco.contribution';
import 'monaco-editor/esm/vs/language/json/monaco.contribution';
import 'monaco-editor/esm/vs/language/typescript/monaco.contribution';

const props = defineProps({
    modelValue: { type: String, default: '' },
    language: { type: String, default: 'css' },
    theme: { type: String, default: 'vs' }, // 'light' | 'dark' | 'vs' | 'vs-dark'
    height: { type: String, default: '300px' },
    options: { type: Object, default: () => ({}) },
});

const emit = defineEmits(['update:modelValue', 'change']);

const editorContainerRef = ref(null);
let editorInstance = null;
let isInternalUpdate = false;

const mediaQuery = typeof window !== 'undefined' && window.matchMedia ? window.matchMedia('(prefers-color-scheme: dark)') : null;
const cleanupFns = [];

const normalizeTheme = (theme) => {
    if (theme === 'dark') return 'vs-dark';
    if (theme === 'light') return 'vs';
    return theme || 'vs';
};

const detectTheme = () => {
    if (typeof document !== 'undefined') {
        const html = document.documentElement;
        const body = document.body;
        const isDark = (el) => el && (el.classList?.contains('dark') || el.dataset?.theme === 'dark');
        if (isDark(html) || isDark(body)) return 'vs-dark';
    }
    if (mediaQuery) return mediaQuery.matches ? 'vs-dark' : 'vs';
    return 'vs';
};

function registerMonacoEnvironment() {
    if (typeof self === 'undefined' || (self.MonacoEnvironment && self.MonacoEnvironment.getWorker)) return;

    const workerMap = {
        css: 'monaco-editor/esm/vs/language/css/css.worker?worker',
        scss: 'monaco-editor/esm/vs/language/css/css.worker?worker',
        less: 'monaco-editor/esm/vs/language/css/css.worker?worker',
        html: 'monaco-editor/esm/vs/language/html/html.worker?worker',
        json: 'monaco-editor/esm/vs/language/json/json.worker?worker',
        javascript: 'monaco-editor/esm/vs/language/typescript/ts.worker?worker',
        typescript: 'monaco-editor/esm/vs/language/typescript/ts.worker?worker',
        default: 'monaco-editor/esm/vs/editor/editor.worker?worker',
    };

    const createWorker = (label) => {
        const workerPath = workerMap[label] || workerMap.default;
        try {
            return new Worker(new URL(workerPath, import.meta.url), { type: 'module' });
        } catch (error) {
            const fallback = new Blob(['self.onmessage = () => {};'], { type: 'application/javascript' });
            return new Worker(URL.createObjectURL(fallback));
        }
    };

    self.MonacoEnvironment = {
        getWorker(_, label) {
            return createWorker(label);
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
    wordWrap: 'on',
    formatOnPaste: true,
    formatOnType: true,
    folding: true,
    glyphMargin: true,
    lineNumbers: 'on',
    tabSize: 2,
    suggestOnTriggerCharacters: true,
    quickSuggestions: true,
    autoClosingBrackets: 'always',
    autoClosingQuotes: 'always',
    ...props.options,
});

function applyTheme(theme) {
    monaco.editor.setTheme(normalizeTheme(theme || detectTheme()));
}

function initEditor() {
    if (!editorContainerRef.value) return;
    if (editorInstance) editorInstance.dispose();

    if (props.language === 'css') {
        monaco.languages.css.cssDefaults.setOptions({
            validate: true,
            lint: {
                important: 'warning',
                duplicateProperties: 'warning',
                emptyRules: 'warning',
                unknownProperties: 'error',
            },
        });
    }

    editorInstance = monaco.editor.create(editorContainerRef.value, baseOptions());
    applyTheme(props.theme);

    editorInstance.onDidChangeModelContent(() => {
        if (isInternalUpdate) return;
        const val = editorInstance.getValue();
        isInternalUpdate = true;
        emit('update:modelValue', val);
        emit('change', val);
        requestAnimationFrame(() => (isInternalUpdate = false));
    });
}

watch(
    () => props.modelValue,
    (val) => {
        if (editorInstance && !isInternalUpdate && val !== editorInstance.getValue()) {
            isInternalUpdate = true;
            editorInstance.setValue(val || '');
            requestAnimationFrame(() => (isInternalUpdate = false));
        }
    },
);

watch(
    () => props.language,
    (lang) => {
        if (editorInstance) {
            monaco.editor.setModelLanguage(editorInstance.getModel(), lang);
        }
    },
);

watch(
    () => props.theme,
    (theme) => {
        applyTheme(theme);
    },
);

function setupThemeListeners() {
    if (mediaQuery) {
        const handler = (e) => applyTheme(e.matches ? 'vs-dark' : 'vs');
        mediaQuery.addEventListener('change', handler);
        cleanupFns.push(() => mediaQuery.removeEventListener('change', handler));
    }

    if (typeof document !== 'undefined') {
        const observer = new MutationObserver(() => applyTheme());
        const targetOptions = { attributes: true, attributeFilter: ['class', 'data-theme'] };
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
    cleanupFns.forEach((fn) => fn());
});

defineExpose({
    focus: () => editorInstance?.focus(),
    format: () => editorInstance?.getAction('editor.action.formatDocument')?.run(),
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
