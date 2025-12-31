/**
 * MQTT Tag 服务
 * 负责 Tag 的 CRUD 操作和值管理
 */

const { logger } = require("../utils/logger");
const { AppError } = require("../utils/AppError");
const { ErrorCodes } = require("../constants/errorCodes");
const {
  DataMqttTag,
  DataMqttSubscription,
  Project,
  User,
} = require("../models");
const { parseMessage, parseMessageBatch } = require("./messageParser");
const socketService = require("./socketService");
const redisTagService = require("./redisTagService");

class MqttTagService {
  /**
   * 创建 Tag
   */
  async createTag(projectId, subscriptionId, tagData, userId) {
    // 验证订阅是否存在
    const subscription = await DataMqttSubscription.findOne({
      where: { id: subscriptionId, projectId },
    });

    if (!subscription) {
      throw new AppError("订阅不存在", ErrorCodes.RESOURCE_NOT_FOUND);
    }

    // 检查 code 是否重复
    const existing = await DataMqttTag.findOne({
      where: { projectId, code: tagData.code },
    });

    if (existing) {
      throw new AppError(
        `变量标识符 ${tagData.code} 已存在`,
        ErrorCodes.DUPLICATE_ENTRY
      );
    }

    // 创建 Tag
    const tag = await DataMqttTag.create({
      ...tagData,
      projectId,
      subscriptionId,
      createdBy: userId,
    });

    logger.info(`[MqttTagService] Created tag: ${tag.code} (${tag.id})`);

    return tag;
  }

  /**
   * 更新 Tag
   */
  async updateTag(tagId, tagData, userId) {
    const tag = await DataMqttTag.findByPk(tagId);

    if (!tag) {
      throw new AppError("Tag不存在", ErrorCodes.RESOURCE_NOT_FOUND);
    }

    // 如果修改了 code，检查是否重复
    if (tagData.code && tagData.code !== tag.code) {
      const existing = await DataMqttTag.findOne({
        where: { projectId: tag.projectId, code: tagData.code },
      });

      if (existing) {
        throw new AppError(
          `变量标识符 ${tagData.code} 已存在`,
          ErrorCodes.DUPLICATE_ENTRY
        );
      }
    }

    // 更新 Tag
    await tag.update({
      ...tagData,
      updatedBy: userId,
    });

    logger.info(`[MqttTagService] Updated tag: ${tag.code} (${tag.id})`);

    return tag;
  }

  /**
   * 删除 Tag
   */
  async deleteTag(tagId) {
    const tag = await DataMqttTag.findByPk(tagId);

    if (!tag) {
      throw new AppError("Tag不存在", ErrorCodes.RESOURCE_NOT_FOUND);
    }

    // 删除数据库记录
    await tag.destroy();

    // 清理 Redis 中的数据
    try {
      await redisTagService.clearTagData(tagId);
    } catch (error) {
      logger.warn(
        `[MqttTagService] Failed to clear Redis data for tag ${tagId}:`,
        error
      );
    }

    logger.info(`[MqttTagService] Deleted tag: ${tag.code} (${tag.id})`);

    return { success: true };
  }

  /**
   * 获取 Tag 详情
   */
  async getTag(tagId) {
    const tag = await DataMqttTag.findByPk(tagId, {
      include: [
        {
          model: DataMqttSubscription,
          as: "subscription",
          attributes: ["id", "name", "topic"],
        },
        {
          model: User,
          as: "creator",
          attributes: ["id", "username", "fullName"],
        },
      ],
    });

    if (!tag) {
      throw new AppError("Tag不存在", ErrorCodes.RESOURCE_NOT_FOUND);
    }

    // 从 Redis 获取当前值
    const currentValue = await redisTagService.getTagValue(tagId);

    // 将值附加到 tag 对象
    const tagData = tag.toJSON();
    tagData.currentValue = currentValue;

    return tagData;
  }

