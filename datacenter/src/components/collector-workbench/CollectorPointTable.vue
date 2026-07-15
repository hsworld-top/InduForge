<template>
  <div class="collector-point-table">
    <div class="collector-point-table__toolbar">
      <div class="collector-point-table__filters">
        <el-input
          v-model="search"
          clearable
          placeholder="搜索变量名称、编码或地址"
          style="width: 250px"
          @keyup.enter="load(1)"
          @clear="load(1)"
        />
        <span>共 {{ total }} 个变量</span>
      </div>
      <div class="collector-point-table__actions">
        <el-button :disabled="!selected.length" @click="removeSelected">删除所选</el-button>
        <el-button type="primary" @click="emit('import')">批量导入</el-button>
      </div>
    </div>

    <div class="collector-point-table__content">
      <el-table
        v-loading="loading"
        :data="items"
        height="100%"
        @selection-change="selected = $event"
      >
        <el-table-column type="selection" width="44" />
        <el-table-column prop="name" label="变量名称" min-width="160">
          <template #default="scope">
            <div class="collector-point-table__identity">
              <strong>{{ scope.row.name }}</strong>
              <small>{{ scope.row.code }}</small>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="addressText" label="变量地址" min-width="190">
          <template #default="scope"
            ><code>{{ scope.row.addressText }}</code></template
          >
        </el-table-column>
        <el-table-column prop="dataType" label="数据类型" width="120" />
        <el-table-column prop="elementCount" label="元素数量" width="100" />
        <el-table-column prop="enabled" label="状态" width="100">
          <template #default="scope">
            <span
              class="collector-point-table__status"
              :class="{ 'is-enabled': scope.row.enabled }"
            >
              <i />{{ scope.row.enabled ? '已启用' : '已停用' }}
            </span>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <div class="collector-point-table__footer">
      <span>当前第 {{ page }} 页</span>
      <el-pagination
        layout="prev, pager, next"
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        @current-change="load"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import { deleteCollectorPointsBatch, listCollectorPoints } from '@/api/collector.api'
import type { CollectorPoint } from '@/api/schemas/collector.schema'

const props = defineProps<{ projectId: string; connectionId: string; groupId: string | null }>()
const emit = defineEmits<{
  selection: [pointIds: string[]]
  import: []
}>()
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
  await ElMessageBox.confirm(`确认删除 ${selected.value.length} 个变量？`, '删除变量', {
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
  min-height: 0;
  flex: 1;
  flex-direction: column;
}
.collector-point-table__toolbar,
.collector-point-table__filters,
.collector-point-table__actions,
.collector-point-table__footer {
  display: flex;
  align-items: center;
}
.collector-point-table__toolbar {
  min-height: 48px;
  justify-content: space-between;
  gap: 12px;
  padding: 0 0 12px 14px;
}
.collector-point-table__filters,
.collector-point-table__actions {
  gap: 9px;
}
.collector-point-table__filters > span {
  color: var(--dc-text-muted);
  font-size: 11px;
}
.collector-point-table__content {
  min-height: 0;
  flex: 1;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md) var(--dc-radius-md) 0 0;
}
.collector-point-table :deep(.el-table) {
  --el-table-border-color: var(--dc-border);
  --el-table-header-bg-color: var(--dc-surface-subtle);
  --el-table-row-hover-bg-color: var(--dc-primary-soft);
}
.collector-point-table :deep(.el-table th.el-table__cell) {
  height: 42px;
  color: var(--dc-text-secondary);
  font-weight: 700;
}
.collector-point-table :deep(.el-table td.el-table__cell) {
  height: 48px;
}
.collector-point-table__identity {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.collector-point-table__identity strong {
  color: var(--dc-text-secondary);
  font-size: 13px;
}
.collector-point-table__identity small {
  color: var(--dc-text-muted);
  font-size: 10px;
}
.collector-point-table code {
  color: var(--dc-primary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
}
.collector-point-table__status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--dc-text-muted);
  font-size: 11px;
}
.collector-point-table__status i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--dc-border-strong);
}
.collector-point-table__status.is-enabled {
  color: var(--dc-success);
}
.collector-point-table__status.is-enabled i {
  background: var(--dc-success);
}
.collector-point-table__footer {
  min-height: 42px;
  justify-content: space-between;
  padding: 0 8px 0 12px;
  border: 1px solid var(--dc-border);
  border-top: none;
  border-radius: 0 0 var(--dc-radius-md) var(--dc-radius-md);
  color: var(--dc-text-muted);
  font-size: 11px;
}
</style>
