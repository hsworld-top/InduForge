const express = require('express');
const bcrypt = require('bcryptjs');
const Joi = require('joi');
const dayjs = require('dayjs');
const { validate } = require('../../middlewares/validate');
const { logger } = require('../../utils/logger');
const ApiResponse = require('../../utils/response');
const ErrorCodes = require('../../constants/errorCodes');
const AppError = require('../../utils/AppError');
const appConfig = require('../../config/app');
const TokenManager = require('../../utils/token');
const { buildAccessTokenPayload, listAccessibleProjectIds } = require('../../utils/authz');
const Captcha = require('../../utils/captcha');
const { TIME_FORMAT } = require('../../constants/time');

const router = express.Router();

// 导入模型
const { User, Tenant, Project, Log } = require('../../models');

/**
 * @swagger
 * /api/v1/auth/captcha:
 *   get:
 *     summary: 获取登录验证码（SVG）
 *     tags: [认证]
 *     responses:
 *       200:
 *         description: 获取成功
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 key:
 *                   type: string
 *                   description: 验证码标识，用于后续校验
 *                 image:
 *                   type: string
 *                   description: SVG 图片（data URL 格式）
 *                 expireSeconds:
 *                   type: integer
 *                   description: 过期时间（秒）
 */
router.get('/captcha', async (req, res) => {
  try {
    const { key, svg, expireSeconds } = await Captcha.generateCaptcha();
    // 以 data URL 返回，便于前端直接 <img src="...">
    const base64 = Buffer.from(svg).toString('base64');
    return ApiResponse.success(res, {
      key,
      image: `data:image/svg+xml;base64,${base64}`,
      expireSeconds
    });
  } catch (error) {
    logger.error('Generate captcha error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/auth/login:
 *   post:
 *     summary: 用户登录
 *     tags: [认证]
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - username
 *               - password
 *             properties:
 *               username:
 *                 type: string
 *                 description: 用户名
 *               password:
 *                 type: string
 *                 description: 密码
 *               tenantCode:
 *                 type: string
 *                 description: 租户代码（多租户环境可选）
 *               captchaKey:
 *                 type: string
 *                 description: 验证码Key（登录失败后需要，从获取验证码接口获取）
 *               captchaCode:
 *                 type: string
 *                 description: 验证码内容（登录失败后需要）
 *     responses:
 *       200:
 *         description: 登录成功
 *       401:
 *         description: 认证失败或验证码错误
 */
router.post('/login',
  validate(Joi.object({
    body: Joi.object({
      username: Joi.string().required(),
      password: Joi.string().required(),
      tenantCode: Joi.string().optional(),
      captchaKey: Joi.string().optional().allow(''),
      captchaCode: Joi.string().optional().allow('')
    }).required()
  })),
  async (req, res) => {
    try {
      const { username, password, tenantCode, captchaKey, captchaCode } = req.body;
      const loginIp = req.ip;
      const loginUserAgent = req.get('user-agent') || null;

      // 验证码校验（只有当提供了有效的验证码参数时才验证）
      if (captchaKey && captchaCode && captchaKey.trim() && captchaCode.trim()) {
        const isValidCaptcha = await Captcha.verifyCaptcha(captchaKey.trim(), captchaCode.trim());
        if (!isValidCaptcha) {
          return ApiResponse.error(res, ErrorCodes.AUTH_INVALID_CAPTCHA, {}, 200);
        }
      }

      // 检查是否为超级管理员登录（租户管理）
      if (username === appConfig.superAdmin.username && password === appConfig.superAdmin.password) {
        // 记录超级管理员登录日志（该账号非数据库用户，userId 置空）
        try {
          await Log.create({
            level: 'info',
            message: `认证登录成功，操作者：${appConfig.superAdmin.username}（SUPER_ADMIN）`,
            action: 'auth.login',
            resource: 'auth',
            userId: null,
            tenantId: null,
            ip: loginIp,
            userAgent: loginUserAgent,
            createdAt: new Date(),
          });
        } catch (logError) {
          logger.warn('Write super admin login log failed', { error: logError.message, requestId: req.requestId });
        }

        const payload = buildAccessTokenPayload({
          userId: appConfig.superAdmin.userId,
          username: appConfig.superAdmin.username,
          role: appConfig.superAdmin.role,
          tenantId: appConfig.defaultTenant.id,
          projectIds: ['*'],
        });

        // 生成双令牌
        const { accessToken, refreshToken } = await TokenManager.generateTokenPair(
          payload,
          appConfig.superAdmin.userId,
          appConfig.defaultTenant.id
        );

        // 设置 accessToken 到 cookie（用于 iframe 共享）
        res.cookie('accessToken', accessToken, {
          httpOnly: true, // 防止 XSS 攻击
          secure: process.env.NODE_ENV === 'production', // 生产环境使用 HTTPS
          sameSite: 'lax', // 允许同站和跨站请求携带 cookie
          maxAge: 24 * 60 * 60 * 1000 // 24 小时
        });

        return ApiResponse.success(res, {
          accessToken,
          refreshToken,
          user: {
            id: appConfig.superAdmin.userId,
            username: appConfig.superAdmin.username,
            email: appConfig.superAdmin.email || '',
            avatarUrl: appConfig.superAdmin.avatarUrl || '',
            role: appConfig.superAdmin.role,
            tenant: {
              id: appConfig.defaultTenant.id,
              name: appConfig.defaultTenant.name,
              code: appConfig.defaultTenant.code
            }
          }
        }, 'login_success');
      }

      // 普通用户登录
      let user;
      if (tenantCode) {
        // 多租户模式
        const tenant = await Tenant.findOne({ where: { code: tenantCode, status: 'active' } });
        if (!tenant) {
          return ApiResponse.error(res, ErrorCodes.AUTH_TENANT_CODE_INVALID, {}, 200);
        }

        user = await User.findOne({
          where: { username, tenantId: tenant.id, status: 'active' },
          include: [{ model: Tenant, as: 'tenant' }]
        });
      } else {
        // 单租户模式或自动检测
        const tenants = await Tenant.findAll({ where: { status: 'active' } });

        if (tenants.length === 1) {
          // 只有单个租户
          user = await User.findOne({
            where: { username, tenantId: tenants[0].id, status: 'active' },
            include: [{ model: Tenant, as: 'tenant' }]
          });
        } else {
          return ApiResponse.error(res, ErrorCodes.AUTH_TENANT_CODE_REQUIRED, {}, 200);
        }
      }

      if (!user || !(await bcrypt.compare(password, user.password))) {
        return ApiResponse.error(res, ErrorCodes.AUTH_INVALID_CREDENTIALS, {}, 200);
      }

      // 更新最后登录时间
      await user.update({ lastLoginAt: new Date() });

      // 记录登录日志（日志写入失败不影响登录成功）
      try {
        await Log.create({
          level: 'info',
          message: `认证登录成功，操作者：${user.username}（${user.role}）`,
          action: 'auth.login',
          resource: 'auth',
          userId: user.id,
          tenantId: user.tenantId,
          ip: loginIp,
          userAgent: loginUserAgent,
          createdAt: new Date(),
        });
      } catch (logError) {
        logger.warn('Write user login log failed', { error: logError.message, requestId: req.requestId });
      }

      const projectIds = await listAccessibleProjectIds(Project, {
        tenantId: user.tenantId,
        role: user.role,
      });
      const payload = buildAccessTokenPayload({
        userId: user.id,
        username: user.username,
        role: user.role,
        tenantId: user.tenantId,
        projectIds,
      });

      // 生成双令牌
      const { accessToken, refreshToken } = await TokenManager.generateTokenPair(
        payload,
        user.id,
        user.tenantId
      );

      // 设置 accessToken 到 cookie（用于 iframe 共享）
      res.cookie('accessToken', accessToken, {
        httpOnly: true, // 防止 XSS 攻击
        secure: process.env.NODE_ENV === 'production', // 生产环境使用 HTTPS
        sameSite: 'lax', // 允许同站和跨站请求携带 cookie
        maxAge: 24 * 60 * 60 * 1000 // 24 小时
      });

      return ApiResponse.success(res, {
        accessToken,
        refreshToken,
        user: {
          id: user.id,
          username: user.username,
          email: user.email || '',
          avatarUrl: user.avatarUrl || '',
          role: user.role,
          tenant: user.tenant
        }
      }, 'login_success');
    } catch (error) {
      logger.error('Login error', { error: error.message, requestId: req.requestId });
      return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
    }
  }
);

/**
 * @swagger
 * /api/v1/auth/me:
 *   get:
 *     summary: 获取当前用户信息
 *     tags: [认证]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: 获取成功
 */
/**
 * @swagger
 * /api/v1/auth/refresh:
 *   post:
 *     summary: 刷新 Access Token
 *     tags: [认证]
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - refreshToken
 *             properties:
 *               refreshToken:
 *                 type: string
 *                 description: Refresh Token
 *     responses:
 *       200:
 *         description: 刷新成功
 *       401:
 *         description: Refresh Token 无效或已过期
 */
router.post('/refresh',
  validate(Joi.object({
    body: Joi.object({
      refreshToken: Joi.string().required()
    }).required()
  })),
  async (req, res) => {
    try {
      const { refreshToken } = req.body;

      // 验证 Refresh Token
      const tokenData = await TokenManager.verifyRefreshToken(refreshToken);
      if (!tokenData) {
        return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_INVALID, {}, 200);
      }

      // 获取用户信息
      const user = await User.findByPk(tokenData.userId, {
        include: [{ model: Tenant, as: 'tenant' }]
      });

      if (!user || user.status !== 'active') {
        return ApiResponse.error(res, ErrorCodes.AUTH_USER_NOT_FOUND, {}, 200);
      }

      // 生成新的 Access Token
      const projectIds = await listAccessibleProjectIds(Project, {
        tenantId: user.tenantId,
        role: user.role,
      });
      const payload = buildAccessTokenPayload({
        userId: user.id,
        username: user.username,
        role: user.role,
        tenantId: user.tenantId,
        projectIds,
      });

      const result = await TokenManager.refreshAccessToken(refreshToken, payload);
      if (!result) {
        return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_INVALID, {}, 200);
      }

      return ApiResponse.success(res, {
        accessToken: result.accessToken,
        refreshToken: result.refreshToken
      }, 'token_refresh_success');
    } catch (error) {
      logger.error('Refresh token error', { error: error.message, requestId: req.requestId });
      return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
    }
  }
);

/**
 * @swagger
 * /api/v1/auth/logout:
 *   post:
 *     summary: 登出
 *     tags: [认证]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - refreshToken
 *             properties:
 *               refreshToken:
 *                 type: string
 *                 description: Refresh Token
 *     responses:
 *       200:
 *         description: 登出成功
 */
router.post('/logout',
  validate(Joi.object({
    body: Joi.object({
      refreshToken: Joi.string().required()
    }).required()
  })),
  async (req, res) => {
    try {
      const { refreshToken } = req.body;
      const accessToken = req.headers.authorization?.split(' ')[1];

      // 撤销 Refresh Token
      await TokenManager.revokeRefreshToken(refreshToken);

      // 将 Access Token 加入黑名单
      if (accessToken) {
        await TokenManager.blacklistAccessToken(accessToken);
      }

      return ApiResponse.success(res, null, 'logout_success');
    } catch (error) {
      logger.error('Logout error', { error: error.message, requestId: req.requestId });
      return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
    }
  }
);

/**
 * @swagger
 * /api/v1/auth/password:
 *   put:
 *     summary: 修改当前用户密码（需验证旧密码）
 *     tags: [认证]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - oldPassword
 *               - newPassword
 *             properties:
 *               oldPassword:
 *                 type: string
 *               newPassword:
 *                 type: string
 *                 minLength: 6
 *     responses:
 *       200:
 *         description: 修改成功
 *       401:
 *         description: 旧密码错误或 token 无效
 */
router.put('/password',
  validate(Joi.object({
    body: Joi.object({
      oldPassword: Joi.string().required(),
      newPassword: Joi.string().min(6).required()
    }).required()
  })),
  async (req, res) => {
    try {
      const token = req.headers.authorization?.split(' ')[1];
      if (!token) {
        return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_REQUIRED, {}, 200);
      }

      const isBlacklisted = await TokenManager.isAccessTokenBlacklisted(token);
      if (isBlacklisted) {
        return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_INVALID, {}, 200);
      }

      const decoded = TokenManager.verifyAccessToken(token);
      if (!decoded) {
        return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_INVALID, {}, 200);
      }

      const user = await User.findByPk(decoded.userId);
      if (!user) {
        return ApiResponse.error(res, ErrorCodes.AUTH_USER_NOT_FOUND, {}, 200);
      }

      const { oldPassword, newPassword } = req.body;
      const isOldPasswordValid = await bcrypt.compare(oldPassword, user.password);
      if (!isOldPasswordValid) {
        return ApiResponse.error(res, ErrorCodes.AUTH_INVALID_CREDENTIALS, {}, 200);
      }

      await user.update({ password: newPassword });
      return ApiResponse.success(res, null, 'password_update_success');
    } catch (error) {
      logger.error('Change password error', { error: error.message, requestId: req.requestId });
      return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
    }
  }
);

