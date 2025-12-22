<template>
  <el-divider
    class="ui-el-divider"
    :direction="direction"
    :content-position="contentPosition"
    :style="mergedStyle"
    v-bind="listeners"
  >
    <slot>分割线</slot>
  </el-divider>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElDivider' });

const props = defineProps({
  direction: { type: String, default: 'horizontal' },
  contentPosition: { type: String, default: 'center' },
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
.ui-el-divider {
  width: 100%;
  height: 100%;
}
</style>
