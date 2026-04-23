const ErrorCodes = require('../constants/errorCodes');
const { getErrorMessage, getSuccessMessage } = require('./i18n');

/**
 * 统一提取语言与请求 ID，确保四字段契约中的 reqId 始终存在（允许为 null）。
 *
 * @param {object} res
 * @returns {{language: string, reqId: string|null}}
 */
function resolveResponseContext(res) {
  const request = res.req || {};
  const locals = res.locals || {};
  const language = locals.language || request.language || 'zh-CN';
  const reqId = locals.requestId || request.requestId || null;

  res.locals = locals;
  res.locals.language = language;
  res.locals.requestId = reqId;
  res.setHeader('Content-Language', language);

  return { language, reqId };
}

/**
 * 构造统一输出契约，仅允许 code/msg/data/reqId 四个字段。
 *
 * @param {number} code
 * @param {string} msg
 * @param {any} data
 * @param {string|null} reqId
 * @returns {{code: number, msg: string, data: any, reqId: string|null}}
 */
function buildContractPayload(code, msg, data, reqId) {
  return { code, msg, data, reqId };
}

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
    const { language, reqId } = resolveResponseContext(res);
    const message = messageKey
      ? getSuccessMessage(language, messageKey, options)
      : getSuccessMessage(language, 'operation_success');

    return res.status(statusCode).json(
      buildContractPayload(
        ErrorCodes.toPublicCode(ErrorCodes.SUCCESS),
        message,
        data,
        reqId
      )
    );
  }

  /**
   * 错误响应
   * @param {object} res - Express 响应对象
   * @param {string} errorCode - 错误码
   * @param {object} options - 插值选项（可选），如果包含 message 字段则直接使用
   * @param {number} statusCode - HTTP 状态码（默认 400）
   */
  static error(res, errorCode, options = {}, statusCode = 400) {
    const normalizedOptions = options && typeof options === 'object' ? options : {};
    const { language, reqId } = resolveResponseContext(res);
    const message = normalizedOptions.message
      || getErrorMessage(language, ErrorCodes.toI18nCode(errorCode), normalizedOptions);

    return res.status(statusCode).json(
      buildContractPayload(
        ErrorCodes.toPublicCode(errorCode),
        message,
        null,
        reqId
      )
    );
  }

  /**
   * 分页响应
   * @param {object} res - Express 响应对象
   * @param {any} data - 响应数据
   * @param {object} pagination - 分页信息
   * @param {string} messageKey - 消息键（可选）
   */
  static paginated(res, data, pagination, messageKey = null) {
    const { language, reqId } = resolveResponseContext(res);
    const message = messageKey
      ? getSuccessMessage(language, messageKey)
      : getSuccessMessage(language, 'operation_success');

    return res.status(200).json(
      buildContractPayload(
        ErrorCodes.toPublicCode(ErrorCodes.SUCCESS),
        message,
        {
          list: data,
          pagination,
        },
        reqId
      )
    );
  }
}

module.exports = ApiResponse;

