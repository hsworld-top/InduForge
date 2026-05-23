import { describe, expect, it } from 'vitest'
import { requireConnectionsPayload } from './datapoint-payload'

describe('requireConnectionsPayload', () => {
  it('兼容 data 直接解包为连接数组的响应', () => {
    const connections = [{ id: 'conn_1', name: '连接1' }]

    expect(requireConnectionsPayload(connections)).toBe(connections)
  })

  it('兼容 connections 包裹格式', () => {
    const connections = [{ id: 'conn_1', name: '连接1' }]

    expect(requireConnectionsPayload({ connections })).toBe(connections)
  })
})
