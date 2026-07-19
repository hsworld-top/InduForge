import type { CollectorAgent } from '@/api/schemas/collector-dev.schema'
import type {
  CollectorConnection,
  CollectorDriverSummary,
  CollectorJsonSchemaProperty,
} from '@/api/schemas/collector.schema'

export type CollectorDriverIconKey =
  | 'modbus'
  | 'siemens'
  | 'melsec'
  | 'omron'
  | 'beckhoff'
  | 'panasonic'
  | 'keyence'
  | 'fatek'
  | 'inovance'
  | 'schneider'

export type CollectorConnectionGroup = {
  key: string
  label: string
  items: CollectorConnection[]
}

export const collectorDriverFeatures = {
  pointElementCount: 'point.elementCount',
} as const

export function collectorDriverSupportsFeature(
  driver: Pick<CollectorDriverSummary, 'features'> | null | undefined,
  feature: string,
) {
  return driver?.features.includes(feature) ?? false
}

export type CollectorDriverTreeNode = {
  id: string
  type: 'family' | 'driver'
  label: string
  protocolFamily?: string
  driver?: CollectorDriverSummary
  children?: CollectorDriverTreeNode[]
}

export function resolveCollectorEnumOptionLabel(
  property: CollectorJsonSchemaProperty,
  option: unknown,
) {
  return property['x-induforge-enum-labels']?.[String(option)] || String(option)
}

const protocolFamilyLabels: Record<string, string> = {
  modbus: 'Modbus',
  opc: 'OPC [开放平台通信]',
  opcua: 'OPC UA [开放平台通信]',
  s7: 'Siemens Plc [西门子]',
  siemens: 'Siemens Plc [西门子]',
  mitsubishi: 'Melsec Plc [三菱]',
  melsec: 'Melsec Plc [三菱]',
  omron: 'Omron Plc [欧姆龙]',
  beckhoff: 'Beckhoff Plc [倍福]',
  panasonic: 'Panasonic Plc [松下]',
  keyence: 'Keyence Plc [基恩士]',
  fatek: 'Fatek Plc [永宏]',
  inovance: 'Inovance Plc [汇川]',
  schneider: 'Schneider Plc [施耐德]',
}

const categoryLabels: Record<string, string> = {
  plc: 'PLC [可编程控制器]',
  fieldbus: 'Fieldbus [现场总线]',
  opc: 'OPC [工业互操作]',
  instrument: 'Instrument [仪器仪表]',
  robot: 'Robot [机器人]',
  cnc: 'CNC [数控机床]',
  sensor: 'Sensor [传感器]',
  custom: 'Custom Protocol [自定义协议]',
  industrial: 'Industrial [工业协议]',
}

const driverChineseLabels: Record<string, string> = {
  'opcua.standard': '标准客户端',
  'modbus.tcp': '以太网',
  'modbus.rtu': '串口',
  'siemens.s7-tcp': '西门子 S7 以太网',
}

export function resolveCollectorDriverIconKey(
  protocolFamily: string,
  driverId = '',
): CollectorDriverIconKey | null {
  const identity = `${protocolFamily} ${driverId}`.toLowerCase()
  if (identity.includes('modbus')) return 'modbus'
  if (identity.includes('siemens') || identity.includes('s7')) return 'siemens'
  if (identity.includes('melsec') || identity.includes('mitsubishi')) return 'melsec'
  if (identity.includes('omron')) return 'omron'
  if (identity.includes('beckhoff')) return 'beckhoff'
  if (identity.includes('panasonic')) return 'panasonic'
  if (identity.includes('keyence')) return 'keyence'
  if (identity.includes('fatek')) return 'fatek'
  if (identity.includes('inovance')) return 'inovance'
  if (identity.includes('schneider')) return 'schneider'
  return null
}

export function formatCollectorAgentName(agent: CollectorAgent) {
  const name = agent.name.trim()
  const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
  if (name && name !== agent.id && !uuidPattern.test(name)) return name
  const platform = agent.os.toLowerCase() === 'windows' ? 'Windows' : agent.os || '本机'
  return agent.ipAddress ? `${platform} 调试代理 · ${agent.ipAddress}` : `${platform} 调试代理`
}

