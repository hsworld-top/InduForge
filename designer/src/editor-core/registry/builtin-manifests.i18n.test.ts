import { afterEach, describe, expect, it } from 'vitest'
import { i18n } from '@/i18n'
import { getManifest } from '@/materials/manifests'
import { registerBuiltinComponents } from './builtin-manifests'
import { componentRegistry } from './component-registry'

describe('builtin-manifests i18n', () => {
  afterEach(() => {
    i18n.global.locale.value = 'zh'
    componentRegistry.clear()
  })

  it('rebuilds component registry and manifest labels with locale changes', () => {
    registerBuiltinComponents()

    expect(componentRegistry.get('ElContainer')?.name).toBe('区域布局')
    expect(
      getManifest('HorizontalLayout')?.props.find((item) => item.name === 'justify')?.label,
    ).toBe('水平排列')
    expect(
      getManifest('ElContainer')?.props.find((item) => item.name === 'showHeader')?.group,
    ).toBe('显示')

    i18n.global.locale.value = 'en'
    registerBuiltinComponents()

    expect(componentRegistry.get('ElContainer')?.name).toBe('Region Layout')
    expect(
      getManifest('HorizontalLayout')?.props.find((item) => item.name === 'justify')?.label,
    ).toBe('Horizontal Distribution')
    expect(
      getManifest('ElContainer')?.props.find((item) => item.name === 'showHeader')?.group,
    ).toBe('Display')
  })

  it('exposes localized high-frequency Button property groups', () => {
    registerBuiltinComponents()

    const buttonManifest = getManifest('Button')
    expect(buttonManifest?.props.find((item) => item.name === 'text')).toMatchObject({
      label: '按钮文字',
      group: '内容',
      defaultValue: '按钮',
    })
    expect(buttonManifest?.props.find((item) => item.name === 'icon')).toMatchObject({
      label: '图标',
      editor: 'icon',
      placeholder: '选择或输入图标名',
      options: [
        { label: '搜索', value: 'Search' },
        { label: '新增', value: 'Plus' },
        { label: '下载', value: 'Download' },
        { label: '上传', value: 'Upload' },
        { label: '删除', value: 'Delete' },
        { label: '编辑', value: 'EditPen' },
        { label: '刷新', value: 'Refresh' },
        { label: '关闭', value: 'Close' },
        { label: '文档', value: 'Document' },
        { label: '文件夹', value: 'Folder' },
      ],
    })
    expect(buttonManifest?.props.find((item) => item.name === 'size')).toMatchObject({
      label: '视觉规格',
      group: '外观',
    })
    expect(buttonManifest?.props.find((item) => item.name === 'shape')?.options).toEqual([
      { label: '默认', value: 'default' },
      { label: '圆角', value: 'round' },
      { label: '圆形', value: 'circle' },
    ])
    expect(buttonManifest?.props.find((item) => item.group === '布局')).toBeUndefined()
    expect(buttonManifest?.props.find((item) => item.group === '更多属性')).toBeUndefined()
    expect(buttonManifest?.props.find((item) => item.name === 'nativeType')).toBeUndefined()

    i18n.global.locale.value = 'en'
    registerBuiltinComponents()

    const enButtonManifest = getManifest('Button')
    expect(enButtonManifest?.props.find((item) => item.name === 'text')).toMatchObject({
      label: 'Button Text',
      group: 'Content',
      defaultValue: '按钮',
    })
    expect(enButtonManifest?.props.find((item) => item.name === 'icon')).toMatchObject({
      label: 'Icon',
      editor: 'icon',
      placeholder: 'Select or enter icon name',
      options: [
        { label: 'Search', value: 'Search' },
        { label: 'Add', value: 'Plus' },
        { label: 'Download', value: 'Download' },
        { label: 'Upload', value: 'Upload' },
        { label: 'Delete', value: 'Delete' },
        { label: 'Edit', value: 'EditPen' },
        { label: 'Refresh', value: 'Refresh' },
        { label: 'Close', value: 'Close' },
        { label: 'Document', value: 'Document' },
        { label: 'Folder', value: 'Folder' },
      ],
    })
    expect(enButtonManifest?.props.find((item) => item.name === 'size')).toMatchObject({
      label: 'Visual Spec',
      group: 'Appearance',
    })
    expect(enButtonManifest?.props.find((item) => item.name === 'shape')?.options).toEqual([
      { label: 'Default', value: 'default' },
      { label: 'Rounded', value: 'round' },
      { label: 'Circle', value: 'circle' },
    ])
    expect(enButtonManifest?.props.find((item) => item.group === 'Layout')).toBeUndefined()
    expect(enButtonManifest?.props.find((item) => item.group === 'More')).toBeUndefined()
    expect(enButtonManifest?.props.find((item) => item.name === 'nativeType')).toBeUndefined()
  })
})
