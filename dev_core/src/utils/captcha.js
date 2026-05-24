const svgCaptcha = require('svg-captcha')
const crypto = require('crypto')
const Redis = require('./redis')
const { logger } = require('./logger')

const DEFAULT_TTL_SECONDS = Number(process.env.CAPTCHA_TTL_SECONDS || 120)
const REDIS_KEY_PREFIX = 'captcha:'

/**
 * 生成一个验证码并保存到 Redis（带过期）
 * @returns {Promise<{ key: string, svg: string, expireSeconds: number }>}
 */
async function generateCaptcha() {
  const options = {
    size: 4,
    noise: 1,
    color: true,
    background: '#ffffff',
    ignoreChars: '0oO1ilI',
    width: 140,
    height: 44,
    fontSize: 56,
  }

  const captcha = svgCaptcha.create(options)
  // 生成请求唯一 key（不暴露明文）
  const key = crypto.randomUUID()
  const storeKey = REDIS_KEY_PREFIX + key

  // 存入 Redis（统一小写，便于校验时忽略大小写）
  const ok = await Redis.set(storeKey, captcha.text.toLowerCase(), DEFAULT_TTL_SECONDS)
  if (!ok) {
    logger.error('Failed to save captcha to Redis')
  }

  return {
    key,
    svg: captcha.data,
    expireSeconds: DEFAULT_TTL_SECONDS,
  }
}

/**
 * 校验验证码
 * @param {string} key - 生成时返回的 key
 * @param {string} code - 用户输入验证码
 * @param {object} options
 * @param {boolean} options.deleteOnCheck - 是否校验后立刻删除（默认 true，防止重放）
 * @returns {Promise<boolean>}
 */
async function verifyCaptcha(key, code, options = {}) {
  try {
    if (!key || !code) return false
    const { deleteOnCheck = true } = options
    const storeKey = REDIS_KEY_PREFIX + key
    const saved = await Redis.get(storeKey)
    if (!saved) return false
    const ok = String(saved).toLowerCase() === String(code).trim().toLowerCase()
    if (deleteOnCheck) {
      await Redis.del(storeKey)
    }
    return ok
  } catch (err) {
    logger.error('Captcha verify error', { error: err.message })
    return false
  }
}

/**
 * 主动失效验证码
 * @param {string} key
 * @returns {Promise<boolean>}
 */
async function invalidateCaptcha(key) {
  if (!key) return false
  const storeKey = REDIS_KEY_PREFIX + key
  return await Redis.del(storeKey)
}

module.exports = {
  generateCaptcha,
  verifyCaptcha,
  invalidateCaptcha,
}
