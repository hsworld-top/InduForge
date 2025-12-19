<template>
  <div class="ui-el-radio-group" :style="mergedStyle">
    <el-radio-group :model-value="modelValue" :size="size" :disabled="disabled" v-bind="listeners" @change="(val) => emit('change', val)">
      <el-radio v-for="item in options" :key="item.value" :label="item.value">
        {{ item.label }}
      </el-radio>
    </el-radio-group>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElRadioGroup' });

const emit = defineEmits(['change']);

const props = defineProps({
  modelValue: { type: [String, Number], default: 'A' },
  size: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  options: {
    type: Array,
    default: () => [
      { label: '选项 A', value: 'A' },
      { label: '选项 B', value: 'B' },
      { label: '选项 C', value: 'C' },
    ],
  },
});

const attrs = useAttrs();

const listeners = computed(() => {
  const res = {};
  Object.entries(attrs).forEach(([k, v]) => {
    if (k.startsWith('on')) res[k] = v;
  });
  return res;
});

const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-el-radio-group {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
}
</style>
