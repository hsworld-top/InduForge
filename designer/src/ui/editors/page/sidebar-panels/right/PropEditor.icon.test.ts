import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import PropEditor from './PropEditor.vue'

const AutocompleteStub = defineComponent({
  name: 'ElAutocomplete',
  props: {
    modelValue: { type: String, default: '' },
    fetchSuggestions: { type: Function, required: true },
    placeholder: { type: String, default: '' },
    clearable: { type: Boolean, default: false },
    size: { type: String, default: '' },
  },
  emits: ['update:modelValue', 'select', 'clear'],
  template: `
    <div class="icon-autocomplete">
      <input
        class="icon-autocomplete__input"
        :value="modelValue"
        :placeholder="placeholder"
        @input="$emit('update:modelValue', $event.target.value)"
      />
      <button class="icon-autocomplete__select" @click="$emit('select', { label: '新增', value: 'Plus' })">
        select
      </button>
      <button class="icon-autocomplete__clear" @click="$emit('clear')">clear</button>
    </div>
  `,
})

describe('PropEditor icon editor', () => {
  const mountIconEditor = () =>
    mount(PropEditor, {
      props: {
        prop: {
          editor: 'icon',
          type: 'string',
          placeholder: '选择或输入图标名',
          options: [
            { label: '搜索', value: 'Search' },
            { label: '新增', value: 'Plus' },
            { label: '下载', value: 'Download' },
          ],
        },
        modelValue: 'Download',
      },
      global: {
        stubs: {
          'el-autocomplete': AutocompleteStub,
          'el-input': true,
          'el-input-number': true,
          'el-switch': true,
          'el-select': true,
          'el-option': true,
        },
      },
    })

  it('shows localized labels while emitting stable icon values', async () => {
    const wrapper = mountIconEditor()
    const autocomplete = wrapper.getComponent(AutocompleteStub)
    const suggestions: Array<{ label: string; value: string }> = []

    expect(autocomplete.props('modelValue')).toBe('下载')

    autocomplete.props('fetchSuggestions')('下', (items: typeof suggestions) => {
      suggestions.push(...items)
    })
    expect(suggestions).toEqual([{ label: '下载', value: 'Download' }])

    await wrapper.find('.icon-autocomplete__select').trigger('click')
    await nextTick()
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['Plus'])
    expect(wrapper.getComponent(AutocompleteStub).props('modelValue')).toBe('新增')

    await wrapper.setProps({
      prop: {
        editor: 'icon',
        type: 'string',
        placeholder: 'Select or enter icon name',
        options: [
          { label: 'Search', value: 'Search' },
          { label: 'Add', value: 'Plus' },
          { label: 'Download', value: 'Download' },
        ],
      },
      modelValue: 'Plus',
    })
    expect(wrapper.getComponent(AutocompleteStub).props('modelValue')).toBe('Add')

    await wrapper.find('.icon-autocomplete__input').setValue('CustomIcon')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['CustomIcon'])

    await wrapper.find('.icon-autocomplete__clear').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([''])
  })
})
