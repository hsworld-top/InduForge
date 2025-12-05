const { Sequelize } = require('sequelize');
const mysql = require('mysql2/promise');
require('dotenv').config();
const { logger } = require('../utils/logger');

const sequelize = new Sequelize(
  process.env.DB_NAME || 'tenant_management',
  process.env.DB_USER || 'root',
  process.env.DB_PASSWORD || '',
  {
    host: process.env.DB_HOST || '127.0.0.1',
    port: Number(process.env.DB_PORT || 3306),
    dialect: 'mysql',
    logging: process.env.NODE_ENV === 'development' ? (msg) => logger.debug(msg) : false,
    // 添加查询超时
    queryTimeout: Number(process.env.DB_QUERY_TIMEOUT || 30000),  // 30 秒查询超时
    // 连接池配置（连接池会自动处理重连）
    pool: {
      max: Number(process.env.DB_MAX_CONNECTIONS || 50),                    // 最大连接数
      min: 5,                    // 最小连接数
      acquire: 10000,            // 获取连接超时时间（毫秒）
      idle: 30000,               // 连接空闲时间（毫秒），超过此时间未使用的连接会被释放
      evict: 5000,               // 检查空闲连接的间隔（毫秒）
      handleDisconnects: true,    // 自动处理断开连接
      // 添加连接验证（高并发时确保连接有效）
      validate: (connection) => {
        return connection && !connection._invalid;
      }
    },
    // 连接选项
    dialectOptions: {
      connectTimeout: Number(process.env.DB_CONNECT_TIMEOUT || 10000), // 连接超时（毫秒）
      // MySQL 特定选项
      supportBigNumbers: true,// 支持大数字
      bigNumberStrings: true, // 大数字以字符串返回
      // 添加连接选项
      multipleStatements: false,  // 禁用多语句执行（防止 SQL 注入）
      // 字符集和时区配置（高并发时重要）
      charset: 'utf8mb4',           // 使用 utf8mb4 支持完整的 Unicode（包括 emoji）
    },
    // 连接后执行（优化：减少日志开销）
    hooks: {
      afterConnect: (connection) => {
        // 仅在开发环境记录日志，生产环境减少日志开销
        if (process.env.NODE_ENV === 'development') {
          logger.debug('Database connection established', {
            threadId: connection.threadId
          });
        }
      },
      afterDisconnect: (connection) => {
        // 仅在开发环境记录日志
        if (process.env.NODE_ENV === 'development') {
          logger.warn('Database connection closed', {
            threadId: connection?.threadId
          });
        }
      }
    }

  }
);


/**
 * 数据库连接状态
 */
let dbStatus = {
  connected: false,
  degraded: false,      // 降级状态（连接失败但服务继续运行）
  lastError: null,
  lastErrorTime: null,
  retryCount: 0,
  maxRetries: Number(process.env.DB_MAX_RETRIES || 10)
};

/**
 * 定期检查数据库连接（用于降级模式下的重连）
 */
let dbHealthCheckInterval = null;

const startDbHealthCheck = () => {
  if (dbHealthCheckInterval) {
    return;
  }

  const checkInterval = Number(process.env.DB_DEGRADED_RETRY_INTERVAL || 30000); // 30秒

  dbHealthCheckInterval = setInterval(async () => {
    if (dbStatus.degraded && !dbStatus.connected) {
      try {
        await sequelize.authenticate();
        dbStatus.connected = true;
        dbStatus.degraded = false;
        dbStatus.retryCount = 0;
        dbStatus.lastError = null;
        logger.info('Database connection recovered from degraded mode');
      } catch (error) {
        dbStatus.retryCount++;
        dbStatus.lastError = error.message;
        dbStatus.lastErrorTime = new Date().toISOString();
        logger.debug(`Database health check failed (attempt ${dbStatus.retryCount}): ${error.message}`);
      }
    }
  }, checkInterval);

  logger.info('Database health check started', { interval: checkInterval });
};

