import { describe, expect, test, vi } from 'vitest'
import { buildProjectOverviewQueryParams } from '@/views/tenant/project-management/use-project-filters'
import {
  buildProjectGroupCardItems,
  resolveProjectStatusMetric,
} from '@/views/tenant/project-management/project-group-utils'
import {
  buildProjectPaginationSummary,
  useProjectOverviewState,
} from '@/views/tenant/project-management/use-project-overview'
import type { ProjectOverviewRuntimeSummary } from '@/views/tenant/project-management/project-overview.types'

const createRuntimeSummary = (): ProjectOverviewRuntimeSummary => ({
  runtimeStatus: 'running',
  deploymentCount: 1,
  runningCount: 1,
  statusCounts: {
    pending: 0,
    deploying: 0,
    running: 1,
    stopped: 0,
    error: 0,
    rollback: 0,
  },
  modeCounts: {
    DEV: 1,
    RELEASE: 0,
  },
  lastDeployedAt: null,
})

describe('project-overview-state', () => {
  test('buildProjectGroupCardItems 会合并接口分组和当前工程列表中的工程名预览', () => {
    const groupItems = buildProjectGroupCardItems(
      [
        { id: ' group-b ', name: ' 乙组 ', sortOrder: 2, projectCount: 8 },
        { id: 'group-a', name: '甲组', sortOrder: 1 },
        { id: '', name: '无效分组' },
      ],
      [
        {
          id: ' project-1 ',
          name: ' 工程一 ',
          group: { id: ' group-a ', name: '甲组' },
          tags: [],
          runtimeSummary: createRuntimeSummary(),
        },
        {
          id: 2002,
          name: '工程二',
          group: { id: 'group-a', name: '甲组' },
          tags: [],
          runtimeSummary: createRuntimeSummary(),
        },
        {
          id: 'project-3',
          name: '工程三',
          group: { id: 'group-b', name: '乙组' },
          tags: [],
          runtimeSummary: createRuntimeSummary(),
        },
        { projectId: 'legacy-project', groupId: 'group-a', name: '兼容工程' },
        { id: '', group: { id: 'group-a' }, name: '无效工程' },
        { projectId: 'project-4', groupId: '', name: '无分组工程' },
        { id: 'project-5', group: { id: 'group-a' }, name: '   ' },
      ],
    )

    expect(groupItems).toEqual([
      expect.objectContaining({
        id: 'group-a',
        name: '甲组',
        projectCount: 3,
        projects: [
          { id: 'project-1', name: '工程一' },
          { id: '2002', name: '工程二' },
          { id: 'legacy-project', name: '兼容工程' },
        ],
      }),
      expect.objectContaining({
        id: 'group-b',
        name: '乙组',
        projectCount: 8,
        projects: [{ id: 'project-3', name: '工程三' }],
      }),
    ])
  })

  test('resolveProjectStatusMetric 会把 running/error/unknown 映射为稳定进度和颜色等级', () => {
    expect(resolveProjectStatusMetric('running')).toEqual({
      labelKey: 'projectManagement.deployStatusRunning',
      percent: 100,
      level: 'good',
    })
    expect(resolveProjectStatusMetric('error')).toEqual({
      labelKey: 'projectManagement.deployStatusError',
      percent: 20,
      level: 'bad',
    })
    expect(resolveProjectStatusMetric('unknown')).toEqual({
      labelKey: 'projectManagement.deployStatusUnknown',
      percent: 0,
      level: 'muted',
    })
  })

  test('查询参数构造会完成筛选归一与去空', () => {
    const query = buildProjectOverviewQueryParams({
      filters: {
        search: '  智慧工厂  ',
        composite: {
          runtimeModes: ['release', 'DEV', ''],
          deployStatuses: ['running', 'ERROR', 'invalid'],
          createdBy: '  张三  ',
        },
        tagIds: ['tag-1', 'tag-2', 'tag-1', ''],
        sortBy: 'runtimeStatus',
        sortOrder: 'desc',
      },
      pagination: {
        page: 2,
        limit: 10,
      },
      groupContext: {
        groupId: 'group-main',
        groupName: '重点项目',
      },
    })

    expect(query).toEqual({
      page: 2,
      limit: 10,
      name: '智慧工厂',
      groupId: 'group-main',
      tagId: ['tag-1', 'tag-2'],
      runtimeMode: ['RELEASE', 'DEV'],
      deployStatus: ['running', 'error'],
      createdBy: '张三',
      sortBy: 'runtimeStatus',
      sortOrder: 'DESC',
    })
  })

  test('分页摘要文案输出轻量区间格式', () => {
    expect(
      buildProjectPaginationSummary({
        page: 2,
        limit: 10,
        total: 35,
      }),
    ).toBe('11-20 / 35')

    expect(
      buildProjectPaginationSummary({
        page: 1,
        limit: 20,
        total: 0,
      }),
    ).toBe('0 / 0')
  })

  test('分组上下文与排序状态在同一状态树中保持一致并共同参与查询', async () => {
    const fetchProjectsApi = vi.fn().mockResolvedValue({
      code: 0,
      msg: 'ok',
      data: {
        list: {
          projects: [
            {
              id: 'project-1',
              name: '示例工程',
            },
          ],
        },
        pagination: {
          total: 1,
          page: 1,
          limit: 20,
          totalPages: 1,
        },
      },
    })

    const overview = useProjectOverviewState({ fetchProjectsApi })
    overview.setGroupContext({
      groupId: 'group-main',
      groupName: '重点项目',
    })
    overview.setSort('runtimeStatus', 'ASC')

    await overview.fetchProjects()

    expect(overview.groupContext).toBe(overview.state.groupContext)
    expect(overview.filters).toBe(overview.state.filters)
    expect(overview.state.groupContext.groupId).toBe('group-main')
    expect(overview.state.filters.sortBy).toBe('runtimeStatus')
    expect(overview.state.filters.sortOrder).toBe('ASC')
    expect(fetchProjectsApi).toHaveBeenCalledWith(
      expect.objectContaining({
        groupId: 'group-main',
        sortBy: 'runtimeStatus',
        sortOrder: 'ASC',
      }),
    )
  })

  test('工程/标签/分组的数字 ID 会稳定归一为字符串', async () => {
    const fetchProjectsApi = vi.fn().mockResolvedValue({
      code: 0,
      msg: 'ok',
      data: {
        list: {
          projects: [
            {
              id: 1001,
              name: '数字 ID 工程',
              createdBy: 88,
              group: {
                id: 7,
                name: '重点分组',
              },
              tags: [
                {
                  id: 9,
                  name: '核心标签',
                },
              ],
            },
          ],
        },
        pagination: {
          total: 1,
          page: 1,
          limit: 20,
          totalPages: 1,
        },
      },
    })

    const overview = useProjectOverviewState({ fetchProjectsApi })
    await overview.fetchProjects()

    expect(overview.projects.value).toHaveLength(1)
    expect(overview.projects.value[0]).toEqual(
      expect.objectContaining({
        id: '1001',
        createdBy: '88',
        group: expect.objectContaining({ id: '7' }),
        tags: [expect.objectContaining({ id: '9' })],
      }),
    )
  })

  test.each([
    {
      label: 'businessData.pagination',
      expectedSummary: '11-20 / 35',
      payload: {
        code: 0,
        msg: 'ok',
        data: {
          list: { projects: [{ id: 'project-a', name: 'A' }] },
          pagination: { total: 35, page: 2, limit: 10, totalPages: 4 },
        },
      },
    },
    {
      label: 'businessData.list.pagination',
      expectedSummary: '11-20 / 26',
      payload: {
        code: 0,
        msg: 'ok',
        data: {
          list: {
            projects: [{ id: 'project-b', name: 'B' }],
            pagination: { total: 26, page: 2, limit: 10, totalPages: 3 },
          },
        },
      },
    },
    {
      label: 'legacy response.data.list.pagination',
      expectedSummary: '11-18 / 18',
      payload: {
        data: {
          list: {
            projects: [{ id: 'project-c', name: 'C' }],
            pagination: { total: 18, page: 2, limit: 10, totalPages: 2 },
          },
        },
      },
    },
  ])('不同分页包裹位置可正确刷新总数与摘要: $label', async ({ payload, expectedSummary }) => {
    const fetchProjectsApi = vi.fn().mockResolvedValue(payload)
    const overview = useProjectOverviewState({ fetchProjectsApi })
    await overview.fetchProjects()

    expect(overview.projects.value.length).toBeGreaterThan(0)
    expect(overview.pagination.page).toBe(2)
    expect(overview.pagination.limit).toBe(10)
    expect(overview.pagination.total).toBeGreaterThan(0)
    expect(overview.pagination.summary).toBe(expectedSummary)
  })

  test('双 data 包裹下可同时提取 list.projects 与 pagination', async () => {
    const fetchProjectsApi = vi.fn().mockResolvedValue({
      data: {
        data: {
          list: {
            projects: [{ id: 'project-double-data', name: '双层包裹工程' }],
            pagination: { total: 12, page: 2, limit: 10, totalPages: 2 },
          },
        },
      },
    })

    const overview = useProjectOverviewState({ fetchProjectsApi })
    await overview.fetchProjects()

    expect(overview.projects.value).toHaveLength(1)
    expect(overview.projects.value[0]).toEqual(
      expect.objectContaining({ id: 'project-double-data' }),
    )
    expect(overview.pagination.total).toBe(12)
    expect(overview.pagination.summary).toBe('11-12 / 12')
  })

  test('分页缺失时会回退为显式 0，避免沿用旧 total', async () => {
    const fetchProjectsApi = vi
      .fn()
      .mockResolvedValueOnce({
        code: 0,
        msg: 'ok',
        data: {
          list: { projects: [{ id: 'project-1', name: 'A' }] },
          pagination: { total: 45, page: 2, limit: 10, totalPages: 5 },
        },
      })
      .mockResolvedValueOnce({
        code: 0,
        msg: 'ok',
        data: {
          list: { projects: [{ id: 'project-2', name: 'B' }] },
        },
      })

    const overview = useProjectOverviewState({ fetchProjectsApi })
    await overview.fetchProjects()
    expect(overview.pagination.total).toBe(45)
    expect(overview.pagination.summary).toBe('11-20 / 45')

    await overview.fetchProjects()
    expect(overview.pagination.total).toBe(0)
    expect(overview.pagination.totalPages).toBe(0)
    expect(overview.pagination.summary).toBe('0 / 0')
  })

  test('页码越界时摘要不会出现反向区间，状态页码与摘要一致', async () => {
    expect(
      buildProjectPaginationSummary({
        page: 5,
        limit: 10,
        total: 12,
      }),
    ).toBe('11-12 / 12')

    const fetchProjectsApi = vi.fn().mockResolvedValue({
      code: 0,
      msg: 'ok',
      data: {
        list: { projects: [{ id: 'project-boundary', name: '边界工程' }] },
        pagination: { total: 12, page: 5, limit: 10, totalPages: 2 },
      },
    })

    const overview = useProjectOverviewState({ fetchProjectsApi })
    await overview.fetchProjects()

    expect(overview.pagination.page).toBe(2)
    expect(overview.pagination.summary).toBe('11-12 / 12')
  })

  test('setGroupContext 部分更新只覆盖传入字段，保留已有 groupName', () => {
    const overview = useProjectOverviewState()
    overview.setGroupContext({
      groupId: 'group-origin',
      groupName: '原始分组',
    })

    overview.setGroupContext({
      groupId: 'group-next',
    })

    expect(overview.groupContext.groupId).toBe('group-next')
    expect(overview.groupContext.groupName).toBe('原始分组')
  })
})
