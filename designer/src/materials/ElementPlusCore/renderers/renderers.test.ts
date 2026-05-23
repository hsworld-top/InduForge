import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import CheckboxRenderer from './CheckboxRenderer.vue'
import InputNumberRenderer from './InputNumberRenderer.vue'
import PaginationRenderer from './PaginationRenderer.vue'
import RadioRenderer from './RadioRenderer.vue'
import SelectRenderer from './SelectRenderer.vue'
import SwitchRenderer from './SwitchRenderer.vue'
import TableRenderer from './TableRenderer.vue'

const globalStubs = {
  ElSelect: {
    inheritAttrs: false,
    emits: ['update:modelValue'],
    template:
      '<div class="el-select" @click="$emit(\'update:modelValue\', \'stopped\')"><slot /></div>',
  },
  ElOption: {
    props: ['label', 'value', 'disabled'],
    template:
      '<div class="el-option" :data-value="String(value)" :data-disabled="String(disabled)">{{ label }}</div>',
  },
  ElRadioGroup: {
    inheritAttrs: false,
    emits: ['update:modelValue'],
    template:
      '<div class="el-radio-group" @click="$emit(\'update:modelValue\', \'option2\')"><slot /></div>',
  },
  ElRadio: {
    props: ['label', 'disabled'],
    template:
      '<label class="el-radio" :data-value="String(label)" :data-disabled="String(disabled)"><slot /></label>',
  },
  ElRadioButton: {
    props: ['label', 'disabled'],
    template:
      '<label class="el-radio-button" :data-value="String(label)" :data-disabled="String(disabled)"><slot /></label>',
  },
  ElCheckboxGroup: {
    inheritAttrs: false,
    emits: ['update:modelValue'],
    template:
      '<div class="el-checkbox-group" @click="$emit(\'update:modelValue\', [\'b\'])"><slot /></div>',
  },
  ElCheckbox: {
    props: ['label', 'disabled'],
    template:
      '<label class="el-checkbox" :data-value="String(label)" :data-disabled="String(disabled)"><slot /></label>',
  },
  ElCheckboxButton: {
    props: ['label', 'disabled'],
    template:
      '<label class="el-checkbox-button" :data-value="String(label)" :data-disabled="String(disabled)"><slot /></label>',
  },
  ElInputNumber: {
    emits: ['update:modelValue'],
    template: '<div class="el-input-number"><slot /></div>',
  },
  ElSwitch: {
    emits: ['update:modelValue'],
    template: '<div class="el-switch"><slot /></div>',
  },
  ElTable: {
    props: ['data'],
    template: '<div class="el-table" :data-rows="data.length"><slot /></div>',
  },
  ElTableColumn: {
    props: ['prop', 'label', 'width', 'align'],
    template:
      '<div class="el-table-column" :data-prop="prop" :data-width="String(width)" :data-align="align">{{ label }}</div>',
  },
}

