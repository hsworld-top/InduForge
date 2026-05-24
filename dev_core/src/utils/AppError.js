const ErrorCodes = require('../constants/errorCodes')

/**
 * 应用错误类
 * 用于统一错误处理
 */
class AppError extends Error {
  constructor(errorCode, statusCode = 400, options = {}) {
    const normalizedOptions = options && typeof options === 'object' ? options : {}
    const message = typeof normalizedOptions.message === 'string' ? normalizedOptions.message : ''

    // Error.message 仍保留，以便日志与调试栈可读。
    super(message || '应用错误')

    // 兼容历史字段（errorCode），并补充新契约字段（code/msg）。
    this.errorCode = errorCode
    this.code = ErrorCodes.toPublicCode(errorCode)
    this.msg = message
    this.statusCode = statusCode
    this.options = normalizedOptions
    this.name = 'AppError'

    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, AppError)
    }
  }
}

module.exports = AppError