export function formatCollectorProtocolFamily(protocolFamily: string) {
  const normalized = protocolFamily.trim().toLowerCase()
  return protocolFamilyLabels[normalized] || protocolFamily.trim().toUpperCase()
}

export function formatCollectorConnectionSummary(connection: CollectorConnection) {
  const config = connection.config
  const host = typeof config.host === 'string' ? config.host.trim() : ''
  const port = typeof config.port === 'number' || typeof config.port === 'string' ? config.port : ''
  const endpointPath = typeof config.endpointPath === 'string' ? config.endpointPath.trim() : ''
  const portName = typeof config.portName === 'string' ? config.portName.trim() : ''

  if (host) {
    const hostText = host.includes(':') && !host.startsWith('[') ? `[${host}]` : host
    const pathText = endpointPath && endpointPath !== '/' ? endpointPath : ''
    return `${hostText}${port ? `:${port}` : ''}${pathText}`
  }
  if (portName) return portName
  return connection.driverId
}

export function formatCollectorConnectionStatus(status: string) {
  const normalized = status.trim().toLowerCase()
  if (['online', 'connected', 'running', 'healthy'].includes(normalized))
    return { label: '在线', tone: 'success' as const }
  if (['failed', 'error', 'invalid', 'unhealthy'].includes(normalized))
    return { label: '异常', tone: 'danger' as const }
  if (['offline', 'disconnected'].includes(normalized))
    return { label: '离线', tone: 'muted' as const }
  if (['configured', 'ready'].includes(normalized))
    return { label: '已配置', tone: 'primary' as const }
  return { label: '未检测', tone: 'warning' as const }
}

export function formatCollectorCategoryLabel(category: string) {
  const normalized = category.trim().toLowerCase()
  return categoryLabels[normalized] || category.trim().toUpperCase()
}

export function formatCollectorDriverDisplayName(driver: CollectorDriverSummary) {
  const chineseLabel = driverChineseLabels[driver.driverId]
  return chineseLabel ? `${driver.displayName} [${chineseLabel}]` : driver.displayName
}

export function buildCollectorDriverTree(
  drivers: CollectorDriverSummary[],
  search = '',
): CollectorDriverTreeNode[] {
  const keyword = search.trim().toLowerCase()
  const families = new Map<string, CollectorDriverSummary[]>()

  for (const driver of drivers) {
    const family = driver.protocolFamily.trim().toLowerCase() || 'unknown'
    const familyLabel = formatCollectorProtocolFamily(family)
    const driverLabel = formatCollectorDriverDisplayName(driver)
    // 分类仅作为搜索元数据，避免在用户目录中增加存在歧义的额外层级。
    const category = driver.category.trim().toLowerCase() || 'industrial'
    const categoryLabel = formatCollectorCategoryLabel(category)
    const matches = [category, categoryLabel, family, familyLabel, driver.driverId, driverLabel]
      .join(' ')
      .toLowerCase()
      .includes(keyword)
    if (keyword && !matches) continue

    const familyDrivers = families.get(family) || []
    familyDrivers.push(driver)
    families.set(family, familyDrivers)
  }

  return [...families.entries()]
    .sort(([left], [right]) =>
      formatCollectorProtocolFamily(left).localeCompare(formatCollectorProtocolFamily(right)),
    )
    .map(([family, familyDrivers]) => ({
      id: `family:${family}`,
      type: 'family' as const,
      label: formatCollectorProtocolFamily(family),
      protocolFamily: family,
      children: familyDrivers
        .sort((left, right) => left.displayName.localeCompare(right.displayName))
        .map((driver) => ({
          id: driver.driverId,
          type: 'driver' as const,
          label: formatCollectorDriverDisplayName(driver),
          protocolFamily: family,
          driver,
        })),
    }))
}

export function groupCollectorConnections(
  connections: CollectorConnection[],
): CollectorConnectionGroup[] {
  const groups = new Map<string, CollectorConnectionGroup>()
  for (const connection of connections) {
    const key = connection.protocolFamily.trim().toLowerCase() || 'unknown'
    const group = groups.get(key) || {
      key,
      label: formatCollectorProtocolFamily(connection.protocolFamily || 'unknown'),
      items: [],
    }
    group.items.push(connection)
    groups.set(key, group)
  }
  return [...groups.values()]
}

