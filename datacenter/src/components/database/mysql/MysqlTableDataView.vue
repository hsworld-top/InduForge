<template>
  <div class="mysql-table-data-view flex flex-col h-full">
    <div v-if="loading" class="h-full flex items-center justify-center text-gray-500">
      <IconTablerLoader class="mr-2 w-5 h-5 animate-spin" />
      数据加载中...
    </div>
    <template v-else>
      <div v-if="tableColumns.length === 0" class="text-center py-12 text-gray-500">
        未查询到数据
      </div>
      <div v-else class="flex flex-col h-full">
        <el-table :data="tableData" size="small" border class="flex-1">
          <el-table-column
            v-for="column in tableColumns"
            :key="column"
            :prop="column"
            :label="column"
            min-width="140"
            show-overflow-tooltip
          />
        </el-table>
        <div class="flex justify-end mt-4">
          <el-pagination
            background
            layout="prev, pager, next, jumper"
            :current-page="tablePagination.page"
            :page-size="tablePagination.limit"
            :total="tablePagination.total"
            @current-change="handlePageChange"
          />
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, inject, watch } from 'vue'
import IconTablerLoader from '~icons/tabler/loader'
import { useMysql } from '@/composables/database/useMysql'

const props = defineProps({
  connectionId: {
    type: String,
    required: true
  },
  tableName: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['back'])

const projectId = inject('projectId')
const { tableData, tableColumns, tablePagination, loading, loadTableData } = useMysql(
  projectId,
  ref(props.connectionId)
)

onMounted(() => {
  if (props.connectionId && props.tableName) {
    loadTableData(props.tableName, 1)
  }
})

watch([() => props.connectionId, () => props.tableName], ([newConnId, newTableName]) => {
  if (newConnId && newTableName) {
    loadTableData(newTableName, 1)
  }
})

const handlePageChange = (page) => {
  loadTableData(props.tableName, page)
}
</script>
