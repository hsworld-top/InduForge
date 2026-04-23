/**
 * DataService - 数据服务
 * 管理 WebSocket 连接和数据点订阅
 */

import type { Socket } from "socket.io-client";
import { io } from "socket.io-client";
import { EventEmitter } from "../editor-core/utils/EventEmitter.ts";
import {
  ApiRequestError,
  normalizeApiEnvelope,
  type ApiEnvelope,
} from "./types.ts";

/** 与 data/types.js 文档对齐 */
export interface DataServiceOptions {
  baseUrl?: string;
  wsPath?: string;
  autoReconnect?: boolean;
  reconnectDelay?: number;
  maxReconnectAttempts?: number;
}

export interface SubscriptionOptions {
  immediate?: boolean;
  throttle?: number;
}

export interface DatapointValue {
  path: string;
  value: unknown;
  timestamp?: number;
  quality?: string;
}

type ConnectionState = "disconnected" | "connecting" | "connected" | "reconnecting" | "error";

type DatapointCallback = (payload: DatapointValue) => void;

const TRAILING_SLASH_RE = /\/$/;

interface PendingRequest {
  resolve: (result: unknown) => void;
  reject: (error: unknown) => void;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object";
}

function parseLegacySuccessEnvelope<T>(payload: unknown): {
  success: boolean;
  msg: string;
  data: T | undefined;
  reqId: string | undefined;
} | null {
  if (!isRecord(payload) || typeof payload.success !== "boolean") {
    return null;
  }

  const msg =
    typeof payload.message === "string"
      ? payload.message
      : typeof payload.msg === "string"
        ? payload.msg
        : "";
  const reqId = typeof payload.reqId === "string" ? payload.reqId : undefined;

  return {
    success: payload.success,
    msg,
    data: payload.data as T,
    reqId,
  };
}

/**
 * 数据服务类
 */
export class DataService extends EventEmitter {
  _baseUrl: string;
  _wsPath: string;
  _autoReconnect: boolean;
  _reconnectDelay: number;
  _maxReconnectAttempts: number;
  _socket: Socket | null;
  _connectionState: ConnectionState;
  _reconnectAttempts: number;
  _valueCache: Map<string, unknown>;
  _subscriptions: Map<string, Set<DatapointCallback>>;
  _serverSubscriptions: Set<string>;
  _pendingRequests: Map<string, PendingRequest>;
  _requestIdCounter: number;

  constructor(options: DataServiceOptions = {}) {
    super();

    this._baseUrl = options.baseUrl || "";
    this._wsPath = options.wsPath || "/socket.io";
    this._autoReconnect = options.autoReconnect ?? true;
    this._reconnectDelay = options.reconnectDelay ?? 3000;
    this._maxReconnectAttempts = options.maxReconnectAttempts ?? 10;
    this._socket = null;
    this._connectionState = "disconnected";
    this._reconnectAttempts = 0;
    this._valueCache = new Map();
    this._subscriptions = new Map();
    this._serverSubscriptions = new Set();
    this._pendingRequests = new Map();
    this._requestIdCounter = 0;
  }

  // ==================== 连接管理 ====================

