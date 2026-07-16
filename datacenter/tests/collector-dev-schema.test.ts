import { describe, expect, it } from 'vitest'
import {
  CollectorAgentPageSchema,
  CollectorAgentSchema,
  CollectorRegistrationCodeSchema,
} from '@/api/schemas/collector-dev.schema'

describe('collector dev schemas', () => {
  it('parses agent capabilities and lifecycle state', () => {
    const agent = CollectorAgentSchema.parse({
      id: 'agent-1',
      name: 'dev-machine',
      os: 'windows',
      arch: 'x64',
      version: '0.1.0',
      ipAddress: '192.168.1.10',
      status: 'online',
      capabilities: [
        {
          driverId: 'opcua.standard',
          driverVersion: '1.0.0',
          schemaVersions: [2],
          operations: ['connection.test', 'device.browse', 'point.read'],
        },
      ],
      lastSeenAt: '2026-07-13 12:00:00',
      createdAt: '2026-07-13 11:00:00',
    })
    expect(agent.status).toBe('online')
    expect(agent.ipAddress).toBe('192.168.1.10')
    expect(agent.capabilities[0]?.operations).toContain('point.read')
  })

  it('requires one-time registration code fields', () => {
    expect(() =>
      CollectorRegistrationCodeSchema.parse({ expiresAt: '2026-07-13 12:10:00' }),
    ).toThrow()
  })

  it('parses paginated agent response', () => {
    const page = CollectorAgentPageSchema.parse({
      list: [],
      pagination: { page: 1, pageSize: 10, total: 0, totalPages: 0 },
    })
    expect(page.pagination.pageSize).toBe(10)
    expect(page.list).toEqual([])
  })
})
