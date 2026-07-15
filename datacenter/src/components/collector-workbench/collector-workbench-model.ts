import type { CollectorAgent } from '@/api/schemas/collector-dev.schema'
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
