/**
 * 数据绑定系统 - 统一导出
 *
 * 包含：
 * - 类型与工厂：createVarDefinition、createDatapointBinding 等
 * - Transform：数值、字符串、日期、映射等转换
 * - VarsStore、ExpressionEngine、MockDataProvider、DataService、BindingResolver
 * - createDataBindingSystem：一键创建完整数据绑定系统
 */

// 类型定义与工厂函数
export {
  createVarDefinition,
  createDatapointBinding,
  createVarBinding,
  createExprBinding,
  getBindingKind,
  isValidValue,
} from "./types.js";

// Transform 操作
export {
  transformRegistry,
  registerTransform,
  executeTransform,
  applyTransforms,
  // 数值转换
  toFixed,
  round,
  floor,
  ceil,
  abs,
  clamp,
  percent,
  // 字符串转换
  prefix,
  suffix,
  format,
  truncate,
  toUpperCase,
  toLowerCase,
  trim,
  // 日期转换
  dateFormat,
  fromNow,
  // 映射
  map,
  boolMap,
  rangeMap,
  // 空值处理
  ifNull,
  ifEmpty,
  ifNaN,
  // 数组
  length,
  first,
  last,
  join,
} from "./transforms.js";

// 变量存储
export { VarsStore } from "./VarsStore.js";

// 表达式引擎
export {
  ExpressionEngine,
  defaultEngine,
  evaluate,
  evaluateTemplate,
} from "./ExpressionEngine.js";

// Mock 数据提供者
export {
  MockDataProvider,
  createMockDataProvider,
} from "./MockDataProvider.js";

// 诊断存储
export {
  DiagnosticsStore,
  createDiagnosticsStore,
} from "./DiagnosticsStore.js";

// 数据服务
export { DataService, createDataService } from "./DataService.js";

// 绑定解析器
export { BindingResolver, createBindingResolver } from "./BindingResolver.js";

// 数据点注册表
export {
  DatapointRegistry,
  DatapointRegistryEvents,
  datapointRegistry,
} from "./datapoint-registry";

// ==================== 工厂函数 ====================

import { VarsStore } from "./VarsStore.js";
import { ExpressionEngine } from "./ExpressionEngine.js";
import {
  MockDataProvider,
  createMockDataProvider,
} from "./MockDataProvider.js";
import {
  DiagnosticsStore,
  createDiagnosticsStore,
} from "./DiagnosticsStore.js";
import { DataService, createDataService } from "./DataService.js";
import { BindingResolver, createBindingResolver } from "./BindingResolver.js";

/**
 * @typedef {import('./types.js').DataMode} DataMode
 * @typedef {import('./types.js').VarsDefinitions} VarsDefinitions
 * @typedef {import('./types.js').DataServiceOptions} DataServiceOptions
 */

/**
 * 数据绑定系统实例
 * @typedef {Object} DataBindingSystem
 * @property {VarsStore} varsStore - 变量存储
 * @property {ExpressionEngine} expressionEngine - 表达式引擎
 * @property {MockDataProvider} mockProvider - Mock 数据提供者
 * @property {DiagnosticsStore} diagnosticsStore - 诊断存储
 * @property {DataService} dataService - 数据服务
 * @property {BindingResolver} bindingResolver - 绑定解析器
 * @property {function(): void} destroy - 销毁函数
 */

/**
 * 创建完整的数据绑定系统
 * @param {Object} [options] - 配置选项
 * @param {DataMode} [options.mode='edit'] - 数据模式
 * @param {string} [options.pageId] - 当前页面 ID
 * @param {VarsDefinitions} [options.varsDefinitions] - 变量定义
 * @param {DataServiceOptions} [options.dataServiceOptions] - 数据服务配置
 * @returns {DataBindingSystem}
 */
export function createDataBindingSystem(options = {}) {
  const {
    mode = "edit",
    pageId = null,
    varsDefinitions = { global: {}, pages: {} },
    dataServiceOptions = {},
  } = options;

  // 创建各组件
  const varsStore = new VarsStore(varsDefinitions);
  const expressionEngine = new ExpressionEngine();
  const mockProvider = new MockDataProvider();
  const diagnosticsStore = new DiagnosticsStore(dataServiceOptions);
  const dataService = new DataService(dataServiceOptions);

  // 创建绑定解析器
  const bindingResolver = new BindingResolver(
    {
      varsStore,
      expressionEngine,
      mockProvider,
      diagnosticsStore,
      dataService,
    },
    { mode, pageId },
  );

  // 设置当前页面
  if (pageId) {
    varsStore.setCurrentPage(pageId);
  }

  return {
    varsStore,
    expressionEngine,
    mockProvider,
    diagnosticsStore,
    dataService,
    bindingResolver,

    /**
     * 销毁系统
     */
    destroy() {
      bindingResolver.destroy();
      dataService.destroy();
      diagnosticsStore.destroy();
      mockProvider.destroy();
      varsStore.removeAllListeners();
    },
  };
}

/**
 * 默认导出所有类和工厂函数
 */
export default {
  // 类
  VarsStore,
  ExpressionEngine,
  MockDataProvider,
  DiagnosticsStore,
  DataService,
  BindingResolver,

  // 工厂函数
  createDataBindingSystem,
  createBindingResolver,
  createDataService,
  createDiagnosticsStore,
  createMockDataProvider,
};
