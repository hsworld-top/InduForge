import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { getDatapoints, getDatapoint } from "@/api/datapoint.api";
import type { Datapoint } from "@/api/schemas/datapoint.schema";

// 数据点目录缓存，支持增量更新
export const useDataCatalogStore = defineStore("dataCatalog", () => {
  // 用 Map 以 id 为键，支持增量 upsert
  const items = ref<Map<string | number, Datapoint>>(new Map());
  // 详情缓存：单独存放已拉取的详情，key 为字符串化 id
  const detailById = ref<Map<string, Datapoint>>(new Map());
  const loading = ref(false);
  // 详情加载状态，key 为字符串化 id
  const detailLoading = ref<Map<string, boolean>>(new Map());
  const lastFetchedProjectId = ref<string>("");

  const list = computed(() => Array.from(items.value.values()));

  const total = computed(() => items.value.size);

  /** 按 path 查找 */
  function getByPath(path: string): Datapoint | undefined {
    return list.value.find((d) => d.path === path);
  }

  /** 增量 upsert 数据点 */
  function upsert(datapoints: Datapoint[]) {
    for (const dp of datapoints) {
      items.value.set(dp.id, dp);
    }
  }

  /** 按 id 删除缓存 */
  function remove(id: string | number) {
    items.value.delete(id);
  }

  /** 清空缓存 */
  function reset() {
    items.value.clear();
    detailById.value.clear();
    detailLoading.value.clear();
    lastFetchedProjectId.value = "";
  }

  /** 按 id 获取详情（优先读缓存，缺失时拉取） */
  async function fetchDetail(projectId: string, datapointId: string): Promise<Datapoint | null> {
    const key = String(datapointId);
    // 已有缓存直接返回
    const cached = detailById.value.get(key);
    if (cached) return cached;
    // 避免重复请求
    if (detailLoading.value.get(key)) return null;
    detailLoading.value.set(key, true);
    try {
      const dp = await getDatapoint(projectId, datapointId);
      detailById.value.set(key, dp);
      return dp;
    } finally {
      detailLoading.value.set(key, false);
    }
  }

  /** 按 id 获取已缓存详情（同步） */
  function getDetailById(id: string | number): Datapoint | undefined {
    return detailById.value.get(String(id));
  }

  /** 全量拉取并刷新缓存 */
  async function fetchAll(projectId: string, params: Record<string, unknown> = {}) {
    loading.value = true;
    try {
      const res = await getDatapoints(projectId, params);
      // 全量刷新：先清空再写入
      items.value.clear();
      upsert(res.list);
      lastFetchedProjectId.value = projectId;
    } finally {
      loading.value = false;
    }
  }

  return {
    items,
    detailById,
    detailLoading,
    loading,
    lastFetchedProjectId,
    list,
    total,
    getByPath,
    getDetailById,
    upsert,
    remove,
    reset,
    fetchAll,
    fetchDetail,
  };
});
