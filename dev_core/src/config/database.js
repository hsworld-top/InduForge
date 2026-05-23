const { Sequelize } = require("sequelize");
const { Client } = require("pg");
const dayjs = require("dayjs");
require("dotenv").config();
const { logger } = require("../utils/logger");
const { TIME_FORMAT } = require("../constants/time");
const { buildMetaStoreConfig } = require("./infra");

const USER_ROLE_VALUES = [
  "SUPER_ADMIN",
  "SYSTEM_ADMIN",
  "PROJECT_ADMIN",
  "OPS_ADMIN",
  "USER_ADMIN",
  "DEVELOPER",
  "OPERATOR",
  "VIEWER",
];

const getDatabaseConfig = buildMetaStoreConfig;

const getPgSslConfig = (dbConfig) =>
  dbConfig.sslEnabled
    ? {
        rejectUnauthorized:
          String(process.env.DB_SSL_REJECT_UNAUTHORIZED || "false").toLowerCase() ===
          "true",
      }
    : false;

const buildAdminClient = (database) => {
  const dbConfig = getDatabaseConfig();
  return new Client({
    host: dbConfig.host,
    port: dbConfig.port,
    user: dbConfig.user,
    password: dbConfig.password,
    database,
    connectionTimeoutMillis: dbConfig.connectTimeout,
    ssl: getPgSslConfig(dbConfig),
  });
};

const sequelizeConfig = getDatabaseConfig();

const sequelize = new Sequelize(
  sequelizeConfig.database,
  sequelizeConfig.user,
  sequelizeConfig.password,
  {
    host: sequelizeConfig.host,
    port: sequelizeConfig.port,
    dialect: "postgres",
    logging: false,
    retry: {
      max: Number(process.env.DB_QUERY_RETRY_MAX || 2),
      match: [
        /SequelizeConnectionError/i,
        /SequelizeConnectionRefusedError/i,
        /SequelizeConnectionAcquireTimeoutError/i,
        /SequelizeHostNotReachableError/i,
        /SequelizeConnectionTimedOutError/i,
        /ECONNRESET/i,
        /ETIMEDOUT/i,
        /Connection terminated unexpectedly/i,
        /terminating connection due to administrator command/i,
      ],
    },
    pool: {
      max: Number(process.env.DB_MAX_CONNECTIONS || 50),
      min: Number(process.env.DB_MIN_CONNECTIONS || 5),
      acquire: 10000,
      idle: 30000,
      evict: 5000,
      validate: (connection) => {
        if (!connection) return false;
        if (connection._ending || connection.ending) return false;
        if (connection._connected === false) return false;
        return true;
      },
    },
    dialectOptions: {
      statement_timeout: sequelizeConfig.queryTimeout,
      query_timeout: sequelizeConfig.queryTimeout,
      keepAlive: true,
      application_name: "dev_core",
      ssl: getPgSslConfig(sequelizeConfig),
    },
    hooks: {
      afterConnect: (connection) => {
        if (process.env.NODE_ENV === "development") {
          logger.debug("Database connection established", {
            processId: connection.processID,
          });
        }
      },
      afterDisconnect: (connection) => {
        if (process.env.NODE_ENV === "development") {
          logger.warn("Database connection closed", {
            processId: connection?.processID,
          });
        }
      },
    },
  },
);

/**
 * 数据库连接状态
 */
let dbStatus = {
  connected: false,
  degraded: false,
  lastError: null,
  lastErrorTime: null,
  retryCount: 0,
  maxRetries: Number(process.env.DB_MAX_RETRIES || 10),
};

/**
 * 定期检查数据库连接（用于降级模式下的重连）
 */
let dbHealthCheckInterval = null;

const startDbHealthCheck = () => {
  if (dbHealthCheckInterval) {
    return;
  }

  const checkInterval = Number(process.env.DB_DEGRADED_RETRY_INTERVAL || 30000);

  dbHealthCheckInterval = setInterval(async () => {
    if (dbStatus.degraded && !dbStatus.connected) {
      try {
        await sequelize.authenticate();
        dbStatus.connected = true;
        dbStatus.degraded = false;
        dbStatus.retryCount = 0;
        dbStatus.lastError = null;
        logger.info("Database connection recovered from degraded mode");
      } catch (error) {
        dbStatus.retryCount += 1;
        dbStatus.lastError = error.message;
        dbStatus.lastErrorTime = dayjs().format(TIME_FORMAT);
        logger.debug(
          `Database health check failed (attempt ${dbStatus.retryCount}): ${error.message}`,
        );
      }
    }
  }, checkInterval);

  logger.info("Database health check started", { interval: checkInterval });
};

const stopDbHealthCheck = () => {
  if (dbHealthCheckInterval) {
    clearInterval(dbHealthCheckInterval);
    dbHealthCheckInterval = null;
    logger.info("Database health check stopped");
  }
};

/**
 * 是否启用启动时数据库结构自动同步
 * 优先读取 DB_AUTO_SCHEMA_SYNC，兼容历史 ENABLE_DB_SYNC 配置。
 * @returns {boolean}
 */
