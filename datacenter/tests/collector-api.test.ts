import { beforeEach, describe, expect, it, vi } from 'vitest'

const { requestMock } = vi.hoisted(() => ({ requestMock: vi.fn() }))

vi.mock('@/utils/request', () => ({ default: requestMock }))

import { listAllCollectorDrivers } from '@/api/collector.api'
import type { CollectorDriverSummary } from '@/api/schemas/collector.schema'

function createDriver(index: number): CollectorDriverSummary {
  return {
    protocolFamily: 'test',
    driverId: `driver-${String(index).padStart(3, '0')}`,
    driverVersion: '1.0.0',
    schemaVersion: 1,
    displayName: `Driver ${index}`,
    category: 'plc',
    transports: ['tcp'],
    operations: ['connection.open'],
    features: [],
    dataTypes: ['int16'],
    acquisitionModes: ['polling'],
    platforms: { devAgent: ['windows-x64'] },
  }
}

function createPage(
  page: number,
  list: CollectorDriverSummary[],
  total: number,
  totalPages: number,
) {
  return {
    code: 0,
    msg: 'ok',
    data: {
      list,
      pagination: { page, pageSize: 100, total, totalPages },
    },
    reqId: 'req-1',
  }
}

describe('collector api', () => {
  beforeEach(() => requestMock.mockReset())

  it('loads and merges every collector driver page in order', async () => {
    const drivers = Array.from({ length: 115 }, (_, index) => createDriver(index + 1))
    requestMock
      .mockResolvedValueOnce(createPage(1, drivers.slice(0, 100), 115, 2))
      .mockResolvedValueOnce(createPage(2, drivers.slice(100), 115, 2))

    const result = await listAllCollectorDrivers()

    expect(result).toHaveLength(115)
    expect(result[0]?.driverId).toBe('driver-001')
    expect(result[114]?.driverId).toBe('driver-115')
    expect(requestMock).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ params: { page: 1, pageSize: 100 } }),
    )
    expect(requestMock).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ params: { page: 2, pageSize: 100 } }),
    )
  })
})
