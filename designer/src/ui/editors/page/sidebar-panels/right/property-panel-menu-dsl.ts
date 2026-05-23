/**
 * PropertyPanel 菜单 DSL 纯函数。
 * 只负责字符串清洗、DSL 配置提取与菜单项规范化，不处理节点更新副作用。
 */

type LooseRecord = Record<string, unknown>
const MENU_DSL_MARKER_RE = /this\s*\.\s*menu\s*\(/
const MENU_DSL_SANITIZE_COMMA_RE = /[，﹐､]/g
const MENU_DSL_SANITIZE_SEMICOLON_RE = /[；﹔]/g
const MENU_DSL_SANITIZE_COLON_RE = /[：﹕]/g

/**
 * 规范化菜单项
 * @param {unknown} items - 菜单项
 * @returns {LooseRecord[]}
 */
export function normalizeMenuItems(items: unknown): LooseRecord[] {
  if (!Array.isArray(items)) return []
  return items
    .map((item): LooseRecord | null => {
      if (!item) return null
      if (typeof item === 'string') {
        return { label: item, index: item }
      }
      if (typeof item !== 'object') return null
      const record = item as LooseRecord
      const label = record.label ?? record.title ?? record.name ?? ''
      const index =
        record.index ?? record.command ?? record.key ?? (label ? String(label) : undefined)
      return { ...record, label, index }
    })
    .filter((item): item is LooseRecord => item !== null)
}

/**
 * 提取 Menu DSL 配置对象
 * @param {string} content - DSL 内容
 * @returns {LooseRecord | null}
 */
export function extractMenuDslConfig(content: string): LooseRecord | null {
  const text = String(content || '')
  if (!text.trim()) return null
  const match = MENU_DSL_MARKER_RE.exec(text)
  if (!match) return null
  let index = match.index + match[0].length
  while (index < text.length && text[index] !== '{') index += 1
  if (index >= text.length) return null

  let depth = 0
  let inString = false
  let quote = ''
  const start = index
  for (; index < text.length; index += 1) {
    const char = text[index]
    if (inString) {
      if (char === '\\' && index + 1 < text.length) {
        index += 1
        continue
      }
      if (char === quote) {
        inString = false
        quote = ''
      }
      continue
    }
    if (char === '"' || char === "'" || char === '`') {
      inString = true
      quote = char
      continue
    }
    if (char === '{') depth += 1
    if (char === '}') {
      depth -= 1
      if (depth === 0) {
        const body = text.slice(start, index + 1)
        try {
          return new Function(`return (${body});`)() as LooseRecord
        } catch {
          return null
        }
      }
    }
  }
  return null
}

/**
 * 捕获 menu(...) DSL 运行结果
 * @param {string} content - DSL 内容
 * @returns {LooseRecord | null}
 */
export function captureMenuDslConfig(content: string): LooseRecord | null {
  const text = String(content || '')
  if (!text.trim()) return null
  let captured: LooseRecord | null = null
  try {
    const runner = new Function(
      'context',
      `"use strict";\nreturn (function() {\n${text}\n}).call(context);`,
    )
    runner({
      menu: (config: unknown) => {
        if (config && typeof config === 'object' && !Array.isArray(config)) {
          captured = config as LooseRecord
        }
      },
    })
  } catch {
    return null
  }
  return captured
}

/**
 * 清理 DSL 中的常见全角符号
 * @param {string} content - DSL 内容
 * @returns {string}
 */
export function sanitizeDslContent(content: string): string {
  return String(content || '')
    .replace(MENU_DSL_SANITIZE_COMMA_RE, ',')
    .replace(MENU_DSL_SANITIZE_SEMICOLON_RE, ';')
    .replace(MENU_DSL_SANITIZE_COLON_RE, ':')
}

/**
 * 生成 Menu DSL 脚本
 * @param {string} content - 原始配置
 * @param {string} methodName - DSL 方法名
 * @returns {string}
 */
export function buildMenuDslContent(content: string, methodName: string): string {
  const text = String(content || '').trim()
  if (!text) return ''
  if (MENU_DSL_MARKER_RE.test(text)) return text
  if (text.startsWith('{') && text.endsWith('}')) {
    return `this.${methodName}(${text});`
  }
  return `this.${methodName}({\n${content}\n});`
}

/**
 * 尝试解析 Menu 配置对象
 * @param {string} content - 配置内容
 * @returns {LooseRecord | null}
 */
export function resolveMenuConfigFromContent(content: string): LooseRecord | null {
  const text = sanitizeDslContent(content).trim()
  if (!text) return null
  const direct = extractMenuDslConfig(text) || captureMenuDslConfig(text)
  if (direct) return direct
  if (text.startsWith('{') && text.endsWith('}')) {
    try {
      return new Function(`return (${text});`)() as LooseRecord
    } catch {
      return null
    }
  }
  try {
    return new Function(`return ({${text}});`)() as LooseRecord
  } catch {
    return null
  }
}
