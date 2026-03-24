import { ElMessage as Raw } from "element-plus";

/** Element Plus 类型对字符串 message 过严，与项目 request.ts 中 `as never` 策略一致 */
export const ElMessage = {
  success: (message: string) => Raw.success(message as never),
  error: (message: string) => Raw.error(message as never),
  warning: (message: string) => Raw.warning(message as never),
  info: (message: string) => Raw.info(message as never),
};
