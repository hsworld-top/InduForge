import { ref, onMounted, onBeforeUnmount } from "vue";
import { io } from "socket.io-client";
import { ElMessage } from "element-plus";

/**
 * MQTT Socket.IO Composable
 * 管理WebSocket连接，接收实时MQTT消息
 */
export function useMqttSocket(projectId) {
  const socket = ref(null);
  const connected = ref(false);
  const messageHandlers = ref(new Map());

  /**
   * 连接到Socket.IO服务器
   */
  const connect = () => {
    if (socket.value?.connected) {
      console.log("[MqttSocket] Already connected");
      return;
    }

    // 获取API基础URL，通过vite define注入（因为Socket.IO不走代理）
    const apiUrl = __VITE_API_URL__ || "http://localhost:9099";
    const socketUrl = apiUrl.replace(/\/$/, ""); // 去掉末尾斜杠

    console.log("[MqttSocket] VITE_API_URL:", __VITE_API_URL__);
    console.log(
      "[MqttSocket] Connecting to",
      socketUrl,
      "with projectId:",
      projectId,
    );

    // 测试基础连接
    setTimeout(() => {
      if (socket.value && socket.value.connected) {
        console.log("[MqttSocket] Socket.IO connected successfully");
      } else {
        console.error("[MqttSocket] Socket.IO connection failed");
        console.log("[MqttSocket] Current socket state:", {
          exists: !!socket.value,
          connected: socket.value?.connected,
          connecting: socket.value?.connecting,
          disconnected: socket.value?.disconnected,
          id: socket.value?.id,
        });
      }
    }, 2000);

    socket.value = io(socketUrl, {
      path: "/socket.io",
      transports: ["websocket", "polling"],
      reconnection: true,
      reconnectionDelay: 1000,
      reconnectionDelayMax: 5000,
      reconnectionAttempts: Infinity,
      query: {
        projectId: projectId,
      },
    });

    // 连接成功
    socket.value.on("connect", () => {
      connected.value = true;
      console.log("[MqttSocket] Connected:", socket.value.id);
      ElMessage({
        type: "success",
        message: "WebSocket已连接",
        duration: 2000,
        offset: 60,
      });
    });

    // 连接断开
    socket.value.on("disconnect", (reason) => {
      connected.value = false;
      console.log("[MqttSocket] Disconnected:", reason);

      if (reason !== "io client disconnect") {
        ElMessage({
          type: "warning",
          message: "WebSocket连接断开",
          duration: 2000,
          offset: 60,
        });
      }
    });

    // 连接错误
    socket.value.on("connect_error", (error) => {
      console.error("[MqttSocket] Connection error:", error);
    });

    // 重连
    socket.value.on("reconnect", (attemptNumber) => {
      console.log("[MqttSocket] Reconnected after", attemptNumber, "attempts");
      ElMessage({
        type: "success",
        message: "WebSocket已重连",
        duration: 2000,
        offset: 60,
      });
    });

    // 接收MQTT消息
    socket.value.on("mqtt:message", (data) => {
      // 调用所有注册的消息处理器
      messageHandlers.value.forEach((handler) => {
        try {
          handler(data);
        } catch (error) {
          console.error("[MqttSocket] Message handler error:", error);
        }
      });
    });

    // 订阅状态变化
    socket.value.on("mqtt:subscription:status", (data) => {
      console.log("[MqttSocket] Subscription status:", data);
    });

    // 连接状态变化
    socket.value.on("mqtt:connection:status", (data) => {
      console.log("[MqttSocket] Connection status:", data);
    });

    // Tag值更新
    socket.value.on("mqtt:tag:value", (data) => {
      console.log("[MqttSocket] Received tag value update:", data);
      // 广播给所有注册的消息处理器
      messageHandlers.value.forEach((handler) => {
        try {
          handler(data);
        } catch (error) {
          console.error("[MqttSocket] Tag value handler error:", error);
        }
      });
    });

    // 监听所有事件用于调试
    socket.value.onAny((event, ...args) => {
      console.log("[MqttSocket] Event received:", event, args);
    });
  };

  /**
   * 断开连接
   */
  const disconnect = () => {
    if (socket.value) {
      socket.value.disconnect();
      socket.value = null;
      connected.value = false;
      console.log("[MqttSocket] Disconnected manually");
    }
  };

  /**
   * 订阅MQTT主题消息
   * @param {string} subscriptionId - 订阅ID
   * @param {Function} handler - 消息处理函数
   * @returns {Function} 取消订阅的函数
   */
  const subscribeMessages = (subscriptionId, handler) => {
    const handlerId = `${subscriptionId}-${Date.now()}`;

    // 创建包装处理器，只处理特定订阅的消息
    const wrappedHandler = (data) => {
      if (data.subscriptionId === subscriptionId) {
        handler(data);
      }
    };

    messageHandlers.value.set(handlerId, wrappedHandler);

    // 通知服务器订阅
    if (socket.value?.connected) {
      console.log(
        `[MqttSocket] Emitting mqtt:subscribe for subscription: ${subscriptionId}`,
      );
      socket.value.emit("mqtt:subscribe", { subscriptionId });
    } else {
      console.log(
        `[MqttSocket] Socket not connected, cannot subscribe to: ${subscriptionId}`,
      );
    }

    // 返回取消订阅函数
    return () => {
      messageHandlers.value.delete(handlerId);
      if (socket.value?.connected) {
        socket.value.emit("mqtt:unsubscribe", { subscriptionId });
      }
    };
  };

  /**
   * 注册全局消息处理器
   * @param {Function} handler - 消息处理函数
   * @returns {Function} 取消注册的函数
   */
  const onMessage = (handler) => {
    const handlerId = `global-${Date.now()}`;
    messageHandlers.value.set(handlerId, handler);

    // 返回取消注册函数
    return () => {
      messageHandlers.value.delete(handlerId);
    };
  };

  /**
   * 发送消息到服务器
   */
  const emit = (event, data) => {
    if (socket.value?.connected) {
      socket.value.emit(event, data);
    } else {
      console.warn("[MqttSocket] Not connected, cannot emit:", event);
    }
  };

  // 组件挂载时自动连接
  onMounted(() => {
    connect();
  });

  // 组件卸载时断开连接
  onBeforeUnmount(() => {
    disconnect();
  });

  return {
    socket,
    connected,
    connect,
    disconnect,
    subscribeMessages,
    onMessage,
    emit,
  };
}
