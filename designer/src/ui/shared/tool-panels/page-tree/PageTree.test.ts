import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { i18n } from '@/i18n'
import PageTree from './PageTree.vue'

const mocks = vi.hoisted(() => {
  const createRef = <T>(value: T) => ({
    value,
    __v_isRef: true,
  })
  const messageSuccess = vi.fn()
  const messageWarning = vi.fn()
  const messageError = vi.fn()
  const storeState = {
    pages: createRef([] as any[]),
    currentPageId: createRef(''),
    entryConfig: createRef({} as Record<string, unknown>),
    canUndo: createRef(false),
    projectId: createRef('project-1'),
  }
  const editorStoreMock = {
    ...storeState,
    loadPage: vi.fn(),
    saveCurrentPage: vi.fn(),
    createHomePage: vi.fn(),
    buildNewPageSchema: vi.fn(),
    createPage: vi.fn(),
    saveEntryPatch: vi.fn(),
    refreshPages: vi.fn(),
    updateEntry: vi.fn(),
    persistEntry: vi.fn(),
    setCurrentPage: vi.fn(),
  }

  return {
    messageSuccess,
    messageWarning,
    messageError,
    storeState,
    editorStoreMock,
  }
})

vi.mock('pinia', () => ({
  storeToRefs: (store: typeof mocks.editorStoreMock) => ({
    pages: store.pages,
    currentPageId: store.currentPageId,
    entryConfig: store.entryConfig,
    canUndo: store.canUndo,
    projectId: store.projectId,
  }),
}))

vi.mock('@/stores/editor-store', () => ({
  useEditorStore: () => mocks.editorStoreMock,
}))

vi.mock('element-plus', () => {
  const messageBox = vi.fn() as any
  messageBox.confirm = vi.fn()
  return {
    ElCheckbox: {
      name: 'ElCheckbox',
      template: `<label><slot /></label>`,
    },
    ElMessage: {
      success: mocks.messageSuccess,
      warning: mocks.messageWarning,
      error: mocks.messageError,
    },
    ElMessageBox: messageBox,
  }
})

const PageTreeBasicPagesSectionStub = {
  name: 'PageTreeBasicPagesSection',
  emits: ['slotClick', 'slotDblClick', 'rowAction'],
  template: `<div class="basic-section-stub" />`,
}

const PageTreePageSearchStub = {
  name: 'PageTreePageSearch',
  props: ['modelValue'],
  emits: ['update:modelValue'],
  template: `<div class="page-search-stub" />`,
}

const PageTreeCreateDialogStub = {
  name: 'PageTreeCreateDialog',
  props: ['modelValue'],
  emits: ['update:modelValue', 'update:form', 'confirm'],
  template: `<div class="page-create-dialog-stub" />`,
}

