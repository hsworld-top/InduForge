import { describe, expect, it } from 'vitest'
import { buildComponentMethodCompletions, flattenComponentTree } from './component-methods'

describe('component-methods', () => {
  it('flattenComponentTree collects named components', () => {
    const tree = [
      {
        type: 'component',
        componentName: 'btn1',
        componentType: 'Button',
        children: [
          { type: 'component', componentName: 'tbl1', componentType: 'Table', children: [] },
        ],
      },
    ]
    expect(flattenComponentTree(tree)).toEqual([
      { componentName: 'btn1', componentType: 'Button' },
      { componentName: 'tbl1', componentType: 'Table' },
    ])
  })

  it('buildComponentMethodCompletions includes common props and type methods', () => {
    const items = buildComponentMethodCompletions([
      { type: 'component', componentName: 'b', componentType: 'Button', children: [] },
    ])
    expect(items.some((i) => i.label === 'Click' && i.prefix === 'components.b.')).toBe(true)
    expect(items.some((i) => i.label === 'Name' && i.kind === 'Property')).toBe(true)
  })
})
