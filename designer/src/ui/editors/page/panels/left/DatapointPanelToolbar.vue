<!--
  数据点面板顶部：快速添加、导入、导出
-->
<script setup>
import IconEpDownload from "~icons/ep/download";
import IconEpLink from "~icons/ep/link";
import IconEpUpload from "~icons/ep/upload";

const emit = defineEmits(["quickAdd", "exportVars", "importVars"]);
</script>

<template>
  <div class="toolbar">
    <el-button class="toolbar-button toolbar-button--ghost" size="small" @click="emit('quickAdd')">
      <IconEpLink class="toolbar-icon" />
      快速添加数据点
    </el-button>
    <el-dropdown @command="(cmd) => emit('exportVars', cmd)">
      <el-button class="toolbar-button" size="small">
        <IconEpUpload class="toolbar-icon" />
        导出变量
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item command="csv">导出 CSV</el-dropdown-item>
          <el-dropdown-item command="xlsx">导出 XLSX</el-dropdown-item>
          <el-dropdown-item command="json">导出 JSON</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
    <el-dropdown @command="(cmd) => emit('importVars', cmd)">
      <el-button class="toolbar-button" size="small">
        <IconEpDownload class="toolbar-icon" />
        导入变量
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item command="csv">导入 CSV</el-dropdown-item>
          <el-dropdown-item command="xlsx">导入 XLSX</el-dropdown-item>
          <el-dropdown-item command="json">导入 JSON</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.toolbar-button {
  height: 32px;
  padding: 0 14px;
  border-radius: 8px;
  font-weight: 600;
  letter-spacing: 0.2px;
  box-shadow: 0 6px 14px rgba(64, 158, 255, 0.18);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 1 1 220px;
  justify-content: center;
}

.toolbar-button--ghost {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary-light-5);
  background-color: var(--el-color-primary-light-9);
  box-shadow: 0 6px 12px rgba(64, 158, 255, 0.12);
}

.toolbar-icon {
  font-size: 14px;
}

.toolbar-button--ghost:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary);
  background-color: var(--el-color-primary-light-8);
}
</style>
