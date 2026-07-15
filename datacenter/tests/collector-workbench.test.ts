import { describe, expect, it } from 'vitest'
import {
  agentSupportsOperation,
  collectorAgentStorageKey,
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
  it('stores agent selection per project', () => {
    expect(collectorAgentStorageKey('project-1')).toBe('induforge:collector-agent:project-1')
  })
})
