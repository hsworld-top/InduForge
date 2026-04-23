import { defineComponent, nextTick } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { describe, expect, test, vi } from 'vitest'

import ProjectRuntimeAccessDialog from '@/views/tenant/components/ProjectRuntimeAccessDialog.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => {
      if (key === 'projectManagement.runtimeAccess.dialogTitle' && params?.name) {
        return `成员与权限 - ${params.name}`
      }
      return key
    },
  }),
}))

vi.mock('@/api/project.api.js', () => ({
  projectAPI: {
    listRuntimeUsers: vi.fn().mockResolvedValue({
      data: [
        {
          id: 'user-1',
          username: 'alice',
          displayName: 'Alice',
          status: 'active',
          roleIds: ['role-1'],
        },
      ],
    }),
    createRuntimeUser: vi.fn(),
    updateRuntimeUserStatus: vi.fn(),
    updateRuntimeUserRoles: vi.fn(),
    resetRuntimeUserPassword: vi.fn(),
    listRuntimeRoles: vi.fn().mockResolvedValue({
      data: [
        {
          id: 'role-1',
          code: 'ADMIN',
          name: '管理员',
          description: '系统管理员',
          status: 'active',
          isSystem: true,
          bindingCount: 1,
          grantCount: 8,
        },
      ],
    }),
    createRuntimeRole: vi.fn(),
    updateRuntimeRole: vi.fn(),
    deleteRuntimeRole: vi.fn(),
  },
}))

const createSlotStub = (name: string) =>
  defineComponent({
    name,
    props: {
      title: {
        type: String,
        default: '',
      },
      label: {
        type: String,
        default: '',
      },
      name: {
        type: String,
        default: '',
      },
    },
    template: `
      <div :class="name">
        <slot />
        <slot name="footer" />
      </div>
    `,
  })

describe('ProjectRuntimeAccessDialog', () => {
  test('成员权限主弹窗应包含统一的视觉壳层结构', async () => {
    const wrapper = shallowMount(ProjectRuntimeAccessDialog, {
      props: {
        visible: true,
        project: {
          id: 'project-1',
          name: '演示工程',
        },
      },
      global: {
        stubs: {
          'el-dialog': createSlotStub('el-dialog-stub'),
          'el-alert': createSlotStub('el-alert-stub'),
          'el-empty': createSlotStub('el-empty-stub'),
          'el-tabs': createSlotStub('el-tabs-stub'),
          'el-tab-pane': createSlotStub('el-tab-pane-stub'),
          'el-table': createSlotStub('el-table-stub'),
          'el-table-column': createSlotStub('el-table-column-stub'),
          'el-button': createSlotStub('el-button-stub'),
          'el-form': createSlotStub('el-form-stub'),
          'el-form-item': createSlotStub('el-form-item-stub'),
          'el-input': createSlotStub('el-input-stub'),
          'el-select': createSlotStub('el-select-stub'),
          'el-option': createSlotStub('el-option-stub'),
          'el-tag': createSlotStub('el-tag-stub'),
        },
      },
    })

    await flushPromises()
    await nextTick()

    expect(wrapper.find('.runtime-access-shell').exists()).toBe(true)
    expect(wrapper.find('.runtime-access-panel').exists()).toBe(true)
    expect(wrapper.find('.runtime-access-tabs-card').exists()).toBe(true)
    expect(wrapper.find('.runtime-access-section-card').exists()).toBe(true)
    expect(wrapper.find('.runtime-access-table-wrap').exists()).toBe(true)
  })
})
