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
const projectOverviewService = require('../../services/projectOverviewService');
const {
  getProjectSettingsRow,
  upsertProjectSettings,
} = require('../../services/projectSettingsStore');
const {
  ensureRuntimeAdminBootstrap,
  listRuntimeUsers,
  createRuntimeUser,
  updateRuntimeUserStatus,
  resetRuntimeUserPassword,
  bindRuntimeUserRoles,
  listRuntimeRoles,
  createRuntimeRole,
  updateRuntimeRole,
  deleteRuntimeRole,
} = require('../../services/projectRuntimeAccessService');

const router = express.Router();

// 导入模型和中间件
const {
  Project,
  Tenant,
  User,
  ProjectTag,
  ProjectTagBinding,
  ProjectGroup,
  ProjectGroupMember,
  DesignPage,
  NodeDeployment,
} = require('../../models');
const { authenticateToken, requireResourceOwnership, hasCapability } = require('../../middlewares/auth');
const { randomUUID, randomBytes } = require('crypto');

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

const buildRelationalConfigsFromArtifact = (connections = []) =>
  normalizeList(connections)
    .filter((connection) => connection?.type === 'relational')
    .map((connection) => {
      const config = connection?.config || {};
      return {
        connectionId: connection.id,
        dbType: config.dbType || 'postgresql',
        host: config.host || '',
        port: config.port ?? 5432,
        database: config.database || '',
        username: config.username || '',
        password: config.password || '',
        schema: config.schema || null,
        charset: config.charset || null,
        timezone: config.timezone || null,
        ssl: Boolean(config.ssl),
        sslConfig: config.sslConfig || {},
      };
    });

const buildSnapshotFromArtifact = (artifact = {}) => {
  const mqttConnections = normalizeList(artifact?.mqtt?.connections);
  return {
    connections: normalizeList(artifact?.connections),
    relationalConfigs: buildRelationalConfigsFromArtifact(artifact?.connections),
    queries: normalizeList(artifact?.queries),
    mqttConfigs: mqttConnections.map((connection) => ({
      connectionId: connection.id,
      brokerUrl: connection.brokerUrl || '',
      protocol: connection.protocol || 'mqtt',
      port: connection.port ?? 1883,
      clientId: connection.clientId ?? null,
      username: connection.username ?? null,
      password: connection.password ?? null,
      keepalive: connection.keepalive ?? 60,
      cleanSession: Boolean(connection.cleanSession),
      qos: connection.qos ?? 0,
      reconnectPeriod: connection.reconnectPeriod ?? 1000,
      connectTimeout: connection.connectTimeout ?? 30000,
      will: connection.will || {},
      sslConfig: connection.sslConfig || {},
    })),
    mqttSubscriptions: normalizeList(artifact?.mqtt?.subscriptions),
    mqttTagGroups: normalizeList(artifact?.mqtt?.tagGroups),
    mqttTags: normalizeList(artifact?.mqtt?.tags),
    datapoints: normalizeList(artifact?.datapoints),
  };
};

const respondRouteError = (res, error, fallbackCode, fallbackStatus) => {
  if (error?.errorCode && error?.statusCode) {
    const normalizedStatus = error.statusCode >= 500 ? error.statusCode : 200;
    return ApiResponse.error(res, error.errorCode, error.options || {}, normalizedStatus);
  }

  return ApiResponse.error(res, fallbackCode, {}, fallbackStatus);
};

const buildInitialRuntimePassword = () => randomBytes(16).toString('hex');
const RUNTIME_ACCESS_ALLOWED_ROLES = ['SYSTEM_ADMIN', 'PROJECT_ADMIN'];
const PROJECT_OVERVIEW_SORT_FIELDS = ['createdAt', 'updatedAt', 'lastDeployedAt', 'runtimeStatus'];
const PROJECT_OVERVIEW_SORT_ORDERS = ['ASC', 'DESC'];
const PROJECT_TAG_LIMIT = 10;
const PROJECT_LABEL_MANAGE_ROLES = ['SYSTEM_ADMIN', 'PROJECT_ADMIN'];

const ensureProjectLabelManagePermission = (req, res) => {
  const role = req.user?.role;
  if (!PROJECT_LABEL_MANAGE_ROLES.includes(role)) {
    ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
    return false;
  }
  return true;
};

