import { datacenterApi } from "@/services";

let runtimeInstance = null;

const connectionCache = new Map();
const queryCache = new Map();
const mappedValueCache = new Map();

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

const resolveMappedGlobalValue = async (projectId, detail) => {
  const source = detail?.source;
  if (!source || source.type !== "dataCenter" || !source.path) {
    return normalizeGlobalValue(detail);
  }
  if (!projectId) return normalizeGlobalValue(detail);
  const [sourceName, ...rest] = String(source.path).split(".");
  const field = rest.join(".");
  if (!sourceName || !field) return normalizeGlobalValue(detail);

  const connection = await resolveConnection(projectId, sourceName);
  if (!connection) return normalizeGlobalValue(detail);

  if (connection.type === "relational") {
    const query = await resolveQuery(projectId, connection.id, field);
    if (!query) return normalizeGlobalValue(detail);
    const result = await datacenterApi.executeQuery(query.id);
    const payload = unwrapApiData(result) || result;
    return payload?.data ?? payload;
  }

  return normalizeGlobalValue(detail);
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

  const globalsProxy = new Proxy(
    {},
    {
      get(_target, prop) {
        if (typeof prop !== "string") return undefined;
        if (overrides.has(prop)) return overrides.get(prop);
        const detail = projectVariables?.[prop];
        if (!detail) return undefined;
        if (detail?.mapped && detail?.source?.type === "dataCenter") {
          if (!mappedValueCache.has(prop)) {
            const promise = resolveMappedGlobalValue(projectId, detail).catch(() => null);
            mappedValueCache.set(prop, promise);
          }
          return mappedValueCache.get(prop);
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
            `"use strict";\nreturn (async () => {\n${code}\n})();`
          );
          return await runner(...localValues);
        } catch (error) {
          console.error(`[Preview] customScripts.${item.name} error:`, error);
          return undefined;
        }
      };
    });
    return handlers;
  };

  const customScripts = buildCustomScripts();

  const runCode = async (code, event) => {
    if (!code || !code.trim()) return;
    const context = {
      $event: event,
      $global: globalsProxy,
      customScripts,
      console,
    };
    try {
      const runner = new Function(
        ...Object.keys(context),
        `"use strict";\nreturn (async () => {\n${code}\n})();`
      );
      return await runner(...Object.values(context));
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

  runtimeInstance = {
    globals: globalsProxy,
    customScripts,
    runCode,
    start,
    stop,
  };

  return runtimeInstance;
};

export const getPreviewRuntime = () => runtimeInstance;

export const clearPreviewRuntime = () => {
  runtimeInstance = null;
  mappedValueCache.clear();
};
