<template>
  <el-pagination
    class="ui-el-pagination"
    :current-page="currentPage"
    :page-size="pageSize"
    :total="total"
    :layout="layout"
    :small="small"
    :background="background"
    :style="mergedStyle"
    v-bind="listeners"
    @current-change="(page) => emit('current-change', page)"
    @size-change="(size) => emit('size-change', size)"
    @prev-click="(page) => emit('prev-click', page)"
    @next-click="(page) => emit('next-click', page)"
  />
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({ name: 'UIElPagination' });

const emit = defineEmits(['current-change', 'size-change', 'prev-click', 'next-click']);

const props = defineProps({
  currentPage: { type: Number, default: 1 },
  pageSize: { type: Number, default: 10 },
  total: { type: Number, default: 100 },
  layout: { type: String, default: 'prev, pager, next' },
  small: { type: Boolean, default: false },
  background: { type: Boolean, default: false },
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
.ui-el-pagination {
  width: 100%;
  height: 100%;
}
</style>
