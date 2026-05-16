import { ElMessageBox } from "element-plus";

// 危险动作二次确认，封装 ElMessageBox.confirm

export function useConfirm() {
  /**
   * 弹出确认对话框，用户点确认后执行 action。
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

  return { confirm, confirmAndRun };
}
