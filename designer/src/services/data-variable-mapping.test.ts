import { describe, expect, it } from 'vitest'
import {
  buildProjectVariableFromDataPoint,
  findMappedProjectVariableName,
  normalizeProjectVariableName,
  resolveProjectVariableSnippet,
} from './data-variable-mapping'

describe('data variable mapping helpers', () => {
  it('将数据点名称规范化为唯一的 JS 工程变量名', () => {
    expect(normalizeProjectVariableName('meter-1.temperature', [])).toBe('meter_1_temperature')
    expect(normalizeProjectVariableName('123_value', [])).toBe('dp_123_value')
    expect(normalizeProjectVariableName('温度点', [])).toBe('datapoint')
    expect(normalizeProjectVariableName('meter-1.temperature', ['meter_1_temperature'])).toBe(
      'meter_1_temperature_2',
    )
  })

  it('为非标识符变量名生成 bracket 访问片段', () => {
    expect(resolveProjectVariableSnippet('temperature')).toBe('$global.temperature')
    expect(resolveProjectVariableSnippet('line-1')).toBe('$global["line-1"]')
  })

  it('基于数据点构建工程变量映射定义', () => {
    const result = buildProjectVariableFromDataPoint(
      {
        id: 'dp-1',
        name: 'Temperature',
        path: 'device.temperature',
        dataType: 'number',
        sourceType: 'mqtt.tag',
        sourceId: 'source-1',
        description: '入口温度',
      },
      { existingNames: [], groupId: 'g1' },
    )

    expect(result).toEqual({
      name: 'Temperature',
      definition: {
        type: 'number',
        default: 0,
        description: '入口温度',
        groupId: 'g1',
        mapped: true,
        source: {
          type: 'dataCenter',
          path: 'device.temperature',
          sourceType: 'mqtt.tag',
          sourceId: 'source-1',
          datapointId: 'dp-1',
        },
      },
    })
  })

  it('能根据 datapointId 或 path 复用已有映射', () => {
    const projectVariables = {
      byId: {
        mapped: true,
        source: { type: 'dataCenter', datapointId: 'dp-1', path: 'a.b' },
      },
      byPath: {
        mapped: true,
        source: { type: 'dataCenter', path: 'device.temperature' },
      },
    }

    expect(findMappedProjectVariableName({ id: 'dp-1', path: 'x.y' }, projectVariables)).toBe(
      'byId',
    )
    expect(
      findMappedProjectVariableName({ id: 'dp-2', path: 'device.temperature' }, projectVariables),
    ).toBe('byPath')
  })
})
