// 参数提示只检查可执行 SQL 片段；字符串、标识符、注释和 PostgreSQL
// dollar-quoted 函数体中的 $1/?/@name 属于 SQL 内容，不能当成工作台绑定参数。
export const sqlCodeForParameterDetection = (sql: string) => {
  const chars = [...sql]
  const masked = [...chars]
  const blank = (start: number, end: number) => {
    for (let index = start; index < end; index += 1) {
      if (masked[index] !== '\n' && masked[index] !== '\r') masked[index] = ' '
    }
  }

  let index = 0
  while (index < chars.length) {
    const current = chars[index]
    const next = chars[index + 1]

    if (current === '-' && next === '-') {
      const start = index
      index += 2
      while (index < chars.length && chars[index] !== '\n') index += 1
      blank(start, index)
      continue
    }

    if (current === '/' && next === '*') {
      const start = index
      let depth = 1
      index += 2
      while (index < chars.length && depth > 0) {
        if (chars[index] === '/' && chars[index + 1] === '*') {
          depth += 1
          index += 2
        } else if (chars[index] === '*' && chars[index + 1] === '/') {
          depth -= 1
          index += 2
        } else {
          index += 1
        }
      }
      blank(start, index)
      continue
    }

    if (current === "'" || current === '"' || current === '`') {
      const quote = current
      const start = index
      index += 1
      while (index < chars.length) {
        if (chars[index] === '\\' && quote === "'") {
          index += 2
          continue
        }
        if (chars[index] === quote) {
          if (chars[index + 1] === quote) {
            index += 2
            continue
          }
          index += 1
          break
        }
        index += 1
      }
      blank(start, index)
      continue
    }

    if (current === '$') {
      const remainder = chars.slice(index).join('')
      const delimiter = remainder.match(/^\$(?:[A-Za-z_][A-Za-z0-9_]*)?\$/)?.[0]
      if (delimiter) {
        const start = index
        index += delimiter.length
        const closing = sql.indexOf(delimiter, index)
        index = closing < 0 ? chars.length : closing + delimiter.length
        blank(start, index)
        continue
      }
    }

    index += 1
  }

  return masked.join('')
}
