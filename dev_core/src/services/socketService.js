const { Server } = require("socket.io");
const { DataMqttSubscription, DataMqttTag } = require("../models");
const { logger } = require("../utils/logger");
const { normalizePath } = require("../utils/datapointPath");

/**
 * Socket.IO 鏈嶅姟
 * 绠＄悊WebSocket杩炴帴锛屽箍鎾璏QTT娑堟伅
 */
class SocketService {
  constructor() {
    this.io = null;
    this.connections = new Map(); // userId -> socket
  }

  async ensureMqttConnectionBySubscription(subscriptionId) {
    if (!subscriptionId) return;
    try {
      const subscription = await DataMqttSubscription.findByPk(subscriptionId);
      if (!subscription?.connectionId) return;
      const mqttService = require("./mqttService");
      await mqttService.startConnection(subscription.connectionId);
    } catch (error) {
      logger.warn(
        `[SocketService] Failed to ensure MQTT connection for subscription ${subscriptionId}: ${error?.message || error}`,
      );
    }
  }

  async ensureMqttConnectionByTag(tagId) {
    if (!tagId) return;
    try {
      const tag = await DataMqttTag.findByPk(tagId);
      if (!tag?.subscriptionId) return;
      await this.ensureMqttConnectionBySubscription(tag.subscriptionId);
    } catch (error) {
      logger.warn(
        `[SocketService] Failed to ensure MQTT connection for tag ${tagId}: ${error?.message || error}`,
      );
    }
  }

  /**
   * 鍒濆鍖朣ocket.IO鏈嶅姟鍣?
   * @param {Object} httpServer - HTTP鏈嶅姟鍣ㄥ疄渚?
   */
  initialize(httpServer) {
    this.io = new Server(httpServer, {
      cors: {
        origin: process.env.CORS_ORIGINS || "*",
        methods: ["GET", "POST"],
        credentials: true,
      },
      path: "/socket.io",
      transports: ["websocket", "polling"],
    });

    this.setupEventHandlers();
    logger.info("[SocketService] Socket.IO server initialized");
  }

  /**
   * 璁剧疆浜嬩欢澶勭悊鍣?
   */
  setupEventHandlers() {
    this.io.on("connection", (socket) => {
      const { projectId } = socket.handshake.query;
      logger.info(
        `[SocketService] Client connected: ${socket.id}, project: ${projectId}`,
      );

      // 鍔犲叆椤圭洰鎴块棿
      if (projectId) {
        socket.join(`project:${projectId}`);
        logger.info(
          `[SocketService] Client ${socket.id} joined room: project:${projectId}`,
        );
      }

      // 璁㈤槄MQTT涓婚
      socket.on("mqtt:subscribe", (data) => {
        const { subscriptionId } = data;
        if (subscriptionId) {
          const roomName = `mqtt:subscription:${subscriptionId}`;
          socket.join(roomName);
          this.ensureMqttConnectionBySubscription(subscriptionId);
          console.log(
            `[SocketService] Client ${socket.id} joined room: ${roomName}`,
          );
          logger.info(
            `[SocketService] Client ${socket.id} subscribed to: ${subscriptionId}`,
          );
        }
      });

      // 鍙栨秷璁㈤槄MQTT涓婚
      socket.on("mqtt:unsubscribe", (data) => {
        const { subscriptionId } = data;
        if (subscriptionId) {
          socket.leave(`mqtt:subscription:${subscriptionId}`);
          logger.info(
            `[SocketService] Client ${socket.id} unsubscribed from: ${subscriptionId}`,
          );
        }
      });

      // 设置Tag订阅处理
      this.setupTagSubscription(socket);

      // 数据点订阅
      this.setupDataPointSubscription(socket);

      // 运维监控订阅
      this.setupOpsSubscription(socket);

      this.setupDataPointSubscription(socket);

      // 鏂紑杩炴帴
      socket.on("disconnect", (reason) => {
        logger.info(
          `[SocketService] Client disconnected: ${socket.id}, reason: ${reason}`,
        );
      });

      // 閿欒澶勭悊
      socket.on("error", (error) => {
        logger.error(`[SocketService] Socket error: ${socket.id}`, error);
      });
    });
  }

  /**
   * 骞挎挱MQTT娑堟伅鍒拌闃呰€?
   * @param {string} subscriptionId - 璁㈤槄ID
   * @param {Object} message - 娑堟伅鍐呭
   */
  broadcastMqttMessage(subscriptionId, message) {
    if (!this.io) {
      logger.warn("[SocketService] Socket.IO not initialized");
      return;
    }

    const room = `mqtt:subscription:${subscriptionId}`;
    this.io.to(room).emit("mqtt:message", {
      subscriptionId,
      message: {
        topic: message.topic,
        payload: message.payload,
        qos: message.qos || 0,
        timestamp: message.timestamp || Date.now(),
      },
    });

    logger.debug(`[SocketService] Broadcasted message to room: ${room}`);
  }

