import type { SourceOutputInput } from '@/api/schemas/source-output.schema'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

type QueryResult = {
  columns?: string[]
  rowCount?: number
}

export type QueryOutputPreviewItem = {
  key: string
  displayName: string
  path: string
  quality: 'preview'
  value: unknown
  error?: string
}

const datasetValue = (result: QueryResult, rows: Array<Record<string, unknown>>) => ({
  fields: result.columns || [],
  rows,
  rowCount: result.rowCount ?? rows.length,
})

const valueAtPath = (root: unknown, segments: Array<string | number>) => {
  let current = root
  for (const segment of segments) {
    if (current === null || current === undefined || typeof current !== 'object') {
      return { found: false, value: undefined }
    }
    if (!(segment in current)) return { found: false, value: undefined }
    current = (current as Record<string | number, unknown>)[segment]
  }
  return { found: true, value: current }
}

// SQL 标量输出与后端保持一致：只有结果恰好一行时才能提取单个字段。
export const buildQueryOutputPreviews = (
  outputs: Array<SourceOutputInput & { datapointPath?: string }>,
  result: QueryResult,
  rows: Array<Record<string, unknown>>,
): QueryOutputPreviewItem[] =>
  outputs.map((output, index) => {
    const base = {
      key: String(output.id || output.key || `output-${index}`),
      displayName: output.displayName || output.key || ui(`输出 ${index + 1}`, `Output ${index + 1}`),
      path: output.datapointPath || '',
      quality: 'preview' as const,
    }
    if (output.selector.kind === 'whole') {
      return { ...base, value: datasetValue(result, rows) }
    }
    if (rows.length !== 1) {
      return {
        ...base,
        value: null,
        error: ui(`需要查询恰好返回 1 行，当前返回 ${rows.length} 行`, `The query must return exactly one row; it returned ${rows.length}`),
      }
    }
    if (output.selector.kind === 'column') {
      if (!Object.prototype.hasOwnProperty.call(rows[0], output.selector.column)) {
        return {
          ...base,
          value: null,
          error: ui(`结果中没有字段“${output.selector.column}”`, `Column “${output.selector.column}” is not present in the result`),
        }
      }
      return { ...base, value: rows[0][output.selector.column] }
    }
    const selected = valueAtPath(rows[0], output.selector.segments)
    return selected.found
      ? { ...base, value: selected.value }
      : { ...base, value: null, error: ui('结果中不存在已选择的嵌套字段', 'The selected nested field is not present in the result') }
  })
