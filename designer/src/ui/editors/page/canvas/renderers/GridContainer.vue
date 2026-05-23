<!--
  GridContainer - Grid 布局容器
  根据 node.props 生成 grid-template-columns/rows、gap
-->
<script setup lang="ts">
import { computed } from 'vue'

interface GridContainerNodeLike {
  props?: {
    columns?: number
    rows?: number
    gap?: number | string
    columnTemplate?: string
    rowTemplate?: string
  }
}

const props = defineProps<{
  /**
   * 节点数据
   */
  node: GridContainerNodeLike
}>()

/**
 * Grid 容器样式
 */
const gridStyle = computed<Record<string, string>>(() => {
  const { columns, rows, gap, columnTemplate, rowTemplate } = props.node.props || {}

  const style: Record<string, string> = {
    display: 'grid',
  }

  // 列模板：优先使用自定义模板，否则使用列数生成
  if (columnTemplate) {
    style.gridTemplateColumns = columnTemplate
  } else if (columns) {
    style.gridTemplateColumns = `repeat(${columns}, 1fr)`
  } else {
    style.gridTemplateColumns = 'repeat(3, 1fr)' // 默认 3 列
  }

  // 行模板：优先使用自定义模板，否则使用行数生成
  if (rowTemplate) {
    style.gridTemplateRows = rowTemplate
  } else if (rows) {
    style.gridTemplateRows = `repeat(${rows}, auto)`
  } else {
    style.gridTemplateRows = 'repeat(3, auto)' // 默认 3 行
  }

  // 间距
  if (gap !== undefined) {
    style.gap = `${gap}px`
  } else {
    style.gap = '8px' // 默认 8px
  }

  return style
})
</script>

<template>
  <div class="grid-container" :style="gridStyle">
    <slot />
  </div>
</template>

<style scoped>
.grid-container {
  width: 100%;
  height: 100%;
  min-height: 100px;
  box-sizing: border-box;
}
</style>
