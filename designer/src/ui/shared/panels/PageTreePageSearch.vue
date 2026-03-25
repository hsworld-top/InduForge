<!--
  页面树顶部搜索框
-->
<script setup>
import IconEpSearch from "~icons/ep/search";

const searchText = defineModel({ type: String, default: "" });
</script>

<template>
  <div class="page-search">
    <el-input
      v-model="searchText"
      size="small"
      placeholder="搜索页面..."
      clearable
      :prefix-icon="IconEpSearch"
    />
  </div>
</template>

<style scoped>
.page-search {
  padding: 6px 10px 8px;
  flex-shrink: 0;
}

.page-search :deep(.el-input__wrapper) {
  min-height: 28px;
  padding-inline: 8px;
  box-shadow: 0 0 0 1px var(--designer-border-color, #e6e6e6) inset;
}

.page-search :deep(.el-input__inner) {
  font-size: 12px;
}
</style>
