import { z } from 'zod'
import { IdSchema } from './common.schema'

export const CanonicalDataPointTypeSchema = z.enum([
  'bool',
  'int8',
  'uint8',
  'int16',
  'uint16',
  'int32',
  'uint32',
  'int64',
  'uint64',
  'float32',
  'float64',
  'decimal',
  'string',
  'bytes',
  'datetime',
  'object',
  'array',
])

export const SourceOutputPathSegmentSchema = z.union([z.string(), z.number().int().nonnegative()])

export const SourceOutputSelectorSchema = z.discriminatedUnion('kind', [
  z.object({ kind: z.literal('whole') }),
  z.object({ kind: z.literal('column'), column: z.string().min(1) }),
  z.object({ kind: z.literal('path'), segments: z.array(SourceOutputPathSegmentSchema).min(1) }),
])

export const SourceOutputInputSchema = z.object({
  id: IdSchema.optional(),
  key: z.string().min(1).max(100),
  displayName: z.string().min(1).max(100),
  selector: SourceOutputSelectorSchema,
  dataType: CanonicalDataPointTypeSchema,
  unit: z.string().optional().nullable(),
  precisionNum: z.number().int().nonnegative().optional().nullable(),
  sortOrder: z.number().int().nonnegative().default(0),
})

export const SourceOutputSchema = SourceOutputInputSchema.extend({
  id: IdSchema,
  datapointId: IdSchema,
  datapointPath: z.string().min(1),
})

export type CanonicalDataPointType = z.infer<typeof CanonicalDataPointTypeSchema>
export type SourceOutputSelector = z.infer<typeof SourceOutputSelectorSchema>
export type SourceOutputInput = z.infer<typeof SourceOutputInputSchema>
export type SourceOutput = z.infer<typeof SourceOutputSchema>

export const createWholeSourceOutput = (
  key = 'result',
  displayName = '完整结果',
  dataType: CanonicalDataPointType = 'object',
): SourceOutputInput => ({
  key,
  displayName,
  selector: { kind: 'whole' },
  dataType,
  unit: null,
  precisionNum: null,
  sortOrder: 0,
})
