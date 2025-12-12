const { DataQuery, DataQueryLog, DataConnection, DataRelationalConfig } = require('../models');
const DriverFactory = require('./drivers/DriverFactory');
const AppError = require('../utils/AppError');
const ErrorCodes = require('../constants/errorCodes');

/**
 * 数据查询服务
 * 处理数据查询的创建、执行等业务逻辑
 */
class DataQueryService {
  /**
   * 获取数据查询列表
   * @param {string} projectId - 工程ID
   * @param {Object} options - 查询选项 { page, limit, connectionId, queryType, isActive }
   * @returns {Promise<Object>} 查询列表和分页信息
   */
  async getQueries(projectId, options = {}) {
    const { page = 1, limit = 20, connectionId, queryType, isActive } = options;
    const offset = (page - 1) * limit;
    const where = { projectId };

    if (connectionId) where.connectionId = connectionId;
    if (queryType) where.queryType = queryType;
    if (isActive !== undefined) where.isActive = isActive === 'true';

    const { count, rows } = await DataQuery.findAndCountAll({
      where,
      include: [
        {
          model: DataConnection,
          as: 'connection',
          attributes: ['id', 'name', 'type']
        }
      ],
      limit: parseInt(limit),
      offset,
      order: [['createdAt', 'DESC']]
    });

    return {
      queries: rows,
      pagination: {
        page: parseInt(page),
        limit: parseInt(limit),
        total: count,
        totalPages: Math.ceil(count / limit)
      }
    };
  }

  /**
   * 创建数据查询
   * @param {string} projectId - 工程ID
   * @param {Object} data - 查询数据 { name, description, category, connectionId, queryType, config }
   * @param {string} userId - 用户ID
   * @returns {Promise<Object>} 创建的查询对象
   */
  async createQuery(projectId, data, userId) {
    const { name, description, category, connectionId, queryType, config } = data;

    // 验证连接ID是否属于此工程
    const connection = await DataConnection.findOne({
      where: { id: connectionId, projectId }
    });

    if (!connection) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '无效的连接ID'
      });
    }

    // 创建数据查询
    const query = await DataQuery.create({
      projectId,
      connectionId,
      name,
      description,
      category,
      queryType,
      config,
      createdBy: userId
    });

    // 重新获取完整数据
    const fullQuery = await DataQuery.findByPk(query.id, {
      include: [
        {
          model: DataConnection,
          as: 'connection',
          attributes: ['id', 'name', 'type']
        }
      ]
    });

    return fullQuery;
  }

  /**
   * 执行数据查询
   * @param {string} queryId - 查询ID
   * @param {Object} parameters - 查询参数对象 { paramName: value }
   * @param {string} userId - 用户ID
   * @returns {Promise<Object>} 查询结果
   */
  async executeQuery(queryId, parameters = {}, userId) {
    const query = await DataQuery.findByPk(queryId, {
      include: [
        {
          model: DataConnection,
          as: 'connection',
          include: [
            {
              model: DataRelationalConfig,
              as: 'relationalConfig',
              required: false
            }
          ]
        }
      ]
    });

    if (!query) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: 'DataQuery',
        id: queryId
      });
    }

    if (!query.isEnabled) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '查询已被禁用'
      });
    }

    const startTime = Date.now();
    let result = null;
    let errorMessage = null;
    let status = 'success';

    try {
      if (query.queryType === 'sql' && query.config) {
        result = await this.executeSqlQuery(query, parameters);
      } else {
        status = 'error';
        errorMessage = '暂不支持此类型的查询执行';
      }
    } catch (error) {
      status = 'error';
      errorMessage = error.message;
    }

    const executionTime = Date.now() - startTime;

    // 记录执行日志
    await DataQueryLog.create({
      queryId,
      connectionId: query.connectionId,
      executedBy: userId,
      parameters,
      executionTime,
      resultCount: result ? result.rowCount : null,
      status,
      errorMessage,
      executedAt: new Date()
    });

    if (status === 'error') {
      throw new AppError(ErrorCodes.DATABASE_ERROR, 500, {
        message: errorMessage
      });
    }

    return {
      data: result,
      executionTime
    };
  }

  /**
   * 执行 SQL 查询
   * @param {Object} query - 查询对象
   * @param {Object} parameters - 查询参数对象
   * @returns {Promise<Object>} 查询结果
   */
  async executeSqlQuery(query, parameters) {
    const sql = query.config.sql;
    const paramDefs = query.config.parameters || [];

    // 构建参数数组
    const replacements = paramDefs.map(param => {
      const value = parameters[param.name];
      return value !== undefined ? value : param.default;
    });

    // 获取连接配置
    const connection = query.connection;
    if (!connection || connection.type !== 'relational') {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '查询关联的连接不是关系型数据库连接'
      });
    }

    const relationalConfig = connection.relationalConfig;
    if (!relationalConfig) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: 'DataRelationalConfig',
        id: connection.id
      });
    }

    // 创建驱动并执行查询
    const driver = DriverFactory.createDriver(relationalConfig.dbType, {
      host: relationalConfig.host,
      port: relationalConfig.port,
      username: relationalConfig.username,
      password: relationalConfig.password,
      database: relationalConfig.database,
      charset: relationalConfig.charset,
      timeout: relationalConfig.queryTimeout
    });

    return await driver.executeQuery(sql, replacements);
  }
}

module.exports = new DataQueryService();

