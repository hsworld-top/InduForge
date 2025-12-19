<template>
  <div class="ui-el-badge" :style="mergedStyle">
    <el-badge :value="value" :max="max" :is-dot="isDot" :type="type" :hidden="hidden" v-bind="listeners">
      <slot>徽章</slot>
    </el-badge>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElBadge' });

const props = defineProps({
  value: { type: [String, Number], default: 12 },
  max: { type: Number, default: 99 },
  isDot: { type: Boolean, default: false },
  type: { type: String, default: 'primary' },
  hidden: { type: Boolean, default: false },
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
.ui-el-badge {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: flex-start;
}
</style>
