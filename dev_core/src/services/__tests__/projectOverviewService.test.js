const mockProjectFindAll = jest.fn();

jest.mock('../../models', () => ({
  Project: {
    findAll: mockProjectFindAll,
  },
  Tenant: {},
  User: {},
  ProjectTagBinding: {},
  ProjectTag: {},
  ProjectGroupMember: {},
  ProjectGroup: {},
  NodeDeployment: {},
}));

jest.mock('../../config/app', () => ({
  pagination: {
    defaultPage: 1,
    defaultLimit: 20,
    maxLimit: 100,
  },
}));

const { listProjectOverviews } = require('../projectOverviewService');

const makeProjectModel = (payload) => ({
  toOverviewPayload: jest.fn(() => payload),
});

describe('projectOverviewService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('会基于 toOverviewPayload 聚合工程总览结构（含 tags/group/runtimeSummary 与创建更新人名称）', async () => {
    const project = makeProjectModel({
      id: 'project-1',
      name: '智慧工厂',
      createdBy: 'user-creator',
      updatedBy: 'user-updater',
      createdAt: '2026-04-01T10:00:00.000Z',
      updatedAt: '2026-04-02T10:00:00.000Z',
      tags: [
        { id: 'tag-1', name: '生产' },
        { id: 'tag-2', name: '核心' },
      ],
      group: { id: 'group-1', name: '重点项目' },
      creator: { id: 'user-creator', username: 'creator', fullName: '创建者甲' },
      updater: { id: 'user-updater', username: 'updater', fullName: '更新者乙' },
      nodeDeployments: [
        {
          id: 'nd-1',
          status: 'running',
          mode: 'RELEASE',
          deployedAt: '2026-04-20T08:00:00.000Z',
          updatedAt: '2026-04-20T08:00:00.000Z',
        },
        {
          id: 'nd-2',
          status: 'stopped',
          mode: 'DEV',
          deployedAt: '2026-04-10T08:00:00.000Z',
          updatedAt: '2026-04-10T08:00:00.000Z',
        },
      ],
    });
    mockProjectFindAll.mockResolvedValue([project]);

    const result = await listProjectOverviews({
      req: {
        user: {
          role: 'PROJECT_ADMIN',
          tenantId: 'tenant-1',
        },
      },
      query: {},
    });

    expect(project.toOverviewPayload).toHaveBeenCalledTimes(1);
    expect(result.projects).toHaveLength(1);
    expect(result.projects[0]).toEqual(
      expect.objectContaining({
        tags: [
          { id: 'tag-1', name: '生产' },
          { id: 'tag-2', name: '核心' },
        ],
        group: { id: 'group-1', name: '重点项目' },
        createdByName: '创建者甲',
        updatedByName: '更新者乙',
        lastDeployedAt: '2026-04-20T08:00:00.000Z',
        runtimeSummary: expect.objectContaining({
          runtimeStatus: 'RUNNING',
          deploymentCount: 2,
          runningCount: 1,
          modeCounts: { DEV: 1, RELEASE: 1 },
        }),
      }),
    );
  });

  test.each([
    {
      label: 'createdAt',
      query: { sortBy: 'createdAt', sortOrder: 'DESC' },
      expected: ['project-c', 'project-b', 'project-a'],
    },
    {
      label: 'updatedAt',
      query: { sortBy: 'updatedAt', sortOrder: 'ASC' },
      expected: ['project-a', 'project-b', 'project-c'],
    },
    {
      label: 'lastDeployedAt',
      query: { sortBy: 'lastDeployedAt', sortOrder: 'DESC' },
      expected: ['project-b', 'project-a', 'project-c'],
    },
    {
      label: 'runtimeStatus',
      query: { sortBy: 'runtimeStatus', sortOrder: 'DESC' },
      expected: ['project-a', 'project-b', 'project-c'],
    },
  ])('支持左侧排序字段：$label', async ({ query, expected }) => {
    const projectA = makeProjectModel({
      id: 'project-a',
      name: 'A',
      createdAt: '2026-04-01T00:00:00.000Z',
      updatedAt: '2026-04-01T00:00:00.000Z',
      nodeDeployments: [
        { status: 'running', mode: 'RELEASE', deployedAt: '2026-04-20T00:00:00.000Z' },
      ],
    });
    const projectB = makeProjectModel({
      id: 'project-b',
      name: 'B',
      createdAt: '2026-04-02T00:00:00.000Z',
      updatedAt: '2026-04-02T00:00:00.000Z',
      nodeDeployments: [
        { status: 'deploying', mode: 'DEV', deployedAt: '2026-04-22T00:00:00.000Z' },
      ],
    });
    const projectC = makeProjectModel({
      id: 'project-c',
      name: 'C',
      createdAt: '2026-04-03T00:00:00.000Z',
      updatedAt: '2026-04-03T00:00:00.000Z',
      nodeDeployments: [
        { status: 'error', mode: 'RELEASE', deployedAt: null, updatedAt: '2026-04-03T00:00:00.000Z' },
      ],
    });

    mockProjectFindAll.mockResolvedValue([projectA, projectB, projectC]);
    const result = await listProjectOverviews({
      req: { user: { role: 'PROJECT_ADMIN', tenantId: 'tenant-1' } },
      query,
    });

    expect(result.projects.map((item) => item.id)).toEqual(expected);
  });

  test('支持按分组、标签、运行模式、部署状态、创建人筛选', async () => {
    const matched = makeProjectModel({
      id: 'project-match',
      name: '满足条件',
      createdBy: 'user-1',
      creator: { id: 'user-1', username: 'zhangsan', fullName: '张三' },
      tags: [{ id: 'tag-core', name: '核心' }],
      group: { id: 'group-main', name: '重点项目' },
      nodeDeployments: [{ status: 'running', mode: 'RELEASE', deployedAt: '2026-04-20T00:00:00.000Z' }],
      createdAt: '2026-04-01T00:00:00.000Z',
      updatedAt: '2026-04-01T00:00:00.000Z',
    });
    const unmatched = makeProjectModel({
      id: 'project-unmatch',
      name: '不满足条件',
      createdBy: 'user-2',
      creator: { id: 'user-2', username: 'lisi', fullName: '李四' },
      tags: [{ id: 'tag-other', name: '其它' }],
      group: { id: 'group-side', name: '边缘项目' },
      nodeDeployments: [{ status: 'stopped', mode: 'DEV', deployedAt: '2026-04-10T00:00:00.000Z' }],
      createdAt: '2026-04-02T00:00:00.000Z',
      updatedAt: '2026-04-02T00:00:00.000Z',
    });
    mockProjectFindAll.mockResolvedValue([matched, unmatched]);

    const result = await listProjectOverviews({
      req: { user: { role: 'PROJECT_ADMIN', tenantId: 'tenant-1' } },
      query: {
        group: '重点',
        tag: '核心',
        runtimeMode: 'RELEASE',
        deployStatus: 'running',
        createdBy: '张三',
      },
    });

    expect(result.projects).toHaveLength(1);
    expect(result.projects[0].id).toBe('project-match');
  });

  test('UUID 精确筛选支持大小写无关匹配（group/tag/createdBy）', async () => {
    const groupId = '6f9619ff-8b86-4d01-b42d-00cf4fc964ff';
    const tagId = '7f9619ff-8b86-4d01-b42d-00cf4fc964ff';
    const creatorId = '8f9619ff-8b86-4d01-b42d-00cf4fc964ff';
    const matched = makeProjectModel({
      id: 'project-uuid-match',
      name: 'UUID命中工程',
      createdBy: creatorId,
      creator: { id: creatorId, username: 'uuid-user', fullName: 'UUID 用户' },
      tags: [{ id: tagId, name: '标签A' }],
      group: { id: groupId, name: '分组A' },
      nodeDeployments: [{ status: 'running', mode: 'RELEASE', deployedAt: '2026-04-20T00:00:00.000Z' }],
      createdAt: '2026-04-01T00:00:00.000Z',
      updatedAt: '2026-04-01T00:00:00.000Z',
    });
    const unmatched = makeProjectModel({
      id: 'project-uuid-unmatch',
      name: 'UUID未命中工程',
      createdBy: '9f9619ff-8b86-4d01-b42d-00cf4fc964ff',
      creator: { id: '9f9619ff-8b86-4d01-b42d-00cf4fc964ff', username: 'other', fullName: '其它用户' },
      tags: [{ id: 'af9619ff-8b86-4d01-b42d-00cf4fc964ff', name: '标签B' }],
      group: { id: 'bf9619ff-8b86-4d01-b42d-00cf4fc964ff', name: '分组B' },
      nodeDeployments: [{ status: 'running', mode: 'RELEASE', deployedAt: '2026-04-21T00:00:00.000Z' }],
      createdAt: '2026-04-02T00:00:00.000Z',
      updatedAt: '2026-04-02T00:00:00.000Z',
    });
    mockProjectFindAll.mockResolvedValue([matched, unmatched]);

    const result = await listProjectOverviews({
      req: { user: { role: 'PROJECT_ADMIN', tenantId: 'tenant-1' } },
      query: {
        groupId: groupId.toUpperCase(),
        tagId: tagId.toUpperCase(),
        createdBy: creatorId.toUpperCase(),
      },
    });

    expect(result.projects).toHaveLength(1);
    expect(result.projects[0].id).toBe('project-uuid-match');
  });
});
