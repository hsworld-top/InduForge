import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SceneContractPanel from './SceneContractPanel.vue'

const { listMock, updateMock, commitMock } = vi.hoisted(() => ({
  listMock: vi.fn(),
  updateMock: vi.fn(),
  commitMock: vi.fn(),
}))

vi.mock('./scene-contract-api', () => ({
  sceneContractApi: {
    list: listMock,
    update: updateMock,
    commit: commitMock,
  },
}))

const contract = {
  id: 'overview',
  kind: '2d' as const,
  name: '产线总览',
  description: '已提交场景',
  parameters: [],
  events: [],
  commands: [],
  datapointRefs: [],
  contractVersion: '2',
  currentRevision: 2,
  draftVersion: 2,
  committedDraftVersion: 2,
}

describe('SceneContractPanel', () => {
  beforeEach(() => {
    listMock.mockReset().mockResolvedValue({ contractVersion: '2', contracts: [contract] })
    updateMock.mockReset().mockResolvedValue({
      contract: { ...contract, draftVersion: 3 },
      contextSync: { status: 'updated' },
    })
    commitMock.mockReset().mockResolvedValue({ revision: 3 })
  })

  it('保存公开契约后提交 revision 并通知预览刷新', async () => {
    const wrapper = mount(SceneContractPanel, {
      props: { projectId: 'project-1', kind: '2d', sceneId: 'overview' },
    })
    await flushPromises()

    await wrapper.get('footer button:last-child').trigger('click')
    await flushPromises()

    expect(updateMock).toHaveBeenCalledWith(
      'project-1',
      expect.objectContaining({ id: 'overview', kind: '2d', draftVersion: 2 }),
    )
    expect(commitMock).toHaveBeenCalledWith('project-1', '2d', 'overview', 3)
    expect(wrapper.emitted('committed')).toEqual([[
      { projectId: 'project-1', target: '2d', sceneId: 'overview', revision: 3 },
    ]])
    expect(wrapper.emitted('synced')).toEqual([['场景设置已保存并应用']])
  })
})
