/**
 * Page Lock Controller - 页面锁控制器
 * 处理页面锁定相关的 HTTP 请求
 */
const designService = require("../services/designService");
const { Project } = require("../models");
const ApiResponse = require("../utils/response");
const AppError = require("../utils/AppError");
const ErrorCodes = require("../constants/errorCodes");
const { getErrorMessage } = require("../utils/i18n");

/**
 * 检查用户是否有权限访问指定工程
 * @param {Object} req - Express 请求对象
 * @param {string} projectId - 工程ID
 */
async function checkProjectAccess(req, projectId) {
  if (req.user.role === "SUPER_ADMIN" || req.user.role === "SYSTEM_ADMIN") {
    return;
  }

  const project = await Project.findOne({
    where: { id: projectId, tenantId: req.user.tenantId },
  });

  if (!project) {
    throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
      message: "无权访问此工程",
    });
  }
}

/**
 * 获取页面锁状态
 * GET /api/v1/pages/:pageId/lock
 */
async function getPageLock(req, res, next) {
  try {
    const { pageId } = req.params;
    const pageDetail = await designService.getPageDetail(pageId);
    await checkProjectAccess(req, pageDetail.projectId);

    const data = await designService.getPageLockStatus(pageId);
    return ApiResponse.success(res, data);
  } catch (error) {
    return next(error);
  }
}

/**
 * 获取页面锁
 * POST /api/v1/pages/:pageId/lock
 */
async function acquirePageLock(req, res, next) {
  try {
    const { pageId } = req.params;
    const pageDetail = await designService.getPageDetail(pageId);
    await checkProjectAccess(req, pageDetail.projectId);

    const result = await designService.acquirePageLock(
      pageId,
      req.user.id,
      req.user.username
    );

    if (result.success) {
      return ApiResponse.success(res, result.data);
    }

    const language = res.locals.language || res.req.language || "zh-CN";
    const requestId = res.locals.requestId || res.req.requestId;
    res.locals.language = language;
    res.locals.requestId = requestId;
    res.setHeader("Content-Language", language);

    return res.status(200).json({
      success: false,
      errorCode: ErrorCodes.DESIGN_PAGE_LOCKED,
      message: getErrorMessage(language, ErrorCodes.DESIGN_PAGE_LOCKED, {
        message: "页面已被其他用户锁定",
      }),
      requestId,
      data: result.data,
    });
  } catch (error) {
    return next(error);
  }
}

/**
 * 释放页面锁
 * DELETE /api/v1/pages/:pageId/lock
 */
async function releasePageLock(req, res, next) {
  try {
    const { pageId } = req.params;
    const pageDetail = await designService.getPageDetail(pageId);
    await checkProjectAccess(req, pageDetail.projectId);

    const data = await designService.releasePageLock(pageId, req.user.id);
    return ApiResponse.success(res, data);
  } catch (error) {
    return next(error);
  }
}

/**
 * 页面锁心跳
 * POST /api/v1/pages/:pageId/lock/heartbeat
 */
async function heartbeatPageLock(req, res, next) {
  try {
    const { pageId } = req.params;
    const pageDetail = await designService.getPageDetail(pageId);
    await checkProjectAccess(req, pageDetail.projectId);

    const data = await designService.heartbeatPageLock(pageId, req.user.id);
    return ApiResponse.success(res, data);
  } catch (error) {
    return next(error);
  }
}

/**
 * 页面锁释放（Beacon 用）
 * POST /api/v1/pages/:pageId/lock/release
 */
async function releasePageLockBeacon(req, res, next) {
  try {
    const { pageId } = req.params;
    const pageDetail = await designService.getPageDetail(pageId);
    await checkProjectAccess(req, pageDetail.projectId);

    const data = await designService.releasePageLock(pageId, req.user.id);
    return ApiResponse.success(res, data);
  } catch (error) {
    return next(error);
  }
}

/**
 * 强制释放页面锁
 * DELETE /api/v1/pages/:pageId/lock/force
 */
async function forceReleasePageLock(req, res, next) {
  try {
    const { pageId } = req.params;
    const pageDetail = await designService.getPageDetail(pageId);
    await checkProjectAccess(req, pageDetail.projectId);

    if (!["SUPER_ADMIN", "SYSTEM_ADMIN"].includes(req.user.role)) {
      throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
        message: "无权限强制释放页面锁",
      });
    }

    const data = await designService.forceReleasePageLock(pageId);
    return ApiResponse.success(res, data);
  } catch (error) {
    return next(error);
  }
}

module.exports = {
  getPageLock,
  acquirePageLock,
  releasePageLock,
  heartbeatPageLock,
  releasePageLockBeacon,
  forceReleasePageLock,
};