  /**
   * 骞挎挱MQTT杩炴帴鐘舵€佸彉鍖?
   * @param {string} projectId - 椤圭洰ID
   * @param {string} connectionId - 杩炴帴ID
   * @param {string} status - 鐘舵€?
   */
  broadcastConnectionStatus(projectId, connectionId, status) {
    if (!this.io) {
      logger.warn("[SocketService] Socket.IO not initialized");
      return;
    }

    const room = `project:${projectId}`;
    this.io.to(room).emit("mqtt:connection:status", {
      connectionId,
      status,
      timestamp: Date.now(),
    });

    logger.debug(
      `[SocketService] Broadcasted connection status to room: ${room}`,
    );
  }

  /**
   * 骞挎挱MQTT璁㈤槄鐘舵€佸彉鍖?
   * @param {string} projectId - 椤圭洰ID
   * @param {string} subscriptionId - 璁㈤槄ID
   * @param {string} status - 鐘舵€?
   */
  broadcastSubscriptionStatus(projectId, subscriptionId, status) {
    if (!this.io) {
      logger.warn("[SocketService] Socket.IO not initialized");
      return;
    }

    const room = `project:${projectId}`;
    this.io.to(room).emit("mqtt:subscription:status", {
      subscriptionId,
      status,
      timestamp: Date.now(),
    });

    logger.debug(
      `[SocketService] Broadcasted subscription status to room: ${room}`,
    );
  }

  /**
   * 骞挎挱Tag鍊兼洿鏂?
   * @param {string} tagId - Tag ID
   * @param {Object} valueData - 鍊兼暟鎹?
   */
  broadcastTagValueUpdate(tagId, valueData) {
    if (!this.io) {
      logger.warn("[SocketService] Socket.IO not initialized");
      return;
    }

    // 骞挎挱鍒拌闃呰Tag鐨勬埧闂?
    const tagRoom = `mqtt:tag:${tagId}`;
    this.io.to(tagRoom).emit("mqtt:tag:value", valueData);

    logger.debug(
      `[SocketService] Broadcasted tag value update to room: ${tagRoom}`,
    );
  }

  /**
   * 璁㈤槄Tag鍊兼洿鏂帮紙瀹㈡埛绔皟鐢級
   * 闇€瑕佸湪setupEventHandlers涓坊鍔犲搴旂殑socket浜嬩欢鐩戝惉
   */
  setupTagSubscription(socket) {
    // 订阅Tag值更新
    socket.on("mqtt:tag:subscribe", (data) => {
      const { tagId } = data;
      if (tagId) {
        socket.join(`mqtt:tag:${tagId}`);
        logger.info(
          `[SocketService] Client ${socket.id} subscribed to tag: ${tagId}`,
        );
      }
    });

    // 鍙栨秷璁㈤槄Tag鍊兼洿鏂?
    socket.on("mqtt:tag:unsubscribe", (data) => {
      const { tagId } = data;
      if (tagId) {
        socket.leave(`mqtt:tag:${tagId}`);
        logger.info(
          `[SocketService] Client ${socket.id} unsubscribed from tag: ${tagId}`,
        );
      }
    });
  }

  /**
   * 订阅数据点值更新
   * @param {object} socket - Socket 实例
   */
  setupDataPointSubscription(socket) {
    socket.on("datapoint:subscribe", (data) => {
      const { projectId, paths } = data || {};
      if (!projectId || !Array.isArray(paths)) {
        return;
      }

      paths
        .map((path) => normalizePath(path))
        .filter(Boolean)
        .forEach((path) => {
          const room = `datapoint:${projectId}:${path}`;
          socket.join(room);
          logger.info(
            `[SocketService] Client ${socket.id} subscribed to datapoint: ${path}`,
          );
        });
    });

    socket.on("datapoint:unsubscribe", (data) => {
      const { projectId, paths } = data || {};
      if (!projectId || !Array.isArray(paths)) {
        return;
      }

      paths
        .map((path) => normalizePath(path))
        .filter(Boolean)
        .forEach((path) => {
          socket.leave(`datapoint:${projectId}:${path}`);
          logger.info(
            `[SocketService] Client ${socket.id} unsubscribed from datapoint: ${path}`,
          );
        });
    });
  }

