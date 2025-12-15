/**
 * SQL 自动补全配置
 * 为 Monaco Editor 提供 SQL 关键字、函数、表名、字段名的自动补全
 */

// MySQL 关键字列表
export const MYSQL_KEYWORDS = [
  'SELECT', 'FROM', 'WHERE', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER',
  'TABLE', 'DATABASE', 'INDEX', 'VIEW', 'PROCEDURE', 'FUNCTION', 'TRIGGER',
  'JOIN', 'INNER', 'LEFT', 'RIGHT', 'OUTER', 'CROSS', 'FULL',
  'ON', 'USING', 'AS', 'AND', 'OR', 'NOT', 'IN', 'EXISTS', 'BETWEEN', 'LIKE',
  'IS', 'NULL', 'TRUE', 'FALSE', 'DISTINCT', 'ALL', 'ANY', 'SOME',
  'ORDER', 'BY', 'GROUP', 'HAVING', 'LIMIT', 'OFFSET',
  'ASC', 'DESC', 'UNION', 'INTERSECT', 'EXCEPT',
  'CASE', 'WHEN', 'THEN', 'ELSE', 'END',
  'IF', 'IFNULL', 'NULLIF', 'COALESCE',
  'PRIMARY', 'KEY', 'FOREIGN', 'REFERENCES', 'UNIQUE', 'CHECK', 'DEFAULT',
  'AUTO_INCREMENT', 'NOT NULL', 'UNSIGNED', 'ZEROFILL',
  'INT', 'INTEGER', 'BIGINT', 'SMALLINT', 'TINYINT', 'MEDIUMINT',
  'DECIMAL', 'NUMERIC', 'FLOAT', 'DOUBLE', 'REAL',
  'CHAR', 'VARCHAR', 'TEXT', 'TINYTEXT', 'MEDIUMTEXT', 'LONGTEXT',
  'BINARY', 'VARBINARY', 'BLOB', 'TINYBLOB', 'MEDIUMBLOB', 'LONGBLOB',
  'DATE', 'TIME', 'DATETIME', 'TIMESTAMP', 'YEAR',
  'ENUM', 'SET', 'JSON',
  'BEGIN', 'COMMIT', 'ROLLBACK', 'SAVEPOINT',
  'GRANT', 'REVOKE', 'PRIVILEGES', 'TO', 'WITH', 'OPTION',
  'SHOW', 'DESCRIBE', 'EXPLAIN', 'USE'
]

// MySQL 常用函数列表
export const MYSQL_FUNCTIONS = [
  // 字符串函数
  'CONCAT', 'CONCAT_WS', 'SUBSTRING', 'SUBSTR', 'LEFT', 'RIGHT', 'LENGTH', 'CHAR_LENGTH',
  'UPPER', 'LOWER', 'TRIM', 'LTRIM', 'RTRIM', 'REPLACE', 'REVERSE', 'REPEAT',
  'LPAD', 'RPAD', 'LOCATE', 'POSITION', 'INSTR', 'STRCMP',
  
  // 数值函数
  'ABS', 'CEIL', 'CEILING', 'FLOOR', 'ROUND', 'TRUNCATE', 'MOD', 'POWER', 'POW',
  'SQRT', 'EXP', 'LN', 'LOG', 'LOG10', 'LOG2',
  'RAND', 'SIGN', 'PI', 'SIN', 'COS', 'TAN', 'ASIN', 'ACOS', 'ATAN',
  
  // 日期时间函数
  'NOW', 'CURDATE', 'CURTIME', 'CURRENT_DATE', 'CURRENT_TIME', 'CURRENT_TIMESTAMP',
  'DATE', 'TIME', 'YEAR', 'MONTH', 'DAY', 'HOUR', 'MINUTE', 'SECOND',
  'DATE_ADD', 'DATE_SUB', 'DATEDIFF', 'TIMEDIFF', 'TIMESTAMPDIFF',
  'DATE_FORMAT', 'STR_TO_DATE', 'UNIX_TIMESTAMP', 'FROM_UNIXTIME',
  'LAST_DAY', 'DAYOFWEEK', 'DAYOFMONTH', 'DAYOFYEAR', 'WEEK', 'WEEKDAY',
  
  // 聚合函数
  'COUNT', 'SUM', 'AVG', 'MAX', 'MIN', 'GROUP_CONCAT',
  'STD', 'STDDEV', 'VARIANCE', 'VAR_POP', 'VAR_SAMP',
  
  // 条件函数
  'IF', 'IFNULL', 'NULLIF', 'COALESCE', 'CASE',
  
  // 类型转换函数
  'CAST', 'CONVERT', 'BINARY',
  
  // JSON 函数
  'JSON_EXTRACT', 'JSON_OBJECT', 'JSON_ARRAY', 'JSON_CONTAINS', 'JSON_KEYS',
  'JSON_LENGTH', 'JSON_VALID', 'JSON_TYPE', 'JSON_SET', 'JSON_INSERT',
  
  // 其他函数
  'DATABASE', 'USER', 'VERSION', 'CONNECTION_ID', 'LAST_INSERT_ID',
  'MD5', 'SHA1', 'SHA2', 'PASSWORD', 'ENCRYPT'
]

