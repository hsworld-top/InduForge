<template>
  <div class="ui-el-collapse" :style="mergedStyle">
    <el-collapse :model-value="modelValue" v-bind="listeners">
      <el-collapse-item v-for="item in items" :key="item.name" v-bind="item">
        <slot :name="item.name">{{ item.content }}</slot>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElCollapse' });

const props = defineProps({
  modelValue: { type: [Array, String], default: () => ['1'] },
  items: {
    type: Array,
    default: () => [
      { name: '1', title: '面板 1', content: '内容 1' },
      { name: '2', title: '面板 2', content: '内容 2' },
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
.ui-el-collapse {
  width: 100%;
  height: 100%;
}
</style>
