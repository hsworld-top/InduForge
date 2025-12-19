<template>
  <el-date-picker
    class="ui-el-date-picker"
    :model-value="modelValue"
    :type="type"
    :placeholder="placeholder"
    :start-placeholder="startPlaceholder"
    :end-placeholder="endPlaceholder"
    :format="format"
    :value-format="valueFormat"
    :clearable="clearable"
    :range-separator="rangeSeparator"
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

defineOptions({ name: 'UIElDatePicker' });

const emit = defineEmits(['change', 'blur', 'focus']);

const props = defineProps({
  modelValue: { type: [String, Array], default: '2025-01-01' },
  type: { type: String, default: 'date' },
  placeholder: { type: String, default: '选择日期' },
  startPlaceholder: { type: String, default: '开始日期' },
  endPlaceholder: { type: String, default: '结束日期' },
  format: { type: String, default: 'YYYY-MM-DD' },
  valueFormat: { type: String, default: 'YYYY-MM-DD' },
  clearable: { type: Boolean, default: true },
  rangeSeparator: { type: String, default: '-' },
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
.ui-el-date-picker {
  width: 100%;
  height: 100%;
}
</style>
