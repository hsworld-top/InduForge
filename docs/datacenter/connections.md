# 数据连接管理

## 概述

数据连接管理提供了统一的连接配置和管理功能，支持数据库连接和消息连接两大类。

## 连接分类

DataCenter 支持以下连接类型：

### 数据库连接（现有）
- **MySQL**: 关系型数据库
- **PostgreSQL**: 关系型数据库
- **SQL Server**: 关系型数据库

### 消息连接（新增）
- **MQTT**: 消息队列遥测传输协议，用于实时数据订阅
- **WebSocket**: 未来支持
- **OPC UA**: 工业自动化协议，未来支持

---

## 支持的数据库连接

### MySQL
- 版本: 5.7+
- 驱动: mysql2
- 连接池: 支持
- 状态: ✅ 完整支持

### PostgreSQL
- 版本: 12+
- 驱动: pg
- 连接池: 支持
- Schema: 支持
- 状态: ✅ 完整支持

### SQL Server
- 版本: 2012+
- 驱动: mssql
- 连接池: 支持
- Schema: 支持
- 状态: ✅ 完整支持

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
2. 点击左上角"新建连接"按钮
3. 在对话框中选择数据库类型（MySQL/PostgreSQL/SQL Server）
4. 填写连接信息：
   - 连接名称（必填）
   - 主机地址（默认 localhost）
   - 端口号（自动填充默认端口）
   - 数据库名（必填）
   - 用户名（必填）
   - 密码（必填）
   - 其他选项（根据数据库类型不同）
5. 点击"测试连接"验证配置
6. 测试成功后点击"保存"

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

## 消息连接支持

### MQTT
- 版本: 3.1.1 / 5.0
- 驱动: mqtt.js
- 协议: mqtt/mqtts/ws/wss
- 状态: 🚧 开发中

**配置示例**：
```json
{
  "name": "MQTT Broker",
  "type": "mqtt",
  "brokerUrl": "mqtt://localhost",
  "port": 1883,
  "protocol": "mqtt",
  "clientId": "induforge_client",
  "username": "admin",
  "password": "******",
  "options": {
    "keepalive": 60,
    "cleanSession": true,
    "reconnectPeriod": 5000,
    "connectTimeout": 30000,
    "qos": 0
  }
}
```

### WebSocket
- 状态: ⏳ 未来支持

### OPC UA
- 状态: ⏳ 未来支持

---

## 连接管理

### 查看连接列表
连接列表显示在左侧面板，按类型分组：

**数据库连接**：
- 连接名称
- 连接状态（已连接/未连接/错误）
- 查询列表（蓝色区域）
- 表列表（绿色区域）

**消息连接**：
- 连接名称
- 连接状态（已连接/未连接/错误）
- 主题订阅列表（蓝色区域）
- 变量列表（绿色区域）

### 连接操作
- **双击连接**: 测试连接并展开/折叠
- **右键菜单**: 
  - 打开连接
  - 断开连接
  - 查看详情
  - 编辑连接
  - 删除连接

### 连接状态
- **已连接** (绿色): 连接测试成功，可以执行查询
- **未连接** (灰色): 尚未连接或已断开
- **错误** (红色): 连接失败或配置错误

### 更新连接
1. 右键点击连接
2. 选择"编辑连接"
3. 修改连接信息
4. 测试连接
5. 保存更改

### 删除连接
1. 右键点击连接
2. 选择"删除连接"
3. 确认删除操作
4. 注意：删除连接会同时删除关联的查询

## 常见问题

### Q: 连接超时怎么办？
A: 
1. 检查网络连接和防火墙设置
2. 确认数据库服务正在运行
3. 验证主机地址和端口号是否正确
4. 检查数据库用户权限

### Q: 连接测试失败？
A: 
1. 验证用户名和密码
2. 确认数据库名称正确
3. 检查数据库是否允许远程连接
4. 查看错误提示信息

### Q: 如何切换数据库？
A: 在查询编辑器中使用连接选择器可以切换到其他已配置的连接

### Q: 密码如何保护？
A: 密码在存储时自动加密，传输时使用 HTTPS

### Q: 支持 SSL/TLS 连接吗？
A: 
- PostgreSQL: 支持 SSL 配置（sslMode、证书等）
- SQL Server: 支持加密连接（encrypt、trustServerCertificate）
- MySQL: 支持 SSL 连接配置

## 最佳实践

### 1. 连接命名
使用有意义的名称，如"生产环境-MySQL"、"测试环境-PostgreSQL"、"开发-SQL Server"

### 2. 连接测试
创建连接后立即测试，确保配置正确

### 3. 安全配置
- 使用强密码
- 生产环境启用 SSL/TLS
- 限制数据库访问 IP
- 定期更换密码
- 使用只读账户进行查询操作

### 4. 连接管理
- 及时断开不使用的连接
- 定期检查连接状态
- 删除无用的连接配置

### 5. 权限控制
- 为不同环境使用不同的数据库账户
- 限制账户权限（避免使用 root/sa）
- 生产环境使用只读账户

## 相关资源

- [查询管理](./queries.md)
- [后端 API 文档](../backend/README.md)
- [MySQL 文档](https://dev.mysql.com/doc/)
- [PostgreSQL 文档](https://www.postgresql.org/docs/)
- [SQL Server 文档](https://docs.microsoft.com/sql/)

---

**版本**: 2.1.0  
**最后更新**: 2025-12-15
