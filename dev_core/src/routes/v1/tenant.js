const express = require('express');
const multer = require('multer');
const Joi = require('joi');
const sharp = require('sharp');
const { randomUUID } = require('crypto');
const { Readable } = require('stream');
const { Op } = require('sequelize');
const { validate } = require('../../middlewares/validate');
const { logger } = require('../../utils/logger');
const ApiResponse = require('../../utils/response');
const ErrorCodes = require('../../constants/errorCodes');
const AppError = require('../../utils/AppError');
const appConfig = require('../../config/app');
const storageService = require('../../services/storageService');

const router = express.Router();

// 导入模型和中间件
const { Tenant, User } = require('../../models');
const { authenticateToken, requireRole } = require('../../middlewares/auth');

/**
 * 构建租户资产访问 URL。
 * @param {string} objectKey - 对象存储键
 * @returns {string} 对外访问 URL
 */
const buildTenantAssetUrl = (objectKey) =>
  `/api/v1/tenants/assets?key=${encodeURIComponent(objectKey)}`;

/**
 * 判断是否为 base64 图片 DataURL。
 * @param {string|null|undefined} value - 图片字段值
 * @returns {boolean} 是否为 DataURL
 */
const isImageDataUrl = (value) =>
  typeof value === 'string' && /^data:image\/[a-zA-Z0-9.+-]+;base64,/.test(value);

/**
 * 解析 base64 图片 DataURL。
 * @param {string} dataUrl - DataURL 字符串
 * @returns {{mimeType: string, buffer: Buffer}} 解析结果
 * @throws {Error} 非法格式抛错
 */
const parseImageDataUrl = (dataUrl) => {
  const matched = dataUrl.match(/^data:(image\/[a-zA-Z0-9.+-]+);base64,(.+)$/);
  if (!matched) {
    throw new Error('图片数据格式不正确');
  }
  return {
    mimeType: matched[1],
    buffer: Buffer.from(matched[2], 'base64'),
  };
};

/**
 * 上传租户品牌资产到对象存储。
 * Logo 生成缩略图并返回缩略图 URL；背景图进行压缩后返回 URL。
 * @param {Object} params - 上传参数
 * @param {string} params.tenantId - 租户 ID
 * @param {'logo'|'background'} params.type - 资产类型
 * @param {Buffer} params.buffer - 原始图片数据
 * @returns {Promise<{fileUrl: string}>} 上传结果
 */
const uploadTenantAsset = async ({ tenantId, type, buffer }) => {
  const assetPrefix = `tenant-assets/${tenantId}`;
  const objectId = randomUUID();

  if (type === 'logo') {
    const thumbnailBuffer = await sharp(buffer)
      .rotate()
      .resize(96, 96, { fit: 'cover' })
      .webp({ quality: 82 })
      .toBuffer();

    const objectKey = `${assetPrefix}/logo-thumb-${objectId}.webp`;
    await storageService.uploadObject(
      'ifp',
      objectKey,
      Readable.from(thumbnailBuffer),
      thumbnailBuffer.length,
      { 'Content-Type': 'image/webp' }
    );
    return { fileUrl: buildTenantAssetUrl(objectKey) };
  }

  const backgroundBuffer = await sharp(buffer)
    .rotate()
    .resize({
      width: 1920,
      height: 1080,
      fit: 'inside',
      withoutEnlargement: true,
    })
    .webp({ quality: 86 })
    .toBuffer();

  const objectKey = `${assetPrefix}/background-${objectId}.webp`;
  await storageService.uploadObject(
    'ifp',
    objectKey,
    Readable.from(backgroundBuffer),
    backgroundBuffer.length,
    { 'Content-Type': 'image/webp' }
  );
  return { fileUrl: buildTenantAssetUrl(objectKey) };
};

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
 * /api/v1/tenants/assets:
 *   get:
 *     summary: 获取租户品牌资产文件
 *     tags: [租户管理]
 *     parameters:
 *       - in: query
 *         name: key
 *         required: true
 *         schema:
 *           type: string
 *         description: 对象存储键
 *     responses:
 *       200:
 *         description: 获取成功
 */
