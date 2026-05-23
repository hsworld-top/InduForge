import { describe, expect, it, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import { saveEntryPatchForStore } from './page-persist-actions'

describe('page-persist-actions', () => {
  it('saveEntryPatchForStore 在缺少文档命令栈时仍可更新入口配置', async () => {
    const updateEntryConfig = vi.fn().mockResolvedValue(undefined)
    const entryConfig = ref({ homePageId: 'home-1' })

    await saveEntryPatchForStore({
      projectId: ref('project-1'),
      doc: shallowRef(null),
      entryConfig,
      projectApi: {
        updateEntryConfig,
      },
      patch: {
        loginPageId: 'login-1',
      },
    })

    expect(updateEntryConfig).toHaveBeenCalledWith('project-1', {
      homePageId: 'home-1',
      loginPageId: 'login-1',
    })
    expect(entryConfig.value).toEqual({
      homePageId: 'home-1',
      loginPageId: 'login-1',
    })
  })
})
