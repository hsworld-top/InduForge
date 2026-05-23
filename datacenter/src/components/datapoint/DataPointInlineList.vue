<template>
  <div v-if="show" class="datapoint-inline">
    <div class="header flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span class="title">自动生成的数据点</span>
        <span class="count">{{ datapoints.length }}</span>
      </div>
      <el-button size="small" @click="loadDataPoints">刷新</el-button>
    </div>
    <el-table :data="datapoints" size="small" stripe class="mt-2">
      <el-table-column label="名称" width="180">
        <template #default="{ row }">
          <span class="text-xs text-gray-700 dark:text-gray-200">
            {{ row.name }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="数据点路径" min-width="240">
        <template #default="{ row }">
          <div class="flex items-center gap-2 min-w-0">
            <span class="text-xs text-gray-500 dark:text-gray-300 truncate">
              {{ row.path }}
            </span>
            <el-button link size="small" @click="copyPath(row.path)"> 复制 </el-button>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag size="small" :type="row.status === 'invalid' ? 'info' : 'success'">
            {{ row.status === 'invalid' ? '失效' : '活跃' }}
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
    ElMessage.error('加载数据点失败：' + getApiErrorMessage(error, '加载数据点失败'))
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
    ElMessage.success('已复制数据点路径')
  } catch (error) {
    ElMessage.error('复制失败，请手动复制')
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
  border-top: 1px dashed #e5e7eb;
}

.header {
  font-size: 13px;
  color: #374151;
}

.title {
  font-weight: 600;
}

.count {
  padding: 0 6px;
  border-radius: 999px;
  background: #f3f4f6;
  font-size: 11px;
  color: #6b7280;
}

.dark .header {
  color: #e5e7eb;
}

.dark .count {
  background: #1f2937;
  color: #9ca3af;
}
</style>
