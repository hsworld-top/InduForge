const { logger } = require("../utils/logger");
const { DataMqttTagGroup } = require("../models");
const ApiResponse = require("../utils/response");

/**
 * MQTT变量组控制器
 */
class MqttTagGroupController {
  /**
   * 获取订阅的所有变量组
   */
  async getGroups(req, res) {
    try {
      const { projectId, subscriptionId } = req.params;

      const groups = await DataMqttTagGroup.findAll({
        where: {
          projectId,
          subscriptionId,
        },
        order: [
          ["order", "ASC"],
          ["createdAt", "ASC"],
        ],
      });

      logger.info(
        `[MqttTagGroupController] Retrieved ${groups.length} tag groups for subscription ${subscriptionId}`
      );

      return ApiResponse.success(res, { groups });
    } catch (error) {
      logger.error(`[MqttTagGroupController] Error getting tag groups:`, error);
      return ApiResponse.error(
        res,
        "C0001",
        { message: "获取变量组失败" },
        500
      );
    }
  }

  /**
   * 获取单个变量组
   */
  async getGroup(req, res) {
    try {
      const { groupId } = req.params;

      const group = await DataMqttTagGroup.findByPk(groupId);

      if (!group) {
        return ApiResponse.error(
          res,
          "C1003",
          { message: "变量组不存在" },
          404
        );
      }

      logger.info(`[MqttTagGroupController] Retrieved tag group ${groupId}`);

      return ApiResponse.success(res, { group });
    } catch (error) {
      logger.error(`[MqttTagGroupController] Error getting tag group:`, error);
      return ApiResponse.error(
        res,
        "C0001",
        { message: "获取变量组失败" },
        500
      );
    }
  }

  /**
   * 创建变量组
   */
  async createGroup(req, res) {
    try {
      const { projectId, subscriptionId } = req.params;
      const { name, code, description, color, icon, order } = req.body;
      const userId = req.user?.id;

      // 检查code是否已存在
      const existing = await DataMqttTagGroup.findOne({
        where: {
          subscriptionId,
          code,
        },
      });

      if (existing) {
        return ApiResponse.error(
          res,
          "C1001",
          { message: "变量组标识符已存在" },
          400
        );
      }

      const group = await DataMqttTagGroup.create({
        projectId,
        subscriptionId,
        name,
        code,
        description,
        color,
        icon,
        order: order || 0,
        createdBy: userId,
      });

      logger.info(
        `[MqttTagGroupController] Created tag group ${group.id} for subscription ${subscriptionId}`
      );

      return ApiResponse.success(res, { group }, null, {}, 201);
    } catch (error) {
      logger.error(`[MqttTagGroupController] Error creating tag group:`, error);
      return ApiResponse.error(
        res,
        "C0001",
        { message: "创建变量组失败" },
        500
      );
    }
  }

  /**
   * 更新变量组
   */
  async updateGroup(req, res) {
    try {
      const { groupId } = req.params;
      const { name, code, description, color, icon, order } = req.body;
      const userId = req.user?.id;

      const group = await DataMqttTagGroup.findByPk(groupId);

      if (!group) {
        return ApiResponse.error(
          res,
          "C1003",
          { message: "变量组不存在" },
          404
        );
      }

      // 如果修改了code，检查是否与其他组冲突
      if (code && code !== group.code) {
        const existing = await DataMqttTagGroup.findOne({
          where: {
            subscriptionId: group.subscriptionId,
            code,
            id: { [require("sequelize").Op.ne]: groupId },
          },
        });

        if (existing) {
          return ApiResponse.error(
            res,
            "C1001",
            { message: "变量组标识符已存在" },
            400
          );
        }
      }

      await group.update({
        name: name !== undefined ? name : group.name,
        code: code !== undefined ? code : group.code,
        description:
          description !== undefined ? description : group.description,
        color: color !== undefined ? color : group.color,
        icon: icon !== undefined ? icon : group.icon,
        order: order !== undefined ? order : group.order,
        updatedBy: userId,
      });

      logger.info(`[MqttTagGroupController] Updated tag group ${groupId}`);

      return ApiResponse.success(res, { group });
    } catch (error) {
      logger.error(`[MqttTagGroupController] Error updating tag group:`, error);
      return ApiResponse.error(
        res,
        "C0001",
        { message: "更新变量组失败" },
        500
      );
    }
  }

  /**
   * 删除变量组
   */
  async deleteGroup(req, res) {
    try {
      const { groupId } = req.params;

      const group = await DataMqttTagGroup.findByPk(groupId);

      if (!group) {
        return ApiResponse.error(
          res,
          "C1003",
          { message: "变量组不存在" },
          404
        );
      }

      await group.destroy();

      logger.info(
        `[MqttTagGroupController] Deleted tag group ${groupId}. Associated tags will have groupId set to NULL.`
      );

      return ApiResponse.success(res, null);
    } catch (error) {
      logger.error(`[MqttTagGroupController] Error deleting tag group:`, error);
      return ApiResponse.error(
        res,
        "C0001",
        { message: "删除变量组失败" },
        500
      );
    }
  }

  /**
   * 批量更新变量组顺序
   */
  async updateGroupsOrder(req, res) {
    try {
      const { groups } = req.body;

      if (!Array.isArray(groups) || groups.length === 0) {
        return ApiResponse.error(
          res,
          "C1001",
          { message: "无效的请求参数" },
          400
        );
      }

      // 批量更新顺序
      await Promise.all(
        groups.map((item) =>
          DataMqttTagGroup.update(
            { order: item.order },
            { where: { id: item.id } }
          )
        )
      );

      logger.info(
        `[MqttTagGroupController] Updated order for ${groups.length} tag groups`
      );

      return ApiResponse.success(res, null);
    } catch (error) {
      logger.error(
        `[MqttTagGroupController] Error updating groups order:`,
        error
      );
      return ApiResponse.error(
        res,
        "C0001",
        { message: "更新变量组顺序失败" },
        500
      );
    }
  }
}

module.exports = new MqttTagGroupController();
