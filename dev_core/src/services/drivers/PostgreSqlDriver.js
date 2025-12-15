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
    const config = {
      host: this.config.host,
      port: Number(this.config.port),
      user: this.config.username,
      password: this.config.password,
      database: this.config.database,
      // 连接超时（默认 3 秒）
      connectionTimeoutMillis: this.config.connectionTimeout || 3000,
      // 查询超时
      query_timeout: this.config.queryTimeout || 30000,
      // 连接池配置
      max: this.config.maxConnections || 10,
      min: 1,
      idleTimeoutMillis: 30000,
    };

    // SSL 配置
    // 优先从 sslConfig JSON 字段读取，否则从直接配置读取（兼容测试连接）
    let sslMode, sslCa, sslCert, sslKey;
    
    if (this.config.sslConfig) {
      // 从数据库保存的 sslConfig JSON 字段读取
      sslMode = this.config.sslConfig.mode;
      sslCa = this.config.sslConfig.ca;
      sslCert = this.config.sslConfig.cert;
      sslKey = this.config.sslConfig.key;
    } else {
      // 从直接配置读取（用于测试连接）
      sslMode = this.config.sslMode;
      sslCa = this.config.sslCa;
      sslCert = this.config.sslCert;
      sslKey = this.config.sslKey;
    }

    // SSL 模式配置
    // disable: 不使用 SSL
    // prefer: 优先 SSL，但如果服务器不支持则使用非 SSL（不设置 ssl 属性，让 pg 自动处理）
    // require: 必须使用 SSL，但不验证证书
    // verify-ca: 必须使用 SSL 并验证证书
    if (sslMode === 'require') {
      config.ssl = { rejectUnauthorized: false };
    } else if (sslMode === 'verify-ca') {
      config.ssl = { 
        rejectUnauthorized: true, 
        ca: sslCa || undefined,
        cert: sslCert || undefined,
        key: sslKey || undefined,
      };
      // 移除 undefined 的属性
      config.ssl = Object.fromEntries(
        Object.entries(config.ssl).filter(([_, v]) => v !== undefined)
      );
    }
    // prefer 和 disable 模式不设置 ssl 属性，让 pg 库自动处理

    // Schema 配置（通过 search_path）
    if (this.config.schema && this.config.schema !== 'public') {
      config.options = `-c search_path=${this.config.schema}`;
    }

    return config;
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
      const schema = this.config.schema || 'public';
      const result = await client.query(`
        SELECT 
          t.tablename as name,
          obj_description(c.oid) as comment,
          c.reltuples::bigint as rows
        FROM pg_tables t
        JOIN pg_class c ON c.relname = t.tablename AND c.relnamespace = (
          SELECT oid FROM pg_namespace WHERE nspname = t.schemaname
        )
        WHERE t.schemaname = $1
        ORDER BY t.tablename
      `, [schema]);
      
      return result.rows.map(item => ({
        name: item.name,
        comment: item.comment || '',
        rows: parseInt(item.rows) || 0
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
   * 获取表结构信息
   */
  async getTableStructure(tableName) {
    const pool = await this.createConnection();
    const client = await pool.connect();
    try {
      const schema = this.config.schema || 'public';
      
      // 获取字段信息
      const columnsResult = await client.query(
        `SELECT 
          column_name as name,
          data_type as type,
          is_nullable as nullable,
          column_default as "defaultValue",
          character_maximum_length as "maxLength",
          numeric_precision as "numericPrecision",
          numeric_scale as "numericScale",
          col_description((table_schema||'.'||table_name)::regclass::oid, ordinal_position) as comment
        FROM information_schema.columns
        WHERE table_schema = $1 AND table_name = $2
        ORDER BY ordinal_position`,
        [schema, tableName]
      );

      // 获取主键信息
      const primaryKeysResult = await client.query(
        `SELECT a.attname as column_name
        FROM pg_index i
        JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
        WHERE i.indrelid = $1::regclass AND i.indisprimary`,
        [tableName]
      );
      const primaryKeys = new Set(primaryKeysResult.rows.map(r => r.column_name));

      // 获取唯一键信息
      const uniqueKeysResult = await client.query(
        `SELECT a.attname as column_name
        FROM pg_index i
        JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
        WHERE i.indrelid = $1::regclass AND i.indisunique AND NOT i.indisprimary`,
        [tableName]
      );
      const uniqueKeys = new Set(uniqueKeysResult.rows.map(r => r.column_name));

      // 获取索引信息
      const indexesResult = await client.query(
        `SELECT 
          i.relname as name,
          a.attname as column_name,
          ix.indisunique as is_unique,
          ix.indisprimary as is_primary,
          am.amname as method,
          array_position(ix.indkey, a.attnum) as seq
        FROM pg_class t
        JOIN pg_index ix ON t.oid = ix.indrelid
        JOIN pg_class i ON i.oid = ix.indexrelid
        JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
        JOIN pg_am am ON i.relam = am.oid
        WHERE t.relname = $1 AND t.relkind = 'r'
        ORDER BY i.relname, seq`,
        [tableName]
      );

      // 获取外键信息
      const foreignKeysResult = await client.query(
        `SELECT 
          tc.constraint_name as name,
          kcu.column_name as column_name,
          ccu.table_name as referenced_table,
          ccu.column_name as referenced_column,
          rc.update_rule as update_rule,
          rc.delete_rule as delete_rule
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu 
          ON tc.constraint_name = kcu.constraint_name
        JOIN information_schema.constraint_column_usage ccu 
          ON ccu.constraint_name = tc.constraint_name
        JOIN information_schema.referential_constraints rc
          ON rc.constraint_name = tc.constraint_name
        WHERE tc.constraint_type = 'FOREIGN KEY' 
          AND tc.table_schema = $1
          AND tc.table_name = $2`,
        [schema, tableName]
      );

      // 处理字段信息
      const formattedColumns = columnsResult.rows.map(col => ({
        name: col.name,
        type: col.type,
        nullable: col.nullable === 'YES',
        defaultValue: col.defaultValue,
        isPrimary: primaryKeys.has(col.name),
        isUnique: uniqueKeys.has(col.name),
        autoIncrement: col.defaultValue && col.defaultValue.includes('nextval'),
        comment: col.comment || '',
        maxLength: col.maxLength,
        numericPrecision: col.numericPrecision,
        numericScale: col.numericScale
      }));

      // 处理索引信息（按索引名分组）
      const indexMap = new Map();
      indexesResult.rows.forEach(idx => {
        if (!indexMap.has(idx.name)) {
          indexMap.set(idx.name, {
            name: idx.name,
            type: idx.is_primary ? 'PRIMARY' : (idx.is_unique ? 'UNIQUE' : 'INDEX'),
            method: idx.method.toUpperCase(),
            columns: []
          });
        }
        indexMap.get(idx.name).columns.push(idx.column_name);
      });
      const formattedIndexes = Array.from(indexMap.values());

      // 处理外键信息
      const formattedForeignKeys = foreignKeysResult.rows.map(fk => ({
        name: fk.name,
        columnName: fk.column_name,
        referencedTable: fk.referenced_table,
        referencedColumn: fk.referenced_column,
        updateRule: fk.update_rule,
        deleteRule: fk.delete_rule
      }));

      return {
        columns: formattedColumns,
        indexes: formattedIndexes,
        foreignKeys: formattedForeignKeys
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

