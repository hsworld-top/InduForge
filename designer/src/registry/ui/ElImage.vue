<template>
  <div class="ui-el-image" :style="mergedStyle">
    <el-image
      :src="src"
      :fit="fit"
      :lazy="lazy"
      :preview-src-list="previewSrcList"
      :style="{ width: '100%', height: '100%' }"
      v-bind="listeners"
    >
      <template #error>
        <slot name="error">加载失败</slot>
      </template>
    </el-image>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElImage' });

const props = defineProps({
  src: { type: String, default: 'https://via.placeholder.com/240x120' },
  fit: { type: String, default: 'cover' },
  lazy: { type: Boolean, default: false },
  previewSrcList: { type: Array, default: () => [] },
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
.ui-el-image {
  width: 100%;
  height: 100%;
}
</style>
