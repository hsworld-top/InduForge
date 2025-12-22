<template>
  <el-cascader
    class="ui-el-cascader"
    :model-value="modelValue"
    :options="options"
    :props="cascaderProps"
    :clearable="clearable"
    :filterable="filterable"
    :placeholder="placeholder"
    :style="mergedStyle"
    v-bind="listeners"
    @change="(val) => emit('change', val)"
    @expand-change="(val) => emit('expand-change', val)"
    @blur="(evt) => emit('blur', evt)"
    @focus="(evt) => emit('focus', evt)"
    @visible-change="(val) => emit('visible-change', val)"
  />
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElCascader' });

const emit = defineEmits(['change', 'expand-change', 'blur', 'focus', 'visible-change']);

const props = defineProps({
  modelValue: { type: [String, Number, Array], default: () => [] },
  options: { type: Array, default: () => [] },
  cascaderProps: { type: Object, default: () => ({ checkStrictly: false, multiple: false }) },
  clearable: { type: Boolean, default: true },
  filterable: { type: Boolean, default: true },
  placeholder: { type: String, default: '请选择' },
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
.ui-el-cascader {
  width: 100%;
  height: 100%;
}
</style>
