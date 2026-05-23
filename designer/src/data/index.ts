/**
 * 数据绑定系统 - 统一导出
 */

import type { VarsDefinitions } from './types.ts'
import { BindingResolver, createBindingResolver } from './BindingResolver.ts'
import { createDataService, DataService } from './DataService.ts'
import { createDiagnosticsStore, DiagnosticsStore } from './DiagnosticsStore.ts'
import { ExpressionEngine } from './ExpressionEngine.ts'
import { createMockDataProvider, MockDataProvider } from './MockDataProvider.ts'
import { VarsStore } from './VarsStore.ts'

export { BindingResolver, createBindingResolver } from './BindingResolver.ts'

export { DatapointRegistry, datapointRegistry, DatapointRegistryEvents } from './datapoint-registry'

export { createDataService, DataService } from './DataService.ts'

export { createDiagnosticsStore, DiagnosticsStore } from './DiagnosticsStore.ts'

export { defaultEngine, evaluate, evaluateTemplate, ExpressionEngine } from './ExpressionEngine.ts'

export { createMockDataProvider, MockDataProvider } from './MockDataProvider.ts'

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
} from './transforms.ts'

export {
  createDatapointBinding,
  createExprBinding,
  createVarBinding,
  createVarDefinition,
  type DataMode,
  type DatapointStatus,
  type DatapointStatusInfo,
  type DiagnosticInfo,
  type DiagnosticsSummary,
  type ExpressionContext,
  type ExpressionResult,
  getBindingKind,
  isValidValue,
  type ResolvedBinding,
  type VarsContext,
  type VarsDefinitions,
} from './types.ts'

export { VarsStore } from './VarsStore.ts'

export function createDataBindingSystem(
  options: {
    mode?: 'edit' | 'preview' | 'runtime'
    pageId?: string | null
    varsDefinitions?: VarsDefinitions
    dataServiceOptions?: Record<string, unknown>
  } = {},
) {
  const mode = options.mode ?? 'edit'
  const pageId = options.pageId ?? null
  const varsDefinitions: VarsDefinitions = options.varsDefinitions ?? { global: {}, pages: {} }
  const dataServiceOptions = options.dataServiceOptions ?? {}

  const varsStore = new VarsStore(varsDefinitions)
  const expressionEngine = new ExpressionEngine()
  const mockProvider = new MockDataProvider()
  const diagnosticsStore = new DiagnosticsStore(dataServiceOptions)
  const dataService = new DataService(dataServiceOptions)

  const bindingResolver = new BindingResolver(
    {
      varsStore,
      expressionEngine,
      mockProvider,
      diagnosticsStore,
      dataService,
    },
    { mode, pageId },
  )

  if (pageId) {
    varsStore.setCurrentPage(pageId)
  }

  return {
    varsStore,
    expressionEngine,
    mockProvider,
    diagnosticsStore,
    dataService,
    bindingResolver,

    destroy() {
      bindingResolver.destroy()
      dataService.destroy()
      diagnosticsStore.destroy()
      mockProvider.destroy()
      varsStore.removeAllListeners()
    },
  }
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
}
