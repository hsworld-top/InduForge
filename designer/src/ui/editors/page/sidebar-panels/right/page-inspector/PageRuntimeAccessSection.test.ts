import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import PageRuntimeAccessSection from './PageRuntimeAccessSection.vue'

describe('PageRuntimeAccessSection', () => {
  const createWrapper = () => {
    const form = {
      runtimeAccessEnabled: false,
      runtimeAccessAllowedRoles: [],
      runtimePermissionSchemes: [
        { id: 'scheme-1', name: '查看方案', roleRefs: [] },
        { id: 'scheme-2', name: '操作方案', roleRefs: [] },
      ],
    }

    const wrapper = mount(PageRuntimeAccessSection, {
      props: {
        form: form as any,
        runtimeRoles: [
          {
            id: 'role-admin',
            roleId: 'role-admin',
            code: 'PROJECT_ADMIN',
            roleCode: 'PROJECT_ADMIN',
            name: '管理员',
            roleName: '管理员',
            status: 'active',
          },
        ] as any,
      },
      global: {
        stubs: {
          'el-switch': {
            props: ['modelValue'],
            template: '<input type="checkbox" :checked="modelValue" />',
          },
          'el-select': {
            props: ['disabled'],
            template: '<select :disabled="disabled"><slot /></select>',
          },
          'el-option': {
            props: ['label', 'value'],
            template: '<option :value="value">{{ label }}</option>',
          },
          'el-button': {
            props: ['disabled'],
            emits: ['click'],
            template:
              '<button class="scheme-button" type="button" :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
          },
          'el-dialog': {
            props: ['modelValue'],
            template: '<section v-if="modelValue" class="dialog-stub"><slot /></section>',
          },
          PagePermissionSchemesSection: {
            template: '<div class="scheme-section-stub" />',
          },
        },
      },
    })

    return { wrapper, form }
  }

  it('在运行态权限控制内展示方案数量，并只显示角色名', () => {
    const { wrapper } = createWrapper()

    expect(wrapper.text()).toContain('2 个')
    expect(wrapper.text()).toContain('管理员')
    expect(wrapper.text()).not.toContain('PROJECT_ADMIN')
  })

  it('启用控制后才能打开权限方案弹窗', async () => {
    const { wrapper, form } = createWrapper()
    const button = wrapper.find('.scheme-button')

    expect(button.attributes('disabled')).toBeDefined()
    expect(wrapper.find('.dialog-stub').exists()).toBe(false)

    const enabledForm = {
      ...form,
      runtimeAccessEnabled: true,
    }
    await wrapper.setProps({ form: enabledForm as any })
    await nextTick()

    expect(wrapper.find('.scheme-button').attributes('disabled')).toBeUndefined()
    await wrapper.find('.scheme-button').trigger('click')

    expect(wrapper.find('.dialog-stub').exists()).toBe(true)
    expect(wrapper.find('.scheme-section-stub').exists()).toBe(true)
  })
})
