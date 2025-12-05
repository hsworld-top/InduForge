# 数据库初始化

本项目使用简化的数据库初始化方式，通过 SQL 文件创建表结构，JavaScript 脚本处理初始数据。

## 使用方法

```bash
# 初始化数据库（创建表结构和基础数据）
npm run db:init

# 重置数据库（删除所有表后重新初始化）
npm run db:reset
```

## 文件说明

- `database/init.sql` - 数据库表结构 SQL 文件，包含：
  - 所有表的创建语句
  - 索引创建语句
  - 外键约束定义

- `scripts/init-database.js` - Node.js 初始化脚本
  - 执行 SQL 文件创建表结构
  - 使用 bcrypt 对密码进行哈希处理
  - 插入初始数据（默认租户和管理员用户）

## 初始数据

系统会自动创建以下初始数据：

### 默认租户
- ID: `550e8400-e29b-41d4-a716-446655440000`
- 名称: 默认租户
- 代码: default

### 默认用户

#### 超级管理员
- 用户名: superadmin
- 密码: admin123
- 角色: SUPER_ADMIN

#### 系统管理员
- 用户名: admin
- 密码: admin123
- 角色: SYSTEM_ADMIN

## 环境变量

脚本会从 `.env` 文件读取以下数据库配置：

```env
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=123456
DB_NAME=tenant_management
```

## 架构优势

这种混合架构结合了 SQL 和 JavaScript 的优点：

1. **表结构清晰可见** - SQL 文件直接展示所有表结构、索引和约束
2. **初始数据灵活处理** - JavaScript 可以进行密码哈希等复杂数据处理
3. **开发友好** - 无需管理复杂的迁移文件版本控制
4. **运行时保持 Sequelize** - 应用运行时仍使用 Sequelize 进行数据操作
5. **快速重置** - 开发时可以快速重建数据库环境
