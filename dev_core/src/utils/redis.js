const Redis = require('ioredis');
const { logger } = require('./logger');

/**
 * Redis 客户端实例
 */
let redisClient = null;

/**
 * Redis 连接状态
 */
let redisStatus = {
  connected: false,
  degraded: false,      // 降级状态（超过最大重试次数）
  lastError: null,
  lastErrorTime: null,
  retryCount: 0,
  maxRetries: Number(process.env.REDIS_MAX_RETRIES || 20)
};

/**
 * 初始化 Redis 连接
 * @returns {Redis} Redis 客户端实例
 */
function initRedis() {
  if (redisClient) {
    return redisClient;
  }

  const redisConfig = {
    host: process.env.REDIS_HOST || '127.0.0.1',
    port: Number(process.env.REDIS_PORT || 6379),
    password: process.env.REDIS_PASSWORD || undefined,
    db: Number(process.env.REDIS_DB || 0),
    // 重连策略：指数退避，最多重试指定次数
    retryStrategy: (times) => {
      redisStatus.retryCount = times;
      
      if (times > redisStatus.maxRetries) {
        // 超过最大重试次数，进入降级模式
        if (!redisStatus.degraded) {
          redisStatus.degraded = true;
          redisStatus.lastErrorTime = new Date().toISOString();
          logger.error('Redis max retry attempts reached, entering degraded mode', {
            maxRetries: redisStatus.maxRetries,
            retryCount: times,
            degraded: true
          });
          logger.warn('⚠️  Redis is in degraded mode. Service will continue but Redis-dependent features are disabled.');
          logger.info('Redis will continue to retry periodically. Service will recover automatically when Redis is available.');
        }
        
        // 降级模式：每 30 秒重试一次（而不是完全停止）
        const degradedRetryInterval = Number(process.env.REDIS_DEGRADED_RETRY_INTERVAL || 30000);
        logger.debug(`Redis degraded mode: retrying in ${degradedRetryInterval}ms`);
        return degradedRetryInterval;
      }
      
      // 正常重连：指数退避
      // 延迟：50ms, 100ms, 200ms, 400ms, 800ms, 1600ms, 2000ms (max)
      const delay = Math.min(times * 50, 2000);
      logger.info(`Redis retry attempt ${times}/${redisStatus.maxRetries}, waiting ${delay}ms`);
      return delay;
    },
    // 每个请求最多重试 3 次
    maxRetriesPerRequest: 3,
    // 启用就绪检查
    enableReadyCheck: true,
    // 禁用离线队列（连接断开时立即失败，而不是排队）
    enableOfflineQueue: false,
    // 连接超时（毫秒）
    connectTimeout: 10000,
    // 命令超时（毫秒）
    commandTimeout: 5000,
    // 自动重连
    lazyConnect: false,
  };

  redisClient = new Redis(redisConfig);

  redisClient.on('connect', () => {
    logger.info('Redis connected');
    redisStatus.connected = true;
    redisStatus.degraded = false;
    redisStatus.retryCount = 0;
    redisStatus.lastError = null;
  });

  redisClient.on('ready', () => {
    logger.info('Redis ready');
    redisStatus.connected = true;
    redisStatus.degraded = false;
    redisStatus.retryCount = 0;
    redisStatus.lastError = null;
  });

  redisClient.on('error', (err) => {
    logger.error('Redis error', { 
      error: err.message,
      code: err.code,
      errno: err.errno
    });
    redisStatus.connected = false;
    redisStatus.lastError = err.message;
    redisStatus.lastErrorTime = new Date().toISOString();
  });

  redisClient.on('close', () => {
    logger.warn('Redis connection closed');
    redisStatus.connected = false;
  });

  redisClient.on('reconnecting', (delay) => {
    logger.info(`Redis reconnecting in ${delay}ms`);
  });

  redisClient.on('end', () => {
    logger.warn('Redis connection ended');
  });

  // 连接断开后，ioredis 会自动重连（根据 retryStrategy）
  redisClient.on('+node', (node) => {
    logger.info('Redis node added', { address: node.address });
  });

  redisClient.on('-node', (node) => {
    logger.warn('Redis node removed', { address: node.address });
  });

  return redisClient;
}

/**
 * 获取 Redis 客户端实例
 * @returns {Redis} Redis 客户端实例
 */
function getRedis() {
  if (!redisClient) {
    return initRedis();
  }
  return redisClient;
}

/**
 * 检查 Redis 连接状态
 * @returns {Promise<boolean>} 是否连接
 */
