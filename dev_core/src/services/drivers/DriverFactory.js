const MySqlDriver = require('./MySqlDriver');
const PostgreSqlDriver = require('./PostgreSqlDriver');
const SqlServerDriver = require('./SqlServerDriver');

/**
 * 数据库驱动工厂
 * 根据数据库类型创建对应的驱动实例
 */
class DriverFactory {
  /**
   * 创建数据库驱动
   * @param {string} dbType - 数据库类型 (mysql, postgresql, sqlserver)
   * @param {Object} config - 数据库配置
   * @returns {BaseDriver} 数据库驱动实例
   */
  static createDriver(dbType, config) {
    switch (dbType.toLowerCase()) {
      case 'mysql':
        return new MySqlDriver(config);
      case 'postgresql':
      case 'postgres':
        return new PostgreSqlDriver(config);
      case 'sqlserver':
      case 'mssql':
        return new SqlServerDriver(config);
      default:
        throw new Error(`不支持的数据库类型: ${dbType}`);
    }
  }

  /**
   * 获取支持的数据库类型列表
   * @returns {Array<string>} 支持的数据库类型
   */
  static getSupportedTypes() {
    return ['mysql', 'postgresql', 'sqlserver'];
  }
}

module.exports = DriverFactory;

