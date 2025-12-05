#!/usr/bin/env node

/**
 * 数据库初始化脚本
 * 执行 database/init.sql 文件来初始化数据库
 */

const mysql = require('mysql2/promise');
const bcrypt = require('bcryptjs');
const fs = require('fs');
const path = require('path');
require('dotenv').config();

// 数据库配置
const dbConfig = {
  host: process.env.DB_HOST || '127.0.0.1',
  port: process.env.DB_PORT || 3306,
  user: process.env.DB_USER || 'root',
  password: process.env.DB_PASSWORD || '123456',
  database: process.env.DB_NAME || 'tenant_management',
  multipleStatements: true, // 允许执行多条语句
  connectTimeout: 60000,
};

// 初始数据配置
const initialData = {
  superAdmin: {
    username: process.env.SUPER_ADMIN_USERNAME || 'superadmin',
    password: process.env.SUPER_ADMIN_PASSWORD || 'admin123',
    userId: process.env.SUPER_ADMIN_USER_ID || '550e8400-e29b-41d4-a716-446655440001',
  },
  defaultTenant: {
    id: process.env.DEFAULT_TENANT_ID || '550e8400-e29b-41d4-a716-446655440000',
    name: process.env.DEFAULT_TENANT_NAME || '默认租户',
    code: process.env.DEFAULT_TENANT_CODE || 'default',
    description: process.env.DEFAULT_TENANT_DESCRIPTION || '系统默认租户',
    contactEmail: process.env.DEFAULT_TENANT_CONTACT_EMAIL || 'admin@example.com',
    maxUsers: Number(process.env.DEFAULT_TENANT_MAX_USERS || 100),
    maxProjects: Number(process.env.DEFAULT_TENANT_MAX_PROJECTS || 50),
  },
  tenantDefaults: {
    defaultAdminUsername: process.env.TENANT_DEFAULT_ADMIN_USERNAME || 'admin',
    defaultAdminPassword: process.env.TENANT_DEFAULT_ADMIN_PASSWORD || 'admin123',
  },
};

/**
 * 插入初始数据
 */
async function insertInitialData(connection) {
  try {
    console.log('📝 插入初始数据...');

    const now = new Date();

    // 创建默认租户
    console.log('🏢 创建默认租户...');
    await connection.query(
      `INSERT INTO tenants (id, name, code, description, status, contactEmail, maxUsers, maxProjects, createdAt, updatedAt)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
       ON DUPLICATE KEY UPDATE updatedAt = VALUES(updatedAt)`,
      [
        initialData.defaultTenant.id,
        initialData.defaultTenant.name,
        initialData.defaultTenant.code,
        initialData.defaultTenant.description,
        'active',
        initialData.defaultTenant.contactEmail,
        initialData.defaultTenant.maxUsers,
        initialData.defaultTenant.maxProjects,
        now,
        now,
      ]
    );

    // 创建超级管理员用户
    console.log('👑 创建超级管理员...');
    const superAdminHashedPassword = await bcrypt.hash(initialData.superAdmin.password, 12);
    await connection.query(
      `INSERT INTO users (id, username, password, fullName, role, status, tenantId, createdAt, updatedAt)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
       ON DUPLICATE KEY UPDATE updatedAt = VALUES(updatedAt)`,
      [
        initialData.superAdmin.userId,
        initialData.superAdmin.username,
        superAdminHashedPassword,
        '超级管理员',
        'SUPER_ADMIN',
        'active',
        initialData.defaultTenant.id,
        now,
        now,
      ]
    );

    // 创建系统管理员用户
    console.log('👤 创建系统管理员...');
    const systemAdminHashedPassword = await bcrypt.hash(initialData.tenantDefaults.defaultAdminPassword, 12);
    const systemAdminId = '550e8400-e29b-41d4-a716-446655440002';
    await connection.query(
      `INSERT INTO users (id, username, password, fullName, role, status, tenantId, createdAt, updatedAt)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
       ON DUPLICATE KEY UPDATE updatedAt = VALUES(updatedAt)`,
      [
        systemAdminId,
        initialData.tenantDefaults.defaultAdminUsername,
        systemAdminHashedPassword,
        '系统管理员',
        'SYSTEM_ADMIN',
        'active',
        initialData.defaultTenant.id,
        now,
        now,
      ]
    );

    console.log('✅ 初始数据插入完成');
  } catch (error) {
    console.error('❌ 插入初始数据失败:', error);
    throw error;
  }
}

/**
 * 执行 SQL 文件
 */
