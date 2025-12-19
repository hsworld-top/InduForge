<template>
  <el-slider
    class="ui-el-slider"
    :model-value="modelValue"
    :min="min"
    :max="max"
    :step="step"
    :range="range"
    :show-stops="showStops"
    :disabled="disabled"
    :style="mergedStyle"
    v-bind="listeners"
    @change="(val) => emit('change', val)"
    @input="(val) => emit('input', val)"
  />
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElSlider' });

const emit = defineEmits(['change', 'input']);

const props = defineProps({
  modelValue: { type: [Number, Array], default: 30 },
  min: { type: Number, default: 0 },
  max: { type: Number, default: 100 },
  step: { type: Number, default: 1 },
  range: { type: Boolean, default: false },
  showStops: { type: Boolean, default: false },
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
.ui-el-slider {
  width: 100%;
  height: 100%;
}
</style>
