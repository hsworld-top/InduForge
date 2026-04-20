/**
 * Socket.IO连接测试工具
 */
import { io } from "socket.io-client";

export function testSocketConnection() {
  console.log("🔍 开始Socket.IO连接测试...");

  const apiUrl = __VITE_API_URL__ || "http://localhost:19601";
  const socketUrl = apiUrl.replace(/\/$/, "");

  console.log("📡 目标URL:", socketUrl);

  const testSocket = io(socketUrl, {
    path: "/socket.io",
    transports: ["websocket", "polling"],
    query: {
      projectId: "test-project",
    },
    timeout: 5000,
  });

  testSocket.on("connect", () => {
    console.log("✅ Socket.IO连接成功! Socket ID:", testSocket.id);
    testSocket.disconnect();
  });

  testSocket.on("connect_error", (error) => {
    console.error("❌ Socket.IO连接失败:", error);
  });

  testSocket.on("disconnect", (reason) => {
    console.log("🔌 Socket.IO断开连接:", reason);
  });

  // 5秒后强制断开
  setTimeout(() => {
    if (testSocket.connected) {
      testSocket.disconnect();
    }
  }, 5000);
}

// 自动运行测试（仅在开发环境）
if (import.meta.env.DEV) {
  setTimeout(() => {
    testSocketConnection();
  }, 3000);
}