async function executeSqlFile() {
  let connection;

  try {
    console.log('🔄 正在连接数据库...');

    // 创建数据库连接（不指定数据库）
    const connectionConfig = { ...dbConfig };
    delete connectionConfig.database;

    connection = await mysql.createConnection(connectionConfig);

    // 确保数据库存在
    console.log(`📦 确保数据库 '${dbConfig.database}' 存在...`);
    await connection.query(`CREATE DATABASE IF NOT EXISTS \`${dbConfig.database}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`);

    // 切换到目标数据库
    await connection.query(`USE \`${dbConfig.database}\``);

    console.log('✅ 数据库连接成功');

    // 读取 SQL 文件
    const sqlFilePath = path.join(__dirname, '..', 'database', 'init.sql');
    console.log(`📖 读取 SQL 文件: ${sqlFilePath}`);

    if (!fs.existsSync(sqlFilePath)) {
      throw new Error(`SQL 文件不存在: ${sqlFilePath}`);
    }

    const sqlContent = fs.readFileSync(sqlFilePath, 'utf8');

    console.log('⚡ 执行 SQL 文件...');

    // 使用 query 方法执行整个 SQL 文件
    try {
      await connection.query(sqlContent);
      console.log('✅ SQL 文件执行成功');
    } catch (error) {
      // 对于某些非关键错误（如索引已存在），继续执行
      if (error.code === 'ER_DUP_KEYNAME' || error.code === 'ER_DUP_ENTRY') {
        console.log(`⚠️  跳过已存在的项目: ${error.message}`);
      } else {
        throw error;
      }
    }

    // 插入初始数据
    await insertInitialData(connection);

    console.log('🎉 数据库初始化完成！');

  } catch (error) {
    console.error('❌ 数据库初始化失败:', error);
    process.exit(1);
  } finally {
    if (connection) {
      await connection.end();
    }
  }
}

/**
 * 重置数据库（删除所有表并重新创建）
 */
async function resetDatabase() {
  let connection;

  try {
    console.log('🔄 正在连接数据库进行重置...');

    // 创建数据库连接（不指定数据库，先确保数据库存在）
    const connectionConfig = { ...dbConfig };
    delete connectionConfig.database;

    connection = await mysql.createConnection(connectionConfig);

    // 确保数据库存在
    console.log(`📦 确保数据库 '${dbConfig.database}' 存在...`);
    await connection.query(`CREATE DATABASE IF NOT EXISTS \`${dbConfig.database}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`);

    // 切换到目标数据库
    await connection.query(`USE \`${dbConfig.database}\``);

    console.log('✅ 数据库连接成功');

    console.log('🗑️  删除现有表...');

    // 按照外键依赖关系的相反顺序删除表
    const dropStatements = [
      // 数据中心相关表
      'DROP TABLE IF EXISTS `data_query_logs`',
      'DROP TABLE IF EXISTS `data_sql_configs`',
      'DROP TABLE IF EXISTS `data_queries`',
      'DROP TABLE IF EXISTS `data_relational_configs`',
      'DROP TABLE IF EXISTS `data_connections`',
      // 低代码平台相关表
      'DROP TABLE IF EXISTS `pages`',
      'DROP TABLE IF EXISTS `blocks`',
      'DROP TABLE IF EXISTS `block_groups`',
      'DROP TABLE IF EXISTS `block_categories`',
      // 基础表
      'DROP TABLE IF EXISTS `logs`',
      'DROP TABLE IF EXISTS `projects`',
      'DROP TABLE IF EXISTS `users`',
      'DROP TABLE IF EXISTS `tenants`',
    ];

    for (const statement of dropStatements) {
      try {
        await connection.query(statement);
        console.log(`✅ ${statement}`);
      } catch (error) {
        console.log(`⚠️  ${statement} 失败: ${error.message}`);
      }
    }

    console.log('✅ 数据库重置完成');

  } catch (error) {
    console.error('❌ 数据库重置失败:', error);
    process.exit(1);
  } finally {
    if (connection) {
      await connection.end();
    }
  }
}

// 主函数
async function main() {
  const command = process.argv[2];

  if (command === 'reset') {
    await resetDatabase();
  } else if (command === 'init' || !command) {
    await executeSqlFile();
  } else {
    console.log('使用方法:');
    console.log('  node scripts/init-database.js init    # 初始化数据库');
    console.log('  node scripts/init-database.js reset   # 重置数据库（删除所有表）');
    process.exit(1);
  }
}

// 运行主函数
if (require.main === module) {
  main().catch(error => {
    console.error('❌ 脚本执行失败:', error);
    process.exit(1);
  });
}

module.exports = { executeSqlFile, resetDatabase };
