/**
 * Design Routes - 设计中心路由
 * RESTful API 端点
 * Requirements: 7.1, 7.2, 7.3, 7.4, 7.5
 */
const express = require("express");
const router = express.Router();
const { authenticateToken } = require("../../middlewares/auth");
const designController = require("../../controllers/designController");

// 所有路由都需要认证
router.use(authenticateToken);

/**
 * 获取项目的页面列表
 * GET /api/v1/projects/:projectId/pages
 * Requirements: 7.1
 */
router.get("/projects/:projectId/pages", designController.getPages);

/**
 * 获取单个页面的完整 Schema
 * GET /api/v1/projects/:projectId/pages/:pageId
 * Requirements: 7.2
 */
router.get("/projects/:projectId/pages/:pageId", designController.getPage);

/**
 * 创建新页面
 * POST /api/v1/projects/:projectId/pages
 * Body: { name: string, type?: 'page' | 'folder' | 'dialog', parentId?: string }
 * Requirements: 7.3
 */
router.post("/projects/:projectId/pages", designController.createPage);

/**
 * 更新页面 Schema
 * PUT /api/v1/projects/:projectId/pages/:pageId
 * Body: { schema: PageSchema }
 * Requirements: 7.4
 */
router.put("/projects/:projectId/pages/:pageId", designController.updatePage);

/**
 * 删除页面
 * DELETE /api/v1/projects/:projectId/pages/:pageId
 * Requirements: 7.5
 */
router.delete(
  "/projects/:projectId/pages/:pageId",
  designController.deletePage
);

/**
 * 重命名页面
 * PATCH /api/v1/projects/:projectId/pages/:pageId/rename
 * Body: { name: string }
 */
router.patch(
  "/projects/:projectId/pages/:pageId/rename",
  designController.renamePage
);

/**
 * 移动页面
 * PATCH /api/v1/projects/:projectId/pages/:pageId/move
 * Body: { parentId?: string | null, sortOrder?: number }
 */
router.patch(
  "/projects/:projectId/pages/:pageId/move",
  designController.movePage
);

/**
 * 工程级别全局变量
 * GET /api/v1/design/projects/:projectId/variables
 * PUT /api/v1/design/projects/:projectId/variables
 */
router.get(
  "/projects/:projectId/variables",
  designController.getProjectVariables
);
router.put(
  "/projects/:projectId/variables",
  designController.updateProjectVariables
);

/**
 * 更新项目入口配置
 * PUT /api/v1/projects/:projectId/entry
 * Body: { homePageId?: string, loginPageId?: string, logoutPageId?: string }
 */
router.put("/projects/:projectId/entry", designController.updateEntryConfig);

router.get(
  "/projects/:projectId/settings",
  designController.getProjectSettings
);
router.put(
  "/projects/:projectId/settings",
  designController.updateProjectSettings
);

module.exports = router;
