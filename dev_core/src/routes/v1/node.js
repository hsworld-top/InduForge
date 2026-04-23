/**
 * 节点管理路由 - 节点注册、心跳、状态查询
 * @description 提供 NodeAgent 注册、心跳上报以及运维界面节点管理 API
 */
const express = require("express");
const router = express.Router();
const nodeService = require("../../services/nodeService");
const { authenticate } = require("../../middlewares/auth");
const ApiResponse = require("../../utils/response");
const AppError = require("../../utils/AppError");
const ErrorCodes = require("../../constants/errorCodes");
const NODE_ADMIN_ROLES = ["SYSTEM_ADMIN", "OPS_ADMIN"];

/**
 * 路由统一错误响应：
 * - 业务错误：HTTP 200 + 非 0 code
 * - 技术异常：保留 5xx
 */
function respondRouteError(res, error, _fallbackCode, fallbackStatus = 500) {
  const isAppError = error instanceof AppError || error?.name === "AppError";
  if (isAppError && error?.errorCode && error?.statusCode) {
    const normalizedStatus = Number(error.statusCode) >= 500 ? Number(error.statusCode) : 200;
    const options = error.options || (error.message ? { message: error.message } : {});
    return ApiResponse.error(res, error.errorCode, options, normalizedStatus);
  }

  const normalizedFallbackStatus = Number(fallbackStatus) >= 500 ? Number(fallbackStatus) : 500;
  const options = error?.message ? { message: error.message } : {};
  return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, options, normalizedFallbackStatus);
}

/**
 * @route POST /api/v1/nodes/:nodeId/heartbeat
 * @desc 节点心跳上报
 * @access Agent（使用节点令牌）
 */
router.post("/:nodeId/heartbeat", async (req, res) => {
  try {
    const { nodeId } = req.params;
    const heartbeatData = req.body;
    const registrationToken =
      req.get("X-Registration-Token") || heartbeatData.registrationToken || "";

    // 从请求中获取 IP
    heartbeatData.ipAddress = heartbeatData.ipAddress || req.ip;

    const result = await nodeService.heartbeat(nodeId, heartbeatData, registrationToken);

    return ApiResponse.success(res, result);
  } catch (error) {
    console.error("心跳上报失败:", error);
    return respondRouteError(res, error, ErrorCodes.PROJECT_OPERATION_FAILED, 200);
  }
});

/**
 * @route POST /api/v1/nodes/:nodeId/deployment-status
 * @desc 节点部署状态回调
 * @access Agent
 */
router.post("/:nodeId/deployment-status", async (req, res) => {
  try {
    const { nodeId } = req.params;
    const { deploymentId, status, error, startedAt, stoppedAt, message } = req.body;
    const registrationToken = req.get("X-Registration-Token") || req.body?.registrationToken || "";

    if (!deploymentId || !status) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: "deploymentId 和 status 不能为空" }, 200);
    }

    const result = await nodeService.updateDeploymentStatus(nodeId, deploymentId, {
      status,
      error,
      startedAt,
      stoppedAt,
      message,
    }, registrationToken);

    return ApiResponse.success(res, { id: result.id, status: result.status });
  } catch (error) {
    console.error("部署状态更新失败:", error);
    return respondRouteError(res, error, ErrorCodes.PROJECT_OPERATION_FAILED, 200);
  }
});

/**
 * @route POST /api/v1/nodes/:nodeId/offline
 * @desc 节点主动下线通知
 * @access Agent
 */
router.post("/:nodeId/offline", async (req, res) => {
  try {
    const { nodeId } = req.params;
    const { reason } = req.body || {};
    const registrationToken = req.get("X-Registration-Token") || req.body?.registrationToken || "";

    const result = await nodeService.offline(nodeId, { reason: reason || "agent_shutdown" }, registrationToken);
    return ApiResponse.success(res, result);
  } catch (error) {
    console.error("节点主动下线失败:", error);
    return respondRouteError(res, error, ErrorCodes.PROJECT_OPERATION_FAILED, 200);
  }
});

/**
 * @route GET /api/v1/nodes
 * @desc 获取节点列表
 * @access 租户管理员
 */
router.get("/", authenticate, async (req, res) => {
  try {
    const tenantId = req.user.tenantId;
    const { page, pageSize, status, search, approvalStatus } = req.query;

    const result = await nodeService.list(tenantId, {
      page: parseInt(page) || 1,
      pageSize: parseInt(pageSize) || 20,
      status,
      search,
      approvalStatus,
    });

    return ApiResponse.success(res, result);
  } catch (error) {
    console.error("获取节点列表失败:", error);
    return respondRouteError(res, error, ErrorCodes.INTERNAL_SERVER_ERROR, 500);
  }
});