/**
 * @swagger
 * /api/auth/me:
 *   get:
 *     summary: 获取当前用户信息
 *     tags: [认证]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: 获取成功
 */
router.get('/me', async (req, res) => {
  try {
    const token = req.headers.authorization?.split(' ')[1];
    if (!token) {
      return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_REQUIRED, {}, 200);
    }

    // 检查是否在黑名单中
    const isBlacklisted = await TokenManager.isAccessTokenBlacklisted(token);
    if (isBlacklisted) {
      return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_INVALID, {}, 200);
    }

    // 验证 Access Token
    const decoded = TokenManager.verifyAccessToken(token);
    if (!decoded) {
      return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_INVALID, {}, 200);
    }

    const user = await User.findByPk(decoded.userId, {
      include: [{ model: Tenant, as: 'tenant' }]
    });

    if (!user) {
      return ApiResponse.error(res, ErrorCodes.AUTH_USER_NOT_FOUND, {}, 200);
    }

    return ApiResponse.success(res, {
      user: {
        id: user.id,
        username: user.username,
        email: user.email || '',
        avatarUrl: user.avatarUrl || '',
        role: user.role,
        tenant: user.tenant
      }
    });
  } catch (error) {
    logger.error('Get me error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/auth/config:
 *   get:
 *     summary: 获取应用配置
 *     tags: [认证]
 *     responses:
 *       200:
 *         description: 获取成功
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 title:
 *                   type: string
 *                   description: 应用标题
 *                 version:
 *                   type: string
 *                   description: 应用版本
 *                 description:
 *                   type: string
 *                   description: 应用描述
 *                 author:
 *                   type: string
 *                   description: 作者
 *                 buildTime:
 *                   type: string
 *                   description: 构建时间
 *                 multiTenant:
 *                   type: boolean
 *                   description: 是否启用多租户模式
 *                 activeTenantsCount:
 *                   type: integer
 *                   description: 活跃租户数量
 */
router.get('/config', validate(Joi.object({
  query: Joi.object({
    tenantCode: Joi.string().optional().allow(''),
  }),
})), async (req, res) => {
  try {
    const { tenantCode } = req.query;
    // 从package.json或其他配置文件读取应用信息
    const packageInfo = require('../../../package.json');

    // 获取活跃租户数量，决定是否启用多租户模式
    const { Tenant } = require('../../models');
    const activeTenantsCount = await Tenant.count({ where: { status: 'active' } });
    let tenantBranding = null;

    const tenantWhere = tenantCode && String(tenantCode).trim()
      ? { code: String(tenantCode).trim(), status: 'active' }
      : { id: appConfig.defaultTenant.id, status: 'active' };
    tenantBranding = await Tenant.findOne({
      where: tenantWhere,
      attributes: [
        'id',
        'name',
        'code',
        'logoUrl',
        'loginBackgroundUrl',
        'companyName',
        'companyPhone',
        'companyAddress',
        'companyWebsite',
        'settings',
      ],
    });

    const loginDisplayConfig = tenantBranding?.settings?.loginDisplay || {};
    const loginDisplay = {
      showCompanyName: Boolean(loginDisplayConfig.showCompanyName),
      showCompanyPhone: Boolean(loginDisplayConfig.showCompanyPhone),
      showCompanyAddress: Boolean(loginDisplayConfig.showCompanyAddress),
      showCompanyWebsite: Boolean(loginDisplayConfig.showCompanyWebsite),
      showIcp: Boolean(loginDisplayConfig.showIcp),
      icpNumber: loginDisplayConfig.icpNumber || '',
      companyName: tenantBranding?.companyName || '',
      companyPhone: tenantBranding?.companyPhone || '',
      companyAddress: tenantBranding?.companyAddress || '',
      companyWebsite: tenantBranding?.companyWebsite || '',
    };

    const config = {
      title: tenantBranding?.name || appConfig.defaultTenant.name,
      name: tenantBranding?.name || appConfig.defaultTenant.name,
      tenantName: tenantBranding?.name || appConfig.defaultTenant.name,
      tenantCode: tenantBranding?.code || appConfig.defaultTenant.code,
      appName: tenantBranding?.name || appConfig.defaultTenant.name,
      version: packageInfo.version || '1.0.0',
      description: packageInfo.description || '',
      author: packageInfo.author || '',
      buildTime: dayjs().format(TIME_FORMAT),
      // 如果活跃租户数量大于1，则启用多租户模式
      multiTenant: activeTenantsCount > 1,
      activeTenantsCount,
      logoUrl: tenantBranding?.logoUrl || null,
      loginBackgroundUrl: tenantBranding?.loginBackgroundUrl || null,
      loginDisplay,
    };

    return ApiResponse.success(res, config);
  } catch (error) {
    logger.error('Get config error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

module.exports = router;
