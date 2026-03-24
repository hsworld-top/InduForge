/**
 * 预览运行时 Composable
 *
 * 从 NodeRenderer 抽取的预览脚本执行逻辑，包括数据查询、全局变量构建、自定义脚本等。
 *
 * @module ui/Canvas/composables/use-preview
 */

import { ref, watch, onMounted, onBeforeUnmount } from "vue";
import { normalizeGlobalValue } from "@/editor-core/utils/variable-utils";
import { getPreviewRuntime } from "@/ui/editors/page/preview/previewRuntime";
import { unwrapApiData } from "@/types/api";
import {
  extractDatapointValue,
  getQueryExecuteData,
  requireConnectionsPayload,
  requireQueriesPayload,
} from "@/utils/datapoint-payload";

export { extractDatapointValue };

/**
 * 创建预览运行时 composable
 * @param {Object} deps - 依赖项
 * @param {import('vue').Ref} deps.node - 节点 ref
 * @param {import('vue').Ref} deps.doc - 文档 ref
 * @param {import('vue').Ref} deps.currentPage - 当前页面 ref
 * @param {import('vue').Ref} deps.projectVariables - 项目变量 ref
 * @param {import('vue').Ref} deps.projectId - 项目 ID ref
 * @param {import('vue').Ref} deps.docVersion - 文档版本 ref
 * @param {import('vue').Ref} deps.globalScripts - 全局脚本 ref
 * @param {Object} deps.datacenterApi - 数据中心 API
 * @param {Function} deps.buildRefInfo - 构建引用信息函数
 * @param {import('vue').Ref} deps.readonly - 只读状态 ref
 * @param {import('vue').ComputedRef} deps.detailConfigText - detailConfig 文本 computed
 * @param {Function} [deps.resolveMenuConfigFromContent] - 解析 Menu 配置函数（可选）
 * @param {Function} [deps.applyMenuDslConfig] - 应用 Menu DSL 配置函数（可选）
 * @returns {{ runPreviewScript: Function, buildPreviewGlobals: Function, buildPreviewCustomScripts: Function, runDetailConfigScript: Function, scheduleDetailConfig: Function }}
 */
