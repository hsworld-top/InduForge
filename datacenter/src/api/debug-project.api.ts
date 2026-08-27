import request from '@/utils/request'

const DEFAULT_DEBUG_PROJECT_NAME = 'test'
const DEBUG_PROJECT_QUERY_LIMIT = 50

const asNonEmptyString = (value) =>
  typeof value === 'string' && value.trim().length > 0 ? value.trim() : null

/**
 * 统一提取 projects 列表。
 *
 * 当前项目接口存在两种包络：
 * 1. 旧结构：data.projects
 * 2. 现结构：data.list.projects
 * debug 默认工程解析只关心工程数组本身，这里做兼容收敛，避免 `/datacenter/debug`
 * 在接口包络升级后拿不到默认工程上下文。
 */
const extractProjects = (payload) => {
  if (Array.isArray(payload?.data?.projects)) {
    return payload.data.projects
  }

  if (Array.isArray(payload?.data?.list?.projects)) {
    return payload.data.list.projects
  }

  return []
}

const normalizeProjectList = (payload) => {
  const projects = extractProjects(payload)

  return projects
    .map((item) => {
      if (!item || typeof item !== 'object') {
        return null
      }

      const id = asNonEmptyString(item.id)
      const name = asNonEmptyString(item.name)
      if (!id || !name) {
        return null
      }

      return {
        id,
        name,
        tenantId: asNonEmptyString(item.tenantId),
      }
    })
    .filter(Boolean)
}

/**
 * `/datacenter/debug` 默认工程解析。
 * 仅在开发态独立调试链路使用，按工程名精确匹配名称为 `test` 的工程。
 */
export const debugProjectAPI = {
  async resolveDefaultProjectByName(projectName = DEFAULT_DEBUG_PROJECT_NAME) {
    const response = await request.get('/projects', {
      params: {
        page: 1,
        limit: DEBUG_PROJECT_QUERY_LIMIT,
        name: projectName,
      },
    })
    return normalizeProjectList(response).find((item) => item.name === projectName) || null
  },
}

export default debugProjectAPI
