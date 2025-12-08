const { Project, DataQuery, DataConnection } = require('../models');
const { getResponseData } = require('../utils/mockResponse');
const AppError = require('../utils/AppError');
const ErrorCodes = require('../constants/errorCodes');

class DatasourceService {
  /**
   * 校验 app(aid) 是否为当前租户的工程，并返回 Project
   */
  async ensureProjectOfTenant(appId, tenantId) {
    if (!appId) {
      throw new AppError(ErrorCodes.INVALID_PARAMS, 400, { message: 'app(aid) 必填' });
    }

    const project = await Project.findByPk(appId);
    if (!project) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { resource: 'Project', id: appId });
    }
    if (tenantId && project.tenantId !== tenantId) {
      throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, { message: '工程不属于当前租户' });
    }
    return project;
  }

  /**
   * 获取数据源列表（桥接 DataQuery 列表）
   * @param {string} appId - TinyEngine 中的 appId，这里直接使用为 projectId
   */
  async list(appId, tenantId, query = {}) {
    await this.ensureProjectOfTenant(appId, tenantId);

    const { page = 1, limit = 100, connectionId, queryType, isActive } = query;

    const where = { projectId: appId };
    if (connectionId) where.connectionId = connectionId;
    if (queryType) where.queryType = queryType;
    if (isActive !== undefined) where.isActive = isActive === 'true' || isActive === true;

    const offset = (page - 1) * limit;

    const { count, rows } = await DataQuery.findAndCountAll({
      where,
      include: [
        {
          model: DataConnection,
          as: 'connection',
          attributes: ['id', 'name', 'type'],
        }
      ],
      limit: parseInt(limit, 10),
      offset,
      order: [['createdAt', 'DESC']],
    });

    const data = {
      list: rows,
      total: count,
      currentPage: parseInt(page, 10),
      pageSize: parseInt(limit, 10),
    };

    return getResponseData(data);
  }

  /**
   * 数据源详情（桥接单个 DataQuery）
   */
  async detail(id, tenantId) {
    const query = await DataQuery.findByPk(id, {
      include: [
        {
          model: DataConnection,
          as: 'connection',
          attributes: ['id', 'name', 'type', 'projectId'],
        }
      ],
    });

    if (!query) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { resource: 'Datasource', id });
    }

    // 权限：验证关联工程是否属于当前租户
    if (tenantId && query.connection && query.connection.projectId) {
      const project = await Project.findByPk(query.connection.projectId);
      if (!project || project.tenantId !== tenantId) {
        throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, { message: '无权访问该数据源' });
      }
    }

    return getResponseData(query);
  }

  /**
   * 创建数据源（桥接 createQuery）
   * body 期望字段：
   * - app: 工程ID（即 projectId）
   * - name, description, category, connectionId, queryType
   * - config: { sql, parameters }
   */
  async create(body, userId, tenantId) {
    const {
      app,
      name,
      description,
      category,
      connectionId,
      queryType,
      config = {},
    } = body;

    await this.ensureProjectOfTenant(app, tenantId);

    // 简单校验
    if (!name || !connectionId || !queryType) {
      throw new AppError(ErrorCodes.INVALID_PARAMS, 400, { message: 'name/connectionId/queryType 为必填' });
    }

    // 校验连接是否属于该工程
    const connection = await DataConnection.findOne({
      where: { id: connectionId, projectId: app },
    });
    if (!connection) {
      throw new AppError(ErrorCodes.INVALID_PARAMS, 400, { message: '无效的 connectionId' });
    }

    const query = await DataQuery.create({
      projectId: app,
      connectionId,
      name,
      description,
      category,
      queryType,
      config,
      createdBy: userId,
      isEnabled: true,
    });

    const fullQuery = await DataQuery.findByPk(query.id, {
      include: [
        {
          model: DataConnection,
          as: 'connection',
          attributes: ['id', 'name', 'type'],
        }
      ],
    });

    return getResponseData(fullQuery);
  }

  /**
   * 更新数据源
   */
  async update(id, body, userId, tenantId) {
    const query = await DataQuery.findByPk(id);
    if (!query) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { resource: 'Datasource', id });
    }

    // 权限：检查工程归属
    await this.ensureProjectOfTenant(query.projectId, tenantId);

    const {
      name,
      description,
      category,
      connectionId,
      queryType,
      isEnabled,
      config,
    } = body;

    if (connectionId && connectionId !== query.connectionId) {
      const connection = await DataConnection.findOne({
        where: { id: connectionId, projectId: query.projectId },
      });
      if (!connection) {
        throw new AppError(ErrorCodes.INVALID_PARAMS, 400, { message: '无效的 connectionId' });
      }
      query.connectionId = connectionId;
    }

    if (name !== undefined) query.name = name;
    if (description !== undefined) query.description = description;
    if (category !== undefined) query.category = category;
    if (queryType !== undefined) query.queryType = queryType;
    if (isEnabled !== undefined) query.isEnabled = !!isEnabled;
    if (config !== undefined) query.config = config;
    if (userId) query.updatedBy = userId;

    await query.save();

    const fullQuery = await DataQuery.findByPk(query.id, {
      include: [
        {
          model: DataConnection,
          as: 'connection',
          attributes: ['id', 'name', 'type'],
        }
      ],
    });

    return getResponseData(fullQuery);
  }

  /**
   * 删除数据源
   */
  async delete(id, tenantId) {
    const query = await DataQuery.findByPk(id);
    if (!query) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { resource: 'Datasource', id });
    }

    await this.ensureProjectOfTenant(query.projectId, tenantId);

    const data = query.toJSON();
    await query.destroy();
    return getResponseData(data);
  }
}

module.exports = new DatasourceService();


