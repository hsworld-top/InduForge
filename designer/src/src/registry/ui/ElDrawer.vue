<template>
  <div class="ui-el-drawer" :style="mergedStyle">
    <el-drawer
      :model-value="modelValue"
      :title="title"
      :size="size"
      :with-header="withHeader"
      :modal="modal"
      :append-to-body="appendToBody"
      v-bind="listeners"
      @open="() => emit('open')"
      @close="() => emit('close')"
    >
      <slot>{{ content }}</slot>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElDrawer' });

const props = defineProps({
  modelValue: { type: Boolean, default: true },
  title: { type: String, default: '抽屉' },
  size: { type: String, default: '320px' },
  withHeader: { type: Boolean, default: true },
  modal: { type: Boolean, default: false },
  appendToBody: { type: Boolean, default: false },
  content: { type: String, default: '抽屉内容' },
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
</script>

<style scoped>
.ui-el-drawer {
  width: 100%;
  height: 100%;
}
</style>
