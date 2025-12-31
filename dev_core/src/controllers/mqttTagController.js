/**
 * MQTT Tag 控制器
 */

const { logger } = require("../utils/logger");
const mqttTagService = require("../services/mqttTagService");

class MqttTagController {
  /**
   * 创建Tag
   * POST /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tags
   */
  async createTag(req, res, next) {
    try {
      const { projectId, subscriptionId } = req.params;
      const tagData = req.body;
      const userId = req.user.id;

      const tag = await mqttTagService.createTag(
        projectId,
        subscriptionId,
        tagData,
        userId
      );

      logger.info(
        `[MqttTagController] Tag created: ${tag.id} by user ${userId}`
      );

      res.status(201).json({
        success: true,
        data: tag,
      });
    } catch (error) {
      next(error);
    }
  }

  /**
   * 批量创建Tag
   * POST /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tags/batch
   */
  async createTagsBatch(req, res, next) {
    try {
      const { projectId, subscriptionId } = req.params;
      const { tags: tagsData } = req.body;
      const userId = req.user.id;

      const tags = await mqttTagService.createTagsBatch(
        projectId,
        subscriptionId,
        tagsData,
        userId
      );

      logger.info(
        `[MqttTagController] ${tags.length} tags created by user ${userId}`
      );

      res.status(201).json({
        success: true,
        data: tags,
      });
    } catch (error) {
      next(error);
    }
  }

  /**
   * 更新Tag
   * PUT /api/v1/data/mqtt/tags/:id
   */
  async updateTag(req, res, next) {
    try {
      const { id } = req.params;
      const tagData = req.body;
      const userId = req.user.id;

      const tag = await mqttTagService.updateTag(id, tagData, userId);

      logger.info(`[MqttTagController] Tag updated: ${id} by user ${userId}`);

      res.json({
        success: true,
        data: tag,
      });
    } catch (error) {
      next(error);
    }
  }

  /**
   * 删除Tag
   * DELETE /api/v1/data/mqtt/tags/:id
   */
  async deleteTag(req, res, next) {
    try {
      const { id } = req.params;

      await mqttTagService.deleteTag(id);

      logger.info(`[MqttTagController] Tag deleted: ${id}`);

      res.json({
        success: true,
        message: "Tag deleted successfully",
      });
    } catch (error) {
      next(error);
    }
  }

  /**
   * 获取Tag详情
   * GET /api/v1/data/mqtt/tags/:id
   */
  async getTag(req, res, next) {
    try {
      const { id } = req.params;

      const tag = await mqttTagService.getTag(id);

      res.json({
        success: true,
        data: tag,
      });
    } catch (error) {
      next(error);
    }
  }

  /**
   * 获取订阅的所有Tag
   * GET /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tags
   */
  async getTagsBySubscription(req, res, next) {
    try {
      const { subscriptionId } = req.params;
      const { page, pageSize, isEnabled } = req.query;

      const result = await mqttTagService.getTagsBySubscription(
        subscriptionId,
        {
          page: page ? parseInt(page) : 1,
          pageSize: pageSize ? parseInt(pageSize) : 50,
          isEnabled: isEnabled !== undefined ? isEnabled === "true" : undefined,
        }
      );

      res.json({
        success: true,
        data: result.tags,
        pagination: result.pagination,
      });
    } catch (error) {
      next(error);
    }
  }

  /**
   * 获取项目的所有Tag
   * GET /api/v1/data/projects/:projectId/mqtt/tags
   */
  async getTagsByProject(req, res, next) {
    try {
      const { projectId } = req.params;
      const { page, pageSize, subscriptionId, isEnabled } = req.query;

      const result = await mqttTagService.getTagsByProject(projectId, {
        page: page ? parseInt(page) : 1,
        pageSize: pageSize ? parseInt(pageSize) : 50,
        subscriptionId,
        isEnabled: isEnabled !== undefined ? isEnabled === "true" : undefined,
      });

      res.json({
        success: true,
        data: result.tags,
        pagination: result.pagination,
      });
    } catch (error) {
      next(error);
    }
  }

  /**
   * 切换Tag启用状态
   * PATCH /api/v1/data/mqtt/tags/:id/toggle
   */
  async toggleTag(req, res, next) {
    try {
      const { id } = req.params;
      const userId = req.user.id;

      const tag = await mqttTagService.toggleTag(id, userId);

      logger.info(
        `[MqttTagController] Tag toggled: ${id} - ${
          tag.isEnabled ? "enabled" : "disabled"
        }`
      );

      res.json({
        success: true,
        data: tag,
      });
    } catch (error) {
      next(error);
    }
  }

  /**
   * 更新Tag顺序
   * PUT /api/v1/data/mqtt/tags/order
   */
  async updateTagsOrder(req, res, next) {
    try {
      const { tagIds } = req.body;
      const userId = req.user.id;

      await mqttTagService.updateTagsOrder(tagIds, userId);

      logger.info(`[MqttTagController] Tags order updated by user ${userId}`);

      res.json({
        success: true,
        message: "Tags order updated successfully",
      });
    } catch (error) {
      next(error);
    }
  }

  /**
   * 获取Tag的当前值
   * GET /api/v1/data/mqtt/tags/:id/value
   */
  async getTagValue(req, res, next) {
    try {
      const { id } = req.params;

      const tagValue = await mqttTagService.getTagValue(id);

      res.json({
        success: true,
        data: tagValue,
      });
    } catch (error) {
      next(error);
    }
  }

  /**
   * 获取多个Tag的当前值
   * POST /api/v1/data/mqtt/tags/values
   */
  async getTagValues(req, res, next) {
    try {
      const { tagIds } = req.body;

      const tagValues = await mqttTagService.getTagValues(tagIds);

      res.json({
        success: true,
        data: tagValues,
      });
    } catch (error) {
      next(error);
    }
  }
}

module.exports = new MqttTagController();