const shouldAutoSyncSchema = () => {
  const configuredValue =
    process.env.DB_AUTO_SCHEMA_SYNC ?? process.env.ENABLE_DB_SYNC ?? "true";
  return String(configuredValue).toLowerCase() === "true";
};

/**
 * 兼容历史 PostgreSQL 库结构：确保 users.role 约束包含当前全部角色。
 * 该修复不阻断启动，仅用于平滑接入旧库。
 * @returns {Promise<void>}
 */
const ensureUserRoleEnumCompatibility = async () => {
  const roleList = USER_ROLE_VALUES.map((value) => `'${value}'`).join(", ");

  try {
    await sequelize.query(`
      ALTER TABLE users
      DROP CONSTRAINT IF EXISTS users_role_check
    `);
    await sequelize.query(`
      ALTER TABLE users
      ADD CONSTRAINT users_role_check
      CHECK ("role" IN (${roleList}))
    `);
    logger.info("✅ users.role 约束兼容检查完成");
  } catch (error) {
    logger.warn("users.role 约束兼容修复失败，将继续启动服务", {
      error: error.message,
    });
  }
};

/**
 * 测试 PostgreSQL 服务连接（连接到管理库，不指定业务库）
 * @returns {Promise<{success: boolean, error?: Error}>}
 */
const testPostgresServerConnection = async () => {
  const dbConfig = getDatabaseConfig();
  const client = buildAdminClient(dbConfig.adminDatabase);

  try {
    await client.connect();
    return { success: true };
  } catch (error) {
    return { success: false, error };
  } finally {
    await client.end().catch(() => {});
  }
};

/**
 * 检查数据库是否存在
 * @returns {Promise<boolean>}
 */
const checkDatabaseExists = async () => {
  const dbConfig = getDatabaseConfig();
  const client = buildAdminClient(dbConfig.adminDatabase);

  try {
    await client.connect();
    const result = await client.query(
      "SELECT 1 FROM pg_database WHERE datname = $1 LIMIT 1",
      [dbConfig.database],
    );
    return result.rowCount > 0;
  } catch (error) {
    logger.error("检查数据库是否存在时出错", { error: error.message });
    return false;
  } finally {
    await client.end().catch(() => {});
  }
};

/**
 * 自动创建并初始化数据库
 * @returns {Promise<boolean>}
 */
const autoInitializeDatabase = async () => {
  try {
    logger.info("🔄 检测到数据库不存在，开始自动创建并同步结构...");
    const { syncDatabaseSchema } = require("../../scripts/init-database");
    const syncResult = await syncDatabaseSchema({ reset: false, seed: true });
    logger.info("✅ 数据库自动创建并同步完成", syncResult);
    return true;
  } catch (error) {
    logger.error("❌ 自动初始化数据库失败", {
      error: error.message,
      stack: error.stack,
    });

    console.log("\n🔧 数据库初始化指南:");
    console.log("=".repeat(50));
    console.log("1. 确保 PostgreSQL 服务正在运行");
    console.log("2. 手动执行数据库初始化脚本:");
    console.log(`   cd ${process.cwd()}`);
    console.log("   node scripts/init-database.js init");
    console.log("=".repeat(50));
    console.log("");

    return false;
  }
};

