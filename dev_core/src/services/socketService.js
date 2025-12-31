const { Server } = require("socket.io");
const { logger } = require("../utils/logger");

/**
 * Socket.IO 服务
 * 管理WebSocket连接，广播MQTT消息
 */
class SocketService {
  constructor() {
    this.io = null;
    this.connections = new Map(); // userId -> socket
  }

  /**
   * 初始化Socket.IO服务器
   * @param {Object} httpServer - HTTP服务器实例
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
   * 设置事件处理器
   */
  setupEventHandlers() {
    this.io.on("connection", (socket) => {
      const { projectId } = socket.handshake.query;
      logger.info(
        `[SocketService] Client connected: ${socket.id}, project: ${projectId}`
      );

      // 加入项目房间
      if (projectId) {
        socket.join(`project:${projectId}`);
        logger.info(
          `[SocketService] Client ${socket.id} joined room: project:${projectId}`
        );
      }

      // 订阅MQTT主题
      socket.on("mqtt:subscribe", (data) => {
        const { subscriptionId } = data;
        if (subscriptionId) {
          const roomName = `mqtt:subscription:${subscriptionId}`;
          socket.join(roomName);
          console.log(
            `[SocketService] Client ${socket.id} joined room: ${roomName}`
          );
          logger.info(
            `[SocketService] Client ${socket.id} subscribed to: ${subscriptionId}`
          );
        }
      });

      // 取消订阅MQTT主题
      socket.on("mqtt:unsubscribe", (data) => {
        const { subscriptionId } = data;
        if (subscriptionId) {
          socket.leave(`mqtt:subscription:${subscriptionId}`);
          logger.info(
            `[SocketService] Client ${socket.id} unsubscribed from: ${subscriptionId}`
          );
        }
      });

      // 设置Tag订阅处理
      this.setupTagSubscription(socket);

      // 断开连接
      socket.on("disconnect", (reason) => {
        logger.info(
          `[SocketService] Client disconnected: ${socket.id}, reason: ${reason}`
        );
      });

      // 错误处理
      socket.on("error", (error) => {
        logger.error(`[SocketService] Socket error: ${socket.id}`, error);
      });
    });
  }

  /**
   * 广播MQTT消息到订阅者
   * @param {string} subscriptionId - 订阅ID
   * @param {Object} message - 消息内容
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
   * 广播MQTT连接状态变化
   * @param {string} projectId - 项目ID
   * @param {string} connectionId - 连接ID
   * @param {string} status - 状态
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
      `[SocketService] Broadcasted connection status to room: ${room}`
    );
  }

  /**
   * 广播MQTT订阅状态变化
   * @param {string} projectId - 项目ID
   * @param {string} subscriptionId - 订阅ID
   * @param {string} status - 状态
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
      `[SocketService] Broadcasted subscription status to room: ${room}`
    );
  }

  /**
   * 广播Tag值更新
   * @param {string} tagId - Tag ID
   * @param {Object} valueData - 值数据
   */
  broadcastTagValueUpdate(tagId, valueData) {
    if (!this.io) {
      logger.warn("[SocketService] Socket.IO not initialized");
      return;
    }

    // 广播到订阅该Tag的房间
    const tagRoom = `mqtt:tag:${tagId}`;
    this.io.to(tagRoom).emit("mqtt:tag:value", valueData);

    logger.debug(
      `[SocketService] Broadcasted tag value update to room: ${tagRoom}`
    );
  }

  /**
   * 订阅Tag值更新（客户端调用）
   * 需要在setupEventHandlers中添加对应的socket事件监听
   */
  setupTagSubscription(socket) {
    // 订阅Tag值更新
    socket.on("mqtt:tag:subscribe", (data) => {
      const { tagId } = data;
      if (tagId) {
        socket.join(`mqtt:tag:${tagId}`);
        logger.info(
          `[SocketService] Client ${socket.id} subscribed to tag: ${tagId}`
        );
      }
    });

    // 取消订阅Tag值更新
    socket.on("mqtt:tag:unsubscribe", (data) => {
      const { tagId } = data;
      if (tagId) {
        socket.leave(`mqtt:tag:${tagId}`);
        logger.info(
          `[SocketService] Client ${socket.id} unsubscribed from tag: ${tagId}`
        );
      }
    });
  }

  /**
   * 关闭Socket.IO服务器
   */
  close() {
    if (this.io) {
      this.io.close();
      this.io = null;
      logger.info("[SocketService] Socket.IO server closed");
    }
  }

  /**
   * 获取Socket.IO实例
   */
  getIO() {
    return this.io;
  }
}

// 导出单例实例
const socketService = new SocketService();
module.exports = socketService;
