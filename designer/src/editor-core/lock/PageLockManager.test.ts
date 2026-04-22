import { describe, expect, it, vi } from "vitest";
import { PageLockManager } from "./PageLockManager";

describe("PageLockManager", () => {
  it("旧释放请求完成后不会清掉新获取的页面锁", async () => {
    let resolveDelete!: () => void;
    const pendingDelete = new Promise<void>((resolve) => {
      resolveDelete = resolve;
    });

    const api = {
      get: vi.fn(),
      post: vi.fn().mockImplementation(async (path: string) => ({
        success: true,
        data: {
          locked: true,
          lockedBy: "user-1",
          lockedByName: `锁-${path}`,
          lockedAt: Date.now(),
        },
      })),
      delete: vi.fn().mockImplementation(async () => {
        await pendingDelete;
        return { success: true };
      }),
    };

    const manager = new PageLockManager({
      api,
      currentUserId: "user-1",
      heartbeatInterval: 60_000,
    });

    await manager.acquireLock("page-a");

    const releasePromise = manager.releaseLock();
    await Promise.resolve();

    await manager.acquireLock("page-b");
    resolveDelete();
    await releasePromise;

    expect(manager.getCurrentPageId()).toBe("page-b");
    expect(manager.getLockState()).toEqual(
      expect.objectContaining({
        pageId: "page-b",
        locked: true,
        isOwner: true,
      }),
    );
  });

  it("释放挂起期间仅丢失当前锁时仍会清理残留状态", async () => {
    let resolveDelete!: () => void;
    const pendingDelete = new Promise<void>((resolve) => {
      resolveDelete = resolve;
    });

    const api = {
      get: vi.fn(),
      post: vi.fn().mockResolvedValue({
        success: true,
        data: {
          locked: true,
          lockedBy: "user-1",
          lockedByName: "当前锁",
          lockedAt: Date.now(),
        },
      }),
      delete: vi.fn().mockImplementation(async () => {
        await pendingDelete;
        return { success: true };
      }),
    };

    const manager = new PageLockManager({
      api,
      currentUserId: "user-1",
      heartbeatInterval: 60_000,
    });

    await manager.acquireLock("page-a");

    const releasePromise = manager.releaseLock();
    await Promise.resolve();

    manager._handleLockLost();
    resolveDelete();
    await releasePromise;

    expect(manager.getCurrentPageId()).toBeNull();
    expect(manager.getLockState()).toBeNull();
  });
});
