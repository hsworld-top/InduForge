import { z } from 'zod'

const ImpactItemSchema = z.object({
  type: z.string(),
  label: z.string().default(''),
  count: z.number().int().nonnegative(),
  examples: z
    .array(
      z.object({
        id: z.string(),
        name: z.string(),
        datapointPath: z.string().optional(),
      }),
    )
    .default([]),
})

export const SourceDeleteImpactSchema = z.object({
  scopeType: z.string(),
  scopeId: z.string(),
  name: z.string(),
  canDelete: z.boolean(),
  generatedDatapoints: z.object({
    count: z.number().int().nonnegative(),
    action: z.literal('mark_invalid'),
  }),
  ownedResources: z.array(ImpactItemSchema).default([]),
  blockingUsages: z.array(ImpactItemSchema).default([]),
})

export type SourceDeleteImpact = z.infer<typeof SourceDeleteImpactSchema>
