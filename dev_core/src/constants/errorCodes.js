/**
 * 统一错误码定义
 * 格式：A/B/C + 模块代码 + 错误序号
 * A: 认证授权相关
 * B: 业务逻辑相关
 * C: 系统错误相关
 */
const ErrorCodes = {
  // 成功
  SUCCESS: '00000',

  // 认证授权相关 (A0xxx)
  AUTH_TOKEN_REQUIRED: 'A0001',
  AUTH_TOKEN_INVALID: 'A0002',
  AUTH_TOKEN_EXPIRED: 'A0003',
  AUTH_USER_NOT_FOUND: 'A0004',
  AUTH_USER_INACTIVE: 'A0005',
  AUTH_TENANT_INACTIVE: 'A0006',
  AUTH_INVALID_CREDENTIALS: 'A0007',
  AUTH_TENANT_CODE_REQUIRED: 'A0008',
  AUTH_TENANT_CODE_INVALID: 'A0009',
  AUTH_INVALID_CAPTCHA: 'A0010',
  // 权限相关 (A1xxx)
  PERMISSION_DENIED: 'A1001',
  PERMISSION_INSUFFICIENT: 'A1002',
  PERMISSION_RESOURCE_OWNERSHIP: 'A1003',
  PERMISSION_TENANT_MISMATCH: 'A1004',
  PERMISSION_RATE_LIMIT_EXCEEDED: 'A1005',
  // 参数验证相关 (B0xxx)
  VALIDATION_FAILED: 'B0001',
  VALIDATION_REQUIRED: 'B0002',
  VALIDATION_INVALID_FORMAT: 'B0003',

  // 资源相关 (B1xxx)
  RESOURCE_NOT_FOUND: 'B1001',
  RESOURCE_ALREADY_EXISTS: 'B1002',
  RESOURCE_DELETE_SELF: 'B1003',
  RESOURCE_DELETE_SUPER_ADMIN: 'B1004',
  RESOURCE_INVALID_TYPE: 'B1005',

  // 租户相关 (B2xxx)
  TENANT_NOT_FOUND: 'B2001',
  TENANT_CODE_EXISTS: 'B2002',
  TENANT_CREATE_FAILED: 'B2003',

  // 用户相关 (B3xxx)
  USER_NOT_FOUND: 'B3001',
  USER_USERNAME_EXISTS: 'B3002',
  USER_CREATE_FAILED: 'B3003',
  USER_UPDATE_FAILED: 'B3004',
  USER_DELETE_FAILED: 'B3005',
  USER_ROLE_CHANGE_FAILED: 'B3006',

  // 工程相关 (B4xxx)
  PROJECT_NOT_FOUND: 'B4001',
  PROJECT_CODE_EXISTS: 'B4002',
  PROJECT_CREATE_FAILED: 'B4003',
  PROJECT_UPDATE_FAILED: 'B4004',
  PROJECT_DELETE_FAILED: 'B4005',
  PROJECT_OPERATION_FAILED: 'B4006',

  // 系统错误 (C0xxx)
  INTERNAL_SERVER_ERROR: 'C0001',
  // 兼容历史写法，避免遗留代码读取 undefined
  C0001: 'C0001',
  DATABASE_ERROR: 'C0002',
  EXTERNAL_SERVICE_ERROR: 'C0003',

  // 请求相关 (C1xxx)
  RATE_LIMIT_EXCEEDED: 'C1001',
  REQUEST_NOT_FOUND: 'C1002',
  REQUEST_METHOD_NOT_ALLOWED: 'C1003',

  // 设计中心相关 (B5xxx)
  DESIGN_PAGE_NOT_FOUND: 'B5001',
  DESIGN_SCHEMA_VALIDATION_FAILED: 'B5002',
  DESIGN_PAGE_LOCKED: 'B5003',
  DESIGN_FOLDER_NOT_EMPTY: 'B5004',
  DESIGN_INVALID_PARENT: 'B5005',
};

