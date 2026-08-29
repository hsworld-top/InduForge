const generatedQueryNamePattern = /^(?:查询|Query)[ _](\d+)$/i

// 自动名称取当前最小可用序号；兼容旧的“查询 1”，避免它与“查询_1”生成同一路径。
export const nextGeneratedQueryName = (
  existingNames: Iterable<string>,
  prefix = '查询',
): string => {
  const occupied = new Set<number>()
  for (const name of existingNames) {
    const match = String(name || '')
      .trim()
      .match(generatedQueryNamePattern)
    if (!match) continue
    const sequence = Number(match[1])
    if (Number.isSafeInteger(sequence) && sequence > 0) occupied.add(sequence)
  }
  let sequence = 1
  while (occupied.has(sequence)) sequence += 1
  return `${prefix}_${sequence}`
}

export const generatedQueryCopyName = (name: string, suffix = '副本'): string => `${name}_${suffix}`
