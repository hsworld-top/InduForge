<!--
  数据点面板：从数据中心批量勾选并快速添加变量
-->
<script setup>
defineProps({
  quickLoading: { type: Boolean, default: false },
  /** @type {unknown[]} */
  filteredFields: { type: Array, default: () => [] },
  quickPageSize: { type: Number, default: 200 },
  quickTotal: { type: Number, default: 0 },
  quickPage: { type: Number, default: 1 },
  buildVarName: { type: Function, required: true },
});

const emit = defineEmits(["confirm", "pageChange", "sizeChange", "selectionChange"]);

const visible = defineModel({ type: Boolean, default: false });
const searchKey = defineModel("searchKey", { type: String, default: "" });
const prefix = defineModel("prefix", { type: String, default: "" });
const suffix = defineModel("suffix", { type: String, default: "" });
const replaceFrom = defineModel("replaceFrom", { type: String, default: "" });
const replaceTo = defineModel("replaceTo", { type: String, default: "" });
</script>

<template>
  <el-dialog
    v-model="visible"
    title="快速添加数据点"
    width="1100px"
    top="3vh"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <el-form :inline="true" class="quick-form" label-width="60px" size="small">
      <el-row :gutter="12" class="quick-form-row">
        <el-col :span="8">
          <el-form-item label="搜索">
            <el-input v-model="searchKey" placeholder="字段名搜索" />
          </el-form-item>
        </el-col>
        <el-col :span="4">
          <el-form-item label="前缀">
            <el-input v-model="prefix" placeholder="前缀" />
          </el-form-item>
        </el-col>
        <el-col :span="4">
          <el-form-item label="后缀">
            <el-input v-model="suffix" placeholder="后缀" />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="替换" class="quick-replace">
            <el-input v-model="replaceFrom" placeholder="替换" />
            <span class="quick-arrow">→</span>
            <el-input v-model="replaceTo" placeholder="为" />
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
      @selection-change="(s) => emit('selectionChange', s)"
    >
      <el-table-column type="selection" width="50" />
      <el-table-column label="变量名" min-width="160">
        <template #default="{ row }">
          {{ buildVarName(row.name) }}
        </template>
      </el-table-column>
      <el-table-column prop="name" label="数据点名称" sortable width="150" />
      <el-table-column prop="path" label="路径" min-width="220" />
      <el-table-column prop="typeLabel" label="类型" width="110" sortable />
      <el-table-column prop="sourceLabel" label="来源" width="120" sortable />
      <el-table-column prop="updatedAtLabel" label="更新时间" width="160" />
    </el-table>
    <div class="quick-pagination">
      <el-pagination
        background
        layout="prev, pager, next, sizes, total"
        :page-size="quickPageSize"
        :page-sizes="[50, 100, 200, 500]"
        :total="quickTotal"
        :current-page="quickPage"
        @current-change="(p) => emit('pageChange', p)"
        @size-change="(s) => emit('sizeChange', s)"
      />
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="emit('confirm')">添加</el-button>
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
  color: #909399;
  flex: 0 0 auto;
}

.quick-pagination {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}
</style>
