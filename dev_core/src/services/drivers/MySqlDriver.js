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
   * 获取表结构信息
   */
  async getTableStructure(tableName) {
    const connection = await this.createConnection();
    try {
      const database = this.config.database;
      
      // 获取字段信息
      const [columns] = await connection.query(
        `SELECT 
          COLUMN_NAME as name,
          COLUMN_TYPE as type,
          IS_NULLABLE as nullable,
          COLUMN_DEFAULT as defaultValue,
          COLUMN_KEY as \`key\`,
          EXTRA as extra,
          COLUMN_COMMENT as comment,
          CHARACTER_MAXIMUM_LENGTH as maxLength,
          NUMERIC_PRECISION as numericPrecision,
          NUMERIC_SCALE as numericScale
        FROM INFORMATION_SCHEMA.COLUMNS
        WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
        ORDER BY ORDINAL_POSITION`,
        [database, tableName]
      );

      // 获取索引信息
      const [indexes] = await connection.query(
        `SELECT 
          INDEX_NAME as name,
          COLUMN_NAME as columnName,
          NON_UNIQUE as nonUnique,
          INDEX_TYPE as type,
          SEQ_IN_INDEX as sequence
        FROM INFORMATION_SCHEMA.STATISTICS
        WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
        ORDER BY INDEX_NAME, SEQ_IN_INDEX`,
        [database, tableName]
      );

      // 获取外键信息
      const [foreignKeys] = await connection.query(
        `SELECT 
          CONSTRAINT_NAME as name,
          COLUMN_NAME as columnName,
          REFERENCED_TABLE_NAME as referencedTable,
          REFERENCED_COLUMN_NAME as referencedColumn
        FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
        WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND REFERENCED_TABLE_NAME IS NOT NULL`,
        [database, tableName]
      );

      // 获取外键的更新和删除规则
      const [foreignKeyRules] = await connection.query(
        `SELECT 
          CONSTRAINT_NAME as name,
          UPDATE_RULE as updateRule,
          DELETE_RULE as deleteRule
        FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS
        WHERE CONSTRAINT_SCHEMA = ? AND TABLE_NAME = ?`,
        [database, tableName]
      );

      // 处理字段信息
      const formattedColumns = columns.map(col => ({
        name: col.name,
        type: col.type,
        nullable: col.nullable === 'YES',
        defaultValue: col.defaultValue,
        isPrimary: col.key === 'PRI',
        isUnique: col.key === 'UNI',
        autoIncrement: col.extra.includes('auto_increment'),
        comment: col.comment || '',
        maxLength: col.maxLength,
        numericPrecision: col.numericPrecision,
        numericScale: col.numericScale
      }));

      // 处理索引信息（按索引名分组）
      const indexMap = new Map();
      indexes.forEach(idx => {
        if (!indexMap.has(idx.name)) {
          indexMap.set(idx.name, {
            name: idx.name,
            type: idx.name === 'PRIMARY' ? 'PRIMARY' : (idx.nonUnique === 0 ? 'UNIQUE' : 'INDEX'),
            method: idx.type,
            columns: []
          });
        }
        indexMap.get(idx.name).columns.push(idx.columnName);
      });
      const formattedIndexes = Array.from(indexMap.values());

      // 处理外键信息（合并规则）
      const foreignKeyMap = new Map();
      foreignKeys.forEach(fk => {
        if (!foreignKeyMap.has(fk.name)) {
          foreignKeyMap.set(fk.name, {
            name: fk.name,
            columnName: fk.columnName,
            referencedTable: fk.referencedTable,
            referencedColumn: fk.referencedColumn,
            updateRule: 'NO ACTION',
            deleteRule: 'NO ACTION'
          });
        }
      });
      
      foreignKeyRules.forEach(rule => {
        if (foreignKeyMap.has(rule.name)) {
          const fk = foreignKeyMap.get(rule.name);
          fk.updateRule = rule.updateRule;
          fk.deleteRule = rule.deleteRule;
        }
      });
      
      const formattedForeignKeys = Array.from(foreignKeyMap.values());

      return {
        columns: formattedColumns,
        indexes: formattedIndexes,
        foreignKeys: formattedForeignKeys
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

