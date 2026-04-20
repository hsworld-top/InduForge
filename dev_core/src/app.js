const express = require('express');
const path = require('path');
const fs = require('fs');
const cors = require('cors');
const helmet = require('helmet');
const cookieParser = require('cookie-parser');
const dayjs = require('dayjs');
// 从项目根目录加载 .env 文件
require('dotenv').config({ path: path.resolve(__dirname, '../../.env') });
const { logger } = require('./utils/logger');
const { requestIdMiddleware } = require('./middlewares/requestId');
const { localeMiddleware } = require('./middlewares/locale');
const ApiResponse = require('./utils/response');
const ErrorCodes = require('./constants/errorCodes');
const { getErrorMessage } = require('./utils/i18n');
const { TIME_FORMAT } = require('./constants/time');

// 导入路由注册器（版本化）
const { registerVersionedRoutes } = require('./routes/register');

/**
 * 获取应用配置
 */
const getAppConfig = () => {
  const PORT = Number(process.env.PORT || 19601);
  const NODE_ENV = process.env.NODE_ENV || 'development';
  const ENABLE_SWAGGER = String(process.env.ENABLE_SWAGGER || 'true') === 'true';

  return { PORT, NODE_ENV, ENABLE_SWAGGER };
};

/**
 * 获取允许本地嵌入的节点前端 origin 列表。
 * 默认使用根 `.env` 中的 VITE_NODE_AGENT_FRONT_PORT，避免端口文档和后端 CSP 漂移。
 */
const getLocalNodeAgentFrontOrigins = () => {
  const nodeAgentFrontPort = Number(process.env.VITE_NODE_AGENT_FRONT_PORT || 18604);
  return [
    `https://127.0.0.1:${nodeAgentFrontPort}`,
    `http://127.0.0.1:${nodeAgentFrontPort}`,
    `https://localhost:${nodeAgentFrontPort}`,
    `http://localhost:${nodeAgentFrontPort}`,
  ];
};

/**
 * 配置并初始化 Swagger（仅在需要时加载模块）
 */
const setupSwagger = (app, port) => {
  // 仅在需要时动态加载 Swagger 模块，避免生产环境加载不必要的依赖
  const swaggerUi = require('swagger-ui-express');
  const swaggerJSDoc = require('swagger-jsdoc');

  const swaggerDefinition = {
    openapi: '3.0.0',
    info: {
      title: '租户管理系统 API',
      version: '1.0.0',
      description: '多租户管理系统后端API文档',
    },
    servers: [
      {
        url: `http://localhost:${port}`,
        description: '开发服务器',
      },
    ],
    components: {
      securitySchemes: {
        bearerAuth: {
          type: 'http',
          scheme: 'bearer',
          bearerFormat: 'JWT',
        },
      },
    },
    security: [
      {
        bearerAuth: [],
      },
    ],
  };

  const options = {
    swaggerDefinition,
    apis: ['./src/routes/**/*.js'],
  };

  const swaggerSpec = swaggerJSDoc(options);
  app.use('/api-docs', swaggerUi.serve, swaggerUi.setup(swaggerSpec));
};

/**
 * 解析 CORS 源配置
 */
const parseCorsOrigins = (origins) => {
  return String(origins)
    .split(',')
    .map(s => s.trim())
    .filter(Boolean);
};

