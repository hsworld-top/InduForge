<template>
  <el-dialog
    :model-value="modelValue"
    title="导出变量"
    width="620px"
    @close="emit('update:modelValue', false)"
  >
    <div class="collector-export">
      <el-radio-group v-model="scope" class="collector-export__scopes">
        <label class="collector-export__scope">
          <el-radio value="current_page">当前页</el-radio>
          <span>第 {{ page }} 页，共 {{ currentPageCount }} 个变量</span>
        </label>
        <label class="collector-export__scope">
          <el-radio value="group">整个分组</el-radio>
          <span>{{ groupName }}</span>
          <el-checkbox
            v-model="includeChildren"
            :disabled="scope !== 'group' || !groupId"
            @click.stop
          >
            包含子分组
          </el-checkbox>
        </label>
        <label class="collector-export__scope" :class="{ 'is-disabled': selectedIds.length === 0 }">
          <el-radio value="selected" :disabled="selectedIds.length === 0">勾选变量</el-radio>
          <span>已跨页勾选 {{ selectedIds.length }} 个变量</span>
        </label>
        <label class="collector-export__scope">
          <el-radio value="pages">指定页面</el-radio>
          <span>支持输入 1,3-5,8</span>
          <el-input
            v-if="scope === 'pages'"
            v-model="pageExpression"
            placeholder="例如：1,3-5,8"
            @click.stop
          />
        </label>
      </el-radio-group>
      <div class="collector-export__summary">
        <span>文件格式</span><strong>CSV</strong> <span>导出范围</span
        ><strong>{{ scopeSummary }}</strong>
      </div>
      <el-alert v-if="validationError" :title="validationError" type="error" show-icon />
    </div>
    <template #footer>
      <el-button @click="emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="exporting" @click="submit">开始导出</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import { exportCollectorPoints } from '@/api/collector.api'
import { getApiErrorMessage } from '@/utils/request'
import { downloadBlob } from '@/utils/tabular-file'
import { estimateCollectorExportRows, parseCollectorExportPages } from './collector-export'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  connectionId: string
  connectionName: string
  groupId: string | null
  groupName: string
  search: string
  page: number
  pageSize: number
  total: number
  currentPageCount: number
  selectedIds: string[]
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const scope = ref<'group' | 'current_page' | 'selected' | 'pages'>('current_page')
const includeChildren = ref(true)
const pageExpression = ref('')
const exporting = ref(false)
const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const parsedPages = computed(() =>
  parseCollectorExportPages(pageExpression.value, totalPages.value),
)
const validationError = computed(() => (scope.value === 'pages' ? parsedPages.value.error : ''))
const scopeSummary = computed(() => {
  if (scope.value === 'current_page') return `${props.currentPageCount} 个变量`
  if (scope.value === 'selected') return `${props.selectedIds.length} 个变量`
  if (scope.value === 'group')
    return includeChildren.value ? `${props.groupName}及子分组` : props.groupName
  if (parsedPages.value.error) return '等待输入有效页码'
  return `${parsedPages.value.pages.length} 页，预计 ${estimateCollectorExportRows(parsedPages.value.pages, props.pageSize, props.total)} 个变量`
})

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return
    scope.value = 'current_page'
    includeChildren.value = true
    pageExpression.value = ''
  },
)

async function submit() {
  if (validationError.value) return
  exporting.value = true
  try {
    const blob = await exportCollectorPoints(props.projectId, props.connectionId, {
      format: 'csv',
      scope: scope.value,
      groupId: props.groupId,
      includeChildren: includeChildren.value,
      search: props.search,
      sortBy: 'sortOrder',
      sortOrder: 'asc',
      page: props.page,
      pageSize: props.pageSize,
      pages: scope.value === 'pages' ? parsedPages.value.pages : undefined,
      pointIds: scope.value === 'selected' ? props.selectedIds : undefined,
    })
    downloadBlob(
      `${safeFileName(props.connectionName)}-变量-${dayjs().format('YYYYMMDDHHmmss')}.csv`,
      blob,
    )
    ElMessage.success('变量导出完成')
    emit('update:modelValue', false)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '变量导出失败'))
  } finally {
    exporting.value = false
  }
}

function safeFileName(value: string) {
  return (value.trim() || '工业连接').replace(/[\\/:*?"<>|]/g, '_')
}
</script>

<style scoped>
.collector-export,
.collector-export__scopes {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.collector-export__scope {
  display: grid;
  grid-template-columns: 116px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  min-height: 58px;
  padding: 12px 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  cursor: pointer;
}
.collector-export__scope:has(.el-radio.is-checked) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.collector-export__scope.is-disabled {
  opacity: 0.56;
}
.collector-export__scope .el-input {
  grid-column: 2 / 4;
}
.collector-export__scope > span {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.collector-export__summary {
  display: grid;
  grid-template-columns: 88px 1fr;
  gap: 8px 14px;
  padding: 12px 14px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}
.collector-export__summary span {
  color: var(--el-text-color-secondary);
}
</style>
