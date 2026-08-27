import { describe, expect, it } from 'vitest'
import type { CollectorDriverSummary } from '@/api/schemas/collector.schema'
import {
  agentSupportsOperation,
  buildCollectorPointCreateDefaults,
  buildCollectorDriverTree,
  collectCollectorBrowseVariables,
  collectorAgentStorageKey,
  collectorDebugConnectionTone,
  collectorDriverFeatures,
  collectorDriverSupportsFeature,
  formatCollectorAgentName,
  formatCollectorConnectionStatus,
  formatCollectorConnectionSummary,
  filterCollectorBrowseNode,
  formatCollectorProtocolFamily,
  groupCollectorConnections,
  resolveCollectorEnumOptionLabel,
  resolveCollectorDriverIconKey,
  validateCollectorConnectionName,
} from '@/components/collector-workbench/collector-workbench-model'

function createDriverSummary(
  protocolFamily: string,
  driverId: string,
  displayName: string,
): CollectorDriverSummary {
  return {
    protocolFamily,
    driverId,
    driverVersion: '1.0.0',
    schemaVersion: 1,
    displayName,
    category: 'plc',
    transports: ['tcp'],
    operations: [],
    features: [],
    dataTypes: ['bool'],
    acquisitionModes: ['polling'],
    platforms: { devAgent: [], runtime: [] },
  }
}

