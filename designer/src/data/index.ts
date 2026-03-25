/**
 * 数据绑定系统 - 统一导出
 */

import { BindingResolver, createBindingResolver } from "./BindingResolver.ts";
import { createDataService, DataService } from "./DataService.ts";
import { createDiagnosticsStore, DiagnosticsStore } from "./DiagnosticsStore.ts";
import { ExpressionEngine } from "./ExpressionEngine.ts";
import { createMockDataProvider, MockDataProvider } from "./MockDataProvider.ts";
import { VarsStore } from "./VarsStore.ts";

export { BindingResolver, createBindingResolver } from "./BindingResolver.ts";

export {
  DatapointRegistry,
  datapointRegistry,
  DatapointRegistryEvents,
} from "./datapoint-registry";

export { createDataService, DataService } from "./DataService.ts";

export { createDiagnosticsStore, DiagnosticsStore } from "./DiagnosticsStore.ts";

export { defaultEngine, evaluate, evaluateTemplate, ExpressionEngine } from "./ExpressionEngine.ts";

export { createMockDataProvider, MockDataProvider } from "./MockDataProvider.ts";

export {
  abs,
  applyTransforms,
  boolMap,
  ceil,
  clamp,
  dateFormat,
  executeTransform,
  first,
  floor,
  format,
  fromNow,
  ifEmpty,
  ifNaN,
  ifNull,
  join,
  last,
  length,
  map,
  percent,
  prefix,
  rangeMap,
  registerTransform,
  round,
  suffix,
  toFixed,
  toLowerCase,
  toUpperCase,
  transformRegistry,
  trim,
  truncate,
} from "./transforms.ts";

export {
  createDatapointBinding,
  createExprBinding,
  createVarBinding,
  createVarDefinition,
  getBindingKind,
  isValidValue,
} from "./types.ts";

export { VarsStore } from "./VarsStore.ts";

export function createDataBindingSystem(
  options: {
    mode?: string;
    pageId?: string | null;
    varsDefinitions?: {
      global: Record<string, unknown>;
      pages: Record<string, Record<string, unknown>>;
    };
    dataServiceOptions?: Record<string, unknown>;
  } = {},
) {
  const {
    mode = "edit",
    pageId = null,
    varsDefinitions = { global: {}, pages: {} },
    dataServiceOptions = {},
  } = options;

  const varsStore = new VarsStore(varsDefinitions);
  const expressionEngine = new ExpressionEngine();
  const mockProvider = new MockDataProvider();
  const diagnosticsStore = new DiagnosticsStore(dataServiceOptions);
  const dataService = new DataService(dataServiceOptions);

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

    destroy() {
      bindingResolver.destroy();
      dataService.destroy();
      diagnosticsStore.destroy();
      mockProvider.destroy();
      varsStore.removeAllListeners();
    },
  };
}

export default {
  VarsStore,
  ExpressionEngine,
  MockDataProvider,
  DiagnosticsStore,
  DataService,
  BindingResolver,
  createDataBindingSystem,
  createBindingResolver,
  createDataService,
  createDiagnosticsStore,
  createMockDataProvider,
};
