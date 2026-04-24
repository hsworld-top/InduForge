import { afterEach, describe, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, type App } from 'vue'
import i18n from '@/lang'
import ProjectManagement from '@/views/tenant/ProjectManagement.vue'

const {
  mockGetProjects,
  mockListProjectTags,
  mockListProjectGroups,
  mockDeleteProject,
  mockGetDeleteImpact,
  mockExportProject,
  mockImportProject,
  mockSocket,
  mockRequestGet,
  mockRequestPost,
  mockRequestDelete,
  mockMessageSuccess,
  mockMessageWarning,
  mockMessageError,
  mockMessageInfo,
  mockMessageBoxConfirm,
} = vi.hoisted(() => ({
  mockGetProjects: vi.fn(),
  mockListProjectTags: vi.fn(),
  mockListProjectGroups: vi.fn(),
  mockDeleteProject: vi.fn(),
  mockGetDeleteImpact: vi.fn(),
  mockExportProject: vi.fn(),
  mockImportProject: vi.fn(),
  mockSocket: {
    on: vi.fn(),
    off: vi.fn(),
  },
  mockRequestGet: vi.fn(),
  mockRequestPost: vi.fn(),
  mockRequestDelete: vi.fn(),
  mockMessageSuccess: vi.fn(),
  mockMessageWarning: vi.fn(),
  mockMessageError: vi.fn(),
  mockMessageInfo: vi.fn(),
  mockMessageBoxConfirm: vi.fn(),
}))

vi.mock('element-plus', () => ({
  ElMessage: {
    success: mockMessageSuccess,
    warning: mockMessageWarning,
    error: mockMessageError,
    info: mockMessageInfo,
  },
  ElMessageBox: {
    confirm: mockMessageBoxConfirm,
  },
}))

vi.mock('@/api/project.api', () => ({
  projectAPI: {
    getProjects: mockGetProjects,
    listProjectTags: mockListProjectTags,
    listProjectGroups: mockListProjectGroups,
    createProject: vi.fn(),
    updateProject: vi.fn(),
    deleteProject: mockDeleteProject,
    getDeleteImpact: mockGetDeleteImpact,
    exportProject: mockExportProject,
    importProject: mockImportProject,
  },
}))

vi.mock('@/store', () => ({
  useAuthStore: () => ({
    userInfo: {
      role: 'SYSTEM_ADMIN',
    },
  }),
}))

vi.mock('@/permissions', () => ({
  can: () => true,
}))

vi.mock('@/utils/socket', () => ({
  initSocket: () => mockSocket,
  getSocket: () => mockSocket,
}))

vi.mock('@/utils/storage', () => ({
  Storage: {
    getTenantId: () => 'tenant-test',
  },
}))

vi.mock('@/utils/request', () => ({
  default: {
    get: mockRequestGet,
    post: mockRequestPost,
    delete: mockRequestDelete,
  },
  getApiErrorMessage: () => 'error',
}))

vi.mock('@/views/tenant/project-management/ProjectOverviewGrid.vue', () => ({
  default: defineComponent({
    name: 'ProjectOverviewGridStub',
    props: {
      projects: {
        type: Array,
        default: () => [],
      },
    },
    emits: ['open-project', 'selection-change', 'open-runtime-access', 'deploy', 'export', 'delete'],
    setup(props, { emit, slots }) {
      return () => {
        const project = (props.projects as Array<Record<string, unknown>>)[0]
        return h('section', { 'data-testid': 'mock-overview-grid' }, [
          h('span', project?.name || 'empty-grid'),
          project
            ? h('button', {
                'data-testid': 'mock-grid-open',
                onClick: () => emit('open-project', project),
              }, 'open-grid')
            : null,
          project
            ? h('button', {
                'data-testid': 'mock-grid-select',
                onClick: () => emit('selection-change', { projectId: project.id, selected: true }),
              }, 'select-grid')
            : null,
          project ? slots.actions?.({ project }) : null,
        ])
      }
    },
  }),
}))

