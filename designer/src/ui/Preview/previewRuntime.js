import { datacenterApi } from "@/services";
import { DataService } from "@/data";
import { Storage } from "@/utils/storage";
import { io } from "socket.io-client";

let runtimeInstance = null;
const componentRefsByPage = new Map();
const componentRefsByName = new Map();
const previewDataServiceState = {
  service: null,
  connectPromise: null,
  projectId: null,
  subscribed: new Set(),
  pending: new Map(),
};
const previewMqttState = {
  socket: null,
  connectPromise: null,
  projectId: null,
  tagValues: new Map(),
  subscriptionValues: new Map(),
  datapointValues: new Map(),
  datapointSubscribed: new Set(),
  tagIdToProps: new Map(),
  subscriptionIdToProps: new Map(),
  onValueUpdate: null,
};
const pendingComponentCalls = new Map();

const connectionCache = new Map();
const queryCache = new Map();
const mappedValueCache = new Map();
const mappedValuePending = new Map();
const mappedDetails = new Map();
const datapointMetaCache = new Map();
const datapointMetaPending = new Map();

const unwrapApiData = (payload) => {
  if (payload && typeof payload === "object" && "data" in payload) {
    return payload.data;
  }
  return payload;
};

const normalizeGlobalValue = (detail) => {
  const type = detail?.type;
  const raw = detail?.default;
  if (type === "function") {
    if (typeof raw === "function") return raw;
    if (typeof raw === "string") {
      const text = raw.trim();
      if (!text) return () => undefined;
      try {
        if (
          text.startsWith("function") ||
          text.startsWith("async function") ||
          text.startsWith("(") ||
          text.startsWith("async (") ||
          text.startsWith("async(")
        ) {
          return new Function(`return (${text});`)();
        }
        return new Function(text);
      } catch (error) {
        return () => undefined;
      }
    }
    return () => undefined;
  }
  if (type === "set") {
    if (raw instanceof Set) return raw;
    if (Array.isArray(raw)) return new Set(raw);
    if (typeof raw === "string") {
      try {
        const parsed = JSON.parse(raw);
        return new Set(Array.isArray(parsed) ? parsed : []);
      } catch (error) {
        return new Set();
      }
    }
    return new Set();
  }
  if (type === "map") {
    if (raw instanceof Map) return raw;
    if (Array.isArray(raw)) return new Map(raw);
    if (raw && typeof raw === "object") return new Map(Object.entries(raw));
    if (typeof raw === "string") {
      try {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) return new Map(parsed);
        if (parsed && typeof parsed === "object") {
          return new Map(Object.entries(parsed));
        }
      } catch (error) {
        return new Map();
      }
    }
    return new Map();
  }
  if (type === "regexp") {
    if (raw instanceof RegExp) return raw;
    if (typeof raw === "string") {
      try {
        const match = raw.match(/^\/(.*)\/([gimsuy]*)$/);
        if (match) return new RegExp(match[1], match[2]);
        return new RegExp(raw);
      } catch (error) {
        return null;
      }
    }
  }
  return raw ?? null;
};

const buildPendingKey = (pageId, name) => `${pageId || "global"}::${name}`;
const buildDatapointCacheKey = (projectId, path) =>
  `${projectId || ""}::${path || ""}`;

const queueComponentCall = (pageId, name, method, args) => {
  if (!name) return;
  const key = buildPendingKey(pageId, name);
  if (!pendingComponentCalls.has(key)) {
    pendingComponentCalls.set(key, []);
  }
  pendingComponentCalls.get(key).push({ method, args });
};

const applyPendingCalls = (pageId, name, refInfo) => {
  if (!name || !refInfo) return;
  const key = buildPendingKey(pageId, name);
  const calls = pendingComponentCalls.get(key);
  if (!calls || calls.length === 0) return;
  calls.forEach((call) => {
    const fn = refInfo?.[call.method];
    if (typeof fn === "function") {
      fn(...(call.args || []));
    }
  });
  pendingComponentCalls.delete(key);
};

