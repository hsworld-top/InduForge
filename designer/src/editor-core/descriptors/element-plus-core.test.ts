import { describe, expect, it } from 'vitest'
import {
  checkboxDescriptor,
  inputDescriptor,
  inputNumberDescriptor,
  paginationDescriptor,
  radioDescriptor,
  selectDescriptor,
  switchDescriptor,
  tableDescriptor,
} from './element-plus-core'

describe('element-plus core descriptors', () => {
  it('defines default render contracts for direct Element Plus components', () => {
    expect(inputDescriptor.renderTag).toBe('el-input')
    expect(inputDescriptor.defaultSize).toEqual({ width: 220, height: 34 })
  })

  it('uses custom renderers for option and table driven components', () => {
    expect(inputNumberDescriptor.renderTag).toBe('div')
    expect(inputNumberDescriptor.defaultSize).toEqual({ width: 160, height: 34 })
    expect(inputNumberDescriptor.customRenderer).toBeTruthy()
    expect(selectDescriptor.customRenderer).toBeTruthy()
    expect(radioDescriptor.customRenderer).toBeTruthy()
    expect(checkboxDescriptor.customRenderer).toBeTruthy()
    expect(switchDescriptor.renderTag).toBe('div')
    expect(switchDescriptor.defaultSize).toEqual({ width: 80, height: 32 })
    expect(switchDescriptor.customRenderer).toBeTruthy()
    expect(tableDescriptor.customRenderer).toBeTruthy()
    expect(paginationDescriptor.customRenderer).toBeTruthy()
  })

  it('normalizes pagination helper props before rendering', () => {
    expect(
      paginationDescriptor.propsFilter?.({
        currentPage: 2,
        pageSize: 20,
        total: 88,
        background: true,
      }),
    ).toEqual({
      currentPage: 2,
      pageSize: 20,
      total: 88,
      background: true,
    })
  })
})
