/**
 * 节点注册路由 - 带用户认证的节点注册接口
 * @description 提供 NodeAgent 初始化向导使用的注册接口
 */
const express = require("express");
const router = express.Router();
const nodeService = require("../../services/nodeService");

/**
 * @route POST /api/v1/node-register/register-with-auth
 * @desc 带用户认证的节点注册
 * @access 公开（通过用户名密码认证）
 */
router.post("/register-with-auth", async (req, res) => {
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
    } = req.body;

    // 参数验证
    if (!username || !password) {
      return res.status(400).json({
        success: false,
        error: "用户名和密码不能为空",
      });
    }

    if (!nodeName) {
      return res.status(400).json({
        success: false,
        error: "节点名称不能为空",
      });
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
      mode: mode || "online",
      userAgent: req.get("user-agent") || "",
    });

    res.status(201).json({
      success: true,
      data: result,
    });
  } catch (error) {
    console.error("节点注册失败:", error);
    res.status(400).json({
      success: false,
      error: error.message,
    });
  }
});

/**
 * @route GET /api/v1/node-register/:nodeId/approval-status
 * @desc 查询节点审批状态（无需认证）
 * @access 公开（用于轮询）
 */
router.get("/:nodeId/approval-status", async (req, res) => {
  try {
    const { nodeId } = req.params;

    if (!nodeId) {
      return res.status(400).json({
        success: false,
        error: "节点ID不能为空",
      });
    }

    const result = await nodeService.getApprovalStatus(nodeId);

    res.json({
      success: true,
      data: result,
    });
  } catch (error) {
    console.error("查询审批状态失败:", error);
    res.status(404).json({
      success: false,
      error: error.message,
    });
  }
});

module.exports = router;