  /**
   * 连接到服务器
   */
  async connect(url?: string): Promise<void> {
    if (this._connectionState === "connected") {
      return;
    }

    this._setConnectionState("connecting");

    const serverUrl = url || this._baseUrl || window.location.origin;
    const finalUrl = serverUrl.replace(TRAILING_SLASH_RE, "");

    return new Promise((resolve, reject) => {
      const queryParams = ((): Record<string, string> | undefined => {
        try {
          const urlObj = new URL(finalUrl);
          const params: Record<string, string> = {};
          for (const [key, value] of urlObj.searchParams.entries()) {
            params[key] = value;
          }
          return params;
        } catch {
          return undefined;
        }
      })();

      this._socket = io(finalUrl, {
        path: this._wsPath,
        transports: ["websocket", "polling"],
        reconnection: this._autoReconnect,
        reconnectionDelay: this._reconnectDelay,
        reconnectionAttempts: this._maxReconnectAttempts,
        ...(queryParams !== undefined ? { query: queryParams } : {}),
      });

      // 连接成功
      this._socket.on("connect", () => {
        this._setConnectionState("connected");
        this._reconnectAttempts = 0;

        // 重新订阅
        this._resubscribeAll();

        resolve();
      });

      // 连接错误
      this._socket.on("connect_error", (error) => {
        console.error("WebSocket connection error:", error);
        this._setConnectionState("error");
        reject(error);
      });

      // 断开连接
      this._socket.on("disconnect", (reason) => {
        console.warn("WebSocket disconnected:", reason);
        this._setConnectionState("disconnected");
        this.emit("disconnected", { reason });
      });

      // 重连中
      this._socket.on("reconnecting", (attemptNumber) => {
        this._reconnectAttempts = attemptNumber;
        this._setConnectionState("reconnecting");
        this.emit("reconnecting", { attempt: attemptNumber });
      });

      // 重连成功
      this._socket.on("reconnect", () => {
        this._setConnectionState("connected");
        this._reconnectAttempts = 0;
        this._resubscribeAll();
        this.emit("reconnected");
      });

      // 重连失败
      this._socket.on("reconnect_failed", () => {
        this._setConnectionState("error");
        this.emit("reconnectFailed");
      });

      // 数据点值更新
      this._socket.on("datapoint:value", (data) => {
        this._handleValueUpdate(data);
      });

      // 数据点批量值更新
      this._socket.on("datapoint:values", (data) => {
        if (Array.isArray(data)) {
          for (const item of data) {
            this._handleValueUpdate(item);
          }
        }
      });

      // 数据点状态更新
      this._socket.on("datapoint:status", (data) => {
        this.emit("statusUpdate", data);
      });

      // 响应处理
      this._socket.on("response", (data) => {
        this._handleResponse(data);
      });

      // 超时处理
      setTimeout(() => {
        if (this._connectionState !== "connected") {
          reject(new Error("Connection timeout"));
        }
      }, 10000);
    });
  }

  /**
   * 断开连接
   */
  disconnect(): void {
    if (this._socket) {
      this._socket.disconnect();
      this._socket = null;
    }

    this._setConnectionState("disconnected");
    this._serverSubscriptions.clear();
  }

  /**
   * 获取连接状态
   */
  getConnectionState(): ConnectionState {
    return this._connectionState;
  }

  /**
   * 检查是否已连接
   */
  isConnected(): boolean {
    return this._connectionState === "connected";
  }

  /**
   * 设置连接状态
   * @private
   */
  _setConnectionState(state: ConnectionState): void {
    const oldState = this._connectionState;
    this._connectionState = state;

    if (oldState !== state) {
      this.emit("connectionStateChange", { oldState, newState: state });
    }
  }

  // ==================== 订阅管理 ====================

  /**
   * 订阅数据点
   */
  subscribe(
    path: string,
    callback: DatapointCallback,
    options: SubscriptionOptions = {},
  ): () => void {
    // 添加本地订阅
    if (!this._subscriptions.has(path)) {
      this._subscriptions.set(path, new Set());
    }
    this._subscriptions.get(path)!.add(callback);

    // 向服务器订阅
    if (!this._serverSubscriptions.has(path)) {
      this._subscribeToServer(path);
    }

    // 如果有缓存值且需要立即通知
    if (options.immediate !== false && this._valueCache.has(path)) {
      const value = this._valueCache.get(path);
      callback({
        path,
        value,
        timestamp: Date.now(),
      });
    }

    // 返回取消订阅函数
    return () => {
      this.unsubscribe(path, callback);
    };
  }

  /**
   * 批量订阅
   */
  subscribeMany(
    paths: string[],
    callback: DatapointCallback,
    options: SubscriptionOptions = {},
  ): () => void {
    const unsubscribes = paths.map((path) => this.subscribe(path, callback, options));

    return () => {
      unsubscribes.forEach((unsub) => unsub());
    };
  }

