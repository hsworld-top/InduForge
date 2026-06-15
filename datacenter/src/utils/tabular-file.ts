import * as XLSX from 'xlsx'

export type TabularCell = string | number | boolean | null | undefined

export type TabularSheet = {
  name: string
  headers: string[]
  rows: Record<string, TabularCell>[]
}

export async function readTabularFile(file: File): Promise<Record<string, string>[]> {
  const name = file.name.toLowerCase()
  if (name.endsWith('.xlsx') || name.endsWith('.xls')) {
    return readWorkbookFile(file)
  }
  const text = await file.text()
  return parseTabularText(text)
}

function readWorkbookFile(file: File): Promise<Record<string, string>[]> {
  return file.arrayBuffer().then((buffer) => {
    const workbook = XLSX.read(buffer, { type: 'array' })
    const sheetName = workbook.SheetNames[0]
    if (!sheetName) return []
    return XLSX.utils.sheet_to_json<Record<string, string>>(workbook.Sheets[sheetName], {
      defval: '',
      raw: false,
    })
  })
}

export function downloadCsv(
  filename: string,
  headers: string[],
  rows: Record<string, TabularCell>[],
) {
  const content = [
    headers.map(escapeCsvCell).join(','),
    ...rows.map((row) => headers.map((header) => escapeCsvCell(row[header])).join(',')),
  ].join('\r\n')
  downloadBlob(filename, new Blob([`\ufeff${content}`], { type: 'text/csv;charset=utf-8' }))
}

export function downloadXlsx(filename: string, sheets: TabularSheet[]) {
  const workbook = XLSX.utils.book_new()
  for (const sheet of sheets) {
    const worksheet = XLSX.utils.json_to_sheet(sheet.rows, { header: sheet.headers })
    XLSX.utils.book_append_sheet(workbook, worksheet, sheet.name.slice(0, 31) || 'Sheet1')
  }
  const buffer = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' })
  downloadBlob(
    filename,
    new Blob([buffer], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    }),
  )
}

export function normalizeHeaderRow(row: Record<string, string>, aliases: Record<string, string[]>) {
  const normalized: Record<string, string> = {}
  const entries = Object.entries(row).map(([header, value]) => [
    header.trim().toLowerCase(),
    String(value ?? '').trim(),
  ])
  for (const [field, names] of Object.entries(aliases)) {
    const lookupNames = new Set(names.map((name) => name.trim().toLowerCase()))
    const matched = entries.find(([header, value]) => lookupNames.has(header) && value !== '')
    normalized[field] = matched ? matched[1] : ''
  }
  return normalized
}

export function parseTabularText(text: string) {
  const records = parseDelimitedRows(text)
  const headers = records.shift()?.map((header) => header.trim()) || []
  if (headers.length === 0) return []
  return records
    .filter((record) => record.some((cell) => cell.trim() !== ''))
    .map((record) =>
      Object.fromEntries(headers.map((header, index) => [header, record[index]?.trim() || ''])),
    )
}

export function parseDelimitedRows(text: string) {
  const rows: string[][] = []
  let row: string[] = []
  let cell = ''
  let quoted = false

  // 这里用状态机解析 CSV，避免现场导出的变量名或描述里带逗号时被错误拆列。
  for (let index = 0; index < text.length; index += 1) {
    const char = text[index]
    const next = text[index + 1]
    if (quoted) {
      if (char === '"' && next === '"') {
        cell += '"'
        index += 1
      } else if (char === '"') {
        quoted = false
      } else {
        cell += char
      }
      continue
    }
    if (char === '"') {
      quoted = true
      continue
    }
    if (char === ',' || char === '\t') {
      row.push(cell)
      cell = ''
      continue
    }
    if (char === '\r' || char === '\n') {
      if (char === '\r' && next === '\n') index += 1
      row.push(cell)
      rows.push(row)
      row = []
      cell = ''
      continue
    }
    cell += char
  }
  row.push(cell)
  rows.push(row)
  return rows
}

function escapeCsvCell(value: TabularCell) {
  const text = value === null || value === undefined ? '' : String(value)
  return /[",\r\n]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text
}

function downloadBlob(filename: string, blob: Blob) {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  URL.revokeObjectURL(url)
}
