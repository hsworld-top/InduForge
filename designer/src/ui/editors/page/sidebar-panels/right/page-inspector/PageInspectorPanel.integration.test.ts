import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { i18n } from '@/i18n'
import PageInspectorPanel from '../PageInspectorPanel.vue'

const currentPage = ref<any>(null)
const currentPageId = ref('')
const pages = ref<any[]>([])
const doc = ref<any>(null)
const renamePage = vi.fn()
const saveEntryPatch = vi.fn()
const updateCurrentPage = vi.fn()

vi.mock('pinia', () => ({
  storeToRefs: (store: any) => store.__refs,
}))

vi.mock('@/stores/editor-store', () => ({
  useEditorStore: () => ({
    __refs: {
      currentPage,
      currentPageId,
      pages,
      doc,
    },
    renamePage,
    saveEntryPatch,
    updateCurrentPage,
  }),
}))

vi.mock('@/ui/shared/widgets/base/FriendlyColorPicker.vue', () => ({
  default: {
    props: ['modelValue'],
    template: "<div class='color-picker-stub'>{{ modelValue }}</div>",
  },
}))

const sharedStubs = {
  'el-tooltip': {
    props: ['content'],
    template: "<span class='tooltip-stub'><slot />{{ content }}</span>",
  },
  'el-input': {
    inheritAttrs: false,
    props: ['modelValue', 'disabled', 'placeholder', 'size'],
    template:
      "<input :value='modelValue' :disabled='disabled' :placeholder='placeholder' :data-testid=\"$attrs['data-testid']\" />",
  },
  'el-select': {
    inheritAttrs: false,
    props: ['modelValue', 'disabled', 'size'],
    template: "<select :disabled='disabled'><slot /></select>",
  },
  'el-option': {
    template: '<option><slot /></option>',
  },
  'el-input-number': {
    props: ['modelValue', 'disabled', 'size'],
    template: "<input type='number' :value='modelValue' :disabled='disabled' />",
  },
  'el-switch': {
    props: ['modelValue', 'disabled'],
    template: "<input type='checkbox' :checked='modelValue' :disabled='disabled' />",
  },
  'el-button': {
    props: ['disabled', 'size'],
    template: "<button type='button' :disabled='disabled'><slot /></button>",
  },
}

describe('PageInspectorPanel integration', () => {
  beforeEach(() => {
    currentPageId.value = 'page-home'
    currentPage.value = {
      id: 'page-home',
      name: '首页',
      path: '/',
      config: {
        meta: {
          title: '首页标题',
          description: '首页描述',
        },
        route: {
          mode: 'auto',
          path: '/',
          slug: '',
        },
        viewport: {
          preset: 'pc',
          width: 1366,
          height: 768,
          autoFit: true,
          lockAspectRatio: false,
          minWidth: 0,
          minHeight: 0,
          overflowMode: 'auto',
        },
        background: {
          kind: 'color',
          value: '#ffffff',
        },
        runtime: {
          openMode: 'replace',
          permission: {
            summary: '0item',
          },
        },
        width: 1366,
        height: 768,
        autoFit: true,
      },
    }
    pages.value = [
      {
        id: 'page-home',
        name: '首页',
        path: '/',
      },
    ]
    doc.value = {
      entry: {
        homePageId: 'page-home',
      },
    }
    renamePage.mockReset()
    saveEntryPatch.mockReset()
    updateCurrentPage.mockReset()
  })

  afterEach(() => {
    i18n.global.locale.value = 'zh'
  })

  it('renders the five-section page inspector and locks route path for system pages', () => {
    const wrapper = mount(PageInspectorPanel, {
      global: {
        plugins: [i18n],
        stubs: sharedStubs,
      },
    })

    expect(wrapper.text()).toContain('页面身份')
    expect(wrapper.text()).toContain('路由与入口')
    expect(wrapper.text()).toContain('布局与适配')
    expect(wrapper.text()).toContain('视觉')
    expect(wrapper.text()).toContain('运行控制')
    expect(wrapper.text()).toContain('页面角色由页面树或工程入口配置决定')
    expect(wrapper.text()).toContain('基础页面固定使用替换式打开')

    const routeInput = wrapper.get("[data-testid='route-path-input']")
    expect(routeInput.attributes('disabled')).toBeDefined()
  })
})
