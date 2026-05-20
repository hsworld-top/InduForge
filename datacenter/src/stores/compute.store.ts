import { defineStore } from "pinia";
import { ref, computed } from "vue";
import {
  getComputeUnits,
  getComputeUnit,
  getComputeFolders,
  createComputeUnit,
  createComputeFolder,
  updateComputeFolder,
  updateComputeUnit,
  deleteComputeUnit,
  toggleComputeUnit,
  getComputeDependencies,
} from "@/api/compute.api";
import type {
  ComputeUnit,
  ComputeUnitDetail,
  ComputeFolder,
  ComputeUnitSave,
  ComputeFolderSave,
  ComputeRunResult,
  ComputeDependency,
} from "@/api/schemas/compute.schema";
import { getApiErrorMessage } from "@/utils/request";

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
  const listError = ref("");
  // 文件夹树
  const folders = ref<ComputeFolder[]>([]);
  const foldersLoading = ref(false);
  const foldersError = ref("");
  // 当前正在编辑的计算单元
  const editing = ref<ComputeUnitDetail | null>(null);
  const detailLoading = ref(false);
  const detailError = ref("");
  const creating = ref(false);
  const createError = ref("");
  const saving = ref(false);
  const deleting = ref(false);
  const dependencies = ref<ComputeDependency[]>([]);
  const dependenciesLoading = ref(false);
  const dependenciesError = ref("");
  // 调试会话（同时只有一个）
  const debugSession = ref<DebugSession | null>(null);

  const hasEditing = computed(() => editing.value !== null);

  async function fetchList(projectId: string, params: Record<string, unknown> = {}) {
    loading.value = true;
    listError.value = "";
    try {
      const res = await getComputeUnits(projectId, params);
      list.value = res.list;
      total.value = res.pagination?.total ?? res.list.length;
    } catch (error) {
      list.value = [];
      total.value = 0;
      listError.value = getApiErrorMessage(error, "计算单元能力未启用");
      throw error;
    } finally {
      loading.value = false;
    }
  }

  async function fetchFolders(projectId: string) {
    foldersLoading.value = true;
    foldersError.value = "";
    try {
      folders.value = await getComputeFolders(projectId);
    } catch (error) {
      folders.value = [];
      foldersError.value = getApiErrorMessage(error, "文件夹能力未启用");
      throw error;
    } finally {
      foldersLoading.value = false;
    }
  }

  async function openForEdit(projectId: string, id: string) {
    detailLoading.value = true;
    detailError.value = "";
    try {
      editing.value = await getComputeUnit(projectId, id);
      return editing.value;
    } catch (error) {
      editing.value = null;
      detailError.value = getApiErrorMessage(error, "计算单元详情不可用");
      throw error;
    } finally {
      detailLoading.value = false;
    }
  }

  function closeEdit() {
    editing.value = null;
    detailError.value = "";
  }

  async function createUnit(projectId: string, data: ComputeUnitSave) {
    creating.value = true;
    createError.value = "";
    try {
      const unit = await createComputeUnit(projectId, data);
      list.value = [unit, ...list.value.filter((item) => item.id !== unit.id)];
      total.value = Math.max(total.value, list.value.length);
      fetchList(projectId).catch(() => undefined);
      editing.value = unit;
      return unit;
    } catch (error) {
      createError.value = getApiErrorMessage(error, "新建计算单元失败");
      throw error;
    } finally {
      creating.value = false;
    }
  }

  async function createFolder(projectId: string, data: ComputeFolderSave) {
    creating.value = true;
    createError.value = "";
    try {
      const folder = await createComputeFolder(projectId, data);
      folders.value = [folder, ...folders.value.filter((item) => item.id !== folder.id)];
      fetchFolders(projectId).catch(() => undefined);
      return folder;
    } catch (error) {
      createError.value = getApiErrorMessage(error, "新建文件夹失败");
      throw error;
    } finally {
      creating.value = false;
    }
  }

  async function saveFolder(
    projectId: string,
    folderId: string,
    data: Partial<ComputeFolderSave>,
  ) {
    saving.value = true;
    try {
      const folder = await updateComputeFolder(projectId, folderId, data);
      fetchFolders(projectId).catch(() => undefined);
      return folder;
    } finally {
      saving.value = false;
    }
  }

  async function saveUnit(
    projectId: string,
    id: string,
    data: Partial<ComputeUnitSave>,
  ) {
    saving.value = true;
    try {
      const unit = await updateComputeUnit(projectId, id, data);
      editing.value = unit;
      list.value = list.value.map((item) =>
        item.id === unit.id ? { ...item, ...unit } : item,
      );
      return unit;
    } finally {
      saving.value = false;
    }
  }

  async function removeUnit(projectId: string, id: string) {
    deleting.value = true;
    try {
      await deleteComputeUnit(projectId, id);
      list.value = list.value.filter((item) => item.id !== id);
      total.value = Math.max(0, total.value - 1);
      if (editing.value?.id === id) {
        closeEdit();
      }
    } finally {
      deleting.value = false;
    }
  }

  async function setUnitEnabled(projectId: string, id: string, enabled: boolean) {
    saving.value = true;
    try {
      const unit = await toggleComputeUnit(projectId, id, enabled);
      editing.value = unit;
      list.value = list.value.map((item) =>
        item.id === unit.id ? { ...item, ...unit } : item,
      );
      return unit;
    } finally {
      saving.value = false;
    }
  }

  async function fetchDependencies(projectId: string) {
    dependenciesLoading.value = true;
    dependenciesError.value = "";
    try {
      dependencies.value = await getComputeDependencies(projectId);
    } catch (error) {
      dependencies.value = [];
      dependenciesError.value = getApiErrorMessage(error, "依赖清单不可用");
      throw error;
    } finally {
      dependenciesLoading.value = false;
    }
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
    listError,
    folders,
    foldersLoading,
    foldersError,
    editing,
    detailLoading,
    detailError,
    creating,
    createError,
    saving,
    deleting,
    dependencies,
    dependenciesLoading,
    dependenciesError,
    debugSession,
    hasEditing,
    fetchList,
    fetchFolders,
    openForEdit,
    closeEdit,
    createUnit,
    createFolder,
    saveFolder,
    saveUnit,
    removeUnit,
    setUnitEnabled,
    fetchDependencies,
    startDebug,
    finishDebug,
    clearDebug,
  };
});