  /**
   * 取消订阅
   */
  unsubscribe(path: string, callback?: DatapointCallback): void {
    const callbacks = this._subscriptions.get(path);
    if (!callbacks) return;

    if (callback) {
      callbacks.delete(callback);
    } else {
      callbacks.clear();
    }

    // 如果没有订阅者了，向服务器取消订阅
    if (callbacks.size === 0) {
      this._subscriptions.delete(path);
      this._unsubscribeFromServer(path);
    }
  }

  /**
   * 取消所有订阅
   */
  unsubscribeAll(): void {
    for (const path of this._subscriptions.keys()) {
      this._unsubscribeFromServer(path);
    }
    this._subscriptions.clear();
    this._serverSubscriptions.clear();
  }

  /**
   * 向服务器订阅
   * @private
   */
  _subscribeToServer(path: string): void {
    if (!this._socket?.connected) {
      this._serverSubscriptions.add(path); // 标记待订阅
      return;
    }

    this._socket.emit("datapoint:subscribe", { path });
    this._serverSubscriptions.add(path);
  }

  /**
   * 向服务器取消订阅
   * @private
   */
  _unsubscribeFromServer(path: string): void {
    if (!this._socket?.connected) {
      this._serverSubscriptions.delete(path);
      return;
    }

    this._socket.emit("datapoint:unsubscribe", { path });
    this._serverSubscriptions.delete(path);
  }

  /**
   * 重新订阅所有
   * @private
   */
  _resubscribeAll(): void {
    const paths = Array.from(this._subscriptions.keys());
    if (paths.length === 0) return;

    if (this._socket?.connected) {
      this._socket.emit("datapoint:subscribe:batch", { paths });
      for (const path of paths) {
        this._serverSubscriptions.add(path);
      }
    }
  }

  // ==================== 值管理 ====================

  /**
   * 获取缓存的值
   */
  getValue(path: string): unknown | undefined {
    return this._valueCache.get(path);
  }

  /**
   * 批量获取缓存的值
   */
  getValues(paths: string[]): Map<string, unknown> {
    const result = new Map();
    for (const path of paths) {
      if (this._valueCache.has(path)) {
        result.set(path, this._valueCache.get(path));
      }
    }
    return result;
  }

  /**
   * 处理值更新
   * @private
   */
  _handleValueUpdate(data: DatapointValue): void {
    const { path, value, timestamp } = data;

    // 更新缓存
    this._valueCache.set(path, value);

    // 通知订阅者
    const callbacks = this._subscriptions.get(path);
    if (callbacks) {
      const payload: DatapointValue =
        timestamp !== undefined ? { path, value, timestamp } : { path, value };
      for (const callback of callbacks) {
        try {
          callback(payload);
        } catch (error) {
          console.error("Subscription callback error:", error);
        }
      }
    }

    // 触发全局事件
    this.emit("valueUpdate", data);
  }

  // ==================== API 请求 ====================

  /**
   * 统一解析 fetch 响应：
   * 1. 2xx + code===0 => 成功
   * 2. 2xx + code!==0 => 业务失败（保留 code/msg/reqId）
   * 3. 非 2xx => 技术失败（保留 HTTP status 与 msg）
   */
  private async _requestApiEnvelope<T>(
    input: RequestInfo | URL,
    init: RequestInit | undefined,
    fallbackMessage: string,
  ): Promise<ApiEnvelope<T>> {
    const response = await fetch(input, init);
    let payload: unknown = null;

    try {
      payload = await response.json();
    } catch {
      payload = null;
    }

    const envelope = normalizeApiEnvelope<T>(payload);

    if (!response.ok) {
      throw new ApiRequestError({
        code: envelope?.code,
        msg: envelope?.msg || `HTTP ${response.status}`,
        reqId: envelope?.reqId,
        status: response.status,
        data: envelope?.data,
        isBusinessError: false,
      });
    }

    if (envelope) {
      if (envelope.code !== 0) {
        throw new ApiRequestError({
          code: envelope.code,
          msg: envelope.msg || fallbackMessage,
          reqId: envelope.reqId,
          status: response.status,
          data: envelope.data,
          isBusinessError: true,
        });
      }

      return envelope;
    }

    const legacyEnvelope = parseLegacySuccessEnvelope<T>(payload);
    if (legacyEnvelope) {
      if (!legacyEnvelope.success) {
        throw new ApiRequestError({
          msg: legacyEnvelope.msg || fallbackMessage,
          reqId: legacyEnvelope.reqId,
          status: response.status,
          data: legacyEnvelope.data,
          isBusinessError: true,
        });
      }

      return {
        code: 0,
        msg: legacyEnvelope.msg,
        ...(legacyEnvelope.data !== undefined ? { data: legacyEnvelope.data } : {}),
        ...(legacyEnvelope.reqId ? { reqId: legacyEnvelope.reqId } : {}),
      };
    }

    // 2xx 响应若既不是新包络，也不是可识别旧包络，按异常处理，避免误判为成功。
    throw new ApiRequestError({
      msg: `${fallbackMessage}: 响应格式不受支持`,
      status: response.status,
      data: payload,
      isBusinessError: false,
    });
  }