// 统一公开错误码前缀：A/B/C 分别映射到 1/2/3 万段。
const PUBLIC_CODE_PREFIX = {
  A: 10000,
  B: 20000,
  C: 30000,
};

const PUBLIC_CODE_UNKNOWN = 30000;
const LEGACY_ERROR_CODE_PATTERN = /^[ABC]\d{4}$/;

/**
 * 已定义的历史字符串错误码集合，用于避免生成不存在的 i18n key。
 */
const KNOWN_LEGACY_CODES = new Set(
  Object.values(ErrorCodes).filter(value => typeof value === 'string' && (LEGACY_ERROR_CODE_PATTERN.test(value) || value === '00000'))
);

/**
 * 将历史字符串错误码转换为对外整数错误码。
 * - 约定：Axxxx => 1xxxx，Bxxxx => 2xxxx，Cxxxx => 3xxxx
 * - 成功码固定为 0
 *
 * @param {string|number} errorCode
 * @returns {number}
 */
function toPublicCode(errorCode) {
  if (Number.isSafeInteger(errorCode)) {
    return errorCode;
  }

  const normalized = String(errorCode || '').trim().toUpperCase();
  if (normalized === '' || normalized === 'NULL' || normalized === 'UNDEFINED') {
    return PUBLIC_CODE_UNKNOWN;
  }
  if (normalized === 'SUCCESS' || normalized === '00000') {
    return 0;
  }

  const abcMatched = normalized.match(/^([ABC])(\d{4})$/);
  if (abcMatched) {
    const moduleCode = abcMatched[1];
    const sequence = Number.parseInt(abcMatched[2], 10);
    return (PUBLIC_CODE_PREFIX[moduleCode] || PUBLIC_CODE_UNKNOWN) + sequence;
  }

  if (/^\d+$/.test(normalized)) {
    const parsed = Number.parseInt(normalized, 10);
    if (Number.isSafeInteger(parsed)) {
      return parsed;
    }
  }

  return PUBLIC_CODE_UNKNOWN;
}

/**
 * 将错误码归一化为 i18n 读取使用的历史键。
 *
 * @param {string|number} errorCode
 * @returns {string}
 */
function toI18nCode(errorCode) {
  const normalized = String(errorCode || '').trim().toUpperCase();
  if (LEGACY_ERROR_CODE_PATTERN.test(normalized) && KNOWN_LEGACY_CODES.has(normalized)) {
    return normalized;
  }
  if (normalized === 'SUCCESS' || normalized === '00000') {
    return '00000';
  }

  if (Number.isSafeInteger(errorCode)) {
    const numeric = Number(errorCode);
    if (numeric === 0) {
      return '00000';
    }
    if (numeric >= 10000 && numeric < 20000) {
      const mappedCode = `A${String(numeric - 10000).padStart(4, '0')}`;
      return KNOWN_LEGACY_CODES.has(mappedCode) ? mappedCode : ErrorCodes.INTERNAL_SERVER_ERROR;
    }
    if (numeric >= 20000 && numeric < 30000) {
      const mappedCode = `B${String(numeric - 20000).padStart(4, '0')}`;
      return KNOWN_LEGACY_CODES.has(mappedCode) ? mappedCode : ErrorCodes.INTERNAL_SERVER_ERROR;
    }
    if (numeric >= 30000 && numeric < 40000) {
      const mappedCode = `C${String(numeric - 30000).padStart(4, '0')}`;
      return KNOWN_LEGACY_CODES.has(mappedCode) ? mappedCode : ErrorCodes.INTERNAL_SERVER_ERROR;
    }
    return ErrorCodes.INTERNAL_SERVER_ERROR;
  }

  if (/^\d+$/.test(normalized)) {
    return toI18nCode(Number.parseInt(normalized, 10));
  }

  return ErrorCodes.INTERNAL_SERVER_ERROR;
}

ErrorCodes.toPublicCode = toPublicCode;
ErrorCodes.toI18nCode = toI18nCode;

module.exports = ErrorCodes;

