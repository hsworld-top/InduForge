<template>
  <el-tabs
    class="ui-el-tabs"
    :model-value="modelValue"
    :type="type"
    :tab-position="tabPosition"
    :closable="closable"
    :addable="addable"
    :editable="editable"
    :stretch="stretch"
    :style="mergedStyle"
    v-bind="listeners"
    @tab-click="(pane, ev) => emit('tab-click', pane, ev)"
    @tab-remove="(name) => emit('tab-remove', name)"
    @tab-add="() => emit('tab-add')"
    @edit="(name, action) => emit('edit', name, action)"
    @tab-change="(name) => emit('tab-change', name)"
  >
    <el-tab-pane v-for="pane in panes" :key="pane.name" v-bind="pane">
      <slot :name="pane.name">{{ pane.content || pane.label }}</slot>
    </el-tab-pane>
  </el-tabs>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElTabs' });

const emit = defineEmits(['tab-click', 'tab-remove', 'tab-add', 'edit', 'tab-change']);

const props = defineProps({
  modelValue: { type: String, default: '1' },
  type: { type: String, default: '' },
  tabPosition: { type: String, default: 'top' },
  closable: { type: Boolean, default: false },
  addable: { type: Boolean, default: false },
  editable: { type: Boolean, default: false },
  stretch: { type: Boolean, default: false },
  panes: {
    type: Array,
    default: () => [
      { label: 'Tab 1', name: '1', content: 'Tab 1 内容' },
      { label: 'Tab 2', name: '2', content: 'Tab 2 内容' },
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
.ui-el-tabs {
  width: 100%;
  height: 100%;
}
</style>
