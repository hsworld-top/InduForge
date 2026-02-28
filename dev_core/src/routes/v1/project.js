const express = require('express');
const Joi = require('joi');
const { validate } = require('../../middlewares/validate');
const { logger } = require('../../utils/logger');
const { buildTenantWhere } = require('../../utils/scope');
const { Op } = require('sequelize');
const archiver = require('archiver');
const ApiResponse = require('../../utils/response');
const ErrorCodes = require('../../constants/errorCodes');
const appConfig = require('../../config/app');

const designAssetService = require('../../services/designAssetService');

const router = express.Router();

// 导入模型和中间件
const {
  Project,
  Tenant,
  User,
  DesignPage,
  DataConnection,
  DataQuery,
  DataMqttConfig,
  DataMqttSubscription,
  DataMqttTagGroup,
  DataMqttTag,
  DataPoint,
  DataRelationalConfig,
} = require('../../models');
const { authenticateToken, requireResourceOwnership } = require('../../middlewares/auth');
const { randomUUID } = require('crypto');

const parseJsonField = (value, fallback) => {
  if (value == null) return fallback;
  if (typeof value === 'object') return value;
  try {
    return JSON.parse(value);
  } catch (error) {
    return fallback;
  }
};

const sanitizeFileName = (value) => {
  const base = String(value || '').trim() || 'file';
  return base.replace(/[\\/:*?"<>|]+/g, '_').slice(0, 120);
};

const toAsciiFileName = (value) => {
  const base = sanitizeFileName(value);
  return base.replace(/[^\x20-\x7E]/g, '_') || 'file';
};

const buildContentDisposition = (fileName) => {
  const safeName = sanitizeFileName(fileName);
  const asciiName = toAsciiFileName(fileName);
  const encoded = encodeURIComponent(safeName);
  return `attachment; filename="${asciiName}.zip"; filename*=UTF-8''${encoded}.zip`;
};

/**
 * @swagger
 * /api/v1/projects:
 *   get:
 *     summary: 获取工程列表
 *     tags: [工程管理]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *       - in: query
 *         name: limit
 *         schema:
 *           type: integer
 *       - in: query
 *         name: name
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: 获取成功
 */
router.get('/', authenticateToken, validate(Joi.object({
  query: Joi.object({
    page: Joi.number().integer().min(1).default(appConfig.pagination.defaultPage),
    limit: Joi.number().integer().min(1).max(appConfig.pagination.maxLimit).default(appConfig.pagination.defaultLimit),
    name: Joi.string().optional()
  })
})), async (req, res) => {
  try {
    const { page, limit, name } = req.query;

    // 转换分页参数为数字
    const pageNum = parseInt(page, 10);
    const limitNum = parseInt(limit, 10);

    let where = buildTenantWhere({}, req);

    if (name) where.name = { [Op.like]: `%${name}%` };

    const offset = (pageNum - 1) * limitNum;

    const { count, rows } = await Project.findAndCountAll({
      where,
      include: [
        { model: Tenant, as: 'tenant' },
        { model: User, as: 'creator', attributes: ['id', 'username', 'fullName'] },
        { model: User, as: 'updater', attributes: ['id', 'username', 'fullName'] }
      ],
      limit: limitNum,
      offset,
      order: [['createdAt', 'DESC']],
    });

    return ApiResponse.paginated(res, { projects: rows }, {
      total: count,
      page: pageNum,
      limit: limitNum,
      totalPages: Math.ceil(count / limitNum),
    });
  } catch (error) {
    logger.error('Get projects error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/projects/{id}/export:
 *   get:
 *     summary: 导出工程
 *     tags: [工程管理]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: 导出成功
 */
router.get('/:id/export', authenticateToken, requireResourceOwnership('project'), validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() })
})), async (req, res) => {
  try {
    const { id } = req.params;
    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 404);
    }

    const pages = await DesignPage.findAll({
      where: { projectId: id },
      order: [['sortOrder', 'ASC']],
    });

    const [settingsRows] = await Project.sequelize.query(
      'SELECT globalVariables, globalScripts FROM design_project_settings WHERE projectId = ? LIMIT 1',
      { replacements: [id] }
    );
    const settingsRow = Array.isArray(settingsRows) ? settingsRows[0] : settingsRows;

    const globalVariables = parseJsonField(settingsRow?.globalVariables, null);
    const globalScripts = parseJsonField(settingsRow?.globalScripts, null);

    const connections = await DataConnection.findAll({ where: { projectId: id } });
    const queries = await DataQuery.findAll({ where: { projectId: id } });
    const connectionIds = connections.map((item) => item.id);
    const relationalConfigs = connectionIds.length
      ? await DataRelationalConfig.findAll({ where: { connectionId: connectionIds } })
      : [];
    const mqttConfigs = connectionIds.length
      ? await DataMqttConfig.findAll({ where: { connectionId: connectionIds } })
      : [];
    const mqttSubscriptions = await DataMqttSubscription.findAll({ where: { projectId: id } });
    const mqttTagGroups = await DataMqttTagGroup.findAll({ where: { projectId: id } });
    const mqttTags = await DataMqttTag.findAll({ where: { projectId: id } });
    const datapoints = await DataPoint.findAll({ where: { projectId: id } });

    const pageIndex = pages.map((page) => {
      const safeName = sanitizeFileName(page.name || page.id);
      const file = page.type === 'page' || page.type === 'dialog'
        ? `${safeName}-${page.id}.json`
        : null;
      return {
        id: page.id,
        name: page.name,
        type: page.type,
        parentId: page.parentId,
        sortOrder: page.sortOrder,
        file,
      };
    });

    const projectJson = {
      project: {
        id: project.id,
        name: project.name,
        description: project.description || '',
        colorTag: project.colorTag || '#3b82f6',
      },
      entryConfig: project.entryConfig || {},
      exportedAt: new Date().toISOString(),
    };

    res.setHeader('Content-Type', 'application/zip');
    const rawName = project.name || 'project';
    res.setHeader('Content-Disposition', buildContentDisposition(rawName));

    const archive = archiver('zip', { zlib: { level: 9 } });
    archive.on('error', (error) => {
      throw error;
    });
    archive.pipe(res);

    archive.append(JSON.stringify(projectJson, null, 2), { name: 'project.json' });
    archive.append(JSON.stringify(globalVariables || {}, null, 2), {
      name: 'designer/global-variables.json',
    });
    archive.append(JSON.stringify(globalScripts || {}, null, 2), {
      name: 'designer/global-scripts.json',
    });
    archive.append(JSON.stringify(project.projectVariables || {}, null, 2), {
      name: 'designer/project-variables.json',
    });
    archive.append(JSON.stringify(pageIndex, null, 2), {
      name: 'designer/pages/index.json',
    });

    pages.forEach((page) => {
      if (page.type !== 'page' && page.type !== 'dialog') return;
      const safeName = sanitizeFileName(page.name || page.id);
      const fileName = `${safeName}-${page.id}.json`;
      const fallbackSchema = {
        page: {
          id: page.id,
          name: page.name,
          type: page.type,
          parentId: page.parentId,
          sortOrder: page.sortOrder,
        },
        nodesById: {},
        graphicsById: {},
      };
      const payload = page.schemaContent || fallbackSchema;
      archive.append(JSON.stringify(payload, null, 2), {
        name: `designer/pages/${fileName}`,
      });
    });

    archive.append(
      JSON.stringify(connections.map((item) => item.toJSON()), null, 2),
      { name: 'datacenter/connections.json' }
    );
    archive.append(
      JSON.stringify(relationalConfigs.map((item) => item.toJSON()), null, 2),
      { name: 'datacenter/relational-configs.json' }
    );
    archive.append(
      JSON.stringify(queries.map((item) => item.toJSON()), null, 2),
      { name: 'datacenter/queries.json' }
    );
    archive.append(
      JSON.stringify(mqttConfigs.map((item) => item.toJSON()), null, 2),
      { name: 'datacenter/mqtt-configs.json' }
    );
    archive.append(
      JSON.stringify(mqttSubscriptions.map((item) => item.toJSON()), null, 2),
      { name: 'datacenter/mqtt-subscriptions.json' }
    );
    archive.append(
      JSON.stringify(mqttTagGroups.map((item) => item.toJSON()), null, 2),
      { name: 'datacenter/mqtt-tag-groups.json' }
    );
    archive.append(
      JSON.stringify(mqttTags.map((item) => item.toJSON()), null, 2),
      { name: 'datacenter/mqtt-tags.json' }
    );
    archive.append(
      JSON.stringify(datapoints.map((item) => item.toJSON()), null, 2),
      { name: 'datacenter/datapoints.json' }
    );

    await archive.finalize();
    return;
  } catch (error) {
    logger.error('Export project error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/projects/import:
 *   post:
 *     summary: 导入工程
 *     tags: [工程管理]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               name:
 *                 type: string
 *               payload:
 *                 type: object
 *     responses:
 *       201:
 *         description: 导入成功
 */
router.post('/import', authenticateToken, validate(Joi.object({
  body: Joi.object({
    name: Joi.string().allow('').optional(),
    payload: Joi.object().required(),
  }).required()
})), async (req, res) => {
  try {
    const { payload, name } = req.body;
    const { tenantId, role, id: userId } = req.user;

    if (!['SYSTEM_ADMIN', 'PROJECT_ADMIN'].includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }

    const projectInfo = payload.project || {};
    const entryConfig = payload.entryConfig || {};
    const settings = payload.settings || {};
    const pageList = Array.isArray(payload.pages) ? payload.pages : [];

    const defaultName = projectInfo.name ? `${projectInfo.name}-导入` : '导入工程';
    const finalName = (name || '').trim() || defaultName;

    const projectVariables =
      settings.projectVariables ||
      settings.globalVariables?.definitions ||
      projectInfo.projectVariables ||
      {};

    const project = await Project.create({
      name: finalName,
      description: projectInfo.description || '',
      colorTag: projectInfo.colorTag || '#3b82f6',
      tenantId,
      createdBy: userId,
      projectVariables,
      entryConfig: {},
    });

    const globalVariables = settings.globalVariables || {
      definitions: projectVariables,
      groups: [],
    };
    const globalScripts = settings.globalScripts || null;

    await Project.sequelize.query(
      `INSERT INTO design_project_settings
        (projectId, schemaVersion, globalVariables, globalScripts, updatedBy, updatedAt)
      VALUES (?, ?, ?, ?, ?, ?)
      ON DUPLICATE KEY UPDATE
        globalVariables = VALUES(globalVariables),
        globalScripts = VALUES(globalScripts),
        updatedBy = VALUES(updatedBy),
        updatedAt = VALUES(updatedAt)`,
      {
        replacements: [
          project.id,
          '1.0.0',
          JSON.stringify(globalVariables),
          JSON.stringify(globalScripts || {}),
          userId,
          new Date(),
        ],
      },
    );

    const pageIdMap = new Map();
    pageList.forEach((item) => {
      const page = item.page || item;
      if (page?.id && !pageIdMap.has(page.id)) {
        pageIdMap.set(page.id, randomUUID());
      }
    });

    for (const item of pageList) {
      const pageData = item.page || item;
      if (!pageData) continue;
      const newPageId = pageIdMap.get(pageData.id) || randomUUID();
      const newParentId = pageData.parentId ? pageIdMap.get(pageData.parentId) : null;
      const schemaContent = item.schemaContent || pageData.schemaContent || null;
      const nextSchema = schemaContent ? JSON.parse(JSON.stringify(schemaContent)) : null;
      if (nextSchema?.page) {
        nextSchema.page.id = newPageId;
        nextSchema.page.parentId = newParentId;
        nextSchema.page.name = pageData.name || nextSchema.page.name;
      }

      await DesignPage.create({
        id: newPageId,
        projectId: project.id,
        parentId: newParentId,
        name: pageData.name || '未命名页面',
        type: pageData.type || 'page',
        sortOrder: pageData.sortOrder || 0,
        schemaContent: nextSchema,
        createdBy: userId,
        updatedBy: userId,
      });
    }

    const nextEntry = {
      ...entryConfig,
    };
    if (entryConfig?.homePageId && pageIdMap.has(entryConfig.homePageId)) {
      nextEntry.homePageId = pageIdMap.get(entryConfig.homePageId);
    }
    if (entryConfig?.loginPageId && pageIdMap.has(entryConfig.loginPageId)) {
      nextEntry.loginPageId = pageIdMap.get(entryConfig.loginPageId);
    }
    if (entryConfig?.logoutPageId && pageIdMap.has(entryConfig.logoutPageId)) {
      nextEntry.logoutPageId = pageIdMap.get(entryConfig.logoutPageId);
    }

    await project.update({ entryConfig: nextEntry });

    const datacenter = payload.datacenter || {};
    const normalizeList = (value) => (Array.isArray(value) ? value : []);
    const stripMeta = (item) => {
      if (!item || typeof item !== 'object') return {};
      const {
        id,
        projectId,
        createdAt,
        updatedAt,
        created_by,
        updated_by,
        createdBy,
        updatedBy,
        ...rest
      } = item;
      return rest;
    };
    /**
     * 重写查询类映射变量的 sourceId
     * @param {Object} definitions - 变量定义
     * @param {Map<string, string>} idMap - 旧查询ID -> 新查询ID
     * @returns {Object} 重写后的变量定义
     */
    const remapVariableDefinitions = (definitions, maps) => {
      if (!definitions || typeof definitions !== 'object') return definitions;
      const next = {};
      Object.entries(definitions).forEach(([name, detail]) => {
        if (!detail || typeof detail !== 'object') {
          next[name] = detail;
          return;
        }
        const nextDetail = { ...detail };
        const pickMappedId = (sourceType, sourceId) => {
          if (!sourceId) return null;
          const type = String(sourceType || '').toLowerCase();
          if (type.includes('query')) return maps.queryIdMap.get(sourceId) || null;
          if (type.includes('tag')) return maps.mqttTagIdMap.get(sourceId) || null;
          if (type.includes('subscription')) {
            return maps.mqttSubscriptionIdMap.get(sourceId) || null;
          }
          return null;
        };
        if (nextDetail.sourceType && nextDetail.sourceId) {
          const mappedId = pickMappedId(nextDetail.sourceType, nextDetail.sourceId);
          if (mappedId) nextDetail.sourceId = mappedId;
        }
        if (nextDetail.source && typeof nextDetail.source === 'object') {
          const sourceType = nextDetail.source.sourceType || nextDetail.sourceType;
          if (nextDetail.source.sourceId) {
            const mappedId = pickMappedId(sourceType, nextDetail.source.sourceId);
            if (mappedId) {
              nextDetail.source = {
                ...nextDetail.source,
                sourceId: mappedId,
              };
            }
          }
          if (
            nextDetail.source.datapointId &&
            maps.datapointIdMap.has(nextDetail.source.datapointId)
          ) {
            nextDetail.source = {
              ...nextDetail.source,
              datapointId: maps.datapointIdMap.get(nextDetail.source.datapointId),
            };
          }
        }
        next[name] = nextDetail;
      });
      return next;
    };
    /**
     * 重映射变量中的 sourceId / datapointId
     * @param {Object} payload - 变量配置
     * @param {Object} maps - 新旧ID映射
     * @returns {Object} 重映射后的配置
     */
    const remapVariables = (payload, maps) => {
      if (!payload || typeof payload !== 'object') return payload;
      if (payload.definitions || payload.groups) {
        return {
          ...payload,
          definitions: remapVariableDefinitions(payload.definitions || {}, maps),
        };
      }
      return remapVariableDefinitions(payload, maps);
    };
    const connectionIdMap = new Map();
    const relationalConfigIdMap = new Map();
    const queryIdMap = new Map();
    const mqttConfigIdMap = new Map();
    const mqttSubscriptionIdMap = new Map();
    const mqttTagGroupIdMap = new Map();
    const mqttTagIdMap = new Map();
    const datapointIdMap = new Map();

    const connectionRows = normalizeList(datacenter.connections).map((item) => {
      const newId = randomUUID();
      connectionIdMap.set(item.id, newId);
      return {
        ...stripMeta(item),
        id: newId,
        projectId: project.id,
        createdBy: userId,
        updatedBy: userId,
      };
    });
    if (connectionRows.length) {
      await DataConnection.bulkCreate(connectionRows);
    }

    const relationalConfigRows = normalizeList(datacenter.relationalConfigs)
      .map((item) => {
        const newConnectionId = connectionIdMap.get(item.connectionId);
        if (!newConnectionId) return null;
        const newId = randomUUID();
        relationalConfigIdMap.set(item.id, newId);
        return {
          ...stripMeta(item),
          id: newId,
          connectionId: newConnectionId,
        };
      })
      .filter(Boolean);
    if (relationalConfigRows.length) {
      await DataRelationalConfig.bulkCreate(relationalConfigRows);
    }

    const queryRows = normalizeList(datacenter.queries)
      .map((item) => {
        const newConnectionId = connectionIdMap.get(item.connectionId);
        if (!newConnectionId) return null;
        const newId = randomUUID();
        queryIdMap.set(item.id, newId);
        return {
          ...stripMeta(item),
          id: newId,
          projectId: project.id,
          connectionId: newConnectionId,
          createdBy: userId,
          updatedBy: userId,
        };
      })
      .filter(Boolean);
    if (queryRows.length) {
      await DataQuery.bulkCreate(queryRows);
    }
    const mqttConfigRows = normalizeList(datacenter.mqttConfigs)
      .map((item) => {
        const newConnectionId = connectionIdMap.get(item.connectionId);
        if (!newConnectionId) return null;
        const newId = randomUUID();
        mqttConfigIdMap.set(item.id, newId);
        return {
          ...stripMeta(item),
          id: newId,
          connectionId: newConnectionId,
        };
      })
      .filter(Boolean);
    if (mqttConfigRows.length) {
      await DataMqttConfig.bulkCreate(mqttConfigRows);
    }

    const mqttSubscriptionRows = normalizeList(datacenter.mqttSubscriptions)
      .map((item) => {
        const newConnectionId = connectionIdMap.get(item.connectionId);
        if (!newConnectionId) return null;
        const newId = randomUUID();
        mqttSubscriptionIdMap.set(item.id, newId);
        return {
          ...stripMeta(item),
          id: newId,
          projectId: project.id,
          connectionId: newConnectionId,
          createdBy: userId,
          updatedBy: userId,
        };
      })
      .filter(Boolean);
    if (mqttSubscriptionRows.length) {
      await DataMqttSubscription.bulkCreate(mqttSubscriptionRows);
    }

    const mqttTagGroupRows = normalizeList(datacenter.mqttTagGroups)
      .map((item) => {
        const newSubscriptionId = mqttSubscriptionIdMap.get(item.subscriptionId);
        if (!newSubscriptionId) return null;
        const newId = randomUUID();
        mqttTagGroupIdMap.set(item.id, newId);
        return {
          ...stripMeta(item),
          id: newId,
          projectId: project.id,
          subscriptionId: newSubscriptionId,
          createdBy: userId,
          updatedBy: userId,
        };
      })
      .filter(Boolean);
    if (mqttTagGroupRows.length) {
      await DataMqttTagGroup.bulkCreate(mqttTagGroupRows);
    }

    const mqttTagRows = normalizeList(datacenter.mqttTags)
      .map((item) => {
        const newSubscriptionId = mqttSubscriptionIdMap.get(item.subscriptionId);
        if (!newSubscriptionId) return null;
        const newId = randomUUID();
        mqttTagIdMap.set(item.id, newId);
        const newGroupId = item.groupId ? mqttTagGroupIdMap.get(item.groupId) : null;
        return {
          ...stripMeta(item),
          id: newId,
          projectId: project.id,
          subscriptionId: newSubscriptionId,
          groupId: newGroupId || null,
          createdBy: userId,
          updatedBy: userId,
        };
      })
      .filter(Boolean);
    if (mqttTagRows.length) {
      await DataMqttTag.bulkCreate(mqttTagRows);
    }

    const resolveSourceId = (sourceType, sourceId) => {
      if (!sourceId) return null;
      const type = String(sourceType || '').toLowerCase();
      if (type.includes('query')) return queryIdMap.get(sourceId) || null;
      if (type.includes('tag')) return mqttTagIdMap.get(sourceId) || null;
      if (type.includes('subscription')) return mqttSubscriptionIdMap.get(sourceId) || null;
      return sourceId;
    };

    const datapointRows = normalizeList(datacenter.datapoints).map((item) => {
      const newId = randomUUID();
      if (item?.id) {
        datapointIdMap.set(item.id, newId);
      }
      return {
        ...stripMeta(item),
        id: newId,
        projectId: project.id,
        sourceId: resolveSourceId(item.sourceType, item.sourceId),
        createdBy: userId,
        updatedBy: userId,
      };
    });
    if (datapointRows.length) {
      await DataPoint.bulkCreate(datapointRows);
    }
    const remapNeeded =
      queryIdMap.size ||
      mqttTagIdMap.size ||
      mqttSubscriptionIdMap.size ||
      datapointIdMap.size;
    if (remapNeeded) {
      const maps = {
        queryIdMap,
        mqttTagIdMap,
        mqttSubscriptionIdMap,
        datapointIdMap,
      };
      const remappedProjectVariables = remapVariables(projectVariables, maps);
      const remappedGlobalVariables = remapVariables(globalVariables, maps);
      await project.update({ projectVariables: remappedProjectVariables });
      await Project.sequelize.query(
        `UPDATE design_project_settings
          SET globalVariables = ?, updatedBy = ?, updatedAt = ?
          WHERE projectId = ?`,
        {
          replacements: [
            JSON.stringify(remappedGlobalVariables),
            userId,
            new Date(),
            project.id,
          ],
        },
      );
    }


    return ApiResponse.success(res, { projectId: project.id }, 'project_import_success', {}, 201);
  } catch (error) {
    logger.error('Import project error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/projects:
 *   post:
 *     summary: 创建工程
 *     tags: [工程管理]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - name
 *             properties:
 *               name:
 *                 type: string
 *               description:
 *                 type: string
 *               colorTag:
 *                 type: string
 *     responses:
 *       201:
 *         description: 创建成功
 */
router.post('/', authenticateToken, validate(Joi.object({
  body: Joi.object({
    name: Joi.string().required(),
    description: Joi.string().allow('').optional(),
    colorTag: Joi.string().valid('#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899', '#6b7280').optional()
  }).required()
})), async (req, res) => {
  try {
    const { name, description, colorTag } = req.body;
    const { tenantId, role, id: userId } = req.user;

    // 检查权限：只有系统管理员和工程管理员可以创建工程
    if (!['SYSTEM_ADMIN', 'PROJECT_ADMIN'].includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }

    // 创建工程
        const defaultProjectVariables = {};

    
    const defaultGlobalVariables = {
      definitions: defaultProjectVariables,
      groups: [],
    };
    const project = await Project.create({
      name,
      description,
      colorTag: colorTag || '#3b82f6',
      tenantId,
      createdBy: userId,
      projectVariables: defaultProjectVariables,
    });
    try {
      await Project.sequelize.query(
        `INSERT INTO design_project_settings
          (projectId, schemaVersion, globalVariables, globalScripts, updatedBy, updatedAt)
        VALUES (?, ?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
          globalVariables = VALUES(globalVariables),
          globalScripts = VALUES(globalScripts),
          updatedBy = VALUES(updatedBy),
          updatedAt = VALUES(updatedAt)`,
        {
          replacements: [
            project.id,
            '1.0.0',
            JSON.stringify(defaultGlobalVariables),
            JSON.stringify(defaultGlobalScripts),
            userId,
            new Date(),
          ],
        },
      );
    } catch (error) {
      logger.warn('Init project settings failed', { error: error.message, projectId: project.id });
    }
    const projectWithRelations = await Project.findByPk(project.id, {
      include: [
        { model: Tenant, as: 'tenant' },
        { model: User, as: 'creator', attributes: ['id', 'username', 'fullName'] }
      ]
    });

    return ApiResponse.success(
      res,
      { project: projectWithRelations },
      'project_create_success',
      {},
      201
    );

  } catch (error) {
    logger.error('Create project error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.PROJECT_CREATE_FAILED, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/projects/{id}:
 *   put:
 *     summary: 更新工程
 *     tags: [工程管理]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               name:
 *                 type: string
 *               description:
 *                 type: string
 *               colorTag:
 *                 type: string
 *     responses:
 *       200:
 *         description: 更新成功
 */
router.put('/:id', authenticateToken, requireResourceOwnership('project'), validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() }),
  body: Joi.object({
    name: Joi.string().optional(),
    description: Joi.string().allow('').optional(),
    colorTag: Joi.string().valid('#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899', '#6b7280').optional(),
  }).min(1)
})), async (req, res) => {
  try {
    const { id } = req.params;
    const updateData = req.body;
    const { role, id: userId } = req.user;

    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 404);
    }

    // 检查权限：只有系统管理员和工程管理员可以更新工程
    const allowedRoles = ['SYSTEM_ADMIN', 'PROJECT_ADMIN'];
    if (!allowedRoles.includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }

    // 更新工程
    await project.update({
      ...updateData,
      updatedBy: userId
    });

    // 返回更新后的工程信息
    const updatedProject = await Project.findByPk(id, {
      include: [
        { model: Tenant, as: 'tenant' },
        { model: User, as: 'creator', attributes: ['id', 'username', 'fullName'] },
        { model: User, as: 'updater', attributes: ['id', 'username', 'fullName'] }
      ]
    });

    return ApiResponse.success(res, { project: updatedProject }, 'update_success');

  } catch (error) {
    logger.error('Update project error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.PROJECT_UPDATE_FAILED, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/projects/{id}:
 *   delete:
 *     summary: 删除工程
 *     tags: [工程管理]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: 删除成功
 */
router.delete('/:id', authenticateToken, requireResourceOwnership('project'), validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() })
})), async (req, res) => {
  try {
    const { id } = req.params;
    const { role } = req.user;

    // 检查权限：只有系统管理员和工程管理员可以删除工程
    if (!['SYSTEM_ADMIN', 'PROJECT_ADMIN'].includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }

    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 404);
    }

    await designAssetService.deleteAssetsByProject(id);
    await designAssetService.deleteFoldersByProject(id);
    await project.destroy();

    return ApiResponse.success(res, null, 'delete_success');

  } catch (error) {
    logger.error('Delete project error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.PROJECT_DELETE_FAILED, {}, 500);
  }
});

/**
 * @swagger
 * /api/v1/projects/{id}/operations/{operation}:
 *   post:
 *     summary: 执行工程运维操作
 *     tags: [工程运维]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *       - in: path
 *         name: operation
 *         required: true
 *         schema:
 *           type: string
 *         enum: [start, stop, restart, deploy, backup]
 *     responses:
 *       200:
 *         description: 操作成功
 */
router.post('/:id/operations/:operation', authenticateToken, requireResourceOwnership('project'), validate(Joi.object({
  params: Joi.object({
    id: Joi.string().uuid().required(),
    operation: Joi.string().valid('start','stop','restart','deploy','backup').required()
  })
})), async (req, res) => {
  try {
    const { id, operation } = req.params;
    const { role, id: userId } = req.user;

    // 检查权限：只有运维管理员可以执行运维操作
    if (!['SYSTEM_ADMIN', 'OPS_ADMIN'].includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }

    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 404);
    }

    // 执行运维操作（这里是模拟，实际项目中需要调用具体的运维接口）
    const operationResults = {
      start: 'Project started successfully',
      stop: 'Project stopped successfully',
      restart: 'Project restarted successfully',
      deploy: 'Project deployed successfully',
      backup: 'Project backup completed successfully'
    };

    // 记录操作日志
    const Log = require('../../models').Log;
    await Log.create({
      level: 'info',
      message: `Project ${operation} operation performed`,
      action: `project:${operation}`,
      resource: 'project',
      resourceId: id,
      userId: userId,
      tenantId: project.tenantId
    });

    return ApiResponse.success(res, {
      message: operationResults[operation] || 'Operation completed',
      operation,
      projectId: id
    }, 'project_operation_success');

  } catch (error) {
    logger.error('Project operation error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.PROJECT_OPERATION_FAILED, {}, 500);
  }
});

module.exports = router;
