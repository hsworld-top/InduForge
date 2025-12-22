<template>
  <div class="ui-el-calendar" :style="mergedStyle">
    <el-calendar :model-value="modelValue" v-bind="listeners" />
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElCalendar' });

const props = defineProps({
  modelValue: { type: [String, Date], default: '2025-01-01' },
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
.ui-el-calendar {
  width: 100%;
  height: 100%;
}
</style>