const stopDbHealthCheck = () => {
  if (dbHealthCheckInterval) {
    clearInterval(dbHealthCheckInterval);
    dbHealthCheckInterval = null;
    logger.info('Database health check stopped');
  }
};

/**
 * 测试 MySQL 服务器连接（不指定数据库）
 * @returns {Promise<{success: boolean, error?: Error}>}
 */
const testMySQLServerConnection = async () => {
  const connectionConfig = {
    host: process.env.DB_HOST || '127.0.0.1',
    port: Number(process.env.DB_PORT || 3306),
    user: process.env.DB_USER || 'root',
    password: process.env.DB_PASSWORD || '',
    connectTimeout: 10000,
  };

  try {
    const connection = await mysql.createConnection(connectionConfig);
    await connection.end();
    return { success: true };
  } catch (error) {
    return { success: false, error };
  }
};

/**
 * 检查数据库是否存在
 */
const checkDatabaseExists = async () => {
  const dbName = process.env.DB_NAME || 'tenant_management';
  const connectionConfig = {
    host: process.env.DB_HOST || '127.0.0.1',
    port: Number(process.env.DB_PORT || 3306),
    user: process.env.DB_USER || 'root',
    password: process.env.DB_PASSWORD || '',
    connectTimeout: 10000,
  };

  try {
    const connection = await mysql.createConnection(connectionConfig);
    const [rows] = await connection.query(
      `SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?`,
      [dbName]
    );
    await connection.end();
    return rows.length > 0;
  } catch (error) {
    logger.error('检查数据库是否存在时出错', { error: error.message });
    return false;
  }
};

/**
 * 自动创建并初始化数据库
 */
const autoInitializeDatabase = async () => {
  try {
    logger.info('🔄 检测到数据库不存在，开始自动创建并初始化...');
    
    // 动态加载初始化脚本（避免循环依赖）
    // 路径：从 src/config/ 向上两级到 backend/，然后进入 scripts/
    const { executeSqlFile } = require('../../scripts/init-database');
    
    // 执行数据库初始化
    await executeSqlFile();
    
    logger.info('✅ 数据库自动创建并初始化完成');
    return true;
  } catch (error) {
    logger.error('❌ 自动初始化数据库失败', {
      error: error.message,
      stack: error.stack
    });
    
    console.log('\n🔧 数据库初始化指南:');
    console.log('='.repeat(50));
    console.log('1. 确保 MySQL 服务正在运行');
    console.log('2. 手动执行数据库初始化脚本:');
    console.log(`   cd ${process.cwd()}`);
    console.log('   node scripts/init-database.js init');
    console.log('='.repeat(50));
    console.log('');
    
    return false;
  }
};

