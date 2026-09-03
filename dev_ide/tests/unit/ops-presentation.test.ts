import { describe, expect, it } from 'vitest'
import {
  capabilityLabel,
  deploymentDetailPresentation,
  deploymentStatePresentation,
  enrollmentCapabilities,
  enrollmentInstallCommand,
  isDeployableNode,
  isDeployableReleaseVersion,
  isFreshNodeHeartbeat,
  runEventPresentation,
  serviceLabel,
  validateEnrollmentServerUrl,
} from '@/views/tenant/utils/ops-presentation'
describe('ops presentation', () => {
  it('只将已审批、在线且拥有入口和数据运行能力的物理节点作为部署目标', () => {
    const now = new Date('2026-08-30T08:00:00Z').getTime()
    const node = {
      id: 'n1',
      name: '节点',
      platform: 'linux' as const,
      capabilities: ['project_entry', 'data_runtime'] as Array<'project_entry' | 'data_runtime'>,
      approvedAt: 'x',
      desiredStatus: 'active',
      observedStatus: 'online',
      lastHeartbeatAt: '2026-08-30T07:59:30Z',
    }
    expect(isFreshNodeHeartbeat(node.lastHeartbeatAt, now)).toBe(true)
    expect(isDeployableNode(node, now)).toBe(true)
    expect(isDeployableNode({ ...node, assignedDeploymentId: 'd1' }, now)).toBe(false)
    expect(isDeployableNode({ ...node, capabilities: ['project_entry'] }, now)).toBe(false)
    expect(isDeployableNode({ ...node, observedStatus: 'offline' }, now)).toBe(false)
  })
  it('使用工程服务而不是运行角色文案', () => {
    expect(capabilityLabel.project_entry).toBe('工程入口')
    expect(serviceLabel.data_runtime).toBe('数据运行')
    expect(serviceLabel.collector).toBe('数据采集')
    expect(
      deploymentStatePresentation({ observedStatus: 'running', entryStatus: 'running' }),
    ).toMatchObject({ label: '运行中' })
    expect(
      deploymentStatePresentation({ observedStatus: 'running', entryStatus: 'failed' }),
    ).toMatchObject({ label: '运行失败' })
    expect(runEventPresentation('queued', 'deployment queued')).toEqual({
      stage: '已受理',
      message: '部署任务已创建，等待目标节点执行',
    })
    expect(runEventPresentation('dispatched', 'Kubernetes 工作负载已提交')).toEqual({
      stage: '已下发',
      message: '工作负载已下发，等待运行服务就绪',
    })
    expect(runEventPresentation('dispatched', '正在停止 Kubernetes 工作负载')).toEqual({
      stage: '处理中',
      message: '正在停止 Kubernetes 工作负载',
    })
    expect(
      deploymentDetailPresentation({
        observedStatus: 'pending',
        entryStatus: 'pending',
        services: [{ observedStatus: 'pending' }],
      }),
    ).toMatchObject({
      active: 1,
      steps: expect.arrayContaining([
        expect.objectContaining({ title: '服务启动', description: '等待运行服务就绪' }),
        expect.objectContaining({ title: '健康检查', description: '等待全部运行服务通过健康检查' }),
      ]),
    })
    expect(
      deploymentDetailPresentation({
        observedStatus: 'running',
        entryStatus: 'running',
        services: [{ observedStatus: 'running' }],
      }),
    ).toMatchObject({
      active: 4,
      steps: expect.arrayContaining([
        expect.objectContaining({ title: '服务启动', description: '全部运行服务已启动' }),
        expect.objectContaining({ title: '健康检查', description: '全部运行服务已通过健康检查' }),
      ]),
    })
    expect(
      deploymentDetailPresentation({
        observedStatus: 'pending',
        latestRunOperation: 'delete',
      }),
    ).toMatchObject({
      active: 0,
      steps: [
        { title: '停止运行资源', description: '正在停止 Kubernetes 工作负载' },
        { title: '清理运行态消息', description: '工作负载停止后清理运行态消息' },
        { title: '释放工程端口', description: '清理完成后释放工程端口并移出部署列表' },
      ],
    })
    expect(
      deploymentDetailPresentation({
        observedStatus: 'failed',
        latestRunOperation: 'delete',
      }),
    ).toMatchObject({
      processStatus: 'error',
      steps: expect.arrayContaining([
        expect.objectContaining({ title: '清理运行态消息', description: '运行态消息尚未完成清理' }),
        expect.objectContaining({ title: '释放工程端口', description: '工程端口仍受保护，尚未释放' }),
      ]),
    })
  })
  it('按平台固定节点能力，不允许生成不完整的运行节点', () => {
    expect(enrollmentCapabilities('linux')).toEqual(['project_entry', 'data_runtime'])
    expect(enrollmentCapabilities('linux', true)).toEqual([
      'project_entry',
      'data_runtime',
      'collector',
    ])
    expect(enrollmentCapabilities('windows', false)).toEqual(['collector'])
    expect(enrollmentCapabilities('windows', true)).toEqual(['collector'])
  })
  it('生成需要用户主动复制执行的平台安装命令', () => {
    expect(
      enrollmentInstallCommand({
        platform: 'linux',
        serverUrl: 'https://center.example.com',
        enrollmentCode: 'one-time-code',
        enableCollector: true,
      }),
    ).toBe(
      "sudo ./install.sh --enable-collector --server-url 'https://center.example.com' --enrollment-code 'one-time-code'",
    )
    expect(
      enrollmentInstallCommand({
        platform: 'windows',
        serverUrl: 'https://center.example.com',
        enrollmentCode: 'one-time-code',
      }),
    ).toBe(
      "& .\\install.ps1 -ServerUrl 'https://center.example.com' -EnrollmentCode 'one-time-code'",
    )
  })
  it('仅接受 HTTPS 或 loopback HTTP 的无参数中心根地址', () => {
    expect(validateEnrollmentServerUrl('https://center.example.com/')).toEqual({
      valid: true,
      normalized: 'https://center.example.com',
      loopback: false,
      error: '',
    })
    expect(validateEnrollmentServerUrl('http://localhost:18601')).toMatchObject({
      valid: true,
      normalized: 'http://localhost:18601',
      loopback: true,
    })
    expect(validateEnrollmentServerUrl('http://127.0.0.2:18601')).toMatchObject({
      valid: true,
      loopback: true,
    })
    expect(validateEnrollmentServerUrl('http://[::1]:18601')).toMatchObject({
      valid: true,
      loopback: true,
    })

    for (const value of [
      'http://center.example.com',
      'http://sub.localhost:18601',
      'center.example.com',
      'ftp://center.example.com',
      'https://user:secret@center.example.com',
      'https://center.example.com/api',
      'https://center.example.com?token=secret',
      'https://center.example.com/?',
      'https://center.example.com#fragment',
    ]) {
      expect(validateEnrollmentServerUrl(value).valid, value).toBe(false)
    }
    expect(() =>
      enrollmentInstallCommand({
        platform: 'linux',
        serverUrl: 'http://center.example.com',
        enrollmentCode: 'one-time-code',
      }),
    ).toThrow('必须使用 HTTPS')
  })
  it('只将结构完整且摘要合法的正式 Release 作为部署候选', () => {
    const release = {
      id: 'v1',
      projectId: 'p1',
      version: '1.0.0',
      status: 'success',
      artifactHash: 'a'.repeat(64),
      manifest: {
        schemaVersion: '2.0',
        artifacts: {
          client: { file: 'client-assets.tar.zst', checksum: `sha256:${'b'.repeat(64)}` },
          runtime: { file: 'runtime-artifact.tar.zst', checksum: `sha256:${'c'.repeat(64)}` },
        },
      },
    }

    expect(isDeployableReleaseVersion(release)).toBe(true)
    expect(isDeployableReleaseVersion(release, true)).toBe(false)
    expect(
      isDeployableReleaseVersion(
        {
          ...release,
          manifest: {
            ...release.manifest,
            artifacts: {
              ...release.manifest.artifacts,
              collector: {
                file: 'collector-artifact.tar.zst',
                checksum: `sha256:${'d'.repeat(64)}`,
              },
            },
          },
        },
        true,
      ),
    ).toBe(true)
    expect(isDeployableReleaseVersion({ ...release, status: 'ready' })).toBe(true)
    expect(isDeployableReleaseVersion({ ...release, status: 'failed' })).toBe(false)
    expect(isDeployableReleaseVersion({ ...release, artifactHash: 'A'.repeat(64) })).toBe(true)
    expect(isDeployableReleaseVersion({ ...release, artifactHash: 'g'.repeat(64) })).toBe(false)
    expect(isDeployableReleaseVersion({ ...release, artifactHash: 'a'.repeat(63) })).toBe(false)
    expect(isDeployableReleaseVersion({ ...release, manifest: undefined })).toBe(false)
    expect(
      isDeployableReleaseVersion({
        ...release,
        manifest: { ...release.manifest, artifacts: { client: release.manifest.artifacts.client } },
      }),
    ).toBe(false)
  })
})
