import type {
  ComputeOutputInput,
  ComputeUnitDetail,
  ComputeUnitSave,
} from '@/api/schemas/compute.schema'

export interface ComputeEditorTab {
  id: string
  name: string
  dirty: boolean
}

export interface ComputeParameterRow {
  uid: string
  name: string
  type: string
  required: boolean
  defaultValue: string
  description: string
}

export interface ComputeDatapointVariableRow {
  uid: string
  alias: string
  path: string
  datapointId?: string
  dataType?: string
}

export interface ComputeDependencyDraft {
  id: string
}

export interface ComputeDraft {
  id: string
  name: string
  folderId: string | null
  lang: string
  status: string
  code: string
  triggerType: string
  triggerConfig: Record<string, unknown>
  inputBindings: Record<string, unknown>
  outputs: ComputeOutputInput[]
  timeoutMs: number
  isEnabled: boolean
  dependencies: ComputeDependencyDraft[]
  parameterRows: ComputeParameterRow[]
  datapointVariableRows: ComputeDatapointVariableRow[]
  dirty: boolean
}

const defaultCode = (lang?: string) => {
  if (lang === 'python') {
    return 'def main(argv, dp, ctx):\n    return argv[0] if len(argv) > 0 else None\n'
  }
  return 'return argv[0] ?? null;\n'
}

export function toComputeDraft(unit: ComputeUnitDetail): ComputeDraft {
  const inputBindings = asRecord(unit.inputBindings)
  return {
    id: String(unit.id),
    name: unit.name || '未命名计算单元',
    folderId: unit.folderId ? String(unit.folderId) : null,
    lang: String(unit.lang || unit.language || 'javascript'),
    status: String(unit.status || (unit.isEnabled === false ? 'disabled' : 'enabled')),
    code:
      String(unit.code || unit.scriptCode || '') || defaultCode(String(unit.lang || unit.language)),
    triggerType: String(unit.triggerType || 'manual'),
    triggerConfig: asRecord(unit.triggerConfig),
    inputBindings,
    outputs: unit.outputs.map((output) => ({
      id: output.id,
      key: output.key,
      name: output.name,
      path: output.path,
      dataType: output.dataType,
      unit: output.unit,
      precisionNum: output.precisionNum,
      nullPolicy: output.nullPolicy,
      description: output.description,
    })),
    timeoutMs: Number(unit.timeoutMs || 3000),
    isEnabled: unit.isEnabled !== false,
    dependencies: toDependencyDrafts(unit.dependencies),
    parameterRows: toParameterRows(inputBindings),
    datapointVariableRows: toDatapointVariableRows(inputBindings),
    dirty: false,
  }
}

export function draftToSavePayload(draft: ComputeDraft): Partial<ComputeUnitSave> {
  return {
    name: draft.name,
    lang: draft.lang,
    code: draft.code,
    folderId: draft.folderId,
    triggerType: draft.triggerType,
    triggerConfig: draft.triggerConfig,
    inputBindings: inputRowsToBindings(draft.parameterRows, draft.datapointVariableRows),
    outputs: draft.outputs,
    timeoutMs: draft.timeoutMs,
    isEnabled: draft.isEnabled,
    dependencies: draft.dependencies.map((item) => ({ id: item.id })),
  }
}

function asRecord(value: unknown): Record<string, unknown> {
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    return { ...(value as Record<string, unknown>) }
  }
  return {}
}

function toDependencyDrafts(value: unknown): ComputeDependencyDraft[] {
  if (!Array.isArray(value)) return []
  return value
    .map((item) => {
      if (typeof item === 'string') return { id: item }
      if (item && typeof item === 'object' && 'id' in item) {
        return { id: String((item as { id: unknown }).id) }
      }
      return null
    })
    .filter((item): item is ComputeDependencyDraft => Boolean(item?.id))
}

function toParameterRows(bindings: Record<string, unknown>): ComputeParameterRow[] {
  const rawParameters = bindings.parameters
  if (!Array.isArray(rawParameters)) return []
  const rows: Array<ComputeParameterRow | null> = rawParameters.map((item) => {
    if (!item || typeof item !== 'object') return null
    const record = item as Record<string, unknown>
    return {
      uid: String(crypto.randomUUID()),
      name: String(record.name || ''),
      type: String(record.type || 'string'),
      required: record.required !== false,
      defaultValue:
        record.defaultValue === undefined || record.defaultValue === null
          ? ''
          : String(record.defaultValue),
      description: String(record.description || ''),
    }
  })
  return rows.filter((item): item is ComputeParameterRow => Boolean(item?.name))
}

function toDatapointVariableRows(bindings: Record<string, unknown>): ComputeDatapointVariableRow[] {
  const rawVariables = bindings.datapointVariables
  if (!Array.isArray(rawVariables)) return []
  const rows: Array<ComputeDatapointVariableRow | null> = rawVariables.map((item) => {
    if (!item || typeof item !== 'object') return null
    const record = item as Record<string, unknown>
    return {
      uid: String(crypto.randomUUID()),
      alias: String(record.alias || ''),
      path: String(record.path || ''),
      datapointId: record.datapointId ? String(record.datapointId) : undefined,
      dataType: record.dataType ? String(record.dataType) : undefined,
    }
  })
  return rows.filter((item): item is ComputeDatapointVariableRow =>
    Boolean(item?.alias && item?.path),
  )
}

function inputRowsToBindings(
  parameters: ComputeParameterRow[],
  datapointVariables: ComputeDatapointVariableRow[],
): Record<string, unknown> {
  return {
    parameters: parameters
      .filter((row) => row.name.trim())
      .map((row) => ({
        name: row.name.trim(),
        type: row.type,
        required: row.required,
        defaultValue: row.defaultValue,
        description: row.description.trim(),
      })),
    datapointVariables: datapointVariables
      .filter((row) => row.alias.trim() && row.path.trim())
      .map((row) => ({
        alias: row.alias.trim(),
        path: row.path.trim(),
        datapointId: row.datapointId,
        dataType: row.dataType,
      })),
  }
}
