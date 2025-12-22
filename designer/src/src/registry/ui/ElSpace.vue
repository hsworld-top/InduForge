<template>
  <el-space
    class="ui-el-space"
    :wrap="wrap"
    :size="size"
    :alignment="alignment"
    :direction="direction"
    :style="mergedStyle"
    v-bind="listeners"
  >
    <slot>
      <el-button size="small" type="primary">按钮 1</el-button>
      <el-button size="small">按钮 2</el-button>
      <el-button size="small" type="success">按钮 3</el-button>
    </slot>
  </el-space>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElSpace' });

const props = defineProps({
  wrap: { type: Boolean, default: true },
  size: { type: [Number, Array, String], default: 8 },
  alignment: { type: String, default: 'center' },
  direction: { type: String, default: 'horizontal' },
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
.ui-el-space {
  width: 100%;
  height: 100%;
}
</style>
