/**
 * MessageHandler - DataCenter消息处理器
 *
 * 处理来自Designer的消息请求
 */
import dataAPI from "@/api/data.api";
import { Storage } from "@/utils/storage";
import { STORAGE_KEYS } from "@/constants";
import {
  applyThemeToDocument,
  isTrustedHostMessage,
} from "../runtime/host-bootstrap.js";

class MessageHandler {
  constructor() {
    this.subscriptions = new Map();
    this.pollingTimers = new Map();

    // 绑定消息处理器
    this.handleMessage = this.handleMessage.bind(this);
    window.addEventListener("message", this.handleMessage);
  }

  /**
   * 处理接收到的消息
   * @param {MessageEvent} event - 消息事件
   */
  async handleMessage(event) {
    const data = event.data;
    if (!data || typeof data !== "object") {
      return;
    }

    // 宿主广播消息必须命中 trusted origin/source，避免其他窗口伪造主题同步。
    if (data.type === "THEME_UPDATE" && data.theme) {
      if (!isTrustedHostMessage(event)) {
        return;
      }

      this.applyTheme(data.theme);
      return;
    }

    const replyTarget = this.createReplyTarget(event.source, event.origin);

    // 处理握手
    if (data.type === "HANDSHAKE" && data.source === "designer") {
      if (!replyTarget) {
        return;
      }

      this.sendMessage(replyTarget, {
        type: "HANDSHAKE_ACK",
        source: "datacenter",
      });
      console.log("DataCenter: Connected to Designer");
      return;
    }

    // 处理请求
    if (data.type === "REQUEST" && data.requestId) {
      if (!replyTarget) {
        return;
      }

      try {
        const result = await this.handleRequest(
          data.action,
          data.payload,
          replyTarget,
        );
        this.sendMessage(replyTarget, {
          type: "RESPONSE",
          requestId: data.requestId,
          payload: result,
        });
      } catch (error) {
        this.sendMessage(replyTarget, {
          type: "RESPONSE",
          requestId: data.requestId,
          error: error.message,
        });
      }
    }
  }

  /**
   * 应用主题并缓存设置。
   * @param {string} theme - 主题
   */
  applyTheme(theme) {
    if (!["light", "dark"].includes(theme)) {
      return;
    }
    Storage.set(STORAGE_KEYS.THEME, theme);
    applyThemeToDocument(theme);
  }

  /**
   * 处理具体请求
   * @param {string} action - 动作类型
   * @param {Object} payload - 请求数据
   * @returns {Promise<any>}
   */
  async handleRequest(action, payload, replyTarget = null) {
    switch (action) {
      case "GET_CONNECTIONS":
        return await this.getConnections(payload);

      case "GET_QUERIES":
        return await this.getQueries(payload);

      case "EXECUTE_QUERY":
        return await this.executeQuery(payload);

      case "EXECUTE_SQL":
        return await this.executeSql(payload);

      case "SUBSCRIBE":
        return await this.subscribe(payload, replyTarget);

      case "UNSUBSCRIBE":
        return await this.unsubscribe(payload);

      case "GET_CONNECTION_TABLES":
        return await this.getConnectionTables(payload);

      case "GET_TABLE_DATA":
        return await this.getTableData(payload);

      default:
        throw new Error(`Unknown action: ${action}`);
    }
  }

  /**
   * 获取数据连接列表
   * @param {Object} payload - { projectId }
   * @returns {Promise<Object>}
   */
  async getConnections({ projectId }) {
    const response = await dataAPI.getConnections(projectId);
    return {
      connections: response.data?.connections || response.data || [],
    };
  }

  /**
   * 获取数据查询列表
   * @param {Object} payload - { projectId, connectionId }
   * @returns {Promise<Object>}
   */
  async getQueries({ projectId, connectionId }) {
    const params = connectionId ? { connectionId } : {};
    const response = await dataAPI.getQueries(projectId, params);
    return {
      queries: response.data?.queries || response.data || [],
    };
  }

  /**
   * 执行数据查询
   * @param {Object} payload - { projectId, queryId, parameters }
   * @returns {Promise<Object>}
   */
  async executeQuery({ projectId: _projectId, queryId, parameters }) {
    const response = await dataAPI.executeQuery(queryId, parameters);
    return response.data || response;
  }