// PostgreSQL 关键字列表
export const POSTGRES_KEYWORDS = [
  'SELECT', 'FROM', 'WHERE', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER',
  'TABLE', 'DATABASE', 'SCHEMA', 'INDEX', 'VIEW', 'PROCEDURE', 'FUNCTION', 'TRIGGER',
  'JOIN', 'INNER', 'LEFT', 'RIGHT', 'OUTER', 'CROSS', 'FULL',
  'ON', 'USING', 'AS', 'AND', 'OR', 'NOT', 'IN', 'EXISTS', 'BETWEEN', 'LIKE', 'ILIKE',
  'IS', 'NULL', 'TRUE', 'FALSE', 'DISTINCT', 'ALL', 'ANY', 'SOME',
  'ORDER', 'BY', 'GROUP', 'HAVING', 'LIMIT', 'OFFSET', 'FETCH',
  'ASC', 'DESC', 'UNION', 'INTERSECT', 'EXCEPT',
  'CASE', 'WHEN', 'THEN', 'ELSE', 'END',
  'NULLIF', 'COALESCE',
  'PRIMARY', 'KEY', 'FOREIGN', 'REFERENCES', 'UNIQUE', 'CHECK', 'DEFAULT',
  'SERIAL', 'BIGSERIAL', 'SMALLSERIAL', 'NOT NULL',
  'INT', 'INTEGER', 'BIGINT', 'SMALLINT', 'INT2', 'INT4', 'INT8',
  'DECIMAL', 'NUMERIC', 'REAL', 'DOUBLE PRECISION', 'FLOAT', 'FLOAT4', 'FLOAT8',
  'CHAR', 'CHARACTER', 'VARCHAR', 'CHARACTER VARYING', 'TEXT',
  'BYTEA', 'BOOLEAN', 'BOOL',
  'DATE', 'TIME', 'TIMESTAMP', 'TIMESTAMPTZ', 'INTERVAL',
  'UUID', 'JSON', 'JSONB', 'XML', 'ARRAY',
  'BEGIN', 'COMMIT', 'ROLLBACK', 'SAVEPOINT',
  'GRANT', 'REVOKE', 'PRIVILEGES', 'TO', 'WITH', 'OPTION',
  'EXPLAIN', 'ANALYZE', 'VACUUM', 'RETURNING'
]

