const jwt = require('jsonwebtoken')
const crypto = require('crypto')
const dayjs = require('dayjs')
const { logger } = require('./logger')
const redis = require('./redis')
const appConfig = require('../config/app')
const { TIME_FORMAT } = require('../constants/time')

/**
 * Token 管理工具类
 * 实现 Access Token (JWT) + Refresh Token (Opaque) 双令牌体系
 */
class TokenManager {
  /**
   * 生成 Access Token (JWT)
   * @param {object} payload - JWT 载荷
   * @returns {string} Access Token
   */
  static generateAccessToken(payload) {
    const secret = process.env.JWT_SECRET || process.env.JWT_ACCESS_SECRET
    if (!secret) {
      throw new Error('JWT_SECRET or JWT_ACCESS_SECRET is not configured')
    }

    return jwt.sign(payload, secret, {
      expiresIn: appConfig.jwt.accessExpiresIn,
      issuer: process.env.JWT_ISSUER || 'tenant-management',
      audience: process.env.JWT_AUDIENCE || 'tenant-management-api',
    })
  }

  /**
   * 生成 Refresh Token (Opaque - 随机字符串)
   * @returns {string} Refresh Token
   */
  static generateRefreshToken() {
    return crypto.randomBytes(32).toString('hex')
  }

  /**
   * 验证 Access Token
   * @param {string} token - Access Token
   * @returns {object|null} 解码后的载荷，失败返回 null
   */
  static verifyAccessToken(token) {
    try {
      const secret = process.env.JWT_SECRET || process.env.JWT_ACCESS_SECRET
      if (!secret) {
        throw new Error('JWT_SECRET or JWT_ACCESS_SECRET is not configured')
      }

      return jwt.verify(token, secret, {
        issuer: process.env.JWT_ISSUER || 'tenant-management',
        audience: process.env.JWT_AUDIENCE || 'tenant-management-api',
      })
    } catch (error) {
      logger.debug('Access token verification failed', { error: error.message })
      return null
    }
  }

  /**
   * 存储 Refresh Token 到 Redis
   * @param {string} refreshToken - Refresh Token
   * @param {string} userId - 用户 ID
   * @param {string} tenantId - 租户 ID
   * @returns {Promise<boolean>} 是否成功
   */
  static async storeRefreshToken(refreshToken, userId, tenantId) {
    try {
      const key = this.getRefreshTokenKey(refreshToken)
      const refreshExpiresIn = this.getRefreshTokenExpireSeconds()

      const tokenData = {
        userId,
        tenantId,
        createdAt: dayjs().format(TIME_FORMAT),
      }

      return await redis.set(key, tokenData, refreshExpiresIn)
    } catch (error) {
      logger.error('Store refresh token error', { error: error.message })
      return false
    }
  }

  /**
   * 验证并获取 Refresh Token 信息
   * @param {string} refreshToken - Refresh Token
   * @returns {Promise<object|null>} Token 信息，失败返回 null
   */
  static async verifyRefreshToken(refreshToken) {
    try {
      const key = this.getRefreshTokenKey(refreshToken)
      const tokenData = await redis.getJSON(key)

      if (!tokenData) {
        return null
      }

      return tokenData
    } catch (error) {
      logger.error('Verify refresh token error', { error: error.message })
      return null
    }
  }

  /**
   * 撤销 Refresh Token
   * @param {string} refreshToken - Refresh Token
   * @returns {Promise<boolean>} 是否成功
   */
  static async revokeRefreshToken(refreshToken) {
    try {
      const key = this.getRefreshTokenKey(refreshToken)
      return await redis.del(key)
    } catch (error) {
      logger.error('Revoke refresh token error', { error: error.message })
      return false
    }
  }

  /**
   * 撤销用户的所有 Refresh Token
   * @param {string} userId - 用户 ID
   * @returns {Promise<number>} 撤销的 Token 数量
   */
  static async revokeAllUserRefreshTokens(userId) {
    try {
      const pattern = this.getUserRefreshTokenPattern(userId)
      const redisClient = redis.getRedis()
      const keys = await redisClient.keys(pattern)
      let count = 0

      // 检查每个 token 是否属于该用户
      for (const key of keys) {
        const tokenData = await redis.getJSON(key)
        if (tokenData && tokenData.userId === userId) {
          await redis.del(key)
          count++
        }
      }

      return count
    } catch (error) {
      logger.error('Revoke all user refresh tokens error', { userId, error: error.message })
      return 0
    }
  }

  /**
   * 将 Access Token 加入黑名单（用于登出时撤销）
   * @param {string} accessToken - Access Token
   * @param {number} expireSeconds - 过期时间（秒），默认使用 Access Token 的剩余时间
   * @returns {Promise<boolean>} 是否成功
   */
  static async blacklistAccessToken(accessToken, expireSeconds = null) {
    try {
      const key = this.getAccessTokenBlacklistKey(accessToken)

      // 如果没有指定过期时间，尝试从 token 中获取剩余时间
      if (!expireSeconds) {
        const decoded = jwt.decode(accessToken)
        if (decoded && decoded.exp) {
          const now = Math.floor(Date.now() / 1000)
          expireSeconds = Math.max(decoded.exp - now, 0)
        } else {
          // 如果无法解码，使用默认的 Access Token 过期时间
          expireSeconds = this.getAccessTokenExpireSeconds()
        }
      }

      if (expireSeconds <= 0) {
        return true // Token 已过期，无需加入黑名单
      }

      return await redis.set(key, '1', expireSeconds)
    } catch (error) {
      logger.error('Blacklist access token error', { error: error.message })
      return false
    }
  }

