import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { runContractCheck, getLatestContractCheck } from '@/api/contract-check.api'
import type { ContractCheckResult } from '@/api/schemas/contract-check.schema'

export const useContractCheckStore = defineStore('contractCheck', () => {
  // 最近一次检查结果
  const latestResult = ref<ContractCheckResult | null>(null)
  // 是否有正在进行的检查
  const running = ref(false)

  const passed = computed(() => latestResult.value?.status === 'passed')
  const failed = computed(() => latestResult.value?.status === 'failed')

  async function fetchLatest(projectId: string) {
    try {
      latestResult.value = await getLatestContractCheck(projectId)
    } catch {
      // 404 表示还没有过检查记录，静默处理
      latestResult.value = null
    }
  }

  async function run(projectId: string) {
    running.value = true
    try {
      latestResult.value = await runContractCheck(projectId)
    } finally {
      running.value = false
    }
  }

  function clear() {
    latestResult.value = null
  }

  return {
    latestResult,
    running,
    passed,
    failed,
    fetchLatest,
    run,
    clear,
  }
})
