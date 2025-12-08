# 数据连接管理

## 概述

数据连接管理提供了统一的数据库连接配置和管理功能，支持多种数据库类型。

## 支持的数据库

### MySQL
- 版本: 5.7+
- 驱动: mysql2
- 连接池: 支持

### PostgreSQL
- 版本: 12+
- 驱动: pg
- 连接池: 支持

### SQL Server
- 版本: 2012+
- 驱动: mssql
- 连接池: 支持

## 连接配置

### MySQL 连接

```json
{
  "name": "MySQL 测试库",
  "type": "mysql",
  "host": "localhost",
  "port": 3306,
  "database": "test_db",
  "username": "root",
  "password": "******",
  "options": {
    "connectionLimit": 10,
    "connectTimeout": 10000
  }
}
```

### PostgreSQL 连接

```json
{
  "name": "PostgreSQL 测试库",
  "type": "postgresql",
  "host": "localhost",
  "port": 5432,
  "database": "test_db",
  "username": "postgres",
  "password": "******",
  "options": {
    "max": 10,
    "idleTimeoutMillis": 30000
  }
}
```

### SQL Server 连接

```json
{
  "name": "SQL Server 测试库",
  "type": "sqlserver",
  "host": "localhost",
  "port": 1433,
  "database": "test_db",
  "username": "sa",
  "password": "******",
  "options": {
    "encrypt": true,
    "trustServerCertificate": true
  }
}
```

## 创建连接

### 通过界面创建

1. 打开数据中心
2. 点击"新建连接"按钮
3. 填写连接信息：
   - 连接名称
   - 数据库类型
   - 主机地址
   - 端口号
   - 数据库名
   - 用户名
   - 密码
4. 点击"测试连接"验证
5. 点击"保存"

### 通过 API 创建

```javascript
const response = await fetch('/api/v1/data/projects/PROJECT_ID/connections', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer TOKEN'
  },
  body: JSON.stringify({
    name: 'MySQL 测试库',
    type: 'mysql',
    host: 'localhost',
    port: 3306,
    database: 'test_db',
    username: 'root',
    password: '******'
  })
})
```

## 测试连接

### 通过界面测试

在连接配置页面点击"测试连接"按钮。

### 通过 API 测试

```javascript
const response = await fetch('/api/v1/data/projects/PROJECT_ID/connections/CONNECTION_ID/test', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer TOKEN'
  }
})
```

## 连接池配置

### MySQL 连接池

```json
{
  "options": {
    "connectionLimit": 10,        // 最大连接数
    "queueLimit": 0,              // 队列限制
    "waitForConnections": true,   // 等待连接
    "connectTimeout": 10000       // 连接超时（毫秒）
  }
}
```

### PostgreSQL 连接池

```json
{
  "options": {
    "max": 10,                    // 最大连接数
    "min": 2,                     // 最小连接数
    "idleTimeoutMillis": 30000,   // 空闲超时（毫秒）
    "connectionTimeoutMillis": 2000 // 连接超时（毫秒）
  }
}
```

## 安全配置

### SSL/TLS 连接

```json
{
  "options": {
    "ssl": {
      "rejectUnauthorized": false,
      "ca": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
      "key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
      "cert": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----"
    }
  }
}
```

### 密码加密

数据库密码在存储时会自动加密，使用 AES-256-CBC 算法。

## 连接管理

### 查看连接列表

```javascript
const response = await fetch('/api/v1/data/projects/PROJECT_ID/connections', {
  headers: {
    'Authorization': 'Bearer TOKEN'
  }
})
```

### 更新连接

```javascript
const response = await fetch('/api/v1/data/projects/PROJECT_ID/connections/CONNECTION_ID', {
  method: 'PUT',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer TOKEN'
  },
  body: JSON.stringify({
    name: '新名称',
    host: 'new-host'
  })
})
```

### 删除连接

```javascript
const response = await fetch('/api/v1/data/projects/PROJECT_ID/connections/CONNECTION_ID', {
  method: 'DELETE',
  headers: {
    'Authorization': 'Bearer TOKEN'
  }
})
```

## 常见问题

### Q: 连接超时怎么办？
A: 
1. 检查网络连接
2. 增加 connectTimeout 配置
3. 检查防火墙设置

### Q: 连接池满了怎么办？
A: 
1. 增加 connectionLimit
2. 优化查询性能
3. 及时释放连接

### Q: 如何配置 SSL 连接？
A: 在 options 中添加 ssl 配置，提供证书文件。

### Q: 密码如何保护？
A: 密码在存储时自动加密，传输时使用 HTTPS。

## 最佳实践

### 1. 连接命名
使用有意义的名称，如"生产环境-MySQL"、"测试环境-PostgreSQL"。

### 2. 连接池配置
根据实际负载调整连接池大小，避免过大或过小。

### 3. 超时配置
设置合理的超时时间，避免长时间等待。

### 4. 安全配置
- 使用强密码
- 启用 SSL/TLS
- 限制访问 IP
- 定期更换密码

### 5. 监控和维护
- 定期检查连接状态
- 监控连接池使用情况
- 及时清理无用连接

## 相关资源

- [查询管理](./queries.md)
- [后端 API 文档](../backend/README.md)
- [MySQL 文档](https://dev.mysql.com/doc/)
- [PostgreSQL 文档](https://www.postgresql.org/docs/)
- [SQL Server 文档](https://docs.microsoft.com/sql/)

---

**版本**: 2.0.0  
**最后更新**: 2025-12-08