  /**
   * 执行SQL查询
   * @param {Object} payload - { projectId, connectionId, sql, parameters }
   * @returns {Promise<Object>}
   */
  async executeSql({ projectId, connectionId, sql, parameters }) {
    const response = await dataAPI.executeSql(
      projectId,
      connectionId,
      sql,
      parameters,
    );
    return response.data || response;
  }

  /**
   * 订阅数据查询（轮询模式）
   * @param {Object} payload - { subscriptionId, projectId, queryId, parameters, interval }
   * @returns {Promise<Object>}
   */
  async subscribe(
    { subscriptionId, projectId, queryId, parameters, interval },
    replyTarget,
  ) {
    if (!replyTarget) {
      throw new Error("订阅请求缺少可回复的消息来源");
    }

    // 立即执行一次
    const initialData = await this.executeQuery({
      projectId,
      queryId,
      parameters,
    });

    // 发送初始数据
    this.sendMessage(replyTarget, {
      type: "SUBSCRIPTION_DATA",
      subscriptionId,
      payload: initialData,
    });

    // 设置轮询定时器
    const timer = setInterval(async () => {
      try {
        const data = await this.executeQuery({
          projectId,
          queryId,
          parameters,
        });
        this.sendMessage(replyTarget, {
          type: "SUBSCRIPTION_DATA",
          subscriptionId,
          payload: data,
        });
      } catch (error) {
        this.sendMessage(replyTarget, {
          type: "SUBSCRIPTION_ERROR",
          subscriptionId,
          error: error.message,
        });
      }
    }, interval);

    this.pollingTimers.set(subscriptionId, timer);
    this.subscriptions.set(subscriptionId, {
      projectId,
      queryId,
      parameters,
      interval,
      replyTarget,
    });

    return { success: true };
  }

  /**
   * 取消订阅
   * @param {Object} payload - { subscriptionId }
   * @returns {Promise<Object>}
   */
  async unsubscribe({ subscriptionId }) {
    const timer = this.pollingTimers.get(subscriptionId);
    if (timer) {
      clearInterval(timer);
      this.pollingTimers.delete(subscriptionId);
    }

    this.subscriptions.delete(subscriptionId);

    return { success: true };
  }

  /**
   * 获取连接的表列表
   * @param {Object} payload - { projectId, connectionId }
   * @returns {Promise<Object>}
   */
  async getConnectionTables({ projectId, connectionId }) {
    const response = await dataAPI.getConnectionTables(projectId, connectionId);
    return {
      tables: response.data?.tables || response.data || [],
    };
  }

  /**
   * 获取表数据
   * @param {Object} payload - { projectId, connectionId, tableName, params }
   * @returns {Promise<Object>}
   */
  async getTableData({ projectId, connectionId, tableName, params }) {
    const response = await dataAPI.getTableData(
      projectId,
      connectionId,
      tableName,
      params,
    );
    return response.data || response;
  }

  /**
   * 发送消息
   * 将当前消息来源规范化成回复目标。
   * Designer 与 DataCenter 的桥接使用“谁发来就回给谁”的链路，不依赖父窗口身份。
   * @param {MessageEventSource|null} source - 消息来源
   * @param {string} origin - 消息 origin
   * @returns {{source: MessageEventSource, origin: string}|null} 回复目标
   */
  createReplyTarget(source, origin) {
    if (!source?.postMessage || typeof origin !== "string" || !origin) {
      return null;
    }

    return {
      source,
      origin,
    };
  }

  /**
   * @param {Object} message - 消息对象
   */
  sendMessage(target, message) {
    if (target?.source?.postMessage && target.origin) {
      target.source.postMessage(message, target.origin);
    }
  }

  /**
   * 销毁处理器
   */
  destroy() {
    window.removeEventListener("message", this.handleMessage);

    // 清除所有定时器
    this.pollingTimers.forEach((timer) => clearInterval(timer));
    this.pollingTimers.clear();
    this.subscriptions.clear();
  }
}

// 创建单例
let instance = null;

export function initMessageHandler() {
  if (!instance) {
    instance = new MessageHandler();
  }
  return instance;
}

export function getMessageHandler() {
  return instance;
}

export default MessageHandler;