  /**
   * 获取数据点当前值（HTTP API）
   */
  async fetchValue(path: string): Promise<unknown> {
    const response = await this._requestApiEnvelope<{ value?: unknown }>(
      `${this._baseUrl}/api/v1/datapoints/${encodeURIComponent(path)}/value`,
      undefined,
      "Failed to fetch value",
    );

    const value = response.data?.value;
    this._valueCache.set(path, value);
    return value;
  }

  /**
   * 批量获取数据点当前值（HTTP API）
   */
  async fetchValues(paths: string[]): Promise<Map<string, unknown>> {
    const response = await this._requestApiEnvelope<Array<{ path: string; value: unknown }>>(
      `${this._baseUrl}/api/v1/datapoints/values`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ paths }),
      },
      "Failed to fetch values",
    );

    const values = new Map<string, unknown>();
    const items = Array.isArray(response.data) ? response.data : [];

    for (const item of items) {
      values.set(item.path, item.value);
      this._valueCache.set(item.path, item.value);
    }

    return values;
  }

  /**
   * 写入数据点值
   */
  async writeValue(path: string, value: unknown): Promise<void> {
    await this._requestApiEnvelope<unknown>(
      `${this._baseUrl}/api/v1/datapoints/${encodeURIComponent(path)}/value`,
      {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ value }),
      },
      "Failed to write value",
    );
  }

  // ==================== Socket 请求 ====================

  /**
   * 发送请求并等待响应
   */
  async request(event: string, data: Record<string, unknown>, timeout = 5000): Promise<unknown> {
    if (!this._socket?.connected) {
      throw new Error("Not connected");
    }

    const requestId = `req_${++this._requestIdCounter}`;

    return new Promise((resolve, reject) => {
      // 设置超时
      const timer = setTimeout(() => {
        this._pendingRequests.delete(requestId);
        reject(new Error("Request timeout"));
      }, timeout);

      // 保存待处理请求
      this._pendingRequests.set(requestId, {
        resolve: (result) => {
          clearTimeout(timer);
          this._pendingRequests.delete(requestId);
          resolve(result);
        },
        reject: (error) => {
          clearTimeout(timer);
          this._pendingRequests.delete(requestId);
          reject(error);
        },
      });

      // 发送请求
      this._socket!.emit(event, { ...data, requestId });
    });
  }

  /**
   * 处理响应
   * @private
   */
  _handleResponse(data: {
    requestId?: string;
    success?: boolean;
    result?: unknown;
    error?: string;
  }): void {
    const { requestId, success, result, error } = data;

    if (requestId == null) return;
    const pending = this._pendingRequests.get(requestId);
    if (!pending) return;

    if (success) {
      pending.resolve(result);
    } else {
      pending.reject(new Error(error || "Request failed"));
    }
  }

  // ==================== 清理 ====================

  /**
   * 清除值缓存
   */
  clearCache(): void {
    this._valueCache.clear();
  }

  /**
   * 销毁服务
   */
  destroy(): void {
    this.disconnect();
    this._subscriptions.clear();
    this._valueCache.clear();
    this._pendingRequests.clear();
    this.removeAllListeners();
  }
}

/**
 * 创建数据服务实例
 */
export function createDataService(options?: DataServiceOptions): DataService {
  return new DataService(options);
}

export default DataService;
