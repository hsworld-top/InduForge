import { describe, expect, it } from 'vitest'
import { projectPublishCta } from '@/views/tenant/project-management/project-publish-cta'

describe('工程发布弹窗 CTA', () => {
  it('覆盖新建、同模式更新和生产相同版本禁用', () => {
    expect(projectPublishCta(null, 'DEV', '')).toEqual({ label: '部署开发版', disabled: false })
    expect(projectPublishCta(null, 'RELEASE', 'v2')).toEqual({ label: '发布并部署', disabled: false })
    expect(projectPublishCta({ mode: 'development', observedStatus: 'running' }, 'DEV', '').label).toBe('更新开发版')
    expect(projectPublishCta({ mode: 'production', observedStatus: 'running', applicationVersionId: 'v1' }, 'RELEASE', 'v1')).toEqual({ label: '当前版本已部署', disabled: true })
    expect(projectPublishCta({ mode: 'production', observedStatus: 'running', applicationVersionId: 'v1' }, 'RELEASE', 'v2').label).toBe('部署新版本')
  })
  it('覆盖切换模式、停止恢复和弹窗期间状态变化', () => {
    expect(projectPublishCta({ mode: 'production', observedStatus: 'running' }, 'DEV', '').label).toBe('切换为开发部署')
    expect(projectPublishCta({ mode: 'development', observedStatus: 'stopped' }, 'RELEASE', 'v2').label).toBe('切换并启动')
    expect(projectPublishCta({ mode: 'development', observedStatus: 'stopped' }, 'DEV', '').label).toBe('更新并启动')
    expect(projectPublishCta({ mode: 'development', observedStatus: 'pending' }, 'DEV', '').disabled).toBe(true)
    expect(projectPublishCta({ mode: 'development', observedStatus: 'failed' }, 'DEV', '').label).toContain('异常')
  })
})
