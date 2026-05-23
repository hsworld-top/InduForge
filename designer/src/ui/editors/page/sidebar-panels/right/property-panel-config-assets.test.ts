import { describe, expect, it } from 'vitest'
import { buildConfigAssetTree, decodeAssetName } from './property-panel-config-assets'

describe('property-panel-config-assets', () => {
  it('decodeAssetName decodes percent-encoding', () => {
    expect(decodeAssetName('a%20b')).toBe('a b')
  })

  it('decodeAssetName returns empty for empty input', () => {
    expect(decodeAssetName('')).toBe('')
  })

  it('buildConfigAssetTree nests folders and assets', () => {
    const tree = buildConfigAssetTree(
      [
        { id: 'f1', name: 'F1', parentId: null },
        { id: 'f2', name: 'F2', parentId: 'f1' },
      ],
      [{ id: 'a1', name: 'pic.png', folderId: 'f2' }],
    )
    const root = tree[0]
    expect(root?.label).toBe('全部资源')
    const f1 = root?.children?.find((c) => c.id === 'f1')
    const f2 = f1?.children?.find((c) => c.id === 'f2')
    expect(f2?.children?.some((c) => c.id === 'a1' && c.type === 'asset')).toBe(true)
  })
})
