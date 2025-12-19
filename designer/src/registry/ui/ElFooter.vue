<template>
  <el-footer class="ui-el-footer" :height="height" :style="mergedStyle" v-bind="listeners">
    <slot>底部</slot>
  </el-footer>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElFooter' });

const props = defineProps({
  height: { type: String, default: '48px' },
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
.ui-el-footer {
  width: 100%;
  height: 100%;
  background: #f5f7fa;
  display: flex;
  align-items: center;
  padding: 0 12px;
  box-sizing: border-box;
}
</style>
