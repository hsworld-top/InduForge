<template>
  <el-steps
    class="ui-el-steps"
    :active="active"
    :direction="direction"
    :align-center="alignCenter"
    :finish-status="finishStatus"
    :process-status="processStatus"
    :style="mergedStyle"
    v-bind="listeners"
  >
    <el-step v-for="(step, idx) in steps" :key="idx" v-bind="step" />
  </el-steps>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElSteps' });

const props = defineProps({
  active: { type: Number, default: 1 },
  direction: { type: String, default: 'horizontal' },
  alignCenter: { type: Boolean, default: true },
  finishStatus: { type: String, default: 'success' },
  processStatus: { type: String, default: 'process' },
  steps: {
    type: Array,
    default: () => [
      { title: '步骤一', description: '描述 1' },
      { title: '步骤二', description: '描述 2' },
      { title: '步骤三', description: '描述 3' },
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
.ui-el-steps {
  width: 100%;
  height: 100%;
}
</style>
