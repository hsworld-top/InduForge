/**
 * 部署管理路由 - 版本管理、部署、停止、回滚
 * @description 提供工程发布版本管理和部署操作 API，支持DEV/RELEASE模式
 */
const express = require("express");
const router = express.Router();
const deploymentService = require("../../services/deploymentService");
const { authenticate, checkProjectAccess, requireCapability } = require("../../middlewares/auth");
const ApiResponse = require("../../utils/response");
const ErrorCodes = require("../../constants/errorCodes");

/**
 * @route GET /api/v1/deployments/project/:projectId
 * @desc 获取工程的发布版本列表
 * @access 工程成员
 */
router.get("/project/:projectId", authenticate, async (req, res) => {
  try {
    const { projectId } = req.params;
    const { page, pageSize, status, type, mode } = req.query;
    await checkProjectAccess(req, projectId);

    const result = await deploymentService.listByProject(projectId, {
      page: parseInt(page) || 1,
      pageSize: parseInt(pageSize) || 20,
      status,
      type,
      mode,
    });

    return ApiResponse.success(res, result);
  } catch (error) {
    console.error("获取发布版本列表失败:", error);
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, { message: error.message }, 500);
  }
});

/**
 * @route GET /api/v1/deployments/:id
 * @desc 获取发布版本详情
 * @access 工程成员
 */
router.get("/:id", authenticate, async (req, res) => {
  try {
    const { id } = req.params;
    const projectId = await deploymentService.getProjectIdByDeploymentId(id);
    await checkProjectAccess(req, projectId);
    const deployment = await deploymentService.getById(id);

    return ApiResponse.success(res, deployment);
  } catch (error) {
    console.error("获取发布版本详情失败:", error);
    return ApiResponse.error(res, ErrorCodes.RESOURCE_NOT_FOUND, { message: error.message }, 404);
  }
});

/**
 * @route POST /api/v1/deployments/project/:projectId/deploy-dev
 * @desc DEV模式按工程直接部署（无需版本）
 * @access 工程管理员
 */
router.post("/project/:projectId/deploy-dev", authenticate, requireCapability("deploy:execute"), async (req, res) => {
  try {
    const { projectId } = req.params;
    const { nodeIds } = req.body;
    const deployedBy = req.user.id;
    await checkProjectAccess(req, projectId);

    if (!nodeIds || !Array.isArray(nodeIds) || nodeIds.length === 0) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: "请选择至少一个目标节点" }, 400);
    }

    const results = await deploymentService.deployDevToNodesByProject(
      projectId,
      nodeIds,
      deployedBy
    );

    const successCount = results.filter((r) => r.success).length;
    const failCount = results.filter((r) => !r.success).length;

    return ApiResponse.success(res, {
        results,
        summary: {
          total: nodeIds.length,
          success: successCount,
          failed: failCount,
        },
      }, "operation_success", { message: `DEV部署任务已下发：${successCount} 个成功，${failCount} 个失败` });
  } catch (error) {
    console.error("DEV部署失败:", error);
    return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, { message: error.message }, 400);
  }
});

/**
 * @route POST /api/v1/deployments/:id/deploy
 * @desc 部署工程到节点（RELEASE模式）
 * @access 工程管理员
 */
router.post("/:id/deploy", authenticate, requireCapability("deploy:execute"), async (req, res) => {
  try {
    const { id: deploymentId } = req.params;
    const { nodeIds, mode, runtimeConfig } = req.body;
    const deployedBy = req.user.id;
    const projectId = await deploymentService.getProjectIdByDeploymentId(deploymentId);
    await checkProjectAccess(req, projectId);

    if (!nodeIds || !Array.isArray(nodeIds) || nodeIds.length === 0) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: "请选择至少一个目标节点" }, 400);
    }

    if (!mode || !["DEV", "RELEASE"].includes(mode)) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: "请指定运行模式 DEV 或 RELEASE" }, 400);
    }

    if (mode === "DEV") {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, {
        message:
          "DEV模式无需版本，请调用 /api/v1/deployments/project/:projectId/deploy-dev",
      }, 400);
    }

    const results = await deploymentService.deployToNodes(
      deploymentId,
      nodeIds,
      mode,
      runtimeConfig || {},
      deployedBy
    );

    const successCount = results.filter((r) => r.success).length;
    const failCount = results.filter((r) => !r.success).length;

    return ApiResponse.success(res, {
        results,
        summary: {
          total: nodeIds.length,
          success: successCount,
          failed: failCount,
        },
      }, "operation_success", { message: `部署任务已下发：${successCount} 个成功，${failCount} 个失败${
        mode === "RELEASE" ? "（将停止DEV实例）" : ""
      }` });
  } catch (error) {
    console.error("部署失败:", error);
    return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, { message: error.message }, 400);
  }
});

/**
 * @route POST /api/v1/deployments/node-deployment/:id/start
 * @desc 启动工程
 * @access 工程管理员
 */
