/**
 * 连接类型配置
 * 定义所有支持的连接类型及其特性
 */

export const CONNECTION_TYPES = {
  // MySQL
  mysql: {
    label: 'MySQL',
    category: 'database',
    icon: 'Database',
    defaultPort: 3306,
    formComponent: 'MysqlConnectionForm',
    contentComponent: 'MysqlContent',
    sqlDialect: 'mysql',
    features: {
      supportTransactions: true,
      supportStoredProcedures: true,
      supportViews: true,
      supportTriggers: true,
      caseSensitive: false,
      identifierQuote: '`',
      stringQuote: "'",
    },
    defaultConfig: {
      host: 'localhost',
      port: 3306,
      database: '',
      username: 'root',
      password: '',
      charset: 'utf8mb4',
      queryTimeout: 30000,
    }
  },

  // PostgreSQL
  postgresql: {
    label: 'PostgreSQL',
    category: 'database',
    icon: 'Database',
    defaultPort: 5432,
    formComponent: 'PostgresConnectionForm',
    contentComponent: 'PostgresContent',
    sqlDialect: 'pgsql',
    features: {
      supportTransactions: true,
      supportStoredProcedures: true,
      supportViews: true,
      supportTriggers: true,
      caseSensitive: true,
      identifierQuote: '"',
      stringQuote: "'",
      supportSchemas: true,
    },
    defaultConfig: {
      host: 'localhost',
      port: 5432,
      database: '',
      username: 'postgres',
      password: '',
      schema: 'public',
      sslMode: 'disable',
      sslCa: '',
      sslCert: '',
      sslKey: '',
      connectionTimeout: 3000,
      queryTimeout: 30000,
      maxConnections: 10,
    }
  },

  // SQL Server
  sqlserver: {
    label: 'SQL Server',
    category: 'database',
    icon: 'Database',
    defaultPort: 1433,
    formComponent: 'SqlServerConnectionForm',
    contentComponent: 'SqlServerContent',
    sqlDialect: 'mssql',
    features: {
      supportTransactions: true,
      supportStoredProcedures: true,
      supportViews: true,
      supportTriggers: true,
      caseSensitive: false,
      identifierQuote: '[',
      identifierQuoteEnd: ']',
      stringQuote: "'",
      supportSchemas: true,
    },
    defaultConfig: {
      host: 'localhost',
      port: 1433,
      database: '',
      username: 'sa',
      password: '',
      timeout: 60000,
      queryTimeout: 30000,
      encrypt: false,
      trustServerCertificate: true,
    }
  },
}

/**
 * 获取连接类型配置
 * @param {string} type - 连接类型
 * @returns {Object} 连接类型配置
 */
export function getConnectionTypeConfig(type) {
  return CONNECTION_TYPES[type] || null
}

/**
 * 获取所有数据库类型
 * @returns {Array} 数据库类型列表
 */
export function getDatabaseTypes() {
  return Object.entries(CONNECTION_TYPES)
    .filter(([_, config]) => config.category === 'database')
    .map(([type, config]) => ({
      value: type,
      label: config.label,
      defaultPort: config.defaultPort
    }))
}

/**
 * 获取连接类型的默认配置
 * @param {string} type - 连接类型
 * @returns {Object} 默认配置
 */
export function getDefaultConfig(type) {
  const config = getConnectionTypeConfig(type)
  return config ? { ...config.defaultConfig } : {}
}
