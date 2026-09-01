import { afterEach, describe, expect, test, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick, type App } from 'vue'
import i18n from '@/lang'
import ProjectManagement from '@/views/tenant/ProjectManagement.vue'

const {
  mockGetProjects,
  mockListProjectTags,
  mockListProjectGroups,
  mockCreateProjectTag,
  mockCreateProjectGroup,
  mockUpdateProjectGroup,
  mockDeleteProjectGroup,
  mockBindProjectGroup,
  mockBindProjectTags,
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
  mockCreateProjectTag: vi.fn(),
  mockCreateProjectGroup: vi.fn(),
  mockUpdateProjectGroup: vi.fn(),
  mockDeleteProjectGroup: vi.fn(),
  mockBindProjectGroup: vi.fn(),
  mockBindProjectTags: vi.fn(),
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
    createProjectTag: mockCreateProjectTag,
    createProjectGroup: mockCreateProjectGroup,
    updateProjectGroup: mockUpdateProjectGroup,
    deleteProjectGroup: mockDeleteProjectGroup,
    bindProjectGroup: mockBindProjectGroup,
    bindProjectTags: mockBindProjectTags,
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
      groupCards: {
        type: Array,
        default: () => [],
      },
    },
    emits: [
      'open-project',
      'selection-change',
      'open-runtime-access',
      'deploy',
      'export',
      'delete',
      'group-select',
      'group-add-project',
      'group-edit',
      'group-delete',
      'group-remove-project',
    ],
    setup(props, { emit, slots }) {
      return () => {
        const project = (props.projects as Array<Record<string, unknown>>)[0]
        const groupCards = props.groupCards as Array<Record<string, unknown>>
        return h('section', { 'data-testid': 'mock-overview-grid' }, [
          h('section', { 'data-testid': 'mock-group-cards' }, [
            h('span', `groups:${groupCards.length}`),
            groupCards.length > 0
              ? h(
                  'button',
                  {
                    'data-testid': 'mock-group-select-first',
                    onClick: () => {
                      const [group] = groupCards
                      emit('group-select', { groupId: group.id, groupName: group.name })
                    },
                  },
                  'select-first-group',
                )
              : null,
            groupCards.length > 0
              ? h(
                  'button',
                  {
                    'data-testid': 'project-group-add-project',
                    onClick: () => {
                      const [group] = groupCards
                      emit('group-add-project', group)
                    },
                  },
                  'add-project',
                )
              : null,
            groupCards.length > 0
              ? h(
                  'button',
                  {
                    'data-testid': 'project-group-edit-trigger',
                    onClick: () => {
                      const [group] = groupCards
                      emit('group-edit', group)
                    },
                  },
                  'edit-group',
                )
              : null,
            groupCards.length > 0
              ? h(
                  'button',
                  {
                    'data-testid': 'project-group-remove-project',
                    onClick: () => {
                      const [group] = groupCards
                      emit('group-remove-project', { groupId: group.id, projectId: 'project-1' })
                    },
                  },
                  'remove-project',
                )
              : null,
          ]),
          h('span', project?.name || 'empty-grid'),
          project
            ? h(
                'button',
                {
                  'data-testid': 'mock-grid-open',
                  onClick: () => emit('open-project', project),
                },
                'open-grid',
              )
            : null,
          project
            ? h(
                'button',
                {
                  'data-testid': 'mock-grid-select',
                  onClick: () =>
                    emit('selection-change', { projectId: project.id, selected: true }),
                },
                'select-grid',
              )
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
      groupCards: {
        type: Array,
        default: () => [],
      },
      grouped: {
        type: Boolean,
        default: true,
      },
      showGroupedProjectItems: {
        type: Boolean,
        default: true,
      },
      showTagColumn: {
        type: Boolean,
        default: true,
      },
    },
    emits: [
      'open-project',
      'selection-change',
      'open-runtime-access',
      'deploy',
      'export',
      'delete',
    ],
    setup(props, { emit, slots }) {
      return () => {
        const project = (props.projects as Array<Record<string, unknown>>)[0]
        const groupCards = props.groupCards as Array<Record<string, unknown>>
        return h('section', { 'data-testid': 'mock-overview-table' }, [
          h('span', project?.name || 'empty-table'),
          h(
            'span',
            { 'data-testid': 'mock-table-tag-column-state' },
            `tag-column:${props.showTagColumn}`,
          ),
          h(
            'span',
            { 'data-testid': 'mock-table-group-card-state' },
            `groups:${groupCards.length}`,
          ),
          h('span', { 'data-testid': 'mock-table-grouped-state' }, `grouped:${props.grouped}`),
          h(
            'span',
            { 'data-testid': 'mock-table-grouped-items-state' },
            `show-grouped-items:${props.showGroupedProjectItems}`,
          ),
          project
            ? h(
                'button',
                {
                  'data-testid': 'mock-table-open',
                  onClick: () => emit('open-project', project),
                },
                'open-table',
              )
            : null,
          project ? slots.actions?.({ project }) : null,
        ])
      }
    },
  }),
}))

