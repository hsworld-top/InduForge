import { describe, expect, it } from 'vitest'
import { buildQueryOutputPreviews } from '@/components/database/sourceOutputPreview'

describe('query output preview', () => {
  const result = { columns: ['id', 'payload'], rowCount: 1 }
  const rows = [{ id: 7, payload: { temperature: 23.5 } }]

  it('预览完整数据集和高级字段提取', () => {
    const previews = buildQueryOutputPreviews(
      [
        {
          key: 'result',
          displayName: '完整结果',
          selector: { kind: 'whole' },
          dataType: 'object',
          sortOrder: 0,
        },
        {
          key: 'temperature',
          displayName: '温度',
          selector: { kind: 'path', segments: ['payload', 'temperature'] },
          dataType: 'float64',
          sortOrder: 1,
        },
      ],
      result,
      rows,
    )

    expect(previews[0]?.value).toEqual({
      fields: ['id', 'payload'],
      rows,
      rowCount: 1,
    })
    expect(previews[1]?.value).toBe(23.5)
  })

  it('多行结果不能预览标量字段', () => {
    const previews = buildQueryOutputPreviews(
      [
        {
          key: 'id',
          displayName: 'ID',
          selector: { kind: 'column', column: 'id' },
          dataType: 'int64',
          sortOrder: 0,
        },
      ],
      { ...result, rowCount: 2 },
      [{ id: 1 }, { id: 2 }],
    )

    expect(previews[0]?.error).toContain('当前返回 2 行')
  })
})
