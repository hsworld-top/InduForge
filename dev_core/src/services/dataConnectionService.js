const {
  DataConnection,
  DataRelationalConfig,
  DataMqttConfig,
} = require("../models");
const DriverFactory = require("./drivers/DriverFactory");
const mqttService = require("./mqttService");
const AppError = require("../utils/AppError");
const ErrorCodes = require("../constants/errorCodes");

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
          as: "relationalConfig",
          required: false,
        },
        {
          model: DataMqttConfig,
          as: "mqttConfig",
          required: false,
        },
      ],
      limit: parseInt(limit),
      offset,
      order: [["createdAt", "DESC"]],
    });

    return {
      connections: rows,
      pagination: {
        page: parseInt(page),
        limit: parseInt(limit),
        total: count,
        totalPages: Math.ceil(count / limit),
      },
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

    // 根据 type 自动推断 category（如果未提供）
    let finalCategory = category;
    if (!finalCategory) {
      const categoryMap = {
        relational: "database",
        mqtt: "message",
        websocket: "message",
        opcua: "protocol",
        modbus: "protocol",
        s7: "protocol",
        http: "api",
      };
      finalCategory = categoryMap[type] || "api";
    }

    // 验证 category 值是否合法
    const validCategories = ["database", "message", "protocol", "api"];
    if (!validCategories.includes(finalCategory)) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: `无效的 category 值: ${finalCategory}，必须是 ${validCategories.join(
          ", "
        )} 之一`,
      });
    }

    // 创建数据连接
    const connection = await DataConnection.create({
      projectId,
      name,
      type,
      category: finalCategory,
      status: "unknown",
      createdBy: userId,
    });

    // 根据类型创建对应的配置
    if (type === "relational" && config) {
      // 提取 SSL 相关字段到 sslConfig
      const { sslMode, sslCa, sslCert, sslKey, ...otherConfig } = config;

      const relationalConfig = {
        connectionId: connection.id,
        ...otherConfig,
      };

      // 如果有 SSL 配置，整合到 sslConfig 字段
      if (sslMode && sslMode !== "disable") {
        relationalConfig.ssl = true;
        relationalConfig.sslConfig = {
          mode: sslMode,
          ca: sslCa || null,
          cert: sslCert || null,
          key: sslKey || null,
        };
      } else {
        relationalConfig.ssl = false;
        relationalConfig.sslConfig = null;
      }

      await DataRelationalConfig.create(relationalConfig);
    } else if (type === "mqtt" && config) {
      // 创建 MQTT 配置
      await DataMqttConfig.create({
        connectionId: connection.id,
        ...config,
      });
    }

    // 重新获取完整数据
    const fullConnection = await DataConnection.findByPk(connection.id, {
      include: [
        {
          model: DataRelationalConfig,
          as: "relationalConfig",
          required: false,
        },
        {
          model: DataMqttConfig,
          as: "mqttConfig",
          required: false,
        },
      ],
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
    if (type === "relational") {
      return await this.testRelationalConnection(config);
    } else if (type === "mqtt") {
      return await mqttService.testConnection(config);
    }
    // 其他类型的连接测试（WebSocket 等）可以在这里扩展
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
      message: `暂不支持测试 ${type} 类型的连接`,
    });
  }

  /**
   * 测试关系型数据库连接
   * @param {Object} config - 数据库配置
   * @returns {Promise<boolean>} 连接是否成功
   */
  async testRelationalConnection(config) {
    const {
      dbType = "mysql",
      host,
      port,
      username,
      password,
      database,
    } = config;

    if (!host || !port || !username || !database) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "请填写完整的数据库连接信息",
      });
    }

    try {
      // 传递完整的配置对象，包括 SQL Server 的 encrypt、trustServerCertificate 等参数
      const driver = DriverFactory.createDriver(dbType, config);

      return await driver.testConnection();
    } catch (error) {
      throw new AppError(ErrorCodes.DATABASE_ERROR, 500, {
        message: "数据库连接失败，请检查配置信息",
        error: error.message,
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
      where: { id: connectionId, projectId, type: "relational" },
      include: [
        {
          model: DataRelationalConfig,
          as: "relationalConfig",
          required: true,
        },
      ],
    });

    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: "DataConnection",
        id: connectionId,
      });
    }

    const relationalConfig = connection.relationalConfig;

    // 构建驱动配置
    const driverConfig = {
      host: relationalConfig.host,
      port: relationalConfig.port,
      username: relationalConfig.username,
      password: relationalConfig.password,
      database: relationalConfig.database,
      charset: relationalConfig.charset,
      timeout: relationalConfig.queryTimeout,
    };

    // 添加 PostgreSQL 特有配置
    if (relationalConfig.dbType === "postgresql") {
      if (relationalConfig.schema) {
        driverConfig.schema = relationalConfig.schema;
      }

      // 添加 SSL 配置
      if (relationalConfig.ssl && relationalConfig.sslConfig) {
        driverConfig.sslConfig = relationalConfig.sslConfig;
      }
    }

    // 添加 SQL Server 特有配置
    if (relationalConfig.dbType === "sqlserver") {
      driverConfig.encrypt =
        relationalConfig.encrypt !== undefined
          ? relationalConfig.encrypt
          : false;
      driverConfig.trustServerCertificate =
        relationalConfig.trustServerCertificate !== undefined
          ? relationalConfig.trustServerCertificate
          : true;
    }

    try {
      const driver = DriverFactory.createDriver(
        relationalConfig.dbType,
        driverConfig
      );
      return await driver.getTables();
    } catch (error) {
      // 将数据库错误包装成 AppError，以便前端能获取详细错误信息
      throw new AppError(ErrorCodes.DATABASE_ERROR, 400, {
        message: error.message,
      });
    }
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
        message: "不支持的表名格式",
      });
    }

    // 获取连接配置
    const connection = await DataConnection.findOne({
      where: { id: connectionId, projectId, type: "relational" },
      include: [
        {
          model: DataRelationalConfig,
          as: "relationalConfig",
          required: true,
        },
      ],
    });

    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: "DataConnection",
        id: connectionId,
      });
    }

    const relationalConfig = connection.relationalConfig;

    // 构建驱动配置
    const driverConfig = {
      host: relationalConfig.host,
      port: relationalConfig.port,
      username: relationalConfig.username,
      password: relationalConfig.password,
      database: relationalConfig.database,
      charset: relationalConfig.charset,
      timeout: relationalConfig.queryTimeout,
    };

    // 添加 PostgreSQL 特有配置
    if (relationalConfig.dbType === "postgresql") {
      if (relationalConfig.schema) {
        driverConfig.schema = relationalConfig.schema;
      }

      // 添加 SSL 配置
      if (relationalConfig.ssl && relationalConfig.sslConfig) {
        driverConfig.sslConfig = relationalConfig.sslConfig;
      }
    }

    // 添加 SQL Server 特有配置
    if (relationalConfig.dbType === "sqlserver") {
      driverConfig.encrypt =
        relationalConfig.encrypt !== undefined
          ? relationalConfig.encrypt
          : false;
      driverConfig.trustServerCertificate =
        relationalConfig.trustServerCertificate !== undefined
          ? relationalConfig.trustServerCertificate
          : true;
    }

    try {
      const driver = DriverFactory.createDriver(
        relationalConfig.dbType,
        driverConfig
      );
      return await driver.getTableData(tableName, options);
    } catch (error) {
      // 将数据库错误包装成 AppError
      throw new AppError(ErrorCodes.DATABASE_ERROR, 400, {
        message: error.message,
      });
    }
  }

  /**
   * 更新数据连接
   * @param {string} projectId - 工程ID
   * @param {string} connectionId - 连接ID
   * @param {Object} data - 更新数据 { name, type, config }
   * @param {string} userId - 用户ID
   * @returns {Promise<Object>} 更新后的连接对象
   */
  async updateConnection(projectId, connectionId, data, userId) {
    const { name, type, config } = data;

    // 查找连接
    const connection = await DataConnection.findOne({
      where: { id: connectionId, projectId },
    });

    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: "DataConnection",
        id: connectionId,
      });
    }

    // 更新基本信息
    if (name) connection.name = name;
    if (type) connection.type = type;
    connection.updatedBy = userId;
    await connection.save();

    // 如果是关系库类型，更新配置
    if (type === "relational" && config) {
      // 提取 SSL 相关字段到 sslConfig
      const { sslMode, sslCa, sslCert, sslKey, ...otherConfig } = config;

      const configToSave = { ...otherConfig };

      // 如果有 SSL 配置，整合到 sslConfig 字段
      if (sslMode && sslMode !== "disable") {
        configToSave.ssl = true;
        configToSave.sslConfig = {
          mode: sslMode,
          ca: sslCa || null,
          cert: sslCert || null,
          key: sslKey || null,
        };
      } else {
        configToSave.ssl = false;
        configToSave.sslConfig = null;
      }

      const relationalConfig = await DataRelationalConfig.findOne({
        where: { connectionId },
      });

      if (relationalConfig) {
        // 更新现有配置
        await relationalConfig.update(configToSave);
      } else {
        // 创建新配置
        await DataRelationalConfig.create({
          connectionId,
          ...configToSave,
        });
      }
    }

    // 重新获取完整数据
    const fullConnection = await DataConnection.findByPk(connectionId, {
      include: [
        {
          model: DataRelationalConfig,
          as: "relationalConfig",
          required: false,
        },
      ],
    });

    return fullConnection;
  }

  /**
   * 删除数据连接
   * @param {string} projectId - 工程ID
   * @param {string} connectionId - 连接ID
   * @returns {Promise<boolean>} 是否删除成功
   */
  async deleteConnection(projectId, connectionId) {
    // 查找连接
    const connection = await DataConnection.findOne({
      where: { id: connectionId, projectId },
    });

    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: "DataConnection",
        id: connectionId,
      });
    }

    // 删除关联的配置
    if (connection.type === "relational") {
      await DataRelationalConfig.destroy({
        where: { connectionId },
      });
    }

    // 删除连接
    await connection.destroy();

    return true;
  }

  /**
   * 更新连接状态
   * @param {string} projectId - 工程ID
   * @param {string} connectionId - 连接ID
   * @param {string} status - 状态 (connected, disconnected, error, unknown)
   * @returns {Promise<Object>} 更新后的连接对象
   */
  async updateConnectionStatus(projectId, connectionId, status) {
    // 验证状态值
    const validStatuses = ["connected", "disconnected", "error", "unknown"];
    if (!validStatuses.includes(status)) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: `无效的状态值: ${status}，必须是 ${validStatuses.join(
          ", "
        )} 之一`,
      });
    }

    // 查找连接
    const connection = await DataConnection.findOne({
      where: { id: connectionId, projectId },
    });

    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: "DataConnection",
        id: connectionId,
      });
    }

    // 更新状态
    connection.status = status;
    await connection.save();

    // 重新获取完整数据
    const fullConnection = await DataConnection.findByPk(connectionId, {
      include: [
        {
          model: DataRelationalConfig,
          as: "relationalConfig",
          required: false,
        },
      ],
    });

    return fullConnection;
  }

  /**
   * 获取表结构信息
   * @param {string} projectId - 工程ID
   * @param {string} connectionId - 连接ID
   * @param {string} tableName - 表名
   * @returns {Promise<Object>} 表结构信息
   */
  async getTableStructure(projectId, connectionId, tableName) {
    // 验证表名格式（防止 SQL 注入）
    if (!/^[A-Za-z0-9_]+$/.test(tableName)) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "不支持的表名格式",
      });
    }

    // 获取连接配置
    const connection = await DataConnection.findOne({
      where: { id: connectionId, projectId, type: "relational" },
      include: [
        {
          model: DataRelationalConfig,
          as: "relationalConfig",
          required: true,
        },
      ],
    });

    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: "DataConnection",
        id: connectionId,
      });
    }

    const relationalConfig = connection.relationalConfig;

    // 构建驱动配置
    const driverConfig = {
      host: relationalConfig.host,
      port: relationalConfig.port,
      username: relationalConfig.username,
      password: relationalConfig.password,
      database: relationalConfig.database,
      charset: relationalConfig.charset,
      timeout: relationalConfig.queryTimeout,
    };

    // 添加 PostgreSQL 特有配置
    if (relationalConfig.dbType === "postgresql") {
      if (relationalConfig.schema) {
        driverConfig.schema = relationalConfig.schema;
      }

      // 添加 SSL 配置
      if (relationalConfig.ssl && relationalConfig.sslConfig) {
        driverConfig.sslConfig = relationalConfig.sslConfig;
      }
    }

    // 添加 SQL Server 特有配置
    if (relationalConfig.dbType === "sqlserver") {
      driverConfig.encrypt =
        relationalConfig.encrypt !== undefined
          ? relationalConfig.encrypt
          : false;
      driverConfig.trustServerCertificate =
        relationalConfig.trustServerCertificate !== undefined
          ? relationalConfig.trustServerCertificate
          : true;
    }

    try {
      const driver = DriverFactory.createDriver(
        relationalConfig.dbType,
        driverConfig
      );
      return await driver.getTableStructure(tableName);
    } catch (error) {
      // 将数据库错误包装成 AppError
      throw new AppError(ErrorCodes.DATABASE_ERROR, 400, {
        message: error.message,
      });
    }
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
    if (!sql || typeof sql !== "string" || !sql.trim()) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "SQL语句不能为空",
      });
    }

    if (!Array.isArray(parameters)) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "参数必须是数组格式",
      });
    }

    // 获取连接配置
    const connection = await DataConnection.findOne({
      where: { id: connectionId, projectId, type: "relational" },
      include: [
        {
          model: DataRelationalConfig,
          as: "relationalConfig",
          required: true,
        },
      ],
    });

    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: "DataConnection",
        id: connectionId,
      });
    }

    const relationalConfig = connection.relationalConfig;

    // 构建驱动配置
    const driverConfig = {
      host: relationalConfig.host,
      port: relationalConfig.port,
      username: relationalConfig.username,
      password: relationalConfig.password,
      database: relationalConfig.database,
      charset: relationalConfig.charset,
      timeout: relationalConfig.queryTimeout,
    };

    // 添加 PostgreSQL 特有配置
    if (relationalConfig.dbType === "postgresql") {
      if (relationalConfig.schema) {
        driverConfig.schema = relationalConfig.schema;
      }

      // 添加 SSL 配置
      if (relationalConfig.ssl && relationalConfig.sslConfig) {
        driverConfig.sslConfig = relationalConfig.sslConfig;
      }
    }

    // 添加 SQL Server 特有配置
    if (relationalConfig.dbType === "sqlserver") {
      driverConfig.encrypt =
        relationalConfig.encrypt !== undefined
          ? relationalConfig.encrypt
          : false;
      driverConfig.trustServerCertificate =
        relationalConfig.trustServerCertificate !== undefined
          ? relationalConfig.trustServerCertificate
          : true;
    }

    const driver = DriverFactory.createDriver(
      relationalConfig.dbType,
      driverConfig
    );

    const startTime = Date.now();
    try {
      const result = await driver.executeQuery(sql, parameters);
      const executionTime = Date.now() - startTime;

      return {
        data: result,
        executionTime,
      };
    } catch (error) {
      // 直接抛出数据库的原始错误信息
      throw new AppError(ErrorCodes.DATABASE_ERROR, 400, {
        message: error.message,
      });
    }
  }
}

module.exports = new DataConnectionService();