vi.mock('@/views/tenant/project-management/ProjectOverviewTable.vue', () => ({
  default: defineComponent({
    name: 'ProjectOverviewTableStub',
    props: {
      projects: {
        type: Array,
        default: () => [],
      },
    },
    emits: ['open-project', 'selection-change', 'open-runtime-access', 'deploy', 'export', 'delete'],
    setup(props, { emit, slots }) {
      return () => {
        const project = (props.projects as Array<Record<string, unknown>>)[0]
        return h('section', { 'data-testid': 'mock-overview-table' }, [
          h('span', project?.name || 'empty-table'),
          project
            ? h('button', {
                'data-testid': 'mock-table-open',
                onClick: () => emit('open-project', project),
              }, 'open-table')
            : null,
          project ? slots.actions?.({ project }) : null,
        ])
      }
    },
  }),
}))

vi.mock('@/views/tenant/project-management/ProjectGroupCards.vue', () => ({
  default: defineComponent({
    name: 'ProjectGroupCardsStub',
    props: {
      groups: {
        type: Array,
        default: () => [],
      },
    },
    emits: ['select'],
    setup(props, { emit }) {
      return () =>
        h('section', { 'data-testid': 'mock-group-cards' }, [
          h('span', `groups:${(props.groups as unknown[]).length}`),
          h(
            'button',
            {
              'data-testid': 'mock-group-select',
              onClick: () => emit('select', { groupId: '', groupName: '全部工程' }),
            },
            'select-all-groups',
          ),
        ])
    },
  }),
}))

vi.mock('@/views/tenant/project-management/ProjectOverviewPagination.vue', () => ({
  default: defineComponent({
    name: 'ProjectOverviewPaginationStub',
    props: {
      page: {
        type: Number,
        default: 1,
      },
      limit: {
        type: Number,
        default: 10,
      },
      summary: {
        type: String,
        default: '',
      },
    },
    emits: ['update:page', 'update:limit', 'change'],
    setup(props, { emit }) {
      return () =>
        h('div', { 'data-testid': 'mock-overview-pagination' }, [
          h('span', props.summary),
          h(
            'button',
            {
              'data-testid': 'mock-page-size-change',
              onClick: () => {
                emit('update:limit', 20)
                emit('update:page', 1)
                emit('change', { page: 1, limit: 20 })
              },
            },
            'change-size',
          ),
          h(
            'button',
            {
              'data-testid': 'mock-page-next',
              onClick: () => {
                emit('update:page', Number(props.page) + 1)
                emit('change', { page: Number(props.page) + 1, limit: Number(props.limit) })
              },
            },
            'next-page',
          ),
        ])
    },
  }),
}))

vi.mock('@/views/tenant/components/ProjectRuntimeAccessDialog.vue', () => ({
  default: defineComponent({
    name: 'ProjectRuntimeAccessDialogStub',
    props: {
      visible: {
        type: Boolean,
        default: false,
      },
      project: {
        type: Object,
        default: null,
      },
    },
    setup(props) {
      return () =>
        props.visible
          ? h(
              'div',
              { 'data-testid': 'runtime-access-dialog' },
              `runtime-access:${(props.project as Record<string, unknown> | null)?.name || ''}`,
            )
          : null
    },
  }),
}))

type MountResult = {
  app: App<Element>
  container: HTMLDivElement
  unmount: () => void
}

const mountedInstances: MountResult[] = []

const ElInputStub = defineComponent({
  name: 'ElInput',
  props: {
    modelValue: {
      type: [String, Number],
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, attrs }) {
    return () =>
      h('input', {
        ...attrs,
        value: String(props.modelValue ?? ''),
        onInput: (event: Event) => {
          const target = event.target as HTMLInputElement
          emit('update:modelValue', target.value)
        },
      })
  },
})

const ElButtonStub = defineComponent({
  name: 'ElButton',
  setup(_, { slots, attrs }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
          type: 'button',
        },
        slots.default?.(),
      )
  },
})

