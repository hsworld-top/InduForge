const { Op } = require("sequelize");
const dayjs = require("dayjs");
const { TIME_FORMAT } = require("../constants/time");
const AppError = require("../utils/AppError");
const { ErrorCodes } = require("../constants/errorCodes");
const { normalizeSegment, normalizePath } = require("../utils/datapointPath");
const {
  DataPoint,
  DataQuery,
  DataConnection,
  DataMqttTag,
  DataMqttTagGroup,
  DataMqttSubscription,
} = require("../models");
const redisTagService = require("./redisTagService");
const socketService = require("./socketService");

/**
 * 数据点服务
 * 负责数据点的同步、查询与实时推送
 */
class DataPointService {
  /**
   * 获取数据点列表
   * @param {string} projectId - 工程ID
   * @param {object} options - 查询参数
   * @returns {Promise<object>} 数据点列表与分页信息
   */
  async getDataPoints(projectId, options = {}) {
    const {
      type,
      status,
      search,
      sourceId,
      sourceIds,
      page = 1,
      pageSize = 50,
    } = options;
    const where = { projectId };

    if (type) {
      where.sourceType = type;
    }

    if (status) {
      where.status = status;
    }

    if (search) {
      where[Op.or] = [
        { name: { [Op.like]: `%${search}%` } },
        { path: { [Op.like]: `%${search}%` } },
      ];
    }

    if (sourceId) {
      where.sourceId = sourceId;
    }

    if (sourceIds) {
      const ids = String(sourceIds)
        .split(",")
        .map((item) => item.trim())
        .filter(Boolean);
      if (ids.length > 0) {
        where.sourceId = { [Op.in]: ids };
      }
    }

    const limit = Math.max(1, Math.min(200, parseInt(pageSize, 10) || 50));
    const currentPage = Math.max(1, parseInt(page, 10) || 1);
    const offset = (currentPage - 1) * limit;

    const { rows, count } = await DataPoint.findAndCountAll({
      where,
      order: [["created_at", "DESC"]],
      limit,
      offset,
    });

    return {
      datapoints: rows,
      pagination: {
        page: currentPage,
        pageSize: limit,
        total: count,
        totalPages: Math.ceil(count / limit),
      },
    };
  }

