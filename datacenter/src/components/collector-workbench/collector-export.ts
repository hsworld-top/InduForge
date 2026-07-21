export type CollectorExportPageParseResult = {
  pages: number[]
  error: string
}

export function parseCollectorExportPages(
  value: string,
  totalPages: number,
): CollectorExportPageParseResult {
  const normalized = value.trim().replaceAll('，', ',')
  if (!normalized) return { pages: [], error: '请输入要导出的页码' }
  const pages = new Set<number>()
  for (const part of normalized.split(',')) {
    const token = part.trim()
    if (!token) continue
    if (/^\d+$/.test(token)) {
      pages.add(Number(token))
      continue
    }
    const match = token.match(/^(\d+)\s*-\s*(\d+)$/)
    if (!match) return { pages: [], error: `页码格式无效：${token}` }
    const start = Number(match[1])
    const end = Number(match[2])
    if (start > end) return { pages: [], error: `页码范围无效：${token}` }
    for (let page = start; page <= end; page += 1) pages.add(page)
  }
  const result = [...pages].sort((left, right) => left - right)
  if (result.length === 0) return { pages: [], error: '请输入要导出的页码' }
  if (result.length > 200) return { pages: [], error: '单次最多指定 200 页' }
  const invalid = result.find((page) => page < 1 || page > totalPages)
  if (invalid) return { pages: [], error: `第 ${invalid} 页超出当前总页数` }
  return { pages: result, error: '' }
}

export function estimateCollectorExportRows(pages: number[], pageSize: number, total: number) {
  return pages.reduce((count, page) => {
    const start = (page - 1) * pageSize
    return count + Math.max(0, Math.min(pageSize, total - start))
  }, 0)
}
