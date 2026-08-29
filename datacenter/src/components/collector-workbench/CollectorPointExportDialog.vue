<template>
  <el-dialog
    :model-value="modelValue"
    :title="ui('导出变量', 'Export Points')"
    width="620px"
    @close="emit('update:modelValue', false)"
  >
    <div class="collector-export">
      <el-radio-group v-model="scope" class="collector-export__scopes">
        <label class="collector-export__scope">
          <el-radio value="current_page">{{ ui('当前页', 'Current Page') }}</el-radio>
          <span>{{ ui(`第 ${page} 页，共 ${currentPageCount} 个变量`, `Page ${page}, ${currentPageCount} point${currentPageCount === 1 ? '' : 's'}`) }}</span>
        </label>
        <label class="collector-export__scope">
          <el-radio value="group">{{ ui('整个分组', 'Entire Group') }}</el-radio>
          <span>{{ groupName }}</span>
          <el-checkbox
            v-model="includeChildren"
            :disabled="scope !== 'group' || !groupId"
            @click.stop
          >
            {{ ui('包含子分组', 'Include Subgroups') }}
          </el-checkbox>
        </label>
        <label class="collector-export__scope" :class="{ 'is-disabled': selectedIds.length === 0 }">
          <el-radio value="selected" :disabled="selectedIds.length === 0">{{ ui('勾选变量', 'Selected Points') }}</el-radio>
          <span>{{ ui(`已跨页勾选 ${selectedIds.length} 个变量`, `${selectedIds.length} point${selectedIds.length === 1 ? '' : 's'} selected across pages`) }}</span>
        </label>
        <label class="collector-export__scope">
          <el-radio value="pages">{{ ui('指定页面', 'Specific Pages') }}</el-radio>
          <span>{{ ui('支持输入 1,3-5,8', 'Enter values such as 1,3-5,8') }}</span>
          <el-input
            v-if="scope === 'pages'"
            v-model="pageExpression"
            :placeholder="ui('例如：1,3-5,8', 'For example: 1,3-5,8')"
            @click.stop
          />
        </label>
      </el-radio-group>
      <div class="collector-export__summary">
        <span>{{ ui('文件格式', 'File Format') }}</span><strong>CSV</strong> <span>{{ ui('导出范围', 'Export Scope') }}</span
        ><strong>{{ scopeSummary }}</strong>
      </div>
      <el-alert v-if="validationError" :title="validationError" type="error" show-icon />
    </div>
    <template #footer>
      <el-button @click="emit('update:modelValue', false)">{{ ui('取消', 'Cancel') }}</el-button>
      <el-button type="primary" :loading="exporting" @click="submit">{{ ui('开始导出', 'Export') }}</el-button>
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
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

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
  parseCollectorExportPages(pageExpression.value, totalPages.value, {
    required: ui('请输入要导出的页码', 'Enter the pages to export'),
    invalidFormat: (token) => ui(`页码格式无效：${token}`, `Invalid page format: ${token}`),
    invalidRange: (token) => ui(`页码范围无效：${token}`, `Invalid page range: ${token}`),
    tooMany: ui('单次最多指定 200 页', 'Up to 200 pages can be exported at once'),
    outOfRange: (page) => ui(`第 ${page} 页超出当前总页数`, `Page ${page} exceeds the available page range`),
  }),
)
const validationError = computed(() => (scope.value === 'pages' ? parsedPages.value.error : ''))
const scopeSummary = computed(() => {
  if (scope.value === 'current_page') return ui(`${props.currentPageCount} 个变量`, `${props.currentPageCount} point${props.currentPageCount === 1 ? '' : 's'}`)
  if (scope.value === 'selected') return ui(`${props.selectedIds.length} 个变量`, `${props.selectedIds.length} point${props.selectedIds.length === 1 ? '' : 's'}`)
  if (scope.value === 'group')
    return includeChildren.value ? ui(`${props.groupName}及子分组`, `${props.groupName} and subgroups`) : props.groupName
  if (parsedPages.value.error) return ui('等待输入有效页码', 'Waiting for valid page numbers')
  const rows = estimateCollectorExportRows(parsedPages.value.pages, props.pageSize, props.total)
  return ui(`${parsedPages.value.pages.length} 页，预计 ${rows} 个变量`, `${parsedPages.value.pages.length} page${parsedPages.value.pages.length === 1 ? '' : 's'}, approximately ${rows} point${rows === 1 ? '' : 's'}`)
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
      `${safeFileName(props.connectionName)}-${ui('变量', 'points')}-${dayjs().format('YYYYMMDDHHmmss')}.csv`,
      blob,
    )
    ElMessage.success(ui('变量导出完成', 'Points exported'))
    emit('update:modelValue', false)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('变量导出失败', 'Failed to export points')))
  } finally {
    exporting.value = false
  }
}

function safeFileName(value: string) {
  return (value.trim() || ui('工业连接', 'industrial-connection')).replace(/[\\/:*?"<>|]/g, '_')
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
