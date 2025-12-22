<template>
  <el-main class="ui-el-main" :style="mergedStyle" v-bind="listeners">
    <slot>内容</slot>
  </el-main>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElMain' });

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
.ui-el-main {
  width: 100%;
  height: 100%;
  background: #fff;
  padding: 12px;
  box-sizing: border-box;
}
</style>
