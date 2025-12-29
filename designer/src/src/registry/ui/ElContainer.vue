<template>
  <el-container class="ui-el-container" :style="mergedStyle" v-bind="listeners">
    <slot>
      <el-header height="40px">头部</el-header>
      <el-main>内容</el-main>
      <el-footer height="40px">底部</el-footer>
    </slot>
  </el-container>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElContainer' });

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
.ui-el-container {
  width: 100%;
  height: 100%;
  border: 1px solid #ebeef5;
}
</style>
