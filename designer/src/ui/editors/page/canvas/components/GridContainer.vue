<!--
  GridContainer - Grid 布局容器
  根据 node.props 生成 grid-template-columns/rows、gap
-->
<template>
  <div class="grid-container" :style="gridStyle">
    <slot />
  </div>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  /**
   * 节点数据
   * @type {import('@/editor-core').ComponentNode}
   */
  node: {
    type: Object,
    required: true,
  },
});

/**
 * Grid 容器样式
 */
const gridStyle = computed(() => {
  const { columns, rows, gap, columnTemplate, rowTemplate } =
    props.node.props || {};

  const style = {
    display: "grid",
  };

  // 列模板：优先使用自定义模板，否则使用列数生成
  if (columnTemplate) {
    style.gridTemplateColumns = columnTemplate;
  } else if (columns) {
    style.gridTemplateColumns = `repeat(${columns}, 1fr)`;
  } else {
    style.gridTemplateColumns = "repeat(3, 1fr)"; // 默认 3 列
  }

  // 行模板：优先使用自定义模板，否则使用行数生成
  if (rowTemplate) {
    style.gridTemplateRows = rowTemplate;
  } else if (rows) {
    style.gridTemplateRows = `repeat(${rows}, auto)`;
  } else {
    style.gridTemplateRows = "repeat(3, auto)"; // 默认 3 行
  }

  // 间距
  if (gap !== undefined) {
    style.gap = `${gap}px`;
  } else {
    style.gap = "8px"; // 默认 8px
  }

  return style;
});
</script>

<style scoped>
.grid-container {
  width: 100%;
  height: 100%;
  min-height: 100px;
  box-sizing: border-box;
}
</style>