/**
 * @route GET /api/v1/nodes/:nodeId
 * @desc 获取节点详情
 * @access 租户管理员
 */
router.get("/:nodeId", authenticate, async (req, res) => {
  try {
    const { nodeId } = req.params;
    const node = await nodeService.getById(nodeId);

    // 检查租户权限
    if (node.tenantId !== req.user.tenantId) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, { message: "无权访问此节点" }, 200);
    }

    return ApiResponse.success(res, node);
  } catch (error) {
    console.error("获取节点详情失败:", error);
    return respondRouteError(res, error, ErrorCodes.RESOURCE_NOT_FOUND, 200);
  }
});

/**
 * @route PUT /api/v1/nodes/:nodeId
 * @desc 更新节点信息
 * @access 租户管理员
 */
router.put("/:nodeId", authenticate, async (req, res) => {
  try {
    const { nodeId } = req.params;
    const { name, description, port, config } = req.body;
    const userId = req.user.id;

    // 先获取节点检查权限
    const existing = await nodeService.getById(nodeId);
    if (existing.tenantId !== req.user.tenantId) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, { message: "无权修改此节点" }, 200);
    }

    const node = await nodeService.update(nodeId, { name, description, port, config }, userId);

    return ApiResponse.success(res, node);
  } catch (error) {
    console.error("更新节点失败:", error);
    return respondRouteError(res, error, ErrorCodes.PROJECT_OPERATION_FAILED, 200);
  }
});

/**
 * @route DELETE /api/v1/nodes/:nodeId
 * @desc 删除节点
 * @access 租户管理员
 */
router.delete("/:nodeId", authenticate, async (req, res) => {
  try {
    const { nodeId } = req.params;
    const role = req.user?.role;
    if (!NODE_ADMIN_ROLES.includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, { message: "仅平台管理员或运维管理员可删除节点" }, 200);
    }

    // 先获取节点检查权限
    const existing = await nodeService.getById(nodeId);
    if (existing.tenantId !== req.user.tenantId) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, { message: "无权删除此节点" }, 200);
    }

    await nodeService.delete(nodeId);

    return ApiResponse.success(res, null);
  } catch (error) {
    console.error("删除节点失败:", error);
    return respondRouteError(res, error, ErrorCodes.PROJECT_OPERATION_FAILED, 200);
  }
});

/**
 * @route PUT /api/v1/nodes/:nodeId/approve
 * @desc 审批通过节点
 * @access 租户管理员
 */
router.put("/:nodeId/approve", authenticate, async (req, res) => {
  try {
    const { nodeId } = req.params;
    const userId = req.user.id;
    const role = req.user?.role;
    if (!NODE_ADMIN_ROLES.includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, { message: "仅平台管理员或运维管理员可审批节点" }, 200);
    }

    // 权限检查
    const node = await nodeService.getById(nodeId);
    if (node.tenantId !== req.user.tenantId) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, { message: "无权审批此节点" }, 200);
    }

    const result = await nodeService.approve(nodeId, userId);

    return ApiResponse.success(res, result);
  } catch (error) {
    console.error("审批节点失败:", error);
    return respondRouteError(res, error, ErrorCodes.PROJECT_OPERATION_FAILED, 200);
  }
});

/**
 * @route PUT /api/v1/nodes/:nodeId/reject
 * @desc 拒绝节点注册
 * @access 租户管理员
 */
router.put("/:nodeId/reject", authenticate, async (req, res) => {
  try {
    const { nodeId } = req.params;
    const userId = req.user.id;
    const role = req.user?.role;
    if (!NODE_ADMIN_ROLES.includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, { message: "仅平台管理员或运维管理员可审批节点" }, 200);
    }

    // 权限检查
    const node = await nodeService.getById(nodeId);
    if (node.tenantId !== req.user.tenantId) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, { message: "无权操作此节点" }, 200);
    }

    const result = await nodeService.reject(nodeId, userId);

    return ApiResponse.success(res, result);
  } catch (error) {
    console.error("拒绝节点失败:", error);
    return respondRouteError(res, error, ErrorCodes.PROJECT_OPERATION_FAILED, 200);
  }
});

module.exports = router;
