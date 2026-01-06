<template>
  <el-row
    class="layout-row"
    :gutter="gutter"
    :justify="justify"
    :align="align"
    :wrap="wrap"
    v-bind="attrs"
    :style="mergedStyle"
  >
    <slot></slot>
  </el-row>
</template>

<script setup>
/**
 * Row - Element Plus 行布局组件
 */
import { computed, useAttrs } from 'vue';

defineOptions({
  inheritAttrs: false,
});

const props = defineProps({
  gutter: {
    type: Number,
    default: 0,
  },
  gutterVertical: {
    type: Number,
    default: 0,
  },
  justify: {
    type: String,
    default: 'start',
  },
  align: {
    type: String,
    default: 'top',
  },
  wrap: {
    type: Boolean,
    default: true,
  },
});

const attrs = useAttrs();

const mergedStyle = computed(() => {
  const styleAttr = attrs.style || {};
  const { left, top, right, bottom, position, zIndex, ...rest } = styleAttr;
  const style = {
    ...rest,
    width: '100%',
  };
  if (props.gutterVertical) {
    style.rowGap = `${props.gutterVertical}px`;
  }
  return style;
});
</script>

<style scoped>
.layout-row {
  width: 100%;
}
</style>
