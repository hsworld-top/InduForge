<template>
  <el-color-picker
    class="ui-el-color-picker"
    :model-value="modelValue"
    :show-alpha="showAlpha"
    :predefine="predefine"
    :style="mergedStyle"
    v-bind="listeners"
    @change="(val) => emit('change', val)"
    @active-change="(val) => emit('active-change', val)"
  />
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElColorPicker' });

const emit = defineEmits(['change', 'active-change']);

const props = defineProps({
  modelValue: { type: String, default: '#409EFF' },
  showAlpha: { type: Boolean, default: false },
  predefine: { type: Array, default: () => [] },
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
.ui-el-color-picker {
  width: 100%;
  height: 100%;
}
</style>