describe('page tree', () => {
  beforeEach(() => {
    mocks.storeState.pages.value = []
    mocks.storeState.currentPageId.value = ''
    mocks.storeState.entryConfig.value = {}
    mocks.storeState.canUndo.value = false
    mocks.storeState.projectId.value = 'project-1'
    mocks.editorStoreMock.createHomePage.mockReset()
    mocks.editorStoreMock.buildNewPageSchema.mockReset()
    mocks.editorStoreMock.createPage.mockReset()
    mocks.editorStoreMock.saveEntryPatch.mockReset()
    mocks.editorStoreMock.refreshPages.mockReset()
    mocks.editorStoreMock.updateEntry.mockReset()
    mocks.editorStoreMock.persistEntry.mockReset()
    mocks.editorStoreMock.setCurrentPage.mockReset()
    mocks.messageSuccess.mockReset()
    mocks.messageWarning.mockReset()
    mocks.messageError.mockReset()
  })

  it('基础页面菜单 create 会创建登录页并持久化入口配置', async () => {
    mocks.editorStoreMock.buildNewPageSchema.mockReturnValue({ schema: 'login' })
    mocks.editorStoreMock.createPage.mockResolvedValue({ id: 'login-page-id' })
    mocks.editorStoreMock.saveEntryPatch.mockResolvedValue(undefined)
    mocks.editorStoreMock.refreshPages.mockResolvedValue(undefined)
    mocks.editorStoreMock.setCurrentPage.mockResolvedValue(undefined)

    const wrapper = mount(PageTree, {
      global: {
        plugins: [i18n],
        stubs: {
          PageTreeBasicPagesSection: PageTreeBasicPagesSectionStub,
          PageTreePageSearch: PageTreePageSearchStub,
          PageTreeCreateDialog: PageTreeCreateDialogStub,
          ElDropdown: true,
          ElDropdownMenu: true,
          ElDropdownItem: true,
          ElButton: true,
        },
      },
    })

    const basicSection = wrapper.getComponent({ name: 'PageTreeBasicPagesSection' })
    basicSection.vm.$emit('rowAction', 'create', {
      type: 'login',
      label: '登录页',
      page: null,
    })
    await wrapper.vm.$nextTick()
    await Promise.resolve()
    await Promise.resolve()

    expect(mocks.editorStoreMock.buildNewPageSchema).toHaveBeenCalledWith({
      name: '登录页',
      path: '/login',
    })
    expect(mocks.editorStoreMock.createPage).toHaveBeenCalledWith({
      name: '登录页',
      type: 'page',
      parentId: null,
      path: '/login',
      schemaContent: { schema: 'login' },
    })
    expect(mocks.editorStoreMock.saveEntryPatch).toHaveBeenCalledWith({
      loginPageId: 'login-page-id',
    })
    expect(mocks.editorStoreMock.refreshPages).toHaveBeenCalledTimes(1)
    expect(mocks.editorStoreMock.setCurrentPage).toHaveBeenCalledWith('login-page-id')
  })

  it('登出页已存在时只提示，不重复创建', async () => {
    mocks.storeState.entryConfig.value = { logoutPageId: 'logout-1' }
    mocks.storeState.pages.value = [
      {
        id: 'logout-1',
        path: '/logout',
      },
    ]

    const wrapper = mount(PageTree, {
      global: {
        plugins: [i18n],
        stubs: {
          PageTreeBasicPagesSection: PageTreeBasicPagesSectionStub,
          PageTreePageSearch: PageTreePageSearchStub,
          PageTreeCreateDialog: PageTreeCreateDialogStub,
          ElDropdown: true,
          ElDropdownMenu: true,
          ElDropdownItem: true,
          ElButton: true,
        },
      },
    })

    const basicSection = wrapper.getComponent({ name: 'PageTreeBasicPagesSection' })
    basicSection.vm.$emit('rowAction', 'create', {
      type: 'logout',
      label: '登出页',
      page: null,
    })
    await wrapper.vm.$nextTick()
    await Promise.resolve()

    expect(mocks.editorStoreMock.createPage).not.toHaveBeenCalled()
    expect(mocks.editorStoreMock.saveEntryPatch).not.toHaveBeenCalled()
    expect(mocks.messageWarning).toHaveBeenCalledWith({ message: '登出页已存在' })
  })

  it('entryConfig 残留旧 ID 且页面不存在时允许重新创建登录页', async () => {
    mocks.storeState.entryConfig.value = { loginPageId: 'stale-login-id' }
    mocks.storeState.pages.value = []
    mocks.editorStoreMock.buildNewPageSchema.mockReturnValue({ schema: 'login' })
    mocks.editorStoreMock.createPage.mockResolvedValue({ id: 'new-login-id' })
    mocks.editorStoreMock.saveEntryPatch.mockResolvedValue(undefined)
    mocks.editorStoreMock.refreshPages.mockResolvedValue(undefined)
    mocks.editorStoreMock.setCurrentPage.mockResolvedValue(undefined)

    const wrapper = mount(PageTree, {
      global: {
        plugins: [i18n],
        stubs: {
          PageTreeBasicPagesSection: PageTreeBasicPagesSectionStub,
          PageTreePageSearch: PageTreePageSearchStub,
          PageTreeCreateDialog: PageTreeCreateDialogStub,
          ElDropdown: true,
          ElDropdownMenu: true,
          ElDropdownItem: true,
          ElButton: true,
        },
      },
    })

    const basicSection = wrapper.getComponent({ name: 'PageTreeBasicPagesSection' })
    basicSection.vm.$emit('rowAction', 'create', {
      type: 'login',
      label: '登录页',
      page: null,
    })
    await wrapper.vm.$nextTick()
    await Promise.resolve()
    await Promise.resolve()

    expect(mocks.editorStoreMock.createPage).toHaveBeenCalledWith({
      name: '登录页',
      type: 'page',
      parentId: null,
      path: '/login',
      schemaContent: { schema: 'login' },
    })
    expect(mocks.editorStoreMock.saveEntryPatch).toHaveBeenCalledWith({
      loginPageId: 'new-login-id',
    })
    expect(mocks.messageWarning).not.toHaveBeenCalled()
  })
})
