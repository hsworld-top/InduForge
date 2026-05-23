import { afterEach, describe, expect, test } from 'vitest'
import { createApp, defineComponent, h, nextTick, type App } from 'vue'
import i18n from '@/lang'
import ProjectGroupProjectPickerDialog from '@/views/tenant/project-management/ProjectGroupProjectPickerDialog.vue'

type MountResult = {
  app: App<Element>
  container: HTMLDivElement
  emitted: Record<string, unknown[][]>
  unmount: () => void
}

const mountedInstances: MountResult[] = []

const ElementContainerStub = defineComponent({
  setup(_, { slots, attrs }) {
    return () => h('div', attrs, slots.default?.())
  },
})

const ElDialogStub = defineComponent({
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
        ? h('section', { 'data-testid': 'picker-dialog' }, [
            h('h2', props.title),
            slots.default?.(),
            slots.footer?.(),
          ])
        : null
  },
})

const ElInputStub = defineComponent({
  props: {
    modelValue: {
      type: String,
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, attrs }) {
    return () =>
      h('input', {
        ...attrs,
        value: props.modelValue,
        onInput: (event: Event) => {
          emit('update:modelValue', (event.target as HTMLInputElement).value)
        },
      })
  },
})

const ElSelectStub = defineComponent({
  props: {
    modelValue: {
      type: [String, Array],
      default: '',
    },
    multiple: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, slots, attrs }) {
    return () =>
      h(
        'select',
        {
          ...attrs,
          multiple: props.multiple,
          value: props.multiple ? undefined : String(props.modelValue ?? ''),
          onChange: (event: Event) => {
            const target = event.target as HTMLSelectElement
            if (props.multiple) {
              emit(
                'update:modelValue',
                [...target.selectedOptions].map((option) => option.value),
              )
              return
            }
            emit('update:modelValue', target.value)
          },
        },
        slots.default?.(),
      )
  },
})

