const {
  DataConnection,
  DataMqttConfig,
  DataMqttSubscription,
} = require("../models");
const MqttProtocol = require("./protocols/MqttProtocol");
const AppError = require("../utils/AppError");
const ErrorCodes = require("../constants/errorCodes");
const { logger } = require("../utils/logger");
const socketService = require("./socketService");

/**
 * MQTT 服务
 * 管理 MQTT 连接的生命周期
 */
class MqttService {
  constructor() {
    // 存储活动的 MQTT 协议实例 connectionId -> MqttProtocol
    this.activeConnections = new Map();
    // 存储订阅信息 subscriptionId -> { topic, qos }
    this.activeSubscriptions = new Map();
    // 消息缓存 subscriptionId -> Array<message>
    this.messageCache = new Map();
    // 缓存消息数量限制
    this.MAX_CACHE_SIZE = 100;
  }

  /**
   * 创建 MQTT 连接配置
   * @param {Object} connectionData - 连接基础数据
   * @param {Object} mqttConfig - MQTT 配置数据
   * @param {string} userId - 用户ID
   * @returns {Promise<Object>}
   */
  async createConnection(connectionData, mqttConfig, userId) {
    // 创建连接记录
    const connection = await DataConnection.create({
      ...connectionData,
      type: "mqtt",
      category: "message",
      createdBy: userId,
    });

    // 创建 MQTT 配置
    await DataMqttConfig.create({
      ...mqttConfig,
      connectionId: connection.id,
    });

    // 重新查询，包含关联数据
    const result = await DataConnection.findByPk(connection.id, {
      include: [
        {
          model: DataMqttConfig,
          as: "mqttConfig",
        },
      ],
    });

    logger.info(`[MqttService] Connection created: ${connection.id}`);
    return result;
  }

  /**
   * 测试 MQTT 连接
   * @param {Object} config - MQTT 配置
   * @returns {Promise<boolean>}
   */
  async testConnection(config) {
    const {
      brokerUrl,
      protocol = "mqtt",
      port = 1883,
      clientId,
      username,
      password,
      keepalive = 60,
      cleanSession = true,
      connectTimeout = 30000,
    } = config;

    if (!brokerUrl) {
      throw new AppError(ErrorCodes.INVALID_PARAMS, 400, {
        message: "Broker 地址不能为空",
      });
    }

    // 构建连接配置
    const mqttConfig = {
      brokerUrl,
      protocol,
      port,
      clientId: clientId || `induforge_test_${Date.now()}`,
      username,
      password,
      keepalive,
      cleanSession,
      connectTimeout,
    };

    // 创建临时协议实例进行测试
    const mqttProtocol = new MqttProtocol(mqttConfig);

    // 测试连接
    await mqttProtocol.testConnection();

    // 连接成功后立即断开
    await mqttProtocol.disconnect();

    logger.info("[MqttService] Connection test successful");
    return true;
  }

