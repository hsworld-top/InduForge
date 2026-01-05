const { DataMqttSubscription } = require("../models");
const mqttService = require("../services/mqttService");
const dataPointService = require("../services/dataPointService");
const AppError = require("../utils/AppError");
const ErrorCodes = require("../constants/errorCodes");
const { logger } = require("../utils/logger");

/**
 * MQTT 订阅控制器
 * 处理 MQTT 主题订阅的 CRUD 操作
 */
class MqttSubscriptionController {
  /**
   * 获取订阅列表
   * GET /api/v1/data/projects/:projectId/mqtt/connections/:connectionId/subscriptions
   * 或 GET /api/v1/data/projects/:projectId/mqtt/subscriptions?connectionId=xxx
   */
  async getSubscriptions(req, res, next) {
    try {
      const { projectId, connectionId: paramsConnectionId } = req.params;
      const {
        connectionId: queryConnectionId,
        page = 1,
        limit = 50,
      } = req.query;

      // 优先使用路径参数中的connectionId，其次使用查询参数
      const connectionId = paramsConnectionId || queryConnectionId;

      const offset = (page - 1) * limit;
      const where = { projectId };

      if (connectionId) {
        where.connectionId = connectionId;
      }

      const { count, rows } = await DataMqttSubscription.findAndCountAll({
        where,
        limit: parseInt(limit),
        offset,
        order: [["createdAt", "DESC"]],
        include: [
          {
            association: "connection",
            attributes: ["id", "name", "status"],
          },
        ],
      });

      res.json({
        success: true,
        data: rows,
        pagination: {
          page: parseInt(page),
          limit: parseInt(limit),
          total: count,
          totalPages: Math.ceil(count / limit),
        },
      });
    } catch (error) {
      logger.error(
        "[MqttSubscriptionController] Get subscriptions error:",
        error
      );
      next(error);
    }
  }

  /**
   * 获取单个订阅
   * GET /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId
   */
  async getSubscription(req, res, next) {
    try {
      const { subscriptionId } = req.params;

      const subscription = await DataMqttSubscription.findByPk(subscriptionId, {
        include: [
          {
            association: "connection",
            attributes: ["id", "name", "status"],
          },
          {
            association: "creator",
            attributes: ["id", "username", "name"],
          },
        ],
      });

      if (!subscription) {
        throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
          resource: "MqttSubscription",
          id,
        });
      }

