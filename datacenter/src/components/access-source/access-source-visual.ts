import IconTablerCpu from '~icons/tabler/cpu'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerMessages from '~icons/tabler/messages'
import IconTablerPlugConnected from '~icons/tabler/plug-connected'
import IconTablerWorldWww from '~icons/tabler/world-www'

export type AccessSourceVisualCategory = 'builtin' | 'database' | 'stream' | 'industrial' | 'other'

const builtinTypes = new Set([
  'builtin.relation',
  'builtin.timeseries',
  'builtin.realtime',
  'builtin.message',
])
const databaseTypes = new Set([
  'relational',
  'mysql',
  'postgresql',
  'sqlserver',
  'tdengine',
  'redis',
])
const messageStreamTypes = new Set(['mqtt', 'kafka'])
const industrialTypes = new Set(['collector', 'opcua', 'opcda', 's7', 'modbus'])

export const resolveAccessSourceVisual = (type?: string) => {
  const normalizedType = type || ''
  if (builtinTypes.has(normalizedType)) {
    return { category: 'builtin' as const, icon: IconTablerDatabase }
  }
  if (databaseTypes.has(normalizedType)) {
    return { category: 'database' as const, icon: IconTablerDatabase }
  }
  if (messageStreamTypes.has(normalizedType)) {
    return { category: 'stream' as const, icon: IconTablerMessages }
  }
  if (normalizedType === 'http') {
    return { category: 'stream' as const, icon: IconTablerWorldWww }
  }
  if (normalizedType === 'websocket') {
    return { category: 'stream' as const, icon: IconTablerPlugConnected }
  }
  if (
    industrialTypes.has(normalizedType) ||
    normalizedType.includes('opc') ||
    normalizedType.includes('modbus')
  ) {
    return { category: 'industrial' as const, icon: IconTablerCpu }
  }
  return { category: 'other' as const, icon: IconTablerPlugConnected }
}
