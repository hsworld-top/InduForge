/**
 * 发布路由 - 工程发布 API
 * @description 提供工程发布、制品下载等 API
 */
const express = require("express");
const router = express.Router();
const path = require("path");
const publishService = require("../../services/publishService");
const { authenticate, checkProjectAccess } = require("../../middlewares/auth");
const ApiResponse = require("../../utils/response");
const ErrorCodes = require("../../constants/errorCodes");

/**
 * @route POST /api/v1/publish/:projectId
 * @desc 发布工程
 * @access 工程管理员
 */
router.post("/:projectId", authenticate, async (req, res) => {
  try {
    const { projectId } = req.params;
    const { version, name, description, type } = req.body;
    const deployedBy = req.user.id;
    await checkProjectAccess(req, projectId);

    if (!version) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: "版本号不能为空" }, 400);
    }

    const result = await publishService.publish(projectId, {
      version,
      name,
      description,
      type,
      deployedBy,
    });

    return ApiResponse.success(res, result);
  } catch (error) {
    console.error("发布失败:", error);
    return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, { message: error.message }, 400);
  }
});

/**
 * @route GET /api/v1/publish/:projectId/versions
 * @desc 获取工程的发布版本列表
 * @access 工程成员
 */
router.get("/:projectId/versions", authenticate, async (req, res) => {
  try {
    const { projectId } = req.params;
    const { page, pageSize } = req.query;
    await checkProjectAccess(req, projectId);

    const result = await publishService.getDeploymentsByProject(projectId, {
      page: parseInt(page) || 1,
      pageSize: parseInt(pageSize) || 20,
    });

    return ApiResponse.success(res, result);
  } catch (error) {
    console.error("获取发布版本列表失败:", error);
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, { message: error.message }, 500);
  }
});

/**
 * @route GET /api/v1/publish/deployment/:id
 * @desc 获取发布详情
 * @access 工程成员
 */
router.get("/deployment/:id", authenticate, async (req, res) => {
  try {
    const { id } = req.params;
    const deployment = await publishService.getDeployment(id);

    if (!deployment) {
      return ApiResponse.error(res, ErrorCodes.RESOURCE_NOT_FOUND, { message: "发布记录不存在" }, 404);
    }
    await checkProjectAccess(req, deployment.projectId);

    return ApiResponse.success(res, deployment);
  } catch (error) {
    console.error("获取发布详情失败:", error);
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, { message: error.message }, 500);
  }
});

/**
 * @route GET /api/v1/publish/deployment/:id/download
 * @desc 下载 IFP 制品
 * @access 工程成员
 */
router.get("/deployment/:id/download", authenticate, async (req, res) => {
  try {
    const { id } = req.params;
    const deployment = await publishService.getDeployment(id);
    if (!deployment) {
      return ApiResponse.error(res, ErrorCodes.RESOURCE_NOT_FOUND, { message: "发布记录不存在" }, 404);
    }
    await checkProjectAccess(req, deployment.projectId);
    const filePath = await publishService.getArtifactPath(id);
    const fileName = path.basename(filePath);

    res.download(filePath, fileName);
  } catch (error) {
    console.error("下载制品失败:", error);
    return ApiResponse.error(res, ErrorCodes.RESOURCE_NOT_FOUND, { message: error.message }, 404);
  }
});

/**
 * @route DELETE /api/v1/publish/deployment/:id
 * @desc 删除发布记录（仅未被节点部署引用的记录）
 * @access 工程管理员
 */
router.delete("/deployment/:id", authenticate, async (req, res) => {
  try {
    const { id } = req.params;
    const deployment = await publishService.getDeployment(id);
    if (!deployment) {
      return ApiResponse.error(res, ErrorCodes.RESOURCE_NOT_FOUND, { message: "发布记录不存在" }, 404);
    }
    await checkProjectAccess(req, deployment.projectId);

    const result = await publishService.deleteDeployment(id);
    return ApiResponse.success(res, result);
  } catch (error) {
    console.error("删除发布记录失败:", error);
    return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, { message: error.message }, 400);
  }
});

module.exports = router;
