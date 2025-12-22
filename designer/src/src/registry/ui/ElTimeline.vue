<template>
  <div class="ui-el-timeline" :style="mergedStyle">
    <el-timeline>
      <el-timeline-item v-for="(item, idx) in items" :key="idx" v-bind="item">
        <slot>{{ item.content }}</slot>
      </el-timeline-item>
    </el-timeline>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElTimeline' });

const props = defineProps({
  items: {
    type: Array,
    default: () => [
      { timestamp: '2025-01-01', type: 'primary', content: '节点 1' },
      { timestamp: '2025-01-02', type: 'success', content: '节点 2' },
      { timestamp: '2025-01-03', type: 'warning', content: '节点 3' },
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
.ui-el-timeline {
  width: 100%;
  height: 100%;
}
</style>
