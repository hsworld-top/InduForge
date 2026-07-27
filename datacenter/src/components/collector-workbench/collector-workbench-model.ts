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
  'allen-bradley': 'Allen-Bradley Plc [罗克韦尔]',
  modbus: 'Modbus',
  opc: 'OPC [开放平台通信]',
  opcua: 'OPC UA [开放平台通信]',
  s7: 'Siemens Plc [西门子]',
  siemens: 'Siemens Plc [西门子]',
  mitsubishi: 'Melsec Plc [三菱]',
  melsec: 'Melsec Plc [三菱]',
  omron: 'Omron Plc [欧姆龙]',
  beckhoff: 'Beckhoff Plc [倍福]',
  iec: 'IEC [国际电工委员会]',
  panasonic: 'Panasonic Plc [松下]',
  lsis: 'LSIS Plc [LS 产电]',
  ge: 'GE Plc [通用电气]',
  keyence: 'Keyence Plc [基恩士]',
  fatek: 'Fatek Plc [永宏]',
  fuji: 'Fuji Plc [富士]',
  inovance: 'Inovance Plc [汇川]',
  megmeet: 'MegMeet Plc [麦格米特]',
  vigor: 'Vigor Plc [丰炜]',
  yokogawa: 'Yokogawa Plc [横河]',
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
  'allen-bradley.ethernet-ip': '罗克韦尔 EtherNet/IP',
  'allen-bradley.connected-cip': '罗克韦尔 Connected CIP',
  'allen-bradley.micro-cip': '罗克韦尔 MicroCIP',
  'allen-bradley.pccc': '罗克韦尔 PCCC',
  'allen-bradley.slc': '罗克韦尔 SLC',
  'allen-bradley.df1-serial': '罗克韦尔 DF1 串口',
  'beckhoff.ads-tcp': '倍福 ADS 以太网',
  'cimon.hmi-protocol': 'Cimon HMI 协议',
  'cjt188.serial': 'CJT188 串口',
  'cjt188.tcp': 'CJT188 串口透传',
  'dam3601.serial': 'DAM3601 温度采集模块',
  'ec-fan.machine-serial': 'EC 风机串口',
  'dcs.nanjing-auto': '南京自动化 DCS',
  'delixi.dtsu6606': '德力西 DTSU6606 电表',
  'dlt645.2007-serial': 'DL/T 645-2007 串口',
  'dlt645.2007-over-tcp': 'DL/T 645-2007 串口透传',
  'dlt645.1997-serial': 'DL/T 645-1997 串口',
  'dlt645.1997-over-tcp': 'DL/T 645-1997 串口透传',
  'dlt698.serial': 'DL/T 698 串口',
  'dlt698.over-tcp': 'DL/T 698 串口透传',
  'dlt698.tcp-net': 'DL/T 698 TCP',
  'delta.tcp': '台达原生以太网',
  'delta.rtu-over-tcp': '台达 RTU 串口透传',
  'delta.ascii-over-tcp': '台达 ASCII 串口透传',
  'delta.rtu': '台达 RTU 串口',
  'delta.ascii': '台达 ASCII 串口',
  'iec.60870-5-104': 'IEC 104 远动通信',
  'panasonic.mewtocol-tcp': '松下 Mewtocol 以太网',
  'panasonic.mewtocol-serial': '松下 Mewtocol 串口',
  'panasonic.mc-binary-tcp': '松下 MC Binary 以太网',
  'lsis.fast-enet': 'LSIS Fast Enet',
  'lsis.cnet': 'LSIS Cnet 串口',
  'lsis.cnet-over-tcp': 'LSIS Cnet 串口透传',
  'lsis.cpu-serial': 'LSIS CPU 串口',
  'ge.srtp-tcp': 'GE SRTP 以太网',
  'inovance.modbus-tcp': '汇川 Modbus TCP',
  'inovance.modbus-serial': '汇川 Modbus 串口',
  'inovance.modbus-rtu-over-tcp': '汇川 Modbus RTU 串口透传',
  'inovance.connected-cip': '汇川 Connected CIP',
  'inovance.easy-net': '汇川 EasyNet 专用以太网',
  'inovance.computer-link': '汇川 ComputerLink 串口',
  'fatek.program-tcp': '永宏编程口 TCP',
  'fatek.program-serial': '永宏编程口串口',
  'freedom.tcp': '自由协议 TCP',
  'freedom.udp': '自由协议 UDP',
  'freedom.serial': '自由协议串口',
  'fuji.command-setting-tcp': '富士 Command Setting 以太网',
  'fuji.sph-tcp': '富士 SPH 以太网',
  'fuji.spb-over-tcp': '富士 SPB 串口透传',
  'fuji.spb': '富士 SPB 串口',
  'keyence.mc-3e-tcp': '基恩士 MC 3E 以太网',
  'keyence.mc-ascii-tcp': '基恩士 MC ASCII 以太网',
  'keyence.kv-old-tcp': '基恩士旧型 KV 以太网',
  'keyence.nano-tcp': '基恩士 KV 上位链路以太网',
  'keyence.nano-serial': '基恩士 KV 上位链路串口',
  'keyence.nano-serial-over-tcp': '基恩士 KV 上位链路串口透传',
  'megmeet.tcp': '麦格米特标准以太网',
  'megmeet.rtu-over-tcp': '麦格米特 RTU 串口透传',
  'megmeet.rtu': '麦格米特 RTU 串口',
  'opcua.standard': '标准客户端',
  'modbus.tcp': '以太网',
  'modbus.rtu': 'RTU 串口',
  'modbus.rtu-over-tcp': 'RTU 串口透传',
  'modbus.ascii': 'ASCII 串口',
  'modbus.ascii-over-tcp': 'ASCII 串口透传',
  'modbus.udp': 'UDP',
  'mqtt.rpc-device': 'MQTT RPC 远程设备',
  'siemens.s7-tcp': '西门子 S7 以太网',
  'siemens.s7-plus': '西门子 S7 Plus',
  'siemens.ppi': '西门子 PPI 串口',
  'siemens.ppi-over-tcp': '西门子 PPI 串口透传',
  'siemens.mpi': '西门子 MPI 串口（实验）',
  'siemens.fetch-write': '西门子 Fetch/Write',
  'siemens.web-api': '西门子 Web API',
  'xinje.tcp': '信捷标准以太网',
  'xinje.rtu-over-tcp': '信捷 RTU 串口透传',
  'xinje.rtu': '信捷 XC/XD/XL 串口',
  'xinje.internal-tcp': '信捷内部以太网',
  'vigor.serial-over-tcp': '丰炜 VS 串口透传',
  'vigor.serial': '丰炜 VS 串口',
  'yamatake.digitron-serial': '山武 Digitron 串口',
  'yamatake.digitron-tcp': '山武 Digitron 以太网',
  'yaskawa.memobus-tcp': '安川 Memobus TCP',
  'yaskawa.memobus-udp': '安川 Memobus UDP',
  'yudian.ai-bus': '宇电 AIBus',
  'yokogawa.link-tcp': '横河 FA-M3 Link 以太网',
  'oriental-motor.eip': '东方马达 EtherNet/IP',
  'rkc.temperature-controller-serial': 'RKC 温控器串口',
  'rkc.temperature-controller-tcp': 'RKC 温控器串口透传',
  'robot.estun-tcp': '埃斯顿机器人 TCP',
  'robot.fanuc-interface': 'FANUC 机器人接口',
  'toyo.puc': '东洋 PUC',
  'turck.reader-tcp': 'Turck Reader 以太网',
  'mitsubishi.mc-3e-tcp': '三菱 MC 3E 以太网',
  'mitsubishi.a1e-ascii-tcp': '三菱 A1E ASCII 以太网',
  'mitsubishi.a1e-binary-tcp': '三菱 A1E Binary 以太网',
  'mitsubishi.mc-ascii-tcp': '三菱 MC ASCII 以太网',
  'mitsubishi.mc-ascii-udp': '三菱 MC ASCII UDP',
  'mitsubishi.mc-binary-udp': '三菱 MC Binary UDP',
  'mitsubishi.mc-r-binary-tcp': '三菱 MC R Binary 以太网',
  'mitsubishi.cip': '三菱 CIP 以太网',
  'mitsubishi.a3c-serial': '三菱 A3C 串口',
  'mitsubishi.a3c-serial-over-tcp': '三菱 A3C 串口透传',
  'mitsubishi.fx-links-serial': '三菱 FX Links 串口',
  'mitsubishi.fx-links-over-tcp': '三菱 FX Links 串口透传',
  'mitsubishi.fx-serial': '三菱 FX 编程口串口',
  'mitsubishi.fx-serial-over-tcp': '三菱 FX 编程口透传',
  'omron.fins-tcp': '欧姆龙 FINS 以太网',
  'omron.fins-udp': '欧姆龙 FINS UDP',
  'omron.cip': '欧姆龙 CIP 以太网',
  'omron.connected-cip': '欧姆龙 Connected CIP',
  'omron.hostlink': '欧姆龙 HostLink 串口',
  'omron.hostlink-over-tcp': '欧姆龙 HostLink 串口透传',
  'omron.hostlink-cmode': '欧姆龙 HostLink C-Mode 串口',
  'omron.hostlink-cmode-over-tcp': '欧姆龙 HostLink C-Mode 串口透传',
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

export function validateCollectorConnectionName(value: string) {
  const name = value.trim().replace(/\s+/g, ' ')
  if (!name) return { name, error: '请填写连接名称' }
  if ([...name].length > 50) return { name, error: '连接名称最多 50 个字符' }
  if (![...name].every((char) => char === ' ' || /[\p{L}\p{N}]/u.test(char))) {
    return { name, error: '连接名称只能包含文字、数字和空格' }
  }
  return { name, error: null }
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
