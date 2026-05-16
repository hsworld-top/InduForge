import { ElMessageBox } from "element-plus";

// 危险动作二次确认，封装 ElMessageBox.confirm

export function useConfirm() {
  /**
   * 弹出确认对话框，用户点确认后返回 true，取消返回 false。
   * 用户取消时静默处理（不抛错）。
   */
  async function confirm(
    message: string,
    options: {
      title?: string;
      confirmText?: string;
      cancelText?: string;
      type?: "warning" | "error" | "info";
    } = {},
  ): Promise<boolean> {
    const {
      title = "确认操作",
      confirmText = "确定",
      cancelText = "取消",
      type = "warning",
    } = options;

    try {
      await ElMessageBox.confirm(message, title, {
        confirmButtonText: confirmText,
        cancelButtonText: cancelText,
        type,
      });
      return true;
    } catch {
      // 用户点了取消，静默返回 false
      return false;
    }
  }

  /**
   * 便捷方法：确认后执行 action，action 抛错会向上透传。
   */
  async function confirmAndRun(
    message: string,
    action: () => Promise<void> | void,
    options?: Parameters<typeof confirm>[1],
  ): Promise<void> {
    const ok = await confirm(message, options);
    if (ok) {
      await action();
    }
  }

  /**
   * 三选一草稿保护对话框：「丢弃」/「保留为草稿」/「取消」。
   * 返回值：
   *   'discard'  - 用户选择丢弃草稿
   *   'keep'     - 用户选择保留为草稿
   *   'cancel'   - 用户取消（不离开当前页面）
   */
  async function confirmDraftAction(options: {
    title?: string;
    message?: string;
  } = {}): Promise<"discard" | "keep" | "cancel"> {
    const {
      title = "有未保存的修改",
      message = "当前有未保存的修改，是否丢弃、保留为草稿还是继续编辑？",
    } = options;

    try {
      // 借助 ElMessageBox 的 distinguishCancelAndClose 区分"丢弃"与"关闭"
      // confirmButtonText = 丢弃，cancelButtonText = 保留为草稿
      // 用户点叉或 Escape 触发 close，视为取消
      await ElMessageBox.confirm(message, title, {
        confirmButtonText: "丢弃草稿",
        cancelButtonText: "保留为草稿",
        distinguishCancelAndClose: true,
        type: "warning",
        closeOnClickModal: false,
        // 关闭按钮会 reject('close')，cancel 按钮会 reject('cancel')
      });
      // 用户点了"丢弃草稿"
      return "discard";
    } catch (action) {
      if (action === "cancel") {
        // 用户点了"保留为草稿"
        return "keep";
      }
      // 用户点叉、Escape 或其他关闭方式，视为取消导航
      return "cancel";
    }
  }

  return { confirm, confirmAndRun, confirmDraftAction };
}
