<template>
  <div class="ui-el-avatar" :style="mergedStyle">
    <el-avatar :size="size" :shape="shape" :src="src" :fit="fit" v-bind="listeners">
      <slot>头像</slot>
    </el-avatar>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElAvatar' });

const props = defineProps({
  size: { type: [Number, String], default: 48 },
  shape: { type: String, default: 'circle' },
  src: { type: String, default: 'https://via.placeholder.com/80' },
  fit: { type: String, default: 'cover' },
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
.ui-el-avatar {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: flex-start;
}
</style>
