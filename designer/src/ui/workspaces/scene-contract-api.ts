import request, { type ApiResponsePayload } from '@/utils/request'

export type SceneKind = '2d' | '3d'
export type SceneEmbedMode = 'standalone' | 'embedded' | 'both'

export interface SceneContractMember {
  name: string
  description?: string
  schema?: Record<string, unknown>
}

export interface SceneContract {
  id: string
  kind: SceneKind
  name: string
  description?: string
  route?: string
  embedMode: SceneEmbedMode
  inputs?: SceneContractMember[]
  events?: SceneContractMember[]
  commands?: SceneContractMember[]
  publicObjects?: SceneContractMember[]
  datapointRefs?: string[]
  permissionRefs?: string[]
  contractVersion: string
}

export interface SceneContractSnapshot {
  contractVersion: string
  contracts: SceneContract[]
}

export interface ContextSyncStatus {
  status: 'updated' | 'stale' | 'not-configured'
  message?: string
}

interface SceneContractSaveResult {
  contract: SceneContract
  contextSync: ContextSyncStatus
}

interface SceneContractDeleteResult {
  deleted: boolean
  contextSync: ContextSyncStatus
}

export const sceneContractApi = {
  async list(projectId: string): Promise<SceneContractSnapshot> {
    const response = await request.get<ApiResponsePayload<SceneContractSnapshot>>(
      `/projects/${encodeURIComponent(projectId)}/scene-contracts`,
    )
    if (!response.data) throw new Error('场景公开契约接口未返回 data')
    return response.data
  },

  async save(projectId: string, contract: SceneContract): Promise<SceneContractSaveResult> {
    const response = await request.put<ApiResponsePayload<SceneContractSaveResult>>(
      `/projects/${encodeURIComponent(projectId)}/scene-contracts/${contract.kind}/${encodeURIComponent(contract.id)}`,
      contract,
    )
    if (!response.data) throw new Error('场景公开契约保存接口未返回 data')
    return response.data
  },

  async remove(projectId: string, kind: SceneKind, id: string): Promise<SceneContractDeleteResult> {
    const response = await request.delete<ApiResponsePayload<SceneContractDeleteResult>>(
      `/projects/${encodeURIComponent(projectId)}/scene-contracts/${kind}/${encodeURIComponent(id)}`,
    )
    if (!response.data) throw new Error('场景公开契约删除接口未返回 data')
    return response.data
  },
}
