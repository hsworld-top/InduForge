const mysql = require('mysql2/promise');
const BaseDriver = require('./BaseDriver');

/**
 * MySQL 数据库驱动
 */
class MySqlDriver extends BaseDriver {
  constructor(config) {
    super(config);
    this.dbType = 'mysql';
  }

  /**
   * 创建 MySQL 连接配置
   */
  createConfig() {
    return {
      host: this.config.host,
      port: Number(this.config.port),
      user: this.config.username,
      password: this.config.password,
      database: this.config.database,
      charset: this.config.charset || 'utf8mb4',
      connectTimeout: this.config.timeout || 60000
    };
  }

  /**
   * 创建数据库连接
   */
  async createConnection() {
    const mysqlConfig = this.createConfig();
    return await mysql.createConnection(mysqlConfig);
  }

  /**
   * 测试数据库连接
   */
  async testConnection() {
    let connection;
    try {
      connection = await this.createConnection();
      await connection.ping();
      return true;
    } catch (error) {
      throw error;
    } finally {
      if (connection) {
        await this.closeConnection(connection);
      }
    }
  }

  /**
   * 执行 SQL 查询
   */
  async executeQuery(sql, parameters = []) {
    const connection = await this.createConnection();
    try {
      const [rows] = await connection.query(sql, parameters);
      
      // 处理结果
      if (Array.isArray(rows) && rows.length > 0) {
        const columns = Object.keys(rows[0]);
        return {
          columns,
          rows: rows.map(row => columns.map(col => row[col])),
          rowCount: rows.length
        };
      } else if (rows.affectedRows !== undefined) {
        // 对于 INSERT, UPDATE, DELETE 等操作
        return {
          columns: ['affectedRows', 'insertId', 'changedRows'],
          rows: [[rows.affectedRows, rows.insertId || null, rows.changedRows || 0]],
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
      await this.closeConnection(connection);
    }
  }

  /**
   * 获取表列表
   */
  async getTables() {
    const connection = await this.createConnection();
    try {
      const [tables] = await connection.query('SHOW TABLE STATUS');
      return tables.map(item => ({
        name: item.Name,
        comment: item.Comment || '',
        rows: item.Rows || 0
      }));
    } finally {
      await this.closeConnection(connection);
    }
  }

  /**
   * 获取表数据
   */
  async getTableData(tableName, options = {}) {
    const connection = await this.createConnection();
    try {
      const { page = 1, limit = 100 } = options;
      const numericLimit = Math.max(1, Math.min(500, parseInt(limit, 10) || 100));
      const numericPage = Math.max(1, parseInt(page, 10) || 1);
      const offset = (numericPage - 1) * numericLimit;
      const escapedTable = this.escapeIdentifier(tableName);

      // 获取总数
      const [countRows] = await connection.query(`SELECT COUNT(*) AS total FROM ${escapedTable}`);
      const total = countRows[0]?.total || 0;

      // 获取分页数据
      const [rows] = await connection.query(
        `SELECT * FROM ${escapedTable} LIMIT ? OFFSET ?`,
        [numericLimit, offset]
      );
      
      const columns = rows.length > 0 ? Object.keys(rows[0]) : [];

      return {
        columns,
        rows,
        pagination: {
          page: numericPage,
          limit: numericLimit,
          total,
          totalPages: Math.ceil(total / numericLimit)
        }
      };
    } finally {
      await this.closeConnection(connection);
    }
  }

  /**
   * 关闭数据库连接
   */
  async closeConnection(connection) {
    if (connection) {
      await connection.end().catch(() => {});
    }
  }

  /**
   * 转义标识符
   */
  escapeIdentifier(identifier) {
    return mysql.escapeId(identifier);
  }
}

module.exports = MySqlDriver;

