export type CollectorPointFormContext = {
  driverId: string
  address: Record<string, unknown>
  dataType: string
  driverDataTypes: string[]
}

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
  if (context.driverId === 'modbus.tcp') return resolveModbusState(context)
  if (context.driverId === 'siemens.s7-tcp') return resolveSiemensS7State(context)
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
  if (!integerInRange(address.station, 0, 255)) error = '请填写 0 到 255 的从站地址'
  else if (!bitArea && !registerArea) error = '请选择 Modbus 数据区域'
  else if (!integerInRange(address.address, 0, 65535)) error = '请填写 0 到 65535 的协议地址'
  else if (showsBitIndex && !integerInRange(address.bitIndex, 0, 15)) {
    error = '寄存器 bool 变量必须填写位索引'
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
  if (!s7Areas.has(area)) error = '请选择 Siemens S7 存储区域'
  else if (showsDbNumber && !integerInRange(address.dbNumber, 1, 65535)) {
    error = '数据块区域必须填写 1 到 65535 的 DB 编号'
  } else if (!integerInRange(address.byteOffset, 0)) error = '请填写大于等于 0 的字节偏移'
  else if (showsBitOffset && !integerInRange(address.bitOffset, 0, 7)) {
    error = 'Siemens S7 bool 变量必须填写位偏移'
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
