import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { getDatapoints } from "@/api/datapoint.api";
import type { Datapoint } from "@/api/schemas/datapoint.schema";

// 数据点目录缓存，支持增量更新
export const useDataCatalogStore = defineStore("dataCatalog", () => {
  // 用 Map 以 id 为键，支持增量 upsert
  const items = ref<Map<string | number, Datapoint>>(new Map());
  const loading = ref(false);
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
    lastFetchedProjectId.value = "";
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
    loading,
    lastFetchedProjectId,
    list,
    total,
    getByPath,
    upsert,
    remove,
    reset,
    fetchAll,
  };
});
