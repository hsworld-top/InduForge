<template>
  <div class="ui-el-dialog" :style="mergedStyle">
    <el-dialog
      :model-value="modelValue"
      :title="title"
      :width="width"
      :modal="modal"
      :append-to-body="appendToBody"
      :close-on-click-modal="closeOnClickModal"
      :show-close="showClose"
      :draggable="draggable"
      :style="dialogStyle"
      v-bind="listeners"
      :lock-scroll="false"
      @open="() => emit('open')"
      @close="() => emit('close')"
    >
      <slot>{{ content }}</slot>
      <template #footer>
        <slot name="footer">
          <div style="text-align: right;">
            <el-button size="small">{{ cancelText }}</el-button>
            <el-button size="small" type="primary">{{ confirmText }}</el-button>
          </div>
        </slot>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElDialog' });

const props = defineProps({
  modelValue: { type: Boolean, default: true },
  title: { type: String, default: '对话框' },
  width: { type: String, default: '40%' },
  modal: { type: Boolean, default: false },
  appendToBody: { type: Boolean, default: false },
  closeOnClickModal: { type: Boolean, default: false },
  showClose: { type: Boolean, default: false },
  draggable: { type: Boolean, default: true },
  content: { type: String, default: '对话框内容' },
  cancelText: { type: String, default: '取消' },
  confirmText: { type: String, default: '确定' },
});

const attrs = useAttrs();

const listeners = computed(() => {
  const res = {};
  Object.entries(attrs).forEach(([k, v]) => {
    if (k.startsWith('on')) res[k] = v;
  });
  return res;
});

const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
const dialogStyle = computed(() => ({ margin: '0' }));
</script>

<style scoped>
.ui-el-dialog {
  width: 100%;
  height: 100%;
}
</style>