const buildComponentStub = (pageId, name) => ({
  setText: (...args) => queueComponentCall(pageId, name, "setText", args),
  setTableHeader: (...args) => queueComponentCall(pageId, name, "setTableHeader", args),
  setTableData: (...args) => queueComponentCall(pageId, name, "setTableData", args),
  setProps: (...args) => queueComponentCall(pageId, name, "setProps", args),
  setStyle: (...args) => queueComponentCall(pageId, name, "setStyle", args),
  button: (...args) => queueComponentCall(pageId, name, "button", args),
});

const getApiBase = () => {
  if (typeof __VITE_API_URL__ !== "undefined" && __VITE_API_URL__) {
    return __VITE_API_URL__;
  }
  if (typeof import.meta !== "undefined" && import.meta.env?.VITE_API_URL) {
    return import.meta.env.VITE_API_URL;
  }
  return "http://localhost:9099";
};

const buildSocketQuery = (projectId) => {
  const token = Storage.getToken();
  const tenantId = Storage.getTenantId();
  const query = new URLSearchParams();
  if (projectId) query.set("projectId", projectId);
  if (token) query.set("token", token);
  if (tenantId) query.set("tenantId", tenantId);
  return query;
};

const resolveConnection = async (projectId, name) => {
  if (connectionCache.has(name)) return connectionCache.get(name);
  if (!projectId) return null;
  const result = await datacenterApi.getConnections(projectId, {
    page: 1,
    limit: 200,
  });
  const data = unwrapApiData(result) || {};
  const connections = data.connections || data.items || data.list || [];
  const found = connections.find((item) => item.name === name);
  if (found) {
    connectionCache.set(name, found);
  }
  return found || null;
};

const resolveQuery = async (projectId, connectionId, queryName) => {
  if (!projectId) return null;
  const cacheKey = `${connectionId}`;
  let queries = queryCache.get(cacheKey);
  if (!queries) {
    const result = await datacenterApi.getQueries(projectId, {
      connectionId,
      page: 1,
      limit: 200,
    });
    const data = unwrapApiData(result) || {};
    queries = data.queries || data.items || data.list || [];
    queryCache.set(cacheKey, queries);
  }
  return queries.find((item) => item.name === queryName || item.id === queryName) || null;
};

const executeQueryByPath = async (projectId, path) => {
  const [sourceName, ...rest] = String(path || "").split(".");
  const field = rest.join(".");
  if (!sourceName || !field) return undefined;
  const connection = await resolveConnection(projectId, sourceName);
  if (!connection || connection.type !== "relational") return undefined;
  const query = await resolveQuery(projectId, connection.id, field);
  if (!query) return undefined;
  const result = await datacenterApi.executeQuery(query.id);
  const payload = unwrapApiData(result) || result;
  return payload?.data ?? payload;
};

