/**
 * 发布路由 - 工程发布 API
 * @description 提供工程发布、制品下载等 API
 */
const express = require("express");
const router = express.Router();
const path = require("path");
const publishService = require("../../services/publishService");
const { authenticate, checkProjectAccess } = require("../../middlewares/auth");

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

    if (!version) {
      return res.status(400).json({ error: "版本号不能为空" });
    }

    const result = await publishService.publish(projectId, {
      version,
      name,
      description,
      type,
      deployedBy,
    });

    res.json({
      success: true,
      data: result,
      message: "发布成功",
    });
  } catch (error) {
    console.error("发布失败:", error);
    res.status(400).json({ error: error.message });
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

    const result = await publishService.getDeploymentsByProject(projectId, {
      page: parseInt(page) || 1,
      pageSize: parseInt(pageSize) || 20,
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
 * @route GET /api/v1/publish/deployment/:id
 * @desc 获取发布详情
 * @access 工程成员
 */
router.get("/deployment/:id", authenticate, async (req, res) => {
  try {
    const { id } = req.params;
    const deployment = await publishService.getDeployment(id);

    if (!deployment) {
      return res.status(404).json({ error: "发布记录不存在" });
    }

    res.json({
      success: true,
      data: deployment,
    });
  } catch (error) {
    console.error("获取发布详情失败:", error);
    res.status(500).json({ error: error.message });
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
    const filePath = await publishService.getArtifactPath(id);
    const fileName = path.basename(filePath);

    res.download(filePath, fileName);
  } catch (error) {
    console.error("下载制品失败:", error);
    res.status(404).json({ error: error.message });
  }
});

module.exports = router;