describe('industrial collector workbench model', () => {
  it('normalizes valid connection names and rejects special characters', () => {
    expect(validateCollectorConnectionName('  1号产线   Modbus  ')).toEqual({
      name: '1号产线 Modbus',
      error: null,
    })
    expect(validateCollectorConnectionName('line_1')).toEqual({
      name: 'line_1',
      error: '连接名称只能包含文字、数字和空格',
    })
  })

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
        schemaVersions: [1],
        operations: ['connection.test', 'device.browse', 'point.read'],
      },
    ],
    createdAt: '2026-07-15 10:00:00',
  }
  it('allows offline editing while device operations require a compatible agent', () => {
    expect(agentSupportsOperation(undefined, 'opcua.standard', '1.0.0', 1, 'device.browse')).toBe(
      false,
    )
    expect(agentSupportsOperation(agent, 'opcua.standard', '1.0.0', 1, 'device.browse')).toBe(true)
    expect(
      agentSupportsOperation(
        { ...agent, status: 'offline' },
        'opcua.standard',
        '1.0.0',
        1,
        'device.browse',
      ),
    ).toBe(false)
  })
  it('rejects mismatched driver versions and operations', () => {
    expect(agentSupportsOperation(agent, 'opcua.standard', '2.0.0', 1, 'point.read')).toBe(false)
    expect(agentSupportsOperation(agent, 'opcua.standard', '1.0.0', 1, 'point.write')).toBe(false)
  })
  it('recursively collects only variables that have not been modeled', async () => {
    const branches = new Map([
      [
        'root',
        [
          {
            nodeId: 'folder',
            browseName: 'folder',
            displayName: 'folder',
            nodeClass: 'object',
            hasChildren: true,
          },
          {
            nodeId: 'existing',
            browseName: 'existing',
            displayName: 'existing',
            nodeClass: 'variable',
            hasChildren: false,
            modeled: true,
          },
        ],
      ],
      [
        'folder',
        [
          {
            nodeId: 'method',
            browseName: 'method',
            displayName: 'method',
            nodeClass: 'method',
            hasChildren: true,
          },
          {
            nodeId: 'temperature',
            browseName: 'temperature',
            displayName: 'temperature',
            nodeClass: 'variable',
            hasChildren: false,
          },
        ],
      ],
      [
        'method',
        [
          {
            nodeId: 'pressure',
            browseName: 'pressure',
            displayName: 'pressure',
            nodeClass: 'variable',
            hasChildren: false,
          },
          {
            nodeId: 'root',
            browseName: 'root',
            displayName: 'root',
            nodeClass: 'object',
            hasChildren: true,
          },
        ],
      ],
    ])
    const root = {
      nodeId: 'root',
      browseName: 'root',
      displayName: 'root',
      nodeClass: 'object',
      hasChildren: true,
    }

    const result = await collectCollectorBrowseVariables(
      root,
      async (nodeId) => branches.get(nodeId) || [],
      2,
    )

    expect(result.variables.map((node) => node.nodeId)).toEqual(['temperature', 'pressure'])
    expect(result.branches.map((branch) => branch.parentNodeId)).toEqual([
      'root',
      'folder',
      'method',
    ])
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
        features: ['point.elementCount'],
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
        features: ['point.elementCount'],
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
  it('places common protocol families and their primary drivers first', () => {
    const drivers = [
      createDriverSummary('beckhoff', 'beckhoff.ads-tcp', 'Beckhoff ADS TCP'),
      createDriverSummary('allen-bradley', 'allen-bradley.pccc', 'Allen-Bradley PCCC'),
      createDriverSummary(
        'allen-bradley',
        'allen-bradley.ethernet-ip',
        'Allen-Bradley EtherNet/IP',
      ),
      createDriverSummary('omron', 'omron.hostlink', 'Omron HostLink'),
      createDriverSummary('omron', 'omron.fins-udp', 'Omron FINS UDP'),
      createDriverSummary('omron', 'omron.fins-tcp', 'Omron FINS TCP'),
      createDriverSummary('mitsubishi', 'mitsubishi.cip', 'Mitsubishi CIP'),
      createDriverSummary('mitsubishi', 'mitsubishi.mc-3e-tcp', 'Mitsubishi MC 3E TCP'),
      createDriverSummary('siemens', 'siemens.web-api', 'Siemens Web API'),
      createDriverSummary('siemens', 'siemens.s7-tcp', 'Siemens S7 TCP'),
      createDriverSummary('opcua', 'opcua.standard', 'OPC UA'),
      createDriverSummary('modbus', 'modbus.udp', 'Modbus UDP'),
      createDriverSummary('modbus', 'modbus.rtu', 'Modbus RTU'),
      createDriverSummary('modbus', 'modbus.tcp', 'Modbus TCP'),
      createDriverSummary('yokogawa', 'yokogawa.link-tcp', 'Yokogawa Link TCP'),
    ]

    const tree = buildCollectorDriverTree(drivers)

    expect(tree.map((node) => node.protocolFamily)).toEqual([
      'modbus',
      'opcua',
      'siemens',
      'mitsubishi',
      'omron',
      'allen-bradley',
      'beckhoff',
      'yokogawa',
    ])
    expect(
      tree.find((node) => node.protocolFamily === 'modbus')?.children?.map((node) => node.id),
    ).toEqual(['modbus.tcp', 'modbus.rtu', 'modbus.udp'])
    expect(
      tree.find((node) => node.protocolFamily === 'omron')?.children?.map((node) => node.id),
    ).toEqual(['omron.fins-tcp', 'omron.fins-udp', 'omron.hostlink'])
    expect(buildCollectorDriverTree(drivers, 'PLC').map((node) => node.protocolFamily)).toEqual([
      'modbus',
      'opcua',
      'siemens',
      'mitsubishi',
      'omron',
      'allen-bradley',
      'beckhoff',
      'yokogawa',
    ])
  })
  it('resolves optional point fields from driver features', () => {
    expect(
      collectorDriverSupportsFeature(
        { features: ['point.elementCount'] },
        collectorDriverFeatures.pointElementCount,
      ),
    ).toBe(true)
    expect(
      collectorDriverSupportsFeature({ features: [] }, collectorDriverFeatures.pointElementCount),
    ).toBe(false)
  })

  it('groups configured connections by protocol family', () => {
    const connection = {
      id: 'connection-1',
      projectId: 'project-1',
      name: '锅炉 PLC',
      code: '锅炉_plc',
      status: 'configured',
      configurationState: 'ready' as const,
      isEnabled: true,
      defaultAcquisition: { intervalMs: 1000, timeoutMs: 3000, retryCount: 0 },
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
  it('filters OPC UA browse nodes only by loaded NodeId values', () => {
    const node = {
      nodeId: 'ns=2;s=Line1.Temperature',
      browseName: '2:Temperature',
      displayName: '入口温度',
      nodeClass: 'variable',
      dataType: 'Float',
      hasChildren: false,
    }

    expect(filterCollectorBrowseNode('', node)).toBe(true)
    expect(filterCollectorBrowseNode('line1', node)).toBe(true)
    expect(filterCollectorBrowseNode('入口温度', node)).toBe(false)
  })

  it('builds editable variable defaults from an OPC UA variable node', () => {
    expect(
      buildCollectorPointCreateDefaults({
        nodeId: 'ns=2;s=Line1.Temperature',
        browseName: '2:Temperature',
        displayName: '入口温度',
        nodeClass: 'variable',
        dataType: 'Float',
        hasChildren: false,
      }),
    ).toEqual({
      name: '入口温度',
      dataType: 'float32',
      elementCount: 1,
      enabled: true,
      address: { nodeId: 'ns=2;s=Line1.Temperature' },
    })
  })

  it('maps page connection session states to consistent tones', () => {
    expect(collectorDebugConnectionTone('connected')).toBe('success')
    expect(collectorDebugConnectionTone('connecting')).toBe('primary')
    expect(collectorDebugConnectionTone('disconnecting')).toBe('primary')
    expect(collectorDebugConnectionTone('error')).toBe('danger')
    expect(collectorDebugConnectionTone('disconnected')).toBe('muted')
  })
  it('formats connection summaries and localized status labels', () => {
    const connection = {
      id: 'connection-1',
      projectId: 'project-1',
      name: 'OPC UA',
      code: 'opc_ua',
      status: 'unknown',
      configurationState: 'ready' as const,
      isEnabled: true,
      defaultAcquisition: { intervalMs: 1000, timeoutMs: 3000, retryCount: 0 },
      displayOrder: 0,
      protocolFamily: 'opcua',
      driverId: 'opcua.standard',
      driverVersion: '1.0.0',
      schemaVersion: 1,
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
