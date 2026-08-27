export type SqlWorkbenchTabCloseMode = 'current' | 'right' | 'all'

// 不依赖浏览器原生 dblclick 事件；节点首次点击选中，限定时间内再次点击时稳定触发打开。
export const createSqlTreeActivationTracker = (thresholdMs = 700) => {
  let previousKey = ''
  let previousAt = 0

  return (key: string, now = Date.now()) => {
    const repeated = previousKey === key && now - previousAt >= 0 && now - previousAt <= thresholdMs
    if (repeated) {
      previousKey = ''
      previousAt = 0
      return true
    }
    previousKey = key
    previousAt = now
    return false
  }
}

export const sqlWorkbenchTablePreviewTabId = (connectionId: string, tableName: string) =>
  `table-data-${connectionId}-${tableName}`

export const sqlWorkbenchTabIdsToClose = (
  tabs: Array<{ id: string }>,
  targetId: string,
  mode: SqlWorkbenchTabCloseMode,
) => {
  if (mode === 'all') return tabs.map((tab) => tab.id)
  const targetIndex = tabs.findIndex((tab) => tab.id === targetId)
  if (targetIndex < 0) return []
  if (mode === 'right') return tabs.slice(targetIndex + 1).map((tab) => tab.id)
  return [targetId]
}
