const { Log } = require('../models')
const { logger } = require('../utils/logger')

const RESOURCE_LABELS = {
  tenants: '租户',
  users: '用户',
  projects: '工程',
  logs: '系统日志',
  roles: '角色',
  data: '数据中心',
  design: '设计器',
  pages: '页面锁',
  nodes: '节点',
  'node-register': '节点注册',
  deployments: '部署',
  publish: '发布',
  auth: '认证',
}

const ACTION_LABELS = {
  create: '新增',
  update: '更新',
  delete: '删除',
  patch: '变更',
  activate: '激活',
  suspend: '暂停',
  upload: '上传',
  import: '导入',
  export: '导出',
  deploy: '部署',
  rollback: '回滚',
  start: '启动',
  stop: '停止',
  restart: '重启',
  approve: '审批通过',
  reject: '审批拒绝',
  lock: '加锁',
  unlock: '解锁',
  password: '修改密码',
}

/**
 * 构建日志 action 字段，长度限制在 100 以内。
 * @param {import('express').Request} req - Express 请求对象
 * @returns {string} action 字段
 */
function buildAction(req) {
  const resource = extractResource(req) || 'unknown'
  const op = detectOperation(req)
  const raw = `${resource}.${op}`
  return raw.length > 100 ? raw.slice(0, 100) : raw
}

/**
 * 从路径提取资源名（一级路由）。
 * 例如 /api/v1/users/xxx -> users
 * @param {import('express').Request} req - Express 请求对象
 * @returns {string|null} 资源名
 */
function extractResource(req) {
  const path = String(req.originalUrl || req.url || '').split('?')[0]
  const parts = path.split('/').filter(Boolean)
  const v1Index = parts.indexOf('v1')
  if (v1Index < 0 || !parts[v1Index + 1]) {
    return null
  }
  return parts[v1Index + 1]
}

/**
 * 判断是否需要写审计日志。
 * 默认仅记录写操作（POST/PUT/PATCH/DELETE）。
 * @param {import('express').Request} req - Express 请求对象
 * @returns {boolean} 是否需要记录
 */
function shouldLog(req) {
  const method = String(req.method || 'GET').toUpperCase()
  if (!['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
    return false
  }

  const path = String(req.originalUrl || req.url || '').split('?')[0]
  const ignored = [
    '/api/v1/auth/captcha',
    '/api/v1/auth/login',
    '/api/v1/auth/config',
    '/api/v1/auth/refresh',
  ]
  if (ignored.includes(path)) {
    return false
  }

  // 高频 Agent 心跳与部署状态回调不写审计日志，避免产生大量无业务价值日志写入。
  if (/^\/api\/v1\/nodes\/[^/]+\/heartbeat$/.test(path)) {
    return false
  }
  if (/^\/api\/v1\/nodes\/[^/]+\/deployment-status$/.test(path)) {
    return false
  }

  return true
}

/**
 * 判断路径片段是否为资源 ID。
 * @param {string} segment - 路径片段
 * @returns {boolean} 是否资源 ID
 */
function isIdSegment(segment) {
  if (!segment) return false
  if (segment === 'me') return false
  return isUuid(segment) || /^\d+$/.test(segment)
}

/**
 * 识别操作类型。
 * @param {import('express').Request} req - Express 请求对象
 * @returns {string} 操作标识
 */
function detectOperation(req) {
  const method = String(req.method || 'GET').toUpperCase()
  const path = String(req.originalUrl || req.url || '').split('?')[0]
  const parts = path.split('/').filter(Boolean)
  const v1Index = parts.indexOf('v1')
  const segments = v1Index >= 0 ? parts.slice(v1Index + 1) : parts
  const routeSegments = segments.slice(1).filter((item) => !isIdSegment(item))

  const explicitOp = routeSegments.find((item) =>
    [
      'activate',
      'suspend',
      'upload',
      'import',
      'export',
      'deploy',
      'rollback',
      'start',
      'stop',
      'restart',
      'approve',
      'reject',
      'lock',
      'unlock',
      'password',
    ].includes(item),
  )
  if (explicitOp) {
    return explicitOp
  }

  if (method === 'POST') return 'create'
  if (method === 'PUT') return 'update'
  if (method === 'PATCH') return 'patch'
  if (method === 'DELETE') return 'delete'
  return method.toLowerCase()
}

/**
 * 构建可读日志消息。
 * @param {import('express').Request} req - Express 请求对象
 * @param {number} statusCode - 响应状态码
 * @returns {string} 日志消息
 */
function buildMessage(req, statusCode) {
  const resource = extractResource(req) || 'unknown'
  const op = detectOperation(req)
  const resourceLabel = RESOURCE_LABELS[resource] || resource
  const actionLabel = ACTION_LABELS[op] || op
  const result = statusCode >= 400 ? '失败' : '成功'
  const operator = req.user?.username || 'unknown'
  if (statusCode >= 400) {
    return `${resourceLabel}${actionLabel}${result}，操作者：${operator}，状态码：${statusCode}`
  }
  return `${resourceLabel}${actionLabel}${result}，操作者：${operator}`
}

/**
 * 判断是否为 UUID 字符串。
 * @param {unknown} value - 待检查值
 * @returns {boolean} 是否 UUID
 */
function isUuid(value) {
  return (
    typeof value === 'string' &&
    /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(value)
  )
}

/**
 * 操作审计日志中间件。
 * 在响应结束后异步写入 logs 表，避免阻塞主流程。
 * @param {import('express').Request} req - Express 请求对象
 * @param {import('express').Response} res - Express 响应对象
 * @param {import('express').NextFunction} next - Next 函数
 */
function operationAuditLog(req, res, next) {
  if (!shouldLog(req)) {
    return next()
  }

  res.on('finish', () => {
    setImmediate(async () => {
      try {
        const statusCode = Number(res.statusCode || 200)
        const user = req.user || {}
        let userId = isUuid(user.id) ? user.id : null
        const tenantId = isUuid(user.tenantId) ? user.tenantId : null

        // 超级管理员账号可能不在 users 表中，避免触发外键错误。
        if (user.role === 'SUPER_ADMIN') {
          userId = null
        }

        const level = statusCode >= 500 ? 'error' : statusCode >= 400 ? 'warning' : 'info'
        const resource = extractResource(req)
        const message = buildMessage(req, statusCode)

        await Log.create({
          level,
          message,
          action: buildAction(req),
          resource,
          userId,
          tenantId,
          ip: req.ip || null,
          userAgent: req.get('user-agent') || null,
          createdAt: new Date(),
        })
      } catch (error) {
        logger.warn('Write operation audit log failed', {
          error: error.message,
          requestId: req.requestId,
        })
      }
    })
  })

  return next()
}

module.exports = { operationAuditLog }
