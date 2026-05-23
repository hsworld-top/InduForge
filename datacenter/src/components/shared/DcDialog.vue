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
    <div
      class="dc-dialog__body"
      :style="{ '--dc-dialog-body-max': bodyMaxHeight }"
    >
      <slot />
    </div>

    <!-- footer slot -->
    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { ElMessageBox } from "element-plus";

// 阻止 $attrs 自动透传到根元素，改由 v-bind="$attrs" 显式传给 el-dialog
defineOptions({ inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    title?: string;
    width?: string | number;
    closeOnClickModal?: boolean;
    confirmOnDirtyClose?: boolean;
    dirty?: boolean;
    closeDisabled?: boolean;
    dirtyCloseMessage?: string;
    appendToBody?: boolean;
    lockScroll?: boolean;
    bodyMaxHeight?: string;
  }>(),
  {
    width: "720px",
    closeOnClickModal: true,
    confirmOnDirtyClose: true,
    dirty: false,
    closeDisabled: false,
    dirtyCloseMessage: "当前有未保存内容，确认关闭吗？",
    appendToBody: true,
    lockScroll: true,
    bodyMaxHeight: "calc(100vh - 200px)",
  },
);

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  close: [];
}>();

const closeConfirmed = ref(false);
const closeConfirming = ref(false);
const internalVisible = ref(props.modelValue);
const dirtyWhileOpen = ref(false);

const needsDirtyConfirm = () => props.dirty || dirtyWhileOpen.value;

const confirmDirtyClose = async () => {
  if (!props.confirmOnDirtyClose || !needsDirtyConfirm()) {
    return true;
  }

  return ElMessageBox.confirm(
    props.dirtyCloseMessage,
    "关闭确认",
    {
      confirmButtonText: "关闭",
      cancelButtonText: "继续编辑",
      type: "warning",
    },
  )
    .then(() => true)
    .catch(() => false);
};

const completeClose = (done?: () => void) => {
  closeConfirmed.value = true;
  dirtyWhileOpen.value = false;

  if (done) {
    done();
  }
  emit("update:modelValue", false);
};

const requestClose = async (done?: () => void) => {
  if (props.closeDisabled || closeConfirming.value) return;

  closeConfirming.value = true;
  const confirmed = await confirmDirtyClose();
  closeConfirming.value = false;
  if (!confirmed) return;

  completeClose(done);
};

const visible = computed({
  get: () => internalVisible.value,
  set: (val: boolean) => {
    if (val) {
      internalVisible.value = true;
      emit("update:modelValue", true);
      return;
    }

    if (closeConfirmed.value) {
      internalVisible.value = false;
      emit("update:modelValue", false);
      return;
    }

    void requestClose();
  },
});

const handleBeforeClose = (done: () => void) => {
  if (closeConfirmed.value) {
    done();
    return;
  }

  void requestClose(done);
};

watch(
  () => props.dirty,
  (dirty) => {
    if (internalVisible.value && dirty) {
      dirtyWhileOpen.value = true;
    }
  },
);

watch(
  () => props.modelValue,
  (value, oldValue) => {
    if (value) {
      internalVisible.value = true;
      closeConfirmed.value = false;
      dirtyWhileOpen.value = props.dirty;
      return;
    }

    if (!oldValue && !internalVisible.value) return;

    if (closeConfirmed.value) {
      internalVisible.value = false;
      dirtyWhileOpen.value = false;
      return;
    }

    if (props.closeDisabled) {
      emit("update:modelValue", true);
      internalVisible.value = true;
      return;
    }

    if (props.confirmOnDirtyClose && needsDirtyConfirm()) {
      emit("update:modelValue", true);
      internalVisible.value = true;
      void requestClose();
      return;
    }

    internalVisible.value = false;
    dirtyWhileOpen.value = false;
  },
  { immediate: true },
);

const handleClosed = () => {
  closeConfirmed.value = false;
  closeConfirming.value = false;
  dirtyWhileOpen.value = false;
  emit("close");
};

defineExpose({ requestClose });
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