  /**
   * 检查 Access Token 是否在黑名单中
   * @param {string} accessToken - Access Token
   * @returns {Promise<boolean>} 是否在黑名单中
   */
  static async isAccessTokenBlacklisted(accessToken) {
    try {
      const key = this.getAccessTokenBlacklistKey(accessToken)
      return await redis.exists(key)
    } catch (error) {
      logger.error('Check access token blacklist error', { error: error.message })
      return false
    }
  }

  /**
   * 生成一对 Token（Access Token + Refresh Token）
   * @param {object} payload - JWT 载荷
   * @param {string} userId - 用户 ID
   * @param {string} tenantId - 租户 ID
   * @returns {Promise<object>} { accessToken, refreshToken }
   */
  static async generateTokenPair(payload, userId, tenantId) {
    const accessToken = this.generateAccessToken(payload)
    const refreshToken = this.generateRefreshToken()

    // 存储 Refresh Token
    await this.storeRefreshToken(refreshToken, userId, tenantId)

    return {
      accessToken,
      refreshToken,
    }
  }

  /**
   * 使用 Refresh Token 刷新 Access Token
   * @param {string} refreshToken - Refresh Token
   * @param {object} newPayload - 新的 JWT 载荷
   * @returns {Promise<object|null>} { accessToken, refreshToken } 或 null
   */
  static async refreshAccessToken(refreshToken, newPayload) {
    // 验证 Refresh Token
    const tokenData = await this.verifyRefreshToken(refreshToken)
    if (!tokenData) {
      return null
    }

    // 生成新的 Access Token
    const accessToken = this.generateAccessToken(newPayload)

    // 可选：生成新的 Refresh Token（刷新令牌轮换）
    const rotateRefreshToken = process.env.ROTATE_REFRESH_TOKEN === 'true'
    let newRefreshToken = refreshToken

    if (rotateRefreshToken) {
      // 撤销旧的 Refresh Token
      await this.revokeRefreshToken(refreshToken)

      // 生成新的 Refresh Token
      newRefreshToken = this.generateRefreshToken()
      await this.storeRefreshToken(newRefreshToken, tokenData.userId, tokenData.tenantId)
    }

    return {
      accessToken,
      refreshToken: newRefreshToken,
    }
  }

  /**
   * 获取 Refresh Token 的 Redis 键
   * @param {string} refreshToken - Refresh Token
   * @returns {string} Redis 键
   */
  static getRefreshTokenKey(refreshToken) {
    return `refresh_token:${refreshToken}`
  }

  /**
   * 获取用户所有 Refresh Token 的模式
   * @param {string} userId - 用户 ID
   * @returns {string} Redis 键模式
   */
  static getUserRefreshTokenPattern(userId) {
    // 注意：由于 Refresh Token 是随机字符串，无法直接通过模式匹配用户
    // 需要在 revokeAllUserRefreshTokens 中遍历所有 token 并检查 userId
    return `refresh_token:*`
  }

  /**
   * 获取 Access Token 黑名单的 Redis 键
   * @param {string} accessToken - Access Token
   * @returns {string} Redis 键
   */
  static getAccessTokenBlacklistKey(accessToken) {
    // 使用 token 的哈希值作为键，避免键过长
    const hash = crypto.createHash('sha256').update(accessToken).digest('hex')
    return `access_token_blacklist:${hash}`
  }

  /**
   * 获取 Refresh Token 过期时间（秒）
   * @returns {number} 过期时间（秒）
   */
  static getRefreshTokenExpireSeconds() {
    const refreshExpiresIn = appConfig.jwt.refreshExpiresIn

    // 解析时间字符串（如 '7d', '30d'）
    const match = refreshExpiresIn.match(/^(\d+)([smhd])$/)
    if (!match) {
      return 7 * 24 * 60 * 60 // 默认 7 天
    }

    const value = parseInt(match[1])
    const unit = match[2]

    switch (unit) {
      case 's':
        return value
      case 'm':
        return value * 60
      case 'h':
        return value * 60 * 60
      case 'd':
        return value * 24 * 60 * 60
      default:
        return 7 * 24 * 60 * 60
    }
  }

  /**
   * 获取 Access Token 过期时间（秒）
   * @returns {number} 过期时间（秒）
   */
  static getAccessTokenExpireSeconds() {
    const accessExpiresIn = appConfig.jwt.accessExpiresIn

    // 解析时间字符串（如 '15m', '1h'）
    const match = accessExpiresIn.match(/^(\d+)([smhd])$/)
    if (!match) {
      return 15 * 60 // 默认 15 分钟
    }

    const value = parseInt(match[1])
    const unit = match[2]

    switch (unit) {
      case 's':
        return value
      case 'm':
        return value * 60
      case 'h':
        return value * 60 * 60
      case 'd':
        return value * 24 * 60 * 60
      default:
        return 15 * 60
    }
  }
}

module.exports = TokenManager