  /**
   * 更新 MQTT 连接配置
   * @param {string} connectionId - 连接ID
   * @param {Object} connectionData - 连接基础数据
   * @param {Object} mqttConfig - MQTT 配置数据
   * @param {string} userId - 用户ID
   * @returns {Promise<Object>}
   */
  async updateConnection(connectionId, connectionData, mqttConfig, userId) {
    const connection = await DataConnection.findByPk(connectionId, {
      include: [
        {
          model: DataMqttConfig,
          as: "mqttConfig",
        },
      ],
    });

    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: "Connection",
        id: connectionId,
      });
    }

    // 如果连接正在运行，先停止
    if (this.activeConnections.has(connectionId)) {
      await this.stopConnection(connectionId);
    }

    // 更新连接基础信息
    await connection.update({
      ...connectionData,
      updatedBy: userId,
    });

    // 更新 MQTT 配置
    if (connection.mqttConfig) {
      await connection.mqttConfig.update(mqttConfig);
    } else {
      await DataMqttConfig.create({
        ...mqttConfig,
        connectionId: connection.id,
      });
    }

    // 重新查询
    const result = await DataConnection.findByPk(connectionId, {
      include: [
        {
          model: DataMqttConfig,
          as: "mqttConfig",
        },
      ],
    });

    logger.info(`[MqttService] Connection updated: ${connectionId}`);
    return result;
  }

  /**
   * 删除 MQTT 连接
   * @param {string} connectionId - 连接ID
   */
  async deleteConnection(connectionId) {
    // 如果连接正在运行，先停止
    if (this.activeConnections.has(connectionId)) {
      await this.stopConnection(connectionId);
    }

    const connection = await DataConnection.findByPk(connectionId);
    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: "Connection",
        id: connectionId,
      });
    }

    await connection.destroy();
    logger.info(`[MqttService] Connection deleted: ${connectionId}`);
  }

  /**
   * 启动 MQTT 连接
   * @param {string} connectionId - 连接ID
   * @returns {Promise<Object>}
   */
  async startConnection(connectionId) {
    // 检查是否已经连接
    if (this.activeConnections.has(connectionId)) {
      const protocol = this.activeConnections.get(connectionId);
      const status = protocol.getStatus();
      if (status.connected) {
        logger.warn(`[MqttService] Connection already active: ${connectionId}`);
        return { status: "connected", message: "Already connected" };
      }
    }

    // 获取连接配置
    const connection = await DataConnection.findByPk(connectionId, {
      include: [
        {
          model: DataMqttConfig,
          as: "mqttConfig",
        },
      ],
    });

    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        resource: "Connection",
        id: connectionId,
      });
    }

    if (!connection.mqttConfig) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: "MQTT 配置未找到",
      });
    }

    // 构建配置对象
    const config = {
      brokerUrl: connection.mqttConfig.brokerUrl,
      protocol: connection.mqttConfig.protocol,
      port: connection.mqttConfig.port,
      clientId: connection.mqttConfig.clientId,
      username: connection.mqttConfig.username,
      password: connection.mqttConfig.password,
      keepalive: connection.mqttConfig.keepalive,
      cleanSession: connection.mqttConfig.cleanSession,
      qos: connection.mqttConfig.qos,
      reconnectPeriod: connection.mqttConfig.reconnectPeriod,
      connectTimeout: connection.mqttConfig.connectTimeout,
      will: connection.mqttConfig.will,
      sslConfig: connection.mqttConfig.sslConfig,
    };

    try {
      // 创建协议实例并连接
      const protocol = new MqttProtocol(config);
      await protocol.connect();

      // 存储活动连接
      this.activeConnections.set(connectionId, protocol);

      // 更新连接状态
      await connection.update({
        status: "connected",
        lastConnectedAt: new Date(),
        lastErrorMessage: null,
      });

      // 自动订阅所有启用的订阅
      try {
        const enabledSubscriptions = await DataMqttSubscription.findAll({
          where: {
            connectionId: connectionId,
            isEnabled: true,
          },
        });

        for (const subscription of enabledSubscriptions) {
          try {
            await this.subscribeToTopic(
              subscription.id,
              connectionId,
              subscription.topic,
              subscription.qos,
              subscription.projectId
            );
            logger.info(
              `[MqttService] Auto-subscribed to topic: ${subscription.topic} for subscription: ${subscription.id}`
            );
          } catch (subscribeError) {
            logger.error(
              `[MqttService] Auto-subscribe error for subscription ${subscription.id}:`,
              subscribeError
            );
          }
        }

        logger.info(
          `[MqttService] Auto-subscribed ${enabledSubscriptions.length} topics for connection: ${connectionId}`
        );
      } catch (autoSubscribeError) {
        logger.error(
          `[MqttService] Auto-subscribe error for connection ${connectionId}:`,
          autoSubscribeError
        );
      }

      // 自动订阅所有启用的订阅
      try {
        const enabledSubscriptions = await DataMqttSubscription.findAll({
          where: {
            connectionId: connectionId,
            isEnabled: true,
          },
        });

        logger.info(
          `[MqttService] Found ${enabledSubscriptions.length} enabled subscriptions for connection: ${connectionId}`
        );

        for (const subscription of enabledSubscriptions) {
          try {
            logger.info(
              `[MqttService] Auto-subscribing to topic: ${subscription.topic} (QoS: ${subscription.qos}) for subscription: ${subscription.id}`
            );
            await this.subscribeToTopic(
              subscription.id,
              connectionId,
              subscription.topic,
              subscription.qos,
              subscription.projectId
            );
            logger.info(
              `[MqttService] Auto-subscribed to topic: ${subscription.topic} for subscription: ${subscription.id}`
            );
          } catch (subscribeError) {
            logger.error(
              `[MqttService] Auto-subscribe error for subscription ${subscription.id}:`,
              subscribeError
            );
          }
        }

        logger.info(
          `[MqttService] Auto-subscribed ${enabledSubscriptions.length} topics for connection: ${connectionId}`
        );
      } catch (autoSubscribeError) {
        logger.error(
          `[MqttService] Auto-subscribe error for connection ${connectionId}:`,
          autoSubscribeError
        );
      }

      logger.info(`[MqttService] Connection started: ${connectionId}`);
      return {
        status: "connected",
        message: "Connection started successfully",
      };
    } catch (error) {
      logger.error("[MqttService] Start connection error:", error);

      // 更新连接状态为错误
      try {
        await DataConnection.update(
          {
            status: "error",
            lastErrorMessage: error.message,
          },
          { where: { id: connectionId } }
        );
      } catch (updateError) {
        logger.error(
          "[MqttService] Update connection status error:",
          updateError
        );
      }

      throw error;
    }
  }

  /**
   * 停止 MQTT 连接
   * @param {string} connectionId - 连接ID
   */
  async stopConnection(connectionId) {
    if (!this.activeConnections.has(connectionId)) {
      logger.warn(`[MqttService] Connection not active: ${connectionId}`);
      return { status: "disconnected", message: "Connection not active" };
    }

    const protocol = this.activeConnections.get(connectionId);
    await protocol.disconnect();
    this.activeConnections.delete(connectionId);

    // 更新连接状态
    await DataConnection.update(
      { status: "disconnected" },
      { where: { id: connectionId } }
    );

    logger.info(`[MqttService] Connection stopped: ${connectionId}`);
    return {
      status: "disconnected",
      message: "Connection stopped successfully",
    };
  }

  /**
   * 获取连接状态
   * @param {string} connectionId - 连接ID
   * @returns {Object}
   */
  getConnectionStatus(connectionId) {
    if (!this.activeConnections.has(connectionId)) {
      return {
        connected: false,
        connecting: false,
        message: "Connection not active",
      };
    }

    const protocol = this.activeConnections.get(connectionId);
    return protocol.getStatus();
  }

  /**
   * 获取活动连接的协议实例
   * @param {string} connectionId - 连接ID
   * @returns {MqttProtocol|null}
   */
  getProtocol(connectionId) {
    return this.activeConnections.get(connectionId) || null;
  }

  /**
   * 获取所有活动连接
   * @returns {Array}
   */
  getActiveConnections() {
    return Array.from(this.activeConnections.keys());
  }

  /**
   * 获取连接对象（包含配置）
   * @param {string} connectionId - 连接ID
   * @returns {Promise<Object>}
   */
  async getConnection(connectionId) {
    return await DataConnection.findByPk(connectionId, {
      include: [
        {
          model: DataMqttConfig,
          as: "mqttConfig",
        },
      ],
    });
  }

  /**
   * 订阅主题（数据库订阅）
   * @param {string} subscriptionId - 订阅ID
   * @param {string} connectionId - 连接ID
   * @param {string} topic - 主题
   * @param {number} qos - QoS 等级
   * @param {string} projectId - 项目ID
   * @returns {Promise<void>}
   */
  async subscribeToTopic(
    subscriptionId,
    connectionId,
    topic,
    qos = 0,
    projectId
  ) {
    const protocol = this.getProtocol(connectionId);
    if (!protocol) {
      throw new AppError(ErrorCodes.INVALID_PARAMS, 400, {
        message: "连接未启动",
      });
    }

    // 注册消息处理器
    const messageHandler = (message) => {
      this.handleMqttMessage(subscriptionId, projectId, message);
    };

    // 订阅主题
    await protocol.subscribe(topic, { qos }, messageHandler);

    // 保存订阅信息
    this.activeSubscriptions.set(subscriptionId, {
      topic,
      qos,
      connectionId,
      projectId,
    });

    logger.info(
      `[MqttService] Subscribed to topic: ${topic} (QoS ${qos}), subscription: ${subscriptionId}`
    );
  }

  /**
   * 处理MQTT消息
   * @param {string} subscriptionId - 订阅ID
   * @param {string} projectId - 项目ID
   * @param {Object} message - 消息
   */
  handleMqttMessage(subscriptionId, projectId, message) {
    // 缓存消息
    if (!this.messageCache.has(subscriptionId)) {
      this.messageCache.set(subscriptionId, []);
    }

    const cache = this.messageCache.get(subscriptionId);
    cache.push({
      topic: message.topic,
      payload: message.payload.toString(),
      qos: message.qos || 0,
      timestamp: Date.now(),
    });

    // 限制缓存大小
    if (cache.length > this.MAX_CACHE_SIZE) {
      cache.shift();
    }

    // 通过Socket.IO广播消息
    socketService.broadcastMqttMessage(subscriptionId, {
      topic: message.topic,
      payload: message.payload.toString(),
      qos: message.qos || 0,
      timestamp: Date.now(),
    });

    // 解析消息并更新Tag值（异步执行，不阻塞消息处理）
    this.processTagsForMessage(
      subscriptionId,
      message.topic,
      message.payload.toString()
    ).catch((error) => {
      logger.error(
        `[MqttService] Error processing tags for subscription ${subscriptionId}:`,
        error
      );
    });

    logger.debug(
      `[MqttService] Handled message for subscription: ${subscriptionId}`
    );
  }

  /**
   * 处理消息的Tag解析（异步）
   * @param {string} subscriptionId - 订阅ID
   * @param {string} topic - 主题
   * @param {string} payload - 消息内容
   */
  async processTagsForMessage(subscriptionId, topic, payload) {
    try {
      const mqttTagService = require("./mqttTagService");
      await mqttTagService.processMessage(subscriptionId, topic, payload);
    } catch (error) {
      logger.error(`[MqttService] Tag processing failed:`, error);
    }
  }

  /**
   * 订阅主题（旧方法，保持兼容）
   * @param {string} connectionId - 连接ID
   * @param {string} topic - 主题
   * @param {number} qos - QoS 等级
   * @returns {Promise<void>}
   */
  async subscribeTopic(connectionId, topic, qos = 0) {
    const protocol = this.getProtocol(connectionId);
    if (!protocol) {
      throw new AppError(ErrorCodes.INVALID_PARAMS, 400, {
        message: "连接未启动",
      });
    }

    await protocol.subscribe(topic, { qos });
    logger.info(`[MqttService] Subscribed to topic: ${topic} (QoS ${qos})`);
  }

  /**
   * 取消订阅（数据库订阅）
   * @param {string} subscriptionId - 订阅ID
   * @param {string} connectionId - 连接ID
   * @param {string} topic - 主题
   * @returns {Promise<void>}
   */
  async unsubscribeFromTopic(subscriptionId, connectionId, topic) {
    const protocol = this.getProtocol(connectionId);
    if (!protocol) {
      logger.warn(
        `[MqttService] Connection not active for unsubscribe: ${connectionId}`
      );
      return;
    }

    await protocol.unsubscribe(topic);

    // 清理订阅信息和缓存
    this.activeSubscriptions.delete(subscriptionId);
    this.messageCache.delete(subscriptionId);

    logger.info(
      `[MqttService] Unsubscribed from topic: ${topic}, subscription: ${subscriptionId}`
    );
  }

  /**
   * 取消订阅主题（旧方法，保持兼容）
   * @param {string} connectionId - 连接ID
   * @param {string} topic - 主题
   * @returns {Promise<void>}
   */
  async unsubscribeTopic(connectionId, topic) {
    const protocol = this.getProtocol(connectionId);
    if (!protocol) {
      logger.warn(
        `[MqttService] Connection not active for unsubscribe: ${connectionId}`
      );
      return;
    }

    await protocol.unsubscribe(topic);
    logger.info(`[MqttService] Unsubscribed from topic: ${topic}`);
  }

  /**
   * 获取订阅的消息（从内存缓存）
   * @param {string} subscriptionId - 订阅ID
   * @param {number} limit - 消息数量限制
   * @returns {Promise<Array>}
   */
  async getSubscriptionMessages(subscriptionId, limit = 100) {
    const cache = this.messageCache.get(subscriptionId);
    if (!cache) {
      return [];
    }

    // 返回最近的消息（限制数量）
    return cache.slice(-limit);
  }

  /**
   * 停止所有连接（用于服务关闭）
   */
  async stopAllConnections() {
    logger.info("[MqttService] Stopping all connections...");
    const promises = [];
    for (const connectionId of this.activeConnections.keys()) {
      promises.push(this.stopConnection(connectionId));
    }
    await Promise.allSettled(promises);
    logger.info("[MqttService] All connections stopped");
  }
}

// 导出单例
module.exports = new MqttService();
