import { describe, expect, it } from 'vitest'
import { ComputeUnitSaveSchema } from '@/api/schemas/compute.schema'

describe('compute schemas', () => {
  it('allows the create dialog to omit outputs so the backend can create the standard result output', () => {
    const parsed = ComputeUnitSaveSchema.parse({ name: '温度换算', lang: 'javascript' })
    expect(parsed.outputs).toEqual([])
  })
})