vi.mock('@/views/tenant/project-management/ProjectPublishDialog.vue', () => ({
  default: defineComponent({
    name: 'ProjectPublishDialogStub',
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
    emits: ['update:visible', 'manage-versions', 'confirm'],
    setup(props) {
      return () =>
        props.visible
          ? h(
              'section',
              { 'data-testid': 'project-publish-dialog' },
              `publish:${(props.project as Record<string, unknown> | null)?.name || ''}`,
            )
          : null
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
    emits: ['select', 'add-project', 'edit', 'delete', 'remove-project'],
    setup(props, { emit }) {
      return () =>
        h('section', { 'data-testid': 'mock-group-cards' }, [
          h('span', `groups:${(props.groups as unknown[]).length}`),
          (props.groups as Array<Record<string, unknown>>).length > 0
            ? h(
                'button',
                {
                  'data-testid': 'mock-group-select-first',
                  onClick: () => {
                    const [group] = props.groups as Array<Record<string, unknown>>
                    emit('select', { groupId: group.id, groupName: group.name })
                  },
                },
                'select-first-group',
              )
            : null,
          (props.groups as Array<Record<string, unknown>>).length > 0
            ? h(
                'button',
                {
                  'data-testid': 'project-group-add-project',
                  onClick: () => {
                    const [group] = props.groups as Array<Record<string, unknown>>
                    emit('add-project', group)
                  },
                },
                'add-project',
              )
            : null,
          (props.groups as Array<Record<string, unknown>>).length > 0
            ? h(
                'button',
                {
                  'data-testid': 'project-group-edit-trigger',
                  onClick: () => {
                    const [group] = props.groups as Array<Record<string, unknown>>
                    emit('edit', group)
                  },
                },
                'edit-group',
              )
            : null,
          (props.groups as Array<Record<string, unknown>>).length > 0
            ? h(
                'button',
                {
                  'data-testid': 'project-group-remove-project',
                  onClick: () => {
                    const [group] = props.groups as Array<Record<string, unknown>>
                    emit('remove-project', { groupId: group.id, projectId: 'project-1' })
                  },
                },
                'remove-project',
              )
            : null,
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

vi.mock('@/views/tenant/project-management/ProjectGroupProjectPickerDialog.vue', () => ({
  default: defineComponent({
    name: 'ProjectGroupProjectPickerDialogStub',
    props: {
      visible: {
        type: Boolean,
        default: false,
      },
      projects: {
        type: Array,
        default: () => [],
      },
      groupId: {
        type: String,
        default: '',
      },
      tagOptions: {
        type: Array,
        default: () => [],
      },
      loading: {
        type: Boolean,
        default: false,
      },
    },
    emits: ['update:visible', 'select', 'query-change'],
    setup(props, { emit }) {
      return () => {
        if (!props.visible) {
          return null
        }
        const availableProjects = (props.projects as Array<Record<string, unknown>>).filter(
          (project) =>
            String((project.group as Record<string, unknown> | null)?.id || '') !== props.groupId,
        )

        return h('section', { 'data-testid': 'mock-group-project-picker' }, [
          h('span', `projects:${availableProjects.length}`),
          ...availableProjects.map((project) =>
            h(
              'button',
              {
                'data-testid': 'project-group-picker-item',
                onClick: () => emit('select', project.id),
              },
              String(project.name || project.id),
            ),
          ),
          availableProjects.length > 1
            ? h(
                'button',
                {
                  'data-testid': 'project-group-picker-batch-item',
                  onClick: () =>
                    emit(
                      'select',
                      availableProjects.map((project) => project.id),
                    ),
                },
                'batch-add',
              )
            : null,
        ])
      }
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
  app.component(
    'ElPopover',
    defineComponent({
      setup(_, { slots, attrs }) {
        return () => h('div', attrs, [slots.reference?.(), slots.default?.()])
      },
    }),
  )
  app.component(
    'el-popover',
    defineComponent({
      setup(_, { slots, attrs }) {
        return () => h('div', attrs, [slots.reference?.(), slots.default?.()])
      },
    }),
  )
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
    'View',
    'PriceTag',
    'UploadFilled',
    'Odometer',
    'VideoPlay',
    'VideoPause',
    'CopyDocument',
    'FolderOpened',
    'Setting',
    'ArrowRight',
    'ArrowDown',
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
  mockCreateProjectGroup.mockResolvedValue({})
  mockCreateProjectTag.mockResolvedValue({
    code: 0,
    msg: 'ok',
    data: {
      tag: { id: 'tag-new', name: '新标签' },
    },
    reqId: 'req-test',
  })
  mockUpdateProjectGroup.mockResolvedValue({})
  mockDeleteProjectGroup.mockResolvedValue({})
  mockBindProjectGroup.mockResolvedValue({})
  mockBindProjectTags.mockResolvedValue({})
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

    const addButton = container.querySelector(
      '[data-testid="project-add-trigger"]',
    ) as HTMLButtonElement
    const importButton = container.querySelector(
      '[data-testid="project-import-trigger"]',
    ) as HTMLButtonElement

    expect(addButton).not.toBeNull()
    expect(importButton).not.toBeNull()
    expect(container.querySelector('[data-testid="project-overview-toolbar"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="overview-sort-field"]')).not.toBeNull()
    expect(
      container.querySelector('[data-testid="mock-overview-pagination"]')?.textContent,
    ).toContain('显示第 11 到 20 条')

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
    const addButton = container.querySelector(
      '[data-testid="project-add-trigger"]',
    ) as HTMLButtonElement
    addButton.click()
    await nextTick()

    expect(container.textContent || '').toContain('Create Project')
    expect(container.textContent || '').not.toContain('Color Tag')
  })

  test('card/list 两个视图分支仍能切换，发布入口在工程管理内打开', async () => {
    primePageMocks()

    const { container } = await mountPage()

    expect(container.querySelector('[data-testid="mock-overview-grid"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="mock-group-cards"]')).not.toBeNull()
    expect(
      container.querySelectorAll('[data-testid="project-edit-action"]').length,
    ).toBeGreaterThan(0)
    expect(
      container.querySelectorAll('[data-testid="project-runtime-access-action"]').length,
    ).toBeGreaterThan(0)
    expect(
      container.querySelectorAll('[data-testid="project-edit-tags-action"]').length,
    ).toBeGreaterThan(0)
    expect(
      container.querySelectorAll('[data-testid="project-deploy-action"]').length,
    ).toBeGreaterThan(0)
    expect(
      container.querySelectorAll('[data-testid="project-export-action"]').length,
    ).toBeGreaterThan(0)
    expect(
      container.querySelectorAll('[data-testid="project-delete-action"]').length,
    ).toBeGreaterThan(0)
    expect(container.querySelector('[data-testid="project-ops-action"]')).toBeNull()

    const runtimeAccessButton = container.querySelector(
      '[data-testid="project-runtime-access-action"]',
    ) as HTMLButtonElement
    runtimeAccessButton.click()
    await nextTick()
    expect(container.querySelector('[data-testid="runtime-access-dialog"]')?.textContent).toContain(
      '示例工程',
    )

    const listViewButton = container.querySelector(
      '[data-testid="overview-view-list"]',
    ) as HTMLButtonElement
    listViewButton.click()
    await nextTick()

    expect(container.querySelector('[data-testid="mock-overview-table"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="mock-group-cards"]')).toBeNull()
    expect(
      container.querySelector('[data-testid="mock-table-tag-column-state"]')?.textContent,
    ).toBe('tag-column:false')
    expect(
      container.querySelector('[data-testid="mock-table-group-card-state"]')?.textContent,
    ).toBe('groups:1')
    expect(container.querySelector('[data-testid="mock-table-grouped-state"]')?.textContent).toBe(
      'grouped:true',
    )
    expect(
      container.querySelector('[data-testid="mock-table-grouped-items-state"]')?.textContent,
    ).toBe('show-grouped-items:false')

    const deployButton = container.querySelector(
      '[data-testid="project-deploy-action"]',
    ) as HTMLButtonElement
    deployButton.click()
    await nextTick()
    expect(container.querySelector('[data-testid="project-publish-dialog"]')?.textContent).toBe(
      'publish:示例工程',
    )
    expect(mockMessageInfo).not.toHaveBeenCalled()
    expect(mockRequestGet).not.toHaveBeenCalled()
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
    const selectButton = container.querySelector(
      '[data-testid="mock-grid-select"]',
    ) as HTMLButtonElement
    selectButton.click()
    await nextTick()

    const batchDeleteButton = container.querySelector(
      '[data-testid="project-batch-delete-trigger"]',
    ) as HTMLButtonElement

    expect(batchDeleteButton).not.toBeNull()
    batchDeleteButton.click()
    await flushPromises()

    expect(mockGetDeleteImpact).toHaveBeenCalledWith('project-1')
    expect(mockDeleteProject).toHaveBeenCalledWith('project-1', { force: true })
    expect(mockMessageBoxConfirm).toHaveBeenCalled()
  })

  test('修改分页条数只触发一次工程列表刷新', async () => {
    primePageMocks()
    const { container } = await mountPage()
    const callsAfterMount = mockGetProjects.mock.calls.length

    const pageSizeButton = container.querySelector(
      '[data-testid="mock-page-size-change"]',
    ) as HTMLButtonElement
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

  test('分组管理弹窗可创建新分组并刷新分组列表', async () => {
    primePageMocks()
    const { container } = await mountPage()
    const callsAfterMount = mockListProjectGroups.mock.calls.length

    const manageButton = container.querySelector(
      '[data-testid="project-group-manage-trigger"]',
    ) as HTMLButtonElement
    manageButton.click()
    await nextTick()

    expect(container.textContent || '').toContain('分组管理')

    const input = container.querySelector(
      '[data-testid="project-group-name-input"]',
    ) as HTMLInputElement
    input.value = '新分组'
    input.dispatchEvent(new Event('input'))
    await nextTick()

    const createButton = container.querySelector(
      '[data-testid="project-group-create-trigger"]',
    ) as HTMLButtonElement
    createButton.click()
    await flushPromises()

    expect(mockCreateProjectGroup).toHaveBeenCalledWith({ name: '新分组' })
    expect(mockListProjectGroups.mock.calls.length).toBeGreaterThan(callsAfterMount)
  })

  test('点击分组编辑先打开编辑弹窗，保存后才提交更新', async () => {
    primePageMocks()
    const promptSpy = vi.spyOn(window, 'prompt').mockImplementation(() => {
      throw new Error('不应调用 prompt')
    })

    try {
      const { container } = await mountPage()
      const manageButton = container.querySelector(
        '[data-testid="project-group-manage-trigger"]',
      ) as HTMLButtonElement
      manageButton.click()
      await nextTick()

      const editButton = container.querySelector(
        '[data-testid="project-group-edit-trigger"]',
      ) as HTMLButtonElement
      editButton.click()
      await nextTick()

      expect(promptSpy).not.toHaveBeenCalled()
      expect(mockUpdateProjectGroup).not.toHaveBeenCalled()
      expect(container.textContent || '').toContain('编辑分组')

      const input = container.querySelector(
        '[data-testid="project-group-edit-dialog-name-input"]',
      ) as HTMLInputElement
      input.value = '重点分组改名'
      input.dispatchEvent(new Event('input'))
      await nextTick()

      const saveButton = container.querySelector(
        '[data-testid="project-group-edit-dialog-save"]',
      ) as HTMLButtonElement
      saveButton.click()
      await flushPromises()

      expect(promptSpy).not.toHaveBeenCalled()
      expect(mockUpdateProjectGroup).toHaveBeenCalledWith('group-a', { name: '重点分组改名' })
    } finally {
      promptSpy.mockRestore()
    }
  })

  test('标签弹窗可创建新标签并自动加入当前工程标签绑定', async () => {
    primePageMocks()
    const { container } = await mountPage()

    const tagButton = container.querySelector(
      '[data-testid="project-edit-tags-action"]',
    ) as HTMLButtonElement
    tagButton.click()
    await nextTick()

    const tagInput = container.querySelector(
      '[data-testid="project-tag-name-input"]',
    ) as HTMLInputElement
    tagInput.value = '新标签'
    tagInput.dispatchEvent(new Event('input'))
    await nextTick()

    const createTagButton = container.querySelector(
      '[data-testid="project-tag-create-trigger"]',
    ) as HTMLButtonElement
    createTagButton.click()
    await flushPromises()

    expect(mockCreateProjectTag).toHaveBeenCalledWith({ name: '新标签' })

    const saveButton = [...container.querySelectorAll('button')].find((button) =>
      (button.textContent || '').includes('保存'),
    ) as HTMLButtonElement
    saveButton.click()
    await flushPromises()

    expect(mockBindProjectTags).toHaveBeenCalledWith('project-1', {
      tagIds: ['tag-new'],
    })
  })

  test('创建分组成功后刷新分组列表失败只记录警告不覆盖成功提示', async () => {
    primePageMocks()
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    try {
      const { container } = await mountPage()
      const refreshError = new Error('refresh groups failed')
      mockListProjectGroups.mockRejectedValueOnce(refreshError)

      const manageButton = container.querySelector(
        '[data-testid="project-group-manage-trigger"]',
      ) as HTMLButtonElement
      manageButton.click()
      await nextTick()

      const input = container.querySelector(
        '[data-testid="project-group-name-input"]',
      ) as HTMLInputElement
      input.value = '新分组'
      input.dispatchEvent(new Event('input'))
      await nextTick()

      const createButton = container.querySelector(
        '[data-testid="project-group-create-trigger"]',
      ) as HTMLButtonElement
      createButton.click()
      await flushPromises()

      expect(mockCreateProjectGroup).toHaveBeenCalledWith({ name: '新分组' })
      expect(mockMessageSuccess).toHaveBeenCalledWith('分组创建成功')
      expect(mockMessageError).not.toHaveBeenCalled()
      expect(warnSpy).toHaveBeenCalledWith('刷新工程分组列表失败:', refreshError)
    } finally {
      warnSpy.mockRestore()
    }
  })

  test('删除当前分组时工程刷新失败不阻断分组列表刷新', async () => {
    primePageMocks()
    const { container } = await mountPage()
    const callsAfterMount = mockListProjectGroups.mock.calls.length

    const selectGroupButton = container.querySelector(
      '[data-testid="mock-group-select-first"]',
    ) as HTMLButtonElement
    selectGroupButton.click()
    await flushPromises()

    mockGetProjects.mockRejectedValueOnce(new Error('project refresh failed'))

    const manageButton = container.querySelector(
      '[data-testid="project-group-manage-trigger"]',
    ) as HTMLButtonElement
    manageButton.click()
    await nextTick()

    const deleteButton = [...container.querySelectorAll('button')].find((button) =>
      (button.textContent || '').includes('删除分组'),
    ) as HTMLButtonElement
    deleteButton.click()
    await flushPromises()

    expect(mockDeleteProjectGroup).toHaveBeenCalledWith('group-a')
    expect(mockMessageError).toHaveBeenCalledWith('获取工程列表失败：error')
    expect(mockListProjectGroups.mock.calls.length).toBeGreaterThan(callsAfterMount)
  })

  test('进入分组后请求 groupId，点击面包屑全部工程后清空 groupId', async () => {
    primePageMocks()
    const { container } = await mountPage()

    const selectGroupButton = container.querySelector(
      '[data-testid="mock-group-select-first"]',
    ) as HTMLButtonElement
    selectGroupButton.click()
    await flushPromises()

    expect(mockGetProjects).toHaveBeenLastCalledWith(
      expect.objectContaining({
        groupId: 'group-a',
      }),
    )

    const listViewButton = container.querySelector(
      '[data-testid="overview-view-list"]',
    ) as HTMLButtonElement
    listViewButton.click()
    await nextTick()
    expect(
      container.querySelector('[data-testid="mock-table-group-card-state"]')?.textContent,
    ).toBe('groups:0')
    expect(container.querySelector('[data-testid="mock-table-grouped-state"]')?.textContent).toBe(
      'grouped:false',
    )

    const breadcrumbAllButton = container.querySelector(
      '[data-testid="project-group-breadcrumb-all"]',
    ) as HTMLButtonElement
    expect(breadcrumbAllButton).not.toBeNull()
    breadcrumbAllButton.click()
    await flushPromises()

    const lastQuery = mockGetProjects.mock.calls.at(-1)?.[0] as Record<string, unknown>
    expect(lastQuery).not.toHaveProperty('groupId')
  })

  test('点击分组工程移出按钮会清空工程分组', async () => {
    primePageMocks()
    const { container } = await mountPage()

    const removeButton = container.querySelector(
      '[data-testid="project-group-remove-project"]',
    ) as HTMLButtonElement
    removeButton.click()
    await flushPromises()

    expect(mockBindProjectGroup).toHaveBeenCalledWith('project-1', { groupId: null })
    expect(mockMessageSuccess).toHaveBeenCalledWith('工程已从分组移除')
  })

  test('进入分组后添加工程仍使用未按当前分组过滤的候选列表', async () => {
    primePageMocks()
    mockGetProjects.mockImplementation((params: Record<string, unknown> = {}) => {
      const projects = params.groupId
        ? [
            {
              id: 'project-1',
              name: '示例工程',
              runtimeSummary: {
                deploymentCount: 1,
              },
              tags: [],
              group: {
                id: 'group-a',
                name: '重点分组',
              },
            },
          ]
        : [
            {
              id: 'project-1',
              name: '示例工程',
              runtimeSummary: {
                deploymentCount: 1,
              },
              tags: [],
              group: {
                id: 'group-a',
                name: '重点分组',
              },
            },
            {
              id: 'project-2',
              name: '候选工程',
              runtimeSummary: {
                deploymentCount: 0,
              },
              tags: [],
              group: null,
            },
          ]

      return Promise.resolve({
        code: 0,
        msg: 'ok',
        data: {
          list: {
            projects,
          },
          pagination: {
            total: projects.length,
            page: 1,
            limit: 10,
            totalPages: 1,
          },
        },
      })
    })
    const { container } = await mountPage()

    const selectGroupButton = container.querySelector(
      '[data-testid="mock-group-select-first"]',
    ) as HTMLButtonElement
    selectGroupButton.click()
    await flushPromises()
    expect(mockGetProjects).toHaveBeenLastCalledWith(
      expect.objectContaining({
        groupId: 'group-a',
      }),
    )

    const addButton = container.querySelector(
      '[data-testid="project-group-add-project"]',
    ) as HTMLButtonElement
    addButton.click()
    await flushPromises()

    expect(container.querySelector('[data-testid="mock-group-project-picker"]')).not.toBeNull()
    expect(
      container.querySelector('[data-testid="mock-group-project-picker"]')?.textContent,
    ).toContain('projects:1')
    expect(
      container.querySelector('[data-testid="mock-group-project-picker"]')?.textContent,
    ).toContain('候选工程')
    const pickerQuery = mockGetProjects.mock.calls
      .map((call) => call[0] as Record<string, unknown>)
      .find(
        (query) => query?.limit === 100 && !Object.prototype.hasOwnProperty.call(query, 'groupId'),
      )
    expect(pickerQuery).toBeDefined()

    const pickerItem = container.querySelector(
      '[data-testid="project-group-picker-item"]',
    ) as HTMLButtonElement
    pickerItem.click()
    await flushPromises()

    expect(mockBindProjectGroup).toHaveBeenCalledWith('project-2', { groupId: 'group-a' })
    expect(mockMessageSuccess).toHaveBeenCalledWith('工程已添加到分组')
  })

  test('添加工程弹窗多选提交时会逐个绑定到当前分组', async () => {
    primePageMocks()
    mockGetProjects.mockImplementation((params: Record<string, unknown> = {}) => {
      const projects = params.groupId
        ? [
            {
              id: 'project-1',
              name: '示例工程',
              runtimeSummary: {
                deploymentCount: 1,
              },
              tags: [],
              group: {
                id: 'group-a',
                name: '重点分组',
              },
            },
          ]
        : [
            {
              id: 'project-1',
              name: '示例工程',
              runtimeSummary: {
                deploymentCount: 1,
              },
              tags: [],
              group: {
                id: 'group-a',
                name: '重点分组',
              },
            },
            {
              id: 'project-2',
              name: '候选工程 A',
              runtimeSummary: {
                deploymentCount: 0,
              },
              tags: [],
              group: null,
            },
            {
              id: 'project-3',
              name: '候选工程 B',
              runtimeSummary: {
                deploymentCount: 0,
              },
              tags: [],
              group: null,
            },
          ]

      return Promise.resolve({
        code: 0,
        msg: 'ok',
        data: {
          list: {
            projects,
          },
          pagination: {
            total: projects.length,
            page: 1,
            limit: 10,
            totalPages: 1,
          },
        },
      })
    })
    const { container } = await mountPage()

    const addButton = container.querySelector(
      '[data-testid="project-group-add-project"]',
    ) as HTMLButtonElement
    addButton.click()
    await flushPromises()

    const batchButton = container.querySelector(
      '[data-testid="project-group-picker-batch-item"]',
    ) as HTMLButtonElement
    batchButton.click()
    await flushPromises()

    expect(mockBindProjectGroup).toHaveBeenCalledWith('project-2', { groupId: 'group-a' })
    expect(mockBindProjectGroup).toHaveBeenCalledWith('project-3', { groupId: 'group-a' })
    expect(mockMessageSuccess).toHaveBeenCalledWith('已添加 2 个工程到分组')
  })
})
