const { DataConnection, DataRelationalConfig } = require('../models');
const DriverFactory = require('./drivers/DriverFactory');
const AppError = require('../utils/AppError');
const ErrorCodes = require('../constants/errorCodes');

/**
 * 数据连接服务
 * 处理数据连接的创建、测试、查询等业务逻辑
 */
class DataConnectionService {
  /**
   * 获取数据连接列表
   * @param {string} projectId - 工程ID
   * @param {Object} options - 查询选项 { page, limit, type, status }
   * @returns {Promise<Object>} 连接列表和分页信息
   */
  async getConnections(projectId, options = {}) {
    const { page = 1, limit = 20, type, status } = options;
    const offset = (page - 1) * limit;
    const where = { projectId };

    if (type) where.type = type;
    if (status) where.status = status;

    const { count, rows } = await DataConnection.findAndCountAll({
      where,
      include: [
        {
          model: DataRelationalConfig,
          as: 'relationalConfig',
          required: false
        }
      ],
      limit: parseInt(limit),
      offset,
      order: [['createdAt', 'DESC']]
    });

    return {
      connections: rows,
      pagination: {
        page: parseInt(page),
        limit: parseInt(limit),
        total: count,
        totalPages: Math.ceil(count / limit)
      }
    };
  }

  /**
   * 创建数据连接
   * @param {string} projectId - 工程ID
   * @param {Object} data - 连接数据 { name, type, category, config }
   * @param {string} userId - 用户ID
   * @returns {Promise<Object>} 创建的连接对象
   */
  async createConnection(projectId, data, userId) {
    const { name, type, category, config } = data;

    // 创建数据连接
    const connection = await DataConnection.create({
      projectId,
      name,
      type,
      category: category || 'external',
      status: 'active',
      createdBy: userId
    });

    // 如果是关系库类型，创建对应的配置
    if (type === 'relational' && config) {
      await DataRelationalConfig.create({
        connectionId: connection.id,
        ...config
      });
    }

    // 重新获取完整数据
    const fullConnection = await DataConnection.findByPk(connection.id, {
      include: [
        {
          model: DataRelationalConfig,
          as: 'relationalConfig',
          required: false
        }
      ]
    });

    return fullConnection;
  }

  /**
   * 测试数据连接
   * @param {string} type - 连接类型
   * @param {Object} config - 连接配置
   * @returns {Promise<boolean>} 连接是否成功
   */
  async testConnection(type, config) {
    if (type === 'relational') {
      return await this.testRelationalConnection(config);
    }
    // 其他类型的连接测试（MQTT、WebSocket 等）可以在这里扩展
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
      message: `暂不支持测试 ${type} 类型的连接`
    });
  }

  /**
   * 测试关系型数据库连接
   * @param {Object} config - 数据库配置
   * @returns {Promise<boolean>} 连接是否成功
   */
  async testRelationalConnection(config) {
    const { dbType = 'mysql', host, port, username, password, database } = config;

    if (!host || !port || !username || !database) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '请填写完整的数据库连接信息'
      });
    }

    try {
      const driver = DriverFactory.createDriver(dbType, {
        host,
        port,
        username,
        password,
        database
      });
      
      return await driver.testConnection();
    } catch (error) {
      throw new AppError(ErrorCodes.DATABASE_ERROR, {
        message: '数据库连接失败，请检查配置信息',
        error: error.message
      });
    }
  }

  /**
   * 获取关系数据库的表列表
   * @param {string} projectId - 工程ID
   * @param {string} connectionId - 连接ID
   * @returns {Promise<Array>} 表列表
   */
  async getTables(projectId, connectionId) {
    // 获取连接配置
    const connection = await DataConnection.findOne({
      where: { id: connectionId, projectId, type: 'relational' },
      include: [
        {
          model: DataRelationalConfig,
          as: 'relationalConfig',
          required: true
        }
      ]
    });

    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: 'DataConnection',
        id: connectionId
      });
    }

    const relationalConfig = connection.relationalConfig;
    const driver = DriverFactory.createDriver(relationalConfig.dbType, {
      host: relationalConfig.host,
      port: relationalConfig.port,
      username: relationalConfig.username,
      password: relationalConfig.password,
      database: relationalConfig.database,
      charset: relationalConfig.charset,
      timeout: relationalConfig.queryTimeout
    });

    return await driver.getTables();
  }

  /**
   * 获取表数据
   * @param {string} projectId - 工程ID
   * @param {string} connectionId - 连接ID
   * @param {string} tableName - 表名
   * @param {Object} options - 查询选项 { page, limit }
   * @returns {Promise<Object>} 表数据和分页信息
   */
  async getTableData(projectId, connectionId, tableName, options = {}) {
    // 验证表名格式（防止 SQL 注入）
    if (!/^[A-Za-z0-9_]+$/.test(tableName)) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '不支持的表名格式'
      });
    }

    // 获取连接配置
    const connection = await DataConnection.findOne({
      where: { id: connectionId, projectId, type: 'relational' },
      include: [
        {
          model: DataRelationalConfig,
          as: 'relationalConfig',
          required: true
        }
      ]
    });

    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: 'DataConnection',
        id: connectionId
      });
    }

    const relationalConfig = connection.relationalConfig;
    const driver = DriverFactory.createDriver(relationalConfig.dbType, {
      host: relationalConfig.host,
      port: relationalConfig.port,
      username: relationalConfig.username,
      password: relationalConfig.password,
      database: relationalConfig.database,
      charset: relationalConfig.charset,
      timeout: relationalConfig.queryTimeout
    });

    return await driver.getTableData(tableName, options);
  }

  /**
   * 执行 SQL 查询
   * @param {string} projectId - 工程ID
   * @param {string} connectionId - 连接ID
   * @param {string} sql - SQL 语句
   * @param {Array} parameters - 参数数组
   * @returns {Promise<Object>} 查询结果
   */
  async executeSql(projectId, connectionId, sql, parameters = []) {
    if (!sql || typeof sql !== 'string' || !sql.trim()) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: 'SQL语句不能为空'
      });
    }

    if (!Array.isArray(parameters)) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '参数必须是数组格式'
      });
    }

    // 获取连接配置
    const connection = await DataConnection.findOne({
      where: { id: connectionId, projectId, type: 'relational' },
      include: [
        {
          model: DataRelationalConfig,
          as: 'relationalConfig',
          required: true
        }
      ]
    });

    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: 'DataConnection',
        id: connectionId
      });
    }

    const relationalConfig = connection.relationalConfig;
    const driver = DriverFactory.createDriver(relationalConfig.dbType, {
      host: relationalConfig.host,
      port: relationalConfig.port,
      username: relationalConfig.username,
      password: relationalConfig.password,
      database: relationalConfig.database,
      charset: relationalConfig.charset,
      timeout: relationalConfig.queryTimeout
    });

    const startTime = Date.now();
    try {
      const result = await driver.executeQuery(sql, parameters);
      const executionTime = Date.now() - startTime;
      
      return {
        data: result,
        executionTime
      };
    } catch (error) {
      throw new AppError(ErrorCodes.DATABASE_ERROR, {
        message: '执行SQL失败',
        error: error.message
      });
    }
  }
}

module.exports = new DataConnectionService();

