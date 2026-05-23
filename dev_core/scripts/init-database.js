#!/usr/bin/env node

/**
 * 数据库初始化脚本
 * 使用 PostgreSQL 创建业务库，并执行 database/init.sql 完成结构同步。
 */

const { Client } = require("pg");
const bcrypt = require("bcryptjs");
const dayjs = require("dayjs");
const fs = require("fs");
const path = require("path");
require("dotenv").config({ path: path.resolve(__dirname, "../../.env") });
const { buildMetaStoreConfig } = require("../src/config/infra");

const dbConfig = {
  ...buildMetaStoreConfig(),
  connectTimeout: Number(process.env.DB_CONNECT_TIMEOUT || 60000),
};

const NON_CRITICAL_SQL_ERROR_CODES = new Set(["42P07", "42710", "23505"]);

const initialData = {
  superAdmin: {
    username: process.env.SUPER_ADMIN_USERNAME || "superadmin",
    password: process.env.SUPER_ADMIN_PASSWORD || "admin123",
    userId:
      process.env.SUPER_ADMIN_USER_ID || "550e8400-e29b-41d4-a716-446655440001",
  },
  defaultTenant: {
    id: process.env.DEFAULT_TENANT_ID || "550e8400-e29b-41d4-a716-446655440000",
    name: process.env.DEFAULT_TENANT_NAME || "InduForge",
    code: process.env.DEFAULT_TENANT_CODE || "default",
    description: process.env.DEFAULT_TENANT_DESCRIPTION || "InduForge 默认租户",
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

const getSslConfig = () =>
  dbConfig.sslEnabled
    ? {
        rejectUnauthorized:
          String(process.env.DB_SSL_REJECT_UNAUTHORIZED || "false").toLowerCase() ===
          "true",
      }
    : false;

const buildClient = (database) =>
  new Client({
    host: dbConfig.host,
    port: dbConfig.port,
    user: dbConfig.user,
    password: dbConfig.password,
    database,
    connectionTimeoutMillis: dbConfig.connectTimeout,
    ssl: getSslConfig(),
  });

const quoteIdentifier = (value) => {
  const normalized = String(value || "").trim();
  if (!normalized) {
    throw new Error("数据库名称不能为空");
  }
  return `"${normalized.replace(/"/g, '""')}"`;
};

async function insertInitialData(client) {
  try {
    console.log("📝 插入初始数据...");

    const now = dayjs().toDate();

    console.log("🏢 创建默认租户...");
    await client.query(
      `
        INSERT INTO tenants (
          "id",
          "name",
          "code",
          "description",
          "status",
          "contactEmail",
          "maxUsers",
          "maxProjects",
          "createdAt",
          "updatedAt"
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT ("code") DO UPDATE
        SET
          "name" = EXCLUDED."name",
          "description" = EXCLUDED."description",
          "status" = EXCLUDED."status",
          "contactEmail" = EXCLUDED."contactEmail",
          "maxUsers" = EXCLUDED."maxUsers",
          "maxProjects" = EXCLUDED."maxProjects",
          "updatedAt" = EXCLUDED."updatedAt"
      `,
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
      ],
    );

    console.log("👑 创建超级管理员...");
    const superAdminHashedPassword = await bcrypt.hash(
      initialData.superAdmin.password,
      12,
    );
    await client.query(
      `
        INSERT INTO users (
          "id",
          "username",
          "password",
          "fullName",
          "role",
          "status",
          "tenantId",
          "createdAt",
          "updatedAt"
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        ON CONFLICT ("tenantId", "username") DO UPDATE
        SET
          "password" = EXCLUDED."password",
          "fullName" = EXCLUDED."fullName",
          "role" = EXCLUDED."role",
          "status" = EXCLUDED."status",
          "updatedAt" = EXCLUDED."updatedAt"
      `,
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
      ],
    );

    console.log("👤 创建系统管理员...");
    const systemAdminHashedPassword = await bcrypt.hash(
      initialData.tenantDefaults.defaultAdminPassword,
      12,
    );
    const systemAdminId = "550e8400-e29b-41d4-a716-446655440002";
    await client.query(
      `
        INSERT INTO users (
          "id",
          "username",
          "password",
          "fullName",
          "role",
          "status",
          "tenantId",
          "createdAt",
          "updatedAt"
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        ON CONFLICT ("tenantId", "username") DO UPDATE
        SET
          "password" = EXCLUDED."password",
          "fullName" = EXCLUDED."fullName",
          "role" = EXCLUDED."role",
          "status" = EXCLUDED."status",
          "updatedAt" = EXCLUDED."updatedAt"
      `,
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
      ],
    );

    console.log("✅ 初始数据插入完成");
  } catch (error) {
    console.error("❌ 插入初始数据失败:", error);
    throw error;
  }
}

