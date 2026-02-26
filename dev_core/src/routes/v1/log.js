const express = require('express');
const Joi = require('joi');
const { validate } = require('../../middlewares/validate');
const { logger } = require('../../utils/logger');
const { Op, fn, col } = require('sequelize');
const dayjs = require('dayjs');
const ApiResponse = require('../../utils/response');
const ErrorCodes = require('../../constants/errorCodes');
const AppError = require('../../utils/AppError');
const appConfig = require('../../config/app');

const router = express.Router();

// 导入模型和中间件
const { Log, User, Tenant } = require('../../models');
const { authenticateToken } = require('../../middlewares/auth');

/**
 * @swagger
 * /api/v1/logs:
 *   get:
 *     summary: 获取系统日志
 *     tags: [系统日志]
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
 *         name: level
 *         schema:
 *           type: string
 *         enum: [info, warning, error, debug]
 *       - in: query
 *         name: action
 *         schema:
 *           type: string
 *       - in: query
 *         name: startDate
 *         schema:
 *           type: string
 *         format: date-time
 *       - in: query
 *         name: endDate
 *         schema:
 *           type: string
 *         format: date-time
 *     responses:
 *       200:
 *         description: 获取成功
 */
router.get('/', authenticateToken, validate(Joi.object({
  query: Joi.object({
    page: Joi.number().integer().min(1).default(appConfig.pagination.defaultPage),
    limit: Joi.number().integer().min(1).max(appConfig.pagination.maxLimit).default(appConfig.pagination.defaultLimit),
    level: Joi.string().valid('info','warning','error','debug').optional(),
    action: Joi.string().optional(),
    startDate: Joi.date().optional(),
    endDate: Joi.date().optional(),
    resource: Joi.string().optional(),
    userId: Joi.string().uuid().optional()
  })
})), async (req, res) => {
  try {
    const {
      page,
      limit,
      level,
      action,
      startDate,
      endDate,
      resource,
      userId
    } = req.query;
    const { tenantId, role } = req.user;

    const where = {};

    // 超级管理员可以查看所有日志，其他用户只能查看本租户日志
    if (role !== 'SUPER_ADMIN') {
      where.tenantId = tenantId;
    }

    if (level) where.level = level;
    if (action) where.action = action;
    if (resource) where.resource = resource;
    if (userId) where.userId = userId;

    // 时间范围查询
    if (startDate || endDate) {
      where.createdAt = {};
      if (startDate) where.createdAt[Op.gte] = dayjs(startDate).toDate();
      if (endDate) where.createdAt[Op.lte] = dayjs(endDate).toDate();
    }

    const pageNum = parseInt(page, 10);
    const limitNum = parseInt(limit, 10);
    const offset = (pageNum - 1) * limitNum;

    const { count, rows } = await Log.findAndCountAll({
      where,
      attributes: { exclude: ['metadata'] },
      include: [
        { model: User, as: 'user', attributes: ['id', 'username', 'fullName'] },
        { model: Tenant, as: 'tenant', attributes: ['id', 'name'] }
      ],
      limit: limitNum,
      offset,
      order: [['createdAt', 'DESC']],
    });

    return ApiResponse.paginated(res, { logs: rows }, {
      total: count,
      page: pageNum,
      limit: limitNum,
      totalPages: Math.ceil(count / limitNum),
    });
  } catch (error) {
    logger.error('Get logs error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/logs/stats:
 *   get:
 *     summary: 获取日志统计信息
 *     tags: [系统日志]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: 获取成功
 */
router.get('/stats', authenticateToken, async (req, res) => {
  try {
    const { tenantId, role } = req.user;

    const where = {};
    if (role !== 'SUPER_ADMIN') {
      where.tenantId = tenantId;
    }

    // 统计不同级别的日志数量
    const levelStats = await Log.findAll({
      where,
      attributes: [
        'level',
        [fn('COUNT', col('level')), 'count']
      ],
      group: ['level'],
      raw: true
    });

    // 统计最近7天的日志趋势
    const sevenDaysAgo = dayjs().subtract(7, 'day').toDate();

    const trendStats = await Log.findAll({
      where: {
        ...where,
        createdAt: {
          [Op.gte]: sevenDaysAgo
        }
      },
      attributes: [
        [fn('DATE', col('createdAt')), 'date'],
        [fn('COUNT', '*'), 'count']
      ],
      group: [fn('DATE', col('createdAt'))],
      order: [[fn('DATE', col('createdAt')), 'ASC']],
      raw: true
    });

    // 统计最活跃的操作
    const actionStats = await Log.findAll({
      where,
      attributes: [
        'action',
        [fn('COUNT', '*'), 'count']
      ],
      group: ['action'],
      order: [[fn('COUNT', '*'), 'DESC']],
      limit: 10,
      raw: true
    });

    return ApiResponse.success(res, {
      levelStats: levelStats.reduce((acc, stat) => {
        acc[stat.level] = parseInt(stat.count);
        return acc;
      }, {}),
      trendStats,
      actionStats
    });

  } catch (error) {
    logger.error('Get log stats error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

module.exports = router;