const ElSelectStub = defineComponent({
  name: 'ElSelect',
  props: {
    modelValue: {
      type: [String, Number],
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, slots, attrs }) {
    return () =>
      h(
        'select',
        {
          ...attrs,
          value: String(props.modelValue ?? ''),
          onChange: (event: Event) => {
            const target = event.target as HTMLSelectElement
            emit('update:modelValue', target.value)
          },
        },
        slots.default?.(),
      )
  },
})

const ElOptionStub = defineComponent({
  name: 'ElOption',
  props: {
    label: {
      type: String,
      default: '',
    },
    value: {
      type: [String, Number],
      default: '',
    },
  },
  setup(props) {
    return () => h('option', { value: String(props.value) }, props.label)
  },
})

const ElTooltipStub = defineComponent({
  name: 'ElTooltip',
  setup(_, { slots }) {
    return () => h('span', slots.default?.())
  },
})

const ElIconStub = defineComponent({
  name: 'ElIcon',
  setup(_, { slots }) {
    return () => h('span', slots.default?.())
  },
})

const ElDialogStub = defineComponent({
  name: 'ElDialog',
  props: {
    modelValue: {
      type: Boolean,
      default: false,
    },
    title: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      props.modelValue
        ? h('div', { class: 'dialog-stub' }, [
            h('div', { class: 'dialog-title' }, props.title),
            h('div', { class: 'dialog-body' }, slots.default?.()),
            h('div', { class: 'dialog-footer' }, slots.footer?.()),
          ])
        : null
  },
})

const IconGlyphStub = defineComponent({
  setup() {
    return () => h('span')
  },
})

const ElementContainerStub = defineComponent({
  setup(_, { slots, attrs }) {
    return () => h('div', attrs, slots.default?.())
  },
})

const registerElementStubs = (app: App) => {
  app.component('ElInput', ElInputStub)
  app.component('el-input', ElInputStub)
  app.component('ElButton', ElButtonStub)
  app.component('el-button', ElButtonStub)
  app.component('ElSelect', ElSelectStub)
  app.component('el-select', ElSelectStub)
  app.component('ElOption', ElOptionStub)
  app.component('el-option', ElOptionStub)
  app.component('ElTooltip', ElTooltipStub)
  app.component('el-tooltip', ElTooltipStub)
  app.component('ElIcon', ElIconStub)
  app.component('el-icon', ElIconStub)
  app.component('ElDialog', ElDialogStub)
  app.component('el-dialog', ElDialogStub)
  app.component('ElPopover', ElementContainerStub)
  app.component('el-popover', ElementContainerStub)
  app.component('ElScrollbar', ElementContainerStub)
  app.component('el-scrollbar', ElementContainerStub)
  app.component('ElCheckboxGroup', ElementContainerStub)
  app.component('el-checkbox-group', ElementContainerStub)
  app.component('ElCheckbox', ElementContainerStub)
  app.component('el-checkbox', ElementContainerStub)
  app.component('ElEmpty', ElementContainerStub)
  app.component('el-empty', ElementContainerStub)
  app.component('ElTag', ElementContainerStub)
  app.component('el-tag', ElementContainerStub)
  app.component('ElForm', ElementContainerStub)
  app.component('el-form', ElementContainerStub)
  app.component('ElFormItem', ElementContainerStub)
  app.component('el-form-item', ElementContainerStub)
  app.component('ElAlert', ElementContainerStub)
  app.component('el-alert', ElementContainerStub)
  app.component('ElRadioGroup', ElementContainerStub)
  app.component('el-radio-group', ElementContainerStub)
  app.component('ElRadio', ElementContainerStub)
  app.component('el-radio', ElementContainerStub)
  app.component('ElTable', ElementContainerStub)
  app.component('el-table', ElementContainerStub)
  app.component('ElTableColumn', defineComponent({ setup: () => () => null }))
  app.component('el-table-column', defineComponent({ setup: () => () => null }))
  ;[
    'Plus',
    'Upload',
    'RefreshRight',
    'Download',
    'Delete',
    'User',
    'UploadFilled',
    'Odometer',
    'VideoPlay',
    'VideoPause',
    'CopyDocument',
  ].forEach((name) => {
    app.component(name, IconGlyphStub)
  })
  app.directive('loading', {})
}