function splitSqlStatements(sqlContent) {
  const statements = [];
  let current = "";
  let inSingleQuote = false;
  let inDoubleQuote = false;
  let inLineComment = false;
  let blockCommentDepth = 0;
  let dollarTag = null;

  /**
   * 按字符扫描 SQL，确保仅在“真实语句边界”的分号处分割：
   * - 兼容单引号/双引号内容
   * - 兼容 -- 行注释与块注释
   * - 兼容 PostgreSQL 的 DO $$ ... $$ / $tag$ ... $tag$ 结构
   */
  for (let index = 0; index < sqlContent.length; index += 1) {
    const char = sqlContent[index];
    const nextChar = sqlContent[index + 1];
    const rest = sqlContent.slice(index);

    if (inLineComment) {
      current += char;
      if (char === "\n") {
        inLineComment = false;
      }
      continue;
    }

    if (blockCommentDepth > 0) {
      current += char;
      if (char === "/" && nextChar === "*") {
        current += nextChar;
        blockCommentDepth += 1;
        index += 1;
        continue;
      }
      if (char === "*" && nextChar === "/") {
        current += nextChar;
        blockCommentDepth -= 1;
        index += 1;
      }
      continue;
    }

    if (dollarTag) {
      if (rest.startsWith(dollarTag)) {
        current += dollarTag;
        index += dollarTag.length - 1;
        dollarTag = null;
      } else {
        current += char;
      }
      continue;
    }

    if (inSingleQuote) {
      current += char;
      if (char === "'" && nextChar === "'") {
        current += nextChar;
        index += 1;
        continue;
      }
      if (char === "'") {
        inSingleQuote = false;
      }
      continue;
    }

    if (inDoubleQuote) {
      current += char;
      if (char === '"' && nextChar === '"') {
        current += nextChar;
        index += 1;
        continue;
      }
      if (char === '"') {
        inDoubleQuote = false;
      }
      continue;
    }

    if (char === "-" && nextChar === "-") {
      current += "--";
      inLineComment = true;
      index += 1;
      continue;
    }

    if (char === "/" && nextChar === "*") {
      current += "/*";
      blockCommentDepth = 1;
      index += 1;
      continue;
    }

    if (char === "'") {
      current += char;
      inSingleQuote = true;
      continue;
    }

    if (char === '"') {
      current += char;
      inDoubleQuote = true;
      continue;
    }

    if (char === "$") {
      const dollarMatch = rest.match(/^\$[A-Za-z_][A-Za-z0-9_]*\$|^\$\$/);
      if (dollarMatch) {
        dollarTag = dollarMatch[0];
        current += dollarTag;
        index += dollarTag.length - 1;
        continue;
      }
    }

    if (char === ";") {
      const statement = current.trim();
      if (statement) {
        statements.push(statement);
      }
      current = "";
      continue;
    }

    current += char;
  }

  const tail = current.trim();
  if (tail) {
    statements.push(tail);
  }

  return statements;
}

async function ensureDatabaseExists(reset = false) {
  const adminClient = buildClient(dbConfig.adminDatabase);

  try {
    await adminClient.connect();

    if (reset) {
      await adminClient.query(
        `
          SELECT pg_terminate_backend(pid)
          FROM pg_stat_activity
          WHERE datname = $1
            AND pid <> pg_backend_pid()
        `,
        [dbConfig.database],
      );
      await adminClient.query(
        `DROP DATABASE IF EXISTS ${quoteIdentifier(dbConfig.database)}`,
      );
    }

    const result = await adminClient.query(
      "SELECT 1 FROM pg_database WHERE datname = $1 LIMIT 1",
      [dbConfig.database],
    );

    if (result.rowCount === 0) {
      await adminClient.query(
        `CREATE DATABASE ${quoteIdentifier(dbConfig.database)}`,
      );
    }
  } finally {
    await adminClient.end().catch(() => {});
  }
}

async function syncDatabaseSchema(options = {}) {
  const { reset = false, seed = true } = options;
  let client;

  try {
    await ensureDatabaseExists(reset);

    client = buildClient(dbConfig.database);
    await client.connect();

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
        await client.query(statement);
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
      await insertInitialData(client);
    }

    return {
      total: statements.length,
      executed,
      skipped,
    };
  } finally {
    if (client) {
      await client.end().catch(() => {});
    }
  }
}

async function executeSqlFile() {
  try {
    console.log("🔄 正在同步数据库结构...");
    const result = await syncDatabaseSchema({ reset: false, seed: true });
    console.log(
      `✅ 数据库结构同步完成，总语句 ${result.total}，执行 ${result.executed}，跳过 ${result.skipped}`,
    );
    console.log("🎉 数据库初始化完成！");
  } catch (error) {
    console.error("❌ 数据库初始化失败:", error);
    process.exit(1);
  }
}

async function resetDatabase() {
  try {
    console.log("🔄 正在重置并同步数据库结构...");
    const result = await syncDatabaseSchema({ reset: true, seed: true });
    console.log(
      `✅ 数据库结构同步完成，总语句 ${result.total}，执行 ${result.executed}，跳过 ${result.skipped}`,
    );
    console.log("🎉 数据库重置完成！");
  } catch (error) {
    console.error("❌ 数据库重置失败:", error);
    process.exit(1);
  }
}

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
      "  node scripts/init-database.js reset   # 重置数据库（删除所有表）",
    );
    process.exit(1);
  }
}

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
  splitSqlStatements,
};
