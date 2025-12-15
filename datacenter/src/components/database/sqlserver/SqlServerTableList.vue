<template>
  <div class="sqlserver-table-list">
    <el-table
      :data="tables"
      v-loading="loading"
      stripe
      border
      size="small"
      @row-dblclick="handleRowDblClick"
    >
      <el-table-column prop="name" label="表名" min-width="200" show-overflow-tooltip />
      <el-table-column prop="comment" label="备注" min-width="200" show-overflow-tooltip />
      <el-table-column prop="rows" label="行数" width="120" align="right">
        <template #default="{ row }">
          {{ row.rows?.toLocaleString() || 0 }}
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { computed, onMounted, inject } from 'vue'
import { useSqlServer } from '@/composables/database/useSqlServer'

const props = defineProps({
  connectionId: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['table-select'])

const projectId = inject('projectId')

const { tables, loading, loadTables } = useSqlServer(
  projectId,
  computed(() => props.connectionId)
)

onMounted(() => {
  loadTables()
})

const handleRowDblClick = (row) => {
  emit('table-select', row.name)
}
</script>

<style scoped>
.sqlserver-table-list {
  height: 100%;
}
</style>
