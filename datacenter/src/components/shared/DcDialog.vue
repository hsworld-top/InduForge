<template>
  <el-dialog
    v-model="visible"
    class="dc-dialog"
    v-bind="$attrs"
    :width="width"
    :close-on-click-modal="closeOnClickModal"
    :append-to-body="appendToBody"
    :lock-scroll="lockScroll"
    :show-close="true"
    :before-close="handleBeforeClose"
    @close="handleClosed"
  >
    <!-- 自定义头部 -->
    <template v-if="$slots.header || title" #header>
      <slot name="header">
        <div v-if="title" class="dc-dialog__header">
          <span class="dc-dialog__title">{{ title }}</span>
        </div>
      </slot>
    </template>

    <!-- 主体内容，限高防溢出 -->
    <div class="dc-dialog__body" :style="{ '--dc-dialog-body-max': bodyMaxHeight }">
      <slot />
    </div>

    <!-- footer slot -->
    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

// 阻止 $attrs 自动透传到根元素，改由 v-bind="$attrs" 显式传给 el-dialog
defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title?: string
    width?: string | number
    closeOnClickModal?: boolean
    confirmOnDirtyClose?: boolean
    dirty?: boolean
    closeDisabled?: boolean
    dirtyCloseMessage?: string
    appendToBody?: boolean
    lockScroll?: boolean
    bodyMaxHeight?: string
  }>(),
  {
    width: '720px',
    closeOnClickModal: true,
    confirmOnDirtyClose: true,
    dirty: false,
    closeDisabled: false,
    dirtyCloseMessage: datacenterLocale.value === 'en' ? 'There are unsaved changes. Close anyway?' : '当前有未保存内容，确认关闭吗？',
    appendToBody: true,
    lockScroll: true,
    bodyMaxHeight: 'calc(100vh - 200px)',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  close: []
}>()

const closeConfirmed = ref(false)
const closeConfirming = ref(false)
const internalVisible = ref(props.modelValue)

// 关闭确认只取当前表单状态。保存成功或用户把内容改回初始值后，调用方会把
// dirty 重置为 false，此时不能因为“本次打开期间曾经修改过”继续误报未保存。
const needsDirtyConfirm = () => props.dirty

const confirmDirtyClose = async () => {
  if (!props.confirmOnDirtyClose || !needsDirtyConfirm()) {
    return true
  }

  return ElMessageBox.confirm(props.dirtyCloseMessage, ui('关闭确认', 'Confirm Close'), {
    confirmButtonText: ui('关闭', 'Close'),
    cancelButtonText: ui('继续编辑', 'Keep Editing'),
    type: 'warning',
  })
    .then(() => true)
    .catch(() => false)
}

const completeClose = (done?: () => void) => {
  closeConfirmed.value = true
  // 程序化关闭时也要同步内部可见状态，避免父组件 modelValue 已关闭但 el-dialog 仍停留。
  internalVisible.value = false

  if (done) {
    done()
  }
  emit('update:modelValue', false)
}

const requestClose = async (done?: () => void) => {
  if (props.closeDisabled || closeConfirming.value) return

  closeConfirming.value = true
  const confirmed = await confirmDirtyClose()
  closeConfirming.value = false
  if (!confirmed) return

  completeClose(done)
}

const closeSilently = () => {
  completeClose()
}

const visible = computed({
  get: () => internalVisible.value,
  set: (val: boolean) => {
    if (val) {
      internalVisible.value = true
      emit('update:modelValue', true)
      return
    }

    if (closeConfirmed.value) {
      internalVisible.value = false
      emit('update:modelValue', false)
      return
    }

    void requestClose()
  },
})

const handleBeforeClose = (done: () => void) => {
  if (closeConfirmed.value) {
    done()
    return
  }

  void requestClose(done)
}

watch(
  () => props.modelValue,
  (value, oldValue) => {
    if (value) {
      internalVisible.value = true
      closeConfirmed.value = false
      return
    }

    if (!oldValue && !internalVisible.value) return

    if (closeConfirmed.value) {
      internalVisible.value = false
      return
    }

    if (props.closeDisabled) {
      emit('update:modelValue', true)
      internalVisible.value = true
      return
    }

    if (props.confirmOnDirtyClose && needsDirtyConfirm()) {
      emit('update:modelValue', true)
      internalVisible.value = true
      void requestClose()
      return
    }

    internalVisible.value = false
  },
  { immediate: true },
)

const handleClosed = () => {
  closeConfirmed.value = false
  closeConfirming.value = false
  emit('close')
}

defineExpose({ requestClose, closeSilently })
</script>

<style scoped>
/* body 内层：高度撑满 el-dialog__body flex 容器，自身滚动 */
.dc-dialog__body {
  overflow: visible;
}

.dc-dialog__header {
  display: flex;
  align-items: center;
  height: 48px;
  padding: 0 4px;
}

.dc-dialog__title {
  color: var(--dc-text);
  font-size: 15px;
  font-weight: 700;
}
</style>

<style>
/* dialog 整体圆角与边框 */
.dc-dialog.el-dialog {
  /* 取消 el-dialog 默认 top:15vh，改用 margin 居中，同时限制最大高度 */
  top: 0 !important;
  margin: 5vh auto !important;
  max-height: calc(90vh) !important;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
}

/* header/footer 不压缩，body 可伸缩 */
.dc-dialog .el-dialog__header {
  flex-shrink: 0;
}

.dc-dialog .el-dialog__body {
  flex: 0 1 auto;
  min-height: 0;
  max-height: calc(90vh - 132px);
  overflow-y: auto;
  padding-top: 8px;
  padding-bottom: 16px;
}

.dc-dialog .el-dialog__footer {
  flex-shrink: 0;
}
</style>
