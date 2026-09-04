import request, { type ApiResponsePayload } from '@/utils/request'

export interface WorkspaceContextRefreshResult {
  contractVersion: string
  pointCount: number
  roleCount: number
  sceneCount?: number
  sceneContractVersion?: string
  updatedAt: string
}

export const contextPackApi = {
  async refresh(projectId: string): Promise<WorkspaceContextRefreshResult> {
    const response = await request.post<ApiResponsePayload<WorkspaceContextRefreshResult>>(
      `/projects/${encodeURIComponent(projectId)}/workspace-context/refresh`,
      undefined,
      { authoringProjectId: projectId },
    )
    if (!response.data) {
      throw new Error('工程上下文接口未返回 data')
    }
    return response.data
  },
}
