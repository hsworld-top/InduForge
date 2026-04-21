<!--
  数据点面板：从数据中心批量勾选并快速添加变量
-->
<script setup lang="ts">
import { useI18n } from "vue-i18n";

interface QuickAddFieldLike {
  name: string;
  [key: string]: unknown;
}

defineProps<{
  quickLoading?: boolean;
  filteredFields?: QuickAddFieldLike[];
  quickPageSize?: number;
  quickTotal?: number;
  quickPage?: number;
  buildVarName: (name: string) => string;
}>();

const emit = defineEmits<{
  (event: "confirm"): void;
  (event: "pageChange", page: number): void;
  (event: "sizeChange", size: number): void;
  (event: "selectionChange", selection: unknown[]): void;
}>();

const visible = defineModel<boolean>({ default: false });
const searchKey = defineModel<string>("searchKey", { default: "" });
const prefix = defineModel<string>("prefix", { default: "" });
const suffix = defineModel<string>("suffix", { default: "" });
const replaceFrom = defineModel<string>("replaceFrom", { default: "" });
const replaceTo = defineModel<string>("replaceTo", { default: "" });

function handleSelectionChange(selection: unknown[]) {
  emit("selectionChange", selection);
}

function handleCurrentChange(page: number) {
  emit("pageChange", page);
}

function handleSizeChange(size: number) {
  emit("sizeChange", size);
}

const { t } = useI18n();
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="t('datapointPanel.quickAddDialog.title')"
    width="1100px"
    top="3vh"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <el-form :inline="true" class="quick-form" label-width="60px" size="small">
      <el-row :gutter="12" class="quick-form-row">
        <el-col :span="8">
          <el-form-item :label="t('datapointPanel.quickAddDialog.search')">
            <el-input v-model="searchKey" :placeholder="t('datapointPanel.quickAddDialog.searchPlaceholder')" />
          </el-form-item>
        </el-col>
        <el-col :span="4">
          <el-form-item :label="t('datapointPanel.quickAddDialog.prefix')">
            <el-input v-model="prefix" :placeholder="t('datapointPanel.quickAddDialog.prefixPlaceholder')" />
          </el-form-item>
        </el-col>
        <el-col :span="4">
          <el-form-item :label="t('datapointPanel.quickAddDialog.suffix')">
            <el-input v-model="suffix" :placeholder="t('datapointPanel.quickAddDialog.suffixPlaceholder')" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item :label="t('datapointPanel.quickAddDialog.replace')" class="quick-replace">
            <el-input v-model="replaceFrom" :placeholder="t('datapointPanel.quickAddDialog.replaceFromPlaceholder')" />
            <span class="quick-arrow">→</span>
            <el-input v-model="replaceTo" :placeholder="t('datapointPanel.quickAddDialog.replaceToPlaceholder')" />
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>

    <el-table
      v-loading="quickLoading"
      :data="filteredFields"
      border
      stripe
      size="small"
      height="520"
      @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="50" />
      <el-table-column :label="t('datapointPanel.quickAddDialog.variableName')" min-width="160">
        <template #default="{ row }">
          {{ buildVarName(row.name) }}
        </template>
      </el-table-column>
      <el-table-column prop="name" :label="t('datapointPanel.quickAddDialog.datapointName')" sortable width="150" />
      <el-table-column prop="path" :label="t('datapointPanel.quickAddDialog.path')" min-width="220" />
      <el-table-column prop="typeLabel" :label="t('datapointPanel.quickAddDialog.type')" width="110" sortable />
      <el-table-column prop="sourceLabel" :label="t('datapointPanel.quickAddDialog.source')" width="120" sortable />
      <el-table-column prop="updatedAtLabel" :label="t('datapointPanel.quickAddDialog.updatedAt')" width="160" />
    </el-table>
    <div class="quick-pagination">
      <el-pagination
        background
        layout="prev, pager, next, sizes, total"
        :page-size="quickPageSize"
        :page-sizes="[50, 100, 200, 500]"
        :total="quickTotal"
        :current-page="quickPage"
        @current-change="handleCurrentChange"
        @size-change="handleSizeChange"
      />
    </div>

    <template #footer>
      <el-button @click="visible = false">{{ t("datapointPanel.quickAddDialog.cancel") }}</el-button>
      <el-button type="primary" @click="emit('confirm')">{{ t("datapointPanel.quickAddDialog.add") }}</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.quick-form {
  --quick-control-height: var(--el-component-size-small, 28px);
  margin-bottom: 12px;
}

.quick-form-row {
  width: 100%;
  margin-bottom: 10px;
}

.quick-form :deep(.el-form-item) {
  width: 100%;
  margin-bottom: 8px;
  align-items: center;
}

.quick-form :deep(.el-form-item__label) {
  line-height: var(--quick-control-height);
}

.quick-form :deep(.el-form-item__content) {
  flex: 1;
  min-width: 0;
}

.quick-form :deep(.el-input),
.quick-form :deep(.el-select) {
  width: 100%;
}

.quick-form :deep(.el-input__wrapper),
.quick-form :deep(.el-select .el-input__wrapper) {
  height: var(--quick-control-height);
  min-height: var(--quick-control-height);
}

.quick-form :deep(.el-input__inner),
.quick-form :deep(.el-select .el-input__inner) {
  height: var(--quick-control-height);
  line-height: var(--quick-control-height);
}

.quick-replace :deep(.el-form-item__content) {
  display: flex;
  align-items: center;
  gap: 6px;
}

.quick-replace :deep(.el-input) {
  flex: 1;
}

.quick-arrow {
  flex-shrink: 0;
  color: var(--designer-text-muted);
  flex: 0 0 auto;
}

.quick-pagination {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}
</style>
