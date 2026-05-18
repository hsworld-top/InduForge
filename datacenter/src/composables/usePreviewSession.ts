// @ts-nocheck
import { ref, watch, onBeforeUnmount, unref } from "vue";
import dayjs from "dayjs";
import { TIME_FORMAT } from "@/constants";
import {
  createPreviewSession,
  heartbeatPreviewSession,
  deletePreviewSession,
} from "@/api/data.api";
import { getApiErrorMessage } from "@/utils/request";
import { usePreviewSessionStore } from "@/stores/preview-session.store";

const HEARTBEAT_INTERVAL_MS = 30_000;

const resolveValue = (value) => {
  if (typeof value === "function") {
    return value();
  }
  return unref(value);
};

/**
 * 管理 datacenter 的 preview session 生命周期。
 * 这个 composable 负责把当前 project 与后端 preview session 对齐，
 * 并在组件存活期间持续续期，避免多个组件各自重复创建会话。
 *
 * 边界说明：
 * 1. 心跳短暂失败时不立即熔断，避免网络抖动导致整页必须重开。
 * 2. 如果 create 请求在组件卸载或 project 切换后才返回，会立即反向删除迟到的 session，避免泄漏。
 *
 * @param {import("vue").MaybeRefOrGetter<string | null | undefined>} projectIdSource
 * @param {{ autoStart?: boolean }} [options]
 */
export function usePreviewSession(projectIdSource, options = {}) {
  const autoStart = options.autoStart !== false;
  const sessionId = ref("");
  const loading = ref(false);
  const error = ref(null);
  // NavRail 徽标依赖 store，session 生命周期内同步状态
  const previewStore = usePreviewSessionStore();

  let heartbeatTimer = null;
  let createPromise = null;
  let destroyPromise = null;
  let sessionGeneration = 0;
  let unmounted = false;

  const clearHeartbeatTimer = () => {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer);
      heartbeatTimer = null;
    }
  };

  const stopSession = () => {
    clearHeartbeatTimer();
    sessionId.value = "";
    previewStore.close();
  };

  const readProjectId = () => resolveValue(projectIdSource) || "";

  const sendHeartbeat = async () => {
    const currentSessionId = sessionId.value;
    if (!currentSessionId) {
      return;
    }

    try {
      await heartbeatPreviewSession(currentSessionId);
      error.value = null;
    } catch (heartbeatError) {
      console.warn(
        "[PreviewSession] 心跳失败:",
        getApiErrorMessage(heartbeatError, "预览会话心跳失败"),
      );
      error.value = heartbeatError;
    }
  };

  const startHeartbeat = () => {
    clearHeartbeatTimer();

    if (!sessionId.value) {
      return;
    }

    heartbeatTimer = setInterval(() => {
      void sendHeartbeat();
    }, HEARTBEAT_INTERVAL_MS);
  };

  const destroySession = async () => {
    const currentSessionId = sessionId.value;
    sessionGeneration += 1;

    clearHeartbeatTimer();
    sessionId.value = "";
    previewStore.close();

    if (!currentSessionId) {
      return;
    }

    if (destroyPromise) {
      return destroyPromise;
    }

    destroyPromise = deletePreviewSession(currentSessionId)
      .catch((destroyError) => {
        console.warn("[PreviewSession] 销毁失败:", destroyError);
      })
      .finally(() => {
        destroyPromise = null;
      });

    return destroyPromise;
  };

  const ensureSession = async () => {
    const currentProjectId = readProjectId();
    if (!currentProjectId) {
      console.warn("[PreviewSession] 缺少 projectId，跳过创建");
      stopSession();
      return "";
    }

    if (sessionId.value) {
      return sessionId.value;
    }

    if (createPromise) {
      return createPromise;
    }

    loading.value = true;
    error.value = null;
    const requestGeneration = ++sessionGeneration;

    createPromise = createPreviewSession(currentProjectId)
      .then((response) => {
        const payload = response?.data || {};
        const nextSessionId =
          payload.previewSessionId || payload.sessionId || payload.id || "";

        if (!nextSessionId) {
          throw new Error("preview session 响应缺少 sessionId");
        }

        if (
          unmounted ||
          sessionGeneration !== requestGeneration ||
          readProjectId() !== currentProjectId
        ) {
          void deletePreviewSession(nextSessionId).catch((destroyError) => {
            console.warn(
              "[PreviewSession] 清理迟到 session 失败:",
              destroyError,
            );
          });
          return "";
        }

        sessionId.value = nextSessionId;
        previewStore.open({
          sessionId: nextSessionId,
          projectId: currentProjectId,
          createdAt: dayjs().format(TIME_FORMAT),
        });
        startHeartbeat();
        return nextSessionId;
      })
      .catch((createError) => {
        console.error(
          "[PreviewSession] 创建失败:",
          getApiErrorMessage(createError, "预览会话创建失败"),
        );
        error.value = createError;
        stopSession();
        return "";
      })
      .finally(() => {
        loading.value = false;
        createPromise = null;
      });

    return createPromise;
  };

  watch(
    () => readProjectId(),
    async (nextProjectId, prevProjectId) => {
      if (unmounted) {
        return;
      }

      if (!nextProjectId) {
        await destroySession();
        return;
      }

      if (prevProjectId && prevProjectId !== nextProjectId) {
        await destroySession();
      }

      if (!autoStart && !sessionId.value) {
        return;
      }

      await ensureSession();
    },
    { immediate: autoStart },
  );

  onBeforeUnmount(async () => {
    unmounted = true;
    await destroySession();
  });

  return {
    sessionId,
    loading,
    error,
    ensureSession,
    destroySession,
    refreshSession: sendHeartbeat,
  };
}
