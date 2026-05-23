<script setup lang="ts">
import { computed } from 'vue'
import { normalizeTableColumns, normalizeTableData } from './shared'

const props = defineProps<{
  resolvedProps?: Record<string, unknown>
}>()

const columns = computed(() => normalizeTableColumns(props.resolvedProps?.columns))
const rows = computed(() => normalizeTableData(props.resolvedProps?.data))
const tableProps = computed(() => {
  const { columns: _columns, data: _data, ...rest } = props.resolvedProps || {}
  return rest
})
</script>

<template>
  <el-table v-bind="tableProps" :data="rows" class="core-table">
    <el-table-column
      v-for="column in columns"
      :key="column.prop"
      :prop="column.prop"
      :label="column.label"
      :width="column.width"
      :align="column.align"
    />
  </el-table>
</template>

<style scoped>
.core-table {
  width: 100%;
  height: 100%;
}
</style>