  /**
   * 获取数据点详情
   * @param {string} projectId - 工程ID
   * @param {string} id - 数据点ID
   * @returns {Promise<object>} 数据点信息
   */
  async getDataPoint(projectId, id) {
    const datapoint = await DataPoint.findOne({
      where: { id, projectId },
    });

    if (!datapoint) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: "数据点不存在",
      });
    }

    return datapoint;
  }

  /**
   * 更新数据点扩展属性
   * @param {string} projectId - 工程ID
   * @param {string} id - 数据点ID
   * @param {object} data - 更新数据
   * @param {string} userId - 用户ID
   * @returns {Promise<object>} 更新后的数据点
   */
  async updateDataPoint(projectId, id, data, userId) {
    const datapoint = await this.getDataPoint(projectId, id);

    await datapoint.update({
      name: data.name ?? datapoint.name,
      description: data.description ?? datapoint.description,
      unit: data.unit ?? datapoint.unit,
      precisionNum: data.precisionNum ?? datapoint.precisionNum,
      minValue: data.minValue ?? datapoint.minValue,
      maxValue: data.maxValue ?? datapoint.maxValue,
      alarmLow: data.alarmLow ?? datapoint.alarmLow,
      alarmHigh: data.alarmHigh ?? datapoint.alarmHigh,
      tags: data.tags ?? datapoint.tags,
      refreshMode: data.refreshMode ?? datapoint.refreshMode,
      refreshInterval: data.refreshInterval ?? datapoint.refreshInterval,
      updatedBy: userId,
    });

    return datapoint;
  }

  /**
   * 删除失效数据点
   * @param {string} projectId - 工程ID
   * @param {string} id - 数据点ID
   * @returns {Promise<void>}
   */
  async deleteInvalidDataPoint(projectId, id) {
    const datapoint = await this.getDataPoint(projectId, id);
    if (datapoint.status !== "invalid") {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "仅允许删除失效数据点",
      });
    }

    await datapoint.destroy();
  }

  /**
   * 批量删除失效数据点
   * @param {string} projectId - 工程ID
   * @param {string[]} ids - 数据点ID列表
   * @returns {Promise<number>} 删除数量
   */
  async deleteInvalidDataPoints(projectId, ids) {
    if (!Array.isArray(ids) || ids.length === 0) {
      throw new AppError(ErrorCodes.VALIDATION_REQUIRED, 400, {
        message: "缺少数据点ID列表",
      });
    }

    const uniqueIds = Array.from(new Set(ids));
    const datapoints = await DataPoint.findAll({
      where: {
        projectId,
        id: {
          [Op.in]: uniqueIds,
        },
      },
    });

    if (datapoints.length !== uniqueIds.length) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: "部分数据点不存在",
      });
    }

    const invalidFound = datapoints.some(
      (datapoint) => datapoint.status !== "invalid"
    );
    if (invalidFound) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "仅允许删除失效数据点",
      });
    }

    const deletedCount = await DataPoint.destroy({
      where: {
        projectId,
        id: {
          [Op.in]: uniqueIds,
        },
        status: "invalid",
      },
    });

    return deletedCount;
  }

  /**
   * 根据路径获取数据点
   * @param {string} projectId - 工程ID
   * @param {string} path - 数据点路径
   * @returns {Promise<object>} 数据点信息
   */
  async getDataPointByPath(projectId, path) {
    const normalizedPath = normalizePath(path);
    const datapoint = await DataPoint.findOne({
      where: { projectId, path: normalizedPath },
    });

    if (!datapoint) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: "数据点不存在",
      });
    }

    return datapoint;
  }

  /**
   * 根据来源获取数据点
   * @param {string} projectId - 工程ID
   * @param {string} sourceType - 来源类型
   * @param {string} sourceId - 来源ID
   * @returns {Promise<object|null>} 数据点
   */
  async getDataPointBySourceId(projectId, sourceType, sourceId) {
    if (!projectId || !sourceType || !sourceId) {
      return null;
    }
    return DataPoint.findOne({
      where: {
        projectId,
        sourceType,
        sourceId,
      },
    });
  }

  /**
   * 获取数据点值
   * @param {string} projectId - 工程ID
   * @param {string} path - 数据点路径
   * @returns {Promise<object>} 数据点值信息
   */
  async getDataPointValue(projectId, path) {
    const datapoint = await this.getDataPointByPath(projectId, path);
    const now = dayjs().format(TIME_FORMAT);

    if (datapoint.sourceType === "mqtt.tag" && datapoint.sourceId) {
      const value = await redisTagService.getTagValue(datapoint.sourceId);
      return {
        path: datapoint.path,
        value: value ? value.parsedValue : null,
        quality: value ? value.quality : "unknown",
        timestamp: value ? value.timestamp : now,
        status: datapoint.status,
      };
    }

    if (datapoint.sourceType === "db.query" && datapoint.sourceId) {
      const dataQueryService = require("./dataQueryService");
      const result = await dataQueryService.executeQuery(
        datapoint.sourceId,
        {},
        null
      );
      return {
        path: datapoint.path,
        value: result?.data || null,
        quality: "good",
        timestamp: now,
        status: datapoint.status,
      };
    }

    if (datapoint.sourceType === "mqtt.subscription" && datapoint.sourceId) {
      const mqttService = require("./mqttService");
      const latest = mqttService.getLatestSubscriptionMessage(
        datapoint.sourceId
      );
      return {
        path: datapoint.path,
        value: latest || null,
        quality: latest ? "good" : "unknown",
        timestamp: latest?.timestamp || now,
        status: datapoint.status,
      };
    }

    return {
      path: datapoint.path,
      value: null,
      quality: "unknown",
      timestamp: now,
      status: datapoint.status,
    };
  }

  /**
   * 同步查询数据点
   * @param {string} projectId - 工程ID
   * @param {string} queryId - 查询ID
   * @param {string} userId - 用户ID
   * @returns {Promise<void>}
   */
  async syncFromQuery(projectId, queryId, userId) {
    const dataQueryService = require("./dataQueryService");
    const query = await DataQuery.findByPk(queryId, {
      include: [
        {
          model: DataConnection,
          as: "connection",
          attributes: ["id", "name", "type"],
        },
      ],
    });

    if (!query) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: "查询不存在",
      });
    }

    const connectionName = normalizeSegment(query.connection?.name || "db");
    const queryName = normalizeSegment(query.name);
    const basePath = `db.${connectionName}.${queryName}`;

    const existingPoints = await DataPoint.findAll({
      where: {
        projectId,
        sourceType: "db.query",
        sourceId: queryId,
      },
    });

    const resultPoint = existingPoints.find(
      (point) => point.sourceConfig?.mode === "result"
    );

    if (resultPoint) {
      await this.ensurePathUnique(projectId, basePath, resultPoint.id);
      await resultPoint.update({
        path: basePath,
        name: query.name,
        dataType: "object",
        sourceConfig: { mode: "result" },
        status: "active",
        updatedBy: userId,
      });
    } else {
      await this.ensurePathUnique(projectId, basePath);
      await DataPoint.create({
        projectId,
        path: basePath,
        name: query.name,
        sourceType: "db.query",
        sourceId: queryId,
        sourceConfig: { mode: "result" },
        dataType: "object",
        status: "active",
        createdBy: userId,
      });
    }

    await Promise.all(
      existingPoints
        .filter((point) => point.sourceConfig?.mode !== "result")
        .map((point) =>
          point.update({ status: "invalid", updatedBy: userId })
        )
    );
  }

  /**
   * 同步 MQTT 变量数据点
   * @param {string} projectId - 工程ID
   * @param {string} tagId - Tag ID
   * @param {string} userId - 用户ID
   * @returns {Promise<void>}
   */
  async syncFromMqttTag(projectId, tagId, userId) {
    const tag = await DataMqttTag.findByPk(tagId);
    if (!tag || tag.projectId !== projectId) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: "变量不存在",
      });
    }

    const subscription = await DataMqttSubscription.findByPk(
      tag.subscriptionId
    );
    if (!subscription) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: "订阅不存在",
      });
    }

    const connection = await DataConnection.findByPk(subscription.connectionId);
    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: "连接不存在",
      });
    }

    const group = tag.groupId
      ? await DataMqttTagGroup.findByPk(tag.groupId)
      : null;

    const connectionName = normalizeSegment(connection.name);
    const groupName = normalizeSegment(group ? group.name : "默认分组");
    const tagName = normalizeSegment(tag.name);
    const path = `mqtt.${connectionName}.${groupName}.${tagName}`;

    const existing = await DataPoint.findOne({
      where: {
        projectId,
        sourceType: "mqtt.tag",
        sourceId: tag.id,
      },
    });

    if (existing) {
      await this.ensurePathUnique(projectId, path, existing.id);
      await existing.update({
        path,
        name: tag.name,
        dataType: tag.dataType,
        unit: tag.unit,
        status: "active",
        updatedBy: userId,
      });
    } else {
      await this.ensurePathUnique(projectId, path);
      await DataPoint.create({
        projectId,
        path,
        name: tag.name,
        sourceType: "mqtt.tag",
        sourceId: tag.id,
        dataType: tag.dataType,
        unit: tag.unit,
        status: "active",
        createdBy: userId,
      });
    }
  }

  /**
   * 同步 MQTT 订阅数据点
   * @param {string} projectId - 工程ID
   * @param {string} subscriptionId - 订阅ID
   * @param {string} userId - 用户ID
   * @returns {Promise<void>}
   */
  async syncFromMqttSubscription(projectId, subscriptionId, userId) {
    const subscription = await DataMqttSubscription.findByPk(subscriptionId);
    if (!subscription || subscription.projectId !== projectId) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: "订阅不存在",
      });
    }

    const connection = await DataConnection.findByPk(subscription.connectionId);
    if (!connection) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: "连接不存在",
      });
    }

    const connectionName = normalizeSegment(connection.name);
    const subscriptionName = normalizeSegment(subscription.name);
    const path = `mqtt.${connectionName}.${subscriptionName}`;

    const existing = await DataPoint.findOne({
      where: {
        projectId,
        sourceType: "mqtt.subscription",
        sourceId: subscription.id,
      },
    });

    if (existing) {
      await this.ensurePathUnique(projectId, path, existing.id);
      await existing.update({
        path,
        name: subscription.name,
        dataType: "object",
        sourceConfig: { mode: "subscription" },
        status: "active",
        updatedBy: userId,
      });
    } else {
      await this.ensurePathUnique(projectId, path);
      await DataPoint.create({
        projectId,
        path,
        name: subscription.name,
        sourceType: "mqtt.subscription",
        sourceId: subscription.id,
        sourceConfig: { mode: "subscription" },
        dataType: "object",
        status: "active",
        createdBy: userId,
      });
    }

    const mqttService = require("./mqttService");
    mqttService.setSubscriptionDatapointCache(subscription.id, path);
  }

  /**
   * 标记数据点失效
   * @param {string} projectId - 工程ID
   * @param {string} sourceType - 来源类型
   * @param {string} sourceId - 来源ID
   * @param {string} userId - 用户ID
   * @returns {Promise<void>}
   */
  async markInvalidBySource(projectId, sourceType, sourceId, userId) {
    await DataPoint.update(
      { status: "invalid", updatedBy: userId },
      {
        where: {
          projectId,
          sourceType,
          sourceId,
        },
      }
    );
  }

  /**
   * 批量获取指定来源的数据点
   * @param {string} projectId - 工程ID
   * @param {string} sourceType - 来源类型
   * @param {string[]} sourceIds - 来源ID列表
   * @returns {Promise<object[]>} 数据点列表
   */
  async getDataPointsBySourceIds(projectId, sourceType, sourceIds) {
    if (!sourceIds || sourceIds.length === 0) {
      return [];
    }

    return DataPoint.findAll({
      where: {
        projectId,
        sourceType,
        sourceId: {
          [Op.in]: sourceIds,
        },
      },
    });
  }

  /**
   * 触发数据点值推送
   * @param {string} projectId - 工程ID
   * @param {string} path - 数据点路径
   * @param {object} valueData - 值数据
   */
  emitDataPointValue(projectId, path, valueData) {
    if (!socketService.getIO()) {
      return;
    }

    const normalizedPath = normalizePath(path);
    const room = `datapoint:${projectId}:${normalizedPath}`;
    socketService.getIO().to(room).emit("datapoint:value", {
      path: normalizedPath,
      ...valueData,
    });
  }

  /**
   * 校验路径唯一性
   * @param {string} projectId - 工程ID
   * @param {string} path - 数据点路径
   * @param {string} ignoreId - 忽略的数据点ID
   * @returns {Promise<void>}
   */
  async ensurePathUnique(projectId, path, ignoreId = null) {
    const existing = await DataPoint.findOne({
      where: { projectId, path },
    });

    if (existing && existing.id !== ignoreId) {
      throw new AppError(ErrorCodes.RESOURCE_ALREADY_EXISTS, 400, {
        message: `数据点路径已存在: ${path}`,
      });
    }
  }

}

module.exports = new DataPointService();
