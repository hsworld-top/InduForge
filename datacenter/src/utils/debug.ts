/** 开发态诊断统一出口；生产构建不会输出内部对象或错误详情。 */
const enabled = import.meta.env.DEV

export const debugLogger = {
  log: (...args: unknown[]) => {
    if (enabled) console.debug(...args)
  },
  warn: (...args: unknown[]) => {
    if (enabled) console.warn(...args)
  },
  error: (...args: unknown[]) => {
    if (enabled) console.error(...args)
  },
}
