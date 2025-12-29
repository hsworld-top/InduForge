<template>
  <el-col
    class="ui-el-col"
    :span="span"
    :offset="offset"
    :style="mergedStyle"
    v-bind="listeners"
  >
    <slot>列内容</slot>
  </el-col>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElCol' });

const props = defineProps({
  span: { type: Number, default: 12 },
  offset: { type: Number, default: 0 },
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
.ui-el-col {
  width: 100%;
  height: 100%;
}
</style>
