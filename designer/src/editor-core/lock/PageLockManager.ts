/**
 * PageLockManager - 页面编辑锁管理
 * 实现简化版多人开发：同一页面同时只能有一人编辑
 *
 * 功能：
 * - 页面级编辑锁获取/释放
 * - 心跳续锁
 * - WebSocket 监听锁状态变化
 * - 只读模式
 */

import { EventEmitter } from "../utils/EventEmitter";
import type {
  PageLockState,
  LockResult,
  EditorReadonlyState,
} from "../document/types.js";

export interface PageLockApi {
  get(path: string): Promise<{ data?: unknown; success?: boolean }>;
  post(path: string, body?: unknown): Promise<LockAcquireResponse>;
  delete(path: string): Promise<unknown>;
}

interface LockAcquireResponse {
  success: boolean;
  data?: {
    locked?: boolean;
    lockedBy?: string;
    lockedByName?: string;
    lockedAt?: number;
  };
}

export interface PageLockSocket {
  on(event: string, handler: (data: unknown) => void): void;
  off(event: string, handler: (data: unknown) => void): void;
}

export interface PageLockManagerOptions {
  api: PageLockApi;
  socket?: PageLockSocket | null;
  currentUserId?: string;
  heartbeatInterval?: number;
}

interface LockEventPayload {
  pageId?: string;
  locked?: boolean;
  lockedBy?: string;
  pageName?: string;
  reason?: string;
}

/**
 * 页面锁管理器
 */
export class PageLockManager extends EventEmitter {
  _api: PageLockApi | undefined;
  _socket: PageLockSocket | null;
  _currentUserId: string;
  _heartbeatInterval: number;
  _currentPageId: string | null;
  _lockState: PageLockState | null;
  _heartbeatTimer: ReturnType<typeof setInterval> | null;
  _isInitialized: boolean;

  /**
   * 创建页面锁管理器
   */
  constructor(options: PageLockManagerOptions) {
    super();

    this._api = options.api;

    this._socket = options.socket ?? null;

    this._currentUserId = options.currentUserId ?? "";

    this._heartbeatInterval = options.heartbeatInterval ?? 5 * 60 * 1000;

    this._currentPageId = null;

    this._lockState = null;

    this._heartbeatTimer = null;

    this._isInitialized = false;

    if (this._socket) {
      this._setupSocketListeners();
    }
  }

  // ==================== 初始化 ====================

  /**
   * 初始化（设置 beforeunload 监听）
   */
  init() {
    if (this._isInitialized) return;

    // 页面卸载时释放锁
    if (typeof window !== "undefined") {
      window.addEventListener(
        "beforeunload",
        this._handleBeforeUnload.bind(this),
      );
      window.addEventListener("unload", this._handleUnload.bind(this));
    }

    this._isInitialized = true;
  }

  /**
   * 销毁（清理资源）
   */
  destroy() {
    this.stopHeartbeat();

    if (typeof window !== "undefined") {
      window.removeEventListener(
        "beforeunload",
        this._handleBeforeUnload.bind(this),
      );
      window.removeEventListener("unload", this._handleUnload.bind(this));
    }

    this._isInitialized = false;
  }

  /**
   * 设置当前用户 ID
   * @param {string} userId - 用户 ID
   */
  setCurrentUserId(userId: string) {
    this._currentUserId = userId;
  }

  /**
   * 设置 Socket 实例
   * @param {Object} socket - Socket.IO 实例
   */
  setSocket(socket: PageLockSocket | null) {
    if (this._socket) {
      this._removeSocketListeners();
    }
    this._socket = socket;
    if (socket) {
      this._setupSocketListeners();
    }
  }

  // ==================== 锁操作 ====================

  /**
   * 尝试获取页面编辑锁
   * @param {string} pageId - 页面 ID
   * @returns {Promise<LockResult>}
   */
  async acquireLock(pageId: string): Promise<LockResult> {
    if (!this._api) {
      return {
        success: false,
        reason: "error",
        error: new Error("API 未初始化"),
      };
    }

    try {
      const response: LockAcquireResponse = await this._api.post(
        `/pages/${pageId}/lock`,
      );

      if (response.success) {
        const d = response.data ?? {};
        this._currentPageId = pageId;
        this._lockState = {
          pageId,
          locked: true,
          lockedBy: d.lockedBy,
          lockedByName: d.lockedByName,
          lockedAt: d.lockedAt,
          isOwner: true,
        };
        this.startHeartbeat();

        this.emit("lockAcquired", { pageId, lockState: this._lockState });

        return { success: true };
      } else {
        // 锁被其他人持有
        this._lockState = {
          pageId,
          locked: true,
          lockedBy: response.data?.lockedBy,
          lockedByName: response.data?.lockedByName,
          lockedAt: response.data?.lockedAt,
          isOwner: false,
        };

        return {
          success: false,
          reason: "locked",
          lockedByName: response.data?.lockedByName,
        };
      }
    } catch (error) {
      console.error("获取页面锁失败:", error);
      return { success: false, reason: "error", error };
    }
  }

