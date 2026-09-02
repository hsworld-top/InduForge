<template>
  <div
    ref="editorContainerRef"
    class="monaco-editor-container"
    :style="{ height: height, width: '100%' }"
    @contextmenu.prevent="openEditorContextMenu"
  ></div>
  <div
    v-if="editorContextMenu.visible"
    class="monaco-editor-context-menu"
    :style="{ left: `${editorContextMenu.x}px`, top: `${editorContextMenu.y}px` }"
    @mousedown.stop
    @contextmenu.prevent
  >
    <button type="button" :disabled="!selectedText" @click="copySelection">{{ ui('复制', 'Copy') }}</button>
    <button
      type="button"
      :disabled="!selectedText || !editorContextMenu.canModify"
      @click="cutSelection"
    >
      {{ ui('剪切', 'Cut') }}
    </button>
    <button type="button" :disabled="!editorContextMenu.canModify" @click="pasteFromClipboard">
      {{ ui('粘贴', 'Paste') }}
    </button>
    <button type="button" @click="formatFromContextMenu">{{ ui('格式化文档', 'Format Document') }}</button>
  </div>
</template>

<script setup lang="ts">
import { secureRandomUUID } from '@/utils/secure-random-uuid'
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { debugLogger } from '@/utils/debug'
import * as monaco from 'monaco-editor'
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import JsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import TypeScriptWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

type EditorDiagnostic = {
  severity?: string
  message: string
  line: number
  column: number
  endLine?: number
  endColumn?: number
  source?: string
}

type CursorChangePayload = {
  line: number
  column: number
  spaces: number
}

type MemberCompletion = {
  label: string
  insertText?: string
  kind?: 'method' | 'property'
  detail?: string
  documentation?: string
}

// 配置 Monaco 官方 worker，恢复 JSON 等语言服务的格式化、校验能力。
if (typeof window !== 'undefined' && !window.MonacoEnvironment) {
  window.MonacoEnvironment = {
    getWorker(_moduleId, label) {
      if (label === 'json') {
        return new JsonWorker()
      }
      if (label === 'typescript' || label === 'javascript') {
        return new TypeScriptWorker()
      }
      return new EditorWorker()
    },
  }
}

const props = defineProps({
  modelValue: {
    type: String,
    default: '',
  },
  language: {
    type: String,
    default: 'sql',
  },
  theme: {
    type: String,
    default: 'vs-dark', // 'vs', 'vs-dark', 'hc-black'
  },
  height: {
    type: String,
    default: '100%',
  },
  options: {
    type: Object,
    default: () => ({}),
  },
  readOnly: {
    type: Boolean,
    default: false,
  },
  typeDefinitions: {
    type: String,
    default: '',
  },
  memberCompletions: {
    type: Object as () => Record<string, MemberCompletion[]>,
    default: () => ({}),
  },
})

const emit = defineEmits(['update:modelValue', 'change', 'cursor-change', 'save'])

const editorContainerRef = ref(null)
let editorInstance = null
let isInternalUpdate = false
let typeDefinitionsDisposable: monaco.IDisposable | null = null
let completionProviderDisposable: monaco.IDisposable | null = null
const typeDefinitionsPath = `file:///induforge/editor-${secureRandomUUID()}.d.ts`
const selectedText = ref('')
const editorContextMenu = ref({
  visible: false,
  x: 0,
  y: 0,
  canModify: true,
})

const emitCursorChange = () => {
  if (!editorInstance) return
  const position = editorInstance.getPosition()
  if (!position) return
  const tabSize = Number(editorInstance.getModel()?.getOptions()?.tabSize ?? 2)
  const payload: CursorChangePayload = {
    line: position.lineNumber,
    column: position.column,
    spaces: tabSize,
  }
  emit('cursor-change', payload)
}

const closeEditorContextMenu = () => {
  editorContextMenu.value.visible = false
}

const updateSelectedText = () => {
  const model = editorInstance?.getModel?.()
  const selection = editorInstance?.getSelection?.()
  if (!model || !selection || selection.isEmpty()) {
    selectedText.value = ''
    return
  }
  selectedText.value = model.getValueInRange(selection)
}

const moveCursorToContextTarget = (event?: any) => {
  if (!editorInstance) return
  const selection = editorInstance.getSelection()
  if (selection && !selection.isEmpty()) return

  const position =
    event?.target?.position ||
    editorInstance.getTargetAtClientPoint?.(
      event?.event?.browserEvent?.clientX ?? event?.clientX ?? 0,
      event?.event?.browserEvent?.clientY ?? event?.clientY ?? 0,
    )?.position
  if (position) {
    editorInstance.setPosition(position)
  }
}

