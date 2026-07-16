# 控制面数据库 Bootstrap

本目录存放 `dev_core` 内部数据库 bootstrap 资产，用于从零初始化 `if_core`。`if_core` 同时承载控制面和设计中心元数据。

## 文件说明

- `init-core-database.js`：内部初始化入口，负责在空数据库中创建 `if_core`、执行最终结构 SQL，并创建默认租户和管理员数据。
- `sql/core-schema.sql`：`if_core` 表结构 SQL，包含表、索引、约束、注释和设计中心相关表。

## 使用方式

开发环境不需要手动初始化空数据库。`DB_AUTO_SCHEMA_SYNC=true` 时，仅当 `if_core` 不存在或没有业务表才执行初始化；已有数据库不会自动修改结构。

生产和离线安装环境由安装脚本调用：

```bash
node scripts/bootstrap/init-core-database.js init
```

该命令属于安装流程内部实现，不作为开发人员日常入口。

## 初始数据

初始化会自动补齐以下数据：

- 默认租户：`InduForge` / `default`
- 超级管理员：`superadmin`
- 默认系统管理员：`admin`

默认密码来自根目录 `.env`：

- `SUPER_ADMIN_PASSWORD`
- `TENANT_DEFAULT_ADMIN_PASSWORD`

## 约束

- 本目录不提供版本迁移和 reset 命令，避免自动修改或误删已有数据。
- 生产运行时应保持 `DB_AUTO_SCHEMA_SYNC=false`，只在安装阶段执行初始化。
- 开发环境如需重置数据库，应通过开发环境清理脚本或显式删除 Docker 数据卷完成。
