<template>
  <el-row
    class="ui-el-row"
    :gutter="gutter"
    :justify="justify"
    :align="align"
    :style="mergedStyle"
    v-bind="listeners"
  >
    <slot>
      <el-col :span="8">列 1</el-col>
      <el-col :span="8">列 2</el-col>
      <el-col :span="8">列 3</el-col>
    </slot>
  </el-row>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElRow' });

const props = defineProps({
  gutter: { type: Number, default: 12 },
  justify: { type: String, default: 'start' },
  align: { type: String, default: 'top' },
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
.ui-el-row {
  width: 100%;
  height: 100%;
}
</style>
