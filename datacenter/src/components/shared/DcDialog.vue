<template>
  <el-dialog
    v-model="visible"
    class="dc-dialog"
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
/* body 限高 + 内部滚动，防止 dialog 内容撑出视口 */
.dc-dialog__body {
  max-height: var(--dc-dialog-body-max, calc(100vh - 200px));
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
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
}
</style>
