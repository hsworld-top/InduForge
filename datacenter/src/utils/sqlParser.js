/**
 * SQL 解析工具
 * 使用 node-sql-parser 处理不同 SQL 方言
 */

import { Parser } from "node-sql-parser";

// 创建不同方言的解析器
const mysqlParser = new Parser();
const postgresParser = new Parser();
const mssqlParser = new Parser();

/**
 * 解析 SQL 语句
 * @param {string} sql - SQL 语句
 * @param {string} dialect - SQL 方言 ('mysql' | 'postgresql' | 'transactsql')
 * @returns {Object} 解析结果
 */
export function parseSql(sql, dialect = "mysql") {
  try {
    const parser = getParser(dialect);
    const ast = parser.astify(sql, { database: dialect });
    return {
      success: true,
      ast,
      type: Array.isArray(ast) ? ast[0]?.type : ast?.type,
      tables: extractTables(ast),
      columns: extractColumns(ast),
    };
  } catch (error) {
    return {
      success: false,
      error: error.message,
      position: error.location,
    };
  }
}

/**
 * 验证 SQL 语法
 * @param {string} sql - SQL 语句
 * @param {string} dialect - SQL 方言
 * @returns {Object} 验证结果
 */
export function validateSql(sql, dialect = "mysql") {
  const result = parseSql(sql, dialect);

  if (!result.success) {
    return {
      valid: false,
      errors: [result.error],
      position: result.position,
    };
  }

  // 检查危险操作
  const warnings = [];
  const dangerousTypes = ["drop", "truncate", "delete", "update"];

  if (dangerousTypes.includes(result.type?.toLowerCase())) {
    warnings.push(`警告：SQL 包含可能危险的操作 ${result.type.toUpperCase()}`);
  }

  return {
    valid: true,
    errors: [],
    warnings,
    type: result.type,
    tables: result.tables,
  };
}

/**
 * 提取 SQL 中的参数占位符
 * @param {string} sql - SQL 语句
 * @returns {Array} 参数列表
 */
export function extractParameters(sql) {
  // 匹配 ? 占位符
  const matches = sql.match(/\?/g);
  const count = matches ? matches.length : 0;

  return Array.from({ length: count }, (_, index) => ({
    index,
    name: `param${index + 1}`,
    value: "",
    type: "string",
  }));
}

/**
 * 提取 SQL 中的表名
 * @param {Object} ast - AST 对象
 * @returns {Array} 表名列表
 */
function extractTables(ast) {
  const tables = new Set();

  const traverse = (node) => {
    if (!node) return;

    if (node.type === "select" && node.from) {
      node.from.forEach((item) => {
        if (item.table) {
          tables.add(item.table);
        }
      });
    }

    // 递归遍历
    Object.values(node).forEach((value) => {
      if (typeof value === "object") {
        traverse(value);
      }
    });
  };

  if (Array.isArray(ast)) {
    ast.forEach(traverse);
  } else {
    traverse(ast);
  }

  return Array.from(tables);
}

/**
 * 提取 SQL 中的列名
 * @param {Object} ast - AST 对象
 * @returns {Array} 列名列表
 */
function extractColumns(ast) {
  const columns = new Set();

  const traverse = (node) => {
    if (!node) return;

    if (node.type === "select" && node.columns) {
      node.columns.forEach((col) => {
        if (col.expr && col.expr.column) {
          columns.add(col.expr.column);
        }
      });
    }

    // 递归遍历
    Object.values(node).forEach((value) => {
      if (typeof value === "object") {
        traverse(value);
      }
    });
  };

  if (Array.isArray(ast)) {
    ast.forEach(traverse);
  } else {
    traverse(ast);
  }

  return Array.from(columns);
}

/**
 * 获取对应方言的解析器
 * @param {string} dialect - SQL 方言
 * @returns {Parser} 解析器实例
 */
function getParser(dialect) {
  switch (dialect) {
    case "mysql":
      return mysqlParser;
    case "postgresql":
    case "pgsql":
      return postgresParser;
    case "transactsql":
    case "mssql":
    case "sqlserver":
      return mssqlParser;
    default:
      return mysqlParser;
  }
}

/**
 * 格式化 SQL（使用 AST 重新生成）
 * @param {string} sql - SQL 语句
 * @param {string} dialect - SQL 方言
 * @returns {string} 格式化后的 SQL
 */
export function formatSqlWithParser(sql, dialect = "mysql") {
  try {
    const parser = getParser(dialect);
    const ast = parser.astify(sql, { database: dialect });
    return parser.sqlify(ast, { database: dialect });
  } catch (error) {
    console.error("SQL 格式化失败:", error);
    return sql;
  }
}

/**
 * 获取 SQL 语句类型
 * @param {string} sql - SQL 语句
 * @param {string} dialect - SQL 方言
 * @returns {string} SQL 类型
 */
export function getSqlType(sql, dialect = "mysql") {
  const result = parseSql(sql, dialect);
  return result.success ? result.type : "unknown";
}
