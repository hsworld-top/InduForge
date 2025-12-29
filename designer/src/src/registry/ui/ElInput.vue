<template>
  <el-input
    class="ui-el-input"
    :model-value="modelValue"
    :placeholder="placeholder"
    :type="type"
    :clearable="clearable"
    :disabled="disabled"
    :show-password="showPassword"
    :maxlength="maxlength"
    :show-word-limit="showWordLimit"
    :prefix-icon="prefixIcon"
    :suffix-icon="suffixIcon"
    :style="mergedStyle"
    v-bind="listeners"
    @input="handleInput"
    @change="(val) => emit('change', val)"
    @focus="(evt) => emit('focus', evt)"
    @blur="(evt) => emit('blur', evt)"
    @clear="() => emit('clear')"
  />
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElInput' });

const emit = defineEmits(['update:modelValue', 'input', 'change', 'focus', 'blur', 'clear']);

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  placeholder: { type: String, default: '请输入' },
  type: { type: String, default: 'text' },
  clearable: { type: Boolean, default: true },
  disabled: { type: Boolean, default: false },
  showPassword: { type: Boolean, default: false },
  maxlength: { type: [Number, null], default: null },
  showWordLimit: { type: Boolean, default: false },
  prefixIcon: { type: String, default: '' },
  suffixIcon: { type: String, default: '' },
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

function handleInput(val) {
  emit('update:modelValue', val);
  emit('input', val);
}
</script>

<style scoped>
.ui-el-input {
  width: 100%;
  height: 100%;
}
</style>
