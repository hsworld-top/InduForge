import { REGEX } from '@/constants'

/**
 * 验证工具函数
 */

export interface ValidationRule {
  validator: 'required' | 'email' | 'password' | 'phone' | ((value: unknown) => boolean)
  message: string
}

/**
 * 验证邮箱格式
 * @param email 邮箱地址
 * @returns 是否有效
 */
export function isValidEmail(email: string): boolean {
  return REGEX.EMAIL.test(email)
}

/**
 * 验证密码强度
 * @param password 密码
 * @returns 是否有效
 */
export function isValidPassword(password: string): boolean {
  return REGEX.PASSWORD.test(password)
}

/**
 * 验证手机号格式
 * @param phone 手机号
 * @returns 是否有效
 */
export function isValidPhone(phone: string): boolean {
  return REGEX.PHONE.test(phone)
}

/**
 * 验证必填字段
 * @param value 值
 * @returns 是否有效
 */
export function isRequired(value: unknown): boolean {
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
 * @param str 字符串
 * @param min 最小长度
 * @param max 最大长度
 * @returns 是否有效
 */
export function isValidLength(str: string, min = 0, max = Infinity): boolean {
  const length = str ? str.length : 0
  return length >= min && length <= max
}

/**
 * 验证数字范围
 * @param num 数字
 * @param min 最小值
 * @param max 最大值
 * @returns 是否有效
 */
export function isValidRange(num: number, min = -Infinity, max = Infinity): boolean {
  return num >= min && num <= max
}

/**
 * 自定义验证规则
 * @param value 值
 * @param validator 验证函数
 * @returns 是否有效
 */
export function customValidate(value: unknown, validator: (value: unknown) => boolean): boolean {
  return validator(value)
}

/**
 * 表单验证器
 */
export class FormValidator {
  private rules: Record<string, ValidationRule[]>

  errors: Record<string, string>

  constructor() {
    this.rules = {}
    this.errors = {}
  }

  /**
   * 添加验证规则
   * @param field 字段名
   * @param rules 验证规则数组
   */
  addRule(field: string, rules: ValidationRule[]): void {
    this.rules[field] = rules
  }

  /**
   * 验证单个字段
   * @param field 字段名
   * @param value 字段值
   * @returns 错误信息或null
   */
  validateField(field: string, value: unknown): string | null {
    const rules = this.rules[field]
    if (!rules) return null

    for (const rule of rules) {
      const { validator, message } = rule

      if (typeof validator === 'string') {
        if (!this[validator](value)) {
          return message
        }
      } else if (typeof validator === 'function') {
        if (!validator(value)) {
          return message
        }
      }
    }

    return null
  }

  /**
   * 验证整个表单
   * @param data 表单数据
   * @returns 验证结果 { isValid: boolean, errors: object }
   */
  validate(data: Record<string, unknown>): {
    isValid: boolean
    errors: Record<string, string>
  } {
    const errors: Record<string, string> = {}
    let isValid = true

    for (const field of Object.keys(this.rules)) {
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
  required(value: unknown): boolean {
    return isRequired(value)
  }

  email(value: unknown): boolean {
    return !value || isValidEmail(String(value))
  }

  password(value: unknown): boolean {
    return !value || isValidPassword(String(value))
  }

  phone(value: unknown): boolean {
    return !value || isValidPhone(String(value))
  }
}
