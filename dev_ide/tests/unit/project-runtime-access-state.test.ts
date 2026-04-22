import { computed, reactive, ref } from 'vue'
import { describe, expect, test, vi } from 'vitest'
import { useProjectRuntimeAccessState } from '@/views/tenant/components/project-runtime-access-state.ts'

describe('project-runtime-access-state', () => {
  test('watch immediate 首次执行时 oldValue 缺失不应抛出异常', () => {
    const visibleRef = ref(true)
    const projectRef = ref({ id: 'project-1' })
    const api = {
      listRuntimeUsers: vi.fn().mockResolvedValue({ data: [] }),
      listRuntimeRoles: vi.fn().mockResolvedValue({ data: [] }),
      createRuntimeUser: vi.fn(),
      updateRuntimeUserRoles: vi.fn(),
      updateRuntimeUserStatus: vi.fn(),
      resetRuntimeUserPassword: vi.fn(),
      createRuntimeRole: vi.fn(),
      updateRuntimeRole: vi.fn(),
      deleteRuntimeRole: vi.fn(),
    }
    const watchStub = (
      _source: unknown,
      callback: (value: [boolean, string], oldValue: [boolean, string] | undefined) => void,
      options?: { immediate?: boolean }
    ) => {
      if (options?.immediate) {
        callback([visibleRef.value, String(projectRef.value?.id || '')], undefined)
      }
    }

    expect(() =>
      useProjectRuntimeAccessState({
        visibleRef,
        projectRef,
        t: (key: string) => key,
        api,
        ref,
        reactive,
        computed,
        watch: watchStub,
        message: {
          error: vi.fn(),
          warning: vi.fn(),
          success: vi.fn(),
        },
        messageBox: {
          prompt: vi.fn(),
          confirm: vi.fn(),
        },
      })
    ).not.toThrow()
  })
})
