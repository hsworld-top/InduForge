import { defineStore } from "pinia";
import { ref, computed } from "vue";
import dayjs from "dayjs";
import { ZodError } from "zod";
import { getDatapoints, getDatapoint } from "@/api/datapoint.api";
import { debugComputeUnit } from "@/api/compute.api";
import type { Datapoint } from "@/api/schemas/datapoint.schema";
import { ApiBusinessError } from "@/utils/request";
import dataAPI from "@/api/data.api";

/** 测试取值的最小入参，不污染 Datapoint schema */
interface DataPointLike {
  id: string;
  sourceType?: string;
  sourceId?: string | null;
  sourceConfig?: Record<string, unknown>;
}

/** 测试取值返回值 */
export type TestValueResult =
  | { ok: true; value: unknown; raw?: unknown; at: string }
  | { ok: false; reason: "unsupported" | "capability-disabled" | "error"; message: string };

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

  /**
   * 测试取值：根据 sourceType 路由到对应 API，统一返回结构化结果。
   * 不在此处 toast，让调用方决定展示策略。
   */
  async function testDatapointValue(
    projectId: string,
    datapoint: DataPointLike,
  ): Promise<TestValueResult> {
    const { sourceType, sourceId } = datapoint;

    try {
      switch (sourceType) {
        case "db.query": {
          if (!sourceId) {
            return { ok: false, reason: "unsupported", message: "该数据点缺少查询 ID，无法测试取值" };
          }
          const raw = await dataAPI.executeQuery(sourceId, {});
          const rows = (raw as Record<string, unknown>)?.rows;
          let value: unknown = raw;
          if (Array.isArray(rows) && rows.length > 0) {
            const firstRow = rows[0] as Record<string, unknown>;
            const firstKey = Object.keys(firstRow).find((k) => firstRow[k] != null);
            value = firstKey ? firstRow[firstKey] : JSON.stringify(firstRow);
          }
          return { ok: true, value, raw, at: dayjs().format("YYYY-MM-DD HH:mm:ss") };
        }

        case "mqtt.tag": {
          if (!sourceId) {
            return { ok: false, reason: "unsupported", message: "该数据点缺少标签 ID，无法测试取值" };
          }
          const raw = await dataAPI.getMqttTagValue(projectId, sourceId);
          const r = raw as Record<string, unknown>;
          const value = r?.value;
          const at = typeof r?.timestamp === "string"
            ? dayjs(r.timestamp).format("YYYY-MM-DD HH:mm:ss")
            : dayjs().format("YYYY-MM-DD HH:mm:ss");
          return { ok: true, value, raw, at };
        }

        case "mqtt.subscription":
          return { ok: false, reason: "unsupported", message: "MQTT 订阅暂不支持单点测试取值" };

        case "calc.output": {
          if (!sourceId) {
            return { ok: false, reason: "unsupported", message: "该数据点缺少计算单元 ID，无法测试取值" };
          }
          const raw = await debugComputeUnit(projectId, sourceId, {});
          const value = raw?.output;
          return { ok: true, value, raw, at: dayjs().format("YYYY-MM-DD HH:mm:ss") };
        }

        case "alarm.state":
          return { ok: false, reason: "unsupported", message: "报警状态不支持测试取值" };

        default:
          return { ok: false, reason: "unsupported", message: "该来源类型暂不支持测试取值" };
      }
    } catch (err: unknown) {
      // 判断是否为"能力未启用"类业务错误
      if (err instanceof ApiBusinessError) {
        const msg = err.message || "";
        const isCapabilityDisabled =
          /未启用|not enabled|not supported/i.test(msg);
        if (isCapabilityDisabled) {
          return {
            ok: false,
            reason: "capability-disabled",
            message: err.message,
          };
        }
        return { ok: false, reason: "error", message: err.message };
      }
      if (err instanceof ZodError) {
        return { ok: false, reason: "error", message: "数据格式异常" };
      }
      const e = err as Error | undefined;
      return { ok: false, reason: "error", message: e?.message || "测试取值失败" };
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
    testDatapointValue,
  };
});
