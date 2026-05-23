/**
 * 调试工程解析 API。
 *
 * `/designer/debug` 不再依赖手工传 pid。
 * 当前实现只在开发态独立调试链路下使用，通过工程名精确匹配默认调试工程。
 */

import request from '@/utils/request'
import { Storage } from '@/utils/storage'

export interface DebugProjectSummary {
  id: string
  name: string
  tenantId: string | null
}

interface DebugProjectListPayload {
  data?: {
    projects?: unknown[]
  }
  projects?: unknown[]
}

const DEFAULT_DEBUG_PROJECT_NAME = 'test'
const DEBUG_PROJECT_QUERY_LIMIT = 50

function asNonEmptyString(value: unknown): string | null {
  return typeof value === 'string' && value.trim().length > 0 ? value.trim() : null
}

function normalizeProjectList(
  payload: DebugProjectListPayload | null | undefined,
): DebugProjectSummary[] {
  const projects = Array.isArray(payload?.data?.projects)
    ? payload.data.projects
    : Array.isArray(payload?.projects)
      ? payload.projects
      : []

  return projects
    .map((item) => {
      if (!item || typeof item !== 'object') {
        return null
      }

      const record = item as Record<string, unknown>
      const id = asNonEmptyString(record.id)
      const name = asNonEmptyString(record.name)
      if (!id || !name) {
        return null
      }

      return {
        id,
        name,
        tenantId: asNonEmptyString(record.tenantId),
      } satisfies DebugProjectSummary
    })
    .filter((item): item is DebugProjectSummary => Boolean(item))
}

export const debugProjectApi = {
  async resolveDefaultProjectByName(
    projectName = DEFAULT_DEBUG_PROJECT_NAME,
  ): Promise<DebugProjectSummary | null> {
    // 调试默认工程依赖登录态；无 token 时直接跳过，避免独立入口主动触发 401/跳登录。
    if (!Storage.getToken()) {
      return null
    }

    const response = await request.get<DebugProjectListPayload>('/projects', {
      params: {
        page: 1,
        limit: DEBUG_PROJECT_QUERY_LIMIT,
        name: projectName,
      },
    })
    const matchedProject = normalizeProjectList(response).find((item) => item.name === projectName)
    return matchedProject ?? null
  },
}

export default debugProjectApi