function buildApp() {
  const app = express();
  const { PORT, NODE_ENV, ENABLE_SWAGGER } = getAppConfig();
  const localNodeAgentFrontOrigins = getLocalNodeAgentFrontOrigins();

  // ==================== 基础中间件 ====================
  app.set('trust proxy', 1);
  app.use(requestIdMiddleware);
  app.use(localeMiddleware);
  // 配置 helmet，允许 frame-src 以支持在 iframe 中加载 designer
  app.use(helmet({
    contentSecurityPolicy: {
      directives: {
        defaultSrc: ["'self'"],
        frameSrc: ["'self'", ...localNodeAgentFrontOrigins],
        scriptSrc: ["'self'", "'unsafe-eval'", "'unsafe-inline'"],
        styleSrc: ["'self'", "'unsafe-inline'"],
        imgSrc: ["'self'", "data:", "blob:"],
        connectSrc: ["*"],
      },
    },
  }));

  // CORS 配置
  if (NODE_ENV === 'development') {
    // 开发环境：完全开放，允许所有来源
    app.use(cors({
      origin: true, // 允许所有来源
      credentials: true,
      methods: ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS'],
      allowedHeaders: ['Content-Type', 'Authorization', 'X-Tenant-ID', 'X-Request-ID', 'Accept-Language']
    }));
  } else {
    // 生产环境：使用白名单配置
    const corsOrigins = parseCorsOrigins(process.env.CORS_ORIGINS);
    app.use(cors({
      origin: (origin, callback) => {
        // 允许无 Origin 的请求（如 Postman / curl）
        if (!origin) {
          return callback(null, true);
        }
        // 检查是否在白名单中
        if (corsOrigins.includes(origin)) {
          callback(null, true);
        } else {
          callback(new Error('Not allowed by CORS'));
        }
      },
      credentials: true,
      methods: ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS'],
      allowedHeaders: ['Content-Type', 'Authorization', 'X-Tenant-ID', 'X-Request-ID', 'Accept-Language']
    }));
  }

  // ==================== 请求解析中间件 ====================
  app.use(cookieParser()); // 解析 cookie
  app.use(express.json({ limit: '10mb' }));
  app.use(express.urlencoded({ extended: true, limit: '10mb' }));

  // ==================== Swagger 文档 ====================
  // 仅在非生产环境且启用 Swagger 时加载相关模块，优化生产环境性能
  if (ENABLE_SWAGGER && NODE_ENV !== 'production') {
    setupSwagger(app, PORT);
  }

  // ==================== API 路由 ====================
  registerVersionedRoutes(app);

  // ==================== 健康检查 ====================
  app.get('/health', async (req, res) => {
    const { checkConnection, getDbStatus, isDbDegraded } = require('./config/database');
    const { isConnected, getStatus: getRedisStatus, isDegraded: isRedisDegraded } = require('./utils/redis');

    const timestamp = dayjs().format(TIME_FORMAT);
    const dbConnected = await checkConnection();
    const redisConnected = await isConnected();
    const dbStatus = getDbStatus();
    const redisStatus = getRedisStatus();

    // 判断整体健康状态
    const isHealthy = dbConnected && redisConnected;
    const isDegraded = isDbDegraded() || isRedisDegraded();

    const healthStatus = {
      status: isHealthy ? 'healthy' : (isDegraded ? 'degraded' : 'unhealthy'),
      timestamp,
      services: {
        database: {
          connected: dbConnected,
          degraded: isDbDegraded(),
          status: dbStatus,
          pool: dbStatus.pool
        },
        redis: {
          connected: redisConnected,
          degraded: isRedisDegraded(),
          status: redisStatus
        }
      }
    };

    const statusCode = isHealthy ? 200 : (isDegraded ? 200 : 503);
    res.status(statusCode).json(healthStatus);
  });

  // 定义主前端（IDE）路由的静态资源路径（部署时的构建产物目录 ide_views）
  const viewsPath = path.join(__dirname, '..', 'ide_views');
  const viewsIndexPath = path.join(viewsPath, 'index.html');
  // 定义 designer 路由的静态资源路径（backend/designer_view）
  const designerViewsPath = path.join(__dirname, '..', 'designer_view');
  const designerIndexPath = path.join(designerViewsPath, 'index.html');

  if (fs.existsSync(viewsPath) && fs.existsSync(viewsIndexPath)) {
    // 提供静态资源文件（JS、CSS、图片等）
    app.use(express.static(viewsPath, {
      maxAge: NODE_ENV === 'production' ? '1y' : '0',
      etag: true,
      lastModified: true
    }));
  }

  // 提供 designer_view 静态资源（如果存在）
  if (fs.existsSync(designerViewsPath) && fs.existsSync(designerIndexPath)) {
    // 为 designer 路由添加 CORS 和 CSP 头，支持 iframe 嵌套
    app.use('/designer', (req, res, next) => {
      res.setHeader('Access-Control-Allow-Origin', '*');
      res.setHeader('Access-Control-Allow-Methods', 'GET,POST,PUT,DELETE,OPTIONS');
      res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
      res.setHeader('Content-Security-Policy', "frame-ancestors *; script-src 'self' 'unsafe-eval' 'unsafe-inline'; connect-src *");
      res.setHeader('X-Frame-Options', 'ALLOWALL');
      next();
    }, express.static(designerViewsPath, {
      maxAge: NODE_ENV === 'production' ? '1y' : '0', // 生产环境缓存一年，开发环境不缓存
      etag: true,
      lastModified: true
    }));

    // designer SPA 路由支持：/designer/** 返回 designer_view/index.html
    app.get('/designer/*splat', (req, res) => {
      // 设置允许 iframe 嵌套的响应头
      res.setHeader('Content-Security-Policy', "frame-ancestors *; script-src 'self' 'unsafe-eval' 'unsafe-inline'; connect-src *");
      res.setHeader('X-Frame-Options', 'ALLOWALL');
      res.sendFile(designerIndexPath);
    });
  }

  if (fs.existsSync(viewsPath) && fs.existsSync(viewsIndexPath)) {
    // SPA 路由支持：所有非 API 路由都返回 index.html
    app.use((req, res, next) => {
      // 跳过 API 路由、健康检查和 Swagger 文档
      if (req.path.startsWith('/api') || req.path === '/health' || req.path === '/api-docs') {
        return next();
      }
      // 跳过静态资源文件（已由 express.static 处理）
      if (req.path.startsWith('/assets/') || /\.(js|css|png|jpg|jpeg|gif|svg|ico|woff|woff2|ttf|eot)$/i.test(req.path)) {
        return next();
      }
      // 设置允许 iframe 的 CSP，支持在页面中嵌入 designer
      res.setHeader('Content-Security-Policy', `default-src 'self'; frame-src 'self' ${localNodeAgentFrontOrigins.join(' ')}; script-src 'self' 'unsafe-eval' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self'`);
      // 返回前端 index.html
      res.sendFile(viewsIndexPath);
    });
  }

  // ==================== 404 处理 ====================
  app.use((req, res) => {
    res.locals.language = req.language || 'zh-CN';
    res.locals.requestId = req.requestId;
    ApiResponse.error(res, ErrorCodes.REQUEST_NOT_FOUND, {}, 404);
  });

  // ==================== 错误处理中间件 ====================
  app.use((err, req, res, next) => {
    const language = req.language || 'zh-CN';
    res.locals.language = language;
    res.locals.requestId = req.requestId;

    // 如果是 AppError，使用统一的错误码和 i18n
    if (err.name === 'AppError') {
      logger.error('AppError', {
        requestId: req.requestId,
        errorCode: err.errorCode,
        statusCode: err.statusCode,
        options: err.options
      });
      return ApiResponse.error(res, err.errorCode, err.options, err.statusCode);
    }

    // Joi 验证错误
    if (err.isJoi) {
      logger.error('Validation error', { requestId: req.requestId, details: err.details });
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { details: err.details }, 400);
    }

    // 其他错误
    const statusCode = err.statusCode || 500;
    const errorCode = statusCode === 500 ? ErrorCodes.INTERNAL_SERVER_ERROR : ErrorCodes.C0001;

    logger.error('Unhandled error', {
      requestId: req.requestId,
      statusCode,
      message: err.message,
      stack: err.stack
    });

    // 设置响应头中的语言信息
    res.setHeader('Content-Language', language);

    const response = {
      success: false,
      errorCode,
      message: getErrorMessage(language, errorCode),
      requestId: req.requestId,
    };

    if (NODE_ENV !== 'production' && err.stack) {
      response.stack = err.stack;
    }

    res.status(statusCode).json(response);
  });

  return app;
}

module.exports = { buildApp };


