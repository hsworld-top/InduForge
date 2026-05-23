import { describe, expect, it } from 'vitest'
import { elementTypeName, formatStyleValue } from './property-panel-utils'

describe('property-panel-utils', () => {
  describe('formatStyleValue', () => {
    it('maps empty-ish values to dash', () => {
      expect(formatStyleValue(undefined)).toBe('-')
      expect(formatStyleValue(null)).toBe('-')
      expect(formatStyleValue('')).toBe('-')
    })

    it('stringifies other values', () => {
      expect(formatStyleValue(12)).toBe('12')
      expect(formatStyleValue(false)).toBe('false')
    })

    it('preserves zero as string', () => {
      expect(formatStyleValue(0)).toBe('0')
    })
  })

  describe('elementTypeName', () => {
    it('trims string types', () => {
      expect(elementTypeName('  ElInput  ')).toBe('ElInput')
    })

    it('returns empty for non-string', () => {
      expect(elementTypeName(undefined)).toBe('')
    })
  })
})
