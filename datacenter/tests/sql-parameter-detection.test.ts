import { describe, expect, it } from 'vitest'
import { sqlCodeForParameterDetection } from '../src/utils/sqlParameterDetection'

describe('sql parameter detection', () => {
  it('保留真正的 PostgreSQL 参数并屏蔽函数体位置参数', () => {
    const sql = `
      SELECT $1;
      CREATE FUNCTION double_value(integer) RETURNS integer
      LANGUAGE sql AS $body$ SELECT $1 * 2 $body$;
    `
    const code = sqlCodeForParameterDetection(sql)
    expect([...code.matchAll(/\$(\d+)/g)].map((match) => Number(match[1]))).toEqual([1])
  })

  it('屏蔽字符串、标识符、行注释、嵌套块注释和匿名 dollar quote', () => {
    const sql = `
      SELECT '?', "@ignored", \`$9\`, $2;
      -- $3 ? @line
      /* $4 /* ? */ @block */
      DO $$ BEGIN RAISE NOTICE '$5 ? @body'; END $$;
    `
    const code = sqlCodeForParameterDetection(sql)
    expect([...code.matchAll(/\$(\d+)/g)].map((match) => Number(match[1]))).toEqual([2])
    expect(code.match(/\?/g)).toBeNull()
    expect(code.match(/@[A-Za-z_][A-Za-z0-9_]*/g)).toBeNull()
  })
})