export function agentSupportsOperation(
  agent: CollectorAgent | undefined,
  driverId: string,
  driverVersion: string,
  schemaVersion: number,
  operation: string,
) {
  return Boolean(
    agent?.status === 'online' &&
    agent.capabilities.some(
      (capability) =>
        capability.driverId === driverId &&
        capability.driverVersion === driverVersion &&
        capability.schemaVersions.includes(schemaVersion) &&
        capability.operations.includes(operation),
    ),
  )
}

export function collectorAgentStorageKey(projectId: string) {
  return `induforge:collector-agent:${projectId}`
}

export type CollectorBrowseNode = {
  nodeId: string
  browseName: string
  displayName: string
  nodeClass: string
  dataType?: string | null
  hasChildren: boolean
  modeled?: boolean
}

export type CollectorBrowseBranch = {
  parentNodeId: string
  nodes: CollectorBrowseNode[]
}

// 父节点勾选必须覆盖尚未展开的后代，因此按受控并发递归读取整棵子树并过滤已建模变量。
export async function collectCollectorBrowseVariables(
  root: CollectorBrowseNode,
  loadChildren: (nodeId: string) => Promise<CollectorBrowseNode[]>,
  concurrency = 3,
) {
  const variables = new Map<string, CollectorBrowseNode>()
  const branches: CollectorBrowseBranch[] = []
  const visited = new Set<string>()
  const queue = [root]
  while (queue.length > 0) {
    const batch = queue.splice(0, Math.max(1, concurrency)).filter((node) => {
      if (visited.has(node.nodeId)) return false
      visited.add(node.nodeId)
      return node.hasChildren
    })
    if (batch.length === 0) continue
    const loaded = await Promise.all(
      batch.map(async (parent) => ({
        parentNodeId: parent.nodeId,
        nodes: await loadChildren(parent.nodeId),
      })),
    )
    for (const branch of loaded) {
      branches.push(branch)
      for (const child of branch.nodes) {
        if (child.nodeClass === 'variable' && !child.modeled) variables.set(child.nodeId, child)
        if (child.hasChildren) queue.push(child)
      }
    }
  }
  return { variables: [...variables.values()], branches }
}

export type CollectorPointCreateDefaults = {
  name: string
  description?: string
  dataType: string
  elementCount: number
  enabled: boolean
  address: Record<string, unknown>
}

const opcuaDataTypeMap: Record<string, string> = {
  boolean: 'bool',
  sbyte: 'int8',
  byte: 'uint8',
  int16: 'int16',
  uint16: 'uint16',
  int32: 'int32',
  uint32: 'uint32',
  int64: 'int64',
  uint64: 'uint64',
  float: 'float32',
  double: 'float64',
  string: 'string',
  bytestring: 'bytes',
  datetime: 'datetime',
}

export function filterCollectorBrowseNode(nodeIdFilter: string, node: CollectorBrowseNode) {
  const keyword = nodeIdFilter.trim().toLowerCase()
  return !keyword || node.nodeId.toLowerCase().includes(keyword)
}

export function buildCollectorPointCreateDefaults(
  node: CollectorBrowseNode,
): CollectorPointCreateDefaults {
  const name = node.displayName.trim() || node.browseName.trim() || '未命名变量'
  const rawDataType = node.dataType?.trim() || ''
  const dataType = opcuaDataTypeMap[rawDataType.toLowerCase()] || rawDataType || 'float64'

  return {
    name,
    dataType,
    elementCount: 1,
    enabled: true,
    address: { nodeId: node.nodeId },
  }
}

export type CollectorDebugConnectionStatus =
  | 'disconnected'
  | 'connecting'
  | 'connected'
  | 'disconnecting'
  | 'error'

export type CollectorDebugConnectionState = {
  status: CollectorDebugConnectionStatus
  connectedAt?: string
  serverName?: string
  message?: string
}

export function collectorDebugConnectionTone(status: CollectorDebugConnectionStatus) {
  if (status === 'connected') return 'success' as const
  if (status === 'connecting' || status === 'disconnecting') return 'primary' as const
  if (status === 'error') return 'danger' as const
  return 'muted' as const
}
