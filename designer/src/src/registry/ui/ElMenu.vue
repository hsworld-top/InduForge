<template>
  <el-menu
    class="ui-el-menu"
    :default-active="defaultActive"
    :mode="mode"
    :ellipsis="ellipsis"
    :collapse="collapse"
    :style="mergedStyle"
    v-bind="listeners"
    @select="(index, indexPath, item) => emit('select', index, indexPath, item)"
    @open="(index, indexPath) => emit('open', index, indexPath)"
    @close="(index, indexPath) => emit('close', index, indexPath)"
  >
    <template v-for="item in items" :key="item.index">
      <el-sub-menu v-if="item.children" :index="item.index">
        <template #title>{{ item.label }}</template>
        <el-menu-item v-for="child in item.children" :key="child.index" :index="child.index">
          {{ child.label }}
        </el-menu-item>
      </el-sub-menu>
      <el-menu-item v-else :index="item.index">{{ item.label }}</el-menu-item>
    </template>
  </el-menu>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElMenu' });

const emit = defineEmits(['select', 'open', 'close']);

const props = defineProps({
  defaultActive: { type: String, default: '1' },
  mode: { type: String, default: 'horizontal' },
  ellipsis: { type: Boolean, default: true },
  collapse: { type: Boolean, default: false },
  items: {
    type: Array,
    default: () => [
      { index: '1', label: '菜单一' },
      {
        index: '2',
        label: '子菜单',
        children: [
          { index: '2-1', label: '选项 1' },
          { index: '2-2', label: '选项 2' },
        ],
      },
      { index: '3', label: '菜单三' },
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
.ui-el-menu {
  width: 100%;
  height: 100%;
}
</style>