const ensurePreviewMqttSocket = async (projectId) => {
  const apiBase = getApiBase();
  const query = new URLSearchParams();
  if (projectId) query.set("projectId", projectId);
  const socketUrl = query.toString() ? `${apiBase}?${query}` : apiBase;

  if (previewMqttState.socket && previewMqttState.projectId === projectId) {
    if (!previewMqttState.connectPromise) {
      previewMqttState.connectPromise = new Promise((resolve, reject) => {
        previewMqttState.socket.once("connect", resolve);
        previewMqttState.socket.once("connect_error", reject);
      });
    }
    await previewMqttState.connectPromise;
    return previewMqttState.socket;
  }

  if (previewMqttState.socket) {
    previewMqttState.socket.disconnect();
  }

  const queryParams = (() => {
    try {
      const urlObj = new URL(socketUrl);
      const params = {};
      for (const [key, value] of urlObj.searchParams.entries()) {
        params[key] = value;
      }
      return params;
    } catch (error) {
      return undefined;
    }
  })();

  const socket = io(apiBase.replace(/\/$/, ""), {
    path: "/socket.io",
    transports: ["websocket", "polling"],
    reconnection: true,
    reconnectionDelay: 1000,
    reconnectionDelayMax: 5000,
    reconnectionAttempts: Infinity,
    query: queryParams,
  });

  previewMqttState.socket = socket;
  previewMqttState.projectId = projectId;
  previewMqttState.connectPromise = new Promise((resolve, reject) => {
    socket.once("connect", resolve);
    socket.once("connect_error", reject);
  });

  socket.on("mqtt:tag:value", (data) => {
    if (!data?.tagId) return;
    const value = data.value ?? data.parsedValue ?? data.payload;
    previewMqttState.tagValues.set(data.tagId, value);
    previewMqttState.onValueUpdate?.("tag", data.tagId, value, data);
  });

  socket.on("connect", () => {
    previewMqttState.tagIdToProps.forEach((_props, tagId) => {
      socket.emit("mqtt:tag:subscribe", { tagId });
    });
    previewMqttState.subscriptionIdToProps.forEach((_props, subscriptionId) => {
      socket.emit("mqtt:subscribe", { subscriptionId });
    });
  });

  socket.on("mqtt:message", (data) => {
    if (!data?.subscriptionId) return;
    const value = data.payload ?? data.message ?? data.value ?? data;
    previewMqttState.subscriptionValues.set(data.subscriptionId, value);
    previewMqttState.onValueUpdate?.("subscription", data.subscriptionId, value, data);
  });

  socket.on("datapoint:value", (data) => {
    if (!data?.path) return;
    const value = data.value ?? data.payload ?? data;
    previewMqttState.datapointValues.set(data.path, value);
    previewMqttState.onValueUpdate?.("datapoint", data.path, value, data);
  });

  await previewMqttState.connectPromise;
  return socket;
};

const cacheDatapointMetaList = (projectId, items) => {
  if (!projectId || !Array.isArray(items)) return;
  items.forEach((item) => {
    if (!item?.path) return;
    const key = buildDatapointCacheKey(projectId, item.path);
    if (datapointMetaCache.has(key)) return;
    datapointMetaCache.set(key, {
      id: item.id,
      path: item.path,
      sourceType: item.sourceType || "",
      sourceId: item.sourceId || "",
      dataType: item.dataType || item.type || "",
    });
  });
};

const resolveDatapointMeta = async (projectId, path) => {
  if (!projectId || !path) return null;
  const key = buildDatapointCacheKey(projectId, path);
  if (datapointMetaCache.has(key)) return datapointMetaCache.get(key);
  if (datapointMetaPending.has(key)) return datapointMetaPending.get(key);

  const pending = (async () => {
    let page = 1;
    const pageSize = 300;
    let totalPages = 1;
    while (page <= totalPages) {
      const result = await datacenterApi.getDataPoints(projectId, {
        page,
        pageSize,
      });
      const data = unwrapApiData(result) || {};
      const items = data.datapoints || data.items || data.list || [];
      cacheDatapointMetaList(projectId, items);
      const hit = items.find((item) => item?.path === path);
      if (hit) {
        const meta = {
          id: hit.id,
          path: hit.path,
          sourceType: hit.sourceType || "",
          sourceId: hit.sourceId || "",
          dataType: hit.dataType || hit.type || "",
        };
        datapointMetaCache.set(key, meta);
        return meta;
      }
      const pagination = data.pagination || {};
      if (pagination.totalPages) {
        totalPages = Number(pagination.totalPages) || 1;
      } else if (items.length < pageSize) {
        break;
      } else {
        totalPages = Math.max(totalPages, page + 1);
      }
      page += 1;
    }
    return null;
  })();

  datapointMetaPending.set(key, pending);
  const result = await pending;
  datapointMetaPending.delete(key);
  return result;
};

const trackMqttProp = (prop, sourceType, sourceId) => {
  if (!prop || !sourceId) return;
  if (sourceType.includes("tag")) {
    if (!previewMqttState.tagIdToProps.has(sourceId)) {
      previewMqttState.tagIdToProps.set(sourceId, new Set());
    }
    previewMqttState.tagIdToProps.get(sourceId).add(prop);
  } else if (sourceType.includes("subscription")) {
    if (!previewMqttState.subscriptionIdToProps.has(sourceId)) {
      previewMqttState.subscriptionIdToProps.set(sourceId, new Set());
    }
    previewMqttState.subscriptionIdToProps.get(sourceId).add(prop);
  }
};

