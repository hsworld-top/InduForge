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
    @close="emit('close')"
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
import { computed } from "vue";

// 阻止 $attrs 自动透传到根元素，改由 v-bind="$attrs" 显式传给 el-dialog
defineOptions({ inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    title?: string;
    width?: string | number;
    closeOnClickModal?: boolean;
    appendToBody?: boolean;
    lockScroll?: boolean;
    bodyMaxHeight?: string;
  }>(),
  {
    width: "720px",
    closeOnClickModal: false,
    appendToBody: true,
    lockScroll: true,
    bodyMaxHeight: "calc(100vh - 200px)",
  },
);

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  close: [];
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (val: boolean) => emit("update:modelValue", val),
});
</script>

<style scoped>
/* body 内层：高度撑满 el-dialog__body flex 容器，自身滚动 */
.dc-dialog__body {
  height: 100%;
  max-height: var(--dc-dialog-body-max, none);
  overflow-y: auto;
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
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding-top: 8px;
  padding-bottom: 8px;
}

.dc-dialog .el-dialog__footer {
  flex-shrink: 0;
}
</style>
