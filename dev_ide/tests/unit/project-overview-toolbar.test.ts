import { afterEach, describe, expect, test } from 'vitest'
import { createApp, defineComponent, h, nextTick, type App } from 'vue'
import i18n from '@/lang'
import ProjectOverviewToolbar from '@/views/tenant/project-management/ProjectOverviewToolbar.vue'

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

const ElPopoverStub = defineComponent({
  name: 'ElPopover',
  setup(_, { slots }) {
    return () =>
      h('div', [
        h('div', { class: 'popover-reference' }, slots.reference?.()),
        h('div', { class: 'popover-content' }, slots.default?.()),
      ])
  },
})

const ElScrollbarStub = defineComponent({
  name: 'ElScrollbar',
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const ElCheckboxGroupStub = defineComponent({
  name: 'ElCheckboxGroup',
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const ElCheckboxStub = defineComponent({
  name: 'ElCheckbox',
  setup(_, { slots }) {
    return () => h('label', slots.default?.())
  },
})

type EmissionBag = {
  search: string[]
  viewMode: string[]
  sortBy: string[]
  sortOrder: string[]
  openGroupManager: number
  compositeFilters: Array<{
    runtimeModes: string[]
    deployStatuses: string[]
    createdBy: string
  }>
}

type MountResult = {
  container: HTMLElement
  unmount: () => void
  emissions: EmissionBag
}

const mountedInstances: MountResult[] = []

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
  app.component('ElPopover', ElPopoverStub)
  app.component('el-popover', ElPopoverStub)
  app.component('ElScrollbar', ElScrollbarStub)
  app.component('el-scrollbar', ElScrollbarStub)
  app.component('ElCheckboxGroup', ElCheckboxGroupStub)
  app.component('el-checkbox-group', ElCheckboxGroupStub)
  app.component('ElCheckbox', ElCheckboxStub)
  app.component('el-checkbox', ElCheckboxStub)
}

const mountToolbar = (extraProps: Record<string, unknown> = {}): MountResult => {
  const emissions: EmissionBag = {
    search: [],
    viewMode: [],
    sortBy: [],
    sortOrder: [],
    openGroupManager: 0,
    compositeFilters: [],
  }

  const container = document.createElement('div')
  document.body.appendChild(container)

  const Root = defineComponent({
    setup() {
      return () =>
        h(ProjectOverviewToolbar, {
          search: '',
          viewMode: 'card',
          sortBy: 'createdAt',
          sortOrder: 'DESC',
          tagOptions: [{ id: 'tag-1', name: '核心' }],
          ...extraProps,
          'onUpdate:search': (value: string) => emissions.search.push(value),
          'onUpdate:viewMode': (value: string) => emissions.viewMode.push(value),
          'onUpdate:sortBy': (value: string) => emissions.sortBy.push(value),
          'onUpdate:sortOrder': (value: string) => emissions.sortOrder.push(value),
          onOpenGroupManager: () => {
            emissions.openGroupManager += 1
          },
          'onUpdate:compositeFilters': (value: {
            runtimeModes: string[]
            deployStatuses: string[]
            createdBy: string
          }) => emissions.compositeFilters.push(value),
        })
    },
  })

  const app = createApp(Root)
  app.use(i18n)
  registerElementStubs(app)
  app.mount(container)

  const result: MountResult = {
    container,
    emissions,
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
    const instance = mountedInstances.pop()
    instance?.unmount()
  }
})

describe('project-overview-toolbar', () => {
  test('左侧存在排序控件与独立标签筛选入口，且不出现运维中心', () => {
    const { container } = mountToolbar()
    expect(container.querySelector('[data-testid="overview-toolbar-left"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="overview-sort-field"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="overview-sort-order"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="overview-tag-filter"]')).not.toBeNull()
    expect(container.textContent || '').not.toContain('运维中心')
  })

  test('关键交互会触发 update emits', async () => {
    const { container, emissions } = mountToolbar()
    const searchInput = container.querySelector('[data-testid="overview-search-input"]') as HTMLInputElement
    searchInput.value = '工厂中台'
    searchInput.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()

    const listButton = container.querySelector('[data-testid="overview-view-list"]') as HTMLButtonElement
    listButton.click()
    await nextTick()

    const sortBySelect = container.querySelector('[data-testid="overview-sort-field"]') as HTMLSelectElement
    sortBySelect.value = 'updatedAt'
    sortBySelect.dispatchEvent(new Event('change', { bubbles: true }))
    await nextTick()

    const sortOrderSelect = container.querySelector('[data-testid="overview-sort-order"]') as HTMLSelectElement
    sortOrderSelect.value = 'ASC'
    sortOrderSelect.dispatchEvent(new Event('change', { bubbles: true }))
    await nextTick()

    expect(emissions.search).toContain('工厂中台')
    expect(emissions.viewMode).toContain('list')
    expect(emissions.sortBy).toContain('updatedAt')
    expect(emissions.sortOrder).toContain('ASC')

    const createdByInput = container.querySelector(
      'input[placeholder="输入创建人账号或姓名"]',
    ) as HTMLInputElement | null
    expect(createdByInput).not.toBeNull()
    if (!createdByInput) {
      throw new Error('未找到综合筛选创建人输入框')
    }
    createdByInput.value = '张三'
    createdByInput.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()

    const applyButton = [...container.querySelectorAll('button')].find((button) =>
      (button.textContent || '').includes('应用筛选'),
    ) as HTMLButtonElement | undefined
    expect(applyButton).toBeDefined()
    if (!applyButton) {
      throw new Error('未找到综合筛选应用按钮')
    }
    applyButton.click()
    await nextTick()

    const latestCompositeFilters = emissions.compositeFilters.at(-1)
    expect(latestCompositeFilters).toEqual(
      expect.objectContaining({
        createdBy: '张三',
      }),
    )
  })

  test('右侧固定图标按顺序展示，未选中时不展示批量操作入口', () => {
    const { container } = mountToolbar()
    const rightToolbar = container.querySelector('.project-overview-toolbar__right')
    expect(rightToolbar).not.toBeNull()

    const triggers = [...rightToolbar!.querySelectorAll('[data-testid]')].map((element) =>
      element.getAttribute('data-testid'),
    )

    expect(triggers).toEqual([
      'project-add-trigger',
      'project-refresh-trigger',
      'project-group-manage-trigger',
      'project-import-trigger',
      'project-settings-trigger',
    ])
    expect(container.querySelector('[data-testid="project-batch-export-trigger"]')).toBeNull()
    expect(container.querySelector('[data-testid="project-batch-delete-trigger"]')).toBeNull()
  })

  test('选中工程后展示批量导出和批量删除入口', () => {
    const { container } = mountToolbar({ selectedCount: 2 })

    expect(container.querySelector('[data-testid="project-batch-export-trigger"]')).not.toBeNull()
    expect(container.querySelector('[data-testid="project-batch-delete-trigger"]')).not.toBeNull()
  })

  test('点击分组管理会触发 open-group-manager emit', async () => {
    const { container, emissions } = mountToolbar()

    const groupManageButton = container.querySelector(
      '[data-testid="project-group-manage-trigger"]',
    ) as HTMLButtonElement
    groupManageButton.click()
    await nextTick()

    expect(emissions.openGroupManager).toBe(1)
  })

  test('右侧纯图标按钮提供稳定的 accessible name', () => {
    const { container } = mountToolbar({ selectedCount: 1 })

    const expectedLabels = {
      'project-batch-export-trigger': i18n.global.t('projectManagement.batchExport'),
      'project-batch-delete-trigger': i18n.global.t('projectManagement.batchDelete'),
      'project-add-trigger': i18n.global.t('projectManagement.addProject'),
      'project-refresh-trigger': i18n.global.t('common.refresh'),
      'project-group-manage-trigger': i18n.global.t('projectManagement.groupManagement'),
      'project-import-trigger': i18n.global.t('projectManagement.importProject'),
      'project-settings-trigger': i18n.global.t('projectManagement.settings'),
    }

    Object.entries(expectedLabels).forEach(([testId, label]) => {
      expect(container.querySelector(`[data-testid="${testId}"]`)?.getAttribute('aria-label')).toBe(
        label,
      )
    })
  })
})
