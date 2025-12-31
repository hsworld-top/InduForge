const mqtt = require("mqtt");
const BaseProtocol = require("./BaseProtocol");
const { logger } = require("../../utils/logger");

/**
 * MQTT 协议实现
 * 支持 MQTT 3.1.1 和 MQTT 5.0
 */
class MqttProtocol extends BaseProtocol {
  constructor(config) {
    super(config);
    this.client = null;
    this.subscriptions = new Map(); // topic -> { qos, callback }
    this.messageHandlers = new Set(); // 消息处理器集合
    this.isConnecting = false;
    this.isConnected = false;
    this.reconnectAttempts = 0;
  }

  /**
   * 构建连接URL
   * @private
   */
  _buildConnectionUrl() {
    const { protocol = "mqtt", brokerUrl, port } = this.config;

    // 如果 brokerUrl 已经包含完整 URL，直接使用
    if (
      brokerUrl.startsWith("mqtt://") ||
      brokerUrl.startsWith("mqtts://") ||
      brokerUrl.startsWith("ws://") ||
      brokerUrl.startsWith("wss://")
    ) {
      return brokerUrl;
    }

    // 否则构建 URL
    const defaultPort =
      protocol === "mqtts" || protocol === "wss" ? 8883 : 1883;
    const actualPort = port || defaultPort;
    return `${protocol}://${brokerUrl}:${actualPort}`;
  }

  /**
   * 构建连接选项
   * @private
   */
  _buildConnectionOptions() {
    const {
      clientId,
      username,
      password,
      keepalive = 60,
      cleanSession = true,
      reconnectPeriod = 5000,
      connectTimeout = 30000,
      will,
      sslConfig,
    } = this.config;

    const options = {
      clientId:
        clientId || `induforge_${Math.random().toString(16).substr(2, 8)}`,
      keepalive,
      clean: cleanSession,
      reconnectPeriod,
      connectTimeout,
    };

    // 添加认证信息
    if (username) {
      options.username = username;
    }
    if (password) {
      options.password = password;
    }

    // 添加遗嘱消息（仅当 topic 不为空时）
    if (will && will.topic && will.topic.trim() !== "") {
      options.will = will;
    }

    // 添加 SSL/TLS 配置（仅当有实际配置内容时）
    if (sslConfig) {
      if (sslConfig.ca && sslConfig.ca.trim() !== "") {
        options.ca = sslConfig.ca;
      }
      if (sslConfig.cert && sslConfig.cert.trim() !== "") {
        options.cert = sslConfig.cert;
      }
      if (sslConfig.key && sslConfig.key.trim() !== "") {
        options.key = sslConfig.key;
      }
      // rejectUnauthorized 对于 mqtts/wss 协议才有意义
      if (sslConfig.rejectUnauthorized !== undefined) {
        options.rejectUnauthorized = sslConfig.rejectUnauthorized;
      }
    }

    return options;
  }

  /**
   * 测试连接
   * @returns {Promise<boolean>}
   */
  async testConnection() {
    return new Promise((resolve, reject) => {
      const url = this._buildConnectionUrl();
      const options = {
        ...this._buildConnectionOptions(),
        reconnectPeriod: 0, // 测试时不重连
      };

      logger.info(
        `[MqttProtocol] Testing connection to ${url}, ${JSON.stringify(
          options
        )}`
      );

      const testClient = mqtt.connect(url, options);

      const timeout = setTimeout(() => {
        testClient.end(true);
        reject(new Error("Connection timeout"));
      }, options.connectTimeout);

      testClient.on("connect", () => {
        clearTimeout(timeout);
        logger.info("[MqttProtocol] Test connection successful");
        testClient.end();
        resolve(true);
      });

      testClient.on("error", (err) => {
        clearTimeout(timeout);
        logger.error("[MqttProtocol] Test connection failed:", err);
        testClient.end(true);
        reject(err);
      });
    });
  }

  /**
   * 建立连接
   * @returns {Promise<Object>}
   */
  async connect() {
    if (this.isConnected) {
      logger.warn("[MqttProtocol] Already connected");
      return this.client;
    }

    if (this.isConnecting) {
      logger.warn("[MqttProtocol] Connection in progress");
      throw new Error("Connection already in progress");
    }

    return new Promise((resolve, reject) => {
      this.isConnecting = true;
      const url = this._buildConnectionUrl();
      const options = this._buildConnectionOptions();

      logger.info(
        `[MqttProtocol] Connecting to ${url}, ${JSON.stringify(options)}`
      );

      this.client = mqtt.connect(url, options);

      const timeout = setTimeout(() => {
        this.isConnecting = false;
        this.client.end(true);
        reject(new Error("Connection timeout"));
      }, options.connectTimeout);

      // 连接成功
      this.client.on("connect", () => {
        clearTimeout(timeout);
        this.isConnecting = false;
        this.isConnected = true;
        this.reconnectAttempts = 0;
        logger.info("[MqttProtocol] Connected successfully");
        resolve(this.client);
      });

      // 连接错误
      this.client.on("error", (err) => {
        logger.error("[MqttProtocol] Connection error:", err);
        if (this.isConnecting) {
          clearTimeout(timeout);
          this.isConnecting = false;
          reject(err);
        }
      });

      // 断线事件
      this.client.on("offline", () => {
        this.isConnected = false;
        logger.warn("[MqttProtocol] Connection offline");
      });

      // 重连事件
      this.client.on("reconnect", () => {
        this.reconnectAttempts++;
        logger.info(
          `[MqttProtocol] Reconnecting... (attempt ${this.reconnectAttempts})`
        );
      });

      // 关闭事件
      this.client.on("close", () => {
        this.isConnected = false;
        logger.info("[MqttProtocol] Connection closed");
      });

      // 接收消息
      this.client.on("message", (topic, payload, packet) => {
        this._handleMessage(topic, payload, packet);
      });
    });
  }