      res.json({
        success: true,
        data: subscription,
      });
    } catch (error) {
      logger.error(
        "[MqttSubscriptionController] Get subscription error:",
        error
      );
      next(error);
    }
  }

  /**
   * 创建订阅
   * POST /api/v1/data/projects/:projectId/mqtt/connections/:connectionId/subscriptions
   */
  async createSubscription(req, res, next) {
    try {
      const { projectId, connectionId: paramsConnectionId } = req.params;
      const {
        connectionId: bodyConnectionId,
        name,
        topic,
        qos = 0,
        description,
        isEnabled = true,
        messageRetention = 100,
      } = req.body;
      const userId = req.user.id;

      // 优先使用路径参数中的connectionId，其次使用请求体
      const connectionId = paramsConnectionId || bodyConnectionId;

      if (!connectionId) {
        throw new AppError(ErrorCodes.INVALID_PARAMS, 400, {
          message: "缺少连接ID参数",
        });
      }

      // 验证连接是否存在且是 MQTT 类型
      const connection = await mqttService.getConnection(connectionId);
      if (!connection || connection.type !== "mqtt") {
        throw new AppError(ErrorCodes.INVALID_CONNECTION_TYPE, 400, {
          message: "连接不存在或不是 MQTT 类型",
        });
      }

      // 创建订阅记录
      const subscription = await DataMqttSubscription.create({
        projectId,
        connectionId,
        name,
        topic,
        qos,
        description,
        isEnabled,
        messageRetention,
        createdBy: userId,
      });

      // 如果启用，则立即订阅主题
      if (isEnabled) {
        await mqttService.subscribeToTopic(
          subscription.id,
          connectionId,
          topic,
          qos,
          projectId
        );
      }

      logger.info(
        `[MqttSubscriptionController] Subscription created: ${subscription.id}`
      );

      try {
        await dataPointService.syncFromMqttSubscription(
          projectId,
          subscription.id,
          userId
        );
      } catch (error) {
        logger.warn(
          "[MqttSubscriptionController] Sync datapoint failed:",
          error
        );
      }

      res.status(201).json({
        success: true,
        data: subscription,
        message: "订阅创建成功",
      });
    } catch (error) {
      logger.error(
        "[MqttSubscriptionController] Create subscription error:",
        error
      );
      next(error);
    }
  }

  /**
   * 更新订阅
   * PUT /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId
   */
  async updateSubscription(req, res, next) {
    try {
      const { subscriptionId } = req.params;
      const { name, topic, qos, description, isEnabled, messageRetention } =
        req.body;
      const userId = req.user.id;

      const subscription = await DataMqttSubscription.findByPk(subscriptionId);
      if (!subscription) {
        throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
          resource: "MqttSubscription",
          id: subscriptionId,
        });
      }

      const oldTopic = subscription.topic;
      const oldQos = subscription.qos;
      const oldEnabled = subscription.isEnabled;

      // 更新订阅记录
      await subscription.update({
        name,
        topic,
        qos,
        description,
        isEnabled,
        messageRetention,
        updatedBy: userId,
      });

      // 如果主题或 QoS 发生变化，需要重新订阅
      if (isEnabled && (topic !== oldTopic || qos !== oldQos)) {
        // 取消旧的订阅
        if (oldEnabled) {
          await mqttService.unsubscribeFromTopic(
            subscriptionId,
            subscription.connectionId,
            oldTopic
          );
        }
        // 订阅新的主题
        await mqttService.subscribeToTopic(
          subscriptionId,
          subscription.connectionId,
          topic,
          qos,
          subscription.projectId
        );
      } else if (isEnabled && !oldEnabled) {
        // 从禁用变为启用
        await mqttService.subscribeToTopic(
          subscriptionId,
          subscription.connectionId,
          topic,
          qos,
          subscription.projectId
        );
      } else if (!isEnabled && oldEnabled) {
        // 从启用变为禁用
        await mqttService.unsubscribeFromTopic(
          subscriptionId,
          subscription.connectionId,
          oldTopic
        );
      }

      logger.info(
        `[MqttSubscriptionController] Subscription updated: ${subscriptionId}`
      );

      try {
        await dataPointService.syncFromMqttSubscription(
          subscription.projectId,
          subscription.id,
          userId
        );
      } catch (error) {
        logger.warn(
          "[MqttSubscriptionController] Sync datapoint failed:",
          error
        );
      }

      res.json({
        success: true,
        data: subscription,
        message: "订阅更新成功",
      });
    } catch (error) {
      logger.error(
        "[MqttSubscriptionController] Update subscription error:",
        error
      );
      next(error);
    }
  }

  /**
   * 删除订阅
   * DELETE /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId
   */
  async deleteSubscription(req, res, next) {
    try {
      const { subscriptionId } = req.params;

      const subscription = await DataMqttSubscription.findByPk(subscriptionId);
      if (!subscription) {
        throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
          resource: "MqttSubscription",
          id: subscriptionId,
        });
      }

      // 如果订阅是启用状态，先取消订阅
      if (subscription.isEnabled) {
        await mqttService.unsubscribeFromTopic(
          subscriptionId,
          subscription.connectionId,
          subscription.topic
        );
      }

      // 删除订阅记录
      await subscription.destroy();

      try {
        await dataPointService.markInvalidBySource(
          subscription.projectId,
          "mqtt.subscription",
          subscription.id,
          subscription.updatedBy || subscription.createdBy || null
        );
        mqttService.clearSubscriptionDatapointCache(subscription.id);
      } catch (error) {
        logger.warn(
          "[MqttSubscriptionController] Mark datapoint invalid failed:",
          error
        );
      }

      logger.info(
        `[MqttSubscriptionController] Subscription deleted: ${subscriptionId}`
      );

      res.json({
        success: true,
        message: "订阅删除成功",
      });
    } catch (error) {
      logger.error(
        "[MqttSubscriptionController] Delete subscription error:",
        error
      );
      next(error);
    }
  }

  /**
   * 启用/禁用订阅
   * PATCH /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/toggle
   */
  async toggleSubscription(req, res, next) {
    try {
      const { subscriptionId } = req.params;
      const userId = req.user.id;

      const subscription = await DataMqttSubscription.findByPk(subscriptionId);
      if (!subscription) {
        throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
          resource: "MqttSubscription",
          id: subscriptionId,
        });
      }

      const newEnabled = !subscription.isEnabled;

      // 更新状态
      await subscription.update({
        isEnabled: newEnabled,
        updatedBy: userId,
      });

      // 根据新状态订阅或取消订阅
      if (newEnabled) {
        await mqttService.subscribeToTopic(
          subscriptionId,
          subscription.connectionId,
          subscription.topic,
          subscription.qos,
          subscription.projectId
        );
      } else {
        await mqttService.unsubscribeFromTopic(
          subscriptionId,
          subscription.connectionId,
          subscription.topic
        );
      }

      logger.info(
        `[MqttSubscriptionController] Subscription toggled: ${subscriptionId} -> ${newEnabled}`
      );

      res.json({
        success: true,
        data: subscription,
        message: newEnabled ? "订阅已启用" : "订阅已禁用",
      });
    } catch (error) {
      logger.error(
        "[MqttSubscriptionController] Toggle subscription error:",
        error
      );
      next(error);
    }
  }

  /**
   * 获取订阅的实时消息
   * GET /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/messages
   */
  async getMessages(req, res, next) {
    try {
      const { subscriptionId } = req.params;
      const { limit = 100 } = req.query;

      const subscription = await DataMqttSubscription.findByPk(subscriptionId);
      if (!subscription) {
        throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
          resource: "MqttSubscription",
          id: subscriptionId,
        });
      }

      // 从 MQTT 服务获取缓存的消息
      const messages = await mqttService.getSubscriptionMessages(
        subscriptionId,
        parseInt(limit)
      );

      res.json({
        success: true,
        data: messages,
      });
    } catch (error) {
      logger.error("[MqttSubscriptionController] Get messages error:", error);
      next(error);
    }
  }
}

module.exports = new MqttSubscriptionController();
