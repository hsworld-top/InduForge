const i18next = require('i18next')
const Backend = require('i18next-fs-backend')
const path = require('path')

// 初始化 i18next
i18next.use(Backend).init({
  lng: 'zh-CN', // 默认语言
  fallbackLng: 'en', // 回退语言
  backend: {
    loadPath: path.join(__dirname, '../locales/{{lng}}.json'),
  },
  interpolation: {
    escapeValue: false, // React 不需要转义
  },
  // 不自动检测语言，由中间件控制
  detection: {
    order: [],
  },
})

/**
 * 获取翻译文本
 * @param {string} lng - 语言代码 (zh-CN, en)
 * @param {string} key - 翻译键
 * @param {object} options - 插值选项
 * @returns {string} 翻译后的文本
 */
function t(lng, key, options = {}) {
  return i18next.getFixedT(lng)(key, options)
}

/**
 * 获取错误消息
 * @param {string} lng - 语言代码
 * @param {string} errorCode - 错误码
 * @param {object} options - 插值选项
 * @returns {string} 错误消息
 */
function getErrorMessage(lng, errorCode, options = {}) {
  return t(lng, `error.${errorCode}`, options) || t(lng, 'error.C0001')
}

/**
 * 获取成功消息
 * @param {string} lng - 语言代码
 * @param {string} messageKey - 消息键
 * @param {object} options - 插值选项
 * @returns {string} 成功消息
 */
function getSuccessMessage(lng, messageKey, options = {}) {
  return t(lng, `message.${messageKey}`, options) || t(lng, 'message.operation_success')
}

module.exports = {
  i18next,
  t,
  getErrorMessage,
  getSuccessMessage,
}
