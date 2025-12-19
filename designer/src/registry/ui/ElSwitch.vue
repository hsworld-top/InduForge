<template>
  <el-switch
    class="ui-el-switch"
    :model-value="modelValue"
    :active-text="activeText"
    :inactive-text="inactiveText"
    :active-color="activeColor"
    :inactive-color="inactiveColor"
    :inline-prompt="inlinePrompt"
    :disabled="disabled"
    :style="mergedStyle"
    v-bind="listeners"
    @change="(val) => emit('change', val)"
  />
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElSwitch' });

const emit = defineEmits(['change']);

const props = defineProps({
  modelValue: { type: Boolean, default: true },
  activeText: { type: String, default: '开' },
  inactiveText: { type: String, default: '关' },
  activeColor: { type: String, default: '#409EFF' },
  inactiveColor: { type: String, default: '#dcdfe6' },
  inlinePrompt: { type: Boolean, default: false },
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
.ui-el-switch {
  width: 100%;
  height: 100%;
}
</style>
