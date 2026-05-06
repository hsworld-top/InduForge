<!--
  数据点面板顶部工具栏
  常态只保留高频入口，低频操作收进更多菜单；选中后显示批量操作条。
-->
<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import IconEpDelete from "~icons/ep/delete";
import IconEpDownload from "~icons/ep/download";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";
import IconEpLink from "~icons/ep/link";
import IconEpMoreFilled from "~icons/ep/more-filled";
import IconEpPlus from "~icons/ep/plus";
import IconEpUpload from "~icons/ep/upload";

type VariableExportCommand = "csv" | "xlsx" | "json";
type ToolbarMoreCommand =
  | "createGroup"
  | "importCsv"
  | "importXlsx"
  | "importJson"
  | "exportCsv"
  | "exportXlsx"
  | "exportJson";

const props = defineProps<{
  /** 是否有选中的节点 */
  hasSelection?: boolean;
  /** 选中节点的类型：variable / group / mixed / none */
  selectionType?: "variable" | "group" | "mixed" | "none";
  /** 当前选中节点数量 */
  selectedCount?: number;
  /** 是否允许编辑（单选时允许） */
  canEdit?: boolean;
  /** 是否允许删除 */
  canDelete?: boolean;
}>();

const emit = defineEmits<{
  (event: "createVar"): void;
  (event: "createGroup"): void;
  (event: "quickAdd"): void;
  (event: "editSelected"): void;
  (event: "deleteSelected"): void;
  (event: "clearSelection"): void;
  (event: "exportVars", command: VariableExportCommand): void;
  (event: "importVars", command: VariableExportCommand): void;
}>();

function handleExport(command: VariableExportCommand) {
  emit("exportVars", command);
}

function handleImport(command: VariableExportCommand) {
  emit("importVars", command);
}

function handleMore(command: ToolbarMoreCommand): void {
  if (command === "createGroup") {
    emit("createGroup");
    return;
  }
  if (command.startsWith("import")) {
    handleImport(command.replace("import", "").toLowerCase() as VariableExportCommand);
    return;
  }
  handleExport(command.replace("export", "").toLowerCase() as VariableExportCommand);
}

const { t } = useI18n();

const selectionLabel = computed(() =>
  t("datapointPanel.toolbar.selectedSummary", { count: props.selectedCount || 0 }),
);
</script>

<template>
  <div class="toolbar">
    <div class="toolbar-main">
      <el-button
        class="toolbar-btn toolbar-btn--primary"
        size="small"
        type="primary"
        @click="emit('createVar')"
      >
        <IconEpPlus class="toolbar-icon" />
        {{ t("datapointPanel.toolbar.createVar") }}
      </el-button>
      <el-button class="toolbar-btn toolbar-btn--ghost" size="small" @click="emit('quickAdd')">
        <IconEpLink class="toolbar-icon" />
        {{ t("datapointPanel.toolbar.quickAdd") }}
      </el-button>
      <el-dropdown trigger="click" @command="handleMore">
        <el-button class="toolbar-more" size="small" :aria-label="t('datapointPanel.toolbar.more')">
          <IconEpMoreFilled class="toolbar-icon" />
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="createGroup">
              <IconEpFolder class="dropdown-icon" />
              {{ t("datapointPanel.toolbar.createGroup") }}
            </el-dropdown-item>
            <el-dropdown-item divided command="importJson">
              <IconEpDownload class="dropdown-icon" />
              {{ t("datapointPanel.toolbar.importJson") }}
            </el-dropdown-item>
            <el-dropdown-item command="importCsv">{{
              t("datapointPanel.toolbar.importCsv")
            }}</el-dropdown-item>
            <el-dropdown-item command="importXlsx">{{
              t("datapointPanel.toolbar.importXlsx")
            }}</el-dropdown-item>
            <el-dropdown-item divided command="exportJson">
              <IconEpUpload class="dropdown-icon" />
              {{ t("datapointPanel.toolbar.exportJson") }}
            </el-dropdown-item>
            <el-dropdown-item command="exportCsv">{{
              t("datapointPanel.toolbar.exportCsv")
            }}</el-dropdown-item>
            <el-dropdown-item command="exportXlsx">{{
              t("datapointPanel.toolbar.exportXlsx")
            }}</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>

    <div v-if="hasSelection" class="toolbar-selection">
      <span class="toolbar-selection__label">{{ selectionLabel }}</span>
      <el-tooltip :content="t('datapointPanel.toolbar.edit')" placement="top">
        <el-button
          class="toolbar-action"
          size="small"
          :disabled="!canEdit"
          @click="emit('editSelected')"
        >
          <IconEpEditPen class="toolbar-icon" />
        </el-button>
      </el-tooltip>
      <el-tooltip :content="t('datapointPanel.toolbar.delete')" placement="top">
        <el-button
          class="toolbar-action toolbar-action--danger"
          size="small"
          :disabled="!canDelete"
          @click="emit('deleteSelected')"
        >
          <IconEpDelete class="toolbar-icon" />
        </el-button>
      </el-tooltip>
      <el-button class="toolbar-clear" size="small" text @click="emit('clearSelection')">
        {{ t("datapointPanel.toolbar.clearSelection") }}
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.toolbar-main,
.toolbar-selection {
  display: flex;
  align-items: center;
  gap: 6px;
}

.toolbar-btn {
  height: 30px;
  border-radius: 8px;
  font-weight: 500;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  justify-content: center;
  margin: 0 !important;
}

.toolbar-btn--primary {
  flex: 1.1 1 0;
  min-width: 0;
}

.toolbar-btn--ghost {
  flex: 1 1 0;
  min-width: 0;
}

.toolbar-more,
.toolbar-action {
  width: 30px;
  height: 30px;
  padding: 0;
  border-radius: 8px;
  justify-content: center;
  margin: 0 !important;
}

.toolbar-btn--ghost {
  color: #606266;
  border-color: #dcdfe6;
  background-color: #fff;
}

.toolbar-selection {
  padding: 5px 6px;
  border: 1px solid var(--designer-border-color);
  border-radius: 8px;
  background: var(--designer-primary-soft);
}

.toolbar-selection__label {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--designer-primary-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.toolbar-action--danger:not(:disabled):hover {
  color: var(--el-color-danger);
  border-color: var(--el-color-danger-light-5);
  background-color: var(--el-color-danger-light-9);
}

.toolbar-icon {
  font-size: 12px;
  flex-shrink: 0;
}

.dropdown-icon {
  font-size: 12px;
  margin-right: 6px;
}

.toolbar-clear {
  height: 30px;
  padding: 0 4px;
  font-size: 12px;
}
</style>
