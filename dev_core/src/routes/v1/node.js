/**
 * 节点管理路由 - 节点注册、心跳、状态查询
 * @description 提供 NodeAgent 注册、心跳上报以及运维界面节点管理 API
 */
const express = require("express");
const router = express.Router();
const nodeService = require("../../services/nodeService");
const { authenticate } = require("../../middlewares/auth");

/**
 * @route POST /api/v1/nodes/:nodeId/heartbeat
 * @desc 节点心跳上报
 * @access Agent（使用节点令牌）
 */
router.post("/:nodeId/heartbeat", async (req, res) => {
  try {
    const { nodeId } = req.params;
    const heartbeatData = req.body;

    // 从请求中获取 IP
    heartbeatData.ipAddress = heartbeatData.ipAddress || req.ip;

    const result = await nodeService.heartbeat(nodeId, heartbeatData);

    res.json({
      success: true,
      data: result,
    });
  } catch (error) {
    console.error("心跳上报失败:", error);
    res.status(400).json({ error: error.message });
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

    if (!deploymentId || !status) {
      return res.status(400).json({ error: "deploymentId 和 status 不能为空" });
    }

    const result = await nodeService.updateDeploymentStatus(nodeId, deploymentId, {
      status,
      error,
      startedAt,
      stoppedAt,
      message,
    });

    res.json({
      success: true,
      data: { id: result.id, status: result.status },
    });
  } catch (error) {
    console.error("部署状态更新失败:", error);
    res.status(400).json({ error: error.message });
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

    res.json({
      success: true,
      data: result,
    });
  } catch (error) {
    console.error("获取节点列表失败:", error);
    res.status(500).json({ error: error.message });
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
      return res.status(403).json({ error: "无权访问此节点" });
    }

    res.json({
      success: true,
      data: node,
    });
  } catch (error) {
    console.error("获取节点详情失败:", error);
    res.status(404).json({ error: error.message });
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
      return res.status(403).json({ error: "无权修改此节点" });
    }

    const node = await nodeService.update(nodeId, { name, description, port, config }, userId);

    res.json({
      success: true,
      data: node,
      message: "节点更新成功",
    });
  } catch (error) {
    console.error("更新节点失败:", error);
    res.status(400).json({ error: error.message });
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

    // 先获取节点检查权限
    const existing = await nodeService.getById(nodeId);
    if (existing.tenantId !== req.user.tenantId) {
      return res.status(403).json({ error: "无权删除此节点" });
    }

    await nodeService.delete(nodeId);

    res.json({
      success: true,
      message: "节点删除成功",
    });
  } catch (error) {
    console.error("删除节点失败:", error);
    res.status(400).json({ error: error.message });
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

    // 权限检查
    const node = await nodeService.getById(nodeId);
    if (node.tenantId !== req.user.tenantId) {
      return res.status(403).json({ error: "无权审批此节点" });
    }

    const result = await nodeService.approve(nodeId, userId);

    res.json({
      success: true,
      data: result,
      message: "节点审批通过",
    });
  } catch (error) {
    console.error("审批节点失败:", error);
    res.status(400).json({ error: error.message });
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

    // 权限检查
    const node = await nodeService.getById(nodeId);
    if (node.tenantId !== req.user.tenantId) {
      return res.status(403).json({ error: "无权操作此节点" });
    }

    const result = await nodeService.reject(nodeId, userId);

    res.json({
      success: true,
      data: result,
      message: "节点申请已拒绝",
    });
  } catch (error) {
    console.error("拒绝节点失败:", error);
    res.status(400).json({ error: error.message });
  }
});

module.exports = router;
