import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  createAlarmItem,
  deleteAlarmItem,
  getAlarmItem,
  listAlarmGroupTree,
  listAlarmItems,
  setAlarmItemEnabled,
  updateAlarmItem,
} from '@/api/alarm.api'
import type { AlarmItem, AlarmItemSave, AlarmGroup } from '@/api/schemas/alarm.schema'
import { getApiErrorMessage } from '@/utils/request'

export const useAlarmStore = defineStore('alarm', () => {
  const list = ref<AlarmItem[]>([])
  const groups = ref<AlarmGroup[]>([])
  const total = ref(0)
  const loading = ref(false)
  const saving = ref(false)
  const error = ref('')
  const editing = ref<AlarmItem | null>(null)
  const hasEditing = computed(() => editing.value !== null)

  async function fetchList(projectId: string, params: Record<string, unknown> = {}) {
    loading.value = true
    error.value = ''
    try {
      const result = await listAlarmItems(projectId, params)
      list.value = result.list
      total.value = result.pagination.total || 0
      return result
    } catch (cause) {
      error.value = getApiErrorMessage(cause, '加载报警配置失败')
      throw cause
    } finally {
      loading.value = false
    }
  }

  async function fetchGroups(projectId: string) {
    groups.value = await listAlarmGroupTree(projectId)
    return groups.value
  }

  async function fetchDetail(projectId: string, policyId: string) {
    editing.value = await getAlarmItem(projectId, policyId)
    return editing.value
  }

  async function save(projectId: string, payload: AlarmItemSave, policyId?: string) {
    saving.value = true
    try {
      const result = policyId
        ? await updateAlarmItem(projectId, policyId, payload)
        : await createAlarmItem(projectId, payload)
      editing.value = result
      return result
    } finally {
      saving.value = false
    }
  }

  async function toggle(projectId: string, policyId: string, enabled: boolean) {
    const result = await setAlarmItemEnabled(projectId, policyId, enabled)
    list.value = list.value.map((item) => (item.id === policyId ? result : item))
    return result
  }

  async function remove(projectId: string, policyId: string) {
    await deleteAlarmItem(projectId, policyId)
    list.value = list.value.filter((item) => item.id !== policyId)
    total.value = Math.max(0, total.value - 1)
    if (editing.value?.id === policyId) editing.value = null
  }

  function clearEditing() {
    editing.value = null
  }

  return {
    list,
    groups,
    total,
    loading,
    saving,
    error,
    editing,
    hasEditing,
    fetchList,
    fetchGroups,
    fetchDetail,
    save,
    toggle,
    remove,
    clearEditing,
  }
})