const primePageMocks = ({
  page = 2,
  limit = 10,
  total = 35,
  deploymentCount = 1,
}: {
  page?: number
  limit?: number
  total?: number
  deploymentCount?: number
} = {}) => {
  mockGetProjects.mockResolvedValue({
    code: 0,
    msg: 'ok',
    data: {
      list: {
        projects: [
          {
            id: 'project-1',
            name: '示例工程',
            runtimeSummary: {
              deploymentCount,
            },
            tags: [],
            group: {
              id: 'group-a',
              name: '重点分组',
            },
          },
        ],
      },
      pagination: {
        total,
        page,
        limit,
        totalPages: Math.ceil(total / limit),
      },
    },
  })
  mockListProjectTags.mockResolvedValue({
    data: {
      list: {
        tags: [{ id: 'tag-1', name: '核心标签' }],
      },
    },
  })
  mockListProjectGroups.mockResolvedValue({
    data: {
      list: {
        groups: [{ id: 'group-a', name: '重点分组', projectCount: 1 }],
      },
    },
  })
  mockGetDeleteImpact.mockResolvedValue({
    totalDeploymentCount: 1,
    hasActiveDeployments: false,
    activeDeploymentCount: 0,
    activeNodeCount: 0,
  })
  mockDeleteProject.mockResolvedValue({})
  mockExportProject.mockResolvedValue(new Blob(['ok'], { type: 'application/zip' }))
  mockImportProject.mockResolvedValue({})
  mockRequestGet.mockResolvedValue({ data: { data: { items: [] } } })
  mockRequestPost.mockResolvedValue({})
  mockRequestDelete.mockResolvedValue({})
  mockMessageBoxConfirm.mockResolvedValue('confirm')
}

const flushPromises = async () => {
  await Promise.resolve()
  await new Promise((resolve) => setTimeout(resolve, 0))
}

const mountPage = async (): Promise<MountResult> => {
  const container = document.createElement('div')
  document.body.appendChild(container)

  const Root = defineComponent({
    setup() {
      return () => h(ProjectManagement)
    },
  })

  const app = createApp(Root)
  app.use(i18n)
  registerElementStubs(app)
  app.mount(container)
  await nextTick()
  await flushPromises()
  await nextTick()

  const result: MountResult = {
    app,
    container,
    unmount: () => {
      app.unmount()
      container.remove()
    },
  }
  mountedInstances.push(result)
  return result
}

afterEach(() => {
  while (mountedInstances.length > 0) {
    mountedInstances.pop()?.unmount()
  }
  i18n.global.locale.value = 'zh'
  vi.clearAllMocks()
})

