/**
 * 数据中心查询 / 数据点值载荷：单一解析路径（与 axios 已解包后的业务体配合）
 */

function isRecord(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === "object";
}

function pickDatapointHit(hit: Record<string, unknown>): unknown {
  return hit.value ?? hit.currentValue ?? hit.dataValue ?? hit.lastValue ?? hit.rawValue;
}

/**
 * 从 getDatapointValues 等业务体中取出指定数据点的值（不认根级数组、不认裸 key 兜底）
 */
export function extractDatapointValue(payload: unknown, datapointId: string): unknown | null {
  if (!payload || !datapointId) return null;
  if (!isRecord(payload)) return null;

  if (payload.values && typeof payload.values === "object") {
    const values = payload.values;
    if (!Array.isArray(values) && datapointId in values) {
      return values[datapointId as keyof typeof values];
    }
    if (Array.isArray(values)) {
      const hit = values.find(
        (item): item is Record<string, unknown> => isRecord(item) && item.id === datapointId,
      );
      if (hit) return pickDatapointHit(hit);
    }
  }

  if (Array.isArray(payload.datapoints)) {
    const hit = payload.datapoints.find(
      (item): item is Record<string, unknown> => isRecord(item) && item.id === datapointId,
    );
    if (hit) return pickDatapointHit(hit);
  }

  return null;
}

/** GET .../connections 解包后的业务体 */
export function requireConnectionsPayload(payload: unknown): unknown[] {
  if (Array.isArray(payload)) {
    return payload;
  }
  if (!isRecord(payload)) {
    throw new Error("连接列表响应无效");
  }
  const list = payload.connections;
  if (!Array.isArray(list)) {
    throw new TypeError("连接列表缺少 connections 数组");
  }
  return list;
}

/** GET .../queries 解包后的业务体 */
export function requireQueriesPayload(payload: unknown): unknown[] {
  if (!isRecord(payload)) {
    throw new Error("查询列表响应无效");
  }
  const list = payload.queries;
  if (!Array.isArray(list)) {
    throw new TypeError("查询列表缺少 queries 数组");
  }
  return list;
}

/** GET .../datapoints 分页列表解包后的业务体 */
export function requireDatapointsPagePayload(payload: unknown): {
  datapoints: unknown[];
  pagination: Record<string, unknown>;
} {
  if (!isRecord(payload)) {
    throw new Error("数据点列表响应无效");
  }
  const list = payload.datapoints;
  if (!Array.isArray(list)) {
    throw new TypeError("数据点列表缺少 datapoints 数组");
  }
  const pag = payload.pagination;
  const pagination = isRecord(pag) ? pag : {};
  return { datapoints: list, pagination };
}

/**
 * executeQuery 等业务体：对象且含 `data` 键则取 `data`，否则原样返回（单一入口，禁止调用处双读）
 */
export function getQueryExecuteData(payload: unknown): unknown {
  if (payload === null || payload === undefined) return payload;
  if (typeof payload !== "object") return payload;
  if (Array.isArray(payload)) return payload;
  if (isRecord(payload) && Object.hasOwn(payload, "data")) {
    return payload.data;
  }
  return payload;
}
