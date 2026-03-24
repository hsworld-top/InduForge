/**
 * 工程设置与 API 负载规范化（变量、全局脚本、菜单默认值等）
 *
 * 从 editor-store 拆出，供 store 与潜在其它模块复用。
 *
 * @module stores/editor/normalize-settings
 */

/**
 * 解包 API 响应
 * @param {*} payload - 原始响应
 * @returns {*}
 */
const unwrapApiData = (payload) => {
  if (payload && typeof payload === "object" && "success" in payload) {
    return payload.data;
  }
  return payload;
};
const getDefaultGlobalScripts = () => ({
  system: {
    startup: { code: "" },
    shutdown: { code: "" },
  },
  timers: { groups: [], items: [] },
  variableChanges: { groups: [], items: [] },
  custom: { groups: [], items: [] },
});

/**
 * 规范化变量定义
 * 将 source 字符串转为 { type, path }，补齐 dataCenter 类型与 mapped 标记
 * @param {Object} detail - 变量定义
 * @returns {Object}
 */
const normalizeVariableDef = (detail) => {
  if (!detail || typeof detail !== "object") return detail;
  const next = { ...detail };
  // 兼容旧版：source 为字符串时转为 { type: 'dataCenter', path }
  if (typeof next.source === "string") {
    next.source = { type: "dataCenter", path: next.source };
    next.mapped = true;
  }
  const mappedPath = next.mappedPath || next.sourcePath || next.path;
  if (!next.source && mappedPath) {
    next.source = { type: "dataCenter", path: mappedPath };
    next.mapped = true;
  }
  if (next.source && typeof next.source === "object") {
    if (!next.source.type && (next.mapped || next.source.path)) {
      next.source.type = "dataCenter";
    }
    if (!next.mapped && next.source.type === "dataCenter") {
      next.mapped = true;
    }
  }
  return next;
};

/**
 * 规范化全局变量配置
 * 支持 definitions + groups 或扁平对象两种结构（不再与 variables 接口合并兜底）
 * @param {*} raw - 原始配置
 * @returns {{ definitions: Object, groups: Array }}
 */
const normalizeGlobalVariables = (raw) => {
  if (!raw || typeof raw !== "object") {
    throw new Error("globalVariables 无效");
  }
  if (raw.definitions || raw.groups) {
    const definitions =
      raw.definitions && typeof raw.definitions === "object"
        ? raw.definitions
        : {};
    const normalizedDefinitions = {};
    Object.entries(definitions).forEach(([name, detail]) => {
      normalizedDefinitions[name] = normalizeVariableDef(detail);
    });
    return {
      definitions: normalizedDefinitions,
      groups: Array.isArray(raw.groups) ? raw.groups : [],
    };
  }
  const normalizedDefinitions = {};
  Object.entries(raw).forEach(([name, detail]) => {
    normalizedDefinitions[name] = normalizeVariableDef(detail);
  });
  return { definitions: normalizedDefinitions, groups: [] };
};

/**
 * 规范化全局脚本配置
 * 合并 system、timers、variableChanges、custom 各组，确保结构完整
 * @param {*} raw - 原始配置
 * @returns {Object}
 */
const normalizeGlobalScripts = (raw) => {
  const system = raw && raw.system ? raw.system : {};
  const timers = raw && raw.timers ? raw.timers : {};
  const variableChanges = raw && raw.variableChanges ? raw.variableChanges : {};
  const custom = raw && raw.custom ? raw.custom : {};

  return {
    system: {
      startup: { code: system.startup?.code || "" },
      shutdown: { code: system.shutdown?.code || "" },
    },
    timers: {
      groups: Array.isArray(timers.groups) ? timers.groups : [],
      items: Array.isArray(timers.items) ? timers.items : [],
    },
    variableChanges: {
      groups: Array.isArray(variableChanges.groups)
        ? variableChanges.groups
        : [],
      items: Array.isArray(variableChanges.items) ? variableChanges.items : [],
    },
    custom: {
      groups: Array.isArray(custom.groups) ? custom.groups : [],
      items: Array.isArray(custom.items) ? custom.items : [],
    },
  };
};

/**
 * 获取 Menu 组件默认详细配置
 * @returns {string}
 */
const getMenuDefaultDetailConfig = () =>
  "this.menu({\n" +
  '  id: "menuNav",\n' +
  '  label: "菜单基础配置",\n' +
  '  type: "Menu",\n' +
  "  props: {\n" +
  '    defaultActive: "2",\n' +
  "    items: [\n" +
  '      { index: "1", label: "导航一", icon: "location" },\n' +
  '      { index: "2", label: "导航二", icon: "menu" },\n' +
  '      { index: "3", label: "导航三", icon: "document", disabled: true },\n' +
  '      { index: "4", label: "导航四", icon: "setting" },\n' +
  "    ],\n" +
  "  },\n" +
  "});";

/**
 * 获取 Menu 组件默认属性
 * @returns {Record<string, any>}
 */
const getMenuDefaultProps = () => ({
  defaultActive: "2",
  items: [
    { index: "1", label: "导航一", icon: "location" },
    { index: "2", label: "导航二", icon: "menu" },
    { index: "3", label: "导航三", icon: "document", disabled: true },
    { index: "4", label: "导航四", icon: "setting" },
  ],
});

export {
  unwrapApiData,
  getDefaultGlobalScripts,
  normalizeVariableDef,
  normalizeGlobalVariables,
  normalizeGlobalScripts,
  getMenuDefaultDetailConfig,
  getMenuDefaultProps,
};
