import { defineStore } from "pinia";
import { ref, computed } from "vue";
import {
  getComputeUnits,
  getComputeUnit,
  getComputeFolders,
} from "@/api/compute.api";
import type {
  ComputeUnit,
  ComputeUnitDetail,
  ComputeFolder,
  ComputeRunResult,
} from "@/api/schemas/compute.schema";

// 调试会话上下文
interface DebugSession {
  unitId: string | number;
  input: Record<string, unknown>;
  result: ComputeRunResult | null;
  running: boolean;
}

export const useComputeStore = defineStore("compute", () => {
  const list = ref<ComputeUnit[]>([]);
  const total = ref(0);
  const loading = ref(false);
  // 文件夹树
  const folders = ref<ComputeFolder[]>([]);
  // 当前正在编辑的计算单元
  const editing = ref<ComputeUnitDetail | null>(null);
  // 调试会话（同时只有一个）
  const debugSession = ref<DebugSession | null>(null);

  const hasEditing = computed(() => editing.value !== null);

  async function fetchList(projectId: string, params: Record<string, unknown> = {}) {
    loading.value = true;
    try {
      const res = await getComputeUnits(projectId, params);
      list.value = res.list;
      total.value = res.pagination?.total ?? res.list.length;
    } finally {
      loading.value = false;
    }
  }

  async function fetchFolders(projectId: string) {
    folders.value = await getComputeFolders(projectId);
  }

  async function openForEdit(projectId: string, id: string) {
    loading.value = true;
    try {
      editing.value = await getComputeUnit(projectId, id);
    } finally {
      loading.value = false;
    }
  }

  function closeEdit() {
    editing.value = null;
  }

  /** 开始调试会话 */
  function startDebug(unitId: string | number, input: Record<string, unknown> = {}) {
    debugSession.value = { unitId, input, result: null, running: true };
  }

  /** 调试完成，写入结果 */
  function finishDebug(result: ComputeRunResult) {
    if (debugSession.value) {
      debugSession.value.result = result;
      debugSession.value.running = false;
    }
  }

  function clearDebug() {
    debugSession.value = null;
  }

  return {
    list,
    total,
    loading,
    folders,
    editing,
    debugSession,
    hasEditing,
    fetchList,
    fetchFolders,
    openForEdit,
    closeEdit,
    startDebug,
    finishDebug,
    clearDebug,
  };
});
