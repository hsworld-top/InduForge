import type { CollectorAgent } from '@/api/schemas/collector-dev.schema'

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
import type { CollectorConnection } from '@/api/schemas/collector.schema'

export type CollectorConnectionGroup = {
  key: string
  label: string
  items: CollectorConnection[]
}

const protocolFamilyLabels: Record<string, string> = {
  modbus: 'Modbus',
  opc: 'OPC',
  opcua: 'OPC UA',
  s7: '西门子 PLC',
  mitsubishi: '三菱 PLC',
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
