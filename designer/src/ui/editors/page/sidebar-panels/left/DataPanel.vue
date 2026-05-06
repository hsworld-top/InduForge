<!--
  DataPanel - 数据与变量面板容器
  常驻区只保留工程变量管理，数据点映射入口收敛到“快速添加数据点”弹窗。
-->
<script setup lang="ts">
import DatapointPanel from "./DatapointPanel.vue";
</script>

<template>
  <div class="data-panel">
    <DatapointPanel class="data-panel__content" />
  </div>
</template>

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
