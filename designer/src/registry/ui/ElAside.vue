<template>
  <el-aside class="ui-el-aside" :width="width" :style="mergedStyle" v-bind="listeners">
    <slot>侧边栏</slot>
  </el-aside>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElAside' });

const props = defineProps({
  width: { type: String, default: '160px' },
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
.ui-el-aside {
  width: 100%;
  height: 100%;
  background: #f0f2f5;
  box-sizing: border-box;
  padding: 12px;
}
</style>
