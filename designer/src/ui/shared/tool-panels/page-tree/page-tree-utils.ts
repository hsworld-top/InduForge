/** localStorage 键前缀：各工程页面树排序 */
export const PAGE_TREE_ORDER_PREFIX = 'designer.pageTreeOrder'

/** 根级容器在排序表中的键 */
export const ROOT_CONTAINER_KEY = '__root__'
const ONLY_DOTS_RE = /^\.+$/
const INVALID_PAGE_NAME_CHAR_RE = /[/?#\\%]/

function hasControlChar(text: string): boolean {
  for (const char of text) {
    const code = char.codePointAt(0) ?? 0
    if (code <= 31 || code === 127) return true
  }
  return false
}

export function getPageTreeOrderStorageKey(projectId: string): string {
  return `${PAGE_TREE_ORDER_PREFIX}:${projectId || 'default'}`
}

export function validatePageName(value: string | undefined | null): {
  valid: boolean
  message: string
} {
  const name = String(value || '').trim()
  if (!name) {
    return { valid: false, message: '名称不能为空' }
  }
  if (ONLY_DOTS_RE.test(name)) {
    return { valid: false, message: '页面名称不能仅包含点号' }
  }
  if (INVALID_PAGE_NAME_CHAR_RE.test(name)) {
    return { valid: false, message: '页面名称不能包含 / ? # % \\' }
  }
  if (hasControlChar(name)) {
    return { valid: false, message: '页面名称不能包含控制字符' }
  }
  return { valid: true, message: '' }
}

export function moveIdBefore(ids: string[], sourceId: string, targetId: string): string[] {
  const nextIds = ids.filter((id) => id !== sourceId)
  const targetIndex = nextIds.indexOf(targetId)
  if (targetIndex === -1) {
    nextIds.push(sourceId)
    return nextIds
  }
  nextIds.splice(targetIndex, 0, sourceId)
  return nextIds
}
