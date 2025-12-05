const express = require('express');
const Joi = require('joi');
const { validate } = require('../../middlewares/validate');
const { logger } = require('../../utils/logger');
const { buildTenantWhere } = require('../../utils/scope');
const { Op } = require('sequelize');
const ApiResponse = require('../../utils/response');
const ErrorCodes = require('../../constants/errorCodes');
const appConfig = require('../../config/app');

const router = express.Router();

// 导入模型和中间件
const { Project, Tenant, User } = require('../../models');
const { authenticateToken, requireResourceOwnership } = require('../../middlewares/auth');

/**
 * @swagger
 * /api/v1/projects:
 *   get:
 *     summary: 获取工程列表
 *     tags: [工程管理]
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
 *         name: name
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
    name: Joi.string().optional()
  })
})), async (req, res) => {
  try {
    const { page, limit, name } = req.query;

    // 转换分页参数为数字
    const pageNum = parseInt(page, 10);
    const limitNum = parseInt(limit, 10);

    let where = buildTenantWhere({}, req);

    if (name) where.name = { [Op.like]: `%${name}%` };

    const offset = (pageNum - 1) * limitNum;

    const { count, rows } = await Project.findAndCountAll({
      where,
      include: [
        { model: Tenant, as: 'tenant' },
        { model: User, as: 'creator', attributes: ['id', 'username', 'fullName'] },
        { model: User, as: 'updater', attributes: ['id', 'username', 'fullName'] }
      ],
      limit: limitNum,
      offset,
      order: [['createdAt', 'DESC']],
    });

    return ApiResponse.paginated(res, { projects: rows }, {
      total: count,
      page: pageNum,
      limit: limitNum,
      totalPages: Math.ceil(count / limitNum),
    });
  } catch (error) {
    logger.error('Get projects error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/projects:
 *   post:
 *     summary: 创建工程
 *     tags: [工程管理]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - name
 *             properties:
 *               name:
 *                 type: string
 *               description:
 *                 type: string
 *               colorTag:
 *                 type: string
 *     responses:
 *       201:
 *         description: 创建成功
 */
router.post('/', authenticateToken, validate(Joi.object({
  body: Joi.object({
    name: Joi.string().required(),
    description: Joi.string().allow('').optional(),
    colorTag: Joi.string().valid('#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899', '#6b7280').optional()
  }).required()
})), async (req, res) => {
  try {
    const { name, description, colorTag } = req.body;
    const { tenantId, role, id: userId } = req.user;

    // 检查权限：只有系统管理员和工程管理员可以创建工程
    if (!['SYSTEM_ADMIN', 'PROJECT_ADMIN'].includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }

    // 创建工程
    const project = await Project.create({
      name,
      description,
      colorTag: colorTag || '#3b82f6',
      tenantId,
      createdBy: userId,
    });

    // 返回工程信息（附带对应的 app 基本信息）
    const projectWithRelations = await Project.findByPk(project.id, {
      include: [
        { model: Tenant, as: 'tenant' },
        { model: User, as: 'creator', attributes: ['id', 'username', 'fullName'] }
      ]
    });

    return ApiResponse.success(
      res,
      { project: projectWithRelations },
      'project_create_success',
      {},
      201
    );

  } catch (error) {
    logger.error('Create project error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.PROJECT_CREATE_FAILED, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/projects/{id}:
 *   put:
 *     summary: 更新工程
 *     tags: [工程管理]
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
 *               name:
 *                 type: string
 *               description:
 *                 type: string
 *               colorTag:
 *                 type: string
 *     responses:
 *       200:
 *         description: 更新成功
 */
router.put('/:id', authenticateToken, requireResourceOwnership('project'), validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() }),
  body: Joi.object({
    name: Joi.string().optional(),
    description: Joi.string().allow('').optional(),
    colorTag: Joi.string().valid('#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899', '#6b7280').optional(),
  }).min(1)
})), async (req, res) => {
  try {
    const { id } = req.params;
    const updateData = req.body;
    const { role, id: userId } = req.user;

    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 404);
    }

    // 检查权限：只有系统管理员和工程管理员可以更新工程
    const allowedRoles = ['SYSTEM_ADMIN', 'PROJECT_ADMIN'];
    if (!allowedRoles.includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }

    // 更新工程
    await project.update({
      ...updateData,
      updatedBy: userId
    });

    // 返回更新后的工程信息
    const updatedProject = await Project.findByPk(id, {
      include: [
        { model: Tenant, as: 'tenant' },
        { model: User, as: 'creator', attributes: ['id', 'username', 'fullName'] },
        { model: User, as: 'updater', attributes: ['id', 'username', 'fullName'] }
      ]
    });

    return ApiResponse.success(res, { project: updatedProject }, 'update_success');

  } catch (error) {
    logger.error('Update project error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.PROJECT_UPDATE_FAILED, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/projects/{id}:
 *   delete:
 *     summary: 删除工程
 *     tags: [工程管理]
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
router.delete('/:id', authenticateToken, requireResourceOwnership('project'), validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() })
})), async (req, res) => {
  try {
    const { id } = req.params;
    const { role } = req.user;

    // 检查权限：只有系统管理员和工程管理员可以删除工程
    if (!['SYSTEM_ADMIN', 'PROJECT_ADMIN'].includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }

    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 404);
    }

    await project.destroy();

    return ApiResponse.success(res, null, 'delete_success');

  } catch (error) {
    logger.error('Delete project error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.PROJECT_DELETE_FAILED, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/projects/{id}/operations/{operation}:
 *   post:
 *     summary: 执行工程运维操作
 *     tags: [工程运维]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *       - in: path
 *         name: operation
 *         required: true
 *         schema:
 *           type: string
 *         enum: [start, stop, restart, deploy, backup]
 *     responses:
 *       200:
 *         description: 操作成功
 */
router.post('/:id/operations/:operation', authenticateToken, requireResourceOwnership('project'), validate(Joi.object({
  params: Joi.object({
    id: Joi.string().uuid().required(),
    operation: Joi.string().valid('start','stop','restart','deploy','backup').required()
  })
})), async (req, res) => {
  try {
    const { id, operation } = req.params;
    const { role, id: userId } = req.user;

    // 检查权限：只有运维管理员可以执行运维操作
    if (!['SYSTEM_ADMIN', 'OPS_ADMIN'].includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }

    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 404);
    }

    // 执行运维操作（这里是模拟，实际项目中需要调用具体的运维接口）
    const operationResults = {
      start: 'Project started successfully',
      stop: 'Project stopped successfully',
      restart: 'Project restarted successfully',
      deploy: 'Project deployed successfully',
      backup: 'Project backup completed successfully'
    };

    // 记录操作日志
    const Log = require('../../models').Log;
    await Log.create({
      level: 'info',
      message: `Project ${operation} operation performed`,
      action: `project:${operation}`,
      resource: 'project',
      resourceId: id,
      userId: userId,
      tenantId: project.tenantId,
      metadata: { operation, projectId: id, projectName: project.name }
    });

    return ApiResponse.success(res, {
      message: operationResults[operation] || 'Operation completed',
      operation,
      projectId: id
    }, 'project_operation_success');

  } catch (error) {
    logger.error('Project operation error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, {}, 500);
  }
});

module.exports = router;
