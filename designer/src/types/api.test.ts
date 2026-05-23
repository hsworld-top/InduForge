import { describe, expect, it } from 'vitest'
import { unwrapApiData } from './api'

describe('types/api.unwrapApiData', () => {
  it('code===0 时返回 data', () => {
    const payload = {
      code: 0,
      msg: 'ok',
      data: { id: 'page-1' },
      reqId: 'req-1',
    }
    expect(unwrapApiData(payload)).toEqual({ id: 'page-1' })
  })

  it('code!=0 时抛出业务错误消息', () => {
    const payload = {
      code: 25003,
      msg: '页面已被其他用户锁定',
      data: { lockedByName: '用户二' },
    }
    expect(() => unwrapApiData(payload)).toThrow('页面已被其他用户锁定')
  })

  it('非包络数据保持原样返回', () => {
    expect(unwrapApiData({ pages: [] })).toEqual({ pages: [] })
  })
})
