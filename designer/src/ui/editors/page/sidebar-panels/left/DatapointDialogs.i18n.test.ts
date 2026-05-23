import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach, describe, expect, it } from 'vitest'
import { i18n } from '@/i18n'
import DatapointQuickAddDialog from './DatapointQuickAddDialog.vue'
import DatapointVariableEditDialog from './DatapointVariableEditDialog.vue'

const sharedStubs = {
  'el-dialog': {
    props: ['title'],
    template: '<div><header>{{ title }}</header><slot /><slot name="footer" /></div>',
  },
  'el-form': {
    template: '<form><slot /></form>',
  },
  'el-form-item': {
    props: ['label'],
    template: '<label>{{ label }}<slot /></label>',
  },
  'el-row': { template: '<div><slot /></div>' },
  'el-col': { template: '<div><slot /></div>' },
  'el-input': {
    props: ['placeholder'],
    template: '<div class="el-input-stub" :data-placeholder="placeholder"><slot /></div>',
  },
  'el-table': { template: '<table><slot /></table>' },
  'el-table-column': {
    props: ['label'],
    template: '<th>{{ label }}</th>',
  },
  'el-tag': { template: '<span><slot /></span>' },
  'el-pagination': { template: '<div>pagination</div>' },
  'el-select': {
    props: ['placeholder'],
    template: '<div class="el-select-stub" :data-placeholder="placeholder"><slot /></div>',
  },
  'el-option': {
    props: ['label'],
    template: '<option>{{ label }}</option>',
  },
  'el-input-number': { template: '<input type="number" />' },
  'el-switch': { template: '<input type="checkbox" />' },
  'el-date-picker': { template: '<input type="datetime-local" />' },
  MonacoEditor: { template: '<div>monaco</div>' },
  'el-button': { template: '<button><slot /></button>' },
  'el-icon': { template: '<span><slot /></span>' },
}

describe('Datapoint dialogs i18n', () => {
  afterEach(() => {
    i18n.global.locale.value = 'zh'
  })

  it('快速添加对话框会随 locale 切换更新文案', async () => {
    const wrapper = mount(DatapointQuickAddDialog, {
      props: {
        modelValue: true,
        filteredFields: [],
        quickPageSize: 50,
        quickTotal: 0,
        quickPage: 1,
        buildVarName: (name: string) => name,
      },
      global: {
        plugins: [i18n],
        directives: { loading: {} },
        stubs: sharedStubs,
      },
    })

    expect(wrapper.text()).toContain('快速添加数据点')
    expect(wrapper.text()).toContain('批量命名规则')
    expect(wrapper.text()).toContain('变量名')
    expect(wrapper.html()).toContain('搜索名称或路径')

    i18n.global.locale.value = 'en'
    await nextTick()

    expect(wrapper.text()).toContain('Quick Add Datapoints')
    expect(wrapper.text()).toContain('Batch Naming Rule')
    expect(wrapper.text()).toContain('Variable Name')
    expect(wrapper.html()).toContain('Search by name or path')
  })

  it('变量编辑对话框会随 locale 切换更新文案', async () => {
    const wrapper = mount(DatapointVariableEditDialog, {
      props: {
        modelValue: true,
        editMode: false,
        types: ['string'],
        groupOptions: [],
        rootGroupId: '__root__',
      },
      global: {
        plugins: [i18n],
        directives: { loading: {} },
        stubs: sharedStubs,
      },
    })

    expect(wrapper.text()).toContain('新增变量')
    expect(wrapper.text()).toContain('变量名')
    expect(wrapper.text()).toContain('初始值')
    expect(wrapper.html()).toContain('请选择分组')

    i18n.global.locale.value = 'en'
    await nextTick()

    expect(wrapper.text()).toContain('New Variable')
    expect(wrapper.text()).toContain('Variable Name')
    expect(wrapper.text()).toContain('Initial Value')
    expect(wrapper.html()).toContain('Select a group')
  })
})
