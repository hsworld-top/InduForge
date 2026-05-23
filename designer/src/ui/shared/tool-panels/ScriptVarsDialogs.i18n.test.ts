import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach, describe, expect, it } from 'vitest'
import { i18n } from '@/i18n'
import ScriptVarsMetaFormDialog from './ScriptVarsMetaFormDialog.vue'
import ScriptVarsSystemScriptDialog from './ScriptVarsSystemScriptDialog.vue'

const sharedStubs = {
  'el-dialog': {
    props: ['title'],
    template: '<div><header>{{ title }}</header><slot /><slot name="footer" /></div>',
  },
  'el-form': { template: '<form><slot /></form>' },
  'el-form-item': {
    props: ['label'],
    template: '<label>{{ label }}<slot /></label>',
  },
  'el-input': {
    props: ['placeholder'],
    template: '<div class="el-input-stub" :data-placeholder="placeholder"><slot /></div>',
  },
  'el-select': {
    props: ['placeholder'],
    template: '<div class="el-select-stub" :data-placeholder="placeholder"><slot /></div>',
  },
  'el-option': { props: ['label'], template: '<option>{{ label }}</option>' },
  'el-input-number': { template: '<input type="number" />' },
  'el-tooltip': {
    props: ['content'],
    template: '<div><slot />{{ content }}</div>',
  },
  'el-tabs': {
    template: '<div><slot /></div>',
  },
  'el-tab-pane': {
    props: ['label'],
    template: '<div>{{ label }}</div>',
  },
  'el-tree': { template: "<div><slot :data=\"{ type: 'group', label: 'node' }\" /></div>" },
  MonacoEditor: { template: '<div>monaco</div>' },
  'el-button': { template: '<button><slot /></button>' },
  'el-icon': { template: '<span><slot /></span>' },
  IconEpFolder: true,
  IconEpEditPen: true,
  IconEpDocument: true,
  IconEpList: true,
}

describe('Script dialogs i18n', () => {
  afterEach(() => {
    i18n.global.locale.value = 'zh'
  })

  it('脚本元信息对话框会随 locale 切换更新文案', async () => {
    const wrapper = mount(ScriptVarsMetaFormDialog, {
      props: {
        modelValue: true,
        title: '新建定时器',
        module: 'custom',
        meta: {
          name: '',
          params: '',
          description: '',
        },
      },
      global: {
        plugins: [i18n],
        stubs: sharedStubs,
      },
    })

    expect(wrapper.text()).toContain('函数名称')
    expect(wrapper.text()).toContain('入参')
    expect(wrapper.html()).toContain('例如: id, value')

    i18n.global.locale.value = 'en'
    await nextTick()

    expect(wrapper.text()).toContain('Function Name')
    expect(wrapper.text()).toContain('Params')
    expect(wrapper.html()).toContain('For example: id, value')
  })

  it('系统脚本编辑器会随 locale 切换更新侧栏与按钮文案', async () => {
    const wrapper = mount(ScriptVarsSystemScriptDialog, {
      props: {
        modelValue: true,
        dialogTitle: 'System Startup Script',
        metaTitle: 'System Startup',
        customScriptSidebarTree: [],
        pageSidebarTree: [],
        filterSidebarNode: () => true,
      },
      global: {
        plugins: [i18n],
        stubs: sharedStubs,
      },
    })

    expect(wrapper.text()).toContain('自定义脚本')
    expect(wrapper.text()).toContain('页面')
    expect(wrapper.text()).toContain('保存 (Ctrl+S)')

    i18n.global.locale.value = 'en'
    await nextTick()

    expect(wrapper.text()).toContain('Custom Scripts')
    expect(wrapper.text()).toContain('Pages')
    expect(wrapper.text()).toContain('Save (Ctrl+S)')
  })
})
