<template>
  <div v-if="show" class="datapoint-inline">
    <div class="header flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span class="title">{{ ui('自动生成的数据点', 'Generated Data Points') }}</span>
        <span class="count">{{ datapoints.length }}</span>
      </div>
      <el-button size="small" @click="loadDataPoints">{{ ui('刷新', 'Refresh') }}</el-button>
    </div>
    <el-table :data="datapoints" size="small" stripe class="mt-2">
      <el-table-column :label="ui('名称', 'Name')" width="180">
        <template #default="{ row }">
          <span class="text-xs text-gray-700 dark:text-gray-200">
            {{ row.name }}
          </span>
        </template>
      </el-table-column>
      <el-table-column :label="ui('数据点路径', 'Data Point Path')" min-width="240">
        <template #default="{ row }">
          <div class="flex items-center gap-2 min-w-0">
            <span class="text-xs text-gray-500 dark:text-gray-300 truncate">
              {{ row.path }}
            </span>
            <el-button link size="small" @click="copyPath(row.path)">{{ ui('复制', 'Copy') }}</el-button>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="ui('状态', 'Status')" width="90">
        <template #default="{ row }">
          <el-tag size="small" :type="row.status === 'invalid' ? 'info' : 'success'">
            {{ row.status === 'invalid' ? ui('失效', 'Invalid') : ui('活跃', 'Active') }}
          </el-tag>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
  queryId: {
    type: String,
    default: '',
  },
})

const datapoints = ref([])
const loading = ref(false)

const show = computed(() => Boolean(props.queryId))

/**
 * 加载数据点
 * @returns {Promise<void>}
 */
const loadDataPoints = async () => {
  if (!props.queryId) {
    datapoints.value = []
    return
  }
  loading.value = true
  try {
    const response = await dataAPI.getDataPoints(props.projectId, {
      type: 'db.query',
      sourceId: props.queryId,
      page: 1,
      pageSize: 200,
    })
    datapoints.value = response.data?.datapoints || []
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('加载数据点失败', 'Failed to load data points')))
  } finally {
    loading.value = false
  }
}

/**
 * 复制路径
 * @param {string} path - 数据点路径
 */
const copyPath = async (path) => {
  if (!path) return
  try {
    await navigator.clipboard.writeText(path)
    ElMessage.success(ui('已复制数据点路径', 'Data point path copied'))
  } catch {
    ElMessage.error(ui('复制失败，请手动复制', 'Copy failed. Please copy manually.'))
  }
}

watch(
  () => props.queryId,
  () => {
    loadDataPoints()
  },
  { immediate: true },
)
</script>

<style scoped>
.datapoint-inline {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px dashed var(--dc-border);
}

.header {
  font-size: 13px;
  color: var(--dc-text);
}

.title {
  font-weight: 600;
}

.count {
  padding: 0 6px;
  border-radius: 999px;
  background: var(--dc-surface-subtle);
  font-size: 11px;
  color: var(--dc-text-secondary);
}

.dark .header {
  color: var(--dc-text);
}

.dark .count {
  background: var(--dc-surface-subtle);
  color: var(--dc-text-secondary);
}
</style>
