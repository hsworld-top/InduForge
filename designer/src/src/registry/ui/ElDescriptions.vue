<template>
  <div class="ui-el-descriptions" :style="mergedStyle">
    <el-descriptions :title="title" :border="border" :column="column" v-bind="listeners">
      <el-descriptions-item v-for="(item, idx) in items" :key="idx" :label="item.label">
        <slot>{{ item.value }}</slot>
      </el-descriptions-item>
    </el-descriptions>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElDescriptions' });

const props = defineProps({
  title: { type: String, default: '信息' },
  border: { type: Boolean, default: true },
  column: { type: Number, default: 2 },
  items: {
    type: Array,
    default: () => [
      { label: '用户名', value: 'admin' },
      { label: '角色', value: 'Editor' },
      { label: '邮箱', value: 'admin@example.com' },
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
.ui-el-descriptions {
  width: 100%;
  height: 100%;
}
</style>
