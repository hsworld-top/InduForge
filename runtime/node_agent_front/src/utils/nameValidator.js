/**
 * 节点名称验证工具
 * @description 验证节点名称是否符合命名规范
 */

// 节点名称正则：仅允许字母、数字、中划线、下划线，长度3-50个字符
const NAME_PATTERN = /^[a-zA-Z0-9_-]{3,50}$/

/**
 * 验证节点名称
 * @param {string} name - 节点名称
 * @returns {Object} 验证结果 { valid: boolean, message: string }
 */
export function validateNodeName(name) {
  if (!name) {
    return {
      valid: false,
      message: '节点名称不能为空',
    }
  }

  if (name.length < 3) {
    return {
      valid: false,
      message: '节点名称长度不能少于3个字符',
    }
  }

  if (name.length > 50) {
    return {
      valid: false,
      message: '节点名称长度不能超过50个字符',
    }
  }

  if (!NAME_PATTERN.test(name)) {
    return {
      valid: false,
      message: '节点名称仅允许字母、数字、中划线、下划线',
    }
  }

  return {
    valid: true,
    message: '名称格式正确',
  }
}

/**
 * 获取名称规则提示
 * @returns {string} 规则提示文本
 */
export function getNameRuleHint() {
  return '仅允许字母、数字、中划线、下划线，长度3-50个字符'
}

/**
 * Element Plus 表单验证器
 * @param {Object} rule - 验证规则
 * @param {string} value - 输入值
 * @param {Function} callback - 回调函数
 */
export function nodeNameValidator(rule, value, callback) {
  const result = validateNodeName(value)
  if (result.valid) {
    callback()
  } else {
    callback(new Error(result.message))
  }
}