// 测试数据库连接
const testConnection = async () => {
  const dbConfig = getDatabaseConfig();

  try {
    logger.info("🔍 测试 PostgreSQL 服务连接...");
    const serverTestResult = await testPostgresServerConnection();

    if (!serverTestResult.success) {
      const error = serverTestResult.error;
      logger.error("❌ PostgreSQL 服务连接失败，无法启动应用", {
        host: dbConfig.host,
        port: dbConfig.port,
        user: dbConfig.user,
        error: error.message,
        code: error.code,
      });

      console.log("\n🔧 PostgreSQL 服务连接问题排查:");
      console.log("=".repeat(50));
      console.log("1. 检查 PostgreSQL 服务是否正在运行");
      console.log("2. 检查连接配置:");
      console.log(`   - DB_HOST: ${dbConfig.host}`);
      console.log(`   - DB_PORT: ${dbConfig.port}`);
      console.log(`   - DB_USER: ${dbConfig.user}`);
      console.log(
        `   - DB_PASSWORD: ${dbConfig.password ? "*** (已设置)" : "⚠️ 未设置"}`,
      );
      console.log("");
      console.log("3. 测试网络连接:");
      console.log(`   - Test-NetConnection ${dbConfig.host} -Port ${dbConfig.port}`);
      console.log("=".repeat(50));
      console.log("");

      logger.warn("应用将退出，请先解决 PostgreSQL 服务连接问题");
      process.exit(1);
    }

    logger.info("✅ PostgreSQL 服务连接成功");

    logger.info(`🔍 检查数据库 '${dbConfig.database}' 是否存在...`);
    const dbExists = await checkDatabaseExists();

    if (!dbExists) {
      logger.warn(`⚠️ 数据库 '${dbConfig.database}' 不存在，将自动创建并初始化`);
      const initSuccess = await autoInitializeDatabase();

      if (!initSuccess) {
        logger.error("❌ 数据库自动初始化失败，应用将退出");
        process.exit(1);
      }

      await new Promise((resolve) => setTimeout(resolve, 1000));
    } else {
      logger.info(`✅ 数据库 '${dbConfig.database}' 已存在`);
    }

    logger.info("🔍 测试业务数据库连接...");
    await sequelize.authenticate();
    logger.info("✅ 数据库连接成功");

    if (shouldAutoSyncSchema()) {
      logger.info("🔄 开始数据库结构一致性同步（init.sql）...");
      const { syncDatabaseSchema } = require("../../scripts/init-database");
      const syncResult = await syncDatabaseSchema({
        reset: false,
        seed: false,
      });
      logger.info("✅ 数据库结构一致性同步完成", syncResult);
    } else {
      logger.info("⏭️ 已禁用数据库结构自动同步（DB_AUTO_SCHEMA_SYNC=false）");
    }

    await ensureUserRoleEnumCompatibility();

    dbStatus.connected = true;
    dbStatus.degraded = false;
    dbStatus.retryCount = 0;
    dbStatus.lastError = null;

    startDbHealthCheck();
  } catch (error) {
    if (error.code === "3D000" || error.message.includes("does not exist")) {
      logger.error("❌ 数据库不存在，自动初始化可能失败", {
        database: dbConfig.database,
        error: error.message,
      });

      console.log("\n🔧 数据库初始化指南:");
      console.log("=".repeat(50));
      console.log("1. 确保 PostgreSQL 服务正在运行");
      console.log("2. 执行数据库初始化脚本:");
      console.log(`   cd ${process.cwd()}`);
      console.log("   node scripts/init-database.js init");
      console.log("=".repeat(50));
      console.log("");

      logger.warn("应用将退出，请先初始化数据库后再启动");
      process.exit(1);
    }

    dbStatus.connected = false;
    dbStatus.lastError = error.message;
    dbStatus.lastErrorTime = dayjs().format(TIME_FORMAT);
    logger.error("Unable to connect to the database", {
      error: error.message,
      code: error.code,
      host: dbConfig.host,
      port: dbConfig.port,
      database: dbConfig.database,
      user: dbConfig.user,
    });

    if (error.message.includes("password authentication failed")) {
      logger.error("Database authentication failed. Please check:", {
        checklist: [
          `✓ DB_USER: ${dbConfig.user}`,
          `✓ DB_PASSWORD: ${dbConfig.password ? "*** (set)" : "⚠️ NOT SET"}`,
          `✓ DB_HOST: ${dbConfig.host}`,
          `✓ DB_PORT: ${dbConfig.port}`,
          `✓ DB_NAME: ${dbConfig.database}`,
        ],
      });
    } else if (
      error.message.includes("ECONNREFUSED") ||
      error.message.includes("ENOTFOUND")
    ) {
      logger.error("Database connection refused. Please check:", {
        checklist: [
          "✓ PostgreSQL service is running",
          `✓ DB_HOST is correct: ${dbConfig.host}`,
          `✓ DB_PORT is correct: ${dbConfig.port}`,
          "✓ Firewall allows connection",
        ],
      });
    }

    dbStatus.degraded = true;
    if (process.env.NODE_ENV === "production") {
      logger.error(
        "Database connection failed in production, entering degraded mode",
        {
          maxRetries: dbStatus.maxRetries,
          degraded: true,
        },
      );
    } else {
      logger.warn("Continuing without database connection for development...");
      logger.warn(
        "⚠️ Database-dependent features will not work until connection is established.",
      );
    }

    startDbHealthCheck();
  }
};

/**
 * 检查数据库连接状态
 * @returns {Promise<boolean>}
 */
const checkConnection = async () => {
  try {
    await sequelize.authenticate();
    if (!dbStatus.connected) {
      dbStatus.connected = true;
      dbStatus.degraded = false;
      dbStatus.retryCount = 0;
      dbStatus.lastError = null;
      logger.info("Database connection recovered");
    }
    return true;
  } catch (error) {
    dbStatus.connected = false;
    dbStatus.lastError = error.message;
    dbStatus.lastErrorTime = dayjs().format(TIME_FORMAT);
    logger.error("Database connection check failed", { error: error.message });
    return false;
  }
};

/**
 * 健康检查：检查数据库连接池状态
 * @returns {object}
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
      min: pool.min,
    };
  } catch (error) {
    return {
      size: 0,
      available: 0,
      using: 0,
      waiting: 0,
      max: 0,
      min: 0,
      error: error.message,
    };
  }
};

/**
 * 获取数据库连接状态信息
 * @returns {object}
 */
const getDbStatus = () => ({
  ...dbStatus,
  pool: getPoolStatus(),
});

/**
 * 检查是否处于降级模式
 * @returns {boolean}
 */
const isDbDegraded = () => dbStatus.degraded;

module.exports = {
  sequelize,
  testConnection,
  checkConnection,
  getPoolStatus,
  getDbStatus,
  isDbDegraded,
  startDbHealthCheck,
  stopDbHealthCheck,
};
