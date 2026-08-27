import { describe, expect, it } from 'vitest'
import { resolveCollectorPointFormState } from '@/components/collector-workbench/collector-point-form-rules'

const dataTypes = ['bool', 'int16', 'uint16', 'int32', 'float32', 'string']

describe('collector point form rules', () => {
  it('restricts Modbus bit areas to bool and removes register-only fields', () => {
    const state = resolveCollectorPointFormState({
      driverId: 'modbus.tcp',
      addressHelper: 'modbus',
      address: { station: 1, area: 'coil', address: 10, bitIndex: 3 },
      dataType: 'int16',
      driverDataTypes: dataTypes,
    })

    expect(state.dataType).toBe('bool')
    expect(state.allowedDataTypes).toEqual(['bool'])
    expect(state.visibleAddressFields).toEqual(['station', 'area', 'address'])
    expect(state.address).toEqual({ station: 1, area: 'coil', address: 10 })
    expect(state.error).toBeNull()
  })

  it('requires a bit index for Modbus register bool values', () => {
    const state = resolveCollectorPointFormState({
      driverId: 'modbus.tcp',
      addressHelper: 'modbus',
      address: { station: 1, area: 'holdingRegister', address: 10 },
      dataType: 'bool',
      driverDataTypes: dataTypes,
    })

    expect(state.visibleAddressFields).toContain('bitIndex')
    expect(state.error).toBe('寄存器 bool 变量必须填写位索引')
  })

  it('removes the Modbus bit index from numeric register values', () => {
    const state = resolveCollectorPointFormState({
      driverId: 'modbus.tcp',
      addressHelper: 'modbus',
      address: { station: 1, area: 'inputRegister', address: 10, bitIndex: 2 },
      dataType: 'int16',
      driverDataTypes: dataTypes,
    })

    expect(state.visibleAddressFields).not.toContain('bitIndex')
    expect(state.address).toEqual({ station: 1, area: 'inputRegister', address: 10 })
  })

  it('shows S7 DB number only for data block addresses', () => {
    const dataBlockState = resolveCollectorPointFormState({
      driverId: 'siemens.s7-tcp',
      addressHelper: 'siemens',
      address: { area: 'dataBlock', dbNumber: 1, byteOffset: 4 },
      dataType: 'int16',
      driverDataTypes: dataTypes,
    })
    const markerState = resolveCollectorPointFormState({
      driverId: 'siemens.s7-tcp',
      addressHelper: 'siemens',
      address: { area: 'marker', dbNumber: 1, byteOffset: 4 },
      dataType: 'int16',
      driverDataTypes: dataTypes,
    })

    expect(dataBlockState.visibleAddressFields).toContain('dbNumber')
    expect(markerState.visibleAddressFields).not.toContain('dbNumber')
    expect(markerState.address).toEqual({ area: 'marker', byteOffset: 4 })
  })

  it('shows S7 bit offset only for bool values in bit-addressable areas', () => {
    const boolState = resolveCollectorPointFormState({
      driverId: 'siemens.s7-tcp',
      addressHelper: 'siemens',
      address: { area: 'input', byteOffset: 4 },
      dataType: 'bool',
      driverDataTypes: dataTypes,
    })
    const numericState = resolveCollectorPointFormState({
      driverId: 'siemens.s7-tcp',
      addressHelper: 'siemens',
      address: { area: 'input', byteOffset: 4, bitOffset: 2 },
      dataType: 'int16',
      driverDataTypes: dataTypes,
    })

    expect(boolState.visibleAddressFields).toContain('bitOffset')
    expect(boolState.error).toBe('Siemens S7 bool 变量必须填写位偏移')
    expect(numericState.visibleAddressFields).not.toContain('bitOffset')
    expect(numericState.address).toEqual({ area: 'input', byteOffset: 4 })
  })

  it('restricts S7 timer and counter values to 16-bit integers', () => {
    const state = resolveCollectorPointFormState({
      driverId: 'siemens.s7-tcp',
      addressHelper: 'siemens',
      address: { area: 'timer', byteOffset: 2, bitOffset: 1 },
      dataType: 'float32',
      driverDataTypes: dataTypes,
    })

    expect(state.dataType).toBe('int16')
    expect(state.allowedDataTypes).toEqual(['int16', 'uint16'])
    expect(state.visibleAddressFields).toEqual(['area', 'byteOffset'])
    expect(state.address).toEqual({ area: 'timer', byteOffset: 2 })
  })

  it('leaves unknown drivers unchanged', () => {
    const address = { custom: 'value' }
    const state = resolveCollectorPointFormState({
      driverId: 'custom.driver',
      address,
      dataType: 'string',
      driverDataTypes: dataTypes,
    })

    expect(state.dataType).toBe('string')
    expect(state.allowedDataTypes).toEqual(dataTypes)
    expect(state.visibleAddressFields).toBeUndefined()
    expect(state.address).toEqual(address)
    expect(state.error).toBeNull()
  })
})
