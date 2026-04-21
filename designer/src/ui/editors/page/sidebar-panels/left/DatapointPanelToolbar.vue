<!--
  数据点面板顶部：快速添加、导入、导出
-->
<script setup lang="ts">
import { useI18n } from "vue-i18n";
import IconEpDownload from "~icons/ep/download";
import IconEpLink from "~icons/ep/link";
import IconEpUpload from "~icons/ep/upload";

type VariableExportCommand = "csv" | "xlsx" | "json";

const emit = defineEmits<{
  (event: "quickAdd"): void;
  (event: "exportVars", command: VariableExportCommand): void;
  (event: "importVars", command: VariableExportCommand): void;
}>();

function handleExport(command: VariableExportCommand) {
  emit("exportVars", command);
}

function handleImport(command: VariableExportCommand) {
  emit("importVars", command);
}

const { t } = useI18n();
</script>

<template>
  <div class="toolbar">
    <el-button class="toolbar-button toolbar-button--ghost" size="small" @click="emit('quickAdd')">
      <IconEpLink class="toolbar-icon" />
      {{ t("datapointPanel.toolbar.quickAdd") }}
    </el-button>
    <el-dropdown @command="handleExport">
      <el-button class="toolbar-button" size="small">
        <IconEpUpload class="toolbar-icon" />
        {{ t("datapointPanel.toolbar.exportVars") }}
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item command="csv">{{ t("datapointPanel.toolbar.exportCsv") }}</el-dropdown-item>
          <el-dropdown-item command="xlsx">{{ t("datapointPanel.toolbar.exportXlsx") }}</el-dropdown-item>
          <el-dropdown-item command="json">{{ t("datapointPanel.toolbar.exportJson") }}</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
    <el-dropdown @command="handleImport">
      <el-button class="toolbar-button" size="small">
        <IconEpDownload class="toolbar-icon" />
        {{ t("datapointPanel.toolbar.importVars") }}
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item command="csv">{{ t("datapointPanel.toolbar.importCsv") }}</el-dropdown-item>
          <el-dropdown-item command="xlsx">{{ t("datapointPanel.toolbar.importXlsx") }}</el-dropdown-item>
          <el-dropdown-item command="json">{{ t("datapointPanel.toolbar.importJson") }}</el-dropdown-item>
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
