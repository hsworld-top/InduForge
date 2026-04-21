<!--
  页面树顶部搜索框
-->
<script setup lang="ts">
import { useI18n } from "vue-i18n";
import IconEpSearch from "~icons/ep/search";

const searchText = defineModel({ type: String, default: "" });
const { t } = useI18n();
</script>

<template>
  <div class="page-search">
    <el-input
      v-model="searchText"
      size="small"
      :placeholder="t('pageTree.searchPlaceholder')"
      clearable
      :prefix-icon="IconEpSearch"
    />
  </div>
</template>

<style scoped>
.page-search {
  padding: 2px 4px 8px;
  flex-shrink: 0;
}

.page-search :deep(.el-input__wrapper) {
  min-height: 30px;
  padding-inline: 10px;
  border-radius: 6px;
  background: rgba(15, 23, 42, 0.025);
  box-shadow: 0 0 0 1px rgba(15, 23, 42, 0.08) inset;
}

.page-search :deep(.el-input__wrapper.is-focus) {
  background: #fff;
  box-shadow: 0 0 0 1px rgba(37, 99, 235, 0.22) inset;
}

.page-search :deep(.el-input__inner) {
  font-size: 12px;
  color: var(--designer-text-primary, #191919);
}

.page-search :deep(.el-input__prefix-inner) {
  color: var(--designer-text-muted, #8c8c8c);
}
</style>
