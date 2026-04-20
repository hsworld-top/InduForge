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
const { dataDomainClient } = require('../../services/dataDomainClient');
const deploymentService = require('../../services/deploymentService');
const {
  getProjectSettingsRow,
  upsertProjectSettings,
} = require('../../services/projectSettingsStore');

const router = express.Router();

// 导入模型和中间件
const {
  Project,
  Tenant,
  User,
  DesignPage,
  NodeDeployment,
} = require('../../models');
const { authenticateToken, requireResourceOwnership, hasCapability } = require('../../middlewares/auth');
const { randomUUID } = require('crypto');

const DEFAULT_GLOBAL_SCRIPTS = {
  system: {
    startup: { code: '' },
    shutdown: { code: '' },
  },
  timers: { groups: [], items: [] },
  variableChanges: { groups: [], items: [] },
  custom: { groups: [], items: [] },
};

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

const normalizeList = (value) => (Array.isArray(value) ? value : []);

const mergeConnectionConfig = (connection, relationalConfig, mqttConfig) => {
  const baseConfig =
    connection?.config && typeof connection.config === 'object'
      ? { ...connection.config }
      : {};

  if (connection?.type === 'relational' && relationalConfig) {
    return {
      ...baseConfig,
      dbType: relationalConfig.dbType || baseConfig.dbType || 'postgresql',
      host: relationalConfig.host || baseConfig.host || '',
      port: relationalConfig.port || baseConfig.port || 5432,
      database: relationalConfig.database || baseConfig.database || '',
      username: relationalConfig.username || baseConfig.username || '',
      password: relationalConfig.password || baseConfig.password || '',
      schema: relationalConfig.schema ?? baseConfig.schema ?? null,
      charset: relationalConfig.charset ?? baseConfig.charset ?? null,
      timezone: relationalConfig.timezone ?? baseConfig.timezone ?? null,
      ssl: Boolean(relationalConfig.ssl ?? baseConfig.ssl),
      sslConfig: relationalConfig.sslConfig || baseConfig.sslConfig || {},
    };
  }

  if (connection?.type === 'mqtt' && mqttConfig) {
    return {
      ...baseConfig,
      brokerUrl: mqttConfig.brokerUrl || baseConfig.brokerUrl || '',
      protocol: mqttConfig.protocol || baseConfig.protocol || 'mqtt',
      port: mqttConfig.port || baseConfig.port || 1883,
      clientId: mqttConfig.clientId ?? baseConfig.clientId ?? null,
      username: mqttConfig.username ?? baseConfig.username ?? null,
      password: mqttConfig.password ?? baseConfig.password ?? null,
      keepalive: mqttConfig.keepalive ?? baseConfig.keepalive ?? 60,
      cleanSession: Boolean(mqttConfig.cleanSession ?? baseConfig.cleanSession ?? true),
      qos: mqttConfig.qos ?? baseConfig.qos ?? 0,
      reconnectPeriod: mqttConfig.reconnectPeriod ?? baseConfig.reconnectPeriod ?? 1000,
      connectTimeout: mqttConfig.connectTimeout ?? baseConfig.connectTimeout ?? 30000,
      will: mqttConfig.will || baseConfig.will || {},
      sslConfig: mqttConfig.sslConfig || baseConfig.sslConfig || {},
    };
  }

  return baseConfig;
};

