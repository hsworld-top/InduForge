<template>
  <el-select
    class="ui-el-select"
    :model-value="modelValue"
    :placeholder="placeholder"
    :multiple="multiple"
    :clearable="clearable"
    :filterable="filterable"
    :style="mergedStyle"
    v-bind="listeners"
    @change="(val) => emit('change', val)"
    @visible-change="(val) => emit('visible-change', val)"
    @remove-tag="(val) => emit('remove-tag', val)"
    @clear="() => emit('clear')"
    @blur="(evt) => emit('blur', evt)"
    @focus="(evt) => emit('focus', evt)"
  >
    <el-option v-for="opt in normalizedOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
  </el-select>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElSelect' });

const emit = defineEmits(['update:modelValue', 'change', 'visible-change', 'remove-tag', 'clear', 'blur', 'focus']);

const props = defineProps({
  modelValue: { type: [String, Number, Array], default: '' },
  placeholder: { type: String, default: '请选择' },
  multiple: { type: Boolean, default: false },
  clearable: { type: Boolean, default: true },
  filterable: { type: Boolean, default: true },
  options: { type: Array, default: () => [] },
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

const normalizedOptions = computed(() => {
  return (props.options || []).map((opt, idx) => ({
    label: opt.label ?? `选项 ${idx + 1}`,
    value: opt.value ?? opt.label ?? idx,
  }));
});
</script>

<style scoped>
.ui-el-select {
  width: 100%;
  height: 100%;
}
</style>