router.get('/assets', validate(Joi.object({
  query: Joi.object({
    key: Joi.string().required(),
  }),
})), async (req, res) => {
  try {
    const { key } = req.query;
    if (!key.startsWith('tenant-assets/')) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: '非法资源路径' }, 400);
    }

    const [stat, stream] = await Promise.all([
      storageService.statObject('ifp', key),
      storageService.getObjectStream('ifp', key),
    ]);

    if (stat.metaData?.['content-type']) {
      res.setHeader('Content-Type', stat.metaData['content-type']);
    } else {
      res.setHeader('Content-Type', 'application/octet-stream');
    }
    res.setHeader('Cache-Control', 'public, max-age=86400');
    return stream.pipe(res);
  } catch (error) {
    logger.error('Get tenant asset error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.RESOURCE_NOT_FOUND, {}, 404);
  }
});

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
    status: Joi.string().valid('active','inactive','suspended').optional(),
    keyword: Joi.string().allow('').optional(),
  })
})), async (req, res) => {
  try {
    const { page, limit, status, keyword } = req.query;

    // 将字符串转换为数字
    const pageNum = parseInt(page, 10);
    const limitNum = parseInt(limit, 10);

    const where = {};
    if (status) {
      where.status = status;
    }
    const normalizedKeyword = typeof keyword === 'string' ? keyword.trim() : '';
    if (normalizedKeyword) {
      where[Op.or] = [
        { name: { [Op.like]: `%${normalizedKeyword}%` } },
        { code: { [Op.like]: `%${normalizedKeyword}%` } },
        { companyName: { [Op.like]: `%${normalizedKeyword}%` } },
        { contactEmail: { [Op.like]: `%${normalizedKeyword}%` } },
        { contactPhone: { [Op.like]: `%${normalizedKeyword}%` } },
      ];
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
    logoUrl: Joi.string().optional().allow('', null),
    loginBackgroundUrl: Joi.string().optional().allow('', null),
    companyName: Joi.string().optional().allow(''),
    companyAddress: Joi.string().optional().allow(''),
    companyPhone: Joi.string().optional().allow(''),
    companyWebsite: Joi.string().optional().allow(''),
    settings: Joi.object().optional().allow(null)
  }).required()
})), async (req, res) => {
  try {
    const {
      name, code, description, contactEmail, contactPhone, maxUsers, maxProjects,
      logoUrl, loginBackgroundUrl, companyName, companyAddress, companyPhone, companyWebsite, settings
    } = req.body;

    // 检查租户代码是否已存在
    const existingTenant = await Tenant.findOne({ where: { code } });
    if (existingTenant) {
      return ApiResponse.error(res, ErrorCodes.TENANT_CODE_EXISTS, {}, 400);
    }

    // 创建租户（品牌图先置空，后续若有 base64 则上传到对象存储并回写 URL）
    const tenant = await Tenant.create({
      name,
      code,
      description: description || null,
      contactEmail: contactEmail || null,
      contactPhone: contactPhone || null,
      maxUsers: maxUsers || appConfig.tenantDefaults.defaultMaxUsers,
      maxProjects: maxProjects || appConfig.tenantDefaults.defaultMaxProjects,
      logoUrl: null,
      loginBackgroundUrl: null,
      companyName: companyName || null,
      companyAddress: companyAddress || null,
      companyPhone: companyPhone || null,
      companyWebsite: companyWebsite || null,
      settings: settings || null,
    });

    const brandingUpdates = {};
    if (isImageDataUrl(logoUrl)) {
      const parsed = parseImageDataUrl(logoUrl);
      brandingUpdates.logoUrl = (await uploadTenantAsset({
        tenantId: tenant.id,
        type: 'logo',
        buffer: parsed.buffer,
      })).fileUrl;
    } else if (logoUrl && !String(logoUrl).includes('/assets/images/default-logo.svg')) {
      brandingUpdates.logoUrl = logoUrl;
    }

    if (isImageDataUrl(loginBackgroundUrl)) {
      const parsed = parseImageDataUrl(loginBackgroundUrl);
      brandingUpdates.loginBackgroundUrl = (await uploadTenantAsset({
        tenantId: tenant.id,
        type: 'background',
        buffer: parsed.buffer,
      })).fileUrl;
    } else if (
      loginBackgroundUrl &&
      !String(loginBackgroundUrl).includes('/assets/images/default-login-bg.svg')
    ) {
      brandingUpdates.loginBackgroundUrl = loginBackgroundUrl;
    }

    if (Object.keys(brandingUpdates).length > 0) {
      await tenant.update(brandingUpdates);
    }

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
    logoUrl: Joi.string().optional().allow('', null),
    loginBackgroundUrl: Joi.string().optional().allow('', null),
    companyName: Joi.string().optional().allow(''),
    companyAddress: Joi.string().optional().allow(''),
    companyPhone: Joi.string().optional().allow(''),
    companyWebsite: Joi.string().optional().allow(''),
    settings: Joi.object().optional().allow(null)
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

    if (isImageDataUrl(processedData.logoUrl)) {
      const parsed = parseImageDataUrl(processedData.logoUrl);
      processedData.logoUrl = (await uploadTenantAsset({
        tenantId: tenant.id,
        type: 'logo',
        buffer: parsed.buffer,
      })).fileUrl;
    }

    if (isImageDataUrl(processedData.loginBackgroundUrl)) {
      const parsed = parseImageDataUrl(processedData.loginBackgroundUrl);
      processedData.loginBackgroundUrl = (await uploadTenantAsset({
        tenantId: tenant.id,
        type: 'background',
        buffer: parsed.buffer,
      })).fileUrl;
    }

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

      const uploadResult = await uploadTenantAsset({
        tenantId: tenant.id,
        type,
        buffer: req.file.buffer,
      });

      // 更新租户的对应字段
      const updateData = {};
      if (type === 'logo') {
        updateData.logoUrl = uploadResult.fileUrl;
      } else if (type === 'background') {
        updateData.loginBackgroundUrl = uploadResult.fileUrl;
      }

      await tenant.update(updateData);

      return ApiResponse.success(res, {
        fileUrl: uploadResult.fileUrl,
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