const openEditorContextMenu = (event?: any) => {
  const browserEvent = event?.event?.browserEvent || event?.browserEvent || event
  browserEvent?.preventDefault?.()
  browserEvent?.stopPropagation?.()

  moveCursorToContextTarget(event)
  updateSelectedText()
  editorContextMenu.value = {
    visible: true,
    x: Number(browserEvent?.clientX) || 0,
    y: Number(browserEvent?.clientY) || 0,
    canModify: !editorInstance?.getOption?.(monaco.editor.EditorOption.readOnly),
  }
}

const copySelection = async () => {
  const text = selectedText.value
  if (!text) return
  try {
    await navigator.clipboard?.writeText(text)
  } finally {
    closeEditorContextMenu()
    editorInstance?.focus?.()
  }
}

const cutSelection = async () => {
  const model = editorInstance?.getModel?.()
  const selection = editorInstance?.getSelection?.()
  if (!editorInstance || !model || !selection || selection.isEmpty()) return

  try {
    await navigator.clipboard?.writeText(model.getValueInRange(selection))
    editorInstance.executeEdits('context-cut', [{ range: selection, text: '' }])
  } finally {
    closeEditorContextMenu()
    editorInstance?.focus?.()
  }
}

const pasteFromClipboard = async () => {
  if (!editorInstance || editorContextMenu.value.canModify === false) return

  try {
    const text = await navigator.clipboard?.readText?.()
    if (!text) return

    const selection = editorInstance.getSelection()
    const position = editorInstance.getPosition()
    const range =
      selection ||
      new monaco.Range(
        position?.lineNumber || 1,
        position?.column || 1,
        position?.lineNumber || 1,
        position?.column || 1,
      )
    editorInstance.executeEdits('context-paste', [{ range, text }])
  } finally {
    closeEditorContextMenu()
    editorInstance?.focus?.()
  }
}

const formatFromContextMenu = async () => {
  await formatCode()
  closeEditorContextMenu()
  editorInstance?.focus?.()
}

// 获取编辑器选项
const getEditorOptions = () => {
  return {
    value: props.modelValue,
    language: props.language,
    theme: props.theme,
    readOnly: props.readOnly,
    automaticLayout: true,
    fontSize: 14,
    fixedOverflowWidgets: true,
    hover: {
      above: false,
      delay: 180,
      sticky: true,
      hidingDelay: 80,
    },
    lightbulb: {
      enabled: monaco.editor.ShowLightbulbIconMode.Off,
    },
    minimap: {
      enabled: true,
    },
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    contextmenu: false,
    formatOnPaste: true,
    formatOnType: true,
    ...props.options,
  }
}

// 初始化编辑器
const initEditor = () => {
  if (!editorContainerRef.value) {
    debugLogger.warn('Monaco Editor: 容器元素不存在')
    return
  }

  // 如果编辑器已存在，先销毁
  if (editorInstance) {
    try {
      editorInstance.dispose()
    } catch (e) {
      debugLogger.warn('销毁旧编辑器失败:', e)
    }
    editorInstance = null
  }

  try {
    // 创建编辑器实例
    const options = getEditorOptions()
    editorInstance = monaco.editor.create(editorContainerRef.value, options)
    updateLanguageServices()

    // 在编辑器聚焦时接管保存快捷键，避免浏览器触发“保存网页”。
    editorInstance.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      emit('save')
    })

    // 监听内容变化
    editorInstance.onDidChangeModelContent(() => {
      if (!isInternalUpdate && editorInstance) {
        const value = editorInstance.getValue()
        isInternalUpdate = true
        emit('update:modelValue', value)
        emit('change', value)
        // 重置标志
        setTimeout(() => {
          isInternalUpdate = false
        }, 0)
      }
      emitCursorChange()
    })

    editorInstance.onDidChangeCursorPosition(() => {
      emitCursorChange()
      closeEditorContextMenu()
    })

    editorInstance.onDidChangeCursorSelection(() => {
      updateSelectedText()
    })

    editorInstance.onContextMenu((event) => {
      openEditorContextMenu(event)
    })

    // Monaco 创建时已经传入 value；这里只处理被 options 覆盖后的补偿设置，避免初始化触发 change。
    if (props.modelValue && editorInstance.getValue() !== props.modelValue) {
      isInternalUpdate = true
      editorInstance.setValue(props.modelValue)
      setTimeout(() => {
        isInternalUpdate = false
      }, 0)
    }

    emitCursorChange()

    // 确保编辑器可聚焦
    setTimeout(() => {
      if (editorInstance) {
        editorInstance.focus()
      }
    }, 100)
  } catch (error) {
    debugLogger.error('Monaco Editor 创建失败:', error)
    throw error
  }
}

// 更新编辑器主题
const updateTheme = () => {
  if (editorInstance) {
    monaco.editor.setTheme(props.theme)
  }
}

