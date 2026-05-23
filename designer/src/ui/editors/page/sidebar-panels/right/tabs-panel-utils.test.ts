import { describe, expect, it } from 'vitest'
import {
  buildTabsChildKeyPatches,
  canRemoveTabsItem,
  createTabsItem,
  normalizeTabsItems,
  normalizeTabsModelValue,
  renameTabsItem,
} from './tabs-panel-utils'

describe('tabs-panel-utils', () => {
  it('为空或异常 tabs 生成可编辑的默认标签页', () => {
    expect(normalizeTabsItems(null)).toEqual([
      { name: 'tab1', label: '标签1', disabled: false, content: '' },
      { name: 'tab2', label: '标签2', disabled: false, content: '' },
    ])
    expect(normalizeTabsItems([])).toEqual([
      { name: 'tab1', label: '标签1', disabled: false, content: '' },
      { name: 'tab2', label: '标签2', disabled: false, content: '' },
    ])
  })

  it('新增标签页时自动生成唯一 name', () => {
    const next = createTabsItem([
      { name: 'tab1', label: '标签1' },
      { name: 'tab2', label: '标签2' },
    ])

    expect(next).toMatchObject({
      name: 'tab3',
      label: '标签3',
      disabled: false,
      content: '',
    })
  })

  it('标准化默认激活值到有效标签页', () => {
    const items = normalizeTabsItems([
      { name: 'base', label: '基本' },
      { name: 'conf', label: '配置' },
    ])

    expect(normalizeTabsModelValue('conf', items)).toBe('conf')
    expect(normalizeTabsModelValue('missing', items)).toBe('base')
  })

  it('忽略旧 DSL 数组 content，避免在画布里显示 JSON 文本', () => {
    expect(
      normalizeTabsItems([{ name: 'base', label: '基本', content: [{ type: 'Text' }] }])[0]
        ?.content,
    ).toBe('')
  })

  it('重命名标签页时校验唯一性并返回同步后的 tabs', () => {
    const items = normalizeTabsItems([
      { name: 'a', label: 'A' },
      { name: 'b', label: 'B' },
    ])

    expect(renameTabsItem(items, 'a', 'b').ok).toBe(false)
    expect(renameTabsItem(items, 'a', 'c')).toEqual({
      ok: true,
      oldName: 'a',
      newName: 'c',
      items: [
        { name: 'c', label: 'A', disabled: false, content: '' },
        { name: 'b', label: 'B', disabled: false, content: '' },
      ],
    })
  })

  it('生成子组件 tabKey 迁移补丁', () => {
    const patches = buildTabsChildKeyPatches(
      [
        { id: 'child-a', props: { tabKey: 'old', text: 'A' } },
        { id: 'child-b', props: { tabKey: 'other' } },
      ],
      'old',
      'next',
    )

    expect(patches).toEqual([{ id: 'child-a', props: { tabKey: 'next', text: 'A' } }])
  })

  it('删除含子组件的标签页时返回阻止结果', () => {
    expect(
      canRemoveTabsItem('a', [
        { id: 'child-a', props: { tabKey: 'a' } },
        { id: 'child-b', props: { tabKey: 'b' } },
      ]),
    ).toEqual({ ok: false, reason: 'hasChildren' })
    expect(canRemoveTabsItem('c', [{ id: 'child-a', props: { tabKey: 'a' } }])).toEqual({
      ok: true,
    })
  })
})
