import { describe, expect, it } from 'vitest'
import {
  CollectorAgentSchema,
  CollectorRegistrationCodeSchema,
} from '@/api/schemas/collector-dev.schema'

describe('collector dev schemas', () => {
  it('parses agent capabilities and online state', () => {
    const agent = CollectorAgentSchema.parse({
      id: 'agent-1',
      name: 'dev-machine',
      os: 'windows',
      arch: 'x64',
      version: '0.1.0',
      online: true,
      capabilities: [
        {
          protocolType: 'opcua',
          capabilityVersion: '1.0',
          operations: ['connection.test', 'opcua.browse', 'opcua.read'],
        },
      ],
      lastSeenAt: '2026-07-13 12:00:00',
      createdAt: '2026-07-13 11:00:00',
    })
    expect(agent.online).toBe(true)
    expect(agent.capabilities[0]?.operations).toContain('opcua.read')
  })

  it('requires one-time registration code fields', () => {
    expect(() =>
      CollectorRegistrationCodeSchema.parse({ expiresAt: '2026-07-13 12:10:00' }),
    ).toThrow()
  })
})