const buildProjectTagGroupScope = (req) => {
  return buildTenantWhere({}, req);
};

const requireRuntimeProjectManagement = async (req, res, next) => {
  try {
    const { id } = req.params;
    const { role, tenantId } = req.user || {};

    if (!RUNTIME_ACCESS_ALLOWED_ROLES.includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
    }

    const project = await Project.findByPk(id, {
      attributes: ['id', 'tenantId'],
    });
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 200);
    }

    if (role !== 'SYSTEM_ADMIN' && project.tenantId !== tenantId) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
    }

    req.runtimeManagedProject = project;
    return next();
  } catch (error) {
    logger.error('Runtime project management guard error', {
      error: error.message,
      requestId: req.requestId,
    });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
};

const cleanupFailedInitializedProject = async (projectRecord, projectId, requestId) => {
  try {
    const targetProject = projectRecord || (projectId ? await Project.findByPk(projectId) : null);
    if (targetProject?.destroy) {
      await targetProject.destroy();
    }
  } catch (cleanupError) {
    logger.error('Cleanup failed initialized project error', {
      error: cleanupError.message,
      projectId,
      requestId,
    });
  }
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
    name: Joi.string().optional(),
    group: Joi.string().optional(),
    groupId: Joi.string().optional(),
    tag: Joi.alternatives().try(Joi.string(), Joi.array().items(Joi.string())).optional(),
    tagId: Joi.alternatives().try(Joi.string(), Joi.array().items(Joi.string())).optional(),
    tags: Joi.alternatives().try(Joi.string(), Joi.array().items(Joi.string())).optional(),
    runtimeMode: Joi.alternatives().try(Joi.string().valid('DEV', 'RELEASE'), Joi.array().items(Joi.string().valid('DEV', 'RELEASE'))).optional(),
    deployStatus: Joi.alternatives().try(
      Joi.string().valid('pending', 'deploying', 'running', 'stopped', 'error', 'rollback'),
      Joi.array().items(Joi.string().valid('pending', 'deploying', 'running', 'stopped', 'error', 'rollback')),
    ).optional(),
    visibility: Joi.alternatives().try(
      Joi.string().valid('private', 'internal'),
      Joi.array().items(Joi.string().valid('private', 'internal')),
    ).optional(),
    createdBy: Joi.string().optional(),
    createdByName: Joi.string().optional(),
    sortBy: Joi.string().valid(...PROJECT_OVERVIEW_SORT_FIELDS).optional(),
    sortField: Joi.string().valid(...PROJECT_OVERVIEW_SORT_FIELDS).optional(),
    sortOrder: Joi.string().valid(...PROJECT_OVERVIEW_SORT_ORDERS, 'asc', 'desc').optional(),
    order: Joi.string().valid(...PROJECT_OVERVIEW_SORT_ORDERS, 'asc', 'desc').optional(),
  })
})), async (req, res) => {
  try {
    const result = await projectOverviewService.listProjectOverviews({
      req,
      query: req.query,
    });

    return ApiResponse.paginated(res, { projects: result.projects }, {
      total: result.total,
      page: result.page,
      limit: result.limit,
      totalPages: result.totalPages,
    });
  } catch (error) {
    logger.error('Get projects error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

router.get('/tags', authenticateToken, validate(Joi.object({
  query: Joi.object({
    keyword: Joi.string().allow('').optional(),
  }),
})), async (req, res) => {
  try {
    const where = buildProjectTagGroupScope(req);
    const keyword = String(req.query.keyword || '').trim();
    if (keyword) {
      where.name = { [Op.like]: `%${keyword}%` };
    }

    const tags = await ProjectTag.findAll({
      where,
      order: [['sortOrder', 'ASC'], ['createdAt', 'ASC']],
    });

    return ApiResponse.success(res, { list: { tags } });
  } catch (error) {
    logger.error('Get project tags error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

router.post('/tags', authenticateToken, validate(Joi.object({
  body: Joi.object({
    name: Joi.string().trim().min(1).max(100).required(),
    description: Joi.string().allow('').optional(),
    sortOrder: Joi.number().integer().default(0),
  }).required(),
})), async (req, res) => {
  try {
    if (!ensureProjectLabelManagePermission(req, res)) return;

    const tenantId = req.user?.tenantId;
    if (!tenantId) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: '租户上下文缺失' }, 200);
    }

    const name = req.body.name.trim();
    const existing = await ProjectTag.findOne({
      where: {
        tenantId,
        name,
      },
    });
    if (existing) {
      return ApiResponse.error(res, ErrorCodes.RESOURCE_ALREADY_EXISTS, { message: '标签名称已存在' }, 200);
    }

    const tagCount = await ProjectTag.count({
      where: {
        tenantId,
      },
    });
    if (tagCount >= PROJECT_TAG_LIMIT) {
      return ApiResponse.error(
        res,
        ErrorCodes.VALIDATION_FAILED,
        { message: `标签最多限制 ${PROJECT_TAG_LIMIT} 个` },
        200,
      );
    }

    const tag = await ProjectTag.create({
      tenantId,
      name,
      description: req.body.description || null,
      sortOrder: req.body.sortOrder ?? 0,
      createdBy: req.user.id,
      updatedBy: req.user.id,
    });
    return ApiResponse.success(res, { tag }, 'create_success', {}, 201);
  } catch (error) {
    logger.error('Create project tag error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

router.put('/tags/:tagId', authenticateToken, validate(Joi.object({
  params: Joi.object({
    tagId: Joi.string().uuid().required(),
  }),
  body: Joi.object({
    name: Joi.string().trim().min(1).max(100).optional(),
    description: Joi.string().allow('').optional(),
    sortOrder: Joi.number().integer().optional(),
  }).min(1).required(),
})), async (req, res) => {
  try {
    if (!ensureProjectLabelManagePermission(req, res)) return;

    const where = buildProjectTagGroupScope(req);
    where.id = req.params.tagId;
    const tag = await ProjectTag.findOne({ where });
    if (!tag) {
      return ApiResponse.error(res, ErrorCodes.RESOURCE_NOT_FOUND, { message: '标签不存在' }, 200);
    }

    if (req.body.name && req.body.name.trim() !== tag.name) {
      const duplicate = await ProjectTag.findOne({
        where: {
          tenantId: tag.tenantId,
          name: req.body.name.trim(),
          id: { [Op.ne]: tag.id },
        },
      });
      if (duplicate) {
        return ApiResponse.error(res, ErrorCodes.RESOURCE_ALREADY_EXISTS, { message: '标签名称已存在' }, 200);
      }
    }

    await tag.update({
      ...(req.body.name ? { name: req.body.name.trim() } : {}),
      ...(Object.prototype.hasOwnProperty.call(req.body, 'description') ? { description: req.body.description || null } : {}),
      ...(Object.prototype.hasOwnProperty.call(req.body, 'sortOrder') ? { sortOrder: req.body.sortOrder } : {}),
      updatedBy: req.user.id,
    });

    return ApiResponse.success(res, { tag }, 'update_success');
  } catch (error) {
    logger.error('Update project tag error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

router.delete('/tags/:tagId', authenticateToken, validate(Joi.object({
  params: Joi.object({
    tagId: Joi.string().uuid().required(),
  }),
})), async (req, res) => {
  try {
    if (!ensureProjectLabelManagePermission(req, res)) return;

    const where = buildProjectTagGroupScope(req);
    where.id = req.params.tagId;
    const tag = await ProjectTag.findOne({ where });
    if (!tag) {
      return ApiResponse.error(res, ErrorCodes.RESOURCE_NOT_FOUND, { message: '标签不存在' }, 200);
    }

    await tag.destroy();
    return ApiResponse.success(res, null, 'delete_success');
  } catch (error) {
    logger.error('Delete project tag error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

router.get('/groups', authenticateToken, validate(Joi.object({
  query: Joi.object({
    keyword: Joi.string().allow('').optional(),
  }),
})), async (req, res) => {
  try {
    const where = buildProjectTagGroupScope(req);
    const keyword = String(req.query.keyword || '').trim();
    if (keyword) {
      where.name = { [Op.like]: `%${keyword}%` };
    }

    const groups = await ProjectGroup.findAll({
      where,
      order: [['sortOrder', 'ASC'], ['createdAt', 'ASC']],
    });

    const groupIds = groups.map((group) => group.id).filter(Boolean);
    const projectCountByGroupId = new Map();
    if (groupIds.length > 0) {
      const members = await ProjectGroupMember.findAll({
        where: {
          ...buildProjectTagGroupScope(req),
          groupId: { [Op.in]: groupIds },
        },
        attributes: ['groupId', 'projectId'],
      });
      const projectIds = [...new Set(members.map((member) => member.projectId).filter(Boolean))];
      const visibleProjectIdSet = new Set();
      if (projectIds.length > 0) {
        const projects = await Project.findAll({
          where: {
            ...buildProjectTagGroupScope(req),
            id: { [Op.in]: projectIds },
          },
          attributes: ['id', 'createdBy', 'visibility'],
        });
        const canSeeSharedProject = hasCapability(req.user?.role, 'project:write');
        const canSeePublicProject = hasCapability(req.user?.role, 'project:read');
        for (const project of projects) {
          const visibility = String(project.visibility || 'private').toLowerCase();
          const isCreator = String(project.createdBy) === String(req.user?.id);
          if (
            isCreator
            || (visibility === 'internal' && canSeeSharedProject)
            || (visibility === 'public' && canSeePublicProject)
          ) {
            visibleProjectIdSet.add(project.id);
          }
        }
      }

      for (const member of members) {
        const groupId = member.groupId;
        if (!groupId || !visibleProjectIdSet.has(member.projectId)) continue;
        projectCountByGroupId.set(groupId, (projectCountByGroupId.get(groupId) || 0) + 1);
      }
    }

    const groupsWithCount = groups.map((group) => {
      const payload = typeof group.toJSON === 'function' ? group.toJSON() : { ...group };
      return {
        ...payload,
        projectCount: projectCountByGroupId.get(group.id) || 0,
      };
    });

    return ApiResponse.success(res, { list: { groups: groupsWithCount } });
  } catch (error) {
    logger.error('Get project groups error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

router.post('/groups', authenticateToken, validate(Joi.object({
  body: Joi.object({
    name: Joi.string().trim().min(1).max(100).required(),
    description: Joi.string().allow('').optional(),
    sortOrder: Joi.number().integer().default(0),
  }).required(),
})), async (req, res) => {
  try {
    if (!ensureProjectLabelManagePermission(req, res)) return;

    const tenantId = req.user?.tenantId;
    if (!tenantId) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: '租户上下文缺失' }, 200);
    }

    const name = req.body.name.trim();
    const existing = await ProjectGroup.findOne({
      where: {
        tenantId,
        name,
      },
    });
    if (existing) {
      return ApiResponse.error(res, ErrorCodes.RESOURCE_ALREADY_EXISTS, { message: '分组名称已存在' }, 200);
    }

    const group = await ProjectGroup.create({
      tenantId,
      name,
      description: req.body.description || null,
      sortOrder: req.body.sortOrder ?? 0,
      createdBy: req.user.id,
      updatedBy: req.user.id,
    });
    return ApiResponse.success(res, { group }, 'create_success', {}, 201);
  } catch (error) {
    logger.error('Create project group error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

router.put('/groups/:groupId', authenticateToken, validate(Joi.object({
  params: Joi.object({
    groupId: Joi.string().uuid().required(),
  }),
  body: Joi.object({
    name: Joi.string().trim().min(1).max(100).optional(),
    description: Joi.string().allow('').optional(),
    sortOrder: Joi.number().integer().optional(),
  }).min(1).required(),
})), async (req, res) => {
  try {
    if (!ensureProjectLabelManagePermission(req, res)) return;

    const where = buildProjectTagGroupScope(req);
    where.id = req.params.groupId;
    const group = await ProjectGroup.findOne({ where });
    if (!group) {
      return ApiResponse.error(res, ErrorCodes.RESOURCE_NOT_FOUND, { message: '分组不存在' }, 200);
    }

    if (req.body.name && req.body.name.trim() !== group.name) {
      const duplicate = await ProjectGroup.findOne({
        where: {
          tenantId: group.tenantId,
          name: req.body.name.trim(),
          id: { [Op.ne]: group.id },
        },
      });
      if (duplicate) {
        return ApiResponse.error(res, ErrorCodes.RESOURCE_ALREADY_EXISTS, { message: '分组名称已存在' }, 200);
      }
    }

    await group.update({
      ...(req.body.name ? { name: req.body.name.trim() } : {}),
      ...(Object.prototype.hasOwnProperty.call(req.body, 'description') ? { description: req.body.description || null } : {}),
      ...(Object.prototype.hasOwnProperty.call(req.body, 'sortOrder') ? { sortOrder: req.body.sortOrder } : {}),
      updatedBy: req.user.id,
    });

    return ApiResponse.success(res, { group }, 'update_success');
  } catch (error) {
    logger.error('Update project group error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

router.delete('/groups/:groupId', authenticateToken, validate(Joi.object({
  params: Joi.object({
    groupId: Joi.string().uuid().required(),
  }),
})), async (req, res) => {
  try {
    if (!ensureProjectLabelManagePermission(req, res)) return;

    const where = buildProjectTagGroupScope(req);
    where.id = req.params.groupId;
    const group = await ProjectGroup.findOne({ where });
    if (!group) {
      return ApiResponse.error(res, ErrorCodes.RESOURCE_NOT_FOUND, { message: '分组不存在' }, 200);
    }

    await group.destroy();
    return ApiResponse.success(res, null, 'delete_success');
  } catch (error) {
    logger.error('Delete project group error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

router.put('/:id/tags', authenticateToken, requireResourceOwnership('project'), validate(Joi.object({
  params: Joi.object({
    id: Joi.string().uuid().required(),
  }),
  body: Joi.object({
    tagIds: Joi.array().items(Joi.string().uuid()).max(PROJECT_TAG_LIMIT).required(),
  }).required(),
})), async (req, res) => {
  try {
    if (!ensureProjectLabelManagePermission(req, res)) return;
    if (
      !Object.prototype.hasOwnProperty.call(req.body || {}, 'tagIds')
      || !Array.isArray(req.body.tagIds)
    ) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: 'tagIds 必须显式传入数组' }, 200);
    }

    const project = await Project.findByPk(req.params.id, {
      attributes: ['id', 'tenantId'],
    });
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 200);
    }

    const tagIds = [...new Set((Array.isArray(req.body.tagIds) ? req.body.tagIds : []).map((item) => String(item)))];
    const tags = tagIds.length
      ? await ProjectTag.findAll({
        where: {
          tenantId: project.tenantId,
          id: { [Op.in]: tagIds },
        },
        attributes: ['id'],
      })
      : [];
    if (tags.length !== tagIds.length) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: '存在无效标签或跨租户标签' }, 200);
    }

    await Project.sequelize.transaction(async (transaction) => {
      await ProjectTagBinding.destroy({
        where: {
          projectId: project.id,
        },
        transaction,
      });

      if (!tagIds.length) return;

      await ProjectTagBinding.bulkCreate(
        tagIds.map((tagId) => ({
          tenantId: project.tenantId,
          projectId: project.id,
          tagId,
          createdBy: req.user.id,
        })),
        { transaction },
      );
    });

    return ApiResponse.success(res, { projectId: project.id, tagIds });
  } catch (error) {
    logger.error('Bind project tags error', { error: error.message, requestId: req.requestId });
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
});

router.put('/:id/group', authenticateToken, requireResourceOwnership('project'), validate(Joi.object({
  params: Joi.object({
    id: Joi.string().uuid().required(),
  }),
  body: Joi.object({
    groupId: Joi.string().uuid().allow(null).required(),
  }).required(),
})), async (req, res) => {
  try {
    if (!ensureProjectLabelManagePermission(req, res)) return;

    const project = await Project.findByPk(req.params.id, {
      attributes: ['id', 'tenantId'],
    });
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 200);
    }

    const groupId = req.body.groupId || null;
    if (!groupId) {
      await ProjectGroupMember.destroy({
        where: { projectId: project.id },
      });
      return ApiResponse.success(res, { projectId: project.id, groupId: null }, 'update_success');
    }

    const group = await ProjectGroup.findOne({
      where: {
        id: groupId,
        tenantId: project.tenantId,
      },
      attributes: ['id'],
    });
    if (!group) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: '分组不存在或不属于当前租户' }, 200);
    }

    await Project.sequelize.transaction(async (transaction) => {
      await ProjectGroupMember.destroy({
        where: { projectId: project.id },
        transaction,
      });
      await ProjectGroupMember.create({
        tenantId: project.tenantId,
        projectId: project.id,
        groupId,
        createdBy: req.user.id,
      }, { transaction });
    });

    return ApiResponse.success(res, { projectId: project.id, groupId }, 'update_success');
  } catch (error) {
    logger.error('Bind project group error', { error: error.message, requestId: req.requestId });
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
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
    }
    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 200);
    }

    const pages = await DesignPage.findAll({
      where: { projectId: id },
      order: [['sortOrder', 'ASC']],
    });

    const settingsRow = await getProjectSettingsRow(Project.sequelize, id);

    const globalVariables = parseJsonField(settingsRow?.globalVariables, null);
    const globalScripts = parseJsonField(settingsRow?.globalScripts, null);

    const artifact = await dataDomainClient.getProjectArtifact(id, req.headers.authorization);
    const snapshot = buildSnapshotFromArtifact(artifact);

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
    archive.append(
      JSON.stringify(artifact || {}, null, 2),
      { name: 'datacenter/artifact.json' }
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
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
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

    let project = null;
    let projectId = null;

    const globalVariables = settings.globalVariables || {
      definitions: projectVariables,
      groups: [],
    };
    const globalScripts = settings.globalScripts || null;

    const pageIdMap = new Map();
    pageList.forEach((item) => {
      const page = item.page || item;
      if (page?.id && !pageIdMap.has(page.id)) {
        pageIdMap.set(page.id, randomUUID());
      }
    });

    await Project.sequelize.transaction(async (transaction) => {
      project = await Project.create({
        name: finalName,
        description: projectInfo.description || '',
        colorTag: projectInfo.colorTag || '#3b82f6',
        tenantId,
        createdBy: userId,
        projectVariables,
        entryConfig: {},
      }, {
        transaction,
      });
      projectId = project.id;

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
        }, {
          transaction,
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

      await project.update({ entryConfig: nextEntry }, { transaction });

      await ensureRuntimeAdminBootstrap({
        project,
        creator: {
          id: userId,
          username: req.user.username,
          fullName: req.user.fullName,
        },
        initialPassword: buildInitialRuntimePassword(),
        transaction,
      });
    });

    try {
      await upsertProjectSettings(Project.sequelize, {
        projectId,
        schemaVersion: '1.0.0',
        globalVariables,
        globalScripts: globalScripts || {},
        updatedBy: userId,
        updatedAt: new Date(),
      });

      const snapshot = buildProjectSnapshot(payload.datacenter || {});
      await dataDomainClient.replaceProjectSnapshot(
        projectId,
        snapshot,
        req.headers.authorization,
      );
    } catch (postCommitError) {
      await cleanupFailedInitializedProject(project, projectId, req.requestId);
      throw postCommitError;
    }


    return ApiResponse.success(res, { projectId }, 'project_import_success', {}, 201);
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
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
    }

    // 创建工程
        const defaultProjectVariables = {};

    
    const defaultGlobalVariables = {
      definitions: defaultProjectVariables,
      groups: [],
    };
    const defaultGlobalScripts = DEFAULT_GLOBAL_SCRIPTS;
    let project = null;
    let projectId = null;

    await Project.sequelize.transaction(async (transaction) => {
      project = await Project.create({
        name,
        description,
        colorTag: colorTag || '#3b82f6',
        tenantId,
        createdBy: userId,
        projectVariables: defaultProjectVariables,
      }, {
        transaction,
      });
      projectId = project.id;

      await ensureRuntimeAdminBootstrap({
        project,
        creator: {
          id: userId,
          username: req.user.username,
          fullName: req.user.fullName,
        },
        initialPassword: buildInitialRuntimePassword(),
        transaction,
      });
    });

    try {
      await upsertProjectSettings(Project.sequelize, {
        projectId,
        schemaVersion: '1.0.0',
        globalVariables: defaultGlobalVariables,
        globalScripts: defaultGlobalScripts,
        updatedBy: userId,
        updatedAt: new Date(),
      });
    } catch (postCommitError) {
      await cleanupFailedInitializedProject(project, projectId, req.requestId);
      throw postCommitError;
    }

    const projectWithRelations = await Project.findByPk(projectId, {
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

router.get('/:id/runtime-users', authenticateToken, requireRuntimeProjectManagement, validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() })
})), async (req, res) => {
  try {
    const { id } = req.params;
    const runtimeUsers = await listRuntimeUsers({ projectId: id });
    return ApiResponse.success(res, { runtimeUsers });
  } catch (error) {
    logger.error('List runtime users error', { error: error.message, requestId: req.requestId });
    return respondRouteError(res, error, ErrorCodes.INTERNAL_SERVER_ERROR, 500);
  }
});

router.post('/:id/runtime-users', authenticateToken, requireRuntimeProjectManagement, validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() }),
  body: Joi.object({
    username: Joi.string().trim().required(),
    displayName: Joi.string().allow('').optional(),
    initialPassword: Joi.string().min(1).required(),
    roleIds: Joi.array().items(Joi.string().uuid()).optional(),
  }).required(),
})), async (req, res) => {
  try {
    const { id } = req.params;
    const { username, displayName, initialPassword, roleIds } = req.body;
    const runtimeUser = await createRuntimeUser({
      projectId: id,
      actorId: req.user.id,
      username,
      displayName,
      initialPassword,
      roleIds,
    });
    return ApiResponse.success(res, { runtimeUser }, 'create_success', {}, 201);
  } catch (error) {
    logger.error('Create runtime user error', { error: error.message, requestId: req.requestId });
    return respondRouteError(res, error, ErrorCodes.INTERNAL_SERVER_ERROR, 500);
  }
});

router.patch('/:id/runtime-users/:runtimeUserId/status', authenticateToken, requireRuntimeProjectManagement, validate(Joi.object({
  params: Joi.object({
    id: Joi.string().uuid().required(),
    runtimeUserId: Joi.string().uuid().required(),
  }),
  body: Joi.object({
    status: Joi.string().valid('active', 'disabled').required(),
  }).required(),
})), async (req, res) => {
  try {
    const { id, runtimeUserId } = req.params;
    const { status } = req.body;
    const runtimeUser = await updateRuntimeUserStatus({
      projectId: id,
      runtimeUserId,
      status,
      actorId: req.user.id,
    });
    return ApiResponse.success(res, { runtimeUser });
  } catch (error) {
    logger.error('Update runtime user status error', { error: error.message, requestId: req.requestId });
    return respondRouteError(res, error, ErrorCodes.INTERNAL_SERVER_ERROR, 500);
  }
});

router.post('/:id/runtime-users/:runtimeUserId/reset-password', authenticateToken, requireRuntimeProjectManagement, validate(Joi.object({
  params: Joi.object({
    id: Joi.string().uuid().required(),
    runtimeUserId: Joi.string().uuid().required(),
  }),
  body: Joi.object({
    newPassword: Joi.string().min(1).required(),
  }).required(),
})), async (req, res) => {
  try {
    const { id, runtimeUserId } = req.params;
    const { newPassword } = req.body;
    const runtimeUser = await resetRuntimeUserPassword({
      projectId: id,
      runtimeUserId,
      newPassword,
      actorId: req.user.id,
    });
    return ApiResponse.success(res, { runtimeUser });
  } catch (error) {
    logger.error('Reset runtime user password error', { error: error.message, requestId: req.requestId });
    return respondRouteError(res, error, ErrorCodes.INTERNAL_SERVER_ERROR, 500);
  }
});

router.put('/:id/runtime-users/:runtimeUserId/roles', authenticateToken, requireRuntimeProjectManagement, validate(Joi.object({
  params: Joi.object({
    id: Joi.string().uuid().required(),
    runtimeUserId: Joi.string().uuid().required(),
  }),
  body: Joi.object({
    roleIds: Joi.array().items(Joi.string().uuid()).default([]),
  }).required(),
})), async (req, res) => {
  try {
    const { id, runtimeUserId } = req.params;
    const { roleIds } = req.body;
    const runtimeUser = await bindRuntimeUserRoles({
      projectId: id,
      runtimeUserId,
      roleIds,
      actorId: req.user.id,
    });
    return ApiResponse.success(res, { runtimeUser });
  } catch (error) {
    logger.error('Bind runtime user roles error', { error: error.message, requestId: req.requestId });
    return respondRouteError(res, error, ErrorCodes.INTERNAL_SERVER_ERROR, 500);
  }
});

router.get('/:id/runtime-roles', authenticateToken, requireRuntimeProjectManagement, validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() })
})), async (req, res) => {
  try {
    const { id } = req.params;
    const runtimeRoles = await listRuntimeRoles({ projectId: id });
    return ApiResponse.success(res, { runtimeRoles });
  } catch (error) {
    logger.error('List runtime roles error', { error: error.message, requestId: req.requestId });
    return respondRouteError(res, error, ErrorCodes.INTERNAL_SERVER_ERROR, 500);
  }
});

router.post('/:id/runtime-roles', authenticateToken, requireRuntimeProjectManagement, validate(Joi.object({
  params: Joi.object({ id: Joi.string().uuid().required() }),
  body: Joi.object({
    code: Joi.string().trim().required(),
    name: Joi.string().trim().required(),
    description: Joi.string().allow('').optional(),
  }).required(),
})), async (req, res) => {
  try {
    const { id } = req.params;
    const role = await createRuntimeRole({
      projectId: id,
      actorId: req.user.id,
      code: req.body.code,
      name: req.body.name,
      description: req.body.description,
    });
    return ApiResponse.success(res, { role }, 'create_success', {}, 201);
  } catch (error) {
    logger.error('Create runtime role error', { error: error.message, requestId: req.requestId });
    return respondRouteError(res, error, ErrorCodes.INTERNAL_SERVER_ERROR, 500);
  }
});

router.put('/:id/runtime-roles/:roleId', authenticateToken, requireRuntimeProjectManagement, validate(Joi.object({
  params: Joi.object({
    id: Joi.string().uuid().required(),
    roleId: Joi.string().uuid().required(),
  }),
  body: Joi.object({
    code: Joi.string().trim().optional(),
    name: Joi.string().trim().optional(),
    description: Joi.string().allow('').optional(),
    status: Joi.string().valid('active', 'disabled').optional(),
  }).min(1).required(),
})), async (req, res) => {
  try {
    const { id, roleId } = req.params;
    const role = await updateRuntimeRole({
      projectId: id,
      roleId,
      actorId: req.user.id,
      ...req.body,
    });
    return ApiResponse.success(res, { role });
  } catch (error) {
    logger.error('Update runtime role error', { error: error.message, requestId: req.requestId });
    return respondRouteError(res, error, ErrorCodes.INTERNAL_SERVER_ERROR, 500);
  }
});

router.delete('/:id/runtime-roles/:roleId', authenticateToken, requireRuntimeProjectManagement, validate(Joi.object({
  params: Joi.object({
    id: Joi.string().uuid().required(),
    roleId: Joi.string().uuid().required(),
  }),
})), async (req, res) => {
  try {
    const { id, roleId } = req.params;
    await deleteRuntimeRole({
      projectId: id,
      roleId,
      actorId: req.user.id,
    });
    return ApiResponse.success(res, null, 'delete_success');
  } catch (error) {
    logger.error('Delete runtime role error', { error: error.message, requestId: req.requestId });
    return respondRouteError(res, error, ErrorCodes.INTERNAL_SERVER_ERROR, 500);
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
    visibility: Joi.string().valid('private', 'internal').optional(),
  }).min(1)
})), async (req, res) => {
  try {
    const { id } = req.params;
    const updateData = req.body;
    const { role, id: userId } = req.user;

    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 200);
    }

    // 检查权限：只有系统管理员和工程管理员可以更新工程
    const allowedRoles = ['SYSTEM_ADMIN', 'PROJECT_ADMIN'];
    if (!allowedRoles.includes(role)) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
    }
    if (
      Object.prototype.hasOwnProperty.call(updateData, 'visibility')
      && String(project.createdBy) !== String(userId)
    ) {
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
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
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
    }
    if (forceDelete && role !== 'SYSTEM_ADMIN') {
      return ApiResponse.error(
        res,
        ErrorCodes.PERMISSION_INSUFFICIENT,
        { message: '仅系统管理员可执行强制删除' },
        200
      );
    }

    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 200);
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
          200
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
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
    }

    const project = await Project.findByPk(id, {
      attributes: ['id', 'name'],
    });
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 200);
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
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
    }

    const project = await Project.findByPk(id);
    if (!project) {
      return ApiResponse.error(res, ErrorCodes.PROJECT_NOT_FOUND, {}, 200);
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
