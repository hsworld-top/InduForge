<template>
  <el-time-picker
    class="ui-el-time-picker"
    :model-value="modelValue"
    :is-range="isRange"
    :start-placeholder="startPlaceholder"
    :end-placeholder="endPlaceholder"
    :placeholder="placeholder"
    :format="format"
    :value-format="valueFormat"
    :arrow-control="arrowControl"
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

defineOptions({ name: 'UIElTimePicker' });

const emit = defineEmits(['change', 'blur', 'focus']);

const props = defineProps({
  modelValue: { type: [String, Array], default: '12:00:00' },
  isRange: { type: Boolean, default: false },
  startPlaceholder: { type: String, default: '开始时间' },
  endPlaceholder: { type: String, default: '结束时间' },
  placeholder: { type: String, default: '选择时间' },
  format: { type: String, default: 'HH:mm:ss' },
  valueFormat: { type: String, default: 'HH:mm:ss' },
  arrowControl: { type: Boolean, default: true },
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
.ui-el-time-picker {
  width: 100%;
  height: 100%;
}
</style>
