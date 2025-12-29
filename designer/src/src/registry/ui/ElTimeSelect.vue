<template>
  <el-time-select
    class="ui-el-time-select"
    :model-value="modelValue"
    :start="start"
    :end="end"
    :step="step"
    :placeholder="placeholder"
    :style="mergedStyle"
    v-bind="listeners"
    @change="(val) => emit('change', val)"
    @blur="(evt) => emit('blur', evt)"
    @focus="(evt) => emit('focus', evt)"
  />
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElTimeSelect' });

const emit = defineEmits(['change', 'blur', 'focus']);

const props = defineProps({
  modelValue: { type: String, default: '10:00' },
  start: { type: String, default: '08:30' },
  end: { type: String, default: '18:30' },
  step: { type: String, default: '00:30' },
  placeholder: { type: String, default: '选择时间' },
});

const attrs = useAttrs();

const listeners = computed(() => {
  const res = {};
  Object.entries(attrs).forEach(([key, val]) => {
    if (key.startsWith('on')) res[key] = val;
  });
  return res;
});

const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-el-time-select {
  width: 100%;
  height: 100%;
}
</style>