const ElOptionStub = defineComponent({
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

const ElButtonStub = defineComponent({
  setup(_, { slots, attrs }) {
    return () => h('button', { ...attrs, type: 'button' }, slots.default?.())
  },
})

const registerElementStubs = (app: App) => {
  app.component('ElDialog', ElDialogStub)
  app.component('el-dialog', ElDialogStub)
  app.component('ElInput', ElInputStub)
  app.component('el-input', ElInputStub)
  app.component('ElSelect', ElSelectStub)
  app.component('el-select', ElSelectStub)
  app.component('ElOption', ElOptionStub)
  app.component('el-option', ElOptionStub)
  app.component('ElButton', ElButtonStub)
  app.component('el-button', ElButtonStub)
  app.component('ElTag', ElementContainerStub)
  app.component('el-tag', ElementContainerStub)
  app.component('ElEmpty', ElementContainerStub)
  app.component('el-empty', ElementContainerStub)
  app.directive('loading', {})
}

const createRuntimeSummary = ({
  runtimeStatus = 'running',
  devCount = 0,
  releaseCount = 0,
}: {
  runtimeStatus?: string
  devCount?: number
  releaseCount?: number
}) => ({
  runtimeStatus,
  deploymentCount: devCount + releaseCount,
  runningCount: runtimeStatus === 'running' ? 1 : 0,
  statusCounts: {
    pending: runtimeStatus === 'pending' ? 1 : 0,
    deploying: runtimeStatus === 'deploying' ? 1 : 0,
    running: runtimeStatus === 'running' ? 1 : 0,
    stopped: runtimeStatus === 'stopped' ? 1 : 0,
    error: runtimeStatus === 'error' ? 1 : 0,
    rollback: runtimeStatus === 'rollback' ? 1 : 0,
  },
  modeCounts: {
    DEV: devCount,
    RELEASE: releaseCount,
  },
  lastDeployedAt: null,
})

const mountDialog = async (): Promise<MountResult> => {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const emitted: Record<string, unknown[][]> = {}

  const Root = defineComponent({
    setup() {
      const capture =
        (eventName: string) =>
        (...args: unknown[]) => {
          emitted[eventName] = emitted[eventName] || []
          emitted[eventName].push(args)
        }
      return () =>
        h(ProjectGroupProjectPickerDialog, {
          visible: true,
          groupId: 'group-current',
          groupName: '目标分组',
          projects: [
            {
              id: 'project-1',
              name: '候选一号',
              description: '生产线核心工程',
              tags: [{ id: 'tag-core', name: '核心' }],
              group: null,
              runtimeSummary: createRuntimeSummary({ runtimeStatus: 'running', devCount: 1 }),
            },
            {
              id: 'project-2',
              name: '候选二号',
              description: '边缘发布工程',
              tags: [{ id: 'tag-edge', name: '边缘' }],
              group: null,
              runtimeSummary: createRuntimeSummary({ runtimeStatus: 'stopped', releaseCount: 1 }),
            },
            {
              id: 'project-3',
              name: '已在目标分组',
              description: '',
              tags: [{ id: 'tag-core', name: '核心' }],
              group: { id: 'group-current', name: '目标分组' },
              runtimeSummary: createRuntimeSummary({ runtimeStatus: 'running', devCount: 1 }),
            },
          ],
          tagOptions: [
            { id: 'tag-core', name: '核心' },
            { id: 'tag-edge', name: '边缘' },
          ],
          runtimeModeOptions: [
            { label: '开发模式', value: 'DEV' },
            { label: '生产模式', value: 'RELEASE' },
          ],
          deployStatusOptions: [
            { label: '运行中', value: 'running' },
            { label: '已停止', value: 'stopped' },
          ],
          'onUpdate:visible': capture('update:visible'),
          onSelect: capture('select'),
          onQueryChange: capture('query-change'),
        })
    },
  })

  const app = createApp(Root)
  app.use(i18n)
  registerElementStubs(app)
  app.mount(container)
  await nextTick()

  const result: MountResult = {
    app,
    container,
    emitted,
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
})

describe('ProjectGroupProjectPickerDialog', () => {
  test('支持搜索和标签筛选，并向父页面同步查询条件', async () => {
    const { container, emitted } = await mountDialog()

    const searchInput = container.querySelector(
      '[data-testid="project-group-picker-search"]',
    ) as HTMLInputElement
    searchInput.value = '二号'
    searchInput.dispatchEvent(new Event('input'))
    await nextTick()

    expect(container.textContent || '').toContain('候选二号')
    expect(container.textContent || '').not.toContain('候选一号')
    expect(emitted['query-change'].at(-1)?.[0]).toMatchObject({ search: '二号' })

    const tagSelect = container.querySelector(
      '[data-testid="project-group-picker-tag-filter"]',
    ) as HTMLSelectElement
    ;[...tagSelect.options].forEach((option) => {
      option.selected = option.value === 'tag-edge'
    })
    tagSelect.dispatchEvent(new Event('change'))
    await nextTick()

    expect(emitted['query-change'].at(-1)?.[0]).toMatchObject({
      search: '二号',
      tagIds: ['tag-edge'],
    })
  })

  test('支持多选工程后一次性提交所选工程 ID', async () => {
    const { container, emitted } = await mountDialog()

    const checkboxes = [
      ...container.querySelectorAll('[data-testid="project-group-picker-checkbox"]'),
    ] as HTMLInputElement[]
    expect(checkboxes).toHaveLength(2)

    checkboxes[0].checked = true
    checkboxes[0].dispatchEvent(new Event('change'))
    checkboxes[1].checked = true
    checkboxes[1].dispatchEvent(new Event('change'))
    await nextTick()

    const submitButton = container.querySelector(
      '[data-testid="project-group-picker-submit"]',
    ) as HTMLButtonElement
    submitButton.click()
    await nextTick()

    expect(emitted.select).toEqual([[['project-1', 'project-2']]])
  })
})
