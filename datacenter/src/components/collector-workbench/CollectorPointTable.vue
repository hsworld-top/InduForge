<template>
  <div class="collector-point-table">
    <div class="collector-point-table__toolbar">
      <el-input
        v-model="search"
        clearable
        placeholder="搜索点位"
        style="width: 220px"
        @keyup.enter="load(1)"
      /><el-button :disabled="!selected.length" @click="removeSelected">删除所选</el-button>
    </div>
    <el-table v-loading="loading" :data="items" height="100%" @selection-change="selected = $event"
      ><el-table-column type="selection" width="44" /><el-table-column
        prop="code"
        label="编码"
        min-width="120"
      /><el-table-column prop="name" label="名称" min-width="150" /><el-table-column
        prop="addressText"
        label="地址"
        min-width="220"
      /><el-table-column prop="dataType" label="数据类型" width="110" /><el-table-column
        prop="enabled"
        label="启用"
        width="70"
        ><template #default="scope"
          ><el-tag :type="scope.row.enabled ? 'success' : 'info'">{{
            scope.row.enabled ? '是' : '否'
          }}</el-tag></template
        ></el-table-column
      ></el-table
    >
    <el-pagination
      layout="total, prev, pager, next"
      :current-page="page"
      :page-size="pageSize"
      :total="total"
      @current-change="load"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import { deleteCollectorPointsBatch, listCollectorPoints } from '@/api/collector.api'
import type { CollectorPoint } from '@/api/schemas/collector.schema'
const props = defineProps<{ projectId: string; connectionId: string; groupId: string | null }>()
const emit = defineEmits<{ selection: [pointIds: string[]] }>()
const items = ref<CollectorPoint[]>([])
const selected = ref<CollectorPoint[]>([])
const loading = ref(false)
const search = ref('')
const page = ref(1)
const pageSize = 50
const total = ref(0)
async function load(nextPage = page.value) {
  page.value = nextPage
  loading.value = true
  try {
    const result = await listCollectorPoints(props.projectId, props.connectionId, {
      page: page.value,
      pageSize,
      search: search.value,
      groupId: props.groupId || undefined,
    })
    items.value = result.list
    total.value = result.pagination.total
  } finally {
    loading.value = false
  }
}
async function removeSelected() {
  await ElMessageBox.confirm(`确认删除 ${selected.value.length} 个点位？`, '删除点位', {
    type: 'warning',
  })
  await deleteCollectorPointsBatch(
    props.projectId,
    props.connectionId,
    selected.value.map((item) => item.id),
  )
  await load()
}
watch(selected, (value) =>
  emit(
    'selection',
    value.map((item) => item.id),
  ),
)
watch(
  () => [props.connectionId, props.groupId],
  () => load(1),
)
onMounted(() => load())
defineExpose({ reload: load })
</script>

<style scoped>
.collector-point-table {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 12px;
}
.collector-point-table__toolbar {
  display: flex;
  justify-content: space-between;
}
.collector-point-table :deep(.el-table) {
  flex: 1;
}
</style>
