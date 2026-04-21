// @ts-nocheck
const DEFAULT_SOCKET_URL = "http://localhost:19601";
const DEFAULT_SOCKET_PATH = "/socket.io/";

/**
 * 构建共享连接键，确保同一 preview session 只复用同一条底层 socket。
 * token 也纳入键值，避免登录态变化后沿用旧连接。
 *
 * @param {{ token: string, projectId: string, previewSessionId: string }} params
 * @returns {string}
 */
export function buildMqttSocketSharedKey({
  token,
  projectId,
  previewSessionId,
}) {
  return [token, projectId, previewSessionId].join("::");
}

/**
 * 创建 datacenter 侧的 MQTT preview 共享连接注册表。
 * 该注册表负责：
 * 1. 按 token + projectId + previewSessionId 复用底层 socket
 * 2. 对 mqtt:subscribe / mqtt:tag:subscribe 做引用计数，避免多个组件互相提前 unsubscribe
 * 3. 在连接重建或重连后恢复共享订阅
 *
 * @param {{
 *   ioFactory: Function,
 *   getApiUrl?: (() => string) | string,
 *   getToken?: () => string,
 *   notifier?: (payload: { type: string, message: string }) => void,
 *   logger?: Pick<Console, "log" | "warn" | "error">
 * }} options
 * @returns {object}
 */
