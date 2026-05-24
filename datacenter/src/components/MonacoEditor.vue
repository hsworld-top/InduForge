<template>
  <div
    ref="editorContainerRef"
    class="monaco-editor-container"
    :style="{ height: height, width: '100%' }"
  ></div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import * as monaco from 'monaco-editor'

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

// 配置 Monaco Editor 的 worker
// 使用内联 worker 避免 worker 文件加载问题
if (typeof window !== 'undefined' && !window.MonacoEnvironment) {
  window.MonacoEnvironment = {
    getWorker: function (moduleId, label) {
      // 创建一个简单的内联 worker，避免 worker 文件加载问题
      // 这不会影响基本的编辑功能，只是某些高级功能可能不可用
      const createInlineWorker = (code) => {
        const blob = new Blob([code], { type: 'application/javascript' })
        return new Worker(URL.createObjectURL(blob))
      }

      // 返回一个简单的 worker，只处理基本消息
      return createInlineWorker(`
        self.onmessage = function(e) {
          // 简单的 worker，不做任何处理，只是避免错误
          if (e.data && e.data.$type === 'ping') {
            self.postMessage({ $type: 'pong' })
          }
        }
      `)
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
})

const emit = defineEmits(['update:modelValue', 'change', 'cursor-change', 'save'])

const editorContainerRef = ref(null)
let editorInstance = null
let isInternalUpdate = false

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
    formatOnPaste: true,
    formatOnType: true,
    ...props.options,
  }
}

// 初始化编辑器
const initEditor = () => {
  if (!editorContainerRef.value) {
    console.warn('Monaco Editor: 容器元素不存在')
    return
  }

  // 如果编辑器已存在，先销毁
  if (editorInstance) {
    try {
      editorInstance.dispose()
    } catch (e) {
      console.warn('销毁旧编辑器失败:', e)
    }
    editorInstance = null
  }

  try {
    // 创建编辑器实例
    const options = getEditorOptions()
    // 强制设置 readOnly 为 false，确保编辑器可编辑
    options.readOnly = false
    editorInstance = monaco.editor.create(editorContainerRef.value, options)

    // 在编辑器聚焦时接管保存快捷键，避免浏览器触发“保存网页”。
    editorInstance.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      emit('save')
    })

    console.log('Monaco Editor 创建成功', {
      container: editorContainerRef.value,
      containerHeight: editorContainerRef.value.offsetHeight,
      containerClientHeight: editorContainerRef.value.clientHeight,
      options: options,
      readOnly: editorInstance.getOption(monaco.editor.EditorOption.readOnly),
      editorInstance: editorInstance,
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
    console.error('Monaco Editor 创建失败:', error)
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
  }
}

// 格式化代码
const formatCode = () => {
  if (editorInstance) {
    editorInstance.getAction('editor.action.formatDocument').run()
  }
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

// 暴露方法给父组件
defineExpose({
  getValue: () => editorInstance?.getValue() || '',
  setValue: (value) => editorInstance?.setValue(value || ''),
  insertText,
  setDiagnostics,
  revealPosition,
  format: formatCode,
  focus: () => editorInstance?.focus(),
  dispose: () => {
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

// 组件挂载时初始化
onMounted(() => {
  console.log('MonacoEditor onMounted 被调用', {
    containerRef: editorContainerRef.value,
    height: props.height,
    containerExists: !!editorContainerRef.value,
  })

  // 使用多个 nextTick 确保 DOM 完全渲染
  nextTick(() => {
    nextTick(() => {
      console.log('MonacoEditor nextTick 执行', {
        containerRef: editorContainerRef.value,
        containerHeight: editorContainerRef.value?.offsetHeight,
        containerClientHeight: editorContainerRef.value?.clientHeight,
        containerExists: !!editorContainerRef.value,
      })

      const checkAndInit = (retryCount = 0) => {
        if (retryCount > 30) {
          console.error('Monaco Editor 初始化超时，已重试30次', {
            containerRef: editorContainerRef.value,
            containerExists: !!editorContainerRef.value,
          })
          return
        }

        // 如果 ref 还没有绑定，等待一下
        if (!editorContainerRef.value) {
          console.warn(`Monaco Editor 容器元素不存在，等待... (${retryCount + 1}/30)`)
          setTimeout(() => checkAndInit(retryCount + 1), 100)
          return
        }

        const container = editorContainerRef.value
        const containerHeight = container.offsetHeight
        const clientHeight = container.clientHeight

        console.log(`Monaco Editor 检查初始化 (尝试 ${retryCount + 1})`, {
          containerHeight,
          clientHeight,
          height: props.height,
          hasHeight: containerHeight > 0 || clientHeight > 0,
          container: container,
          containerTagName: container.tagName,
          containerClassName: container.className,
        })

        // 如果容器有高度，或者已经重试多次，就初始化
        if (containerHeight > 0 || clientHeight > 0 || retryCount > 10) {
          try {
            initEditor()
            console.log('Monaco Editor 初始化成功', {
              containerHeight,
              clientHeight,
              height: props.height,
              readOnly: getEditorOptions().readOnly,
            })

            // 确保编辑器可以聚焦和编辑
            setTimeout(() => {
              if (editorInstance) {
                editorInstance.updateOptions({ readOnly: false })
                editorInstance.focus()
                console.log('Monaco Editor 已设置为可编辑并聚焦')
              }
            }, 200)
          } catch (error) {
            console.error('Monaco Editor 初始化失败:', error)
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
</style>
