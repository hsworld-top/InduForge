const express = require('express');
const Joi = require('joi');
const { validate } = require('../../middlewares/validate');
const { authenticateToken, requireRole } = require('../../middlewares/auth');
const { logger } = require('../../utils/logger');
const ApiResponse = require('../../utils/response');
const ErrorCodes = require('../../constants/errorCodes');
const AppError = require('../../utils/AppError');
const { User } = require('../../models');

const router = express.Router();

// 角色与权限说明（与中间件保持一致）
const ROLE_PERMISSIONS = {
  SUPER_ADMIN: {
    description: '超级管理员（全局权限）'
  },
  SYSTEM_ADMIN: {
    description: '系统管理员（管理用户与工程）'
  },
  PROJECT_ADMIN: {
    description: '工程管理员（工程全权限）'
  },
  OPS_ADMIN: {
    description: '运维管理员（工程运维操作与更新）'
  },
  USER_ADMIN: {
    description: '用户管理员（用户全权限）'
  }
};

/**
 * @swagger
 * /api/v1/roles:
 *   get:
 *     summary: 获取系统支持的角色列表
 *     tags: [角色]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: 获取成功
 */
router.get('/', authenticateToken, (req, res) => {
  return ApiResponse.success(res, { roles: Object.entries(ROLE_PERMISSIONS).map(([key, meta]) => ({ key, ...meta })) });
});

// 内置角色禁止增删改：统一返回 405
router.post('/', authenticateToken, (req, res) => {
  return ApiResponse.error(res, ErrorCodes.REQUEST_METHOD_NOT_ALLOWED, {}, 405);
});
router.put('/', authenticateToken, (req, res) => {
  return ApiResponse.error(res, ErrorCodes.REQUEST_METHOD_NOT_ALLOWED, {}, 405);
});
router.delete('/', authenticateToken, (req, res) => {
  return ApiResponse.error(res, ErrorCodes.REQUEST_METHOD_NOT_ALLOWED, {}, 405);
});

/**
 * @swagger
 * /api/v1/roles/me:
 *   get:
 *     summary: 获取当前用户角色信息
 *     tags: [角色]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: 获取成功
 */
router.get('/me', authenticateToken, (req, res) => {
  const role = req.user.role;
  return ApiResponse.success(res, {
    role,
    tenantId: req.user.tenantId,
    description: ROLE_PERMISSIONS[role]?.description || ''
  });
});

/**
 * @swagger
 * /api/v1/roles/users/{id}:
 *   put:
 *     summary: 修改指定用户的角色
 *     tags: [角色]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - role
 *             properties:
 *               role:
 *                 type: string
 *                 enum: [SYSTEM_ADMIN, PROJECT_ADMIN, OPS_ADMIN, USER_ADMIN]
 *     responses:
 *       200:
 *         description: 更新成功
 */
router.put('/users/:id',
  authenticateToken,
  requireRole('SUPER_ADMIN', 'SYSTEM_ADMIN'),
  validate(Joi.object({
    params: Joi.object({ id: Joi.string().uuid().required() }),
    body: Joi.object({ role: Joi.string().valid('SYSTEM_ADMIN','PROJECT_ADMIN','OPS_ADMIN','USER_ADMIN').required() })
  })),
  async (req, res) => {
    try {
      const { id } = req.params;
      const { role } = req.body;

      const user = await User.findByPk(id);
      if (!user) {
        return ApiResponse.error(res, ErrorCodes.USER_NOT_FOUND, {}, 404);
      }

      // 非超级管理员只能修改同租户用户
      if (req.user.role !== 'SUPER_ADMIN' && user.tenantId !== req.user.tenantId) {
        return ApiResponse.error(res, ErrorCodes.PERMISSION_TENANT_MISMATCH, {}, 403);
      }

      // 不允许将任何人升级为 SUPER_ADMIN
      if (role === 'SUPER_ADMIN') {
        return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: 'Cannot assign SUPER_ADMIN via API' }, 400);
      }

      await user.update({ role });
      return ApiResponse.success(res, { user: { id: user.id, role: user.role } }, 'role_update_success');
    } catch (error) {
      logger.error('Update user role error', { error: error.message, requestId: req.requestId });
      return ApiResponse.error(res, ErrorCodes.USER_ROLE_CHANGE_FAILED, {}, 500);
    }
  }
);

module.exports = router;
