#!/usr/bin/env node

/**
 * 数据库初始化脚本
 * 执行 database/init.sql 文件来初始化数据库
 */

const mysql = require("mysql2/promise");
const bcrypt = require("bcryptjs");
const dayjs = require("dayjs");
const fs = require("fs");
const path = require("path");
require("dotenv").config({ path: path.resolve(__dirname, "../../.env") });

// 数据库配置
const dbConfig = {
  host: process.env.DB_HOST || "127.0.0.1",
  port: process.env.DB_PORT || 3306,
  user: process.env.DB_USER || "root",
  password: process.env.DB_PASSWORD || "123456",
  database: process.env.DB_NAME || "tenant_management",
  multipleStatements: true, // 允许执行多条语句
  connectTimeout: 60000,
};

const NON_CRITICAL_SQL_ERROR_CODES = new Set([
  "ER_TABLE_EXISTS_ERROR",
  "ER_DUP_KEYNAME",
  "ER_DUP_FIELDNAME",
  "ER_FK_DUP_NAME",
  "ER_DUP_ENTRY",
]);

// 初始数据配置
const initialData = {
  superAdmin: {
    username: process.env.SUPER_ADMIN_USERNAME || "superadmin",
    password: process.env.SUPER_ADMIN_PASSWORD || "admin123",
    userId:
      process.env.SUPER_ADMIN_USER_ID || "550e8400-e29b-41d4-a716-446655440001",
  },
  defaultTenant: {
    id: process.env.DEFAULT_TENANT_ID || "550e8400-e29b-41d4-a716-446655440000",
    name: process.env.DEFAULT_TENANT_NAME || "默认租户",
    code: process.env.DEFAULT_TENANT_CODE || "default",
    description: process.env.DEFAULT_TENANT_DESCRIPTION || "系统默认租户",
    contactEmail:
      process.env.DEFAULT_TENANT_CONTACT_EMAIL || "admin@example.com",
    maxUsers: Number(process.env.DEFAULT_TENANT_MAX_USERS || 100),
    maxProjects: Number(process.env.DEFAULT_TENANT_MAX_PROJECTS || 50),
  },
  tenantDefaults: {
    defaultAdminUsername: process.env.TENANT_DEFAULT_ADMIN_USERNAME || "admin",
    defaultAdminPassword:
      process.env.TENANT_DEFAULT_ADMIN_PASSWORD || "admin123",
  },
};

/**
 * 插入初始数据
 */
async function insertInitialData(connection) {
  try {
    console.log("📝 插入初始数据...");

    const now = dayjs().toDate();

    // 创建默认租户
    console.log("🏢 创建默认租户...");
    await connection.query(
      `INSERT INTO tenants (id, name, code, description, status, contactEmail, maxUsers, maxProjects, createdAt, updatedAt)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
       ON DUPLICATE KEY UPDATE updatedAt = VALUES(updatedAt)`,
      [
        initialData.defaultTenant.id,
        initialData.defaultTenant.name,
        initialData.defaultTenant.code,
        initialData.defaultTenant.description,
        "active",
        initialData.defaultTenant.contactEmail,
        initialData.defaultTenant.maxUsers,
        initialData.defaultTenant.maxProjects,
        now,
        now,
      ]
    );

    // 创建超级管理员用户
    console.log("👑 创建超级管理员...");
    const superAdminHashedPassword = await bcrypt.hash(
      initialData.superAdmin.password,
      12
    );
    await connection.query(
      `INSERT INTO users (id, username, password, fullName, role, status, tenantId, createdAt, updatedAt)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
       ON DUPLICATE KEY UPDATE updatedAt = VALUES(updatedAt)`,
      [
        initialData.superAdmin.userId,
        initialData.superAdmin.username,
        superAdminHashedPassword,
        "超级管理员",
        "SUPER_ADMIN",
        "active",
        initialData.defaultTenant.id,
        now,
        now,
      ]
    );

    // 创建系统管理员用户
    console.log("👤 创建系统管理员...");
    const systemAdminHashedPassword = await bcrypt.hash(
      initialData.tenantDefaults.defaultAdminPassword,
      12
    );
    const systemAdminId = "550e8400-e29b-41d4-a716-446655440002";
    await connection.query(
      `INSERT INTO users (id, username, password, fullName, role, status, tenantId, createdAt, updatedAt)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
       ON DUPLICATE KEY UPDATE updatedAt = VALUES(updatedAt)`,
      [
        systemAdminId,
        initialData.tenantDefaults.defaultAdminUsername,
        systemAdminHashedPassword,
        "系统管理员",
        "SYSTEM_ADMIN",
        "active",
        initialData.defaultTenant.id,
        now,
        now,
      ]
    );

    console.log("✅ 初始数据插入完成");
  } catch (error) {
    console.error("❌ 插入初始数据失败:", error);
    throw error;
  }
}

