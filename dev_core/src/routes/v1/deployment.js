/**
 * 部署管理路由 - 版本管理、部署、停止、回滚
 * @description 提供工程发布版本管理和部署操作 API，支持DEV/RELEASE模式
 */
const express = require("express");
const router = express.Router();
const deploymentService = require("../../services/deploymentService");
const { authenticate } = require("../../middlewares/auth");

/**
 * @route GET /api/v1/deployments/project/:projectId
 * @desc 获取工程的发布版本列表
 * @access 工程成员
 */
router.get("/project/:projectId", authenticate, async (req, res) => {
  try {
    const { projectId } = req.params;
    const { page, pageSize, status, type, mode } = req.query;

    const result = await deploymentService.listByProject(projectId, {
      page: parseInt(page) || 1,
      pageSize: parseInt(pageSize) || 20,
      status,
      type,
      mode,
    });

    res.json({
      success: true,
      data: result,
    });
  } catch (error) {
    console.error("获取发布版本列表失败:", error);
    res.status(500).json({ error: error.message });
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
    const deployment = await deploymentService.getById(id);

    res.json({
      success: true,
      data: deployment,
    });
  } catch (error) {
    console.error("获取发布版本详情失败:", error);
    res.status(404).json({ error: error.message });
  }
});

/**
 * @route POST /api/v1/deployments/:id/deploy
 * @desc 部署工程到节点（支持DEV/RELEASE模式）
 * @access 工程管理员
 */
router.post("/:id/deploy", authenticate, async (req, res) => {
  try {
    const { id: deploymentId } = req.params;
    const { nodeIds, mode, runtimeConfig } = req.body;
    const deployedBy = req.user.id;

    if (!nodeIds || !Array.isArray(nodeIds) || nodeIds.length === 0) {
      return res.status(400).json({ error: "请选择至少一个目标节点" });
    }

    if (!mode || !["DEV", "RELEASE"].includes(mode)) {
      return res.status(400).json({ error: "请指定运行模式 DEV 或 RELEASE" });
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

    res.json({
      success: true,
      data: {
        results,
        summary: {
          total: nodeIds.length,
          success: successCount,
          failed: failCount,
        },
      },
      message: `部署任务已下发：${successCount} 个成功，${failCount} 个失败${
        mode === "RELEASE" ? "（将停止DEV实例）" : ""
      }`,
    });
  } catch (error) {
    console.error("部署失败:", error);
    res.status(400).json({ error: error.message });
  }
});

/**
 * @route POST /api/v1/deployments/node-deployment/:id/start
 * @desc 启动工程
 * @access 工程管理员
 */
router.post("/node-deployment/:id/start", authenticate, async (req, res) => {
  try {
    const { id } = req.params;
    const result = await deploymentService.start(id);

    res.json({
      success: true,
      data: { id: result.id, status: result.status },
      message: "启动指令已下发",
    });
  } catch (error) {
    console.error("启动失败:", error);
    res.status(400).json({ error: error.message });
  }
});

/**
 * @route POST /api/v1/deployments/node-deployment/:id/stop
 * @desc 停止工程
 * @access 工程管理员
 */
router.post("/node-deployment/:id/stop", authenticate, async (req, res) => {
  try {
    const { id } = req.params;
    const result = await deploymentService.stop(id);

    res.json({
      success: true,
      data: { id: result.id, status: result.status },
      message: "停止指令已下发",
    });
  } catch (error) {
    console.error("停止失败:", error);
    res.status(400).json({ error: error.message });
  }
});

/**
 * @route POST /api/v1/deployments/node-deployment/:id/restart
 * @desc 重启工程
 * @access 工程管理员
 */
router.post("/node-deployment/:id/restart", authenticate, async (req, res) => {
  try {
    const { id } = req.params;
    const result = await deploymentService.restart(id);

    res.json({
      success: true,
      data: { id: result.id, status: result.status },
      message: "重启指令已下发",
    });
  } catch (error) {
    console.error("重启失败:", error);
    res.status(400).json({ error: error.message });
  }
});

/**
 * @route DELETE /api/v1/deployments/node-deployment/:id
 * @desc 撤销部署
 * @access 工程管理员
 */
router.delete("/node-deployment/:id", authenticate, async (req, res) => {
  try {
    const { id } = req.params;
    const result = await deploymentService.undeploy(id);

    res.json({
      success: true,
      data: { id: result.id, status: result.status },
      message: "撤销部署成功",
    });
  } catch (error) {
    console.error("撤销部署失败:", error);
    res.status(400).json({ error: error.message });
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
      const mode = await deploymentService.getNodeProjectMode(
        projectId,
        nodeId
      );

      res.json({
        success: true,
        data: { mode },
      });
    } catch (error) {
      console.error("获取部署模式失败:", error);
      res.status(400).json({ error: error.message });
    }
  }
);

/**
 * @route POST /api/v1/deployments/:id/rollback
 * @desc 回滚到指定版本（仅RELEASE模式）
 * @access 工程管理员
 */
router.post("/:id/rollback", authenticate, async (req, res) => {
  try {
    const { id: deploymentId } = req.params;
    const { nodeId } = req.body;
    const deployedBy = req.user.id;

    if (!nodeId) {
      return res.status(400).json({ error: "请指定目标节点" });
    }

    const result = await deploymentService.rollback(
      nodeId,
      deploymentId,
      deployedBy
    );

    res.json({
      success: true,
      data: { id: result.id, status: result.status },
      message: "回滚任务已创建",
    });
  } catch (error) {
    console.error("回滚失败:", error);
    res.status(400).json({ error: error.message });
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

    res.json({
      success: true,
      data: result,
    });
  } catch (error) {
    console.error("获取部署历史失败:", error);
    res.status(500).json({ error: error.message });
  }
});

module.exports = router;
