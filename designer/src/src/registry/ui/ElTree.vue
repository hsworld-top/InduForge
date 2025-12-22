<template>
  <div class="ui-el-tree" :style="mergedStyle">
    <el-tree
      :data="data"
      :props="treeProps"
      :show-checkbox="showCheckbox"
      :highlight-current="highlightCurrent"
      :expand-on-click-node="expandOnClickNode"
      :style="{ width: '100%', height: '100%' }"
      v-bind="listeners"
      @node-click="(data, node, comp) => emit('node-click', data, node, comp)"
      @check="(data, checked) => emit('check', data, checked)"
    />
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElTree' });

const emit = defineEmits(['node-click', 'check']);

const props = defineProps({
  data: {
    type: Array,
    default: () => [
      { label: '一级 1', children: [{ label: '二级 1-1' }] },
      { label: '一级 2', children: [{ label: '二级 2-1' }] },
    ],
  },
  treeProps: { type: Object, default: () => ({ children: 'children', label: 'label' }) },
  showCheckbox: { type: Boolean, default: false },
  highlightCurrent: { type: Boolean, default: false },
  expandOnClickNode: { type: Boolean, default: true },
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
.ui-el-tree {
  width: 100%;
  height: 100%;
}
</style>
