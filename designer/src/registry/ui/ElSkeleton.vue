<template>
  <div class="ui-el-skeleton" :style="mergedStyle">
    <el-skeleton :loading="loading" :rows="rows" :animated="animated" v-bind="listeners">
      <template #template>
        <slot name="template"></slot>
      </template>
      <template #default>
        <slot>加载完成内容</slot>
      </template>
    </el-skeleton>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElSkeleton' });

const props = defineProps({
  loading: { type: Boolean, default: true },
  rows: { type: Number, default: 3 },
  animated: { type: Boolean, default: true },
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
.ui-el-skeleton {
  width: 100%;
  height: 100%;
}
</style>
