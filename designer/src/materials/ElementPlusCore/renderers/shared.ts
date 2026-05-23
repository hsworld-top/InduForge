export interface OptionItem {
  label: string
  value: string | number | boolean
  disabled?: boolean
}

export interface TableColumnItem {
  prop: string
  label: string
  width?: string | number
  align?: string
}

export function normalizeOptions(value: unknown): OptionItem[] {
  if (!Array.isArray(value)) return []
  return value
    .map((item, index) => {
      if (item && typeof item === 'object') {
        const record = item as Record<string, unknown>
        const rawValue = record.value ?? record.label ?? `option${index + 1}`
        return {
          label: String(record.label ?? rawValue),
          value: normalizeOptionValue(rawValue),
          disabled: record.disabled === true,
        }
      }
      return {
        label: String(item ?? ''),
        value: normalizeOptionValue(item ?? `option${index + 1}`),
      }
    })
    .filter((item) => item.label)
}

export function normalizeTableColumns(value: unknown): TableColumnItem[] {
  if (!Array.isArray(value)) return []
  const columns: TableColumnItem[] = []
  value.forEach((item) => {
    if (!item || typeof item !== 'object') return
    const record = item as Record<string, unknown>
    const prop = String(record.prop || '').trim()
    const label = String(record.label || prop).trim()
    if (!prop || !label) return
    const column: TableColumnItem = { prop, label }
    if (typeof record.width === 'number' || typeof record.width === 'string') {
      column.width = record.width
    }
    if (typeof record.align === 'string') {
      column.align = record.align
    }
    columns.push(column)
  })
  return columns
}

export function normalizeTableData(value: unknown): Array<Record<string, unknown>> {
  if (!Array.isArray(value)) return []
  return value.filter(
    (item): item is Record<string, unknown> => Boolean(item) && typeof item === 'object',
  )
}

function normalizeOptionValue(value: unknown): string | number | boolean {
  if (typeof value === 'number' || typeof value === 'boolean') return value
  return String(value ?? '')
}
