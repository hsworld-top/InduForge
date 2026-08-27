export type KafkaSampleFieldType = 'string' | 'float64' | 'bool' | 'object' | 'array'

export type KafkaSampleFieldCandidate = {
  path: string
  segments: Array<string | number>
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
  if (typeof value === 'number') return 'float64'
  if (typeof value === 'boolean') return 'bool'
  return 'string'
}

export const formatKafkaValuePath = (segments: Array<string | number>) =>
  segments
    .map((segment, index) =>
      typeof segment === 'number'
        ? `[${segment}]`
        : /^[A-Za-z_$][\w$]*$/.test(segment)
          ? `${index === 0 ? '' : '.'}${segment}`
          : `[${JSON.stringify(segment)}]`,
    )
    .join('')

const defaultNameFromSegments = (segments: Array<string | number>) =>
  String(segments.at(-1) ?? 'value')

export const inferKafkaSampleFields = (
  samples: unknown[],
  existingPaths: Iterable<string | Array<string | number>> = [],
): KafkaSampleFieldCandidate[] => {
  const existing = new Set(
    Array.from(existingPaths).map((path) =>
      JSON.stringify(Array.isArray(path) ? path : [String(path)]),
    ),
  )
  const seen = new Map<string, KafkaSampleFieldCandidate>()

  const addCandidate = (segments: Array<string | number>, value: unknown) => {
    const path = formatKafkaValuePath(segments)
    if (!path || seen.has(path)) return
    seen.set(path, {
      path,
      segments,
      name: defaultNameFromSegments(segments),
      dataType: resolveKafkaSampleType(value),
      sampleValue: value,
      exists: existing.has(JSON.stringify(segments)),
    })
  }

  const visit = (segments: Array<string | number>, value: unknown) => {
    if (Array.isArray(value)) {
      if (value.length === 0) {
        addCandidate(segments, value)
        return
      }
      value.forEach((item, index) => visit([...segments, index], item))
      return
    }
    if (isPlainObject(value)) {
      const entries = Object.entries(value)
      if (entries.length === 0) {
        addCandidate(segments, value)
        return
      }
      entries.forEach(([key, child]) => visit([...segments, key], child))
      return
    }
    addCandidate(segments, value)
  }

  samples.forEach((sample) => visit([], normalizeKafkaSampleRoot(sample)))
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
