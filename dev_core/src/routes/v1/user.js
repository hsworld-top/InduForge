const express = require('express');
const Joi = require('joi');
const bcrypt = require('bcryptjs');
const { validate } = require('../../middlewares/validate');
const { logger } = require('../../utils/logger');
const { buildTenantWhere } = require('../../utils/scope');
const { Op } = require('sequelize');
const ApiResponse = require('../../utils/response');
const ErrorCodes = require('../../constants/errorCodes');
const AppError = require('../../utils/AppError');
const appConfig = require('../../config/app');

const router = express.Router();

// 导入模型和中间件
const { User, Tenant } = require('../../models');
const { authenticateToken } = require('../../middlewares/auth');

/**
 * @swagger
 * /api/v1/users:
 *   get:
 *     summary: 获取用户列表
 *     tags: [用户管理]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *       - in: query
 *         name: limit
 *         schema:
 *           type: integer
 *       - in: query
 *         name: username
 *         schema:
 *           type: string
 *       - in: query
 *         name: role
 *         schema:
 *           type: string
 *       - in: query
 *         name: status
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: 获取成功
 */
router.get('/', authenticateToken, validate(Joi.object({
  query: Joi.object({
    page: Joi.number().integer().min(1).default(appConfig.pagination.defaultPage),
    limit: Joi.number().integer().min(1).max(appConfig.pagination.maxLimit).default(appConfig.pagination.defaultLimit),
    username: Joi.string().optional(),
    role: Joi.string().valid('SYSTEM_ADMIN','PROJECT_ADMIN','OPS_ADMIN','USER_ADMIN').optional(),
    status: Joi.string().valid('active','inactive','suspended').optional()
  })
})), async (req, res) => {
  try {
    const { page, limit, username, role, status } = req.query;
    const { tenantId: currentUserTenantId, role: currentUserRole } = req.user;

    // 构建查询条件，包含租户隔离和权限控制逻辑
    let where = {};

    // 检查当前用户是否有权限查看用户
    const canViewUsers = ['SYSTEM_ADMIN', 'USER_ADMIN'].includes(currentUserRole);
    if (!canViewUsers) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, {}, 403);
    }

    // 平台管理员和用户管理员都只能查看本租户用户
    where.tenantId = currentUserTenantId;

    // 角色筛选条件：永远不显示超级管理员
    if (role) {
      where.role = {
        [Op.and]: [
          { [Op.ne]: 'SUPER_ADMIN' },  // 永远不显示超级管理员
          { [Op.eq]: role }            // 匹配指定角色
        ]
      };
    } else {
      // 没有角色筛选时，也要排除超级管理员
      where.role = { [Op.ne]: 'SUPER_ADMIN' };
    }

    if (username) where.username = { [Op.like]: `%${username}%` };
    if (status) where.status = status;

    const offset = (parseInt(page) - 1) * parseInt(limit);
    const limitNum = parseInt(limit);
    const { count, rows } = await User.findAndCountAll({
      where,
      include: [{ model: Tenant, as: 'tenant' }],
      limit: limitNum,
      offset,
      order: [['createdAt', 'DESC']],
      attributes: { exclude: ['password'] } // 不返回密码
    });

    return ApiResponse.paginated(res, { users: rows }, {
        total: count,
        page,
        limit: limitNum,
        totalPages: Math.ceil(count / limitNum),
    });
  } catch (error) {
    logger.error('Get users error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/users:
 *   post:
 *     summary: 创建用户
 *     tags: [用户管理]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - username
 *               - password
 *               - role
 *             properties:
 *               username:
 *                 type: string
 *               password:
 *                 type: string
 *               email:
 *                 type: string
 *               fullName:
 *                 type: string
 *               role:
 *                 type: string
 *                 enum: [SYSTEM_ADMIN, PROJECT_ADMIN, OPS_ADMIN, USER_ADMIN]
 *               tenantId:
 *                 type: string
 *     responses:
 *       201:
 *         description: 创建成功
 */
router.post('/', authenticateToken, validate(Joi.object({
  body: Joi.object({
    username: Joi.string().required(),
    password: Joi.string().min(6).required(),
    email: Joi.string().email().optional().allow(''),
    fullName: Joi.string().optional().allow(''),
    role: Joi.string().valid('SYSTEM_ADMIN','PROJECT_ADMIN','OPS_ADMIN','USER_ADMIN').required(),
    tenantId: Joi.string().uuid().optional()
  }).required()
})), async (req, res) => {
  try {
    const { username, password, email, fullName, role, tenantId } = req.body;
    const { tenantId: currentUserTenantId, role: currentUserRole } = req.user;

    // 检查创建用户的权限
    // 只有系统管理员和用户管理员可以创建用户
    const canCreateUsers = ['SYSTEM_ADMIN', 'USER_ADMIN'].includes(currentUserRole);
    if (!canCreateUsers) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, {}, 403);
    }

    // 检查权限
    let targetTenantId = tenantId;
    if (currentUserRole !== 'SYSTEM_ADMIN') {
      // 非系统管理员只能在自己的租户内创建用户
      targetTenantId = currentUserTenantId;
    }

    // 检查用户名是否已存在
    const existingUser = await User.findOne({
      where: {
        username,
        tenantId: targetTenantId
      }
    });
    if (existingUser) {
      return ApiResponse.error(res, ErrorCodes.USER_USERNAME_EXISTS, {}, 400);
    }

    // 创建用户
    const user = await User.create({
      username,
      password,
      email,
      fullName,
      role,
      tenantId: targetTenantId,
    });

    // 返回用户信息（不包含密码）
    const userResponse = { ...user.toJSON() };
    delete userResponse.password;

    return ApiResponse.success(res, { user: userResponse }, 'user_create_success', {}, 201);

  } catch (error) {
    logger.error('Create user error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.USER_CREATE_FAILED, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/users/{id}:
 *   put:
 *     summary: 更新用户
 *     tags: [用户管理]
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
 *             properties:
 *               email:
 *                 type: string
 *               fullName:
 *                 type: string
 *               role:
 *                 type: string
 *               status:
 *                 type: string
 *     responses:
 *       200:
 *         description: 更新成功
 */
router.put('/:id', authenticateToken, validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() }),
  body: Joi.object({
    email: Joi.string().email().optional().allow(''),
    fullName: Joi.string().optional().allow(''),
    role: Joi.string().valid('SYSTEM_ADMIN','PROJECT_ADMIN','OPS_ADMIN','USER_ADMIN').optional(),
    status: Joi.string().valid('active','inactive','suspended').optional().default('active')
  }).min(1)
})), async (req, res) => {
  try {
    const { id } = req.params;
    const updateData = req.body;
    const { tenantId: currentUserTenantId, role: currentUserRole } = req.user;

    const user = await User.findByPk(id);
    if (!user) {
      return ApiResponse.error(res, ErrorCodes.USER_NOT_FOUND, {}, 404);
    }

    // 检查权限
    if (currentUserRole !== 'SYSTEM_ADMIN' && user.tenantId !== currentUserTenantId) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, {}, 403);
    }

    // 防止修改超级管理员的角色
    if (user.role === 'SUPER_ADMIN' && updateData.role && updateData.role !== 'SUPER_ADMIN') {
      return ApiResponse.error(res, ErrorCodes.RESOURCE_DELETE_SUPER_ADMIN, {}, 400);
    }

    // 只允许系统管理员和用户管理员修改用户角色
    if (updateData.role && !['SYSTEM_ADMIN', 'USER_ADMIN'].includes(currentUserRole)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }

    await user.update(updateData);

    // 返回更新后的用户信息
    const userResponse = { ...user.toJSON() };
    delete userResponse.password;

    return ApiResponse.success(res, { user: userResponse }, 'update_success');

  } catch (error) {
    logger.error('Update user error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.USER_UPDATE_FAILED, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/users/{id}/password:
 *   put:
 *     summary: 修改用户密码
 *     tags: [用户管理]
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
 *               - newPassword
 *             properties:
 *               newPassword:
 *                 type: string
 *     responses:
 *       200:
 *         description: 修改成功
 */
router.put('/:id/password', authenticateToken, validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() }),
  body: Joi.object({ newPassword: Joi.string().min(6).required() })
})), async (req, res) => {
  try {
    const { id } = req.params;
    const { newPassword } = req.body;
    const { tenantId: currentUserTenantId, role: currentUserRole, id: currentUserId } = req.user;

    const user = await User.findByPk(id);
    if (!user) {
      return ApiResponse.error(res, ErrorCodes.USER_NOT_FOUND, {}, 404);
    }

    // 检查权限：用户可以修改自己的密码，系统管理员和用户管理员可以修改本租户用户的密码
    const canModifyPassword =
      id === currentUserId || // 修改自己的密码
      (['SYSTEM_ADMIN', 'USER_ADMIN'].includes(currentUserRole) && user.tenantId === currentUserTenantId); // 同租户管理员
    if (!canModifyPassword) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, {}, 403);
    }

    await user.update({ password: newPassword });

    return ApiResponse.success(res, null, 'password_update_success');

  } catch (error) {
    logger.error('Update password error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/users/{id}:
 *   delete:
 *     summary: 删除用户
 *     tags: [用户管理]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: 删除成功
 */
router.delete('/:id', authenticateToken, validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() })
})), async (req, res) => {
  try {
    const { id } = req.params;
    const { tenantId: currentUserTenantId, role: currentUserRole, id: currentUserId } = req.user;

    // 检查删除用户的权限
    // 只有系统管理员和用户管理员可以删除用户
    const canDeleteUsers = ['SYSTEM_ADMIN', 'USER_ADMIN'].includes(currentUserRole);
    if (!canDeleteUsers) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, {}, 403);
    }

    const user = await User.findByPk(id);
    if (!user) {
      return ApiResponse.error(res, ErrorCodes.USER_NOT_FOUND, {}, 404);
    }

    // 防止删除自己
    if (id === currentUserId) {
      return ApiResponse.error(res, ErrorCodes.RESOURCE_DELETE_SELF, {}, 400);
    }

    // 防止删除超级管理员
    if (user.role === 'SUPER_ADMIN') {
      return ApiResponse.error(res, ErrorCodes.RESOURCE_DELETE_SUPER_ADMIN, {}, 400);
    }

    // 检查权限
    if (currentUserRole !== 'SYSTEM_ADMIN' && user.tenantId !== currentUserTenantId) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_DENIED, {}, 403);
    }

    await user.destroy();

    return ApiResponse.success(res, null, 'delete_success');

  } catch (error) {
    logger.error('Delete user error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.USER_DELETE_FAILED, {}, 500);
  }
});

module.exports = router;