export function createMqttSocketSharedRegistry(options = {}) {
  const {
    ioFactory,
    getApiUrl = () => DEFAULT_SOCKET_URL,
    getToken = () => "",
    notifier = null,
    logger = console,
  } = options;

  if (typeof ioFactory !== "function") {
    throw new Error("ioFactory is required");
  }

  const entries = new Map();

  const notify = (type, message) => {
    if (typeof notifier === "function") {
      notifier({ type, message });
    }
  };

  const resolveSocketUrl = () => {
    const raw =
      typeof getApiUrl === "function"
        ? getApiUrl()
        : getApiUrl || DEFAULT_SOCKET_URL;
    return String(raw || DEFAULT_SOCKET_URL).replace(/\/$/, "");
  };

  const emitSharedEvent = (entry, event, payload) => {
    if (!entry?.socket?.connected) {
      return;
    }
    entry.socket.emit(event, payload);
  };

  const dispatchMessage = (entry, data) => {
    entry.messageHandlers.forEach((handler) => {
      try {
        handler(data);
      } catch (error) {
        logger.error?.("[MqttSocket] Message handler error:", error);
      }
    });
  };

  const notifyStateChange = (entry) => {
    const snapshot = {
      socket: entry.socket,
      connected: Boolean(entry.connected),
    };
    entry.stateListeners.forEach((listener) => {
      try {
        listener(snapshot);
      } catch (error) {
        logger.error?.("[MqttSocket] State listener error:", error);
      }
    });
  };

  const replaySharedSubscriptions = (entry) => {
    entry.subscriptionRefs.forEach((count, subscriptionId) => {
      if (count > 0) {
        emitSharedEvent(entry, "mqtt:subscribe", { subscriptionId });
      }
    });
    entry.tagRefs.forEach((count, tagId) => {
      if (count > 0) {
        emitSharedEvent(entry, "mqtt:tag:subscribe", { tagId });
      }
    });
  };

  const destroyEntry = (key) => {
    const entry = entries.get(key);
    if (!entry) {
      return;
    }

    entries.delete(key);
    entry.stateListeners.clear();
    entry.messageHandlers.clear();
    entry.subscriptionRefs.clear();
    entry.tagRefs.clear();
    entry.consumerCount = 0;
    entry.connected = false;

    if (entry.socket) {
      entry.socket.disconnect();
    }
  };

  const createEntry = ({ key, token, projectId, previewSessionId }) => {
    const socketUrl = resolveSocketUrl();
    const socket = ioFactory(socketUrl, {
      path: DEFAULT_SOCKET_PATH,
      transports: ["websocket"],
      reconnection: true,
      reconnectionDelay: 1000,
      reconnectionDelayMax: 5000,
      reconnectionAttempts: Infinity,
      auth: {
        token,
        projectId,
        previewSessionId,
      },
    });

    const entry = {
      key,
      token,
      projectId,
      previewSessionId,
      socket,
      connected: Boolean(socket.connected),
      consumerCount: 0,
      stateListeners: new Set(),
      messageHandlers: new Map(),
      subscriptionRefs: new Map(),
      tagRefs: new Map(),
    };

    socket.on("connect", () => {
      entry.connected = true;
      logger.log?.("[MqttSocket] Connected:", socket.id);
      notify("success", "WebSocket已连接");
      replaySharedSubscriptions(entry);
      notifyStateChange(entry);
    });

    socket.on("disconnect", (reason) => {
      entry.connected = false;
      logger.log?.("[MqttSocket] Disconnected:", reason);
      if (reason !== "io client disconnect") {
        notify("warning", "WebSocket连接断开");
      }
      notifyStateChange(entry);
    });

    socket.on("connect_error", (error) => {
      logger.error?.("[MqttSocket] Connection error:", error);
    });

    socket.on("reconnect", (attemptNumber) => {
      logger.log?.("[MqttSocket] Reconnected after", attemptNumber, "attempts");
      notify("success", "WebSocket已重连");
    });

    socket.on("mqtt:message", (data) => {
      dispatchMessage(entry, data);
    });

    socket.on("mqtt:tag:value", (data) => {
      dispatchMessage(entry, data);
    });

    socket.on("mqtt:subscription:status", (data) => {
      logger.log?.("[MqttSocket] Subscription status:", data);
    });

    socket.on("mqtt:connection:status", (data) => {
      logger.log?.("[MqttSocket] Connection status:", data);
    });

    socket.onAny?.((event, ...args) => {
      logger.log?.("[MqttSocket] Event received:", event, args);
    });

    entries.set(key, entry);
    return entry;
  };

  const ensureEntry = ({ projectId, previewSessionId }) => {
    const token = String(getToken?.() || "").trim();
    const normalizedProjectId = String(projectId || "").trim();
    const normalizedPreviewSessionId = String(previewSessionId || "").trim();

    if (!token) {
      logger.warn?.(
        "[MqttSocket] 缺少 token，跳过连接。请确认 Storage.getToken() 已就绪。",
      );
      return null;
    }

    if (!normalizedProjectId) {
      logger.warn?.(
        "[MqttSocket] 缺少 projectId，跳过连接。当前不会发起 socket 握手。",
      );
      return null;
    }

    if (!normalizedPreviewSessionId) {
      logger.warn?.(
        "[MqttSocket] 缺少 previewSessionId，跳过连接。等待 preview session 创建完成后再重试。",
      );
      return null;
    }

    const key = buildMqttSocketSharedKey({
      token,
      projectId: normalizedProjectId,
      previewSessionId: normalizedPreviewSessionId,
    });
    const entry =
      entries.get(key) ||
      createEntry({
        key,
        token,
        projectId: normalizedProjectId,
        previewSessionId: normalizedPreviewSessionId,
      });

    return { key, entry };
  };

  return {
    acquire(params) {
      const result = ensureEntry(params);
      if (!result) {
        return null;
      }

      result.entry.consumerCount += 1;
      return result;
    },

    release(key) {
      if (!key) {
        return;
      }

      const entry = entries.get(key);
      if (!entry) {
        return;
      }

      entry.consumerCount = Math.max(0, entry.consumerCount - 1);
      if (entry.consumerCount === 0) {
        destroyEntry(key);
      }
    },

    watchState(key, listener) {
      const entry = entries.get(key);
      if (!entry || typeof listener !== "function") {
        return () => {};
      }

      entry.stateListeners.add(listener);
      listener({
        socket: entry.socket,
        connected: Boolean(entry.connected),
      });

      return () => {
        entry.stateListeners.delete(listener);
      };
    },

    addMessageHandler(key, handlerId, handler) {
      const entry = entries.get(key);
      if (!entry || !handlerId || typeof handler !== "function") {
        return;
      }
      entry.messageHandlers.set(handlerId, handler);
    },

    removeMessageHandler(key, handlerId) {
      const entry = entries.get(key);
      if (!entry || !handlerId) {
        return;
      }
      entry.messageHandlers.delete(handlerId);
    },

    subscribeSubscription(key, subscriptionId) {
      const entry = entries.get(key);
      const normalizedSubscriptionId = String(subscriptionId || "").trim();
      if (!entry || !normalizedSubscriptionId) {
        return;
      }

      const nextCount =
        (entry.subscriptionRefs.get(normalizedSubscriptionId) || 0) + 1;
      entry.subscriptionRefs.set(normalizedSubscriptionId, nextCount);

      if (nextCount === 1) {
        emitSharedEvent(entry, "mqtt:subscribe", {
          subscriptionId: normalizedSubscriptionId,
        });
      }
    },

    unsubscribeSubscription(key, subscriptionId) {
      const entry = entries.get(key);
      const normalizedSubscriptionId = String(subscriptionId || "").trim();
      if (!entry || !normalizedSubscriptionId) {
        return;
      }

      const currentCount =
        entry.subscriptionRefs.get(normalizedSubscriptionId) || 0;
      if (currentCount <= 1) {
        entry.subscriptionRefs.delete(normalizedSubscriptionId);
        emitSharedEvent(entry, "mqtt:unsubscribe", {
          subscriptionId: normalizedSubscriptionId,
        });
        return;
      }

      entry.subscriptionRefs.set(normalizedSubscriptionId, currentCount - 1);
    },

    subscribeTag(key, tagId) {
      const entry = entries.get(key);
      const normalizedTagId = String(tagId || "").trim();
      if (!entry || !normalizedTagId) {
        return;
      }

      const nextCount = (entry.tagRefs.get(normalizedTagId) || 0) + 1;
      entry.tagRefs.set(normalizedTagId, nextCount);

      if (nextCount === 1) {
        emitSharedEvent(entry, "mqtt:tag:subscribe", {
          tagId: normalizedTagId,
        });
      }
    },

    unsubscribeTag(key, tagId) {
      const entry = entries.get(key);
      const normalizedTagId = String(tagId || "").trim();
      if (!entry || !normalizedTagId) {
        return;
      }

      const currentCount = entry.tagRefs.get(normalizedTagId) || 0;
      if (currentCount <= 1) {
        entry.tagRefs.delete(normalizedTagId);
        emitSharedEvent(entry, "mqtt:tag:unsubscribe", {
          tagId: normalizedTagId,
        });
        return;
      }

      entry.tagRefs.set(normalizedTagId, currentCount - 1);
    },

    emit(key, event, payload) {
      const entry = entries.get(key);
      if (!entry?.socket?.connected) {
        logger.warn?.("[MqttSocket] Not connected, cannot emit:", event);
        return false;
      }

      entry.socket.emit(event, payload);
      return true;
    },

    getSnapshot(key) {
      const entry = entries.get(key);
      if (!entry) {
        return {
          socket: null,
          connected: false,
        };
      }

      return {
        socket: entry.socket,
        connected: Boolean(entry.connected),
      };
    },

    resetForTests() {
      Array.from(entries.keys()).forEach((key) => {
        destroyEntry(key);
      });
    },
  };
}
