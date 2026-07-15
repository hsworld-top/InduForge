import { describe, expect, it } from 'vitest'
import { CollectorConnectionSchema, splitCollectorFormValues } from '@/api/schemas/collector.schema'

describe('collector schemas', () => {
  it('keeps secret values out of connection responses', () => {
    const parsed = CollectorConnectionSchema.parse({
      id: 'connection-1',
      projectId: 'project-1',
      name: 'OPC UA',
      status: 'offline',
      enabled: true,
      displayOrder: 0,
      protocolFamily: 'opcua',
      driverId: 'opcua.standard',
      driverVersion: '1.0.0',
      schemaVersion: 1,
      config: { endpointUrl: 'opc.tcp://127.0.0.1:4840' },
      metadata: {},
      secretStatus: { password: true },
      createdAt: '2026-07-15 10:00:00',
      updatedAt: '2026-07-15 10:00:00',
    })
    expect(parsed.secretStatus.password).toBe(true)
    expect(JSON.stringify(parsed)).not.toContain('plain-password')
  })

  it('splits secret fields from public config', () => {
    const result = splitCollectorFormValues(
      {
        properties: {
          host: { type: 'string' },
          password: { type: 'string', 'x-induforge-secret': true },
        },
      },
      { host: '127.0.0.1', password: 'plain-password' },
    )
    expect(result.config).toEqual({ host: '127.0.0.1' })
    expect(result.secrets).toEqual({ password: 'plain-password' })
  })
})
