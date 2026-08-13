type DashboardActivity = {
  action?: string
  resource?: string
  path?: string
  user?: {
    username?: string
    fullName?: string
  }
}

type Translate = (key: string, params?: Record<string, unknown>) => string

const actionByPath: Array<[RegExp, string]> = [
  [/\/activate$/, 'activate'],
  [/\/suspend$/, 'suspend'],
  [/\/approve$/, 'approve'],
  [/\/reject$/, 'reject'],
  [/\/operations\/archive$/, 'archive'],
  [/\/operations\/restore$/, 'restore'],
  [/\/projects\/import$/, 'import'],
  [/\/publish\/[^/]+$/, 'publish'],
  [/\/deploy$/, 'deploy'],
  [/\/deploy-dev$/, 'deploy'],
  [/\/rollback$/, 'rollback'],
  [/\/start$/, 'start'],
  [/\/stop$/, 'stop'],
  [/\/restart$/, 'restart'],
  [/\/rebuild$/, 'rebuild'],
  [/\/reset-password$/, 'resetPassword'],
  [/\/workspace-context\/refresh$/, 'refresh'],
  [/\/upload$/, 'update'],
  [/\/design-files\/actions$/, 'operate'],
  [/\/design-files\/export$/, 'export'],
]

const resourceByPath: Array<[RegExp, string]> = [
  [/\/dashboard-notes(?:\/|$)/, 'note'],
  [/\/projects\/groups(?:\/|$)/, 'projectGroup'],
  [/\/projects\/tags(?:\/|$)/, 'projectTag'],
  [/\/runtime-users\/[^/]+\/reset-password$/, 'runtimeUserPassword'],
  [/\/runtime-users(?:\/|$)/, 'runtimeUser'],
  [/\/runtime-roles(?:\/|$)/, 'runtimeRole'],
  [/\/code-workspace(?:\/|$)/, 'codeWorkspace'],
  [/\/scene-contracts(?:\/|$)/, 'sceneContract'],
  [/\/design-files(?:\/|$)/, 'designFile'],
  [/\/workspace-context(?:\/|$)/, 'workspaceContext'],
  [/\/tenants\/[^/]+\/upload$/, 'tenantBranding'],
  [/\/publish(?:\/|$)/, 'version'],
  [/\/deployments(?:\/|$)/, 'deployment'],
]

const resourceKeyByName: Record<string, string> = {
  tenants: 'tenant',
  users: 'user',
  projects: 'project',
  nodes: 'node',
  logs: 'log',
}

export function formatDashboardActivity(activity: DashboardActivity, t: Translate): string {
  const actor =
    activity.user?.fullName?.trim() ||
    activity.user?.username?.trim() ||
    t('dashboard.systemAction')
  const pathAction = actionByPath.find(([pattern]) => pattern.test(activity.path || ''))?.[1]
  const action =
    pathAction ||
    (['create', 'update', 'delete'].includes(activity.action || '') ? activity.action! : 'operate')
  const pathResource = resourceByPath.find(([pattern]) => pattern.test(activity.path || ''))?.[1]
  const resource = pathResource || resourceKeyByName[activity.resource || ''] || 'resource'

  return t('dashboard.businessActivity', {
    actor,
    action: t(`dashboard.activityAction.${action}`),
    resource: t(`dashboard.activityResource.${resource}`),
  })
}
