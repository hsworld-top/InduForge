import { describe, expect, it } from 'vitest'
import {
  collectorDebugFailureText,
  collectorDebugQualityTone,
  collectorDebugTime,
  currentPageCollectorPointIds,
  formatCollectorDebugValue,
  hasCollectorDebugSuccess,
  summarizeCollectorPointRead,
} from '../src/components/collector-workbench/collector-debug-snapshot'

describe('collector debug snapshot helpers', () => {
  it('formats scalar and structured values without changing content', () => {
    expect(formatCollectorDebugValue('running')).toBe('running')
    expect(formatCollectorDebugValue(12.5)).toBe('12.5')
    expect(formatCollectorDebugValue(null)).toBe('null')
    expect(formatCollectorDebugValue([1, 2])).toBe('[1,2]')
    expect(formatCollectorDebugValue({ value: true })).toBe('{"value":true}')
    expect(formatCollectorDebugValue(12.5, '12.500')).toBe('12.500')
  })

  it('distinguishes a successful null value from no historical success', () => {
    const failedOnly = {
      value: null,
      valueText: null,
      dataType: null,
      quality: null,
      sourceTimestamp: null,
      serverTimestamp: null,
      readAt: null,
      lastAttemptStatus: 'failed' as const,
      lastAttemptAt: '2026-07-20 10:00:00',
      lastErrorCode: 'READ_FAILED',
      lastErrorMessage: '读取失败',
    }
    expect(hasCollectorDebugSuccess(failedOnly)).toBe(false)
    expect(hasCollectorDebugSuccess({ ...failedOnly, readAt: '2026-07-20 09:59:00' })).toBe(true)
  })

  it('maps OPC UA quality to visual tones', () => {
    expect(collectorDebugQualityTone('Good')).toBe('success')
    expect(collectorDebugQualityTone('UncertainLastUsableValue')).toBe('warning')
    expect(collectorDebugQualityTone('BadOutOfService')).toBe('danger')
    expect(collectorDebugQualityTone(null)).toBe('info')
  })

  it('builds one current-page batch including disabled points', () => {
    const points = [
      { id: 'point-1', name: '温度', enabled: true },
      { id: 'point-2', name: '压力', enabled: false },
    ]
    expect(currentPageCollectorPointIds(points)).toEqual(['point-1', 'point-2'])
    expect(
      summarizeCollectorPointRead(points, {
        values: [
          {
            pointId: 'point-1',
            succeeded: true,
            value: 12.5,
            dataType: 'float64',
            quality: 'Good',
            sourceTimestamp: null,
            serverTimestamp: null,
            errorCode: null,
            errorMessage: null,
          },
          {
            pointId: 'point-2',
            succeeded: false,
            value: null,
            dataType: null,
            quality: null,
            sourceTimestamp: null,
            serverTimestamp: null,
            errorCode: 'READ_FAILED',
            errorMessage: '读取失败',
          },
        ],
        diagnostics: [],
      }),
    ).toEqual({
      successCount: 1,
      failures: [{ index: 1, name: '压力', code: 'READ_FAILED', message: '读取失败' }],
    })
  })

  it('keeps the last success visible while describing a failed attempt', () => {
    expect(
      collectorDebugFailureText({
        value: 12.5,
        valueText: '12.5',
        dataType: 'float64',
        quality: 'Good',
        sourceTimestamp: '2026-07-20 10:00:00',
        serverTimestamp: '2026-07-20 10:00:01',
        readAt: '2026-07-20 10:00:02',
        lastAttemptStatus: 'failed',
        lastAttemptAt: '2026-07-20 10:01:00',
        lastErrorCode: 'READ_FAILED',
        lastErrorMessage: '连接已断开',
      }),
    ).toBe('连接已断开')
    expect(collectorDebugTime(null)).toBe('—')
  })
})
