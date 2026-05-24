import { Storage } from '@/utils/storage'

interface HostProjectContext {
  projectId?: string | null
  tenantId?: string | null
}

interface ProjectContextStorage {
  getProjectId: () => string | null
  getTenantId: () => string | null
}

export const resolveRouteProjectContext = (
  hostContext: HostProjectContext = {},
  storage: ProjectContextStorage = Storage,
) => ({
  id: hostContext.projectId ?? storage.getProjectId(),
  tenantId: hostContext.tenantId ?? storage.getTenantId(),
})