// 测试数据库连接
const testConnection = async () => {
  try {
    // 1. 先测试 MySQL 服务器连接（不指定数据库）
    logger.info('🔍 测试 MySQL 服务器连接...');
    const serverTestResult = await testMySQLServerConnection();
    
    if (!serverTestResult.success) {
      const error = serverTestResult.error;
      logger.error('❌ MySQL 服务器连接失败，无法启动应用', {
        host: process.env.DB_HOST || '127.0.0.1',
        port: process.env.DB_PORT || 3306,
        user: process.env.DB_USER || 'root',
        error: error.message,
        code: error.code,
        errno: error.errno
      });
      
      console.log('\n🔧 MySQL 服务器连接问题排查:');
      console.log('='.repeat(50));
      console.log('1. 检查 MySQL 服务是否正在运行');
      console.log('   - Linux: sudo service mysql status');
      console.log('   - Mac: brew services list | grep mysql');
      console.log('   - Windows: 检查服务管理器');
      console.log('');
      console.log('2. 检查连接配置:');
      console.log(`   - DB_HOST: ${process.env.DB_HOST || '127.0.0.1'}`);
      console.log(`   - DB_PORT: ${process.env.DB_PORT || 3306}`);
      console.log(`   - DB_USER: ${process.env.DB_USER || 'root'}`);
      console.log(`   - DB_PASSWORD: ${process.env.DB_PASSWORD ? '*** (已设置)' : '⚠️ 未设置'}`);
      console.log('');
      console.log('3. 测试网络连接:');
      console.log(`   - ping ${process.env.DB_HOST || '127.0.0.1'}`);
      console.log(`   - telnet ${process.env.DB_HOST || '127.0.0.1'} ${process.env.DB_PORT || 3306}`);
      console.log('='.repeat(50));
      console.log('');
      
      logger.warn('应用将退出，请先解决 MySQL 服务器连接问题');
      process.exit(1);
    }
    
    logger.info('✅ MySQL 服务器连接成功');
    
    // 2. 检查数据库是否存在
    const dbName = process.env.DB_NAME || 'tenant_management';
    logger.info(`🔍 检查数据库 '${dbName}' 是否存在...`);
    const dbExists = await checkDatabaseExists();
    
    if (!dbExists) {
      logger.warn(`⚠️  数据库 '${dbName}' 不存在，将自动创建并初始化`);
      
      // 3. 自动创建并初始化数据库
      const initSuccess = await autoInitializeDatabase();
      
      if (!initSuccess) {
        logger.error('❌ 数据库自动初始化失败，应用将退出');
        process.exit(1);
      }
      
      // 等待一下，确保数据库创建完成
      await new Promise(resolve => setTimeout(resolve, 1000));
    } else {
      logger.info(`✅ 数据库 '${dbName}' 已存在`);
    }
    
    // 4. 测试 Sequelize 连接（使用指定数据库）
    logger.info('🔍 测试数据库连接...');
    await sequelize.authenticate();
    logger.info('✅ 数据库连接成功');
    
    dbStatus.connected = true;
    dbStatus.degraded = false;
    dbStatus.retryCount = 0;
    dbStatus.lastError = null;

    // 启动健康检查（用于降级模式下的重连）
    startDbHealthCheck();
  } catch (error) {
    // 检测数据库不存在的情况（双重检查，防止自动初始化失败）
    if (error.message.includes('Unknown database') ||
      error.message.includes('ER_BAD_DB_ERROR') ||
      error.sqlState === '42000' && error.errno === 1049) {
      logger.error('❌ 数据库不存在，自动初始化可能失败', {
        database: process.env.DB_NAME || 'tenant_management',
        error: error.message
      });

      console.log('\n🔧 数据库初始化指南:');
      console.log('='.repeat(50));
      console.log('1. 确保 MySQL 服务正在运行');
      console.log('2. 执行数据库初始化脚本:');
      console.log(`   cd ${process.cwd()}`);
      console.log('   node scripts/init-database.js init');
      console.log('='.repeat(50));
      console.log('');

      logger.warn('应用将退出，请先初始化数据库后再启动');
      process.exit(1);
    }
    dbStatus.connected = false;
    dbStatus.lastError = error.message;
    dbStatus.lastErrorTime = new Date().toISOString();
    logger.error('Unable to connect to the database', {
      error: error.message,
      code: error.code,
      errno: error.errno,
      sqlState: error.sqlState,
      host: process.env.DB_HOST || '127.0.0.1',
      port: process.env.DB_PORT || 3306,
      database: process.env.DB_NAME || 'tenant_management',
      user: process.env.DB_USER || 'root'
    });

    // 提供详细的错误诊断信息
    if (error.message.includes('Access denied')) {
      logger.error('Database authentication failed. Please check:', {
        checklist: [
          `✓ DB_USER: ${process.env.DB_USER || 'root'}`,
          `✓ DB_PASSWORD: ${process.env.DB_PASSWORD ? '*** (set)' : '⚠️ NOT SET'}`,
          `✓ DB_HOST: ${process.env.DB_HOST || '127.0.0.1'}`,
          `✓ DB_PORT: ${process.env.DB_PORT || 3306}`,
          `✓ DB_NAME: ${process.env.DB_NAME || 'tenant_management'}`
        ],
        solutions: [
          '1. Verify password in .env file matches MySQL root password',
          '2. Check MySQL user permissions: SELECT user, host FROM mysql.user;',
          '3. If using Docker, ensure DB_HOST points to correct container/service name',
          '4. Grant permissions: GRANT ALL PRIVILEGES ON *.* TO \'root\'@\'%\' IDENTIFIED BY \'password\'; FLUSH PRIVILEGES;'
        ]
      });
    } else if (error.message.includes('ECONNREFUSED') || error.message.includes('ENOTFOUND')) {
      logger.error('Database connection refused. Please check:', {
        checklist: [
          `✓ MySQL service is running`,
          `✓ DB_HOST is correct: ${process.env.DB_HOST || '127.0.0.1'}`,
          `✓ DB_PORT is correct: ${process.env.DB_PORT || 3306}`,
          `✓ Firewall allows connection`
        ],
        solutions: [
          '1. Start MySQL service: sudo service mysql start (Linux) or brew services start mysql (Mac)',
          '2. Check if MySQL is listening: netstat -an | grep 3306',
          '3. If using Docker, check container status: docker ps',
          '4. Verify network connectivity: ping <DB_HOST>'
        ]
      });
    }

    if (process.env.NODE_ENV === 'production') {
      // 生产环境：进入降级模式而不是退出
      dbStatus.degraded = true;
      logger.error('Database connection failed in production, entering degraded mode', {
        maxRetries: dbStatus.maxRetries,
        degraded: true
      });
      logger.warn('⚠️  Database is in degraded mode. Service will continue but database-dependent features are disabled.');
      logger.info('Database will continue to retry periodically. Service will recover automatically when database is available.');

      // 启动健康检查（用于降级模式下的重连）
      startDbHealthCheck();
    } else {
      // 开发模式：进入降级模式
      dbStatus.degraded = true;
      logger.warn('Continuing without database connection for development...');
      logger.warn('⚠️  Database-dependent features will not work until connection is established.');

      // 启动健康检查（用于降级模式下的重连）
      startDbHealthCheck();
    }
  }
};

