const { Pool } = require('pg');
const BaseDriver = require('./BaseDriver');

/**
 * PostgreSQL 数据库驱动
 */
class PostgreSqlDriver extends BaseDriver {
  constructor(config) {
    super(config);
    this.dbType = 'postgresql';
  }

  /**
   * 创建 PostgreSQL 连接配置
   */
  createConfig() {
    return {
      host: this.config.host,
      port: Number(this.config.port),
      user: this.config.username,
      password: this.config.password,
      database: this.config.database,
      connectionTimeoutMillis: this.config.timeout || 60000
    };
  }

  /**
   * 创建数据库连接池
   */
  async createConnection() {
    const pgConfig = this.createConfig();
    return new Pool(pgConfig);
  }

  /**
   * 测试数据库连接
   */
  async testConnection() {
    const pool = await this.createConnection();
    try {
      await pool.query('SELECT 1');
      return true;
    } catch (error) {
      throw error;
    } finally {
      await this.closeConnection(pool);
    }
  }

  /**
   * 执行 SQL 查询
   */
  async executeQuery(sql, parameters = []) {
    const pool = await this.createConnection();
    const client = await pool.connect();
    try {
      // PostgreSQL 使用 $1, $2, $3 作为参数占位符
      // 需要将 ? 占位符转换为 $1, $2, $3 格式
      const convertedSql = this.convertPlaceholders(sql, parameters.length);
      
      const result = await client.query(convertedSql, parameters);
      
      if (result.rows && result.rows.length > 0) {
        const columns = Object.keys(result.rows[0]);
        return {
          columns,
          rows: result.rows.map(row => columns.map(col => row[col])),
          rowCount: result.rowCount
        };
      } else if (result.command && ['INSERT', 'UPDATE', 'DELETE'].includes(result.command)) {
        return {
          columns: ['affectedRows', 'insertId', 'changedRows'],
          rows: [[result.rowCount || 0, result.insertId || null, result.rowCount || 0]],
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
      client.release();
      await this.closeConnection(pool);
    }
  }

  /**
   * 将 MySQL 风格的 ? 占位符转换为 PostgreSQL 的 $1, $2, $3 格式
   */
  convertPlaceholders(sql, paramCount) {
    let convertedSql = sql;
    for (let i = paramCount; i > 0; i--) {
      convertedSql = convertedSql.replace(/\?/, `$${i}`);
    }
    return convertedSql;
  }

  /**
   * 获取表列表
   */
  async getTables() {
    const pool = await this.createConnection();
    const client = await pool.connect();
    try {
      const result = await client.query(`
        SELECT 
          tablename as name,
          obj_description(c.oid) as comment,
          (SELECT COUNT(*) FROM information_schema.tables WHERE table_name = tablename) as rows
        FROM pg_tables t
        JOIN pg_class c ON c.relname = t.tablename
        WHERE schemaname = 'public'
        ORDER BY tablename
      `);
      
      return result.rows.map(item => ({
        name: item.name,
        comment: item.comment || '',
        rows: item.rows || 0
      }));
    } finally {
      client.release();
      await this.closeConnection(pool);
    }
  }

  /**
   * 获取表数据
   */
  async getTableData(tableName, options = {}) {
    const pool = await this.createConnection();
    const client = await pool.connect();
    try {
      const { page = 1, limit = 100 } = options;
      const numericLimit = Math.max(1, Math.min(500, parseInt(limit, 10) || 100));
      const numericPage = Math.max(1, parseInt(page, 10) || 1);
      const offset = (numericPage - 1) * numericLimit;
      const escapedTable = this.escapeIdentifier(tableName);

      // 获取总数
      const countResult = await client.query(`SELECT COUNT(*) AS total FROM ${escapedTable}`);
      const total = parseInt(countResult.rows[0]?.total || 0);

      // 获取分页数据
      const dataResult = await client.query(
        `SELECT * FROM ${escapedTable} LIMIT $1 OFFSET $2`,
        [numericLimit, offset]
      );
      
      const columns = dataResult.rows.length > 0 ? Object.keys(dataResult.rows[0]) : [];

      return {
        columns,
        rows: dataResult.rows,
        pagination: {
          page: numericPage,
          limit: numericLimit,
          total,
          totalPages: Math.ceil(total / numericLimit)
        }
      };
    } finally {
      client.release();
      await this.closeConnection(pool);
    }
  }

  /**
   * 关闭数据库连接
   */
  async closeConnection(pool) {
    if (pool && pool.end) {
      await pool.end().catch(() => {});
    }
  }

  /**
   * 转义标识符
   */
  escapeIdentifier(identifier) {
    return `"${identifier.replace(/"/g, '""')}"`;
  }
}

module.exports = PostgreSqlDriver;

