import { datacenterLocale } from '@/i18n/runtime'

export type CollectorPointFormContext = {
  driverId: string
  addressHelper?: string
  address: Record<string, unknown>
  dataType: string
  driverDataTypes: string[]
}

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

export type CollectorPointFormState = {
  visibleAddressFields: string[] | undefined
  requiredAddressFields: string[] | undefined
  allowedDataTypes: string[]
  dataType: string
  address: Record<string, unknown>
  error: string | null
}

const modbusBitAreas = new Set(['coil', 'discreteInput'])
const modbusRegisterAreas = new Set(['inputRegister', 'holdingRegister'])
const s7Areas = new Set(['input', 'output', 'marker', 'dataBlock', 'timer', 'counter'])
const s7WordAreas = new Set(['timer', 'counter'])

export function resolveCollectorPointFormState(
  context: CollectorPointFormContext,
): CollectorPointFormState {
  // 地址助手由 Manifest 声明，前端不再通过驱动 ID 猜测协议族。
  if (context.addressHelper === 'modbus') return resolveModbusState(context)
  if (
    context.addressHelper === 'siemens' &&
    ('area' in context.address || 'byteOffset' in context.address)
  )
    return resolveSiemensS7State(context)
  return {
    visibleAddressFields: undefined,
    requiredAddressFields: undefined,
    allowedDataTypes: context.driverDataTypes,
    dataType: context.dataType,
    address: { ...context.address },
    error: null,
  }
}

function resolveModbusState(context: CollectorPointFormContext): CollectorPointFormState {
  const area = stringValue(context.address.area)
  const bitArea = modbusBitAreas.has(area)
  const registerArea = modbusRegisterAreas.has(area)
  const allowedDataTypes = bitArea
    ? context.driverDataTypes.filter((item) => item === 'bool')
    : context.driverDataTypes
  const dataType = allowedDataTypes.includes(context.dataType)
    ? context.dataType
    : allowedDataTypes[0] || context.dataType
  const showsBitIndex = registerArea && dataType === 'bool'
  const address = pickAddressFields(
    context.address,
    showsBitIndex ? ['station', 'area', 'address', 'bitIndex'] : ['station', 'area', 'address'],
  )

  let error: string | null = null
  if (!integerInRange(address.station, 0, 255)) error = ui('请填写 0 到 255 的从站地址', 'Enter a station address from 0 to 255')
  else if (!bitArea && !registerArea) error = ui('请选择 Modbus 数据区域', 'Select a Modbus data area')
  else if (!integerInRange(address.address, 0, 65535)) error = ui('请填写 0 到 65535 的协议地址', 'Enter a protocol address from 0 to 65535')
  else if (showsBitIndex && !integerInRange(address.bitIndex, 0, 15)) {
    error = ui('寄存器 bool 变量必须填写位索引', 'Boolean register points require a bit index')
  }

  return {
    visibleAddressFields: showsBitIndex
      ? ['station', 'area', 'address', 'bitIndex']
      : ['station', 'area', 'address'],
    requiredAddressFields: showsBitIndex
      ? ['station', 'area', 'address', 'bitIndex']
      : ['station', 'area', 'address'],
    allowedDataTypes,
    dataType,
    address,
    error,
  }
}

function resolveSiemensS7State(context: CollectorPointFormContext): CollectorPointFormState {
  const area = stringValue(context.address.area)
  const wordArea = s7WordAreas.has(area)
  const allowedDataTypes = wordArea
    ? context.driverDataTypes.filter((item) => item === 'int16' || item === 'uint16')
    : context.driverDataTypes
  const dataType = allowedDataTypes.includes(context.dataType)
    ? context.dataType
    : allowedDataTypes[0] || context.dataType
  const showsDbNumber = area === 'dataBlock'
  const showsBitOffset = dataType === 'bool' && !wordArea
  const visibleAddressFields = [
    'area',
    ...(showsDbNumber ? ['dbNumber'] : []),
    'byteOffset',
    ...(showsBitOffset ? ['bitOffset'] : []),
  ]
  const address = pickAddressFields(context.address, visibleAddressFields)

  let error: string | null = null
  if (!s7Areas.has(area)) error = ui('请选择 Siemens S7 存储区域', 'Select a Siemens S7 memory area')
  else if (showsDbNumber && !integerInRange(address.dbNumber, 1, 65535)) {
    error = ui('数据块区域必须填写 1 到 65535 的 DB 编号', 'Data block areas require a DB number from 1 to 65535')
  } else if (!integerInRange(address.byteOffset, 0)) error = ui('请填写大于等于 0 的字节偏移', 'Enter a byte offset greater than or equal to 0')
  else if (showsBitOffset && !integerInRange(address.bitOffset, 0, 7)) {
    error = ui('Siemens S7 bool 变量必须填写位偏移', 'Siemens S7 boolean points require a bit offset')
  }

  return {
    visibleAddressFields,
    requiredAddressFields: visibleAddressFields,
    allowedDataTypes,
    dataType,
    address,
    error,
  }
}

function pickAddressFields(address: Record<string, unknown>, fields: string[]) {
  return Object.fromEntries(
    fields.filter((field) => field in address).map((field) => [field, address[field]]),
  )
}

function stringValue(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function integerInRange(value: unknown, minimum: number, maximum = Number.MAX_SAFE_INTEGER) {
  return (
    typeof value === 'number' && Number.isInteger(value) && value >= minimum && value <= maximum
  )
}
