<template>
  <el-input-number
    class="ui-el-input-number"
    :model-value="modelValue"
    :min="min"
    :max="max"
    :step="step"
    :precision="precision"
    :controls="controls"
    :disabled="disabled"
    :style="mergedStyle"
    v-bind="listeners"
    @change="(val) => emit('change', val)"
    @input="(val) => emit('input', val)"
    @focus="(evt) => emit('focus', evt)"
    @blur="(evt) => emit('blur', evt)"
  />
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElInputNumber' });

const emit = defineEmits(['update:modelValue', 'change', 'input', 'focus', 'blur']);

const props = defineProps({
  modelValue: { type: Number, default: 0 },
  min: { type: Number, default: 0 },
  max: { type: Number, default: 100 },
  step: { type: Number, default: 1 },
  precision: { type: Number, default: null },
  controls: { type: Boolean, default: true },
  disabled: { type: Boolean, default: false },
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
.ui-el-input-number {
  width: 100%;
  height: 100%;
}
</style>
