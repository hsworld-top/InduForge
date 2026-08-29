import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, postMock } = vi.hoisted(() => ({ getMock: vi.fn(), postMock: vi.fn() }))

vi.mock('@/utils/request', () => ({ default: { get: getMock, post: postMock } }))

import { opsAPI } from '@/api/ops.api'

describe('opsAPI', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
  })

  it('使用统一 ops 前缀获取运行集群和节点', async () => {
    getMock.mockResolvedValueOnce({ data: { items: [{ id: 'cluster-1' }], total: 1 } })
    getMock.mockResolvedValueOnce({ data: [{ id: 'node-1' }] })

    await expect(opsAPI.listRuntimeClusters({ page: 1, pageSize: 20 })).resolves.toEqual({
      items: [{ id: 'cluster-1' }],
      total: 1,
    })
    await expect(opsAPI.listHostNodes()).resolves.toEqual({
      items: [{ id: 'node-1', name: 'node-1' }],
      total: 1,
    })
    expect(getMock).toHaveBeenNthCalledWith(1, '/ops/runtime-clusters', {
      params: { page: 1, pageSize: 20 },
      skipErrorToast: true,
    })
    expect(getMock).toHaveBeenNthCalledWith(2, '/ops/host-nodes', {
      params: {},
      skipErrorToast: true,
    })
  })

  it('解包接入与部署创建响应，并保留一次性接入码和异步任务 ID', async () => {
    postMock
      .mockResolvedValueOnce({
        data: { enrollment: { id: 'enrollment-1', status: 'created' }, code: 'one-time-code' },
      })
      .mockResolvedValueOnce({
        data: {
          deployment: { id: 'deployment-1', projectId: 'project-1' },
          run: { id: 'run-1' },
        },
      })
      .mockResolvedValueOnce({
        data: { deployment: { id: 'deployment-1' }, run: { id: 'run-2' } },
      })

    await expect(
      opsAPI.createEnrollment({
        role: 'runtime_linux',
        displayName: '运行节点 01',
        ttlMinutes: 60,
        runtimeClusterId: 'cluster-1',
        packageId: 'package-1',
      }),
    ).resolves.toMatchObject({
      id: 'enrollment-1',
      enrollmentCode: 'one-time-code',
      packageId: 'package-1',
    })
    await expect(
      opsAPI.createProjectDeployment({
        projectId: 'project-1',
        runtimeClusterId: 'cluster-1',
        deploymentMode: 'development',
        workloads: [{ role: 'compute' }],
      }),
    ).resolves.toMatchObject({ id: 'deployment-1', runId: 'run-1' })
    await expect(opsAPI.runWorkloadAction('deployment-1', 'compute', 'restart')).resolves.toEqual({
      id: 'run-2',
    })

    expect(postMock).toHaveBeenNthCalledWith(
      1,
      '/ops/node-enrollments',
      {
        role: 'runtime_linux',
        displayName: '运行节点 01',
        ttlMinutes: 60,
        runtimeClusterId: 'cluster-1',
        packageId: 'package-1',
      },
      { skipErrorToast: true },
    )
    expect(postMock).toHaveBeenNthCalledWith(
      2,
      '/ops/project-deployments',
      {
        projectId: 'project-1',
        runtimeClusterId: 'cluster-1',
        deploymentMode: 'development',
        workloads: [{ role: 'compute' }],
      },
      { skipErrorToast: true },
    )
    expect(postMock).toHaveBeenNthCalledWith(
      3,
      '/ops/project-deployments/deployment-1/workloads/compute/restart',
      undefined,
      { skipErrorToast: true },
    )
  })

  it('可获取异步任务事件列表', async () => {
    getMock.mockResolvedValueOnce({
      data: { items: [{ id: 'event-1', stage: 'starting' }], total: 1 },
    })

    await expect(opsAPI.listDeploymentRunEvents('run-1')).resolves.toEqual({
      items: [{ id: 'event-1', stage: 'starting' }],
      total: 1,
    })
    expect(getMock).toHaveBeenCalledWith('/ops/deployment-runs/run-1/events', {
      skipErrorToast: true,
    })
  })

  it('将后端 observedStatus 规范化为轮询使用的 status', async () => {
    getMock.mockResolvedValueOnce({
      data: { id: 'run-1', observedStatus: 'running', progress: 50 },
    })

    await expect(opsAPI.getDeploymentRun('run-1')).resolves.toMatchObject({
      id: 'run-1',
      observedStatus: 'running',
      status: 'running',
      progress: 50,
    })
    expect(getMock).toHaveBeenCalledWith('/ops/deployment-runs/run-1', {
      skipErrorToast: true,
    })
  })

  it('所有运维请求都显式交由页面展示错误', async () => {
    getMock.mockImplementation((url: string) => {
      if (
        url.includes('node-packages') ||
        url.includes('runtime-clusters') ||
        url.includes('host-nodes') ||
        url.includes('project-deployments')
      ) {
        return Promise.resolve({ data: { items: [], total: 0 } })
      }
      return Promise.resolve({ data: { id: 'item-1' } })
    })
    postMock.mockImplementation((url: string) => {
      if (url === '/ops/node-enrollments') {
        return Promise.resolve({ data: { enrollment: { id: 'enrollment-1', status: 'created' } } })
      }
      if (url === '/ops/project-deployments') {
        return Promise.resolve({
          data: { deployment: { id: 'deployment-1', projectId: 'project-1' } },
        })
      }
      return Promise.resolve({ data: { id: 'item-1' } })
    })

    await Promise.all([
      opsAPI.listRuntimeClusters(),
      opsAPI.getRuntimeCluster('cluster-1'),
      opsAPI.createRuntimeCluster({ name: '运行集群', code: 'runtime-1' }),
      opsAPI.listEnrollments(),
      opsAPI.getEnrollment('enrollment-1'),
      opsAPI.createEnrollment({ role: 'runtime_linux', displayName: '节点', ttlMinutes: 60 }),
      opsAPI.approveEnrollment('enrollment-1'),
      opsAPI.rejectEnrollment('enrollment-1'),
      opsAPI.listHostNodes(),
      opsAPI.getHostNode('node-1'),
      opsAPI.listNodePackages(),
      opsAPI.downloadNodePackage('package-1'),
      opsAPI.listProjectDeployments(),
      opsAPI.getProjectDeployment('deployment-1'),
      opsAPI.createProjectDeployment({
        projectId: 'project-1',
        runtimeClusterId: 'cluster-1',
        deploymentMode: 'development',
        workloads: [{ role: 'compute' }],
      }),
      opsAPI.runWorkloadAction('deployment-1', 'compute', 'start'),
      opsAPI.getDeploymentRun('run-1'),
      opsAPI.listDeploymentRunEvents('run-1'),
    ])

    getMock.mock.calls.forEach(([, config]) => {
      expect(config).toMatchObject({ skipErrorToast: true })
    })
    postMock.mock.calls.forEach(([, , config]) => {
      expect(config).toMatchObject({ skipErrorToast: true })
    })
  })
})
