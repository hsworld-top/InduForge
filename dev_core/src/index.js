// 从项目根目录加载 .env 文件
const path = require('path')
require('dotenv').config({ path: path.resolve(__dirname, '../../.env') })
const { logger } = require('./utils/logger')
const { buildApp } = require('./app')
const { sequelize, testConnection, stopDbHealthCheck } = require('./config/database')
const { initRedis, close: closeRedis } = require('./utils/redis')
const { initStorage } = require('./services/storageService')
const socketService = require('./services/socketService')
const nodeService = require('./services/nodeService')

// 环境变量配置
const config = {
  NODE_ENV: process.env.NODE_ENV || 'development',
  PORT: Number(process.env.PORT || 18101),
  ENABLE_SWAGGER: String(process.env.ENABLE_SWAGGER || 'true') === 'true',
  CORS_ORIGINS:
    process.env.CORS_ORIGINS ||
    'http://localhost:18601,http://localhost:18602,http://localhost:18603,http://localhost:18604',
  DB_HOST: process.env.DB_HOST || '127.0.0.1',
  NODE_OFFLINE_CHECK_INTERVAL_MS: Number(process.env.NODE_OFFLINE_CHECK_INTERVAL_MS || 5000),
  NODE_OFFLINE_TIMEOUT_SECONDS: Number(process.env.NODE_OFFLINE_TIMEOUT_SECONDS || 30),
}

let nodeOfflineCheckTimer = null

// 全局未捕获异常处理 - 必须在应用启动前设置
process.on('uncaughtException', (error) => {
  logger.error('Uncaught Exception - Application will exit', {
    error: error.message,
    stack: error.stack,
    name: error.name,
  })

  // 记录错误后，优雅退出
  // 注意：uncaughtException 后应用处于不稳定状态，应该退出
  setTimeout(() => {
    process.exit(1)
  }, 1000)
})

// 全局未处理的 Promise 拒绝处理
process.on('unhandledRejection', (reason, promise) => {
  logger.error('Unhandled Rejection', {
    reason: reason instanceof Error ? reason.message : String(reason),
    stack: reason instanceof Error ? reason.stack : undefined,
    promise: promise,
  })

  // 对于未处理的 Promise 拒绝，可以选择：
  // 1. 记录日志并继续运行（推荐用于生产环境）
  // 2. 退出进程（更严格的方式）
  // 这里我们记录日志但不退出，因为可能是异步操作中的错误
})

/**
 * 打印启动横幅信息
 */
const printStartupBanner = () => {
  const innerWidth = 64
  const top = '┌' + '─'.repeat(innerWidth) + '┐'
  const bottom = '└' + '─'.repeat(innerWidth) + '┘'

  const line = (text) => {
    const content = `  ${text}`
    return '│' + content.padEnd(innerWidth, ' ') + '│'
  }

  const swaggerStatus =
    config.ENABLE_SWAGGER && config.NODE_ENV !== 'production' ? 'enabled at /api-docs' : 'disabled'

  const banner = [
    top,
    line(`Server started : http://localhost:${config.PORT}`),
    line(`Environment    : ${config.NODE_ENV}`),
    line(`PID/Node       : ${process.pid} / ${process.versions.node}`),
    line(
      `CORS Policy    : ${
        config.NODE_ENV === 'development'
          ? 'open (all origins)'
          : `restricted (${config.CORS_ORIGINS})`
      }`,
    ),
    line(`DB Host        : ${config.DB_HOST}`),
    line(`Swagger        : ${swaggerStatus}`),
    bottom,
  ].join('\n')

  logger.info('\n' + banner)
}

/**
 * 初始化服务依赖（数据库、Redis等）
 */
const initializeServices = async () => {
  // 1. 测试数据库连接
  logger.info('Testing database connection...')
  await testConnection()
  logger.info('Database connection established')

  // 注意：数据库表结构通过 scripts/bootstrap/init-core-database.js 同步。

  // 2. 初始化 Redis 连接（非阻塞，失败不影响启动）
  try {
    logger.info('Initializing Redis connection...')
    initRedis()
    logger.info('Redis connection initialized')
  } catch (error) {
    logger.warn('Redis connection failed', { error: error.message })
    logger.info('Continuing without Redis (some features may be unavailable)...')
  }

  // 3. 初始化对象存储（非阻塞，失败不影响启动）
  try {
    logger.info('Initializing object storage connection...')
    await initStorage()
    logger.info('Object storage connection initialized')
  } catch (error) {
    logger.warn('Object storage connection failed', { error: error.message })
    logger.info('Continuing without object storage (storage features may be unavailable)...')
  }
}