describe('ElementPlusCore renderers', () => {
  it('emits model updates from input number controls', async () => {
    const wrapper = mount(InputNumberRenderer, {
      props: {
        resolvedProps: {
          modelValue: 2,
          min: 0,
          max: 5,
          step: 2,
        },
      },
      global: { stubs: globalStubs },
    })

    await wrapper.find('.core-input-number__hit--increase').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([4])

    await wrapper.setProps({ resolvedProps: { modelValue: 0, min: 0, max: 5, step: 2 } })
    await wrapper.find('.core-input-number__hit--decrease').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([0])
  })

  it('emits model updates from switch overlay', async () => {
    const wrapper = mount(SwitchRenderer, {
      props: {
        resolvedProps: {
          modelValue: true,
        },
      },
      global: { stubs: globalStubs },
    })

    await wrapper.find('.core-switch__hit').trigger('click')

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([false])
  })

  it('renders select options from configurable data', () => {
    const wrapper = mount(SelectRenderer, {
      props: {
        resolvedProps: {
          modelValue: 'running',
          options: [
            { label: '运行', value: 'running' },
            { label: '停机', value: 'stopped', disabled: true },
          ],
        },
      },
      global: { stubs: globalStubs },
    })

    const options = wrapper.findAll('.el-option')
    expect(options).toHaveLength(2)
    expect(wrapper.text()).toContain('运行')
    expect(options.at(1)?.attributes('data-disabled')).toBe('true')
  })

  it('emits model updates from select interactions', async () => {
    const wrapper = mount(SelectRenderer, {
      props: {
        resolvedProps: {
          modelValue: 'running',
          options: [
            { label: '运行', value: 'running' },
            { label: '停机', value: 'stopped' },
          ],
        },
      },
      global: { stubs: globalStubs },
    })

    await wrapper.find('.el-select').trigger('click')

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['stopped'])
  })

  it('switches radio renderer between normal and button style', () => {
    const wrapper = mount(RadioRenderer, {
      props: {
        resolvedProps: {
          modelValue: 'option1',
          buttonStyle: true,
          options: ['选项一', '选项二'],
        },
      },
      global: { stubs: globalStubs },
    })

    expect(wrapper.findAll('.el-radio-button')).toHaveLength(2)
    expect(wrapper.find('.el-radio').exists()).toBe(false)
  })

  it('emits model updates from radio interactions', async () => {
    const wrapper = mount(RadioRenderer, {
      props: {
        resolvedProps: {
          modelValue: 'option1',
          options: ['选项一', '选项二'],
        },
      },
      global: { stubs: globalStubs },
    })

    await wrapper.find('.el-radio-group').trigger('click')

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['option2'])
  })

  it('switches checkbox renderer between normal and button style', () => {
    const wrapper = mount(CheckboxRenderer, {
      props: {
        resolvedProps: {
          modelValue: ['option1'],
          buttonStyle: false,
          options: [
            { label: 'A类', value: 'a' },
            { label: 'B类', value: 'b' },
          ],
        },
      },
      global: { stubs: globalStubs },
    })

    expect(wrapper.findAll('.el-checkbox')).toHaveLength(2)
    expect(wrapper.find('.el-checkbox-button').exists()).toBe(false)
    expect(wrapper.text()).toContain('A类')
  })

  it('emits model updates from checkbox interactions', async () => {
    const wrapper = mount(CheckboxRenderer, {
      props: {
        resolvedProps: {
          modelValue: ['a'],
          options: [
            { label: 'A类', value: 'a' },
            { label: 'B类', value: 'b' },
          ],
        },
      },
      global: { stubs: globalStubs },
    })

    await wrapper.find('.el-checkbox-group').trigger('click')

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([['b']])
  })

  it('renders table columns and rows from configurable data', () => {
    const wrapper = mount(TableRenderer, {
      props: {
        resolvedProps: {
          columns: [
            { prop: 'name', label: '名称', width: 120 },
            { prop: 'status', label: '状态', align: 'center' },
          ],
          data: [
            { name: '设备A', status: '运行' },
            { name: '设备B', status: '停机' },
          ],
        },
      },
      global: { stubs: globalStubs },
    })

    expect(wrapper.find('.el-table').attributes('data-rows')).toBe('2')
    const columns = wrapper.findAll('.el-table-column')
    expect(columns).toHaveLength(2)
    expect(columns.at(0)?.attributes('data-width')).toBe('120')
    expect(columns.at(1)?.attributes('data-align')).toBe('center')
  })

  it('renders pagination fallback view from configurable data', () => {
    const wrapper = mount(PaginationRenderer, {
      props: {
        resolvedProps: {
          currentPage: 2,
          pageSize: 20,
          total: 88,
        },
      },
    })

    expect(wrapper.text()).toContain('共 88 条')
    expect(wrapper.find('.core-pagination__pager.is-active').text()).toBe('2')
    expect(wrapper.findAll('.core-pagination__pager')).toHaveLength(5)
  })

  it('emits page updates from pagination interactions', async () => {
    const wrapper = mount(PaginationRenderer, {
      props: {
        resolvedProps: {
          currentPage: 2,
          pageSize: 20,
          total: 88,
        },
      },
    })

    await wrapper.findAll('.core-pagination__pager').at(2)?.trigger('click')

    expect(wrapper.emitted('update:currentPage')?.at(-1)).toEqual([3])
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([3])
  })
})
