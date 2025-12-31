const redis = require("../utils/redis");
const { logger } = require("../utils/logger");

/**
 * Redis Tag 值存储服务
 * 用于管理 MQTT 变量的实时值
 */
class RedisTagService {
  /**
   * 获取 Tag 值的 Redis Key
   * @param {string} tagId - Tag ID
   * @returns {string} Redis Key
   */
  getTagKey(tagId) {
    return `mqtt:tag:${tagId}`;
  }

  /**
   * 获取 Tag 历史值的 Redis Key
   * @param {string} tagId - Tag ID
   * @returns {string} Redis Key
   */
  getTagHistoryKey(tagId) {
    return `mqtt:tag:${tagId}:history`;
  }

  /**
   * 设置 Tag 的当前值
   * @param {string} tagId - Tag ID
   * @param {object} value - 值对象
   * @param {string} value.parsedValue - 解析后的值
   * @param {string} value.quality - 数据质量 (good/bad/uncertain)
   * @param {string} value.timestamp - 数据时间戳
   * @param {string} value.receivedAt - 接收时间
   * @param {string} value.error - 错误信息（可选）
   * @param {number} ttl - 过期时间（秒），默认 24 小时
   */
  async setTagValue(tagId, value, ttl = 86400) {
    try {
      const key = this.getTagKey(tagId);
      const data = {
        parsedValue: value.parsedValue,
        quality: value.quality || "good",
        timestamp: value.timestamp || new Date().toISOString(),
        receivedAt: value.receivedAt || new Date().toISOString(),
        error: value.error || null,
      };

      const client = redis.getRedis();
      await client.setex(key, ttl, JSON.stringify(data));

      logger.debug(
        `[RedisTagService] Set tag value for ${tagId}: ${data.parsedValue}`
      );
    } catch (error) {
      logger.error(
        `[RedisTagService] Error setting tag value for ${tagId}:`,
        error
      );
      throw error;
    }
  }

  /**
   * 获取 Tag 的当前值
   * @param {string} tagId - Tag ID
   * @returns {object|null} 值对象，如果不存在返回 null
   */
  async getTagValue(tagId) {
    try {
      const key = this.getTagKey(tagId);
      const client = redis.getRedis();
      const data = await client.get(key);

      if (!data) {
        return null;
      }

      return JSON.parse(data);
    } catch (error) {
      logger.error(
        `[RedisTagService] Error getting tag value for ${tagId}:`,
        error
      );
      throw error;
    }
  }

  /**
   * 批量获取 Tag 的当前值
   * @param {string[]} tagIds - Tag ID 数组
   * @returns {object} Tag ID 到值对象的映射
   */
  async getTagValues(tagIds) {
    try {
      if (!tagIds || tagIds.length === 0) {
        return {};
      }

      const keys = tagIds.map((id) => this.getTagKey(id));
      const client = redis.getRedis();
      const values = await client.mget(keys);

      const result = {};
      tagIds.forEach((tagId, index) => {
        if (values[index]) {
          try {
            result[tagId] = JSON.parse(values[index]);
          } catch (error) {
            logger.error(
              `[RedisTagService] Error parsing value for ${tagId}:`,
              error
            );
            result[tagId] = null;
          }
        } else {
          result[tagId] = null;
        }
      });

      return result;
    } catch (error) {
      logger.error("[RedisTagService] Error getting tag values:", error);
      throw error;
    }
  }

  /**
   * 删除 Tag 的当前值
   * @param {string} tagId - Tag ID
   */
  async deleteTagValue(tagId) {
    try {
      const key = this.getTagKey(tagId);
      const client = redis.getRedis();
      await client.del(key);

      logger.debug(`[RedisTagService] Deleted tag value for ${tagId}`);
    } catch (error) {
      logger.error(
        `[RedisTagService] Error deleting tag value for ${tagId}:`,
        error
      );
      throw error;
    }
  }