/**
 * 优雅关闭处理
 */
const setupGracefulShutdown = (server) => {
  const gracefulShutdown = async (signal) => {
    try {
      logger.info(`Received ${signal}, shutting down gracefully...`)

      // 停止数据库健康检查
      stopDbHealthCheck()

      // 停止节点离线检测任务
      if (nodeOfflineCheckTimer) {
        clearTimeout(nodeOfflineCheckTimer)
        nodeOfflineCheckTimer = null
      }

      // 关闭 Socket.IO 服务器
      socketService.close()
      logger.info('Socket.IO server closed')

      // 关闭数据库连接
      await sequelize.close()
      logger.info('Database connection closed')

      // 关闭 Redis 连接
      await closeRedis()
      logger.info('Redis connection closed')

      // 关闭 HTTP 服务器
      server.close(() => {
        logger.info('HTTP server closed')
        process.exit(0)
      })

      // 如果关闭超时，强制退出
      setTimeout(() => {
        logger.warn('Force exit after timeout')
        process.exit(1)
      }, 10000).unref()
    } catch (error) {
      logger.error('Error during graceful shutdown', { error: error.message })
      process.exit(1)
    }
  }

  // 监听关闭信号
  process.on('SIGINT', () => gracefulShutdown('SIGINT'))
  process.on('SIGTERM', () => gracefulShutdown('SIGTERM'))
}

/**
 * 启动节点离线检测任务
 * 定期将超过心跳超时时间的在线节点标记为离线。
 */
const startNodeOfflineCheck = () => {
  const intervalMs = Math.max(1000, config.NODE_OFFLINE_CHECK_INTERVAL_MS)
  const timeoutSeconds = Math.max(5, config.NODE_OFFLINE_TIMEOUT_SECONDS)
  const timeoutMinutes = timeoutSeconds / 60
  const runCheck = async () => {
    let nextDelay = intervalMs
    try {
      const affectedCount = await nodeService.markOfflineNodes(timeoutMinutes)
      if (affectedCount > 0) {
        logger.info('Node offline check updated stale nodes', {
          affectedCount,
          timeoutSeconds,
        })
      }

      const commandResult = await nodeService.processCommandTimeouts()
      if (commandResult.retried > 0 || commandResult.deadLetter > 0) {
        logger.warn('Node command timeout processed', {
          retried: commandResult.retried,
          deadLetter: commandResult.deadLetter,
        })
      }

      nextDelay = await nodeService.getNextOfflineCheckDelayMs(timeoutSeconds, intervalMs)
    } catch (error) {
      logger.error('Node offline check failed', { error: error.message })
      nextDelay = intervalMs
    }

    nodeOfflineCheckTimer = setTimeout(runCheck, nextDelay)
  }

  runCheck()

  logger.info('Node offline check started', {
    baseIntervalMs: intervalMs,
    timeoutSeconds,
    mode: 'adaptive',
  })
}

/**
 * 启动服务器
 */
const startServer = async () => {
  try {
    // 1. 初始化服务依赖
    await initializeServices()

    // 2. 构建应用
    const app = buildApp()

    // 3. 启动 HTTP 服务器
    const server = app.listen(config.PORT, () => {
      printStartupBanner()
    })

    // 4. 初始化 Socket.IO 服务器
    socketService.initialize(server)
    logger.info('Socket.IO server initialized on the same port as HTTP server')

    // 5. 启动节点离线检测
    startNodeOfflineCheck()

    // 6. 设置优雅关闭
    setupGracefulShutdown(server)
  } catch (error) {
    logger.error('Failed to start server', {
      error: error.message,
      stack: error.stack,
    })
    process.exit(1)
  }
}

// 启动服务器
startServer()
