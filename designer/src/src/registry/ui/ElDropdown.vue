<template>
  <el-dropdown
    class="ui-el-dropdown"
    :trigger="trigger"
    :style="mergedStyle"
    v-bind="listeners"
    @command="(cmd) => emit('command', cmd)"
    @visible-change="(val) => emit('visible-change', val)"
  >
    <el-button size="small">
      <slot>{{ text }}</slot>
      <el-icon class="el-icon--right"><arrow-down /></el-icon>
    </el-button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item v-for="(item, idx) in items" :key="idx" :command="item.command" :divided="item.divided">
          {{ item.label }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { ArrowDown } from '@element-plus/icons-vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElDropdown' });

const emit = defineEmits(['command', 'visible-change']);

const props = defineProps({
  trigger: { type: String, default: 'click' },
  text: { type: String, default: '下拉菜单' },
  items: {
    type: Array,
    default: () => [
      { command: 'A', label: '选项 A' },
      { command: 'B', label: '选项 B' },
      { command: 'C', label: '选项 C', divided: true },
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
.ui-el-dropdown {
  width: 100%;
  height: 100%;
}
</style>