  /**
   * 订阅运维监控动态
   * @param {object} socket - Socket 实例
   */
  setupOpsSubscription(socket) {
    socket.on("ops:subscribe", (data) => {
      const { tenantId } = data || {};
      if (tenantId) {
        const room = `ops:tenant:${tenantId}`;
        socket.join(room);
        logger.info(
          `[SocketService] Client ${socket.id} subscribed to ops: ${tenantId}`,
        );
      }
    });

    socket.on("ops:unsubscribe", (data) => {
      const { tenantId } = data || {};
      if (tenantId) {
        socket.leave(`ops:tenant:${tenantId}`);
        logger.info(
          `[SocketService] Client ${socket.id} unsubscribed from ops: ${tenantId}`,
        );
      }
    });
  }

  /**
   * 广播节点指标更新
   */
  broadcastNodeMetrics(tenantId, nodeId, metrics) {
    if (!this.io) return;
    const room = `ops:tenant:${tenantId}`;
    this.io
      .to(room)
      .emit("ops:node:metrics", { nodeId, metrics, timestamp: Date.now() });
  }

  /**
   * 广播节点状态变化
   */
  broadcastNodeStatus(tenantId, nodeId, status) {
    if (!this.io) return;
    const room = `ops:tenant:${tenantId}`;
    this.io
      .to(room)
      .emit("ops:node:status", { nodeId, status, timestamp: Date.now() });
  }

  /**
   * 广播工程运行指标更新
   */
  broadcastProjectMetrics(tenantId, nodeId, projectId, metrics) {
    if (!this.io) return;
    const room = `ops:tenant:${tenantId}`;
    this.io.to(room).emit("ops:project:metrics", {
      nodeId,
      projectId,
      metrics,
      timestamp: Date.now(),
    });
  }

  /**
   * 广播新的节点待审核申请
   * @param {string} tenantId - 租户ID
   * @param {object} payload - 待审核申请信息
   */
  broadcastNodePendingRequest(tenantId, payload) {
    if (!this.io) return;
    const room = `ops:tenant:${tenantId}`;
    logger.info(
      `[SocketService] Broadcast ops:node:pending to room ${room}, nodeId=${payload?.nodeId || "-"}, nodeName=${payload?.nodeName || "-"}`,
    );
    this.io.to(room).emit("ops:node:pending", {
      ...payload,
      timestamp: Date.now(),
    });
  }

  /**
   * 广播部署日志
   */
  broadcastDeployLog(tenantId, deploymentId, log) {
    if (!this.io) return;
    const room = `ops:tenant:${tenantId}`;
    this.io.to(room).emit("ops:deploy:log", {
      deploymentId,
      log,
      timestamp: Date.now(),
    });
  }

  /**
   * 璁㈤槄鏁版嵁鐐瑰€兼洿鏂?   * @param {object} socket - Socket 瀹炰緥
   */
  setupDataPointSubscription(socket) {
    socket.on("datapoint:subscribe", (data) => {
      const { projectId, paths, path } = data || {};
      const targetPaths = Array.isArray(paths) ? paths : path ? [path] : [];
      if (!projectId || targetPaths.length === 0) {
        return;
      }

      targetPaths
        .map((item) => normalizePath(item))
        .filter(Boolean)
        .forEach((normalized) => {
          const room = `datapoint:${projectId}:${normalized}`;
          socket.join(room);
          logger.info(
            `[SocketService] Client ${socket.id} subscribed to datapoint: ${normalized}`,
          );
        });
    });

    socket.on("datapoint:unsubscribe", (data) => {
      const { projectId, paths, path } = data || {};
      const targetPaths = Array.isArray(paths) ? paths : path ? [path] : [];
      if (!projectId || targetPaths.length === 0) {
        return;
      }

      targetPaths
        .map((item) => normalizePath(item))
        .filter(Boolean)
        .forEach((normalized) => {
          socket.leave(`datapoint:${projectId}:${normalized}`);
          logger.info(
            `[SocketService] Client ${socket.id} unsubscribed from datapoint: ${normalized}`,
          );
        });
    });
  }

  /**
   * 鍏抽棴Socket.IO鏈嶅姟鍣?
   */
  close() {
    if (this.io) {
      this.io.close();
      this.io = null;
      logger.info("[SocketService] Socket.IO server closed");
    }
  }

  /**
   * 鑾峰彇Socket.IO瀹炰緥
   */
  getIO() {
    return this.io;
  }
}

// 瀵煎嚭鍗曚緥瀹炰緥
const socketService = new SocketService();
module.exports = socketService;