  /**
   * 释放页面编辑锁
   * @returns {Promise<void>}
   */
  async releaseLock() {
    if (!this._currentPageId || !this._lockState?.isOwner) {
      return;
    }

    const pageId = this._currentPageId;

    try {
      if (this._api) {
        await this._api.delete(`/pages/${pageId}/lock`);
      }
    } catch (error) {
      console.error("释放页面锁失败:", error);
    } finally {
      this.stopHeartbeat();
      this._currentPageId = null;
      this._lockState = null;

      this.emit("lockReleased", { pageId });
    }
  }

  /**
   * 查询页面锁状态
   * @param {string} pageId - 页面 ID
   * @returns {Promise<PageLockState>}
   */
  async queryLockStatus(pageId: string): Promise<PageLockState> {
    if (!this._api) {
      throw new Error("API 未初始化");
    }

    try {
      const response = await this._api.get(`/pages/${pageId}/lock`);
      const data = (response.data ?? response) as Record<string, unknown>;

      return {
        pageId,
        locked: Boolean(data.locked),
        lockedBy: data.lockedBy as string | undefined,
        lockedByName: data.lockedByName as string | undefined,
        lockedAt: data.lockedAt as number | undefined,
        isOwner: data.lockedBy === this._currentUserId,
      };
    } catch (error) {
      console.error("查询页面锁状态失败:", error);
      throw error;
    }
  }

  /**
   * 强制释放锁（需要管理员权限）
   * @param {string} pageId - 页面 ID
   * @returns {Promise<void>}
   */
  async forceReleaseLock(pageId: string): Promise<void> {
    if (!this._api) {
      throw new Error("API 未初始化");
    }

    await this._api.delete(`/pages/${pageId}/lock/force`);

    this.emit("lockForceReleased", { pageId });
  }

  // ==================== 心跳续锁 ====================

  /**
   * 开始心跳
   */
  startHeartbeat() {
    this.stopHeartbeat();

    this._heartbeatTimer = setInterval(async () => {
      if (this._currentPageId && this._lockState?.isOwner && this._api) {
        try {
          await this._api.post(`/pages/${this._currentPageId}/lock/heartbeat`);
        } catch (error) {
          console.error("心跳续锁失败:", error);
          this._handleLockLost();
        }
      }
    }, this._heartbeatInterval);
  }

  /**
   * 停止心跳
   */
  stopHeartbeat() {
    if (this._heartbeatTimer) {
      clearInterval(this._heartbeatTimer);
      this._heartbeatTimer = null;
    }
  }

  // ==================== Socket 监听 ====================

  /**
   * 设置 Socket 监听器
   * @private
   */
  _setupSocketListeners() {
    if (!this._socket) return;

    // 监听页面锁状态变化
    this._socket.on("page:lock:changed", this._handleLockChanged.bind(this));

    // 监听强制释放通知
    this._socket.on(
      "page:lock:force_release",
      this._handleForceRelease.bind(this),
    );
  }

  /**
   * 移除 Socket 监听器
   * @private
   */
  _removeSocketListeners() {
    if (!this._socket) return;

    this._socket.off("page:lock:changed", this._handleLockChanged.bind(this));
    this._socket.off(
      "page:lock:force_release",
      this._handleForceRelease.bind(this),
    );
  }

  /**
   * 处理锁状态变化
   * @param {Object} data - 事件数据
   * @private
   */
  _handleLockChanged(data: unknown) {
    const payload = data as LockEventPayload;
    // 通知 UI 更新锁状态
    this.emit("lockStatusChanged", payload);

    // 检查是否影响当前页面
    if (payload.pageId === this._currentPageId) {
      if (payload.locked && payload.lockedBy !== this._currentUserId) {
        // 其他人获取了锁
        this._handleLockLost();
      }
    }
  }

