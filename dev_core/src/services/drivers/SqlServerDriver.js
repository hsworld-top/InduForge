const sql = require('mssql');
const BaseDriver = require('./BaseDriver');

/**
 * SQL Server 数据库驱动
 */
class SqlServerDriver extends BaseDriver {
  constructor(config) {
    super(config);
    this.dbType = 'sqlserver';
  }

  /**
   * 创建 SQL Server 连接配置
   */
  createConfig() {
    // SQL Server 2022+ 默认需要加密或信任证书
    const encrypt = this.config.encrypt !== undefined ? this.config.encrypt : false;
    const trustServerCertificate = this.config.trustServerCertificate !== undefined 
      ? this.config.trustServerCertificate 
      : true; // 默认信任证书，适合开发环境
    
    // 处理 localhost 和 127.0.0.1
    let server = this.config.host;
    if (server === '127.0.0.1') {
      server = 'localhost';
    }
    
    return {
      server: server,
      port: Number(this.config.port),
      user: this.config.username,
      password: this.config.password,
      database: this.config.database,
      options: {
        encrypt: encrypt,
        trustServerCertificate: trustServerCertificate,
        enableArithAbort: true,
        // 添加这些选项可能有助于连接
        instanceName: '', // 默认实例
        useUTC: false,
      },
      connectionTimeout: this.config.timeout || 60000,
      requestTimeout: this.config.requestTimeout || 60000,
      pool: {
        max: 10,
        min: 0,
        idleTimeoutMillis: 30000
      }
    };
  }

  /**
   * 创建数据库连接
   */
  async createConnection() {
    const sqlConfig = this.createConfig();
    return await sql.connect(sqlConfig);
  }

  /**
   * 测试数据库连接
   */
  async testConnection() {
    let pool;
    try {
      const config = this.createConfig();
      console.log('SQL Server 连接配置:', JSON.stringify(config, null, 2));
      pool = await this.createConnection();
      await pool.request().query('SELECT 1');
      return true;
    } catch (error) {
      console.error('SQL Server 连接错误:', error.message);
      throw error;
    } finally {
      if (pool) {
        await this.closeConnection(pool);
      }
    }
  }

  /**
   * 执行 SQL 查询
   */
  async executeQuery(querySql, parameters = []) {
    const pool = await this.createConnection();
    try {
      const request = pool.request();
      
      // 添加参数
      parameters.forEach((param, index) => {
        request.input(`param${index + 1}`, param);
      });

      // SQL Server 使用 @param1, @param2 作为参数占位符
      // 需要将 ? 占位符转换为 @param1, @param2 格式
      const convertedSql = this.convertPlaceholders(querySql, parameters.length);
      
      const result = await request.query(convertedSql);
      
      if (result.recordset && result.recordset.length > 0) {
        const columns = Object.keys(result.recordset[0]);
        return {
          columns,
          rows: result.recordset.map(row => columns.map(col => row[col])),
          rowCount: result.recordset.length
        };
      } else if (result.rowsAffected && result.rowsAffected[0] > 0) {
        return {
          columns: ['affectedRows', 'insertId', 'changedRows'],
          rows: [[result.rowsAffected[0], null, result.rowsAffected[0]]],
          rowCount: 1
        };
      } else {
        return {
          columns: [],
          rows: [],
          rowCount: 0
        };
      }
    } finally {
      await this.closeConnection(pool);
    }
  }

  /**
   * 将 MySQL 风格的 ? 占位符转换为 SQL Server 的 @param1, @param2 格式
   */
  convertPlaceholders(sql, paramCount) {
    let convertedSql = sql;
    for (let i = paramCount; i > 0; i--) {
      convertedSql = convertedSql.replace(/\?/, `@param${i}`);
    }
    return convertedSql;
  }

  /**
   * 获取表列表
   */
  async getTables() {
    const pool = await this.createConnection();
    try {
      const result = await pool.request().query(`
        SELECT 
          t.name AS name,
          ep.value AS comment,
          p.rows AS rows
        FROM sys.tables t
        INNER JOIN sys.partitions p ON t.object_id = p.object_id
        LEFT JOIN sys.extended_properties ep ON t.object_id = ep.major_id AND ep.minor_id = 0
        WHERE p.index_id IN (0, 1)
        GROUP BY t.name, ep.value, p.rows
        ORDER BY t.name
      `);
      
      return result.recordset.map(item => ({
        name: item.name,
        comment: item.comment || '',
        rows: item.rows || 0
      }));
    } finally {
      await this.closeConnection(pool);
    }
  }

  /**
   * 获取表数据
   */
  async getTableData(tableName, options = {}) {
    const pool = await this.createConnection();
    try {
      const { page = 1, limit = 100 } = options;
      const numericLimit = Math.max(1, Math.min(500, parseInt(limit, 10) || 100));
      const numericPage = Math.max(1, parseInt(page, 10) || 1);
      const offset = (numericPage - 1) * numericLimit;
      const escapedTable = this.escapeIdentifier(tableName);

      // 获取总数
      const countResult = await pool.request().query(`SELECT COUNT(*) AS total FROM ${escapedTable}`);
      const total = countResult.recordset[0]?.total || 0;

      // 获取分页数据（SQL Server 2012+ 支持 OFFSET FETCH）
      const dataResult = await pool.request().query(
        `SELECT * FROM ${escapedTable} ORDER BY (SELECT NULL) OFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY`,
        {
          offset: offset,
          limit: numericLimit
        }
      );
      
      const columns = dataResult.recordset.length > 0 ? Object.keys(dataResult.recordset[0]) : [];

      return {
        columns,
        rows: dataResult.recordset,
        pagination: {
          page: numericPage,
          limit: numericLimit,
          total,
          totalPages: Math.ceil(total / numericLimit)
        }
      };
    } finally {
      await this.closeConnection(pool);
    }
  }

  /**
   * 关闭数据库连接
   */
  async closeConnection(pool) {
    if (pool && pool.close) {
      await pool.close().catch(() => {});
    }
  }

  /**
   * 转义标识符
   */
  escapeIdentifier(identifier) {
    return `[${identifier.replace(/\]/g, ']]')}]`;
  }
}

module.exports = SqlServerDriver;

