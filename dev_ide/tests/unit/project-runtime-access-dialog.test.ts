import { defineComponent, createApp, h, nextTick, type App } from 'vue'
import { afterEach, describe, expect, test, vi } from 'vitest'

import ProjectRuntimeAccessDialog from '@/views/tenant/components/ProjectRuntimeAccessDialog.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => {
      if (key === 'projectManagement.runtimeAccess.dialogTitle' && params?.name) {
        return `成员与权限 - ${params.name}`
      }
      return key
    },
  }),
}))

vi.mock('@/api/project.api.js', () => ({
  projectAPI: {
    listRuntimeUsers: vi.fn().mockResolvedValue({
      data: [
        {
          id: 'user-1',
          username: 'alice',
          displayName: 'Alice',
          status: 'active',
          roleIds: ['role-1'],
        },
      ],
    }),
    createRuntimeUser: vi.fn(),
    updateRuntimeUserStatus: vi.fn(),
    updateRuntimeUserRoles: vi.fn(),
    resetRuntimeUserPassword: vi.fn(),
    listRuntimeRoles: vi.fn().mockResolvedValue({
      data: [
        {
          id: 'role-1',
          code: 'PROJECT_ADMIN',
          name: '管理员',
          description: '系统管理员',
          status: 'active',
          isSystem: true,
          bindingCount: 1,
          grantCount: 8,
        },
      ],
    }),
    createRuntimeRole: vi.fn(),
    updateRuntimeRole: vi.fn(),
    deleteRuntimeRole: vi.fn(),
  },
}))

const createSlotStub = (name: string) =>
  defineComponent({
    name,
    props: {
      title: {
        type: String,
        default: '',
      },
      label: {
        type: String,
        default: '',
      },
      name: {
        type: String,
        default: '',
      },
    },
    setup(_, { slots }) {
      return () =>
        h('div', { class: name }, [slots.prepend?.(), slots.default?.(), slots.footer?.()])
    },
  })

const mountedApps: Array<{ app: App; root: HTMLElement }> = []

const mountDialog = async () => {
  const root = document.createElement('div')
  document.body.appendChild(root)
  const app = createApp(ProjectRuntimeAccessDialog, {
    visible: true,
    project: {
      id: 'project-1',
      name: '演示工程',
    },
  })

  ;[
    'ElDialog',
    'ElAlert',
    'ElEmpty',
    'ElTabs',
    'ElTabPane',
    'ElTable',
    'ElButton',
    'ElForm',
    'ElFormItem',
    'ElInput',
    'ElSelect',
    'ElOption',
    'ElTag',
  ].forEach((name) => {
    app.component(name, createSlotStub(`${name}-stub`))
  })
  app.component(
    'ElTableColumn',
    defineComponent({
      name: 'ElTableColumnStub',
      setup() {
        return () => null
      },
    }),
  )

  app.mount(root)
  mountedApps.push({ app, root })
  await Promise.resolve()
  await nextTick()
  return root
}

afterEach(() => {
  while (mountedApps.length > 0) {
    const mounted = mountedApps.pop()
    mounted?.app.unmount()
    mounted?.root.remove()
  }
})

describe('ProjectRuntimeAccessDialog', () => {
  test('成员权限主弹窗应包含统一的视觉壳层结构', async () => {
    const root = await mountDialog()

    expect(root.querySelector('.runtime-access-shell')).not.toBeNull()
    expect(root.querySelector('.runtime-access-tabs-card')).not.toBeNull()
    expect(root.querySelector('.runtime-access-table-wrap')).not.toBeNull()
    expect(root.textContent || '').toContain('PROJECT_')
  })
})