router.post("/node-deployment/:id/start", authenticate, requireCapability("runtime:operate"), async (req, res) => {
  try {
    const { id } = req.params;
    const projectId = await deploymentService.getProjectIdByNodeDeploymentId(id);
    await checkProjectAccess(req, projectId);
    const result = await deploymentService.start(id);

    return ApiResponse.success(res, { id: result.id, status: result.status }, "operation_success", { message: "启动指令已下发" });
  } catch (error) {
    console.error("启动失败:", error);
    return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, { message: error.message }, 400);
  }
});

/**
 * @route POST /api/v1/deployments/node-deployment/:id/stop
 * @desc 停止工程
 * @access 工程管理员
 */
router.post("/node-deployment/:id/stop", authenticate, requireCapability("runtime:operate"), async (req, res) => {
  try {
    const { id } = req.params;
    const projectId = await deploymentService.getProjectIdByNodeDeploymentId(id);
    await checkProjectAccess(req, projectId);
    const result = await deploymentService.stop(id);

    return ApiResponse.success(res, { id: result.id, status: result.status }, "operation_success", { message: "停止指令已下发" });
  } catch (error) {
    console.error("停止失败:", error);
    return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, { message: error.message }, 400);
  }
});

/**
 * @route POST /api/v1/deployments/node-deployment/:id/restart
 * @desc 重启工程
 * @access 工程管理员
 */
router.post("/node-deployment/:id/restart", authenticate, requireCapability("runtime:operate"), async (req, res) => {
  try {
    const { id } = req.params;
    const projectId = await deploymentService.getProjectIdByNodeDeploymentId(id);
    await checkProjectAccess(req, projectId);
    const result = await deploymentService.restart(id);

    return ApiResponse.success(res, { id: result.id, status: result.status }, "operation_success", { message: "重启指令已下发" });
  } catch (error) {
    console.error("重启失败:", error);
    return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, { message: error.message }, 400);
  }
});

/**
 * @route DELETE /api/v1/deployments/node-deployment/:id
 * @desc 撤销部署
 * @access 工程管理员
 */
router.delete("/node-deployment/:id", authenticate, requireCapability("runtime:operate"), async (req, res) => {
  try {
    const { id } = req.params;
    const projectId = await deploymentService.getProjectIdByNodeDeploymentId(id);
    await checkProjectAccess(req, projectId);
    const result = await deploymentService.undeploy(id);

    return ApiResponse.success(res, { id: result.id, status: result.status }, "operation_success", { message: "撤销部署成功" });
  } catch (error) {
    console.error("撤销部署失败:", error);
    return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, { message: error.message }, 400);
  }
});

/**
 * @route GET /api/v1/deployments/project/:projectId/node/:nodeId/mode
 * @desc 获取工程在节点上的当前模式
 * @access 工程成员
 */
router.get(
  "/project/:projectId/node/:nodeId/mode",
  authenticate,
  async (req, res) => {
    try {
      const { projectId, nodeId } = req.params;
      await checkProjectAccess(req, projectId);
      const mode = await deploymentService.getNodeProjectMode(
        projectId,
        nodeId
      );

      return ApiResponse.success(res, { mode });
    } catch (error) {
      console.error("获取部署模式失败:", error);
      return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, { message: error.message }, 400);
    }
  }
);

/**
 * @route POST /api/v1/deployments/:id/rollback
 * @desc 回滚到指定版本（仅RELEASE模式）
 * @access 工程管理员
 */
router.post("/:id/rollback", authenticate, requireCapability("runtime:operate"), async (req, res) => {
  try {
    const { id: deploymentId } = req.params;
    const { nodeId } = req.body;
    const deployedBy = req.user.id;
    const projectId = await deploymentService.getProjectIdByDeploymentId(deploymentId);
    await checkProjectAccess(req, projectId);

    if (!nodeId) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: "请指定目标节点" }, 400);
    }

    const result = await deploymentService.rollback(
      nodeId,
      deploymentId,
      deployedBy
    );

    return ApiResponse.success(res, { id: result.id, status: result.status }, "operation_success", { message: "回滚任务已创建" });
  } catch (error) {
    console.error("回滚失败:", error);
    return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, { message: error.message }, 400);
  }
});

/**
 * @route GET /api/v1/deployments/project/:projectId/nodes
 * @desc 获取工程在节点上的部署关系
 * @access 工程成员
 */
router.get("/project/:projectId/nodes", authenticate, async (req, res) => {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);
    const items = await deploymentService.listNodeDeploymentsByProject(projectId);
    return ApiResponse.success(res, items);
  } catch (error) {
    console.error("获取工程节点部署关系失败:", error);
    return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, { message: error.message }, 400);
  }
});

/**
 * @route GET /api/v1/deployments/node/:nodeId/history
 * @desc 获取节点的部署历史
 * @access 租户管理员
 */
router.get("/node/:nodeId/history", authenticate, async (req, res) => {
  try {
    const { nodeId } = req.params;
    const { page, pageSize } = req.query;

    const result = await deploymentService.getNodeDeploymentHistory(nodeId, {
      page: parseInt(page) || 1,
      pageSize: parseInt(pageSize) || 20,
    });

    return ApiResponse.success(res, result);
  } catch (error) {
    console.error("获取部署历史失败:", error);
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, { message: error.message }, 500);
  }
});

module.exports = router;
