import { describe, expect, it } from 'vitest'
import {
  CollectorConnectionDiagnosticSchema,
  CollectorConnectionSchema,
  splitCollectorFormValues,
} from '@/api/schemas/collector.schema'

describe('collector schemas', () => {
  it('keeps secret values out of connection responses', () => {
    const parsed = CollectorConnectionSchema.parse({
      id: 'connection-1',
      projectId: 'project-1',
      name: 'OPC UA',
      code: 'opc_ua',
      configurationState: 'ready',
      displayOrder: 0,
      protocolFamily: 'opcua',
      driverId: 'opcua.standard',
      driverVersion: '1.0.0',
      schemaVersion: 1,
      config: { host: '127.0.0.1', port: 4840, endpointPath: '/' },
      metadata: {},
      secretStatus: { password: true },
      createdAt: '2026-07-15 10:00:00',
      updatedAt: '2026-07-15 10:00:00',
    })
    expect(parsed.secretStatus.password).toBe(true)
    expect(JSON.stringify(parsed)).not.toContain('plain-password')
  })

  it('accepts the backend response without legacy status and normalizes nullable maps', () => {
    const parsed = CollectorConnectionSchema.parse({
      id: 'connection-2',
      projectId: 'project-1',
      name: 'Modbus TCP',
      code: 'modbus_tcp',
      configurationState: 'incomplete',
      displayOrder: 0,
      protocolFamily: 'modbus',
      driverId: 'modbus.tcp',
      driverVersion: '1.0.0',
      schemaVersion: 1,
      config: null,
      metadata: null,
      secretStatus: null,
      createdAt: '2026-07-15 10:00:00',
      updatedAt: '2026-07-15 10:00:00',
    })
    expect(parsed.config).toEqual({})
    expect(parsed.metadata).toEqual({})
    expect(parsed.secretStatus).toEqual({})
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

  it('parses the nested collector diagnostic contract', () => {
    const parsed = CollectorConnectionDiagnosticSchema.parse({
      agent: { ready: false, reason: '未选择调试代理' },
      lastTest: { status: 'not_tested', durationMs: 0 },
      pointCount: 0,
      attemptedPointCount: 0,
      succeededPointCount: 0,
      failedPointCount: 0,
      recentReadSuccessRate: null,
      failedPoints: [],
    })
    expect(parsed.agent.ready).toBe(false)
    expect(parsed.lastTest.status).toBe('not_tested')
  })
})