  /**
   * 获取订阅的所有 Tag
   */
  async getTagsBySubscription(subscriptionId, options = {}) {
    const { page = 1, pageSize = 50, isEnabled } = options;

    const where = { subscriptionId };
    if (isEnabled !== undefined) {
      where.isEnabled = isEnabled;
    }

    const { rows: tags, count: total } = await DataMqttTag.findAndCountAll({
      where,
      order: [
        ["order", "ASC"],
        ["createdAt", "ASC"],
      ],
      limit: pageSize,
      offset: (page - 1) * pageSize,
    });

    // 批量从 Redis 获取当前值
    if (tags.length > 0) {
      const tagIds = tags.map((t) => t.id);
      const values = await redisTagService.getTagValues(tagIds);

      // 将值附加到每个 tag
      tags.forEach((tag) => {
        const tagData = tag.toJSON ? tag.toJSON() : tag;
        tagData.currentValue = values[tag.id] || null;
        Object.assign(tag, tagData);
      });
    }

    return {
      tags,
      pagination: {
        total,
        page,
        pageSize,
        totalPages: Math.ceil(total / pageSize),
      },
    };
  }

  /**
   * 获取项目的所有 Tag
   */
  async getTagsByProject(projectId, options = {}) {
    const { page = 1, pageSize = 50, subscriptionId, isEnabled } = options;

    const where = { projectId };
    if (subscriptionId) {
      where.subscriptionId = subscriptionId;
    }
    if (isEnabled !== undefined) {
      where.isEnabled = isEnabled;
    }

    const { rows: tags, count: total } = await DataMqttTag.findAndCountAll({
      where,
      include: [
        {
          model: DataMqttSubscription,
          as: "subscription",
          attributes: ["id", "name", "topic"],
        },
      ],
      order: [
        ["order", "ASC"],
        ["createdAt", "ASC"],
      ],
      limit: pageSize,
      offset: (page - 1) * pageSize,
    });

    // 批量从 Redis 获取当前值
    if (tags.length > 0) {
      const tagIds = tags.map((t) => t.id);
      const values = await redisTagService.getTagValues(tagIds);

      // 将值附加到每个 tag
      tags.forEach((tag) => {
        const tagData = tag.toJSON ? tag.toJSON() : tag;
        tagData.currentValue = values[tag.id] || null;
        Object.assign(tag, tagData);
      });
    }

    return {
      tags,
      pagination: {
        total,
        page,
        pageSize,
        totalPages: Math.ceil(total / pageSize),
      },
    };
  }

  /**
   * 处理 MQTT 消息并更新相关 Tag 值
   * 当订阅收到消息时调用此方法
   */
  async processMessage(subscriptionId, topic, message) {
    try {
      // 获取该订阅下所有启用的 Tag
      const tags = await DataMqttTag.findAll({
        where: { subscriptionId, isEnabled: true },
      });

      if (tags.length === 0) {
        return;
      }

      // 批量解析消息
      const parseResults = parseMessageBatch(message, tags);

      // 当前时间戳
      const now = new Date().toISOString();

      // 更新每个 Tag 的值
      const updatePromises = Object.entries(parseResults).map(
        async ([tagCode, result]) => {
          const tag = tags.find((t) => t.code === tagCode);
          if (!tag) return;

          try {
            // 将值存储到 Redis
            await redisTagService.setTagValue(tag.id, {
              parsedValue:
                result.value !== null && result.value !== undefined
                  ? typeof result.value === "object"
                    ? JSON.stringify(result.value)
                    : String(result.value)
                  : null,
              quality: result.quality,
              timestamp: now,
              receivedAt: now,
              error: result.error,
            });

            // 可选：同时保存历史记录（最近100条）
            await redisTagService.addTagHistory(tag.id, {
              parsedValue: result.value,
              quality: result.quality,
              timestamp: now,
            });

            // 通过 Socket.IO 广播值更新
            socketService.broadcastTagValueUpdate(tag.id, {
              tagId: tag.id,
              tagCode: tag.code,
              tagName: tag.name,
              value: result.value,
              quality: result.quality,
              timestamp: now,
              error: result.error,
            });

            logger.debug(
              `[MqttTagService] Updated tag value: ${tag.code} = ${result.value}`
            );
          } catch (error) {
            logger.error(
              `[MqttTagService] Failed to update tag ${tag.code}:`,
              error
            );
          }
        }
      );

      await Promise.all(updatePromises);
    } catch (error) {
      logger.error(
        `[MqttTagService] Error processing message for subscription ${subscriptionId}:`,
        error
      );
    }
  }

