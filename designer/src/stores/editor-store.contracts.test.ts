import { describe, expect, it } from 'vitest'
import { normalizeEditorPageTabsForState } from './editor-store.contracts'

describe('editor-store.contracts', () => {
  it('normalizeEditorPageTabsForState shallow-clones tab items', () => {
    const tabs = [{ id: 'p1', name: 'A', isDirty: true }]
    const next = normalizeEditorPageTabsForState(tabs)
    expect(next).toEqual(tabs)
    expect(next[0]).not.toBe(tabs[0])
  })

  it('normalizeEditorPageTabsForState returns [] for non-array', () => {
    expect(normalizeEditorPageTabsForState(null as unknown as [])).toEqual([])
  })
})