const resolveSourceInfo = async (projectId, detail) => {
  const source = detail?.source || {};
  const path = source.path;
  let sourceType = String(source.sourceType || "");
  let sourceId = source.sourceId || "";
  let datapointId = source.datapointId || "";
  if ((!sourceType || !sourceId || !datapointId) && path) {
    const meta = await resolveDatapointMeta(projectId, path);
    if (meta) {
      sourceType = sourceType || meta.sourceType || "";
      sourceId = sourceId || meta.sourceId || "";
      datapointId = datapointId || meta.id || "";
    }
  }
  return { path, sourceType, sourceId, datapointId };
};

const registerMqttMapping = (prop, detail) => {
  if (!detail?.mapped || detail?.source?.type !== "dataCenter") return;
  mappedDetails.set(prop, detail);
  const sourceType = String(detail?.source?.sourceType || "");
  const sourceId = detail?.source?.sourceId;
  if (!sourceId) return;
  trackMqttProp(prop, sourceType, sourceId);
};

const subscribeMqttSource = async (projectId, detail) => {
  if (!detail?.mapped || detail?.source?.type !== "dataCenter") return;
  const resolved = await resolveSourceInfo(projectId, detail);
  const sourceType = String(resolved.sourceType || "");
  const sourceId = resolved.sourceId;
  const path = resolved.path;
  const socket = await ensurePreviewMqttSocket(projectId);
  if (!socket?.connected) return;
  if (sourceId) {
    if (sourceType.includes("tag")) {
      socket.emit("mqtt:tag:subscribe", { tagId: sourceId });
    } else if (sourceType.includes("subscription")) {
      socket.emit("mqtt:subscribe", { subscriptionId: sourceId });
    }
  }
  if (path && !previewMqttState.datapointSubscribed.has(path)) {
    socket.emit("datapoint:subscribe", {
      projectId,
      paths: [path],
    });
    previewMqttState.datapointSubscribed.add(path);
  }
};

const resolveMappedGlobalValue = async (projectId, detail) => {
  const source = detail?.source;
  if (!source || source.type !== "dataCenter" || !source.path) {
    return normalizeGlobalValue(detail);
  }
  if (!projectId) return normalizeGlobalValue(detail);
  const fallbackValue = normalizeGlobalValue(detail);

  if (source.datapointId || source.sourceType || source.sourceId || source.path) {
    const resolved = await resolveSourceInfo(projectId, detail);
    const sourceType = String(resolved.sourceType || "");
    if (sourceType.includes("query") && source.sourceId) {
      try {
        const result = await datacenterApi.executeQuery(source.sourceId);
        const payload = unwrapApiData(result) || result;
        const value = payload?.data ?? payload;
        return value ?? fallbackValue;
      } catch (error) {
        try {
          const fallbackResult = await executeQueryByPath(projectId, source.path);
          if (fallbackResult !== undefined) return fallbackResult;
        } catch (fallbackError) {
          // ignore
        }
        return fallbackValue;
      }
    }
    if (sourceType.includes("subscription") || sourceType.includes("tag")) {
      await subscribeMqttSource(projectId, detail);
      if (sourceType.includes("tag")) {
        const value = previewMqttState.tagValues.get(resolved.sourceId || source.sourceId);
        if (value !== undefined) return value;
      }
      if (sourceType.includes("subscription")) {
        const value = previewMqttState.subscriptionValues.get(
          resolved.sourceId || source.sourceId
        );
        if (value !== undefined) return value;
      }
      if (resolved.path) {
        const pathValue = previewMqttState.datapointValues.get(resolved.path);
        return pathValue ?? fallbackValue;
      }
      return fallbackValue;
    }
    if (resolved.datapointId || source.datapointId) {
      try {
        const result = await datacenterApi.getDatapointValues(projectId, [
          resolved.datapointId || source.datapointId,
        ]);
        const payload = unwrapApiData(result) || result;
        const picked = extractDatapointValue(
          payload,
          resolved.datapointId || source.datapointId
        );
        return picked ?? payload?.data ?? payload ?? fallbackValue;
      } catch (error) {
        return fallbackValue;
      }
    }
  }

  if (String(source.path).startsWith("mqtt.")) {
    await subscribeMqttSource(projectId, detail);
    const pathValue = previewMqttState.datapointValues.get(source.path);
    return pathValue ?? fallbackValue;
  }

  const [sourceName, ...rest] = String(source.path).split(".");
  const field = rest.join(".");
  if (!sourceName || !field) return normalizeGlobalValue(detail);

  const connection = await resolveConnection(projectId, sourceName);
  if (!connection) return normalizeGlobalValue(detail);

  if (connection.type === "relational") {
    const query = await resolveQuery(projectId, connection.id, field);
    if (!query) return normalizeGlobalValue(detail);
    try {
      const result = await datacenterApi.executeQuery(query.id);
      const payload = unwrapApiData(result) || result;
      return payload?.data ?? payload ?? fallbackValue;
    } catch (error) {
      return fallbackValue;
    }
  }

  return fallbackValue;
};