// PostgreSQL 常用函数列表
export const POSTGRES_FUNCTIONS = [
  // 字符串函数
  'CONCAT', 'CONCAT_WS', 'SUBSTRING', 'SUBSTR', 'LEFT', 'RIGHT', 'LENGTH', 'CHAR_LENGTH',
  'UPPER', 'LOWER', 'INITCAP', 'TRIM', 'LTRIM', 'RTRIM', 'REPLACE', 'REVERSE', 'REPEAT',
  'LPAD', 'RPAD', 'POSITION', 'STRPOS', 'SPLIT_PART', 'REGEXP_REPLACE', 'REGEXP_MATCH',
  
  // 数值函数
  'ABS', 'CEIL', 'CEILING', 'FLOOR', 'ROUND', 'TRUNC', 'MOD', 'POWER', 'POW',
  'SQRT', 'EXP', 'LN', 'LOG', 'LOG10',
  'RANDOM', 'SIGN', 'PI', 'SIN', 'COS', 'TAN', 'ASIN', 'ACOS', 'ATAN', 'ATAN2',
  
  // 日期时间函数
  'NOW', 'CURRENT_DATE', 'CURRENT_TIME', 'CURRENT_TIMESTAMP', 'LOCALTIME', 'LOCALTIMESTAMP',
  'DATE_PART', 'DATE_TRUNC', 'EXTRACT', 'AGE', 'TO_CHAR', 'TO_DATE', 'TO_TIMESTAMP',
  'MAKE_DATE', 'MAKE_TIME', 'MAKE_TIMESTAMP', 'MAKE_INTERVAL',
  
  // 聚合函数
  'COUNT', 'SUM', 'AVG', 'MAX', 'MIN', 'STRING_AGG', 'ARRAY_AGG', 'JSON_AGG', 'JSONB_AGG',
  'STDDEV', 'STDDEV_POP', 'STDDEV_SAMP', 'VARIANCE', 'VAR_POP', 'VAR_SAMP',
  
  // 条件函数
  'NULLIF', 'COALESCE', 'GREATEST', 'LEAST', 'CASE',
  
  // 类型转换函数
  'CAST', 'TO_CHAR', 'TO_NUMBER', 'TO_DATE', 'TO_TIMESTAMP',
  
  // JSON/JSONB 函数
  'JSON_BUILD_OBJECT', 'JSON_BUILD_ARRAY', 'JSON_OBJECT', 'JSON_ARRAY',
  'JSONB_BUILD_OBJECT', 'JSONB_BUILD_ARRAY', 'JSONB_SET', 'JSONB_INSERT',
  'JSON_EXTRACT_PATH', 'JSONB_EXTRACT_PATH', 'JSON_ARRAY_ELEMENTS', 'JSONB_ARRAY_ELEMENTS',
  
  // 数组函数
  'ARRAY_LENGTH', 'ARRAY_APPEND', 'ARRAY_PREPEND', 'ARRAY_CAT', 'UNNEST',
  
  // 其他函数
  'CURRENT_DATABASE', 'CURRENT_SCHEMA', 'CURRENT_USER', 'SESSION_USER', 'VERSION',
  'PG_BACKEND_PID', 'GEN_RANDOM_UUID', 'MD5', 'ENCODE', 'DECODE'
]

// SQL Server 关键字列表
export const SQLSERVER_KEYWORDS = [
  'SELECT', 'FROM', 'WHERE', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'ALTER',
  'TABLE', 'DATABASE', 'SCHEMA', 'INDEX', 'VIEW', 'PROCEDURE', 'FUNCTION', 'TRIGGER',
  'JOIN', 'INNER', 'LEFT', 'RIGHT', 'OUTER', 'CROSS', 'FULL',
  'ON', 'AS', 'AND', 'OR', 'NOT', 'IN', 'EXISTS', 'BETWEEN', 'LIKE',
  'IS', 'NULL', 'DISTINCT', 'ALL', 'ANY', 'SOME',
  'ORDER', 'BY', 'GROUP', 'HAVING', 'TOP', 'OFFSET', 'FETCH',
  'ASC', 'DESC', 'UNION', 'INTERSECT', 'EXCEPT',
  'CASE', 'WHEN', 'THEN', 'ELSE', 'END',
  'ISNULL', 'NULLIF', 'COALESCE',
  'PRIMARY', 'KEY', 'FOREIGN', 'REFERENCES', 'UNIQUE', 'CHECK', 'DEFAULT',
  'IDENTITY', 'NOT NULL',
  'INT', 'INTEGER', 'BIGINT', 'SMALLINT', 'TINYINT',
  'DECIMAL', 'NUMERIC', 'FLOAT', 'REAL', 'MONEY', 'SMALLMONEY',
  'CHAR', 'NCHAR', 'VARCHAR', 'NVARCHAR', 'TEXT', 'NTEXT',
  'BINARY', 'VARBINARY', 'IMAGE',
  'BIT', 'DATETIME', 'DATETIME2', 'SMALLDATETIME', 'DATE', 'TIME', 'DATETIMEOFFSET',
  'UNIQUEIDENTIFIER', 'XML', 'SQL_VARIANT',
  'BEGIN', 'COMMIT', 'ROLLBACK', 'TRANSACTION', 'SAVE',
  'GRANT', 'REVOKE', 'DENY', 'TO', 'WITH', 'OPTION',
  'GO', 'USE', 'EXEC', 'EXECUTE', 'PRINT', 'DECLARE', 'SET'
]

