import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, ref, type App } from 'vue'
import i18n from '@/lang'

const { listVersions, restoreDevelopment } = vi.hoisted(() => ({
  listVersions: vi.fn(),
  restoreDevelopment: vi.fn(),
}))

vi.mock('@/api/ops.api', () => ({
  opsAPI: {
    listRuntimeEnvironments: vi.fn().mockResolvedValue({
      items: [{ id: 'environment-1', name: '默认环境', isDefault: true, status: 'available' }],
      total: 1,
    }),
    listProjectVersions: listVersions,
    listProjectDeployments: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    listRuntimeEnvironmentNodes: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    getDevelopmentDeploymentRequirements: vi.fn().mockResolvedValue(['base']),
    restoreProjectDevelopment: restoreDevelopment,
    getDevelopmentRestoreTask: vi.fn(),
  },
}))

vi.mock('@/utils/request', () => ({
  default: { delete: vi.fn() },
  getApiErrorMessage: (error: unknown, fallback: string) =>
    error instanceof Error ? error.message : fallback,
}))

vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn() },
  ElMessageBox: { confirm: vi.fn() },
}))

import ProjectPublishDialog from '@/views/tenant/project-management/ProjectPublishDialog.vue'

const ButtonStub = defineComponent({
  inheritAttrs: false,
  props: { disabled: Boolean },
  setup(props, { attrs, slots }) {
    return () => h('button', { ...attrs, disabled: props.disabled }, slots.default?.())
  },
})

describe('版本管理恢复开发态', () => {
  let app: App | null = null
  afterEach(() => {
    app?.unmount()
    app = null
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  it('在同一弹窗确认并展示同步成功结果，之后发送工作区刷新目标', async () => {
    listVersions.mockResolvedValue({
      items: [
        {
          id: 'version-1',
          projectId: 'project-1',
          version: '1.0.1',
          status: 'ready',
          restorable: true,
          authoringProjectRevision: 3,
        },
      ],
      total: 1,
    })
    restoreDevelopment.mockResolvedValue({
      taskId: 'restore-1',
      state: 'succeeded',
      backupId: 'backup-1',
      completedAt: '2026-09-04T08:00:00Z',
    })
    const restored = vi.fn()
    const opened = vi.fn()
    const root = document.createElement('div')
    document.body.appendChild(root)
    const visible = ref(false)
    app = createApp(
      defineComponent({
        setup() {
          return () =>
            h(ProjectPublishDialog, {
              visible: visible.value,
              project: { id: 'project-1', name: '演示工程' },
              initialDeployment: {
                id: 'deployment-1',
                environmentId: 'environment-1',
                environmentName: '默认环境',
                mode: 'production',
                applicationVersionId: 'version-1',
                desiredStatus: 'running',
                observedStatus: 'running',
                operationInProgress: false,
                placements: {},
              },
              initialAction: { kind: 'publish-release', label: '发布新版本' },
              onRestoreComplete: restored,
              onOpenRestoredWorkspace: opened,
            })
        },
      }),
    )
    app.use(i18n)
    app.component('ElDialog', defineComponent({
      setup(_, { slots }) {
        return () => h('section', [slots.default?.(), slots.footer?.()])
      },
    }))
    app.component('ElButton', ButtonStub)
    app.component('ElTooltip', defineComponent({ setup(_, { slots }) { return () => slots.default?.() } }))
    app.component('ElTag', defineComponent({ setup(_, { slots }) { return () => h('span', slots.default?.()) } }))
    app.component('ElIcon', defineComponent({ setup(_, { slots }) { return () => h('span', slots.default?.()) } }))
    app.directive('loading', () => {})
    app.mount(root)
    visible.value = true
    await new Promise((resolve) => setTimeout(resolve, 0))
    await nextTick()
    expect(root.textContent).toContain('1.0.1')

    const restore = [...root.querySelectorAll('button')].find((item) =>
      item.textContent?.includes('基于此版本继续开发'),
    ) as HTMLButtonElement
    restore.click()
    await nextTick()
    expect(root.textContent).toContain('系统将先备份当前开发内容')

    ;(root.querySelector('[data-testid="restore-development-confirm"]') as HTMLButtonElement).click()
    await Promise.resolve()
    await nextTick()
    expect(restoreDevelopment).toHaveBeenCalledWith('version-1')
    expect(root.textContent).toContain('已基于 v1.0.1 创建新的开发起点')
    expect(root.textContent).toContain('恢复前备份')
    expect(root.textContent).toContain('已创建')
    expect(root.textContent).not.toContain('backup-1')
    expect(restored).toHaveBeenCalledWith({ projectId: 'project-1' })

    const designer = [...root.querySelectorAll('button')].find((item) =>
      item.textContent?.includes('进入设计中心'),
    ) as HTMLButtonElement
    designer.click()
    expect(opened).toHaveBeenCalledWith({ projectId: 'project-1', target: 'designer' })
  })
})
