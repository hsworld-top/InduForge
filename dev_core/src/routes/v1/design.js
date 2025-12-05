/**
 * Design Routes - 设计中心路由
 * RESTful API 端点
 * Requirements: 7.1, 7.2, 7.3, 7.4, 7.5
 */
const express = require('express');
const router = express.Router();
const { authenticateToken } = require('../../middlewares/auth');
const designController = require('../../controllers/designController');

// 所有路由都需要认证
router.use(authenticateToken);

/**
 * 获取项目的页面列表
 * GET /api/v1/projects/:projectId/design/pages
 * Requirements: 7.1
 */
router.get('/projects/:projectId/design/pages', designController.getPages);

/**
 * 获取单个页面的完整 Schema
 * GET /api/v1/projects/:projectId/design/pages/:pageId
 * Requirements: 7.2
 */
router.get('/projects/:projectId/design/pages/:pageId', designController.getPage);

/**
 * 创建新页面
 * POST /api/v1/projects/:projectId/design/pages
 * Body: { name: string, type?: 'page' | 'folder' | 'dialog', parentId?: string }
 * Requirements: 7.3
 */
router.post('/projects/:projectId/design/pages', designController.createPage);

/**
 * 更新页面 Schema
 * PUT /api/v1/projects/:projectId/design/pages/:pageId
 * Body: { schema: PageSchema }
 * Requirements: 7.4
 */
router.put('/projects/:projectId/design/pages/:pageId', designController.updatePage);

/**
 * 删除页面
 * DELETE /api/v1/projects/:projectId/design/pages/:pageId
 * Requirements: 7.5
 */
router.delete('/projects/:projectId/design/pages/:pageId', designController.deletePage);

/**
 * 重命名页面
 * PATCH /api/v1/projects/:projectId/design/pages/:pageId/rename
 * Body: { name: string }
 */
router.patch('/projects/:projectId/design/pages/:pageId/rename', designController.renamePage);

/**
 * 移动页面
 * PATCH /api/v1/projects/:projectId/design/pages/:pageId/move
 * Body: { parentId?: string | null, sortOrder?: number }
 */
router.patch('/projects/:projectId/design/pages/:pageId/move', designController.movePage);

module.exports = router;
