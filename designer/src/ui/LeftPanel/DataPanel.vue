<!--
  DataPanel - 数据面板容器
  包装 DatapointPanel，提供数据点/变量管理的入口
-->
<template>
  <div class="data-panel">
    <DatapointPanel class="data-panel__content" />
  </div>
</template>

<script setup>
/** 数据面板：包装 DatapointPanel */
import DatapointPanel from "./DatapointPanel.vue";
</script>

<style scoped>
.data-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.data-panel__content {
  flex: 1;
  min-height: 0;
}
</style>
