import { describe, expect, it } from 'vitest'
import type { ProjectSchema } from '@/editor-core/document/types'
import {
  createComponentNode,
  createEmptySchema,
  createPageNode,
} from '@/editor-core/document/types'
import {
  joinI18nResource,
  normalizeProjectI18nSettings,
  scanProjectI18nResources,
  translatePropsWithI18n,
} from './project-i18n'

function buildSchema(): ProjectSchema {
  const schema = createEmptySchema({ projectId: 'proj_1' })
  const page = createPageNode({ id: 'page_home', name: '首页' })
  const root = createComponentNode('FreeContainer', { id: page.rootNodeId })
  const button = createComponentNode('Button', {
    id: 'node_submit',
    label: '提交按钮',
    props: { text: '提交', icon: 'Search' },
  })
  root.children = [button.id]
  schema.pagesById[page.id] = page
  schema.nodesById[root.id] = root
  schema.nodesById[button.id] = button
  schema.entry.homePageId = page.id
  return schema
}

describe('project i18n', () => {
  it('扫描工程静态文案候选', () => {
    const schema = buildSchema()
    const settings = normalizeProjectI18nSettings(null)
    const result = scanProjectI18nResources(
      [{ pageId: 'page_home', pageName: '首页', schema }],
      settings,
    )

    expect(
      result.rows.some((row) => row.nodeId === 'node_submit' && row.fieldPath === 'text'),
    ).toBe(true)
    expect(result.rows.some((row) => row.fieldPath === 'icon')).toBe(false)
  })

  it('扫描明确文案字段里的英文按钮文本', () => {
    const schema = buildSchema()
    const node = schema.nodesById.node_submit
    if (node) node.props = { text: 'button', icon: 'Search', type: 'primary' }

    const result = scanProjectI18nResources(
      [{ pageId: 'page_home', pageName: '首页', schema }],
      normalizeProjectI18nSettings(null),
    )

    expect(
      result.rows.some(
        (row) =>
          row.nodeId === 'node_submit' && row.fieldPath === 'text' && row.sourceText === 'button',
      ),
    ).toBe(true)
    expect(result.rows.some((row) => row.fieldPath === 'icon')).toBe(false)
    expect(result.rows.some((row) => row.fieldPath === 'type')).toBe(false)
  })

  it('不扫描布局组件 text 字段里的英文内部状态值', () => {
    const schema = createEmptySchema({ projectId: 'proj_1' })
    const page = createPageNode({ id: 'page_home', name: '首页' })
    const root = createComponentNode('FreeContainer', { id: page.rootNodeId })
    const tabs = createComponentNode('Tabs', {
      id: 'node_tabs',
      label: '选项卡布局',
      props: {
        text: 'activeTab',
        tabs: [{ name: 'base', label: 'Base' }],
      },
    })
    root.children = [tabs.id]
    schema.pagesById[page.id] = page
    schema.nodesById[root.id] = root
    schema.nodesById[tabs.id] = tabs
    schema.entry.homePageId = page.id

    const result = scanProjectI18nResources(
      [{ pageId: 'page_home', pageName: '首页', schema }],
      normalizeProjectI18nSettings(null),
    )

    expect(
      result.rows.some(
        (row) =>
          row.nodeId === 'node_tabs' &&
          row.fieldPath === 'tabs.0.label' &&
          row.sourceText === 'Base',
      ),
    ).toBe(true)
    expect(
      result.rows.some(
        (row) =>
          row.nodeId === 'node_tabs' && row.fieldPath === 'text' && row.sourceText === 'activeTab',
      ),
    ).toBe(false)
  })

  it('扫描物料默认 props 中的静态文案', () => {
    const schema = createEmptySchema({ projectId: 'proj_1' })
    const page = createPageNode({ id: 'page_home', name: '首页' })
    const root = createComponentNode('FreeContainer', { id: page.rootNodeId })
    const button = createComponentNode('Button', {
      id: 'node_default_button',
      label: '默认按钮',
    })
    const collapse = createComponentNode('Collapse', {
      id: 'node_collapse',
      label: '折叠面板',
    })
    root.children = [button.id, collapse.id]
    schema.pagesById[page.id] = page
    schema.nodesById[root.id] = root
    schema.nodesById[button.id] = button
    schema.nodesById[collapse.id] = collapse
    schema.entry.homePageId = page.id

    const result = scanProjectI18nResources(
      [{ pageId: 'page_home', pageName: '首页', schema }],
      normalizeProjectI18nSettings(null),
    )

    expect(
      result.rows.some(
        (row) =>
          row.nodeId === 'node_default_button' &&
          row.fieldPath === 'text' &&
          row.sourceText === '按钮',
      ),
    ).toBe(true)
    expect(
      result.rows.some(
        (row) =>
          row.nodeId === 'node_collapse' &&
          row.fieldPath === 'items.0.title' &&
          row.sourceText === '面板1',
      ),
    ).toBe(true)
    expect(
      result.rows.some(
        (row) =>
          row.nodeId === 'node_collapse' &&
          row.fieldPath === 'items.0.content' &&
          row.sourceText === '内容1',
      ),
    ).toBe(true)
  })

  it('扫描按钮别名类型的默认文案', () => {
    const schema = createEmptySchema({ projectId: 'proj_1' })
    const page = createPageNode({ id: 'page_home', name: '首页' })
    const root = createComponentNode('FreeContainer', { id: page.rootNodeId })
    const button = createComponentNode('button', {
      id: 'node_lower_button',
      label: '按钮',
    })
    const elButton = createComponentNode('ElButton', {
      id: 'node_el_button',
      label: '确认按钮',
    })
    root.children = [button.id, elButton.id]
    schema.pagesById[page.id] = page
    schema.nodesById[root.id] = root
    schema.nodesById[button.id] = button
    schema.nodesById[elButton.id] = elButton
    schema.entry.homePageId = page.id

    const result = scanProjectI18nResources(
      [{ pageId: 'page_home', pageName: '首页', schema }],
      normalizeProjectI18nSettings(null),
    )

    expect(
      result.rows.some(
        (row) =>
          row.nodeId === 'node_lower_button' &&
          row.fieldPath === 'text' &&
          row.sourceText === '按钮',
      ),
    ).toBe(true)
    expect(
      result.rows.some(
        (row) =>
          row.nodeId === 'node_el_button' && row.fieldPath === 'text' && row.sourceText === '按钮',
      ),
    ).toBe(true)
  })

  it('加入资源后按语言解析 props', () => {
    const schema = buildSchema()
    const node = schema.nodesById.node_submit
    expect(node).toBeTruthy()
    const settings = normalizeProjectI18nSettings({ enabled: true })
    const [row] = scanProjectI18nResources(
      [{ pageId: 'page_home', pageName: '首页', schema }],
      settings,
    ).rows.filter((item) => item.fieldPath === 'text')
    expect(row).toBeTruthy()
    const resourceRow = row!

    const nextSettings = joinI18nResource(
      settings,
      node!,
      { ...resourceRow, currentValue: 'Submit' },
      '2026-05-18 10:00:00',
    )
    const translated = translatePropsWithI18n(node!.props, node!, nextSettings, 'en-US')

    expect(node!.i18n?.props?.text).toBe(resourceRow.resourceKey)
    expect(translated.text).toBe('Submit')
  })
})