describe('project-management-page', () => {
  test('页面保留新增/导入入口，并在中文创建对话框中不再出现颜色标签文案', async () => {
    primePageMocks()
    const { container } = await mountPage()

    const originalCreateElement = document.createElement.bind(document)
    const createElementSpy = vi.spyOn(document, 'createElement')
    createElementSpy.mockImplementation(((tagName: string, options?: ElementCreationOptions) => {
      const element = originalCreateElement(tagName, options)
      if (tagName === 'input') {
        vi.spyOn(element as HTMLInputElement, 'click').mockImplementation(() => {})
      }
      return element
    }) as typeof document.createElement)

    const addButton = container.querySelector('[data-testid="project-add-trigger"]') as HTMLButtonElement
    const importButton = container.querySelector('[data-testid="project-import-trigger"]') as HTMLButtonElement

    expect(addButton).not.toBeNull()
    expect(importButton).not.toBeNull()
    expect(container.querySelector('[data-testid="project-overview-toolbar"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="overview-sort-field"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="mock-overview-pagination"]')?.textContent).toContain('显示第 11 到 20 条')

    addButton.click()
    await nextTick()

    expect(container.textContent || '').toContain('创建工程')
    expect(container.textContent || '').not.toContain('颜色标签')

    importButton.click()
    expect(createElementSpy).toHaveBeenCalledWith('input')
  })

  test('英文视图下创建对话框也不再出现 Color Tag 文案', async () => {
    primePageMocks()
    i18n.global.locale.value = 'en'

    const { container } = await mountPage()
    const addButton = container.querySelector('[data-testid="project-add-trigger"]') as HTMLButtonElement
    addButton.click()
    await nextTick()

    expect(container.textContent || '').toContain('Create Project')
    expect(container.textContent || '').not.toContain('Color Tag')
  })

  test('card/list 两个视图分支仍能切换，成员与权限/发布部署/导出/删除接线点仍可见', async () => {
    primePageMocks()

    const { container } = await mountPage()

    expect(container.querySelector('[data-testid="mock-overview-grid"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="mock-group-cards"]')).not.toBeNull()
    expect(container.querySelectorAll('[data-testid="project-runtime-access-action"]').length).toBeGreaterThan(0)
    expect(container.querySelectorAll('[data-testid="project-deploy-action"]').length).toBeGreaterThan(0)
    expect(container.querySelectorAll('[data-testid="project-export-action"]').length).toBeGreaterThan(0)
    expect(container.querySelectorAll('[data-testid="project-delete-action"]').length).toBeGreaterThan(0)
    expect(container.querySelector('[data-testid="project-ops-action"]')).toBeNull()

    const runtimeAccessButton = container.querySelector('[data-testid="project-runtime-access-action"]') as HTMLButtonElement
    runtimeAccessButton.click()
    await nextTick()
    expect(container.querySelector('[data-testid="runtime-access-dialog"]')?.textContent).toContain('示例工程')

    const listViewButton = container.querySelector('[data-testid="overview-view-list"]') as HTMLButtonElement
    listViewButton.click()
    await nextTick()

    expect(container.querySelector('[data-testid="mock-overview-table"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="mock-group-cards"]')).toBeNull()

    const deployButton = container.querySelector('[data-testid="project-deploy-action"]') as HTMLButtonElement
    deployButton.click()
    await flushPromises()
    expect(mockRequestGet).toHaveBeenCalled()
  })

  test('批量删除会复用删除影响评估与强制删除保护路径', async () => {
    primePageMocks()
    mockGetDeleteImpact.mockResolvedValue({
      totalDeploymentCount: 2,
      hasActiveDeployments: true,
      activeDeploymentCount: 2,
      activeNodeCount: 1,
    })

    const { container } = await mountPage()
    const selectButton = container.querySelector('[data-testid="mock-grid-select"]') as HTMLButtonElement
    selectButton.click()
    await nextTick()

    const batchDeleteButton = [...container.querySelectorAll('button')].find((button) =>
      (button.textContent || '').includes('批量删除'),
    ) as HTMLButtonElement | undefined

    expect(batchDeleteButton).toBeDefined()
    batchDeleteButton?.click()
    await flushPromises()

    expect(mockGetDeleteImpact).toHaveBeenCalledWith('project-1')
    expect(mockDeleteProject).toHaveBeenCalledWith('project-1', { force: true })
    expect(mockMessageBoxConfirm).toHaveBeenCalled()
  })

  test('修改分页条数只触发一次工程列表刷新', async () => {
    primePageMocks()
    const { container } = await mountPage()
    const callsAfterMount = mockGetProjects.mock.calls.length

    const pageSizeButton = container.querySelector('[data-testid="mock-page-size-change"]') as HTMLButtonElement
    pageSizeButton.click()
    await flushPromises()

    expect(mockGetProjects).toHaveBeenCalledTimes(callsAfterMount + 1)
    expect(mockGetProjects).toHaveBeenLastCalledWith(
      expect.objectContaining({
        page: 1,
        limit: 20,
      }),
    )
  })
})
