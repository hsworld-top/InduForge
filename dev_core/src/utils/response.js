const ErrorCodes = require('../constants/errorCodes');
const { getErrorMessage, getSuccessMessage } = require('./i18n');

/**
 * 统一响应格式
 */
class ApiResponse {
  /**
   * 成功响应
   * @param {object} res - Express 响应对象
   * @param {any} data - 响应数据
   * @param {string} messageKey - 消息键（可选）
   * @param {object} options - 插值选项（可选）
   * @param {number} statusCode - HTTP 状态码（默认 200）
   */
  static success(res, data = null, messageKey = null, options = {}, statusCode = 200) {
    // 自动从 req 或 res.locals 获取 language 和 requestId
    const language = res.locals.language || res.req.language || 'zh-CN';
    const requestId = res.locals.requestId || res.req.requestId;
    
    // 确保 res.locals 中有这些值
    res.locals.language = language;
    res.locals.requestId = requestId;
    
    // 设置响应头中的语言信息
    res.setHeader('Content-Language', language);
    
    const response = {
      success: true,
      errorCode: ErrorCodes.SUCCESS,
      message: messageKey ? getSuccessMessage(language, messageKey, options) : getSuccessMessage(language, 'operation_success'),
      requestId,
    };

    if (data !== null) {
      response.data = data;
    }

    return res.status(statusCode).json(response);
  }

  /**
   * 错误响应
   * @param {object} res - Express 响应对象
   * @param {string} errorCode - 错误码
   * @param {object} options - 插值选项（可选）
   * @param {number} statusCode - HTTP 状态码（默认 400）
   */
  static error(res, errorCode, options = {}, statusCode = 400) {
    // 自动从 req 或 res.locals 获取 language 和 requestId
    const language = res.locals.language || res.req.language || 'zh-CN';
    const requestId = res.locals.requestId || res.req.requestId;
    
    // 确保 res.locals 中有这些值
    res.locals.language = language;
    res.locals.requestId = requestId;
    
    // 设置响应头中的语言信息
    res.setHeader('Content-Language', language);
    
    const response = {
      success: false,
      errorCode,
      message: getErrorMessage(language, errorCode, options),
      requestId,
    };

    return res.status(statusCode).json(response);
  }

  /**
   * 分页响应
   * @param {object} res - Express 响应对象
   * @param {any} data - 响应数据
   * @param {object} pagination - 分页信息
   * @param {string} messageKey - 消息键（可选）
   */
  static paginated(res, data, pagination, messageKey = null) {
    // 自动从 req 或 res.locals 获取 language 和 requestId
    const language = res.locals.language || res.req.language || 'zh-CN';
    const requestId = res.locals.requestId || res.req.requestId;
    
    // 确保 res.locals 中有这些值
    res.locals.language = language;
    res.locals.requestId = requestId;
    
    // 设置响应头中的语言信息
    res.setHeader('Content-Language', language);
    
    const response = {
      success: true,
      errorCode: ErrorCodes.SUCCESS,
      message: messageKey ? getSuccessMessage(language, messageKey) : getSuccessMessage(language, 'operation_success'),
      data,
      pagination,
      requestId,
    };

    return res.status(200).json(response);
  }
}

module.exports = ApiResponse;

