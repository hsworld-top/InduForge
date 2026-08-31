import { beforeEach, describe, expect, it, vi } from 'vitest'
const { getMock, postMock } = vi.hoisted(() => ({ getMock: vi.fn(), postMock: vi.fn() }))
vi.mock('@/utils/request', () => ({ default: { get: getMock, post: postMock } }))
import { opsAPI } from '@/api/ops.api'
describe('opsAPI', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
  })
  it('只请求物理节点接口', async () => {
    getMock.mockResolvedValue({
      data: { items: [{ id: 'node-1', platform: 'linux', capabilities: [] }], total: 1 },
    })
    await expect(opsAPI.listNodes()).resolves.toMatchObject({
      items: [{ id: 'node-1', name: 'node-1' }],
    })
    expect(getMock).toHaveBeenCalledWith('/ops/nodes', { params: {}, skipErrorToast: true })
  })
  it('接入与部署使用平台、能力和单 nodeId', async () => {
    postMock
      .mockResolvedValueOnce({
        data: { enrollment: { id: 'e1', platform: 'linux', capabilities: [] }, code: 'code' },
      })
      .mockResolvedValueOnce({
        data: {
          deployment: {
            id: 'd1',
            projectId: 'p1',
            projectName: 'p',
            nodeId: 'n1',
            applicationVersionId: 'v1',
          },
          run: { id: 'r1' },
        },
      })
    await opsAPI.createEnrollment({
      platform: 'linux',
      capabilities: ['project_entry', 'data_runtime'],
      displayName: '节点',
      ttlMinutes: 60,
    })
    await opsAPI.createProjectDeployment({
      projectId: 'p1',
      nodeId: 'n1',
      applicationVersionId: 'v1',
      enableCollector: true,
    })
    expect(postMock).toHaveBeenNthCalledWith(
      1,
      '/ops/node-enrollments',
      expect.objectContaining({
        platform: 'linux',
        capabilities: ['project_entry', 'data_runtime'],
      }),
      { skipErrorToast: true },
    )
    expect(postMock).toHaveBeenNthCalledWith(
      2,
      '/ops/project-deployments',
      { projectId: 'p1', nodeId: 'n1', applicationVersionId: 'v1', enableCollector: true },
      { skipErrorToast: true },
    )
  })
  it('服务操作不再使用 workload 路径', async () => {
    postMock.mockResolvedValue({ data: { run: { id: 'r1' } } })
    await opsAPI.runServiceAction('d1', 'project_entry', 'restart')
    expect(postMock).toHaveBeenCalledWith(
      '/ops/project-deployments/d1/services/project_entry/restart',
      undefined,
      { skipErrorToast: true },
    )
  })

  it('按页查询真实发布版本', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [
          {
            id: 'v1',
            projectId: 'p1',
            version: '1.0.0',
            status: 'success',
            artifactHash: 'a'.repeat(64),
            manifest: {
              schemaVersion: '2.0',
              artifacts: { client: { file: 'client-assets.tar.zst' }, runtime: {} },
            },
          },
        ],
        total: 1,
      },
    })

    await expect(
      opsAPI.listProjectVersions('p1', { page: 2, pageSize: 50 }),
    ).resolves.toMatchObject({
      items: [{ id: 'v1', version: '1.0.0', manifest: { schemaVersion: '2.0' } }],
      total: 1,
    })
    expect(getMock).toHaveBeenCalledWith('/publish/p1/versions', {
      params: { page: 2, pageSize: 50 },
      skipErrorToast: true,
    })
  })

  it('可按工程 ID 精确查询现有单节点部署', async () => {
    getMock.mockResolvedValue({ data: { items: [], total: 0 } })
    await opsAPI.listProjectDeployments({ page: 1, pageSize: 1, projectId: 'p1' })
    expect(getMock).toHaveBeenCalledWith('/ops/project-deployments', {
      params: { page: 1, pageSize: 1, projectId: 'p1' },
      skipErrorToast: true,
    })
  })
})
