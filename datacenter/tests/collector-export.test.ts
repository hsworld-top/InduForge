import { describe, expect, it } from 'vitest'
import {
  estimateCollectorExportRows,
  parseCollectorExportPages,
} from '@/components/collector-workbench/collector-export'

describe('collector export pages', () => {
  it('parses discrete pages and ranges', () => {
    expect(parseCollectorExportPages('1,3-5,8', 10)).toEqual({
      pages: [1, 3, 4, 5, 8],
      error: '',
    })
  })

  it('rejects invalid and out-of-range pages', () => {
    expect(parseCollectorExportPages('4-2', 10).error).toContain('范围无效')
    expect(parseCollectorExportPages('11', 10).error).toContain('超出')
  })

  it('estimates the last partial page correctly', () => {
    expect(estimateCollectorExportRows([1, 3], 50, 120)).toBe(70)
  })
})
