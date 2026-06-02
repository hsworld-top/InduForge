export type BuiltinStoreType =
  | 'builtin.relation'
  | 'builtin.timeseries'
  | 'builtin.realtime'
  | 'builtin.message'

export const BUILTIN_STORE_TYPES: Array<{
  type: BuiltinStoreType
  name: string
  description: string
  defaultConfig: Record<string, unknown>
}> = [
  {
    type: 'builtin.relation',
    name: 'IF关系库',
    description: '业务表、表单数据和普通 SQL',
    defaultConfig: {},
  },
  {
    type: 'builtin.timeseries',
    name: 'IF时序库',
    description: '数据点历史、趋势和聚合',
    defaultConfig: {},
  },
  {
    type: 'builtin.realtime',
    name: 'IF实时库',
    description: '当前值、短期状态和缓存',
    defaultConfig: { defaultTtlSeconds: 300, group: 'rt' },
  },
  {
    type: 'builtin.message',
    name: 'IF消息库',
    description: '设备消息、模拟数据和订阅',
    defaultConfig: { topic: 'mock-data', samplePayload: { value: 1 } },
  },
]

export const isBuiltinStoreType = (type?: string): type is BuiltinStoreType =>
  BUILTIN_STORE_TYPES.some((item) => item.type === type)

export const getBuiltinStoreInfo = (type?: string) =>
  BUILTIN_STORE_TYPES.find((item) => item.type === type)