// 更新编辑器语言
const updateLanguage = () => {
  if (editorInstance) {
    monaco.editor.setModelLanguage(editorInstance.getModel(), props.language)
    updateLanguageServices()
  }
}

const completionKind = (kind?: MemberCompletion['kind']) =>
  kind === 'property'
    ? monaco.languages.CompletionItemKind.Property
    : monaco.languages.CompletionItemKind.Method

const updateLanguageServices = () => {
  typeDefinitionsDisposable?.dispose()
  typeDefinitionsDisposable = null
  completionProviderDisposable?.dispose()
  completionProviderDisposable = null

  if (props.typeDefinitions.trim()) {
    const defaults =
      props.language === 'typescript'
        ? monaco.languages.typescript.typescriptDefaults
        : monaco.languages.typescript.javascriptDefaults
    typeDefinitionsDisposable = defaults.addExtraLib(props.typeDefinitions, typeDefinitionsPath)
  }

  if (!Object.keys(props.memberCompletions).length) return
  completionProviderDisposable = monaco.languages.registerCompletionItemProvider(props.language, {
    triggerCharacters: ['.'],
    provideCompletionItems(model, position) {
      if (model !== editorInstance?.getModel()) return { suggestions: [] }
      const prefix = model.getLineContent(position.lineNumber).slice(0, position.column - 1)
      const receiver = prefix.match(/([A-Za-z_][A-Za-z0-9_]*)\.[A-Za-z0-9_]*$/)?.[1]
      const items = receiver ? props.memberCompletions[receiver] || [] : []
      const word = model.getWordUntilPosition(position)
      const range = new monaco.Range(
        position.lineNumber,
        word.startColumn,
        position.lineNumber,
        word.endColumn,
      )
      return {
        suggestions: items.map((item) => ({
          label: item.label,
          insertText: item.insertText || item.label,
          kind: completionKind(item.kind),
          detail: item.detail,
          documentation: item.documentation,
          range,
        })),
      }
    },
  })
}

// 格式化代码
const formatCode = () => {
  if (editorInstance) {
    return editorInstance.getAction('editor.action.formatDocument')?.run()
  }
  return Promise.resolve()
}

const insertText = (text) => {
  if (!editorInstance) return
  const selection = editorInstance.getSelection()
  const range =
    selection ||
    new monaco.Range(
      editorInstance.getPosition()?.lineNumber || 1,
      editorInstance.getPosition()?.column || 1,
      editorInstance.getPosition()?.lineNumber || 1,
      editorInstance.getPosition()?.column || 1,
    )
  editorInstance.executeEdits('insert-text', [{ range, text }])
  editorInstance.focus()
}

const severityToMonaco = (severity?: string) => {
  switch (String(severity || '').toLowerCase()) {
    case 'warning':
    case 'warn':
      return monaco.MarkerSeverity.Warning
    case 'info':
      return monaco.MarkerSeverity.Info
    case 'hint':
      return monaco.MarkerSeverity.Hint
    default:
      return monaco.MarkerSeverity.Error
  }
}

const setDiagnostics = (diagnostics: EditorDiagnostic[] = [], owner = 'compute-syntax') => {
  const model = editorInstance?.getModel?.()
  if (!model) return
  monaco.editor.setModelMarkers(
    model,
    owner,
    diagnostics.map((item) => ({
      severity: severityToMonaco(item.severity),
      message: item.message,
      startLineNumber: Math.max(1, Number(item.line) || 1),
      startColumn: Math.max(1, Number(item.column) || 1),
      endLineNumber: Math.max(1, Number(item.endLine || item.line) || Number(item.line) || 1),
      endColumn: Math.max(
        2,
        Number(item.endColumn || item.column + 1) || Number(item.column || 1) + 1,
      ),
      source: item.source || owner,
    })),
  )
}

const revealPosition = (line: number, column = 1) => {
  if (!editorInstance) return
  const position = {
    lineNumber: Math.max(1, Number(line) || 1),
    column: Math.max(1, Number(column) || 1),
  }
  editorInstance.setPosition(position)
  editorInstance.revealPositionInCenter(position)
  editorInstance.focus()
}

const revealEnd = () => {
  const model = editorInstance?.getModel?.()
  if (!editorInstance || !model) return false
  const line = model.getLineCount()
  const column = model.getLineMaxColumn(line)
  editorInstance.setPosition({ lineNumber: line, column })
  editorInstance.revealPositionInCenter({ lineNumber: line, column })
  editorInstance.focus()
  return true
}

// 暴露方法给父组件
defineExpose({
  getValue: () => editorInstance?.getValue() || '',
  setValue: (value) => editorInstance?.setValue(value || ''),
  insertText,
  setDiagnostics,
  revealPosition,
  revealEnd,
  format: formatCode,
  focus: () => editorInstance?.focus(),
  dispose: () => {
    typeDefinitionsDisposable?.dispose()
    completionProviderDisposable?.dispose()
    if (editorInstance) {
      editorInstance.dispose()
      editorInstance = null
    }
  },
})

