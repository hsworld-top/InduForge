const express = require('express');
const multer = require('multer');
const Joi = require('joi');
const { validate } = require('../../middlewares/validate');
const { logger } = require('../../utils/logger');
const ApiResponse = require('../../utils/response');
const ErrorCodes = require('../../constants/errorCodes');
const AppError = require('../../utils/AppError');
const appConfig = require('../../config/app');

const router = express.Router();

// 导入模型和中间件
const { Tenant, User } = require('../../models');
const { authenticateToken, requireRole } = require('../../middlewares/auth');

// 配置文件上传

// 配置multer存储到内存
const storage = multer.memoryStorage();

// 文件过滤器
const fileFilter = (req, file, cb) => {
  // 只允许图片文件
  if (file.mimetype.startsWith('image/')) {
    cb(null, true);
  } else {
    cb(new Error('只允许上传图片文件'), false);
  }
};

// 配置multer上传
const upload = multer({
  storage: storage,
  fileFilter: fileFilter,
  limits: {
    fileSize: 5 * 1024 * 1024, // 5MB
  }
});

// 使用统一的认证和角色检查中间件

/**
 * @swagger
 * /api/v1/tenants:
 *   get:
 *     summary: 获取租户列表
 *     tags: [租户管理]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *         description: 页码
 *       - in: query
 *         name: limit
 *         schema:
 *           type: integer
 *         description: 每页数量
 *       - in: query
 *         name: status
 *         schema:
 *           type: string
 *         description: 状态筛选
 *     responses:
 *       200:
 *         description: 获取成功
 */
