<template>
  <el-card
    class="ui-el-card"
    :shadow="shadow"
    :style="mergedStyle"
    v-bind="listeners"
  >
    <template #header>
      <slot name="header">{{ header }}</slot>
    </template>
    <slot>{{ content }}</slot>
  </el-card>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElCard' });

const props = defineProps({
  header: { type: String, default: '卡片标题' },
  content: { type: String, default: '卡片内容' },
  shadow: { type: String, default: 'always' },
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
.ui-el-card {
  width: 100%;
  height: 100%;
}
</style>