// 监听 props 变化 - 使用标志防止循环更新
watch(
  () => props.modelValue,
  (newValue) => {
    if (editorInstance && !isInternalUpdate) {
      const currentValue = editorInstance.getValue()
      if (currentValue !== newValue) {
        isInternalUpdate = true
        editorInstance.setValue(newValue || '')
        emitCursorChange()
        // 重置标志
        setTimeout(() => {
          isInternalUpdate = false
        }, 0)
      }
    }
  },
)

watch(
  () => props.theme,
  () => {
    updateTheme()
  },
)

watch(
  () => props.language,
  () => {
    updateLanguage()
  },
)

watch(
  () => props.readOnly,
  (readOnly) => {
    if (editorInstance) {
      editorInstance.updateOptions({ readOnly })
    }
  },
)

watch(
  () => props.typeDefinitions,
  () => updateLanguageServices(),
)

watch(
  () => props.memberCompletions,
  () => updateLanguageServices(),
  { deep: true },
)

// 组件挂载时初始化
onMounted(() => {
  window.addEventListener('mousedown', closeEditorContextMenu)
  window.addEventListener('blur', closeEditorContextMenu)

  // 使用多个 nextTick 确保 DOM 完全渲染
  nextTick(() => {
    nextTick(() => {
      const checkAndInit = (retryCount = 0) => {
        if (retryCount > 30) {
          debugLogger.error('Monaco Editor 初始化超时，已重试30次', {
            containerRef: editorContainerRef.value,
            containerExists: !!editorContainerRef.value,
          })
          return
        }

        // 如果 ref 还没有绑定，等待一下
        if (!editorContainerRef.value) {
          debugLogger.warn(`Monaco Editor 容器元素不存在，等待... (${retryCount + 1}/30)`)
          setTimeout(() => checkAndInit(retryCount + 1), 100)
          return
        }

        const container = editorContainerRef.value
        const containerHeight = container.offsetHeight
        const clientHeight = container.clientHeight

        // 如果容器有高度，或者已经重试多次，就初始化
        if (containerHeight > 0 || clientHeight > 0 || retryCount > 10) {
          try {
            initEditor()
            // 初始化后仅聚焦；只读状态始终遵循调用方配置。
            setTimeout(() => {
              if (editorInstance) {
                editorInstance.focus()
              }
            }, 200)
          } catch (error) {
            debugLogger.error('Monaco Editor 初始化失败:', error)
            // 如果初始化失败，等待一下再试
            setTimeout(() => checkAndInit(retryCount + 1), 100)
          }
        } else {
          // 如果容器高度为0，等待一下再试
          setTimeout(() => checkAndInit(retryCount + 1), 100)
        }
      }
      checkAndInit()
    })
  })
})

// 组件卸载时清理
onBeforeUnmount(() => {
  window.removeEventListener('mousedown', closeEditorContextMenu)
  window.removeEventListener('blur', closeEditorContextMenu)

  typeDefinitionsDisposable?.dispose()
  completionProviderDisposable?.dispose()
  if (editorInstance) {
    editorInstance.dispose()
    editorInstance = null
  }
})
</script>

<style scoped>
.monaco-editor-container {
  width: 100%;
  min-height: 200px;
  display: block;
  position: relative;
}

/* 确保 Monaco Editor 的样式正确应用 */
.monaco-editor-container :deep(.monaco-editor) {
  width: 100% !important;
  height: 100% !important;
}

.monaco-editor-container :deep(.monaco-editor .monaco-editor-background) {
  background-color: var(--vscode-editor-background, #ffffff);
}

.monaco-editor-container :deep(.monaco-editor .margin) {
  background-color: var(--vscode-editor-background, #ffffff);
}

.monaco-editor-container :deep(.monaco-hover .hover-row.status-bar) {
  display: none;
}

.monaco-editor-container :deep(.monaco-hover .hover-row:has(.markdown-hover)) {
  display: none;
}

.monaco-editor-container :deep(.monaco-hover) {
  max-width: min(520px, calc(100vw - 48px));
}

.monaco-editor-context-menu {
  position: fixed;
  z-index: 3000;
  min-width: 128px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: 6px;
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-lg);
}

.monaco-editor-context-menu button {
  display: block;
  width: 100%;
  padding: 6px 10px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--dc-text);
  font-size: 13px;
  line-height: 20px;
  text-align: left;
  cursor: pointer;
}

.monaco-editor-context-menu button:hover:not(:disabled) {
  background: var(--dc-surface-muted);
}

.monaco-editor-context-menu button:disabled {
  color: #a8abb2;
  cursor: not-allowed;
}
</style>