const buildProjectSnapshot = (datacenter = {}) => {
  const relationalConfigs = normalizeList(datacenter.relationalConfigs);
  const mqttConfigs = normalizeList(datacenter.mqttConfigs);
  const relationalConfigMap = new Map(
    relationalConfigs
      .filter((item) => item?.connectionId)
      .map((item) => [item.connectionId, item])
  );
  const mqttConfigMap = new Map(
    mqttConfigs
      .filter((item) => item?.connectionId)
      .map((item) => [item.connectionId, item])
  );

  const connections = normalizeList(datacenter.connections).map((item) => ({
    ...item,
    config: mergeConnectionConfig(
      item,
      relationalConfigMap.get(item.id),
      mqttConfigMap.get(item.id),
    ),
  }));

  return {
    connections,
    relationalConfigs,
    queries: normalizeList(datacenter.queries),
    mqttConfigs,
    mqttSubscriptions: normalizeList(datacenter.mqttSubscriptions),
    mqttTagGroups: normalizeList(datacenter.mqttTagGroups),
    mqttTags: normalizeList(datacenter.mqttTags),
    datapoints: normalizeList(datacenter.datapoints),
  };
};

const respondRouteError = (res, error, fallbackCode, fallbackStatus) => {
  if (error?.errorCode && error?.statusCode) {
    return ApiResponse.error(res, error.errorCode, error.options || {}, error.statusCode);
  }

  return ApiResponse.error(res, fallbackCode, {}, fallbackStatus);
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
    const { role } = req.user;
    if (!['SYSTEM_ADMIN', 'PROJECT_ADMIN'].includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }
    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 404);
    }

    const pages = await DesignPage.findAll({
      where: { projectId: id },
      order: [['sortOrder', 'ASC']],
    });

    const settingsRow = await getProjectSettingsRow(Project.sequelize, id);

    const globalVariables = parseJsonField(settingsRow?.globalVariables, null);
    const globalScripts = parseJsonField(settingsRow?.globalScripts, null);

    const snapshot = await dataDomainClient.getProjectSnapshot(id, req.headers.authorization);

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
      JSON.stringify(snapshot.connections, null, 2),
      { name: 'datacenter/connections.json' }
    );
    archive.append(
      JSON.stringify(snapshot.relationalConfigs, null, 2),
      { name: 'datacenter/relational-configs.json' }
    );
    archive.append(
      JSON.stringify(snapshot.queries, null, 2),
      { name: 'datacenter/queries.json' }
    );
    archive.append(
      JSON.stringify(snapshot.mqttConfigs, null, 2),
      { name: 'datacenter/mqtt-configs.json' }
    );
    archive.append(
      JSON.stringify(snapshot.mqttSubscriptions, null, 2),
      { name: 'datacenter/mqtt-subscriptions.json' }
    );
    archive.append(
      JSON.stringify(snapshot.mqttTagGroups, null, 2),
      { name: 'datacenter/mqtt-tag-groups.json' }
    );
    archive.append(
      JSON.stringify(snapshot.mqttTags, null, 2),
      { name: 'datacenter/mqtt-tags.json' }
    );
    archive.append(
      JSON.stringify(snapshot.datapoints, null, 2),
      { name: 'datacenter/datapoints.json' }
    );

    await archive.finalize();
    return;
  } catch (error) {
    logger.error('Export project error', { error: error.message, requestId: req.requestId });
    return respondRouteError(res, error, ErrorCodes.INTERNAL_SERVER_ERROR, 500);
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

    await upsertProjectSettings(Project.sequelize, {
      projectId: project.id,
      schemaVersion: '1.0.0',
      globalVariables,
      globalScripts: globalScripts || {},
      updatedBy: userId,
      updatedAt: new Date(),
    });

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

    const snapshot = buildProjectSnapshot(payload.datacenter || {});
    await dataDomainClient.replaceProjectSnapshot(
      project.id,
      snapshot,
      req.headers.authorization,
    );


    return ApiResponse.success(res, { projectId: project.id }, 'project_import_success', {}, 201);
  } catch (error) {
    logger.error('Import project error', { error: error.message, requestId: req.requestId });
    return respondRouteError(res, error, ErrorCodes.INTERNAL_SERVER_ERROR, 500);
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
    const defaultGlobalScripts = DEFAULT_GLOBAL_SCRIPTS;
    const project = await Project.create({
      name,
      description,
      colorTag: colorTag || '#3b82f6',
      tenantId,
      createdBy: userId,
      projectVariables: defaultProjectVariables,
    });
    try {
      await upsertProjectSettings(Project.sequelize, {
        projectId: project.id,
        schemaVersion: '1.0.0',
        globalVariables: defaultGlobalVariables,
        globalScripts: defaultGlobalScripts,
        updatedBy: userId,
        updatedAt: new Date(),
      });
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
  params: Joi.object({ id: Joi.string().uuid().required() }),
  body: Joi.object({
    force: Joi.boolean().optional().default(false)
  })
})), async (req, res) => {
  try {
    const { id } = req.params;
    const { role } = req.user;
    const forceDelete = Boolean(req.body?.force);

    // 检查权限：只有系统管理员和工程管理员可以删除工程
    if (!['SYSTEM_ADMIN', 'PROJECT_ADMIN'].includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }
    if (forceDelete && role !== 'SYSTEM_ADMIN') {
      return ApiResponse.error(
        res,
        ErrorCodes.PERMISSION_INSUFFICIENT,
        { message: '仅系统管理员可执行强制删除' },
        403
      );
    }

    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 404);
    }

    const activeStatuses = ['pending', 'deploying', 'running'];
    const activeDeployments = await NodeDeployment.findAll({
      where: {
        projectId: id,
        status: { [Op.in]: activeStatuses },
        deletedAt: null,
      },
      attributes: ['id', 'nodeId', 'status'],
    });

    if (activeDeployments.length > 0) {
      if (!forceDelete) {
        return ApiResponse.error(
          res,
          ErrorCodes.VALIDATION_FAILED,
          {
            message: `工程存在 ${activeDeployments.length} 个运行中部署，请先在运维中心下线后再删除`,
            hasActiveDeployments: true,
            activeDeploymentCount: activeDeployments.length,
            canForceDelete: role === 'SYSTEM_ADMIN',
          },
          400
        );
      }
    }

    // 强制删除：先撤销该工程在所有节点的部署，再删除工程。
    if (forceDelete) {
      const allNodeDeployments = await NodeDeployment.findAll({
        where: { projectId: id, deletedAt: null },
        attributes: ['id'],
      });
      for (const item of allNodeDeployments) {
        // 复用运维撤销流程，确保节点运行指针被清理。
        await deploymentService.undeploy(item.id);
      }
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
 * /api/v1/projects/{id}/delete-impact:
 *   get:
 *     summary: 获取删除工程影响评估
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
 *         description: 获取成功
 */
router.get('/:id/delete-impact', authenticateToken, requireResourceOwnership('project'), validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() })
})), async (req, res) => {
  try {
    const { id } = req.params;
    const { role } = req.user;
    if (!['SYSTEM_ADMIN', 'PROJECT_ADMIN'].includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 403);
    }

    const project = await Project.findByPk(id, {
      attributes: ['id', 'name'],
    });
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 404);
    }

    const allDeployments = await NodeDeployment.findAll({
      where: { projectId: id, deletedAt: null },
      attributes: ['id', 'nodeId', 'status'],
    });
    const activeStatuses = new Set(['pending', 'deploying', 'running']);
    const activeDeployments = allDeployments.filter((item) => activeStatuses.has(item.status));
    const activeNodeIdSet = new Set(activeDeployments.map((item) => item.nodeId));

    return ApiResponse.success(res, {
      projectId: project.id,
      projectName: project.name,
      totalDeploymentCount: allDeployments.length,
      activeDeploymentCount: activeDeployments.length,
      activeNodeCount: activeNodeIdSet.size,
      hasActiveDeployments: activeDeployments.length > 0,
      canForceDelete: role === 'SYSTEM_ADMIN',
    });
  } catch (error) {
    logger.error('Get project delete impact error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
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

    // 检查权限：具备 runtime:operate 能力方可执行运维操作
    if (!hasCapability(role, 'runtime:operate')) {
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
