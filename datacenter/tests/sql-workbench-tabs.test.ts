import { describe, expect, it } from 'vitest'
import {
  createSqlTreeActivationTracker,
  sqlWorkbenchTablePreviewTabId,
  sqlWorkbenchTabIdsToClose,
} from '../src/components/database/sqlWorkbenchTabs'

const tabs = [{ id: 'a' }, { id: 'b' }, { id: 'c' }, { id: 'd' }]

describe('SQL workbench tab close targets', () => {
  it('关闭当前只选择右键标签', () => {
    expect(sqlWorkbenchTabIdsToClose(tabs, 'b', 'current')).toEqual(['b'])
  })

  it('关闭右侧保持当前及左侧标签', () => {
    expect(sqlWorkbenchTabIdsToClose(tabs, 'b', 'right')).toEqual(['c', 'd'])
    expect(sqlWorkbenchTabIdsToClose(tabs, 'd', 'right')).toEqual([])
  })

  it('关闭全部选择所有标签且不存在的目标不会误关当前', () => {
    expect(sqlWorkbenchTabIdsToClose(tabs, 'missing', 'all')).toEqual(['a', 'b', 'c', 'd'])
    expect(sqlWorkbenchTabIdsToClose(tabs, 'missing', 'current')).toEqual([])
  })
})

describe('SQL workbench tree activation', () => {
  it('同一节点连续两次点击只打开一次', () => {
    const register = createSqlTreeActivationTracker(700)
    expect(register('table:a', 1000)).toBe(false)
    expect(register('table:a', 1500)).toBe(true)
    expect(register('table:a', 1600)).toBe(false)
  })

  it('超时或切换节点时重新等待第二次点击', () => {
    const register = createSqlTreeActivationTracker(700)
    expect(register('query:a', 1000)).toBe(false)
    expect(register('query:a', 1800)).toBe(false)
    expect(register('query:b', 1900)).toBe(false)
    expect(register('query:b', 2200)).toBe(true)
  })

  it('同一连接和表始终复用同一个预览标签', () => {
    expect(sqlWorkbenchTablePreviewTabId('conn-a', 'device_a')).toBe('table-data-conn-a-device_a')
    expect(sqlWorkbenchTablePreviewTabId('conn-a', 'device_a')).not.toBe(
      sqlWorkbenchTablePreviewTabId('conn-a', 'device_b'),
    )
  })
})