const ensurePreviewDataService = async (projectId) => {
  const apiBase = getApiBase();
  const query = buildSocketQuery(projectId);
  const wsUrl = query.toString() ? `${apiBase}?${query}` : apiBase;
  if (
    previewDataServiceState.service &&
    previewDataServiceState.projectId === projectId
  ) {
    if (!previewDataServiceState.connectPromise) {
      previewDataServiceState.connectPromise =
        previewDataServiceState.service.connect(wsUrl) || Promise.resolve();
    }
    await previewDataServiceState.connectPromise;
    return previewDataServiceState.service;
  }

  if (previewDataServiceState.service) {
    previewDataServiceState.service.destroy();
  }
  const service = new DataService({ baseUrl: apiBase });
  previewDataServiceState.service = service;
  previewDataServiceState.projectId = projectId;
  previewDataServiceState.subscribed = new Set();
  previewDataServiceState.pending = new Map();
  previewDataServiceState.connectPromise =
    service.connect(wsUrl) || Promise.resolve();
  await previewDataServiceState.connectPromise;
  return service;
};

const subscribeDatapointPath = (service, path) => {
  if (!service || !path) return;
  if (previewDataServiceState.subscribed.has(path)) return;
  service.subscribe(path, (payload) => {
    const pending = previewDataServiceState.pending.get(path);
    if (pending) {
      pending.resolve(payload.value);
      previewDataServiceState.pending.delete(path);
    }
  });
  previewDataServiceState.subscribed.add(path);
};

const getSubscriptionValue = async (projectId, path) => {
  if (!path) return null;
  const service = await ensurePreviewDataService(projectId);
  subscribeDatapointPath(service, path);
  const cached = service.getValue(path);
  if (cached !== undefined) return cached;
  const existing = previewDataServiceState.pending.get(path);
  if (existing) return existing.promise;
  let resolver = null;
  const promise = new Promise((resolve) => {
    resolver = resolve;
    setTimeout(() => resolve(null), 2000);
  });
  previewDataServiceState.pending.set(path, { promise, resolve: resolver });
  return promise;
};

const extractDatapointValue = (payload, datapointId) => {
  if (!payload || !datapointId) return null;
  if (payload.values && typeof payload.values === "object") {
    if (datapointId in payload.values) return payload.values[datapointId];
  }
  if (Array.isArray(payload.values)) {
    const hit = payload.values.find((item) => item?.id === datapointId);
    if (hit) return hit.value ?? hit.currentValue ?? hit.dataValue ?? hit.lastValue ?? hit.rawValue;
  }
  if (Array.isArray(payload.datapoints)) {
    const hit = payload.datapoints.find((item) => item?.id === datapointId);
    if (hit) return hit.value ?? hit.currentValue ?? hit.dataValue ?? hit.lastValue ?? hit.rawValue;
  }
  if (Array.isArray(payload)) {
    const hit = payload.find((item) => item?.id === datapointId);
    if (hit) return hit.value ?? hit.currentValue ?? hit.dataValue ?? hit.lastValue ?? hit.rawValue;
  }
  if (payload && typeof payload === "object" && datapointId in payload) {
    return payload[datapointId];
  }
  return null;
};

