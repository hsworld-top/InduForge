import { describe, expect, it } from 'vitest'
import {
  agentSupportsOperation,
  buildCollectorDriverTree,
  collectorAgentStorageKey,
  formatCollectorAgentName,
  formatCollectorConnectionStatus,
  formatCollectorConnectionSummary,
  formatCollectorProtocolFamily,
  groupCollectorConnections,
  resolveCollectorEnumOptionLabel,
  resolveCollectorDriverIconKey,
} from '@/components/collector-workbench/collector-workbench-model'

describe('industrial collector workbench model', () => {
  const agent = {
    id: 'agent-1',
    name: 'dev',
    os: 'windows',
    arch: 'x64',
    version: '1',
    ipAddress: '127.0.0.1',
    status: 'online' as const,
    capabilities: [
      {
        driverId: 'opcua.standard',
        driverVersion: '1.0.0',
        schemaVersions: [2],
        operations: ['connection.test', 'device.browse', 'point.read'],
      },
    ],
    createdAt: '2026-07-15 10:00:00',
  }
  it('allows offline editing while device operations require a compatible agent', () => {
    expect(agentSupportsOperation(undefined, 'opcua.standard', '1.0.0', 2, 'device.browse')).toBe(
      false,
    )
    expect(agentSupportsOperation(agent, 'opcua.standard', '1.0.0', 2, 'device.browse')).toBe(true)
    expect(
      agentSupportsOperation(
        { ...agent, status: 'offline' },
        'opcua.standard',
        '1.0.0',
        2,
        'device.browse',
      ),
    ).toBe(false)
  })
  it('rejects mismatched driver versions and operations', () => {
    expect(agentSupportsOperation(agent, 'opcua.standard', '2.0.0', 2, 'point.read')).toBe(false)
    expect(agentSupportsOperation(agent, 'opcua.standard', '1.0.0', 2, 'point.write')).toBe(false)
  })
  it('stores agent selection per project', () => {
    expect(collectorAgentStorageKey('project-1')).toBe('induforge:collector-agent:project-1')
  })
  it('uses readable agent labels instead of opaque ids', () => {
    expect(formatCollectorAgentName({ ...agent, name: agent.id })).toBe(
      'Windows 调试代理 · 127.0.0.1',
    )
    expect(formatCollectorAgentName({ ...agent, name: '车间开发机' })).toBe('车间开发机')
  })
  it('maps HSL driver icons and keeps unsupported protocols on fallback', () => {
    expect(resolveCollectorDriverIconKey('modbus', 'modbus.tcp')).toBe('modbus')
    expect(resolveCollectorDriverIconKey('plc', 'siemens.s7')).toBe('siemens')
    expect(resolveCollectorDriverIconKey('opcua', 'opcua.standard')).toBeNull()
  })
  it('shows bilingual enum labels and falls back to raw values', () => {
    const property = {
      type: 'string' as const,
      enum: ['none', 'odd'],
      'x-induforge-enum-labels': { none: '无校验（none）' },
    }
    expect(resolveCollectorEnumOptionLabel(property, 'none')).toBe('无校验（none）')
    expect(resolveCollectorEnumOptionLabel(property, 'odd')).toBe('odd')
  })
  it('builds a protocol family and driver tree with bilingual labels', () => {
    const drivers = [
      {
        protocolFamily: 'siemens',
        driverId: 'siemens.s7-tcp',
        driverVersion: '1.0.0',
        schemaVersion: 1,
        displayName: 'Siemens S7 TCP',
        category: 'plc',
        transports: ['tcp'],
        operations: [],
        dataTypes: ['bool'],
        acquisitionModes: ['polling'],
        platforms: { devAgent: [], runtime: [] },
      },
      {
        protocolFamily: 'modbus',
        driverId: 'modbus.rtu',
        driverVersion: '1.0.0',
        schemaVersion: 1,
        displayName: 'Modbus RTU',
        category: 'fieldbus',
        transports: ['serial'],
        operations: [],
        dataTypes: ['bool'],
        acquisitionModes: ['polling'],
        platforms: { devAgent: [], runtime: [] },
      },
    ]

    const tree = buildCollectorDriverTree(drivers)
    const siemens = tree.find((node) => node.protocolFamily === 'siemens')
    expect(siemens?.label).toBe('Siemens Plc [西门子]')
    expect(siemens?.children?.[0]?.label).toBe('Siemens S7 TCP [西门子 S7 以太网]')
    expect(buildCollectorDriverTree(drivers, 'PLC')[0]?.protocolFamily).toBe('siemens')
    expect(buildCollectorDriverTree(drivers, '串口')[0]?.protocolFamily).toBe('modbus')
  })
  it('groups configured connections by protocol family', () => {
    const connection = {
      id: 'connection-1',
      projectId: 'project-1',
      name: '锅炉 PLC',
      status: 'configured',
      displayOrder: 0,
      protocolFamily: 'modbus',
      driverId: 'modbus.tcp',
      driverVersion: '1.0.0',
      schemaVersion: 1,
      config: {},
      metadata: {},
      secretStatus: {},
      createdAt: '2026-07-15 10:00:00',
      updatedAt: '2026-07-15 10:00:00',
    }
    const groups = groupCollectorConnections([
      connection,
      { ...connection, id: 'connection-2', name: '温控仪', driverId: 'modbus.rtu' },
      { ...connection, id: 'connection-3', protocolFamily: 'opcua', driverId: 'opcua.standard' },
    ])

    expect(groups.map((group) => [group.label, group.items.length])).toEqual([
      ['Modbus', 2],
      ['OPC UA [开放平台通信]', 1],
    ])
    expect(formatCollectorProtocolFamily('custom')).toBe('CUSTOM')
  })
  it('formats connection summaries and localized status labels', () => {
    const connection = {
      id: 'connection-1',
      projectId: 'project-1',
      name: 'OPC UA',
      status: 'unknown',
      displayOrder: 0,
      protocolFamily: 'opcua',
      driverId: 'opcua.standard',
      driverVersion: '1.0.0',
      schemaVersion: 2,
      config: { host: 'fe80::1', port: 4840, endpointPath: '/induforge/sim' },
      metadata: {},
      secretStatus: {},
      createdAt: '2026-07-16 10:00:00',
      updatedAt: '2026-07-16 10:00:00',
    }

    expect(formatCollectorConnectionSummary(connection)).toBe('[fe80::1]:4840/induforge/sim')
    expect(formatCollectorConnectionStatus(connection.status)).toEqual({
      label: '未检测',
      tone: 'warning',
    })
    expect(formatCollectorConnectionStatus('connected').label).toBe('在线')
    expect(formatCollectorConnectionStatus('error').label).toBe('异常')
    expect(formatCollectorConnectionStatus('configured').label).toBe('已配置')
  })
})
