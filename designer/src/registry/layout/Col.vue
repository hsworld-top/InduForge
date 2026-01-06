<template>
  <el-col
    class="layout-col"
    :span="span"
    :offset="offset"
    :push="push"
    :pull="pull"
    v-bind="attrs"
    :style="mergedStyle"
  >
    <slot></slot>
  </el-col>
</template>

<script setup>
/**
 * Col - Element Plus 列布局组件
 */
import { computed, useAttrs } from 'vue';

defineOptions({
  inheritAttrs: false,
});

const props = defineProps({
  span: {
    type: Number,
    default: 24,
    validator: (value) => value >= 0 && value <= 24,
  },
  offset: {
    type: Number,
    default: 0,
  },
  push: {
    type: Number,
    default: 0,
  },
  pull: {
    type: Number,
    default: 0,
  },
  padding: {
    type: Number,
    default: 0,
  },
  minHeight: {
    type: Number,
    default: 0,
  },
});

const attrs = useAttrs();

const mergedStyle = computed(() => {
  const styleAttr = attrs.style || {};
  const { left, top, right, bottom, position, zIndex, width, height, ...rest } = styleAttr;
  const style = {
    ...rest,
  };
  if (props.padding) {
    style.padding = `${props.padding}px`;
  }
  if (props.minHeight) {
    style.minHeight = `${props.minHeight}px`;
  }
  return style;
});
</script>

<style scoped>
.layout-col {
  position: relative;
  box-sizing: border-box;
}
</style>
