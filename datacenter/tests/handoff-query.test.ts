import { describe, expect, it } from 'vitest'
import { hasHandoffQuery, stripHandoffQuery } from '@/router/handoff-query'

describe('handoff-query', () => {
  it('移除 handoff 相关查询参数并保留业务查询参数', () => {
    expect(
      stripHandoffQuery({
        handoff: 'handoff_1',
        handoffId: 'handoff_1',
        keyword: 'demo',
        page: '1',
      }),
    ).toEqual({
      keyword: 'demo',
      page: '1',
    })
  })

  it('识别是否包含 handoff 查询参数', () => {
    expect(hasHandoffQuery({ keyword: 'demo' })).toBe(false)
    expect(hasHandoffQuery({ handoff: 'handoff_1' })).toBe(true)
    expect(hasHandoffQuery({ handoffId: 'handoff_1' })).toBe(true)
  })
})
