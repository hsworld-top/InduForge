import { describe, expect, it } from 'vitest'
import { parseGridTemplateParts } from './DragDropManager'

describe('parseGridTemplateParts', () => {
  it('returns empty for none or empty', () => {
    expect(parseGridTemplateParts('')).toEqual([])
    expect(parseGridTemplateParts('none')).toEqual([])
  })

  it('splits space-separated tracks', () => {
    expect(parseGridTemplateParts('1fr 2fr auto')).toEqual(['1fr', '2fr', 'auto'])
  })

  it('expands repeat(n, value)', () => {
    expect(parseGridTemplateParts('repeat(3, minmax(0, 1fr))')).toEqual([
      'minmax(0, 1fr)',
      'minmax(0, 1fr)',
      'minmax(0, 1fr)',
    ])
  })

  it('when template is repeat-only, expands tracks (suffix tracks are not merged)', () => {
    expect(parseGridTemplateParts('repeat(2, 80px)')).toEqual(['80px', '80px'])
  })
})
