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

import type { LockResult, PageLockState } from "../document/types.ts";
import { EventEmitter } from "../utils/EventEmitter";
import {
  getApiErrorCode,
  getApiErrorData,
  isApiBusinessError,
} from "@/utils/request";

const PAGE_LOCK_URL_RE = /\/pages\/(.+)\/lock/;
const PAGE_LOCKED_BUSINESS_CODE = 25003;
const DIGITS_ONLY_RE = /^\d+$/;

export interface PageLockApi {
  get: (path: string) => Promise<unknown>;
  post: (path: string, body?: unknown) => Promise<unknown>;
  delete: (path: string) => Promise<unknown>;
}

interface LockPayload {
  locked?: boolean | undefined;
  lockedBy?: string | undefined;
  lockedByName?: string | undefined;
  lockedAt?: number | undefined;
}

export interface PageLockSocket {
  on: (event: string, handler: (data: unknown) => void) => void;
  off: (event: string, handler: (data: unknown) => void) => void;
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

interface LockBusinessError extends Error {
  code: number | undefined;
  reqId: string | undefined;
  data: unknown;
  isBusinessError?: boolean;
}

interface LegacyEnvelope {
  success: boolean;
  msg: string;
  data: unknown;
  reqId: string | undefined;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object";
}

function toNumericCode(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && DIGITS_ONLY_RE.test(value.trim())) {
    return Number.parseInt(value.trim(), 10);
  }
  return undefined;
}

/**
 * PageLock 接口在迁移期可能出现以下形态：
 * - 统一包络：{ code, msg, data }
 * - 旧包络：{ success, data }
 * - 直接业务体：{ locked, lockedBy, ... }
 * 这里做单点收敛，避免各调用处反复兼容判断。
 */
function extractLockPayload(response: unknown): LockPayload {
  if (!response || typeof response !== "object") {
    return {};
  }

  const record = response as Record<string, unknown>;
  const nested = record.data;
  const raw = nested && typeof nested === "object" ? (nested as Record<string, unknown>) : record;

  return {
    locked: typeof raw.locked === "boolean" ? raw.locked : undefined,
    lockedBy: typeof raw.lockedBy === "string" ? raw.lockedBy : undefined,
    lockedByName: typeof raw.lockedByName === "string" ? raw.lockedByName : undefined,
    lockedAt: typeof raw.lockedAt === "number" ? raw.lockedAt : undefined,
  };
}

function extractApiCode(response: unknown): number | undefined {
  if (!isRecord(response)) {
    return undefined;
  }
  return toNumericCode(response.code);
}

function extractApiMessage(response: unknown): string {
  if (!isRecord(response)) {
    return "";
  }
  if (typeof response.msg === "string") {
    return response.msg;
  }
  if (typeof response.message === "string") {
    return response.message;
  }
  return "";
}

function extractApiReqId(response: unknown): string | undefined {
  if (!isRecord(response)) {
    return undefined;
  }
  if (typeof response.reqId === "string") {
    return response.reqId;
  }
  return undefined;
}

function extractLegacyEnvelope(response: unknown): LegacyEnvelope | null {
  if (!isRecord(response) || typeof response.success !== "boolean") {
    return null;
  }

  const msg =
    typeof response.message === "string"
      ? response.message
      : typeof response.msg === "string"
        ? response.msg
        : "";
  const reqId = extractApiReqId(response);

  return {
    success: response.success,
    msg,
    data: response.data,
    reqId,
  };
}

function resolveLockFailureByPayload(
  pageId: string,
  payload: LockPayload,
  currentUserId: string,
): LockResult | null {
  if (!isLockedByOtherUser(payload, currentUserId)) {
    return null;
  }

  return {
    success: false,
    reason: "locked",
    lockedByName: payload.lockedByName,
  };
}

function applyLockedState(
  pageId: string,
  payload: LockPayload,
): PageLockState {
  return {
    pageId,
    locked: true,
    lockedBy: payload.lockedBy,
    lockedByName: payload.lockedByName,
    lockedAt: payload.lockedAt,
    isOwner: false,
  };
}

