<template>
  <div class="ui-el-tag" :style="mergedStyle">
    <el-tag :type="type" :effect="effect" :closable="closable" :hit="hit" v-bind="listeners">
      <slot>{{ text }}</slot>
    </el-tag>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElTag' });

const props = defineProps({
  text: { type: String, default: '标签' },
  type: { type: String, default: 'success' },
  effect: { type: String, default: 'light' },
  closable: { type: Boolean, default: false },
  hit: { type: Boolean, default: false },
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
.ui-el-tag {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: flex-start;
}
</style>
