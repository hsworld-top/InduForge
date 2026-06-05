<template>
  <div class="http-kv-editor">
    <div class="http-kv-editor__head">
      <el-button size="small" :icon="Plus" @click="addRow">添加</el-button>
    </div>
    <el-table
      :data="rows"
      size="small"
      border
      stripe
      class="http-kv-editor__table"
      empty-text="暂无键值对"
    >
      <el-table-column label="键" min-width="160">
        <template #default="{ row }">
          <el-input
            :model-value="row.key"
            placeholder="键"
            size="small"
            @input="(value: string) => updateRow(row, 'key', value)"
          />
        </template>
      </el-table-column>
      <el-table-column label="值" min-width="200">
        <template #default="{ row }">
          <el-input
            :model-value="row.value"
            placeholder="值"
            size="small"
            @input="(value: string) => updateRow(row, 'value', value)"
          />
        </template>
      </el-table-column>
      <el-table-column label="说明" min-width="160">
        <template #default="{ row }">
          <el-input
            :model-value="row.description"
            placeholder="说明"
            size="small"
            @input="(value: string) => updateRow(row, 'description', value)"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="72" align="center">
        <template #default="{ row }">
          <el-button
            size="small"
            circle
            plain
            :icon="IconTablerTrash"
            title="删除该行"
            aria-label="删除该行"
            class="http-kv-editor__remove"
            @click="removeRow(row)"
          />
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import IconTablerTrash from '~icons/tabler/trash'
import type { HttpKeyValueRow } from '@/api/schemas/http-workbench.schema'

const props = defineProps<{
  modelValue: HttpKeyValueRow[]
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: HttpKeyValueRow[]): void
  (event: 'change', value: HttpKeyValueRow[]): void
}>()

// 用 computed 双绑：外层 v-model 与本地状态保持一致
const rows = computed<HttpKeyValueRow[]>({
  get: () => props.modelValue || [],
  set: (value) => {
    emit('update:modelValue', value)
    emit('change', value)
  },
})

const updateRow = (row: HttpKeyValueRow, key: keyof HttpKeyValueRow, value: unknown) => {
  const next = rows.value.map((item) => (item === row ? { ...item, [key]: value } : item))
  rows.value = next
}

const addRow = () => {
  rows.value = [...rows.value, { enabled: true, key: '', value: '', description: '' }]
}

const removeRow = (row: HttpKeyValueRow) => {
  rows.value = rows.value.filter((item) => item !== row)
}
</script>

<style scoped>
.http-kv-editor {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 0;
}

.http-kv-editor__head {
  display: flex;
  justify-content: flex-start;
}

.http-kv-editor__table {
  width: 100%;
}

/* 单元格内的输入框去重边框，由 el-table 自身 border 统一控制，避免双层边框 */
.http-kv-editor__table :deep(.el-input__wrapper) {
  padding: 0 8px;
  background: transparent;
  box-shadow: none;
}

.http-kv-editor__table :deep(.el-input__wrapper:hover),
.http-kv-editor__table :deep(.el-input__wrapper.is-focus) {
  background: var(--dc-surface-raised);
  box-shadow: 0 0 0 1px var(--dc-border-strong) inset;
}

/* 表格 striped 行内输入框 hover 时与背景对比度更高 */
.http-kv-editor__table :deep(.el-table__row--striped .el-input__wrapper:hover),
.http-kv-editor__table :deep(.el-table__row--striped .el-input__wrapper.is-focus) {
  background: var(--dc-surface-muted);
}

.http-kv-editor__remove {
  color: var(--dc-text-muted);
}

.http-kv-editor__remove:hover {
  color: var(--dc-danger);
}
</style>
