export type KafkaSampleFieldType = 'string' | 'number' | 'boolean' | 'object' | 'array'

export type KafkaSampleFieldCandidate = {
  path: string
  name: string
  dataType: KafkaSampleFieldType
  sampleValue: unknown
  exists: boolean
}

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  Object.prototype.toString.call(value) === '[object Object]'

export const parseKafkaJsonValue = (value: unknown): unknown => {
  if (typeof value !== 'string') return value
  const trimmed = value.trim()
  if (!trimmed) return value
  try {
    return JSON.parse(trimmed)
  } catch {
    return value
  }
}

export const normalizeKafkaSampleRoot = (sample: unknown): unknown => {
  if (isPlainObject(sample) && Object.prototype.hasOwnProperty.call(sample, 'value')) {
    return parseKafkaJsonValue(sample.value)
  }
  return parseKafkaJsonValue(sample)
}

export const resolveKafkaSampleType = (value: unknown): KafkaSampleFieldType => {
  if (Array.isArray(value)) return 'array'
  if (value !== null && typeof value === 'object') return 'object'
  if (typeof value === 'number') return 'number'
  if (typeof value === 'boolean') return 'boolean'
  return 'string'
}

const defaultNameFromPath = (path: string) => {
  const last = path.split('.').filter(Boolean).at(-1)
  return last || path
}

export const inferKafkaSampleFields = (
  samples: unknown[],
  existingPaths: Iterable<string> = [],
): KafkaSampleFieldCandidate[] => {
  const existing = new Set(Array.from(existingPaths).map(String))
  const seen = new Map<string, KafkaSampleFieldCandidate>()

  const addCandidate = (path: string, value: unknown) => {
    if (!path || seen.has(path)) return
    seen.set(path, {
      path,
      name: defaultNameFromPath(path),
      dataType: resolveKafkaSampleType(value),
      sampleValue: value,
      exists: existing.has(path),
    })
  }

  const visit = (prefix: string, value: unknown) => {
    if (Array.isArray(value)) {
      if (value.length === 0) {
        addCandidate(prefix, value)
        return
      }
      value.forEach((item, index) => visit(prefix ? `${prefix}.${index}` : String(index), item))
      return
    }
    if (isPlainObject(value)) {
      const entries = Object.entries(value)
      if (entries.length === 0) {
        addCandidate(prefix, value)
        return
      }
      entries.forEach(([key, child]) => visit(prefix ? `${prefix}.${key}` : key, child))
      return
    }
    addCandidate(prefix, value)
  }

  samples.forEach((sample) => visit('', normalizeKafkaSampleRoot(sample)))
  return Array.from(seen.values())
}

export const normalizeKafkaSampleEditorText = (sample: unknown): string => {
  const root = normalizeKafkaSampleRoot(sample)
  if (typeof root === 'string') return root
  try {
    return JSON.stringify(root ?? {}, null, 2)
  } catch {
    return String(root ?? '')
  }
}
