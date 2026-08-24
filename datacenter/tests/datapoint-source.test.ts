import { describe, expect, test } from 'vitest'
import {
  resolveDatapointSourceDisplayName,
  resolveDatapointSourceNavigation,
} from '../src/models/datapoint-source'

describe('datapoint source navigation', () => {
  test('计算输出定位到计算单元', () => {
    expect(
      resolveDatapointSourceNavigation({
        sourceType: 'calc.output',
        sourceId: 'compute-1',
      }),
    ).toEqual({
      module: 'compute',
      objectId: 'compute-1',
      label: '计算单元',
      actionLabel: '打开计算单元',
    })
  })

  test('工业采集点定位到采集连接', () => {
    expect(
      resolveDatapointSourceNavigation({
        sourceType: 'collector.point',
        accessSourceId: 'collector-1',
      }),
    ).toMatchObject({ module: 'industrial-collector', objectId: 'collector-1' })
  })

  test.each(['db.query', 'mqtt.subscription', 'mqtt.tag'])(
    '%s 使用后端解析的接入源连接',
    (sourceType) => {
      expect(
        resolveDatapointSourceNavigation({ sourceType, accessSourceId: 'connection-1' }),
      ).toMatchObject({ module: 'access-source', objectId: 'connection-1' })
    },
  )

  test.each(['http.request', 'websocket.session', 'realtime.key', 'kafka.field', 'kafka.raw'])(
    '%s 可从 sourceId 回退解析连接',
    (sourceType) => {
      expect(
        resolveDatapointSourceNavigation({ sourceType, sourceId: 'connection-1' }),
      ).toMatchObject({ module: 'access-source', objectId: 'connection-1' })
    },
  )

  test('无可导航来源时不产生错误目标', () => {
    expect(resolveDatapointSourceNavigation({ sourceType: 'static.var' })).toBeNull()
    expect(resolveDatapointSourceDisplayName({ sourceType: 'static.var' })).toBe('静态变量')
  })
})