const parseParamNames = (value) => {
  if (!value || typeof value !== "string") return [];
  return value
    .split(",")
    .map((name) => name.trim())
    .filter((name) => /^[A-Za-z_$][\w$]*$/.test(name));
};

export const initPreviewRuntime = (options) => {
  const { projectId, projectVariables, globalScripts } = options || {};
  const overrides = new Map();
  const timerIds = new Set();

  const updateMappedValue = (prop, nextValue, detail) => {
    const fallbackValue = normalizeGlobalValue(detail);
    const resolvedValue = nextValue ?? fallbackValue;
    const previous = mappedValueCache.has(prop)
      ? mappedValueCache.get(prop)
      : fallbackValue;
    mappedValueCache.set(prop, resolvedValue);
    if (previous !== resolvedValue) {
      triggerVariableChange(prop, resolvedValue, previous);
    }
  };

  const preloadMappedGlobals = async () => {
    const entries = Object.entries(projectVariables || {});
    const tasks = [];
    entries.forEach(([name, detail]) => {
      if (!detail?.mapped || detail?.source?.type !== "dataCenter") return;
      if (mappedValueCache.has(name) || mappedValuePending.has(name)) return;
      const sourceType = String(detail?.source?.sourceType || "");
      if (!sourceType.includes("query")) return;
      mappedValuePending.set(name, true);
      tasks.push(
        resolveMappedGlobalValue(projectId, detail)
          .then((value) => {
            updateMappedValue(name, value, detail);
          })
          .finally(() => {
            mappedValuePending.delete(name);
          })
      );
    });
    if (tasks.length > 0) {
      await Promise.allSettled(tasks);
    }
  };

  previewMqttState.onValueUpdate = (type, id, value) => {
    if (type === "datapoint") {
      mappedDetails.forEach((detail, name) => {
        const path = detail?.source?.path;
        if (path && path === id) {
          updateMappedValue(name, value, detail);
        }
      });
      return;
    }
    const detailMap =
      type === "tag"
        ? previewMqttState.tagIdToProps
        : previewMqttState.subscriptionIdToProps;
    const props = detailMap.get(id);
    if (!props || props.size === 0) return;
    props.forEach((prop) => {
      const detail = projectVariables?.[prop] || mappedDetails.get(prop);
      updateMappedValue(prop, value, detail);
    });
  };

  const globalsProxy = new Proxy(
    {},
    {
      get(_target, prop) {
        if (typeof prop !== "string") return undefined;
        if (overrides.has(prop)) return overrides.get(prop);
        const detail = projectVariables?.[prop];
        if (!detail) return undefined;
        if (detail?.mapped && detail?.source?.type === "dataCenter") {
          registerMqttMapping(prop, detail);
          const sourceType = String(detail?.source?.sourceType || "");
          const sourceId = detail?.source?.sourceId;
          const path = detail?.source?.path;
          if ((!sourceId || !sourceType) && path) {
            resolveSourceInfo(projectId, detail).then((resolved) => {
              if (resolved?.sourceId && resolved?.sourceType) {
                trackMqttProp(prop, resolved.sourceType, resolved.sourceId);
              }
            });
          }
          if (sourceId) {
            if (sourceType.includes("tag")) {
              const liveValue = previewMqttState.tagValues.get(sourceId);
              if (liveValue !== undefined) {
                updateMappedValue(prop, liveValue, detail);
                return liveValue;
              }
            }
            if (sourceType.includes("subscription")) {
              const liveValue = previewMqttState.subscriptionValues.get(sourceId);
              if (liveValue !== undefined) {
                updateMappedValue(prop, liveValue, detail);
                return liveValue;
              }
            }
          }
          if (path) {
            const pathValue = previewMqttState.datapointValues.get(path);
            if (pathValue !== undefined) {
              updateMappedValue(prop, pathValue, detail);
              return pathValue;
            }
          }
          if (!mappedValuePending.has(prop)) {
            mappedValuePending.set(prop, true);
            resolveMappedGlobalValue(projectId, detail)
              .then((value) => {
                updateMappedValue(
                  prop,
                  value ?? normalizeGlobalValue(detail),
                  detail
                );
              })
              .finally(() => {
                mappedValuePending.delete(prop);
              });
          }
          if (mappedValueCache.has(prop)) {
            return mappedValueCache.get(prop);
          }
          return normalizeGlobalValue(detail);
        }
        return normalizeGlobalValue(detail);
      },
      set(_target, prop, value) {
        if (typeof prop !== "string") return false;
        const prev = overrides.has(prop) ? overrides.get(prop) : projectVariables?.[prop]?.default;
        overrides.set(prop, value);
        if (prev !== value) {
          triggerVariableChange(prop, value, prev);
        }
        return true;
      },
    }
  );

  const buildCustomScripts = () => {
    const items = globalScripts?.custom?.items || [];
    const handlers = {};
    items.forEach((item) => {
      if (!item?.name) return;
      const paramNames = parseParamNames(item.params || item.args);
      handlers[item.name] = async (...args) => {
        const code = item.code || "";
        if (!code.trim()) return undefined;
        const scope = {
          $global: globalsProxy,
          customScripts: handlers,
          console,
          $event: undefined,
        };
        const localKeys = [...paramNames, ...Object.keys(scope)];
        const localValues = [
          ...paramNames.map((_, index) => args[index]),
          ...Object.values(scope),
        ];
        try {
          const runner = new Function(
            ...localKeys,
            `"use strict";\nreturn (async function() {\n${code}\n}).call(this);`
          );
          return await runner.call(undefined, ...localValues);
        } catch (error) {
          console.error(`[Preview] customScripts.${item.name} error:`, error);
          return undefined;
        }
      };
    });
    return handlers;
  };

  const customScripts = buildCustomScripts();

  const buildPageProxy = () =>
    new Proxy(
      {},
      {
        get(_target, prop) {
          if (typeof prop !== "string") return undefined;
          const pageMap = componentRefsByPage.get(prop);
          if (!pageMap) {
            return new Proxy(
              {},
              {
                get(_subTarget, name) {
                  if (typeof name !== "string") return undefined;
                  return buildComponentStub(prop, name);
                },
              }
            );
          }
          return new Proxy(
            {},
            {
              get(_subTarget, name) {
                if (typeof name !== "string") return undefined;
                return pageMap.get(name) || buildComponentStub(prop, name);
              },
            }
          );
        },
      }
    );

  const getComponentsProxy = (pageId) =>
    new Proxy(
      {},
      {
        get(_target, prop) {
          if (typeof prop !== "string") return undefined;
          if (prop === "pages") return buildPageProxy();
          const pageMap = componentRefsByPage.get(pageId);
          if (pageMap && pageMap.has(prop)) return pageMap.get(prop);
          if (componentRefsByName.has(prop)) return componentRefsByName.get(prop);
          return buildComponentStub(pageId, prop);
        },
      }
    );

  const runCode = async (code, event, thisArg, pageId) => {
    if (!code || !code.trim()) return;
    await preloadMappedGlobals();
    const components = getComponentsProxy(pageId || options?.pageId);
    const context = {
      $event: event,
      $global: globalsProxy,
      customScripts,
      components,
      console,
    };
    try {
      const runner = new Function(
        ...Object.keys(context),
        `"use strict";\nreturn (async function() {\n${code}\n}).call(this);`
      );
      return await runner.call(thisArg || null, ...Object.values(context));
    } catch (error) {
      console.error("[Preview] Script error:", error);
    }
  };

  const triggerVariableChange = async (name, value, previous) => {
    const items = globalScripts?.variableChanges?.items || [];
    const hits = items.filter(
      (item) => (item.variable || item.name) === name && item?.code
    );
    for (const item of hits) {
      await runCode(item.code, { name, value, previous });
    }
  };

  const start = async () => {
    const systemCode = globalScripts?.system?.startup?.code;
    await runCode(systemCode, { type: "startup" });
    const timers = globalScripts?.timers?.items || [];
    timers.forEach((item) => {
      const interval = Number(item.interval || item.time || 1000);
      if (!item?.code) return;
      const id = setInterval(() => {
        void runCode(item.code, { type: "timer", name: item.name });
      }, Math.max(100, interval));
      timerIds.add(id);
    });
  };

  const stop = async () => {
    timerIds.forEach((id) => clearInterval(id));
    timerIds.clear();
    const shutdownCode = globalScripts?.system?.shutdown?.code;
    await runCode(shutdownCode, { type: "shutdown" });
  };

  const registerComponentRef = (pageIdValue, name, refInfo) => {
    if (!pageIdValue || !name || !refInfo) return;
    if (!componentRefsByPage.has(pageIdValue)) {
      componentRefsByPage.set(pageIdValue, new Map());
    }
    componentRefsByPage.get(pageIdValue).set(name, refInfo);
    componentRefsByName.set(name, refInfo);
    applyPendingCalls(pageIdValue, name, refInfo);
    const alias = String(name).replace(/\d+$/, "");
    if (alias && alias !== name) {
      const pageMap = componentRefsByPage.get(pageIdValue);
      if (pageMap && !pageMap.has(alias)) {
        pageMap.set(alias, refInfo);
      }
      if (!componentRefsByName.has(alias)) {
        componentRefsByName.set(alias, refInfo);
      }
      applyPendingCalls(pageIdValue, alias, refInfo);
    }
  };

  const unregisterComponentRef = (pageIdValue, name, refInfo) => {
    if (!pageIdValue || !name) return;
    const pageMap = componentRefsByPage.get(pageIdValue);
    if (pageMap && pageMap.get(name) === refInfo) {
      pageMap.delete(name);
    }
    if (componentRefsByName.get(name) === refInfo) {
      componentRefsByName.delete(name);
    }
    const alias = String(name).replace(/\d+$/, "");
    if (alias && alias !== name) {
      if (pageMap && pageMap.get(alias) === refInfo) {
        pageMap.delete(alias);
      }
      if (componentRefsByName.get(alias) === refInfo) {
        componentRefsByName.delete(alias);
      }
    }
  };

  runtimeInstance = {
    globals: globalsProxy,
    customScripts,
    runCode,
    start,
    stop,
    registerComponentRef,
    unregisterComponentRef,
  };

  return runtimeInstance;
};

export const getPreviewRuntime = () => runtimeInstance;

export const clearPreviewRuntime = () => {
  runtimeInstance = null;
  mappedValueCache.clear();
  mappedValuePending.clear();
  mappedDetails.clear();
  componentRefsByPage.clear();
  componentRefsByName.clear();
  pendingComponentCalls.clear();
  if (previewDataServiceState.service) {
    previewDataServiceState.service.destroy();
  }
  previewDataServiceState.service = null;
  previewDataServiceState.connectPromise = null;
  previewDataServiceState.projectId = null;
  previewDataServiceState.subscribed.clear();
  previewDataServiceState.pending.clear();
  if (previewMqttState.socket) {
    previewMqttState.socket.disconnect();
  }
  previewMqttState.socket = null;
  previewMqttState.connectPromise = null;
  previewMqttState.projectId = null;
  previewMqttState.tagValues.clear();
  previewMqttState.subscriptionValues.clear();
  previewMqttState.datapointValues.clear();
  previewMqttState.datapointSubscribed.clear();
  previewMqttState.tagIdToProps.clear();
  previewMqttState.subscriptionIdToProps.clear();
  previewMqttState.onValueUpdate = null;
};
