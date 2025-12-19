<template>
  <el-transfer
    class="ui-el-transfer"
    :model-value="modelValue"
    :data="data"
    :titles="titles"
    :filterable="filterable"
    :style="mergedStyle"
    v-bind="listeners"
    @change="(val, dir, moved) => emit('change', val, dir, moved)"
  />
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElTransfer' });

const emit = defineEmits(['change']);

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  data: { type: Array, default: () => [] },
  titles: { type: Array, default: () => ['源列表', '目标列表'] },
  filterable: { type: Boolean, default: true },
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
.ui-el-transfer {
  width: 100%;
  height: 100%;
}
</style>
