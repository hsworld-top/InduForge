import { ElMessage } from "element-plus";
import { ZodError } from "zod";
import { ApiBusinessError } from "@/utils/request";

// 统一 API 错误处理 composable
// 识别 ApiBusinessError / ZodError / 网络错误，给统一 toast

/**
 * 识别错误类型并给出 toast 提示。
 * @param err 捕获到的错误
 * @param fallback 兜底提示文本，默认"操作失败"
 */
export function handle(err: unknown, fallback = "操作失败"): void {
  if (err instanceof ApiBusinessError) {
    ElMessage.error(err.message || fallback);
    return;
  }

  if (err instanceof ZodError) {
    // 数据格式与预期不符，通常是后端响应结构变了
    const first = err.issues[0];
    const msg = first ? `数据格式异常：${first.path.join(".")} ${first.message}` : "数据格式异常";
    ElMessage.error(msg);
    return;
  }

  if (err instanceof Error) {
    // 网络错误（axios 网络层失败）或其他 JS 错误
    if (err.message.includes("Network Error") || err.message.includes("timeout")) {
      ElMessage.error("网络异常，请检查连接后重试");
      return;
    }
    ElMessage.error(err.message || fallback);
    return;
  }

  ElMessage.error(fallback);
}

export function useApiError() {
  return { handle };
}
