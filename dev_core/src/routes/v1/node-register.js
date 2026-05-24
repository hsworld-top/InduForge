/**
 * 节点注册路由 - 带用户认证的节点注册接口
 * @description 提供 NodeAgent 初始化向导使用的注册接口
 */
const express = require('express')
const router = express.Router()
const nodeService = require('../../services/nodeService')
const ApiResponse = require('../../utils/response')
const AppError = require('../../utils/AppError')
const ErrorCodes = require('../../constants/errorCodes')

/**
 * 路由统一错误响应：
 * - 业务错误：HTTP 200 + 非 0 code
 * - 技术异常：保留 5xx
 */
function respondRouteError(res, error, _fallbackCode, fallbackStatus = 500) {
  const isAppError = error instanceof AppError || error?.name === 'AppError'
  if (isAppError && error?.errorCode && error?.statusCode) {
    const normalizedStatus = Number(error.statusCode) >= 500 ? Number(error.statusCode) : 200
    const options = error.options || (error.message ? { message: error.message } : {})
    return ApiResponse.error(res, error.errorCode, options, normalizedStatus)
  }

  const normalizedFallbackStatus = Number(fallbackStatus) >= 500 ? Number(fallbackStatus) : 500
  const options = error?.message ? { message: error.message } : {}
  return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, options, normalizedFallbackStatus)
}

/**
 * @route POST /api/v1/node-register/register-with-auth
 * @desc 带用户认证的节点注册
 * @access 公开（通过用户名密码认证）
 */
router.post('/register-with-auth', async (req, res) => {
  try {
    const {
      username,
      password,
      nodeId,
      nodeName,
      nodeDescription,
      agentVersion,
      ipAddress,
      port,
      mode,
    } = req.body

    // 参数验证
    if (!username || !password) {
      return ApiResponse.error(
        res,
        ErrorCodes.VALIDATION_FAILED,
        { message: '用户名和密码不能为空' },
        200,
      )
    }

    if (!nodeName) {
      return ApiResponse.error(
        res,
        ErrorCodes.VALIDATION_FAILED,
        { message: '节点名称不能为空' },
        200,
      )
    }

    const result = await nodeService.registerWithAuth({
      username,
      password,
      nodeId,
      nodeName,
      nodeDescription,
      agentVersion,
      ipAddress: ipAddress || req.ip,
      port,
      mode: mode || 'online',
      userAgent: req.get('user-agent') || '',
    })

    return ApiResponse.success(res, result, null, {}, 201)
  } catch (error) {
    console.error('节点注册失败:', error)
    return respondRouteError(res, error, ErrorCodes.PROJECT_OPERATION_FAILED, 200)
  }
})

/**
 * @route GET /api/v1/node-register/:nodeId/approval-status
 * @desc 查询节点审批状态（无需认证）
 * @access 公开（用于轮询）
 */
router.get('/:nodeId/approval-status', async (req, res) => {
  try {
    const { nodeId } = req.params

    if (!nodeId) {
      return ApiResponse.error(
        res,
        ErrorCodes.VALIDATION_FAILED,
        { message: '节点ID不能为空' },
        200,
      )
    }

    const result = await nodeService.getApprovalStatus(nodeId)

    return ApiResponse.success(res, result)
  } catch (error) {
    console.error('查询审批状态失败:', error)
    return respondRouteError(res, error, ErrorCodes.RESOURCE_NOT_FOUND, 200)
  }
})

module.exports = router
