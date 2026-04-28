/**
 * 预览运行时 Composable
 *
 * 从 NodeRenderer 抽取的预览脚本执行逻辑，包括数据查询、全局变量构建、自定义脚本等。
 *
 * @module ui/Canvas/composables/use-preview
 */

import type {
  PreviewScriptItem,
  ScriptSectionWithItems,
  UsePreviewDeps,
  UsePreviewReturn,
} from "@/ui/editors/page/preview/preview-runtime.types";
import { onBeforeUnmount, onMounted, watch } from "vue";
import { normalizeGlobalValue } from "@/editor-core/utils/variable-utils";
import { unwrapApiData } from "@/types/api";
import { getPreviewRuntime } from "@/ui/editors/page/preview/previewRuntime";
import {
  extractDatapointValue,
  getQueryExecuteData,
  requireConnectionsPayload,
  requireQueriesPayload,
} from "@/utils/datapoint-payload";

const PARAM_NAME_RE = /^[A-Z_$][\w$]*$/i;

/**
 * 仅接受 { items }；顶层数组等旧格式返回空（与 previewRuntime 一致）。
 */
function itemsFromScriptSection(section: unknown): PreviewScriptItem[] {
  if (!section || typeof section !== "object" || Array.isArray(section)) return [];
  const items = (section as ScriptSectionWithItems).items;
  return Array.isArray(items) ? (items as PreviewScriptItem[]) : [];
}

export { extractDatapointValue };