// SQL Server 常用函数列表
export const SQLSERVER_FUNCTIONS = [
  // 字符串函数
  'CONCAT', 'CONCAT_WS', 'SUBSTRING', 'LEFT', 'RIGHT', 'LEN', 'DATALENGTH',
  'UPPER', 'LOWER', 'TRIM', 'LTRIM', 'RTRIM', 'REPLACE', 'REVERSE', 'REPLICATE',
  'CHARINDEX', 'PATINDEX', 'STUFF', 'FORMAT', 'STRING_AGG',
  
  // 数值函数
  'ABS', 'CEILING', 'FLOOR', 'ROUND', 'POWER', 'SQRT', 'EXP', 'LOG', 'LOG10',
  'RAND', 'SIGN', 'PI', 'SIN', 'COS', 'TAN', 'ASIN', 'ACOS', 'ATAN', 'ATN2',
  
  // 日期时间函数
  'GETDATE', 'GETUTCDATE', 'SYSDATETIME', 'SYSUTCDATETIME', 'CURRENT_TIMESTAMP',
  'DATEPART', 'DATENAME', 'DATEADD', 'DATEDIFF', 'DATEDIFF_BIG',
  'YEAR', 'MONTH', 'DAY', 'EOMONTH', 'DATEFROMPARTS', 'TIMEFROMPARTS',
  'FORMAT', 'CONVERT', 'CAST',
  
  // 聚合函数
  'COUNT', 'SUM', 'AVG', 'MAX', 'MIN', 'STRING_AGG',
  'STDEV', 'STDEVP', 'VAR', 'VARP',
  
  // 条件函数
  'IIF', 'ISNULL', 'NULLIF', 'COALESCE', 'CASE', 'CHOOSE',
  
  // 类型转换函数
  'CAST', 'CONVERT', 'TRY_CAST', 'TRY_CONVERT', 'PARSE', 'TRY_PARSE',
  
  // JSON 函数
  'ISJSON', 'JSON_VALUE', 'JSON_QUERY', 'JSON_MODIFY',
  'OPENJSON', 'FOR JSON',
  
  // 窗口函数
  'ROW_NUMBER', 'RANK', 'DENSE_RANK', 'NTILE', 'LAG', 'LEAD',
  'FIRST_VALUE', 'LAST_VALUE',
  
  // 其他函数
  'NEWID', 'NEWSEQUENTIALID', 'DB_NAME', 'SCHEMA_NAME', 'OBJECT_NAME',
  'USER_NAME', 'SUSER_NAME', 'HOST_NAME', 'APP_NAME',
  'HASHBYTES', 'CHECKSUM', 'BINARY_CHECKSUM'
]

/**
 * 创建补全提供器
 * @param {Object} monaco - Monaco Editor 实例
 * @param {Array} keywords - 关键字列表
 * @param {Array} functions - 函数列表
 */
function createCompletionProvider(monaco, keywords, functions) {
  return {
    provideCompletionItems: (model, position) => {
      const word = model.getWordUntilPosition(position)
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn
      }

      const suggestions = []

      // SQL 关键字补全
      keywords.forEach(keyword => {
        suggestions.push({
          label: keyword,
          kind: monaco.languages.CompletionItemKind.Keyword,
          detail: 'SQL 关键字',
          insertText: keyword,
          range: range
        })
      })

      // SQL 函数补全
      functions.forEach(func => {
        suggestions.push({
          label: func,
          kind: monaco.languages.CompletionItemKind.Function,
          detail: 'SQL 函数',
          insertText: `${func}()`,
          insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          range: range
        })
      })

      return { suggestions }
    }
  }
}

/**
 * 注册 SQL 自动补全提供器
 * @param {Object} monaco - Monaco Editor 实例
 */
export function registerSqlCompletionProvider(monaco) {
  // 注销之前的提供器（如果存在）
  if (window.__sqlCompletionDisposables) {
    window.__sqlCompletionDisposables.forEach(d => d.dispose())
  }

  // 为不同数据库注册各自的补全提供器
  const disposables = [
    monaco.languages.registerCompletionItemProvider(
      'mysql',
      createCompletionProvider(monaco, MYSQL_KEYWORDS, MYSQL_FUNCTIONS)
    ),
    monaco.languages.registerCompletionItemProvider(
      'pgsql',
      createCompletionProvider(monaco, POSTGRES_KEYWORDS, POSTGRES_FUNCTIONS)
    ),
    monaco.languages.registerCompletionItemProvider(
      'sql',
      createCompletionProvider(monaco, SQLSERVER_KEYWORDS, SQLSERVER_FUNCTIONS)
    )
  ]

  // 保存 disposables 以便后续注销
  window.__sqlCompletionDisposables = disposables
  
  return disposables
}

/**
 * 注销 SQL 自动补全提供器
 */
export function unregisterSqlCompletionProvider() {
  if (window.__sqlCompletionDisposables) {
    window.__sqlCompletionDisposables.forEach(d => d.dispose())
    window.__sqlCompletionDisposables = null
  }
}
