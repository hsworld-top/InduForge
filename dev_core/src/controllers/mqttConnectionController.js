const mqttService = require("../services/mqttService");
const { DataConnection, DataMqttConfig } = require("../models");
const ApiResponse = require("../utils/response");
const AppError = require("../utils/AppError");
const { logger } = require("../utils/logger");

/**
 * MQTT 连接控制器
 */
class MqttConnectionController {
  /**
   * 创建 MQTT 连接
   */
  async create(req, res, next) {
    try {
      const { projectId } = req.params;
      const userId = req.user.id;
      const {
        name,
        isEnabled,
        retryCount,
        retryInterval,
        healthCheckInterval,
        // MQTT 特定配置
        brokerUrl,
        protocol,
        port,
        clientId,
        username,
        password,
        keepalive,
        cleanSession,
        qos,
        reconnectPeriod,
        connectTimeout,
        will,
        sslConfig,
      } = req.body;

      // 验证必填字段
      if (!name || !brokerUrl) {
        throw new AppError(
          "VALIDATION_ERROR",
          400,
          "Name and brokerUrl are required"
        );
      }

      // 构建连接数据
      const connectionData = {
        projectId,
        name,
        isEnabled: isEnabled !== undefined ? isEnabled : true,
        retryCount: retryCount || 3,
        retryInterval: retryInterval || 5000,
        healthCheckInterval: healthCheckInterval || 30000,
      };

      // 构建 MQTT 配置
      const mqttConfig = {
        brokerUrl,
        protocol: protocol || "mqtt",
        port: port || 1883,
        clientId,
        username,
        password,
        keepalive: keepalive || 60,
        cleanSession: cleanSession !== undefined ? cleanSession : true,
        qos: qos !== undefined ? qos : 0,
        reconnectPeriod: reconnectPeriod || 5000,
        connectTimeout: connectTimeout || 30000,
        will,
        sslConfig,
      };

      // 创建连接
      const connection = await mqttService.createConnection(
        connectionData,
        mqttConfig,
        userId
      );

      logger.info(
        `[MqttConnectionController] MQTT connection created: ${connection.id}`
      );

      return ApiResponse.success(
        res,
        connection,
        "mqtt_connection_create_success",
        {},
        201
      );
    } catch (error) {
      next(error);
    }
  }

  /**
   * 获取 MQTT 连接列表
   */
  async list(req, res, next) {
    try {
      const { projectId } = req.params;
      const { page = 1, pageSize = 50 } = req.query;

      const offset = (page - 1) * pageSize;
      const limit = parseInt(pageSize);

      const { rows, count } = await DataConnection.findAndCountAll({
        where: {
          projectId,
          type: "mqtt",
        },
        include: [
          {
            model: DataMqttConfig,
            as: "mqttConfig",
            attributes: { exclude: ["password"] }, // 不返回密码
          },
        ],
        limit,
        offset,
        order: [["createdAt", "DESC"]],
      });

      // 添加连接状态信息
      const connections = rows.map((conn) => {
        const status = mqttService.getConnectionStatus(conn.id);
        return {
          ...conn.toJSON(),
          runtimeStatus: status,
        };
      });

      return ApiResponse.success(res, {
        connections,
        pagination: {
          total: count,
          page: parseInt(page),
          pageSize: limit,
          totalPages: Math.ceil(count / limit),
        },
      }, "mqtt_connection_list_success");
    } catch (error) {
      next(error);
    }
  }

  /**
   * 获取单个 MQTT 连接详情
   */
  async getOne(req, res, next) {
    try {
      const { id } = req.params;

      const connection = await DataConnection.findByPk(id, {
        include: [
          {
            model: DataMqttConfig,
            as: "mqttConfig",
            attributes: { exclude: ["password"] },
          },
        ],
      });

      if (!connection) {
        throw new AppError("CONNECTION_NOT_FOUND", 404, "Connection not found");
      }

      // 添加运行时状态
      const status = mqttService.getConnectionStatus(id);
      const result = {
        ...connection.toJSON(),
        runtimeStatus: status,
      };

      return ApiResponse.success(res, result, "mqtt_connection_get_success");
    } catch (error) {
      next(error);
    }
  }

