<template>
  <div class="datapoint-workspace">
    <DataPointList
      :project-id="projectId"
      mode="management"
      :filter-q="filterQ"
      :filter-status="filterStatus"
      :filter-source="filterSource"
      :filter-tags="filterTags"
      :sort-field="sortField"
      :sort-order="sortOrder"
      :page="page"
      :detail-object-id="detailObjectId"
      @update:filter="handleFilterUpdate"
      @open-detail="handleOpenDetail"
      @close-detail="handleCloseDetail"
      @navigate="handleNavigate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DataPointList from '@/components/datapoint/DataPointList.vue'

const props = defineProps<{
  projectId: string
}>()

const route = useRoute()
const router = useRouter()

/* 从 URL query 读取筛选参数 */
const filterQ = computed(() => String(route.query.q || ''))
const filterStatus = computed(() => String(route.query.status || ''))
const filterSource = computed(() => String(route.query.source || ''))
const filterTags = computed(() => {
  const raw = route.query.tags
  if (!raw) return []
  return (Array.isArray(raw) ? raw : [raw]).map(String)
})
const sortField = computed(() => String(route.query.sort || 'createdAt'))
const sortOrder = computed(() => String(route.query.order || 'desc'))
const page = computed(() => {
  const n = Number(route.query.page)
  return n > 0 ? n : 1
})
const detailObjectId = computed(() => {
  if (route.params.module !== 'datapoint' || route.params.tab === 'workbench') return ''
  return String(route.params.objectId || '')
})

/* 列表发出筛选变化时同步到 URL */
function handleFilterUpdate(params: Record<string, unknown>) {
  const query: Record<string, string | string[]> = {}
  if (params.q) query.q = String(params.q)
  if (params.status) query.status = String(params.status)
  if (params.source) query.source = String(params.source)
  if (Array.isArray(params.tags) && params.tags.length > 0) {
    query.tags = params.tags as string[]
  }
  if (params.sort && params.sort !== 'createdAt') query.sort = String(params.sort)
  if (params.order && params.order !== 'desc') query.order = String(params.order)
  if (params.page && Number(params.page) > 1) query.page = String(params.page)
  // 保留 objectId 参数（打开详情时）
  if (route.params.objectId) {
    // objectId 在 path 里，不在 query 里，保持不动
  }
  void router.replace({ query })
}

/* 打开详情抽屉 → 加 objectId 到路径 */
function handleOpenDetail(row: { id: string }) {
  void router.replace({
    params: { ...route.params, objectId: row.id },
    query: route.query,
  })
}

/* 关闭详情抽屉 → 去掉 objectId */
function handleCloseDetail() {
  const params = { ...route.params }
  delete params.objectId
  void router.replace({ params, query: route.query })
}

/* LinkChip 跳转：切换到对应模块 + 打开目标对象 */
function handleNavigate(payload: { module: string; objectId: string; tab?: string }) {
  void router.push({
    name: route.name || 'datacenter',
    params: {
      ...route.params,
      module: payload.module,
      objectId: payload.objectId,
      tab: payload.tab || undefined,
    },
    query: route.query,
  })
}
</script>

<style scoped>
.datapoint-workspace {
  height: 100%;
  min-height: 0;
  overflow: hidden;
}
</style>
