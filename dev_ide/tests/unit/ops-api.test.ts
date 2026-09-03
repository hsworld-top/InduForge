import { beforeEach, describe, expect, it, vi } from 'vitest'
const { getMock, postMock, patchMock, deleteMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  patchMock: vi.fn(),
  deleteMock: vi.fn(),
}))
vi.mock('@/utils/request', () => ({
  default: { get: getMock, post: postMock, patch: patchMock, delete: deleteMock },
}))
import { opsAPI } from '@/api/ops.api'
describe('opsAPI', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
    patchMock.mockReset()
    deleteMock.mockReset()
  })
  it('运行环境编辑和删除使用受控接口与名称确认', async () => {
    const environmentId = '66666666-6666-4666-8666-666666666666'
    patchMock.mockResolvedValue({ data: { id: environmentId, name: '新名称' } })
    deleteMock.mockResolvedValue({ data: { status: 'deleting' } })
    await opsAPI.updateRuntimeEnvironment(environmentId, { name: '新名称' })
    await opsAPI.deleteRuntimeEnvironment(environmentId, '新名称')
    expect(patchMock).toHaveBeenCalledWith(
      `/ops/runtime-environments/${environmentId}`,
      { name: '新名称' },
      { skipErrorToast: true },
    )
    expect(deleteMock).toHaveBeenCalledWith(`/ops/runtime-environments/${environmentId}`, {
      skipErrorToast: true,
      data: { confirmationName: '新名称' },
    })
  })
  it('运行环境管理使用正式分页、关联和移除接口', async () => {
    getMock
      .mockResolvedValueOnce({ data: { items: [], total: 0 } })
      .mockResolvedValueOnce({ data: { id: '66666666-6666-4666-8666-666666666666' } })
      .mockResolvedValueOnce({ data: { items: [], total: 0 } })
      .mockResolvedValueOnce({ data: { items: [], total: 0 } })
    postMock
      .mockResolvedValueOnce({ data: { id: '66666666-6666-4666-8666-666666666666' } })
      .mockResolvedValueOnce({ data: { items: [], total: 0 } })
    deleteMock.mockResolvedValue({ data: { removed: true } })
    const environmentId = '66666666-6666-4666-8666-666666666666'
    await opsAPI.listRuntimeEnvironments({ page: 1, pageSize: 10, status: 'uninitialized' })
    await opsAPI.getRuntimeEnvironment(environmentId)
    await opsAPI.listRuntimeEnvironmentNodes(environmentId, { page: 1, pageSize: 10 })
    await opsAPI.listRuntimeEnvironmentEvents(environmentId, { page: 1, pageSize: 10 })
    await opsAPI.createRuntimeEnvironment({ name: '生产环境' })
    await opsAPI.addRuntimeEnvironmentNodes(environmentId, ['33333333-3333-4333-8333-333333333333'])
    await opsAPI.removeRuntimeEnvironmentNode(environmentId, '33333333-3333-4333-8333-333333333333')
    expect(getMock).toHaveBeenNthCalledWith(1, '/ops/runtime-environments', {
      params: { page: 1, pageSize: 10, status: 'uninitialized' },
      skipErrorToast: true,
    })
    expect(postMock).toHaveBeenNthCalledWith(
      2,
      `/ops/runtime-environments/${environmentId}/nodes`,
      { nodeIds: ['33333333-3333-4333-8333-333333333333'] },
      { skipErrorToast: true },
    )
    expect(deleteMock).toHaveBeenCalledWith(
      `/ops/runtime-environments/${environmentId}/nodes/33333333-3333-4333-8333-333333333333`,
      { skipErrorToast: true },
    )
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
  it('物理工作节点使用独立安全移除接口', async () => {
    deleteMock.mockResolvedValue({ data: { status: 'removing' } })
    await expect(opsAPI.removeNode('33333333-3333-4333-8333-333333333333')).resolves.toEqual({
      status: 'removing',
    })
    expect(deleteMock).toHaveBeenCalledWith('/ops/nodes/33333333-3333-4333-8333-333333333333', {
      skipErrorToast: true,
    })
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
      accessPort: 17800,
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
      {
        projectId: 'p1',
        nodeId: 'n1',
        applicationVersionId: 'v1',
        accessPort: 17800,
        enableCollector: true,
      },
      { skipErrorToast: true },
    )
  })
  it('工程部署操作使用正式生命周期路径', async () => {
    postMock.mockResolvedValue({
      data: {
        deployment: { id: 'd1', projectId: 'p1', projectName: '工程', nodeId: 'n1' },
        run: { id: 'r1', deploymentId: 'd1', status: 'pending' },
      },
    })
    await expect(opsAPI.operateProjectDeployment('d1', 'restart')).resolves.toMatchObject({
      deployment: { id: 'd1', projectName: '工程' },
      run: { id: 'r1' },
    })
    expect(postMock).toHaveBeenCalledWith('/ops/project-deployments/d1/restart', undefined, {
      skipErrorToast: true,
    })
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

  it('从当前工程快照创建服务端递增的生产版本', async () => {
    postMock.mockResolvedValue({
      data: { id: 'v2', projectId: 'p1', version: '1.0.22', status: 'success' },
    })

    await expect(opsAPI.createProjectVersion('p1')).resolves.toMatchObject({
      id: 'v2',
      version: '1.0.22',
    })
    expect(postMock).toHaveBeenCalledWith(
      '/publish/p1',
      {},
      {
        skipErrorToast: true,
        timeout: 120_000,
      },
    )
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
