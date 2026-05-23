import { describe, expect, it } from 'vitest'
import {
  checkboxGroupManifest,
  inputManifest,
  inputNumberManifest,
  paginationManifest,
  radioGroupManifest,
  selectManifest,
  switchManifest,
  tableManifest,
} from './manifest'

describe('ElementPlusCore manifests', () => {
  it('defines the first batch of core UI material types', () => {
    expect(
      [
        inputManifest,
        inputNumberManifest,
        selectManifest,
        radioGroupManifest,
        checkboxGroupManifest,
        switchManifest,
        tableManifest,
        paginationManifest,
      ].map((item) => item.type),
    ).toEqual([
      'Input',
      'InputNumber',
      'Select',
      'Radio',
      'Checkbox',
      'Switch',
      'Table',
      'Pagination',
    ])
  })

  it('keeps high-frequency fields configurable and bindable', () => {
    expect(inputManifest.props.find((item) => item.name === 'modelValue')).toMatchObject({
      label: '默认值',
      bindable: true,
    })
    expect(selectManifest.props.find((item) => item.name === 'options')).toMatchObject({
      type: 'array',
      label: '选项',
    })
    expect(tableManifest.props.find((item) => item.name === 'data')).toMatchObject({
      type: 'array',
      bindable: true,
    })
    expect(paginationManifest.props.find((item) => item.name === 'total')).toMatchObject({
      type: 'number',
      bindable: true,
    })
  })

  it('exposes practical configuration details for each component', () => {
    expect(inputManifest.props.map((item) => item.name)).toEqual([
      'modelValue',
      'placeholder',
      'type',
      'size',
      'clearable',
      'disabled',
    ])
    expect(inputNumberManifest.props.map((item) => item.name)).toEqual([
      'modelValue',
      'min',
      'max',
      'step',
      'size',
      'disabled',
    ])
    expect(selectManifest.props.map((item) => item.name)).toEqual([
      'modelValue',
      'placeholder',
      'options',
      'size',
      'clearable',
      'disabled',
    ])
    expect(radioGroupManifest.props.find((item) => item.name === 'buttonStyle')).toMatchObject({
      type: 'boolean',
      label: '按钮样式',
    })
    expect(checkboxGroupManifest.props.find((item) => item.name === 'buttonStyle')).toMatchObject({
      type: 'boolean',
      label: '按钮样式',
    })
    expect(tableManifest.props.map((item) => item.name)).toEqual([
      'columns',
      'data',
      'stripe',
      'border',
      'size',
    ])
    expect(paginationManifest.props.map((item) => item.name)).toEqual([
      'currentPage',
      'pageSize',
      'total',
      'background',
      'small',
    ])
  })

  it('sets practical default sizes for form and data components', () => {
    expect(inputManifest.defaultSize).toEqual({ width: 220, height: 34 })
    expect(selectManifest.defaultSize).toEqual({ width: 220, height: 34 })
    expect(tableManifest.defaultSize).toEqual({ width: 420, height: 180 })
    expect(paginationManifest.defaultSize).toEqual({ width: 420, height: 36 })
  })
})
