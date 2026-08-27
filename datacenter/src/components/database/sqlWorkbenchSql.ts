export const quoteSqlIdentifier = (dbType: string, identifier: string): string => {
  const normalizedType = String(dbType || '').toLowerCase()
  if (normalizedType === 'mysql' || normalizedType === 'tdengine') {
    return `\`${identifier.replaceAll('`', '``')}\``
  }
  if (normalizedType === 'sqlserver') {
    return `[${identifier.replaceAll(']', ']]')}]`
  }
  return `"${identifier.replaceAll('"', '""')}"`
}

export const buildTablePreviewSql = (dbType: string, tableName: string): string => {
  const quotedTable = quoteSqlIdentifier(dbType, tableName)
  if (String(dbType || '').toLowerCase() === 'sqlserver') {
    return `SELECT * FROM ${quotedTable} ORDER BY (SELECT NULL) OFFSET 0 ROWS FETCH NEXT 100 ROWS ONLY`
  }
  return `SELECT * FROM ${quotedTable} LIMIT 100`
}
