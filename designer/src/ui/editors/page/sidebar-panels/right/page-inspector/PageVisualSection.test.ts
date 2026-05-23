import { mount } from '@vue/test-utils'
import { reactive } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { i18n } from '@/i18n'
import PageVisualSection from './PageVisualSection.vue'
import type { PageInspectorFormState } from './page-inspector-types'

vi.mock('@/ui/shared/widgets/base/FriendlyColorPicker.vue', () => ({
  default: {
    props: ['modelValue', 'layout', 'showRecent'],
    template:
      "<div class='color-picker-stub' :data-layout='layout' :data-show-recent='String(showRecent)'>{{ modelValue }}</div>",
  },
}))

vi.mock('@/ui/shared/widgets/base/MonacoEditor.vue', () => ({
  default: {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template:
      "<textarea class='monaco-editor-stub' :value='modelValue' @input=\"$emit('update:modelValue', $event.target.value)\" />",
  },
}))

vi.mock('@/ui/shared/widgets/base/monaco-editor-async', () => ({
  default: {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template:
      "<textarea class='monaco-editor-stub' :value='modelValue' @input=\"$emit('update:modelValue', $event.target.value)\" />",
  },
}))

function createForm(
  backgroundType: PageInspectorFormState['backgroundType'],
): PageInspectorFormState {
  return reactive({
    name: '首页',
    title: '首页标题',
    description: '首页描述',
    role: 'home',
    routeMode: 'auto',
    routePath: '/',
    routeSlug: '',
    parentRoutePath: '/',
    viewportPreset: 'pc',
    width: 1366,
    height: 768,
    autoFit: true,
    lockAspectRatio: false,
    minWidth: 0,
    minHeight: 0,
    overflowMode: 'auto',
    backgroundType,
    backgroundValue: backgroundType === 'color' ? '#ffffff' : 'asset://background',
    backgroundSize: 'cover',
    backgroundPosition: 'center',
    backgroundRepeat: 'no-repeat',
    styleConfig: '',
    transitionType: 'none',
    openMode: 'replace',
    popupWidth: 800,
    popupHeight: 600,
    popupCenter: true,
    popupMaskClosable: true,
    permissionSummary: '未配置',
    cacheMode: 'default',
    preloadMode: 'lazy',
  })
}

function mountSection(form: PageInspectorFormState) {
  return mount(PageVisualSection, {
    props: { form },
    global: {
      plugins: [i18n],
      stubs: {
        'el-select': {
          props: ['modelValue', 'disabled', 'size'],
          template: "<select :disabled='disabled'><slot /></select>",
        },
        'el-option': {
          template: '<option><slot /></option>',
        },
        'el-input': {
          props: ['modelValue', 'disabled', 'placeholder', 'size'],
          template: "<input :value='modelValue' :disabled='disabled' :placeholder='placeholder' />",
        },
        'el-button': {
          props: ['size', 'type'],
          template: "<button type='button' @click='$emit(\"click\", $event)'><slot /></button>",
        },
        'el-dialog': {
          props: ['modelValue', 'title', 'width', 'top'],
          emits: ['update:modelValue'],
          template:
            "<div v-if='modelValue' class='dialog-stub'><div class='dialog-title'>{{ title }}</div><slot /><footer><slot name='footer' /></footer></div>",
        },
        AssetManagerDialog: {
          props: ['modelValue'],
          emits: ['update:modelValue', 'select'],
          template:
            "<div v-if='modelValue' class='asset-manager-dialog-stub'><button class='select-asset-stub' @click='$emit(\"select\", { id: \"asset-1\", url: \"/assets/bg.png\" })'>select</button></div>",
        },
      },
    },
  })
}

describe('PageVisualSection', () => {
  it('hides image-only background controls for solid color background', () => {
    const wrapper = mountSection(createForm('color'))

    expect(wrapper.text()).toContain('背景类型')
    expect(wrapper.text()).toContain('背景值')
    expect(wrapper.text()).not.toContain('背景填充')
    expect(wrapper.text()).not.toContain('背景定位')
    expect(wrapper.text()).not.toContain('背景重复')
    expect(wrapper.find('.color-picker-stub').exists()).toBe(true)
    expect(wrapper.find('.color-picker-stub').attributes('data-layout')).toBe('block')
    expect(wrapper.find('.color-picker-stub').attributes('data-show-recent')).toBe('false')
    expect(wrapper.find('.background-gradient-preview').exists()).toBe(false)
  })

  it('shows image-only background controls for image background', () => {
    const wrapper = mountSection(createForm('image'))

    expect(wrapper.text()).toContain('背景填充')
    expect(wrapper.text()).toContain('背景定位')
    expect(wrapper.text()).toContain('背景重复')
    expect(wrapper.find('.background-image-picker__button').exists()).toBe(true)
  })

  it('hides image-only background controls for gradient background', () => {
    const wrapper = mountSection(createForm('gradient'))

    expect(wrapper.text()).not.toContain('背景填充')
    expect(wrapper.text()).not.toContain('背景定位')
    expect(wrapper.text()).not.toContain('背景重复')
    expect(wrapper.find('.background-gradient-preview').exists()).toBe(true)
  })

  it('applies gradient preset as background value', async () => {
    const form = createForm('gradient')
    const wrapper = mountSection(form)

    await wrapper.find('.background-gradient-preview').trigger('click')
    await wrapper.find('.background-editor__preset').trigger('click')

    expect(form.backgroundValue).toContain('linear-gradient')
    expect(wrapper.emitted('updateConfig')).toHaveLength(1)
  })

  it('opens style editor and persists css draft', async () => {
    const form = createForm('color')
    const wrapper = mountSection(form)

    await wrapper.find('.style-config-button').trigger('click')
    await wrapper.find('.monaco-editor-stub').setValue('.page-title { color: red; }')
    await wrapper.findAll('button').at(-1)?.trigger('click')

    expect(form.styleConfig).toBe('.page-title { color: red; }')
    expect(wrapper.emitted('updateConfig')).toHaveLength(1)
  })

  it('selects image resource as page background value', async () => {
    const form = createForm('image')
    const wrapper = mountSection(form)

    await wrapper.find('.background-image-picker__button').trigger('click')
    await wrapper.find('.select-asset-stub').trigger('click')

    expect(form.backgroundValue).toBe('/assets/bg.png')
    expect(wrapper.emitted('updateConfig')).toHaveLength(1)
  })
})
