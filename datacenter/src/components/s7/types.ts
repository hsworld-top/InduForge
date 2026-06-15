export type S7Profile = {
  id?: string
  projectId?: string
  connectionId?: string
  plcFamily: string
  communicationMode: string
  host: string
  port: number
  rack: number
  slot: number
  localTsap?: string | null
  remoteTsap?: string | null
  pollIntervalMs: number
  connectTimeoutMs: number
  readTimeoutMs: number
  pduSize?: number | null
  maxReadBytes?: number | null
  maxGapBytes: number
  maxConcurrentReads: number
  byteOrder: string
  wordOrder: string
  optimizedBlockAccess: boolean
  allowAbsoluteAddress: boolean
  allowSymbolAddress: boolean
  supportedAreas: string[]
  options?: Record<string, unknown>
  configured?: boolean
}

export type S7VariableGroup = {
  id: string
  parentId?: string | null
  name: string
  code: string
  description?: string | null
  sortOrder?: number
}

export type S7Variable = {
  id: string
  groupId?: string | null
  name: string
  code: string
  description?: string | null
  area: string
  dbNumber?: number | null
  byteOffset: number
  bitOffset?: number | null
  addressText: string
  normalizedAddress: string
  addressType: string
  readLength: number
  dataType: string
  length?: number | null
  arrayLength?: number | null
  byteOrder: string
  wordOrder: string
  scale: number
  offset: number
  unit?: string | null
  pollIntervalMs: number
  accessLevel: string
  lastValue?: unknown
  quality?: string
  lastUpdatedAt?: string | null
  sortOrder?: number
  status: string
  datapointId?: string | null
  datapointPath?: string | null
  datapointStatus?: string | null
}

export type S7ValidationIssue = {
  severity: string
  code: string
  groupId?: string | null
  variableId?: string | null
  variableName?: string
  message: string
}

export type S7ReadPlan = {
  id: string
  connectionId: string
  area: string
  dbNumber?: number | null
  startByte: number
  endByte: number
  readLength: number
  pollIntervalMs: number
  variableIds: string[]
  variableCount: number
  maxGapBytes: number
  readMode: string
  displayRange?: string
  readsPerSecond?: number
}

export type S7ReadPlanEstimate = {
  variableCount: number
  blockCount: number
  totalReadBytes: number
  readsPerSecond: number
  estimatedCycleMs: number
  largestBlockBytes: number
  fragmentedGroupCount: number
  plans: S7ReadPlan[]
  diagnostics: string[]
}

export type S7ReadValue = {
  variableId: string
  address: string
  rawValue: unknown
  value: unknown
  dataType: string
  quality: string
  timestamp: string
  error?: string | null
}