/**
 * 将 SQL 脚本拆分为逐条可执行语句（去除注释）
 * @param {string} sqlContent SQL 文件内容
 * @returns {string[]}
 */
function splitSqlStatements(sqlContent) {
  const lines = sqlContent
    .split("\n")
    .filter((line) => {
      const trimmedLine = line.trim();
      return !trimmedLine.startsWith("--");
    });

  return lines
    .join("\n")
    .split(";")
    .map((statement) => statement.trim())
    .filter(Boolean);
}

/**
 * 执行 init.sql 的结构同步，确保数据库结构与 SQL 定义一致
 * @param {object} options 配置
 * @param {boolean} [options.reset=false] 是否先重置数据库
 * @param {boolean} [options.seed=true] 是否插入初始数据
 * @returns {Promise<{total:number, executed:number, skipped:number}>}
 */
async function syncDatabaseSchema(options = {}) {
  const { reset = false, seed = true } = options;
  let connection;

  try {
    const connectionConfig = { ...dbConfig };
    delete connectionConfig.database;
    connection = await mysql.createConnection(connectionConfig);

    if (reset) {
      await connection.query(`DROP DATABASE IF EXISTS \`${dbConfig.database}\``);
    }

    await connection.query(
      `CREATE DATABASE IF NOT EXISTS \`${dbConfig.database}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`
    );
    await connection.query(`USE \`${dbConfig.database}\``);

    const sqlFilePath = path.join(__dirname, "..", "database", "init.sql");
    if (!fs.existsSync(sqlFilePath)) {
      throw new Error(`SQL 文件不存在: ${sqlFilePath}`);
    }

    const sqlContent = fs.readFileSync(sqlFilePath, "utf8");
    const statements = splitSqlStatements(sqlContent);
    let executed = 0;
    let skipped = 0;

    for (const statement of statements) {
      try {
        await connection.query(statement);
        executed += 1;
      } catch (error) {
        if (NON_CRITICAL_SQL_ERROR_CODES.has(error.code)) {
          skipped += 1;
          continue;
        }
        throw error;
      }
    }

    if (seed) {
      await insertInitialData(connection);
    }

    return {
      total: statements.length,
      executed,
      skipped,
    };
  } finally {
    if (connection) {
      await connection.end();
    }
  }
}

/**
 * 执行 SQL 文件
 */
async function executeSqlFile() {
  try {
    console.log("🔄 正在同步数据库结构...");
    const result = await syncDatabaseSchema({ reset: false, seed: true });
    console.log(
      `✅ 数据库结构同步完成，总语句 ${result.total}，执行 ${result.executed}，跳过 ${result.skipped}`
    );
    console.log("🎉 数据库初始化完成！");
  } catch (error) {
    console.error("❌ 数据库初始化失败:", error);
    process.exit(1);
  }
}

/**
 * 重置数据库（删除数据库并重新创建所有表）
 */
async function resetDatabase() {
  try {
    console.log("🔄 正在重置并同步数据库结构...");
    const result = await syncDatabaseSchema({ reset: true, seed: true });
    console.log(
      `✅ 数据库结构同步完成，总语句 ${result.total}，执行 ${result.executed}，跳过 ${result.skipped}`
    );
    console.log("🎉 数据库重置完成！");
  } catch (error) {
    console.error("❌ 数据库重置失败:", error);
    process.exit(1);
  }
}

// 主函数
async function main() {
  const command = process.argv[2];

  if (command === "reset") {
    await resetDatabase();
  } else if (command === "init" || !command) {
    await executeSqlFile();
  } else {
    console.log("使用方法:");
    console.log("  node scripts/init-database.js init    # 初始化数据库");
    console.log(
      "  node scripts/init-database.js reset   # 重置数据库（删除所有表）"
    );
    process.exit(1);
  }
}

// 运行主函数
if (require.main === module) {
  main().catch((error) => {
    console.error("❌ 脚本执行失败:", error);
    process.exit(1);
  });
}

module.exports = {
  executeSqlFile,
  resetDatabase,
  syncDatabaseSchema,
};