function resolveLockFailureResult(
  pageId: string,
  code: number | undefined,
  msg: string,
  data: unknown,
  reqId: string | undefined,
  currentUserId: string,
): {
  lockState: PageLockState | null;
  result: LockResult;
} {
  const payload = extractLockPayload(data);
  const lockedResult = resolveLockFailureByPayload(pageId, payload, currentUserId);
  if (lockedResult) {
    return {
      lockState: applyLockedState(pageId, payload),
      result: lockedResult,
    };
  }

  return {
    lockState: null,
    result: {
      success: false,
      reason: "error",
      error: buildBusinessError(code, msg, data, reqId),
    },
  };
}

function ensureNoEnvelopeFailure(
  pageId: string,
  response: unknown,
  currentUserId: string,
): {
  lockState: PageLockState | null;
  result: LockResult;
} | null {
  const code = extractApiCode(response);
  if (code !== undefined) {
    if (code === 0) {
      return null;
    }
    return resolveLockFailureResult(
      pageId,
      code,
      extractApiMessage(response),
      response,
      extractApiReqId(response),
      currentUserId,
    );
  }

  const legacyEnvelope = extractLegacyEnvelope(response);
  if (!legacyEnvelope || legacyEnvelope.success) {
    return null;
  }
  return resolveLockFailureResult(
    pageId,
    undefined,
    legacyEnvelope.msg || "获取页面锁失败",
    legacyEnvelope.data,
    legacyEnvelope.reqId,
    currentUserId,
  );
}

function ensureNoQueryFailure(response: unknown): void {
  const code = extractApiCode(response);
  if (code !== undefined) {
    if (code !== 0) {
      throw buildBusinessError(
        code,
        extractApiMessage(response),
        response,
        extractApiReqId(response),
      );
    }
    return;
  }

  const legacyEnvelope = extractLegacyEnvelope(response);
  if (!legacyEnvelope || legacyEnvelope.success) {
    return;
  }

  throw buildBusinessError(
    undefined,
    legacyEnvelope.msg || "查询页面锁状态失败",
    legacyEnvelope.data,
    legacyEnvelope.reqId,
  );
}

function buildBusinessError(
  code: number | undefined,
  msg: string,
  data: unknown,
  reqId: string | undefined,
): LockBusinessError {
  const error = new Error(msg || "获取页面锁失败") as LockBusinessError;
  error.name = "PageLockBusinessError";
  error.code = code;
  error.reqId = reqId;
  error.data = data;
  error.isBusinessError = true;
  return error;
}

function toLockPayloadSource(response: unknown): unknown {
  const legacyEnvelope = extractLegacyEnvelope(response);
  if (legacyEnvelope?.success) {
    return legacyEnvelope.data;
  }
  return response;
}

function isLockedByOtherUser(payload: LockPayload, currentUserId: string): boolean {
  if (payload.locked !== true) {
    return false;
  }
  if (typeof payload.lockedBy !== "string" || payload.lockedBy.length === 0) {
    return true;
  }
  return payload.lockedBy !== currentUserId;
}

/**
 * 页面锁管理器
 */
export class PageLockManager extends EventEmitter {
  _api: PageLockApi | undefined;
  _socket: PageLockSocket | null;
  _currentUserId: string;
  _heartbeatInterval: number;
  _lockVersion: number;
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

    this._lockVersion = 0;

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
      window.addEventListener("beforeunload", this._handleBeforeUnload.bind(this));
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
      window.removeEventListener("beforeunload", this._handleBeforeUnload.bind(this));
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
   * @param {object} socket - Socket.IO 实例
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
      const response = await this._api.post(`/pages/${pageId}/lock`);
      const envelopeFailure = ensureNoEnvelopeFailure(pageId, response, this._currentUserId);
      if (envelopeFailure) {
        this._lockState = envelopeFailure.lockState;
        return envelopeFailure.result;
      }

