<template>
  <div class="ui-el-carousel" :style="mergedStyle">
    <el-carousel :height="height" :type="type" :autoplay="autoplay" v-bind="listeners">
      <el-carousel-item v-for="(item, idx) in items" :key="idx">
        <slot>{{ item }}</slot>
      </el-carousel-item>
    </el-carousel>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElCarousel' });

const props = defineProps({
  height: { type: String, default: '160px' },
  type: { type: String, default: 'card' },
  autoplay: { type: Boolean, default: false },
  items: {
    type: Array,
    default: () => ['轮播 1', '轮播 2', '轮播 3'],
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
.ui-el-carousel {
  width: 100%;
  height: 100%;
}
</style>