async function isConnected() {
  try {
    if (!redisClient) {
      return false;
    }
    const status = redisClient.status;
    if (status === 'ready' || status === 'connect') {
      // 执行一个简单的命令来验证连接
      await redisClient.ping();
      return true;
    }
    return false;
  } catch (error) {
    logger.error('Redis connection check failed', { error: error.message });
    return false;
  }
}

/**
 * 获取 Redis 连接状态信息
 * @returns {object} 连接状态
 */
function getStatus() {
  return {
    ...redisStatus,
    clientStatus: redisClient ? redisClient.status : 'not_initialized'
  };
}

/**
 * 检查是否处于降级模式
 * @returns {boolean} 是否降级
 */
function isDegraded() {
  return redisStatus.degraded;
}

/**
 * 设置键值对
 * @param {string} key - 键
 * @param {string|number|object} value - 值
 * @param {number} expireSeconds - 过期时间（秒），可选
 * @returns {Promise<boolean>} 是否成功
 */
async function set(key, value, expireSeconds = null) {
  try {
    const redis = getRedis();
    const stringValue = typeof value === 'object' ? JSON.stringify(value) : String(value);
    
    if (expireSeconds) {
      await redis.setex(key, expireSeconds, stringValue);
    } else {
      await redis.set(key, stringValue);
    }
    
    return true;
  } catch (error) {
    logger.error('Redis set error', { key, error: error.message });
    return false;
  }
}

/**
 * 获取值
 * @param {string} key - 键
 * @returns {Promise<string|null>} 值
 */
async function get(key) {
  try {
    const redis = getRedis();
    const value = await redis.get(key);
    return value;
  } catch (error) {
    logger.error('Redis get error', { key, error: error.message });
    return null;
  }
}

/**
 * 获取 JSON 对象
 * @param {string} key - 键
 * @returns {Promise<object|null>} JSON 对象
 */
async function getJSON(key) {
  try {
    const value = await get(key);
    if (!value) return null;
    return JSON.parse(value);
  } catch (error) {
    logger.error('Redis getJSON error', { key, error: error.message });
    return null;
  }
}

/**
 * 删除键
 * @param {string} key - 键
 * @returns {Promise<boolean>} 是否成功
 */
async function del(key) {
  try {
    const redis = getRedis();
    await redis.del(key);
    return true;
  } catch (error) {
    logger.error('Redis del error', { key, error: error.message });
    return false;
  }
}

/**
 * 批量删除键（支持模式匹配）
 * @param {string} pattern - 键模式
 * @returns {Promise<number>} 删除的数量
 */
async function delPattern(pattern) {
  try {
    const redis = getRedis();
    const keys = await redis.keys(pattern);
    if (keys.length === 0) return 0;
    await redis.del(...keys);
    return keys.length;
  } catch (error) {
    logger.error('Redis delPattern error', { pattern, error: error.message });
    return 0;
  }
}

/**
 * 检查键是否存在
 * @param {string} key - 键
 * @returns {Promise<boolean>} 是否存在
 */
async function exists(key) {
  try {
    const redis = getRedis();
    const result = await redis.exists(key);
    return result === 1;
  } catch (error) {
    logger.error('Redis exists error', { key, error: error.message });
    return false;
  }
}

/**
 * 设置过期时间
 * @param {string} key - 键
 * @param {number} seconds - 过期时间（秒）
 * @returns {Promise<boolean>} 是否成功
 */
async function expire(key, seconds) {
  try {
    const redis = getRedis();
    await redis.expire(key, seconds);
    return true;
  } catch (error) {
    logger.error('Redis expire error', { key, seconds, error: error.message });
    return false;
  }
}

/**
 * 获取剩余过期时间
 * @param {string} key - 键
 * @returns {Promise<number>} 剩余时间（秒），-1 表示永不过期，-2 表示键不存在
 */
async function ttl(key) {
  try {
    const redis = getRedis();
    return await redis.ttl(key);
  } catch (error) {
    logger.error('Redis ttl error', { key, error: error.message });
    return -2;
  }
}

/**
 * 关闭 Redis 连接
 */
async function close() {
  if (redisClient) {
    await redisClient.quit();
    redisClient = null;
    logger.info('Redis connection closed');
  }
}

module.exports = {
  initRedis,
  getRedis,
  isConnected,
  getStatus,
  isDegraded,
  set,
  get,
  getJSON,
  del,
  delPattern,
  exists,
  expire,
  ttl,
  close,
};

