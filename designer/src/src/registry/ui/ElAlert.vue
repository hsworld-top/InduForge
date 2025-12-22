<template>
  <div class="ui-el-alert" :style="mergedStyle">
    <el-alert
      :title="title"
      :type="type"
      :description="description"
      :closable="closable"
      :show-icon="showIcon"
      :effect="effect"
      v-bind="listeners"
      @close="() => emit('close')"
    />
  </div>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElAlert' });

const emit = defineEmits(['close']);

const props = defineProps({
  title: { type: String, default: '成功提示' },
  type: { type: String, default: 'success' },
  description: { type: String, default: '这是一条提示' },
  closable: { type: Boolean, default: false },
  showIcon: { type: Boolean, default: true },
  effect: { type: String, default: 'light' },
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
.ui-el-alert {
  width: 100%;
  height: 100%;
}
</style>
