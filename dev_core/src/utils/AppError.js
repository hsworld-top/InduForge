const ErrorCodes = require('../constants/errorCodes');

/**
 * 应用错误类
 * 用于统一错误处理
 */
class AppError extends Error {
  constructor(errorCode, statusCode = 400, options = {}) {
    super();
    this.errorCode = errorCode;
    this.statusCode = statusCode;
    this.options = options;
    this.name = 'AppError';
  }
}

module.exports = AppError;

