import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

function readProtocolJson(protocol: string, fileName: string) {
  return JSON.parse(
    readFileSync(
      new URL(`../../contracts/collector-protocols/${protocol}/${fileName}`, import.meta.url),
      'utf8',
    ),
  ) as Record<string, any>
}

describe('collector protocol manifests', () => {
  it('keeps browsing exclusive to OPC UA while all implemented drivers support point reads', () => {
    const opcUa = readProtocolJson('opcua.standard', 'manifest.json')
    const modbus = readProtocolJson('modbus.tcp', 'manifest.json')
    const siemensS7 = readProtocolJson('siemens.s7-tcp', 'manifest.json')

    expect(opcUa.operations).toContain('device.browse')
    expect(modbus.operations).not.toContain('device.browse')
    expect(siemensS7.operations).not.toContain('device.browse')
    expect(opcUa.operations).toContain('point.read')
    expect(modbus.operations).toContain('point.read')
    expect(siemensS7.operations).toContain('point.read')
  })

  it('provides bilingual Modbus address labels and usable defaults', () => {
    const schema = readProtocolJson('modbus.tcp', 'address.schema.json')

    expect(schema.properties.station.title).toBe('从站地址（station）')
    expect(schema.properties.area['x-induforge-enum-labels'].holdingRegister).toBe(
      '保持寄存器（holdingRegister）',
    )
    expect(schema.properties.station.default).toBe(1)
    expect(schema.properties.area.default).toBe('holdingRegister')
    expect(schema.properties.address.default).toBe(0)
    expect(schema.properties.bitIndex.default).toBe(0)
  })

  it('provides a valid default S7 data block address', () => {
    const schema = readProtocolJson('siemens.s7-tcp', 'address.schema.json')

    expect(schema.properties.area.default).toBe('dataBlock')
    expect(schema.properties.dbNumber.default).toBe(1)
    expect(schema.properties.byteOffset.default).toBe(0)
    expect(schema.properties.bitOffset.default).toBe(0)
  })
})
