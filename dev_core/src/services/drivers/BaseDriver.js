/**
 * 数据库驱动基类
 * 所有数据库驱动都需要继承此类并实现相应方法
 */
class BaseDriver {
  constructor(config) {
    this.config = config;
  }

  /**
   * 创建数据库连接
   * @returns {Promise<Object>} 数据库连接对象
   */
  async createConnection() {
    throw new Error('createConnection method must be implemented');
  }

  /**
   * 测试数据库连接
   * @returns {Promise<boolean>} 连接是否成功
   */
  async testConnection() {
    throw new Error('testConnection method must be implemented');
  }

  /**
   * 执行 SQL 查询
   * @param {string} sql - SQL 语句
   * @param {Array} parameters - 参数数组
   * @returns {Promise<Object>} 查询结果
   */
  async executeQuery(sql, parameters = []) {
    throw new Error('executeQuery method must be implemented');
  }

  /**
   * 获取表列表
   * @returns {Promise<Array>} 表列表
   */
  async getTables() {
    throw new Error('getTables method must be implemented');
  }

  /**
   * 获取表数据
   * @param {string} tableName - 表名
   * @param {Object} options - 查询选项 { page, limit }
   * @returns {Promise<Object>} 表数据
   */
  async getTableData(tableName, options = {}) {
    throw new Error('getTableData method must be implemented');
  }

  /**
   * 关闭数据库连接
   * @param {Object} connection - 数据库连接对象
   */
  async closeConnection(connection) {
    throw new Error('closeConnection method must be implemented');
  }

  /**
   * 转义表名（防止 SQL 注入）
   * @param {string} identifier - 标识符
   * @returns {string} 转义后的标识符
   */
  escapeIdentifier(identifier) {
    throw new Error('escapeIdentifier method must be implemented');
  }
}

module.exports = BaseDriver;