      const lockPayload = extractLockPayload(toLockPayloadSource(response));
      if (!isLockedByOtherUser(lockPayload, this._currentUserId)) {
        this._lockVersion += 1;
        this._currentPageId = pageId;
        this._lockState = {
          pageId,
          locked: true,
          lockedBy: lockPayload.lockedBy,
          lockedByName: lockPayload.lockedByName,
          lockedAt: lockPayload.lockedAt,
          isOwner: true,
        };
        this.startHeartbeat();

        this.emit("lockAcquired", { pageId, lockState: this._lockState });

        return { success: true };
      } else {
        this._lockState = applyLockedState(pageId, lockPayload);

        return {
          success: false,
          reason: "locked",
          lockedByName: lockPayload.lockedByName,
        };
      }
    } catch (error) {
      if (isApiBusinessError(error)) {
        const businessCode = getApiErrorCode(error);
        const businessData = getApiErrorData(error);
        const businessPayload = extractLockPayload(businessData);
        if (
          businessCode === PAGE_LOCKED_BUSINESS_CODE ||
          isLockedByOtherUser(businessPayload, this._currentUserId)
        ) {
          this._lockState = {
            pageId,
            locked: true,
            lockedBy: businessPayload.lockedBy,
            lockedByName: businessPayload.lockedByName,
            lockedAt: businessPayload.lockedAt,
            isOwner: false,
          };
          return {
            success: false,
            reason: "locked",
            lockedByName: businessPayload.lockedByName,
          };
        }

        return {
          success: false,
          reason: "error",
          error: buildBusinessError(
            businessCode,
            error instanceof Error ? error.message : "获取页面锁失败",
            businessData,
            undefined,
          ),
        };
      }

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
    const lockVersion = this._lockVersion;

    try {
      if (this._api) {
        await this._api.delete(`/pages/${pageId}/lock`);
      }
    } catch (error) {
      console.error("释放页面锁失败:", error);
    } finally {
      /**
       * 释放请求可能在网络层悬挂较久；期间既可能是同一把锁被改写（如锁丢失），
       * 也可能是用户已经重新拿到了另一把锁。这里用锁版本区分“同一会话”与“新会话”：
       * 只有版本未变化时才清理，避免旧请求误删新锁状态，同时保证同一把锁的残留能被收口。
       */
      if (this._currentPageId === pageId && this._lockVersion === lockVersion) {
        this.stopHeartbeat();
        this._currentPageId = null;
        this._lockState = null;
      }

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
      ensureNoQueryFailure(response);
      const data = extractLockPayload(toLockPayloadSource(response));

      return {
        pageId,
        locked: Boolean(data.locked),
        lockedBy: data.lockedBy,
        lockedByName: data.lockedByName,
        lockedAt: data.lockedAt,
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
    this._socket.on("page:lock:force_release", this._handleForceRelease.bind(this));
  }

  /**
   * 移除 Socket 监听器
   * @private
   */
  _removeSocketListeners() {
    if (!this._socket) return;

    this._socket.off("page:lock:changed", this._handleLockChanged.bind(this));
    this._socket.off("page:lock:force_release", this._handleForceRelease.bind(this));
  }

  /**
   * 处理锁状态变化
   * @param {object} data - 事件数据
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
   * @param {object} data - 事件数据
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
 * @returns {object}
 */
export function createMockApiClient(): PageLockApi {
  const locks = new Map<string, LockPayload>();

  return {
    async get(url: string) {
      const match = url.match(PAGE_LOCK_URL_RE);
      const pageId = match?.[1];
      if (pageId) {
        const lock = locks.get(pageId);
        return {
          code: 0,
          msg: "ok",
          data: lock || { locked: false },
        };
      }
      throw new Error("Unknown endpoint");
    },

    async post(url: string): Promise<unknown> {
      const match = url.match(PAGE_LOCK_URL_RE);
      const pageId = match?.[1];
      if (pageId) {
        const existingLock = locks.get(pageId);

        if (existingLock && existingLock.locked) {
          return {
            code: PAGE_LOCKED_BUSINESS_CODE,
            msg: "页面已被其他用户锁定",
            data: existingLock as LockPayload,
          };
        }

        const lock: LockPayload = {
          locked: true,
          lockedBy: "test-user",
          lockedByName: "测试用户",
          lockedAt: Date.now(),
        };
        locks.set(pageId, lock);

        return {
          code: 0,
          msg: "ok",
          data: lock,
        };
      }
      throw new Error("Unknown endpoint");
    },

    async delete(url: string) {
      const match = url.match(PAGE_LOCK_URL_RE);
      const pageId = match?.[1];
      if (pageId) {
        locks.delete(pageId);
        return { code: 0, msg: "ok", data: null };
      }
      throw new Error("Unknown endpoint");
    },
  };
}

export default PageLockManager;
