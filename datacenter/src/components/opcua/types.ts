export type OpcuaNodeGroup = {
  id: string
  parentId?: string | null
  name: string
  description?: string | null
  sortOrder?: number
}

export type OpcuaNode = {
  id: string
  groupId?: string | null
  name: string
  code: string
  nodeId: string
  browseName?: string | null
  displayName?: string | null
  dataType: string
  unit?: string | null
  samplingMs: number
  deadband?: number | null
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

export type OpcuaValidationIssue = {
  severity: string
  code: string
  groupId?: string | null
  nodeId?: string | null
  nodeName?: string
  message: string
}