/**
 * 检查数据库连接状态
 * @returns {Promise<boolean>} 是否连接
 */
const checkConnection = async () => {
  try {
    await sequelize.authenticate();
    if (!dbStatus.connected) {
      dbStatus.connected = true;
      dbStatus.degraded = false;
      dbStatus.retryCount = 0;
      dbStatus.lastError = null;
      logger.info('Database connection recovered');
    }
    return true;
  } catch (error) {
    dbStatus.connected = false;
    dbStatus.lastError = error.message;
    dbStatus.lastErrorTime = new Date().toISOString();
    logger.error('Database connection check failed', { error: error.message });
    return false;
  }
};

/**
 * 健康检查：检查数据库连接池状态
 * @returns {object} 连接池状态
 */
const getPoolStatus = () => {
  try {
    const pool = sequelize.connectionManager.pool;
    return {
      size: pool.size,
      available: pool.available,
      using: pool.using,
      waiting: pool.waiting,
      max: pool.max,
      min: pool.min
    };
  } catch (error) {
    return {
      size: 0,
      available: 0,
      using: 0,
      waiting: 0,
      max: 0,
      min: 0,
      error: error.message
    };
  }
};

/**
 * 获取数据库连接状态信息
 * @returns {object} 连接状态
 */
const getDbStatus = () => {
  return {
    ...dbStatus,
    pool: getPoolStatus()
  };
};

/**
 * 检查是否处于降级模式
 * @returns {boolean} 是否降级
 */
const isDbDegraded = () => {
  return dbStatus.degraded;
};

// 注意：Sequelize 的连接池会自动处理重连
// 连接池会在连接断开时自动创建新连接
// 通过 pool.handleDisconnects: true 配置启用自动重连

module.exports = {
  sequelize,
  testConnection,
  checkConnection,
  getPoolStatus,
  getDbStatus,
  isDbDegraded,
  startDbHealthCheck,
  stopDbHealthCheck
};