export function usePreview(deps) {
  const {
    node,
    doc,
    currentPage,
    projectVariables,
    projectId,
    docVersion,
    globalScripts,
    datacenterApi,
    buildRefInfo,
    readonly,
    detailConfigText,
    resolveMenuConfigFromContent,
    applyMenuDslConfig,
  } = deps;

  const connectionCache = new Map();
  const queryCache = new Map();
  const mappedValueCache = new Map();
  const previewOverrides = new Map();

  /**
   * 解析连接
   * @param {string} name - 连接名称
   * @returns {Promise<any>}
   */
  const resolveConnection = async (name) => {
    if (connectionCache.has(name)) return connectionCache.get(name);
    if (!projectId.value) return null;
    try {
      const result = await datacenterApi.getConnections(projectId.value, {
        page: 1,
        limit: 200,
      });
      const body = unwrapApiData(result);
      const connections = requireConnectionsPayload(body);
      const found = connections.find((item) => item.name === name);
      if (found) {
        connectionCache.set(name, found);
      }
      return found || null;
    } catch {
      return null;
    }
  };

  /**
   * 解析查询
   * @param {string} connectionId - 连接 ID
   * @param {string} queryName - 查询名称
   * @returns {Promise<any>}
   */
  const resolveQuery = async (connectionId, queryName) => {
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
      queries.find((item) => item.name === queryName || item.id === queryName) ||
      null
    );
  };

  /**
   * 通过路径执行查询
   * @param {string} path - 查询路径
   * @returns {Promise<any>}
   */
  const executeQueryByPath = async (path) => {
    const [sourceName, ...rest] = String(path || "").split(".");
    const field = rest.join(".");
    if (!sourceName || !field) return undefined;
    const connection = await resolveConnection(sourceName);
    if (!connection || connection.type !== "relational") return undefined;
    const query = await resolveQuery(connection.id, field);
    if (!query) return undefined;
    const result = await datacenterApi.executeQuery(query.id);
    return getQueryExecuteData(unwrapApiData(result));
  };

  /**
   * 解析映射的全局变量值
   * @param {string} name - 变量名
   * @param {any} detail - 变量详情
   * @returns {Promise<any>}
   */
  const resolveMappedGlobalValue = async (name, detail) => {
    const source = detail?.source;
    if (!source || source.type !== "dataCenter" || !source.path) {
      return normalizeGlobalValue(detail);
    }
    if (!projectId.value) return normalizeGlobalValue(detail);

    if (source.datapointId || source.sourceType || source.sourceId) {
      const sourceType = String(source.sourceType || "");
      if (sourceType.includes("query") && source.sourceId) {
        try {
          const result = await datacenterApi.executeQuery(source.sourceId);
          return getQueryExecuteData(unwrapApiData(result));
        } catch (error) {
          return normalizeGlobalValue(detail);
        }
      }
      if (source.datapointId) {
        const result = await datacenterApi.getDatapointValues(projectId.value, [
          source.datapointId,
        ]);
        const payload = unwrapApiData(result);
        const picked = extractDatapointValue(payload, source.datapointId);
        return picked ?? normalizeGlobalValue(detail);
      }
    }

    const [sourceName, ...rest] = String(source.path).split(".");
    const field = rest.join(".");
    if (!sourceName || !field) return normalizeGlobalValue(detail);

    const connection = await resolveConnection(sourceName);
    if (!connection) return normalizeGlobalValue(detail);

    if (connection.type === "relational") {
      const query = await resolveQuery(connection.id, field);
      if (!query) return normalizeGlobalValue(detail);
      const result = await datacenterApi.executeQuery(query.id);
      return getQueryExecuteData(unwrapApiData(result));
    }

    return normalizeGlobalValue(detail);
  };

  /**
   * 构建预览全局变量
   * @returns {Proxy}
   */
  const buildPreviewGlobals = () => {
    const vars = projectVariables.value || {};
    return new Proxy(
      {},
      {
        get(_target, prop) {
          if (typeof prop !== "string") return undefined;
          if (previewOverrides.has(prop)) {
            return previewOverrides.get(prop);
          }
          const detail = vars[prop];
          if (!detail) return undefined;
          if (detail?.mapped && detail?.source?.type === "dataCenter") {
            if (!mappedValueCache.has(prop)) {
              const promise = resolveMappedGlobalValue(prop, detail).catch(
                () => null,
              );
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

  /**
   * 解析参数名称
   * @param {string} value - 参数字符串
   * @returns {string[]}
   */
  const parseParamNames = (value) => {
    if (!value || typeof value !== "string") return [];
    return value
      .split(",")
      .map((name) => name.trim())
      .filter((name) => /^[A-Za-z_$][\w$]*$/.test(name));
  };

  /**
   * 构建预览自定义脚本
   * @param {Proxy} globals - 全局变量代理
   * @returns {Record<string, Function>}
   */
  const buildPreviewCustomScripts = (globals) => {
    const rawItems = globalScripts.value?.custom?.items;
    const items = Array.isArray(rawItems) ? rawItems : [];
    const handlers = {};
    items.forEach((item) => {
      if (!item?.name) return;
      const paramNames = parseParamNames(item.params || item.args);
      handlers[item.name] = async (...args) => {
        const code = item.code || "";
        if (!code.trim()) return undefined;
        const scope = {
          $global: globals,
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
            `"use strict";\nreturn (async () => {\n${code}\n})();`,
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

  /**
   * 运行预览脚本
   * @param {string} eventName - 事件名称
   * @param {any} event - 事件对象
   * @returns {Promise<any>}
   */
  const runPreviewScript = async (eventName, event) => {
    if (!node.value) return;
    const handlers = node.value?.events?.[eventName];
    if (!Array.isArray(handlers) || handlers.length === 0) return;
    const action = handlers.find((item) => item?.type === "script" || item?.code);
    if (!action) return;
    if (action?.enabled === false) return;
    const code = typeof action === "string" ? action : action?.code || "";
    if (!code.trim()) return;

    const runtime = getPreviewRuntime();
    if (runtime?.runCode) {
      const pageId = currentPage.value?.name || currentPage.value?.id;
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
      console,
    };

    try {
      const keys = Object.keys(context);
      const values = Object.values(context);
      const runner = new Function(
        ...keys,
        `"use strict";\nreturn (async function() {\n${code}\n}).call(this);`,
      );
      return await runner.call(instance || null, ...values);
    } catch (error) {
      console.error("[Preview] Script error:", error);
    }
  };

  // detailConfig 相关状态
  let detailConfigTimer = null;
  let lastDetailConfigKey = "";
  let isRunningDetailConfig = false;
  let detailConfigListener = null;

  /**
   * 运行 detailConfig 脚本
   * @param {string} code - 脚本代码
   * @returns {Promise<void>}
   */
  const runDetailConfigScript = async (code) => {
    if (!code || !code.trim()) return;
    const runtime = getPreviewRuntime();
    const pageId = currentPage.value?.name || currentPage.value?.id;
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
        console,
      };
      const keys = Object.keys(context);
      const values = Object.values(context);
      const runner = new Function(
        ...keys,
        `"use strict";\nreturn (async function() {\n${code}\n}).call(this);`,
      );
      await runner.call(instance || null, ...values);
    } catch (error) {
      console.error("[Preview] Detail script error:", error);
    } finally {
      isRunningDetailConfig = false;
    }
  };

  /**
   * 调度执行 detailConfig 脚本
   * @param {boolean} force - 是否强制执行
   */
  const scheduleDetailConfig = (force = false) => {
    const code = detailConfigText?.value;
    if (!code) return;
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
    detailConfigListener = (event) => {
      const payload = event?.detail || {};
      if (!payload || payload.nodeId !== node.value?.id) return;
      const code = String(payload.code || "").trim();
      if (!code) return;
      runDetailConfigScript(code);
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