  /**
   * 更新 MQTT 连接
   */
  async update(req, res, next) {
    try {
      const { id } = req.params;
      const userId = req.user.id;
      const {
        name,
        isEnabled,
        retryCount,
        retryInterval,
        healthCheckInterval,
        // MQTT 特定配置
        brokerUrl,
        protocol,
        port,
        clientId,
        username,
        password,
        keepalive,
        cleanSession,
        qos,
        reconnectPeriod,
        connectTimeout,
        will,
        sslConfig,
      } = req.body;

      // 构建连接数据
      const connectionData = {
        name,
        isEnabled,
        retryCount,
        retryInterval,
        healthCheckInterval,
      };

      // 移除 undefined 值
      Object.keys(connectionData).forEach(
        (key) => connectionData[key] === undefined && delete connectionData[key]
      );

      // 构建 MQTT 配置
      const mqttConfig = {
        brokerUrl,
        protocol,
        port,
        clientId,
        username,
        password,
        keepalive,
        cleanSession,
        qos,
        reconnectPeriod,
        connectTimeout,
        will,
        sslConfig,
      };

      // 移除 undefined 值
      Object.keys(mqttConfig).forEach(
        (key) => mqttConfig[key] === undefined && delete mqttConfig[key]
      );

      // 更新连接
      const connection = await mqttService.updateConnection(
        id,
        connectionData,
        mqttConfig,
        userId
      );

      logger.info(`[MqttConnectionController] MQTT connection updated: ${id}`);

      return ApiResponse.success(
        res,
        connection,
        "mqtt_connection_update_success"
      );
    } catch (error) {
      next(error);
    }
  }

  /**
   * 删除 MQTT 连接
   */
  async delete(req, res, next) {
    try {
      const { id } = req.params;

      await mqttService.deleteConnection(id);

      logger.info(`[MqttConnectionController] MQTT connection deleted: ${id}`);

      return ApiResponse.success(res, null, "mqtt_connection_delete_success");
    } catch (error) {
      next(error);
    }
  }

  /**
   * 测试 MQTT 连接
   */
  async test(req, res, next) {
    try {
      const {
        brokerUrl,
        protocol,
        port,
        clientId,
        username,
        password,
        keepalive,
        cleanSession,
        qos,
        reconnectPeriod,
        connectTimeout,
        will,
        sslConfig,
      } = req.body;

      if (!brokerUrl) {
        throw new AppError("VALIDATION_ERROR", 400, "brokerUrl is required");
      }

      // 构建配置
      const config = {
        brokerUrl,
        protocol: protocol || "mqtt",
        port,
        clientId,
        username,
        password,
        keepalive: keepalive || 60,
        cleanSession: cleanSession !== undefined ? cleanSession : true,
        qos: qos !== undefined ? qos : 0,
        reconnectPeriod: reconnectPeriod || 5000,
        connectTimeout: connectTimeout || 30000,
        will,
        sslConfig,
      };

      // 测试连接
      await mqttService.testConnection(config);

      logger.info("[MqttConnectionController] MQTT connection test successful");

      return ApiResponse.success(
        res,
        { success: true },
        "mqtt_connection_test_success"
      );
    } catch (error) {
      logger.error(
        "[MqttConnectionController] MQTT connection test failed:",
        error
      );
      next(error);
    }
  }

  /**
   * 启动 MQTT 连接
   */
  async start(req, res, next) {
    try {
      const { id } = req.params;

      const result = await mqttService.startConnection(id);

      logger.info(`[MqttConnectionController] MQTT connection started: ${id}`);

      return ApiResponse.success(
        res,
        result,
        "mqtt_connection_start_success"
      );
    } catch (error) {
      next(error);
    }
  }

  /**
   * 停止 MQTT 连接
   */
  async stop(req, res, next) {
    try {
      const { id } = req.params;

      const result = await mqttService.stopConnection(id);

      logger.info(`[MqttConnectionController] MQTT connection stopped: ${id}`);

      return ApiResponse.success(
        res,
        result,
        "mqtt_connection_stop_success"
      );
    } catch (error) {
      next(error);
    }
  }

  /**
   * 获取连接状态
   */
  async getStatus(req, res, next) {
    try {
      const { id } = req.params;

      const status = mqttService.getConnectionStatus(id);

      return ApiResponse.success(
        res,
        status,
        "mqtt_connection_status_success"
      );
    } catch (error) {
      next(error);
    }
  }
}

module.exports = new MqttConnectionController();