  /**
   * 处理强制释放
   * @param {Object} data - 事件数据
   * @private
   */
  _handleForceRelease(data: unknown) {
    const payload = data as LockEventPayload;
    if (payload.pageId === this._currentPageId && this._lockState?.isOwner) {
      this.stopHeartbeat();
      this._lockState = null;
      this._currentPageId = null;

      this.emit("lockForceReleased", {
        pageId: payload.pageId,
        pageName: payload.pageName,
        reason: payload.reason,
      });
    }
  }

  /**
   * 处理锁丢失
   * @private
   */
  _handleLockLost() {
    if (!this._lockState) return;

    this._lockState = { ...this._lockState, isOwner: false };
    this.stopHeartbeat();

    this.emit("lockLost", { pageId: this._currentPageId });
  }

  // ==================== 页面事件处理 ====================

  /**
   * 处理 beforeunload 事件
   * @param {Event} event
   * @private
   */
  _handleBeforeUnload(event: BeforeUnloadEvent) {
    // 如果有未释放的锁，提示用户
    if (this._currentPageId && this._lockState?.isOwner) {
      event.preventDefault();
      event.returnValue = "您有未保存的更改，确定要离开吗？";
    }
  }

  /**
   * 处理 unload 事件
   * @private
   */
  _handleUnload() {
    // 使用 sendBeacon 确保请求发出
    if (this._currentPageId && this._lockState?.isOwner) {
      if (typeof navigator !== "undefined" && navigator.sendBeacon) {
        navigator.sendBeacon(
          `/api/v1/pages/${this._currentPageId}/lock/release`,
          JSON.stringify({ userId: this._currentUserId }),
        );
      }
    }
  }

  // ==================== 生命周期方法 ====================

  /**
   * 页面离开时调用
   * @returns {Promise<void>}
   */
  async onPageLeave() {
    await this.releaseLock();
  }

  /**
   * 应用关闭/刷新时调用
   * @returns {Promise<void>}
   */
  async onAppUnload() {
    await this.releaseLock();
  }

  // ==================== 状态查询 ====================

  /**
   * 获取当前锁状态
   * @returns {PageLockState | null}
   */
  getLockState() {
    return this._lockState;
  }

  /**
   * 获取当前锁定的页面 ID
   * @returns {string | null}
   */
  getCurrentPageId() {
    return this._currentPageId;
  }

  /**
   * 是否拥有当前页面的锁
   * @returns {boolean}
   */
  isLockOwner() {
    return this._lockState?.isOwner === true;
  }

  /**
   * 当前页面是否被锁定（不管是谁）
   * @returns {boolean}
   */
  isLocked() {
    return this._lockState?.locked === true;
  }

  /**
   * 获取编辑器只读状态
   * @returns {EditorReadonlyState}
   */
  getReadonlyState() {
    if (!this._lockState) {
      return { readonly: false };
    }

    if (this._lockState.isOwner) {
      return { readonly: false };
    }

    if (this._lockState.locked) {
      return {
        readonly: true,
        reason: "page_locked",
        lockedByName: this._lockState.lockedByName,
      };
    }

    return { readonly: false };
  }
}

/**
 * 创建 Mock API 客户端（用于测试）
 * @returns {Object}
 */
export function createMockApiClient(): PageLockApi {
  const locks = new Map<string, Record<string, unknown>>();

  return {
    async get(url: string) {
      const match = url.match(/\/pages\/(.+)\/lock/);
      const pageId = match?.[1];
      if (pageId) {
        const lock = locks.get(pageId);
        return {
          success: true,
          data: lock || { locked: false },
        };
      }
      throw new Error("Unknown endpoint");
    },

    async post(url: string): Promise<LockAcquireResponse> {
      const match = url.match(/\/pages\/(.+)\/lock/);
      const pageId = match?.[1];
      if (pageId) {
        const existingLock = locks.get(pageId);

        if (existingLock && existingLock.locked) {
          return {
            success: false,
            data: existingLock as NonNullable<LockAcquireResponse["data"]>,
          };
        }

        const lock: NonNullable<LockAcquireResponse["data"]> = {
          locked: true,
          lockedBy: "test-user",
          lockedByName: "测试用户",
          lockedAt: Date.now(),
        };
        locks.set(pageId, lock);

        return {
          success: true,
          data: lock,
        };
      }
      throw new Error("Unknown endpoint");
    },

    async delete(url: string) {
      const match = url.match(/\/pages\/(.+)\/lock/);
      const pageId = match?.[1];
      if (pageId) {
        locks.delete(pageId);
        return { success: true };
      }
      throw new Error("Unknown endpoint");
    },
  };
}

export default PageLockManager;