/** 创建预览运行时 composable */
export function usePreview(deps: UsePreviewDeps): UsePreviewReturn {
  const {
    node,
    doc: _doc,
    currentPage,
    projectVariables,
    projectId,
    docVersion: _docVersion,
    globalScripts,
    datacenterApi,
    buildRefInfo,
    readonly,
    detailConfigText,
    resolveMenuConfigFromContent,
    applyMenuDslConfig,
  } = deps;
  void _doc;
  void _docVersion;

  const connectionCache = new Map<string, unknown>();
  const queryCache = new Map<string, unknown[]>();
  const mappedValueCache = new Map<string, Promise<unknown>>();
  const previewOverrides = new Map<string, unknown>();

  const resolveConnection = async (name: string): Promise<unknown | null> => {
    if (connectionCache.has(name)) return connectionCache.get(name);
    if (!projectId.value) return null;
    try {
      const result = await datacenterApi.getConnections(projectId.value, {
        page: 1,
        limit: 200,
      });
      const body = unwrapApiData(result);
      const connections = requireConnectionsPayload(body);
      const found = connections.find((item: unknown) => (item as { name?: string }).name === name);
      if (found) {
        connectionCache.set(name, found);
      }
      return found || null;
    } catch {
      return null;
    }
  };

  const resolveQuery = async (
    connectionId: string | undefined,
    queryName: string,
  ): Promise<unknown | null> => {
    if (!projectId.value) return null;
    const cacheKey = `${connectionId}`;
    let queries = queryCache.get(cacheKey);
    if (!queries) {
      try {
        const result = await datacenterApi.getQueries(projectId.value, {
          connectionId,
          page: 1,
          limit: 200,
        });
        const body = unwrapApiData(result);
        queries = requireQueriesPayload(body);
        queryCache.set(cacheKey, queries);
      } catch {
        queries = [];
        queryCache.set(cacheKey, queries);
      }
    }
    return (
      queries.find((item: unknown) => {
        const row = item as { name?: string; id?: string };
        return row.name === queryName || row.id === queryName;
      }) || null
    );
  };

  const resolveMappedGlobalValue = async (_name: string, detail: unknown): Promise<unknown> => {
    const d = detail as Record<string, unknown>;
    const source = d?.source as Record<string, unknown> | undefined | null;
    if (!source || source.type !== "dataCenter" || !source.path) {
      return normalizeGlobalValue(detail);
    }
    if (!projectId.value) return normalizeGlobalValue(detail);

    if (source.datapointId || source.sourceType || source.sourceId) {
      const sourceType = String(source.sourceType || "");
      if (sourceType.includes("query") && source.sourceId) {
        try {
          const result = await datacenterApi.executeQuery(String(source.sourceId));
          return getQueryExecuteData(unwrapApiData(result));
        } catch {
          return normalizeGlobalValue(detail);
        }
      }
      if (source.datapointId) {
        const dpId = String(source.datapointId);
        const result = await datacenterApi.getDatapointValues(projectId.value, [dpId]);
        const payload = unwrapApiData(result);
        const picked = extractDatapointValue(payload, dpId);
        return picked ?? normalizeGlobalValue(detail);
      }
    }

    const [sourceName, ...rest] = String(source.path).split(".");
    const field = rest.join(".");
    if (!sourceName || !field) return normalizeGlobalValue(detail);

    const connection = (await resolveConnection(sourceName)) as {
      type?: string;
      id?: string;
    } | null;
    if (!connection) return normalizeGlobalValue(detail);

    if (connection.type === "relational") {
      const query = (await resolveQuery(connection.id, field)) as {
        id?: string;
      } | null;
      if (!query?.id) return normalizeGlobalValue(detail);
      const result = await datacenterApi.executeQuery(query.id);
      return getQueryExecuteData(unwrapApiData(result));
    }

    return normalizeGlobalValue(detail);
  };

  /**
   * 构建预览全局变量
   * @returns {Proxy}
   */
  const buildPreviewGlobals = (): object => {
    const vars = (projectVariables.value || {}) as Record<string, unknown>;
    return new Proxy(
      {},
      {
        get(_target, prop) {
          if (typeof prop !== "string") return undefined;
          if (previewOverrides.has(prop)) {
            return previewOverrides.get(prop);
          }
          const detail = vars[prop] as Record<string, unknown> | undefined;
          if (!detail) return undefined;
          const src = detail.source as Record<string, unknown> | undefined;
          if (detail.mapped && src?.type === "dataCenter") {
            if (!mappedValueCache.has(prop)) {
              const promise = resolveMappedGlobalValue(prop, detail).catch(() => null);
              mappedValueCache.set(prop, promise);
            }
            return mappedValueCache.get(prop);
          }
          return normalizeGlobalValue(detail);
        },
        set(_target, prop, value) {
          if (typeof prop !== "string") return false;
          previewOverrides.set(prop, value);
          return true;
        },
      },
    );
  };

  const parseParamNames = (value: unknown): string[] => {
    if (!value || typeof value !== "string") return [];
    return value
      .split(",")
      .map((name) => name.trim())
      .filter((name) => PARAM_NAME_RE.test(name));
  };

  const buildPreviewCustomScripts = (
    globals: object,
  ): Record<string, (...args: unknown[]) => Promise<unknown>> => {
    const items = itemsFromScriptSection(globalScripts.value?.custom);
    const handlers: Record<string, (...args: unknown[]) => Promise<unknown>> = {};
    items.forEach((item) => {
      if (!item.name) return;
      const name = String(item.name);
      const paramNames = parseParamNames(item.params || item.args);
      handlers[name] = async (...args: unknown[]) => {
        const code = String(item.code || "");
        if (!code.trim()) return undefined;
        const scope = {
          $global: globals,
          customScripts: handlers,
          console,
          $event: undefined,
        };
        const localKeys = [...paramNames, ...Object.keys(scope)];
        const localValues = [...paramNames.map((_, index) => args[index]), ...Object.values(scope)];
        try {
          // eslint-disable-next-line no-new-func
          const runner = new Function(
            ...localKeys,
            `"use strict";\nreturn (async () => {\n${code}\n})();`,
          ) as (...args: unknown[]) => Promise<unknown>;
          return await runner(...localValues);
        } catch (error) {
          console.error(`[Preview] customScripts.${name} error:`, error);
          return undefined;
        }
      };
    });
    return handlers;
  };

  const runPreviewScript = async (eventName: string, event: unknown): Promise<unknown> => {
    if (!node.value) return;
    const handlers = node.value?.events?.[eventName];
    if (!Array.isArray(handlers) || handlers.length === 0) return;
    const action = handlers.find((raw: unknown) => {
      const it = raw as { type?: string; code?: string };
      return it?.type === "script" || Boolean(it?.code);
    });
    if (!action) return;
    const act = action as { enabled?: boolean; code?: string };
    if (act?.enabled === false) return;
    const code = typeof action === "string" ? action : String(act?.code || "");
    if (!code.trim()) return;

    const runtime = getPreviewRuntime();
    if (runtime?.runCode) {
      const pageId = currentPage.value?.name || currentPage.value?.id || null;
      const instance = buildRefInfo();
      return await runtime.runCode(code, event, instance, pageId);
    }
    const instance = buildRefInfo();
    const globals = buildPreviewGlobals();
    const customScripts = buildPreviewCustomScripts(globals);
    const context = {
      $event: event,
      $global: globals,
      customScripts,
      components: instance || {},
      console,
    };

    try {
      const keys = Object.keys(context);
      const values = Object.values(context);
      // eslint-disable-next-line no-new-func
      const runner = new Function(
        ...keys,
        `"use strict";\nreturn (async function() {\n${code}\n}).call(this);`,
      ) as (...args: unknown[]) => Promise<unknown>;
      return await runner.call(instance || null, ...values);
    } catch (error) {
      console.error("[Preview] Script error:", error);
    }
  };

  let detailConfigTimer: ReturnType<typeof setTimeout> | null = null;
  let lastDetailConfigKey = "";
  let isRunningDetailConfig = false;
  let detailConfigListener: ((event: Event) => void) | null = null;

  const runDetailConfigScript = async (code: string): Promise<void> => {
    if (!code || !code.trim()) return;
    const runtime = getPreviewRuntime();
    const pageId = currentPage.value?.name || currentPage.value?.id || null;
    const instance = buildRefInfo();
    if (!instance) return;
    if (node.value?.type === "Menu" && resolveMenuConfigFromContent && applyMenuDslConfig) {
      const config = resolveMenuConfigFromContent(code);
      if (config) applyMenuDslConfig(config);
    }
    isRunningDetailConfig = true;
    try {
      if (readonly.value && runtime?.runCode) {
        await runtime.runCode(code, { type: "detail" }, instance, pageId);
        return;
      }
      const globals = buildPreviewGlobals();
      const customScripts = buildPreviewCustomScripts(globals);
      const context = {
        $event: { type: "detail" },
        $global: globals,
        customScripts,
        components: instance || {},
        console,
      };
      const keys = Object.keys(context);
      const values = Object.values(context);
      // eslint-disable-next-line no-new-func
      const runner = new Function(
        ...keys,
        `"use strict";\nreturn (async function() {\n${code}\n}).call(this);`,
      ) as (...args: unknown[]) => Promise<unknown>;
      await runner.call(instance || null, ...values);
    } catch (error) {
      console.error("[Preview] Detail script error:", error);
    } finally {
      isRunningDetailConfig = false;
    }
  };

  const scheduleDetailConfig = (force = false): void => {
    const code = detailConfigText?.value;
    if (!code) return;
    // Collapse/Tabs 的结构已迁移到可视化 props；旧 detailConfig 自动重放会覆盖用户配置。
    if (node.value?.type === "Collapse" && Array.isArray(node.value?.props?.items)) return;
    if (node.value?.type === "Tabs" && Array.isArray(node.value?.props?.tabs)) return;
    const key = `${node.value?.id || ""}::${code}`;
    if (!force && key === lastDetailConfigKey) return;
    lastDetailConfigKey = key;
    if (detailConfigTimer) {
      clearTimeout(detailConfigTimer);
    }
    detailConfigTimer = setTimeout(() => {
      runDetailConfigScript(code);
    }, 120);
  };

  // 监听 detailConfig 变化
  if (detailConfigText) {
    watch(
      () => [readonly?.value, node.value?.id, detailConfigText.value],
      () => {
        scheduleDetailConfig(true);
      },
      { immediate: true },
    );
  }

  // 监听 detail-config 事件
  onMounted(() => {
    detailConfigListener = (event: Event) => {
      const payload = (event as CustomEvent<Record<string, unknown>>)?.detail || {};
      if (!payload || payload.nodeId !== node.value?.id) return;
      const code = String(payload.code || "").trim();
      if (!code) return;
      // Collapse/Tabs 的结构配置由属性面板直接落到 props，避免事件重放把 DSL 顶层字段写回 props。
      if (node.value?.type === "Collapse" && Array.isArray(node.value?.props?.items)) return;
      if (node.value?.type === "Tabs" && Array.isArray(node.value?.props?.tabs)) return;
      void runDetailConfigScript(code);
    };
    window.addEventListener("designer:detail-config", detailConfigListener);
  });

  onBeforeUnmount(() => {
    if (detailConfigTimer) {
      clearTimeout(detailConfigTimer);
      detailConfigTimer = null;
    }
    if (detailConfigListener) {
      window.removeEventListener("designer:detail-config", detailConfigListener);
      detailConfigListener = null;
    }
  });

  return {
    runPreviewScript,
    buildPreviewGlobals,
    buildPreviewCustomScripts,
    runDetailConfigScript,
    scheduleDetailConfig,
    isRunningDetailConfig: () => isRunningDetailConfig,
  };
}