  /**
   * 批量创建 Tag
   */
  async createTagsBatch(projectId, subscriptionId, tagsData, userId) {
    // 验证订阅是否存在
    const subscription = await DataMqttSubscription.findOne({
      where: { id: subscriptionId, projectId },
    });

    if (!subscription) {
      throw new AppError("订阅不存在", ErrorCodes.RESOURCE_NOT_FOUND);
    }

    // 检查 code 是否重复
    const codes = tagsData.map((t) => t.code);
    const existingTags = await DataMqttTag.findAll({
      where: { projectId, code: codes },
    });

    if (existingTags.length > 0) {
      const duplicateCodes = existingTags.map((t) => t.code).join(", ");
      throw new AppError(
        `变量标识符已存在: ${duplicateCodes}`,
        ErrorCodes.DUPLICATE_ENTRY
      );
    }

    // 批量创建
    const tags = await DataMqttTag.bulkCreate(
      tagsData.map((data, index) => ({
        ...data,
        projectId,
        subscriptionId,
        createdBy: userId,
        order: data.order !== undefined ? data.order : index,
      }))
    );

    logger.info(
      `[MqttTagService] Created ${tags.length} tags for subscription ${subscriptionId}`
    );

    return tags;
  }

  /**
   * 切换 Tag 启用状态
   */
  async toggleTag(tagId, userId) {
    const tag = await DataMqttTag.findByPk(tagId);

    if (!tag) {
      throw new AppError("Tag不存在", ErrorCodes.RESOURCE_NOT_FOUND);
    }

    await tag.update({
      isEnabled: !tag.isEnabled,
      updatedBy: userId,
    });

    logger.info(
      `[MqttTagService] Toggled tag ${tag.code}: ${
        tag.isEnabled ? "enabled" : "disabled"
      }`
    );

    return tag;
  }

  /**
   * 更新 Tag 顺序
   */
  async updateTagsOrder(tagIds, userId) {
    const updatePromises = tagIds.map((tagId, index) =>
      DataMqttTag.update(
        { order: index, updatedBy: userId },
        { where: { id: tagId } }
      )
    );

    await Promise.all(updatePromises);

    logger.info(`[MqttTagService] Updated order for ${tagIds.length} tags`);

    return { success: true };
  }

  /**
   * 获取 Tag 的当前值（从 Redis）
   */
  async getTagValue(tagId) {
    // 从 Redis 获取值
    const value = await redisTagService.getTagValue(tagId);

    if (!value) {
      return null;
    }

    // 获取 Tag 信息
    const tag = await DataMqttTag.findByPk(tagId, {
      attributes: ["id", "code", "name", "dataType", "unit"],
    });

    return {
      tagId,
      tag,
      ...value,
    };
  }

  /**
   * 获取多个 Tag 的当前值（从 Redis）
   */
  async getTagValues(tagIds) {
    if (!tagIds || tagIds.length === 0) {
      return [];
    }

    // 批量从 Redis 获取值
    const values = await redisTagService.getTagValues(tagIds);

    // 批量获取 Tag 信息
    const tags = await DataMqttTag.findAll({
      where: { id: tagIds },
      attributes: ["id", "code", "name", "dataType", "unit"],
    });

    // 组合结果
    return tagIds
      .map((tagId) => {
        const tag = tags.find((t) => t.id === tagId);
        const value = values[tagId];

        return {
          tagId,
          tag,
          ...(value || { parsedValue: null, quality: null, timestamp: null }),
        };
      })
      .filter((item) => item.tag); // 过滤掉不存在的 Tag
  }
}

module.exports = new MqttTagService();