router.get('/', authenticateToken, requireRole('SUPER_ADMIN'), validate(Joi.object({
  query: Joi.object({
    page: Joi.number().integer().min(1).default(appConfig.pagination.defaultPage),
    limit: Joi.number().integer().min(1).max(appConfig.pagination.maxLimit).default(appConfig.pagination.defaultLimit),
    status: Joi.string().valid('active','inactive','suspended').optional()
  })
})), async (req, res) => {
  try {
    const { page, limit, status } = req.query;

    // 将字符串转换为数字
    const pageNum = parseInt(page, 10);
    const limitNum = parseInt(limit, 10);

    const where = {};
    if (status) {
      where.status = status;
    }

    const offset = (pageNum - 1) * limitNum;

    const { count, rows } = await Tenant.findAndCountAll({
      where,
      limit: limitNum,
      offset,
      order: [['createdAt', 'DESC']],
    });

    return ApiResponse.paginated(res, { tenants: rows }, {
        total: count,
        page: pageNum,
        limit: limitNum,
        totalPages: Math.ceil(count / limitNum),
    });
  } catch (error) {
    logger.error('Get tenants error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/tenants:
 *   post:
 *     summary: 创建租户
 *     tags: [租户管理]
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
 *               - code
 *             properties:
 *               name:
 *                 type: string
 *                 description: 租户名称
 *               code:
 *                 type: string
 *                 description: 租户代码
 *               description:
 *                 type: string
 *                 description: 描述
 *               contactEmail:
 *                 type: string
 *                 description: 联系邮箱
 *               contactPhone:
 *                 type: string
 *                 description: 联系电话
 *               maxUsers:
 *                 type: integer
 *                 description: 最大用户数
 *               maxProjects:
 *                 type: integer
 *                 description: 最大工程数
 *     responses:
 *       201:
 *         description: 创建成功
 */
router.post('/', authenticateToken, requireRole('SUPER_ADMIN'), validate(Joi.object({
  body: Joi.object({
    name: Joi.string().required(),
    code: Joi.string().min(2).max(50).required(),
    description: Joi.string().allow('').optional(),
    contactEmail: Joi.string().email().optional().allow(''),
    contactPhone: Joi.string().optional().allow(''),
    maxUsers: Joi.number().integer().min(1).optional(),
    maxProjects: Joi.number().integer().min(1).optional(),
    // 新增字段
    logoUrl: Joi.string().optional().allow(''),
    loginBackgroundUrl: Joi.string().optional().allow(''),
    companyName: Joi.string().optional().allow(''),
    companyAddress: Joi.string().optional().allow(''),
    companyPhone: Joi.string().optional().allow(''),
    companyWebsite: Joi.string().optional().allow('')
  }).required()
})), async (req, res) => {
  try {
    const {
      name, code, description, contactEmail, contactPhone, maxUsers, maxProjects,
      logoUrl, loginBackgroundUrl, companyName, companyAddress, companyPhone, companyWebsite
    } = req.body;

    // 检查租户代码是否已存在
    const existingTenant = await Tenant.findOne({ where: { code } });
    if (existingTenant) {
      return ApiResponse.error(res, ErrorCodes.TENANT_CODE_EXISTS, {}, 400);
    }

    // 创建租户
    const tenant = await Tenant.create({
      name,
      code,
      description: description || null,
      contactEmail: contactEmail || null,
      contactPhone: contactPhone || null,
      maxUsers: maxUsers || appConfig.tenantDefaults.defaultMaxUsers,
      maxProjects: maxProjects || appConfig.tenantDefaults.defaultMaxProjects,
      logoUrl: logoUrl || null,
      loginBackgroundUrl: loginBackgroundUrl || null,
      companyName: companyName || null,
      companyAddress: companyAddress || null,
      companyPhone: companyPhone || null,
      companyWebsite: companyWebsite || null,
    });

    // 创建默认的系统管理员用户
    await User.create({
      username: appConfig.tenantDefaults.defaultAdminUsername,
      password: appConfig.tenantDefaults.defaultAdminPassword,
      fullName: `${name}系统管理员`,
      role: 'SYSTEM_ADMIN',
      tenantId: tenant.id,
    });

    return ApiResponse.success(res, {
      tenant,
      message: `Tenant created successfully. Default admin user created with username: ${appConfig.tenantDefaults.defaultAdminUsername}, password: ${appConfig.tenantDefaults.defaultAdminPassword}`
    }, 'tenant_create_success', {}, 201);

  } catch (error) {
    logger.error('Create tenant error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.TENANT_CREATE_FAILED, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/tenants/{id}:
 *   put:
 *     summary: 更新租户
 *     tags: [租户管理]
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
 *               status:
 *                 type: string
 *                 enum: [active, inactive, suspended]
 *               contactEmail:
 *                 type: string
 *               contactPhone:
 *                 type: string
 *               maxUsers:
 *                 type: integer
 *               maxProjects:
 *                 type: integer
 *     responses:
 *       200:
 *         description: 更新成功
 */
router.put('/:id', authenticateToken, requireRole('SUPER_ADMIN'), validate(Joi.object({
  params: Joi.object({ id: Joi.string().required() }), // 支持UUID或租户代码
  body: Joi.object({
    name: Joi.string().optional(),
    code: Joi.string().min(2).max(50).optional(),
    description: Joi.string().allow('').optional(),
    status: Joi.string().valid('active','inactive','suspended').optional(),
    contactEmail: Joi.string().email().optional().allow(''),
    contactPhone: Joi.string().optional().allow(''),
    maxUsers: Joi.number().integer().min(1).optional(),
    maxProjects: Joi.number().integer().min(1).optional(),
    // 新增字段
    logoUrl: Joi.string().optional().allow(''),
    loginBackgroundUrl: Joi.string().optional().allow(''),
    companyName: Joi.string().optional().allow(''),
    companyAddress: Joi.string().optional().allow(''),
    companyPhone: Joi.string().optional().allow(''),
    companyWebsite: Joi.string().optional().allow('')
  }).min(1)
})), async (req, res) => {
  try {
    const { id } = req.params;
    const updateData = req.body;

    // 支持通过UUID或租户代码查找租户
    let tenant = await Tenant.findByPk(id);
    if (!tenant) {
      tenant = await Tenant.findOne({ where: { code: id } });
    }
    if (!tenant) {
      return ApiResponse.error(res, ErrorCodes.TENANT_NOT_FOUND, {}, 404);
    }

    // 如果更新代码，检查是否重复
    if (updateData.code && updateData.code !== tenant.code) {
      const existingTenant = await Tenant.findOne({ where: { code: updateData.code } });
      if (existingTenant) {
        return ApiResponse.error(res, ErrorCodes.TENANT_CODE_EXISTS, {}, 400);
      }
    }

    // 将空字符串转换为null
    const processedData = { ...updateData };
    const stringFields = [
      'description', 'contactEmail', 'contactPhone',
      'logoUrl', 'loginBackgroundUrl', 'companyName',
      'companyAddress', 'companyPhone', 'companyWebsite'
    ];

    stringFields.forEach(field => {
      if (processedData[field] === '') {
        processedData[field] = null;
      }
    });

    await tenant.update(processedData);

    return ApiResponse.success(res, { tenant }, 'update_success');

  } catch (error) {
    logger.error('Update tenant error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/tenants/{id}:
 *   delete:
 *     summary: 删除租户
 *     tags: [租户管理]
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
router.delete('/:id', authenticateToken, requireRole('SUPER_ADMIN'), validate(Joi.object({
  params: Joi.object({ id: Joi.string().required() }) // 支持UUID或租户代码
})), async (req, res) => {
  try {
    const { id } = req.params;

    // 支持通过UUID或租户代码查找租户
    let tenant = await Tenant.findByPk(id);
    if (!tenant) {
      tenant = await Tenant.findOne({ where: { code: id } });
    }
    if (!tenant) {
      return ApiResponse.error(res, ErrorCodes.TENANT_NOT_FOUND, {}, 404);
    }

    await tenant.destroy();

    return ApiResponse.success(res, null, 'delete_success');

  } catch (error) {
    logger.error('Delete tenant error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/tenants/{id}/upload:
 *   post:
 *     summary: 上传租户文件（Logo或背景图）
 *     tags: [租户管理]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *         description: 租户ID
 *       - in: query
 *         name: type
 *         required: true
 *         schema:
 *           type: string
 *           enum: [logo, background]
 *         description: 上传类型
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             type: object
 *             properties:
 *               file:
 *                 type: string
 *                 format: binary
 *                 description: 图片文件
 *     responses:
 *       200:
 *         description: 上传成功
 */
router.post('/:id/upload', authenticateToken, requireRole('SUPER_ADMIN'),
  upload.single('file'),
  validate(Joi.object({
    params: Joi.object({ id: Joi.string().required() }), // 支持UUID或租户代码
    query: Joi.object({
      type: Joi.string().valid('logo', 'background').required()
    })
  })),
  async (req, res) => {
    try {
      const { id } = req.params;
      const { type } = req.query;

      if (!req.file) {
        return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: '没有上传文件' }, 400);
      }

      // 检查租户是否存在（支持通过UUID或租户代码查找）
      let tenant;
      // 尝试通过UUID查找
      tenant = await Tenant.findByPk(id);
      // 如果没找到，尝试通过租户代码查找
      if (!tenant) {
        tenant = await Tenant.findOne({ where: { code: id } });
      }
      if (!tenant) {
        return ApiResponse.error(res, ErrorCodes.TENANT_NOT_FOUND, {}, 404);
      }

      // 将文件转换为base64格式
      const fileBuffer = req.file.buffer;
      const mimeType = req.file.mimetype;
      const base64Data = `data:${mimeType};base64,${fileBuffer.toString('base64')}`;

      // 更新租户的对应字段
      const updateData = {};
      if (type === 'logo') {
        updateData.logoUrl = base64Data;
      } else if (type === 'background') {
        updateData.loginBackgroundUrl = base64Data;
      }

      await tenant.update(updateData);

      return ApiResponse.success(res, {
        fileUrl: base64Data,
        message: `${type === 'logo' ? 'Logo' : '背景图'}上传成功`
      });

    } catch (error) {
      logger.error('Upload tenant file error', { error: error.message, requestId: req.requestId });

      // 如果是multer错误
      if (error instanceof multer.MulterError) {
        if (error.code === 'LIMIT_FILE_SIZE') {
          return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: '文件大小超过限制（最大5MB）' }, 400);
        }
      }

      // 如果是自定义错误（如文件类型不匹配）
      if (error.message === '只允许上传图片文件') {
        return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: error.message }, 400);
      }

      return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
    }
  }
);

module.exports = router;
