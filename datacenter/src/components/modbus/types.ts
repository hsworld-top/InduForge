export type ModbusRegisterGroup = {
  id: string
  parentId?: string | null
  name: string
  description?: string | null
  sortOrder?: number
}

export type ModbusRegister = {
  id: string
  groupId?: string | null
  name: string
  code: string
  unitId: number
  area: string
  address: number
  addressBase: string
  protocolAddress: number
  quantity: number
  dataType: string
  byteOrder: string
  wordOrder: string
  bitIndex?: number | null
  scale: number
  offset: number
  unit?: string | null
  pollIntervalMs: number
  timeoutMs?: number | null
  retryCount?: number | null
  accessLevel: string
  description?: string | null
  sortOrder?: number
  status: string
  datapointId?: string | null
  datapointPath?: string | null
  datapointStatus?: string | null
  lastValue?: unknown
  quality?: string
  lastUpdatedAt?: string | null
}

export type ModbusValidationIssue = {
  severity: string
  code: string
  groupId?: string | null
  registerId?: string | null
  registerName?: string
  message: string
}

export type ModbusReadPlan = {
  id: string
  connectionId: string
  unitId: number
  area: string
  startAddress: number
  endAddress: number
  protocolStart: number
  quantity: number
  pollIntervalMs: number
  registerIds: string[]
  registerCount: number
  readsPerSecond: number
  displayRange: string
  displayArea: string
  displayCycle: string
  estimatedPressure: string
}

export type ModbusReadPlanEstimate = {
  registerCount: number
  unitCount: number
  readCount: number
  readsPerSecond: number
  plans: ModbusReadPlan[]
  diagnostics: string[]
}

export type ModbusReadValue = {
  registerId: string
  slaveId: number
  area: string
  address: number
  rawValue: number[]
  value: unknown
  dataType: string
  timestamp: string
  error?: string | null
}