  /**
   * 添加 Tag 历史值（可选功能，保留最近 N 条）
   * @param {string} tagId - Tag ID
   * @param {object} value - 值对象
   * @param {number} maxHistory - 最大历史记录数，默认 100
   */
  async addTagHistory(tagId, value, maxHistory = 100) {
    try {
      const key = this.getTagHistoryKey(tagId);
      const data = {
        parsedValue: value.parsedValue,
        quality: value.quality || "good",
        timestamp: value.timestamp || new Date().toISOString(),
      };

      const client = redis.getRedis();

      // 添加到列表头部（最新的在前面）
      await client.lpush(key, JSON.stringify(data));

      // 保留最新的 N 条记录
      await client.ltrim(key, 0, maxHistory - 1);

      // 设置过期时间（24 小时）
      await client.expire(key, 86400);

      logger.debug(`[RedisTagService] Added history for ${tagId}`);
    } catch (error) {
      logger.error(
        `[RedisTagService] Error adding tag history for ${tagId}:`,
        error
      );
      // 历史记录失败不抛出异常，不影响主流程
    }
  }

  /**
   * 获取 Tag 历史值
   * @param {string} tagId - Tag ID
   * @param {number} limit - 获取数量，默认 100
   * @returns {array} 历史值数组
   */
  async getTagHistory(tagId, limit = 100) {
    try {
      const key = this.getTagHistoryKey(tagId);
      const client = redis.getRedis();
      const data = await client.lrange(key, 0, limit - 1);

      return data
        .map((item) => {
          try {
            return JSON.parse(item);
          } catch (error) {
            logger.error(
              `[RedisTagService] Error parsing history item:`,
              error
            );
            return null;
          }
        })
        .filter((item) => item !== null);
    } catch (error) {
      logger.error(
        `[RedisTagService] Error getting tag history for ${tagId}:`,
        error
      );
      throw error;
    }
  }

  /**
   * 清除 Tag 的所有数据（当前值 + 历史值）
   * @param {string} tagId - Tag ID
   */
  async clearTagData(tagId) {
    try {
      const valueKey = this.getTagKey(tagId);
      const historyKey = this.getTagHistoryKey(tagId);

      const client = redis.getRedis();
      await client.del(valueKey, historyKey);

      logger.debug(`[RedisTagService] Cleared all data for ${tagId}`);
    } catch (error) {
      logger.error(
        `[RedisTagService] Error clearing tag data for ${tagId}:`,
        error
      );
      throw error;
    }
  }

  /**
   * 批量删除 Tag 数据
   * @param {string[]} tagIds - Tag ID 数组
   */
  async clearTagsData(tagIds) {
    try {
      if (!tagIds || tagIds.length === 0) {
        return;
      }

      const keys = [];
      tagIds.forEach((tagId) => {
        keys.push(this.getTagKey(tagId));
        keys.push(this.getTagHistoryKey(tagId));
      });

      const client = redis.getRedis();
      await client.del(...keys);

      logger.debug(`[RedisTagService] Cleared data for ${tagIds.length} tags`);
    } catch (error) {
      logger.error("[RedisTagService] Error clearing tags data:", error);
      throw error;
    }
  }

  /**
   * 检查 Redis 连接状态
   * @returns {Promise<boolean>} 是否连接
   */
  async isConnected() {
    return await redis.isConnected();
  }

  /**
   * 获取 Tag 的 TTL（剩余生存时间）
   * @param {string} tagId - Tag ID
   * @returns {number} 剩余秒数，-1 表示永久，-2 表示不存在
   */
  async getTagTTL(tagId) {
    try {
      const key = this.getTagKey(tagId);
      const client = redis.getRedis();
      return await client.ttl(key);
    } catch (error) {
      logger.error(`[RedisTagService] Error getting TTL for ${tagId}:`, error);
      throw error;
    }
  }
}

module.exports = new RedisTagService();
