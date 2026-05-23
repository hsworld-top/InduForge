import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getAccessSources, getAccessSource, deleteAccessSource } from '@/api/access-source.api'
import type { AccessSourceSummary, AccessSourceDetail } from '@/api/schemas/access-source.schema'

export const useAccessSourceStore = defineStore('accessSource', () => {
  const list = ref<AccessSourceSummary[]>([])
  const total = ref(0)
  const loading = ref(false)
  // 当前选中的接入源 id
  const selectedId = ref<string | number | null>(null)
  // 当前详情（编辑/查看时加载）
  const detail = ref<AccessSourceDetail | null>(null)

  const selected = computed(() => list.value.find((s) => s.id === selectedId.value) ?? null)

  async function fetchList(projectId: string, params: Record<string, unknown> = {}) {
    loading.value = true
    try {
      const res = await getAccessSources(projectId, params)
      list.value = res.list
      total.value = res.pagination?.total ?? res.list.length
    } finally {
      loading.value = false
    }
  }

  async function fetchDetail(projectId: string, sourceId: string) {
    loading.value = true
    try {
      detail.value = await getAccessSource(projectId, sourceId)
    } finally {
      loading.value = false
    }
  }

  async function remove(projectId: string, sourceId: string) {
    await deleteAccessSource(projectId, sourceId)
    list.value = list.value.filter((s) => String(s.id) !== sourceId)
    if (String(selectedId.value) === sourceId) {
      selectedId.value = null
      detail.value = null
    }
  }

  function select(id: string | number | null) {
    selectedId.value = id
  }

  function clearDetail() {
    detail.value = null
  }

  return {
    list,
    total,
    loading,
    selectedId,
    detail,
    selected,
    fetchList,
    fetchDetail,
    remove,
    select,
    clearDetail,
  }
})
