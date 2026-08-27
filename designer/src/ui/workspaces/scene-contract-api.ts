import request, { type ApiResponsePayload } from '@/utils/request'

export type SceneKind = '2d' | '3d'
export interface SceneContractMember {
  name: string
  description?: string
  schema: Record<string, unknown>
}

export interface SceneContractParameter extends SceneContractMember {
  required: boolean
}

export interface SceneContractCommand {
  name: string
  description?: string
  inputSchema: Record<string, unknown>
  outputSchema: Record<string, unknown>
}

export interface SceneContract {
  id: string
  kind: SceneKind
  name: string
  description?: string
  parameters?: SceneContractParameter[]
  events?: SceneContractMember[]
  commands?: SceneContractCommand[]
  managedEvents?: SceneContractMember[]
  managedCommands?: SceneContractCommand[]
  datapointRefs?: string[]
  contractVersion: string
  currentRevision?: number
  draftVersion?: number
  committedDraftVersion?: number
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

export interface SceneCreateInput {
  kind: SceneKind
  name: string
}

interface SceneContractDeleteResult {
  deleted: boolean
  contextSync: ContextSyncStatus
}

interface SceneDocumentResponse {
  sceneId: string
  kind: SceneKind
  name: string
  publicContract: Omit<SceneContract, 'id' | 'kind' | 'name' | 'contractVersion'>
  managedContract: Omit<SceneContract, 'id' | 'kind' | 'name' | 'contractVersion'>
  datapointRefs: string[]
  currentRevision: number
  draftVersion: number
  committedDraftVersion: number
}

interface SceneListResponse {
  items: SceneDocumentResponse[]
  total: number
}

interface EditorSessionResponse {
  sessionId: string
  url: string
  expiresAt: string
}

export interface ViewerSessionResponse {
  sessionId: string
  url: string
  expiresAt: string
  sceneId: string
  kind: SceneKind
  revision: number
  contract: Record<string, unknown>
  datapointRefs: string[]
}

function toContract(scene: SceneDocumentResponse): SceneContract {
  return {
    ...scene.publicContract,
    id: scene.sceneId,
    kind: scene.kind,
    name: scene.name,
    datapointRefs: scene.datapointRefs,
    managedEvents: scene.managedContract?.events || [],
    managedCommands: scene.managedContract?.commands || [],
    contractVersion: String(scene.currentRevision),
    currentRevision: scene.currentRevision,
    draftVersion: scene.draftVersion,
    committedDraftVersion: scene.committedDraftVersion,
  }
}

export const sceneContractApi = {
  async list(projectId: string): Promise<SceneContractSnapshot> {
    const response = await request.get<ApiResponsePayload<SceneListResponse>>(
      `/projects/${encodeURIComponent(projectId)}/scenes`,
      { params: { page: 1, limit: 200, sort: 'name', order: 'asc' } },
    )
    if (!response.data) throw new Error('场景公开契约接口未返回 data')
    const contracts = response.data.items.map(toContract)
    return { contractVersion: contracts.map((item) => item.contractVersion).join('-'), contracts }
  },

  async create(projectId: string, input: SceneCreateInput): Promise<SceneContract> {
    const response = await request.post<ApiResponsePayload<SceneDocumentResponse>>(
      `/projects/${encodeURIComponent(projectId)}/scenes`,
      {
        kind: input.kind,
        name: input.name,
        publicContract: { description: '', parameters: [], events: [], commands: [] },
      },
    )
    if (!response.data) throw new Error('场景创建接口未返回 data')
    return toContract(response.data)
  },

  async update(projectId: string, contract: SceneContract): Promise<SceneContractSaveResult> {
    const encodedProjectId = encodeURIComponent(projectId)
    const encodedSceneId = encodeURIComponent(contract.id)
    const response = await request.put<ApiResponsePayload<SceneDocumentResponse>>(
      `/projects/${encodedProjectId}/scenes/${encodedSceneId}`,
      { name: contract.name, publicContract: publicContract(contract), baseDraftVersion: contract.draftVersion },
      { params: { kind: contract.kind } },
    )
    if (!response.data) throw new Error('场景公开契约保存接口未返回 data')
    return { contract: toContract(response.data), contextSync: { status: 'updated' } }
  },

  async remove(projectId: string, kind: SceneKind, id: string): Promise<SceneContractDeleteResult> {
    const response = await request.delete<ApiResponsePayload<{ deleted: boolean }>>(
      `/projects/${encodeURIComponent(projectId)}/scenes/${encodeURIComponent(id)}`,
      { params: { kind } },
    )
    if (!response.data) throw new Error('场景公开契约删除接口未返回 data')
    return { deleted: response.data.deleted, contextSync: { status: 'updated' } }
  },

  async commit(projectId: string, kind: SceneKind, id: string, baseDraftVersion: number) {
    const response = await request.post<ApiResponsePayload<{ revision: number }>>(
      `/projects/${encodeURIComponent(projectId)}/scenes/${encodeURIComponent(id)}/commit`,
      { baseDraftVersion },
      { params: { kind } },
    )
    if (!response.data) throw new Error('场景提交接口未返回 data')
    return response.data
  },

  async editorSession(
    projectId: string,
    kind: SceneKind,
    id: string,
  ): Promise<EditorSessionResponse> {
    const response = await request.post<ApiResponsePayload<EditorSessionResponse>>(
      `/projects/${encodeURIComponent(projectId)}/scenes/${encodeURIComponent(id)}/editor-session`,
      undefined,
      { params: { kind } },
    )
    if (!response.data) throw new Error('编辑会话接口未返回 data')
    return response.data
  },

  async viewerSession(
    projectId: string,
    kind: SceneKind,
    id: string,
  ): Promise<ViewerSessionResponse> {
    const response = await request.post<ApiResponsePayload<ViewerSessionResponse>>(
      `/projects/${encodeURIComponent(projectId)}/scenes/${encodeURIComponent(id)}/viewer-session`,
      undefined,
      { params: { kind } },
    )
    if (!response.data) throw new Error('Viewer 会话接口未返回 data')
    return response.data
  },
}

function publicContract(contract: SceneContract): Record<string, unknown> {
  return {
    description: contract.description || '',
    parameters: contract.parameters || [],
    events: contract.events || [],
    commands: contract.commands || [],
  }
}
