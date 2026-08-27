import { describe, expect, test } from 'vitest'
import {
  buildHistoryStoragePayload,
  createHistoryStorageDraft,
  historyStorageSummary,
  validateHistoryStorageDraft,
} from '../src/models/history-storage'
import {
  HistoryStorageConfigurationSchema,
  HistoryStorageSourceListSchema,
} from '../src/api/schemas/history-storage.schema'

const targets = [
  {
    id: 'if-ts-1',
    name: 'IF时序库',
    type: 'builtin.timeseries' as const,
    lastTestStatus: 'succeeded' as const,
  },
]

describe('history storage model', () => {
  test('首次开启自动选中唯一 IF 时序库和推荐值', () => {
    const draft = createHistoryStorageDraft('custom', null, targets)
    expect(draft.writeMode).toBe('on_change')
    expect(draft.deadband).toBe(0)
    expect(draft.maxSilenceMs).toBe(3_600_000)
    expect(draft.targets).toEqual([{ connectionId: 'if-ts-1', retentionDays: 30 }])
  })

  test('多个 IF 时序库时不替用户选择', () => {
    const draft = createHistoryStorageDraft('custom', null, [
      ...targets,
      {
        id: 'if-ts-2',
        name: 'IF时序库2',
        type: 'builtin.timeseries',
        lastTestStatus: 'succeeded',
      },
    ])
    expect(draft.targets).toEqual([{ connectionId: '', retentionDays: 30 }])
    expect(validateHistoryStorageDraft(draft)).toBe('请选择主存储目标')
  })

  test('关闭配置返回空目标后重新启用仍提供默认主目标', () => {
    const draft = createHistoryStorageDraft(
      'off',
      {
        writeMode: 'on_change',
        intervalMs: null,
        deadband: null,
        maxSilenceMs: null,
        offlineBehavior: 'store_stale',
        targets: [],
      },
      targets,
    )
    expect(draft.targets).toEqual([{ connectionId: 'if-ts-1', retentionDays: 30 }])
  })

  test('按模式只发送相关参数并生成唯一主目标', () => {
    const draft = createHistoryStorageDraft('custom', null, targets)
    draft.writeMode = 'periodic_snapshot'
    draft.intervalMs = 120_000
    draft.offlineBehavior = 'skip'
    const payload = buildHistoryStoragePayload(draft)
    expect(payload.configuration?.intervalMs).toBe(120_000)
    expect(payload.configuration?.deadband).toBeNull()
    expect(payload.configuration?.targets[0]?.isPrimary).toBe(true)
  })

  test('最长静默为空时保持不设置语义', () => {
    const draft = createHistoryStorageDraft(
      'custom',
      {
        writeMode: 'on_change',
        deadband: 0,
        maxSilenceMs: null,
        offlineBehavior: 'store_stale',
        targets: [
          {
            connectionId: 'if-ts-1',
            connectionName: 'IF时序库',
            connectionType: 'builtin.timeseries',
            lastTestStatus: 'succeeded',
            isPrimary: true,
            sortOrder: 0,
            retentionDays: null,
          },
        ],
      },
      targets,
    )
    expect(draft.maxSilenceMs).toBeNull()
    expect(buildHistoryStoragePayload(draft).configuration?.maxSilenceMs).toBeNull()
  })

  test('配置响应拒绝不受支持的目标类型', () => {
    expect(() =>
      HistoryStorageConfigurationSchema.parse({
        writeMode: 'every_sample',
        offlineBehavior: 'store_stale',
        targets: [
          {
            connectionId: 'mysql-1',
            connectionName: 'MySQL',
            connectionType: 'relational',
            lastTestStatus: 'succeeded',
            isPrimary: true,
            sortOrder: 0,
            retentionDays: 30,
          },
        ],
      }),
    ).toThrow()
  })

  test('解析来源列表并生成人类可读摘要', () => {
    const result = HistoryStorageSourceListSchema.parse({
      list: [
        {
          scope: { type: 'access_source', id: 'source-1', name: '生产 MQTT', sourceType: 'mqtt' },
          datapointCount: 20,
          pointOverrideCount: 1,
          historyState: 'enabled',
          writeMode: 'on_change',
          targetCount: 1,
          primaryTargetName: 'IF时序库',
          retentionDays: 30,
        },
      ],
      pagination: { page: 1, pageSize: 20, total: 1, totalPages: 1 },
    })
    expect(historyStorageSummary(result.list[0])).toBe('变化时保存 · IF时序库 · 30天')
  })
})
