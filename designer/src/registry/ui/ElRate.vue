<template>
  <el-rate
    class="ui-el-rate"
    :model-value="modelValue"
    :max="max"
    :allow-half="allowHalf"
    :show-score="showScore"
    :texts="texts"
    :style="mergedStyle"
    v-bind="listeners"
    @change="(val) => emit('change', val)"
  />
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElRate' });

const emit = defineEmits(['change']);

const props = defineProps({
  modelValue: { type: Number, default: 4 },
  max: { type: Number, default: 5 },
  allowHalf: { type: Boolean, default: false },
  showScore: { type: Boolean, default: false },
  texts: { type: Array, default: () => [] },
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
.ui-el-rate {
  width: 100%;
  height: 100%;
}
</style>
