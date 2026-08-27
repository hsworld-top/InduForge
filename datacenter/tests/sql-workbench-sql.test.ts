import { describe, expect, it } from 'vitest'
import {
  buildTablePreviewSql,
  quoteSqlIdentifier,
} from '../src/components/database/sqlWorkbenchSql'

describe('SQL workbench dialect SQL', () => {
  it('TDengine 和 MySQL 使用反引号，PostgreSQL 使用双引号', () => {
    expect(buildTablePreviewSql('tdengine', 'device_a')).toBe('SELECT * FROM `device_a` LIMIT 100')
    expect(buildTablePreviewSql('mysql', 'device_a')).toBe('SELECT * FROM `device_a` LIMIT 100')
    expect(buildTablePreviewSql('postgresql', 'device_a')).toBe(
      'SELECT * FROM "device_a" LIMIT 100',
    )
  })

  it('SQL Server 使用方括号分页并正确转义标识符', () => {
    expect(buildTablePreviewSql('sqlserver', 'event]log')).toBe(
      'SELECT * FROM [event]]log] ORDER BY (SELECT NULL) OFFSET 0 ROWS FETCH NEXT 100 ROWS ONLY',
    )
    expect(quoteSqlIdentifier('tdengine', 'event`log')).toBe('`event``log`')
  })
})
