import { REGEX } from '@/constants'

/**
 * 验证工具函数
 */

/**
 * 验证邮箱格式
 * @param {string} email - 邮箱地址
 * @returns {boolean} 是否有效
 */
export function isValidEmail(email) {
  return REGEX.EMAIL.test(email)
}

/**
 * 验证密码强度
 * @param {string} password - 密码
 * @returns {boolean} 是否有效
 */
export function isValidPassword(password) {
  return REGEX.PASSWORD.test(password)
}

/**
 * 验证手机号格式
 * @param {string} phone - 手机号
 * @returns {boolean} 是否有效
 */
export function isValidPhone(phone) {
  return REGEX.PHONE.test(phone)
}

/**
 * 验证必填字段
 * @param {*} value - 值
 * @returns {boolean} 是否有效
 */
export function isRequired(value) {
  if (Array.isArray(value)) {
    return value.length > 0
  }
  if (typeof value === 'string') {
    return value.trim().length > 0
  }
  return value !== null && value !== undefined
}

/**
 * 验证字符串长度
 * @param {string} str - 字符串
 * @param {number} min - 最小长度
 * @param {number} max - 最大长度
 * @returns {boolean} 是否有效
 */
export function isValidLength(str, min = 0, max = Infinity) {
  const length = str ? str.length : 0
  return length >= min && length <= max
}

/**
 * 验证数字范围
 * @param {number} num - 数字
 * @param {number} min - 最小值
 * @param {number} max - 最大值
 * @returns {boolean} 是否有效
 */
export function isValidRange(num, min = -Infinity, max = Infinity) {
  return num >= min && num <= max
}

/**
 * 自定义验证规则
 * @param {*} value - 值
 * @param {Function} validator - 验证函数
 * @returns {boolean} 是否有效
 */
export function customValidate(value, validator) {
  return validator(value)
}

/**
 * 表单验证器
 */
export class FormValidator {
  constructor() {
    this.rules = {}
    this.errors = {}
  }

  /**
   * 添加验证规则
   * @param {string} field - 字段名
   * @param {Array} rules - 验证规则数组
   */
  addRule(field, rules) {
    this.rules[field] = rules
  }

  /**
   * 验证单个字段
   * @param {string} field - 字段名
   * @param {*} value - 字段值
   * @returns {string|null} 错误信息或null
   */
  validateField(field, value) {
    const rules = this.rules[field]
    if (!rules) return null

    for (const rule of rules) {
      const { validator, message } = rule

      if (typeof validator === 'string') {
        // 预定义验证器
        if (!this[validator](value)) {
          return message
        }
      } else if (typeof validator === 'function') {
        // 自定义验证函数
        if (!validator(value)) {
          return message
        }
      }
    }

    return null
  }

  /**
   * 验证整个表单
   * @param {object} data - 表单数据
   * @returns {object} 验证结果 { isValid: boolean, errors: object }
   */
  validate(data) {
    const errors = {}
    let isValid = true

    for (const field in this.rules) {
      const error = this.validateField(field, data[field])
      if (error) {
        errors[field] = error
        isValid = false
      }
    }

    this.errors = errors
    return { isValid, errors }
  }

  // 预定义验证器
  required(value) {
    return isRequired(value)
  }

  email(value) {
    return !value || isValidEmail(value)
  }

  password(value) {
    return !value || isValidPassword(value)
  }

  phone(value) {
    return !value || isValidPhone(value)
  }
}
