<template>
  <el-breadcrumb
    class="ui-el-breadcrumb"
    :separator="separator"
    :separator-class="separatorClass"
    :style="mergedStyle"
    v-bind="listeners"
  >
    <slot>
      <el-breadcrumb-item v-for="(item, idx) in items" :key="idx">{{ item }}</el-breadcrumb-item>
    </slot>
  </el-breadcrumb>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElBreadcrumb' });

const props = defineProps({
  separator: { type: String, default: '/' },
  separatorClass: { type: String, default: '' },
  items: { type: Array, default: () => ['首页', '列表', '详情'] },
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
.ui-el-breadcrumb {
  width: 100%;
  height: 100%;
}
</style>
