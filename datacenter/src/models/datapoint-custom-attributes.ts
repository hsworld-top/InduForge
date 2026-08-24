export interface DatapointAttributeRow {
  id: number
  key: string
  value: string
}

const attributeKeyPattern = /^[a-z][a-z0-9_.-]{0,63}$/

const reservedAttributeKeys = new Set([
  'alarm',
  'alarm_high',
  'alarm_low',
  'attribute_defaults',
  'attributes',
  'data_type',
  'default_value',
  'description',
  'history',
  'id',
  'max',
  'max_value',
  'min',
  'min_value',
  'name',
  'path',
  'permissions',
  'precision',
  'precision_num',
  'project_id',
  'quality',
  'runtime_permissions',
  'source',
  'source_config',
  'source_id',
  'source_type',
  'status',
  'tags',
  'timestamp',
  'unit',
  'value',
])

export function createDatapointAttributeRows(
  attributes: Record<string, string>,
): DatapointAttributeRow[] {
  return Object.entries(attributes)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, value], index) => ({ id: index + 1, key, value }))
}

export function validateDatapointAttributeRows(rows: DatapointAttributeRow[]): string | null {
  const seen = new Set<string>()
  for (const row of rows) {
    const key = row.key.trim()
    if (!key) return '属性 Key 不能为空'
    if (reservedAttributeKeys.has(key.toLowerCase())) {
      return `属性 Key“${key}”已被内置属性占用`
    }
    if (!attributeKeyPattern.test(key)) {
      return `属性 Key“${key}”格式无效，仅支持小写字母开头及小写字母、数字、点、短横线和下划线`
    }
    if (seen.has(key)) return `属性 Key“${key}”重复`
    seen.add(key)
  }
  return null
}

export function buildDatapointAttributeDefaults(
  rows: DatapointAttributeRow[],
): Record<string, string> {
  const result: Record<string, string> = {}
  for (const row of rows) result[row.key.trim()] = row.value
  return result
}
