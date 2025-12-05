# 数据库驱动架构说明

## 概述

本目录包含数据库驱动的抽象层实现，支持多种数据库类型，采用策略模式设计，便于扩展。

## 架构设计

### 基类：BaseDriver

所有数据库驱动都需要继承 `BaseDriver` 并实现以下方法：

- `createConnection()` - 创建数据库连接
- `testConnection()` - 测试连接
- `executeQuery(sql, parameters)` - 执行 SQL 查询
- `getTables()` - 获取表列表
- `getTableData(tableName, options)` - 获取表数据
- `closeConnection(connection)` - 关闭连接
- `escapeIdentifier(identifier)` - 转义标识符

### 已实现的驱动

1. **MySqlDriver** - MySQL/MariaDB 驱动
2. **PostgreSqlDriver** - PostgreSQL 驱动
3. **SqlServerDriver** - SQL Server 驱动

### 驱动工厂：DriverFactory

使用工厂模式创建驱动实例：

```javascript
const driver = DriverFactory.createDriver('mysql', config);
const result = await driver.executeQuery('SELECT * FROM users WHERE id = ?', [1]);
```

## 添加新数据库驱动

1. 创建新的驱动类，继承 `BaseDriver`
2. 实现所有必需的方法
3. 在 `DriverFactory.createDriver()` 中添加新的 case

示例：

```javascript
const BaseDriver = require('./BaseDriver');

class OracleDriver extends BaseDriver {
  constructor(config) {
    super(config);
    this.dbType = 'oracle';
  }

  async createConnection() {
    // 实现创建连接逻辑
  }

  async executeQuery(sql, parameters) {
    // 实现查询逻辑
  }

  // ... 实现其他方法
}

module.exports = OracleDriver;
```

然后在 `DriverFactory.js` 中添加：

```javascript
case 'oracle':
  return new OracleDriver(config);
```

## 注意事项

1. 参数占位符转换：不同数据库使用不同的占位符
   - MySQL: `?`
   - PostgreSQL: `$1, $2, $3`
   - SQL Server: `@param1, @param2`

2. 连接管理：确保在 finally 块中正确关闭连接

3. 错误处理：使用统一的错误处理机制

