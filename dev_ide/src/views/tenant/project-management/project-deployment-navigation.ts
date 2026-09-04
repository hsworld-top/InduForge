import { shallowRef } from 'vue'

export interface ProjectDeploymentNavigation {
  projectId: string
  projectName?: string
  deploymentId?: string
  requestId: number
}

// 单槽请求在运维页挂载前保留，避免用定时重发猜测懒加载时机。
export const deploymentNavigation = shallowRef<ProjectDeploymentNavigation | null>(null)
let sequence = 0
export function requestDeploymentNavigation(
  projectId: string,
  deploymentId?: string,
  projectName?: string,
) {
  deploymentNavigation.value = { projectId, deploymentId, projectName, requestId: ++sequence }
}
