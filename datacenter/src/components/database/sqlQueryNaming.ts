const generatedQueryNamePattern = /^查询[ _](\d+)$/

// 自动名称取当前最小可用序号；兼容旧的“查询 1”，避免它与“查询_1”生成同一路径。
export const nextGeneratedQueryName = (existingNames: Iterable<string>): string => {
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
  return `查询_${sequence}`
}

export const generatedQueryCopyName = (name: string): string => `${name}_副本`