  /**
   * 断开连接
   * @param {Object} connection - 连接对象（可选，用于兼容接口）
   */
  async disconnect(connection = null) {
    if (!this.client) {
      logger.warn("[MqttProtocol] No active connection to disconnect");
      return;
    }

    return new Promise((resolve) => {
      logger.info("[MqttProtocol] Disconnecting...");

      this.client.end(false, {}, () => {
        this.isConnected = false;
        this.isConnecting = false;
        this.client = null;
        this.subscriptions.clear();
        logger.info("[MqttProtocol] Disconnected successfully");
        resolve();
      });
    });
  }

  /**
   * 订阅主题
   * @param {string|string[]} topics - 主题或主题数组
   * @param {Object} options - 订阅选项 { qos }
   * @returns {Promise<Object>}
   */
  async subscribe(topics, options = {}, messageHandler = null) {
    if (!this.isConnected || !this.client) {
      throw new Error("Not connected to MQTT broker");
    }

    const { qos = 0 } = options;
    const topicsArray = Array.isArray(topics) ? topics : [topics];

    return new Promise((resolve, reject) => {
      this.client.subscribe(topicsArray, { qos }, (err, granted) => {
        if (err) {
          logger.error("[MqttProtocol] Subscribe error:", err);
          reject(err);
          return;
        }

        // 如果提供了消息处理器，添加到处理器集合
        if (messageHandler && typeof messageHandler === "function") {
          this.addMessageHandler(messageHandler);
          logger.info(`[MqttProtocol] Added message handler for subscription`);
        }

        // 记录订阅信息
        granted.forEach(({ topic, qos }) => {
          this.subscriptions.set(topic, { qos });
          logger.info(`[MqttProtocol] Subscribed to ${topic} with QoS ${qos}`);
        });

        resolve(granted);
      });
    });
  }

  /**
   * 取消订阅
   * @param {string|string[]} topics - 主题或主题数组
   * @returns {Promise<void>}
   */
  async unsubscribe(topics) {
    if (!this.isConnected || !this.client) {
      throw new Error("Not connected to MQTT broker");
    }

    const topicsArray = Array.isArray(topics) ? topics : [topics];

    return new Promise((resolve, reject) => {
      this.client.unsubscribe(topicsArray, (err) => {
        if (err) {
          logger.error("[MqttProtocol] Unsubscribe error:", err);
          reject(err);
          return;
        }

        // 删除订阅记录
        topicsArray.forEach((topic) => {
          this.subscriptions.delete(topic);
          logger.info(`[MqttProtocol] Unsubscribed from ${topic}`);
        });

        resolve();
      });
    });
  }

  /**
   * 发布消息
   * @param {string} topic - 主题
   * @param {string|Buffer} payload - 消息内容
   * @param {Object} options - 发布选项 { qos, retain }
   * @returns {Promise<void>}
   */
  async publish(topic, payload, options = {}) {
    if (!this.isConnected || !this.client) {
      throw new Error("Not connected to MQTT broker");
    }

    const { qos = 0, retain = false } = options;

    return new Promise((resolve, reject) => {
      this.client.publish(topic, payload, { qos, retain }, (err) => {
        if (err) {
          logger.error("[MqttProtocol] Publish error:", err);
          reject(err);
          return;
        }

        logger.info(`[MqttProtocol] Published to ${topic}`);
        resolve();
      });
    });
  }

  /**
   * 添加消息处理器
   * @param {Function} handler - 处理函数 (topic, payload, packet) => void
   */
  addMessageHandler(handler) {
    if (typeof handler === "function") {
      this.messageHandlers.add(handler);
    }
  }

  /**
   * 移除消息处理器
   * @param {Function} handler - 处理函数
   */
  removeMessageHandler(handler) {
    this.messageHandlers.delete(handler);
  }

  /**
   * 处理接收到的消息
   * @private
   */
  _handleMessage(topic, payload, packet) {
    const message = {
      topic,
      payload: payload.toString(),
      qos: packet.qos,
      retain: packet.retain,
      timestamp: Date.now(),
    };

    // 调用所有注册的消息处理器
    this.messageHandlers.forEach((handler) => {
      try {
        handler(message);
      } catch (err) {
        logger.error("[MqttProtocol] Message handler error:", err);
      }
    });
  }

  /**
   * 获取连接状态
   * @returns {Object}
   */
  getStatus() {
    return {
      connected: this.isConnected,
      connecting: this.isConnecting,
      reconnectAttempts: this.reconnectAttempts,
      subscriptions: Array.from(this.subscriptions.keys()),
    };
  }

  /**
   * 发送数据（别名方法，兼容基类接口）
   */
  async send(connection, data) {
    const { topic, payload, options } = data;
    return this.publish(topic, payload, options);
  }

  /**
   * 接收数据（别名方法，兼容基类接口）
   */
  async receive(connection, callback) {
    this.addMessageHandler(callback);
  }
}

module.exports = MqttProtocol;
