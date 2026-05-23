import { describe, expect, it } from 'vitest'
import {
  buildCollapseChildKeyPatches,
  canRemoveCollapseItem,
  createCollapseItem,
  normalizeCollapseItems,
  normalizeCollapseModelValue,
  renameCollapseItem,
} from './collapse-panel-utils'

describe('collapse-panel-utils', () => {
  it('为空或异常 items 生成可编辑的默认面板项', () => {
    expect(normalizeCollapseItems(null)).toEqual([
      { name: '1', title: '面板1', disabled: false, content: '内容1' },
      { name: '2', title: '面板2', disabled: false, content: '内容2' },
    ])
    expect(normalizeCollapseItems([])).toEqual([
      { name: '1', title: '面板1', disabled: false, content: '内容1' },
      { name: '2', title: '面板2', disabled: false, content: '内容2' },
    ])
    expect(normalizeCollapseItems('collapseItems')).toEqual([
      { name: '1', title: '面板1', disabled: false, content: '内容1' },
      { name: '2', title: '面板2', disabled: false, content: '内容2' },
    ])
  })

  it('新增面板项时自动生成唯一 name', () => {
    const next = createCollapseItem([
      { name: 'panel1', title: '面板一' },
      { name: 'panel2', title: '面板二' },
    ])

    expect(next).toMatchObject({
      name: 'panel3',
      title: '面板3',
      disabled: false,
      content: '',
    })
  })

  it('默认数字标识面板新增时从面板3开始', () => {
    const next = createCollapseItem([
      { name: '1', title: '面板1' },
      { name: '2', title: '面板2' },
    ])

    expect(next).toMatchObject({
      name: 'panel3',
      title: '面板3',
    })
  })

  it('标准化旧的自动生成标题为阿拉伯数字格式', () => {
    expect(normalizeCollapseItems([{ name: 'panel1', title: '面板一' }])[0]?.title).toBe('面板1')
    expect(normalizeCollapseItems([{ name: 'panel2', title: '面板 2' }])[0]?.title).toBe('面板2')
  })

  it('切换手风琴模式时把展开值转换为单值或数组', () => {
    const items = normalizeCollapseItems([
      { name: 'a', title: 'A' },
      { name: 'b', title: 'B' },
    ])

    expect(normalizeCollapseModelValue(['a', 'b'], items, true)).toBe('a')
    expect(normalizeCollapseModelValue('a', items, false)).toEqual(['a'])
    expect(normalizeCollapseModelValue(['missing', 'b'], items, false)).toEqual(['b'])
  })

  it('重命名面板项时校验唯一性并返回同步后的 items', () => {
    const items = normalizeCollapseItems([
      { name: 'a', title: 'A' },
      { name: 'b', title: 'B' },
    ])

    expect(renameCollapseItem(items, 'a', 'b').ok).toBe(false)
    expect(renameCollapseItem(items, 'a', 'c')).toEqual({
      ok: true,
      oldName: 'a',
      newName: 'c',
      items: [
        { name: 'c', title: 'A', disabled: false, content: '' },
        { name: 'b', title: 'B', disabled: false, content: '' },
      ],
    })
  })

  it('生成子组件 collapseKey 迁移补丁', () => {
    const patches = buildCollapseChildKeyPatches(
      [
        { id: 'child-a', props: { collapseKey: 'old', text: 'A' } },
        { id: 'child-b', props: { collapseKey: 'other' } },
      ],
      'old',
      'next',
    )

    expect(patches).toEqual([{ id: 'child-a', props: { collapseKey: 'next', text: 'A' } }])
  })

  it('删除含子组件的面板项时返回阻止结果', () => {
    expect(
      canRemoveCollapseItem('a', [
        { id: 'child-a', props: { collapseKey: 'a' } },
        { id: 'child-b', props: { collapseKey: 'b' } },
      ]),
    ).toEqual({ ok: false, reason: 'hasChildren' })
    expect(canRemoveCollapseItem('c', [{ id: 'child-a', props: { collapseKey: 'a' } }])).toEqual({
      ok: true,
    })
  })
})
