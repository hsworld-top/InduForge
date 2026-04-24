const { Op } = require('sequelize');
const appConfig = require('../config/app');
const { buildTenantWhere } = require('../utils/scope');
const {
  Project,
  Tenant,
  User,
  ProjectTagBinding,
  ProjectTag,
  ProjectGroupMember,
  ProjectGroup,
  NodeDeployment,
} = require('../models');

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const DEFAULT_SORT_BY = 'createdAt';
const DEFAULT_SORT_ORDER = 'DESC';
const SUPPORTED_SORT_FIELDS = new Set(['createdAt', 'updatedAt', 'lastDeployedAt', 'runtimeStatus']);
const SUPPORTED_SORT_DIRECTIONS = new Set(['ASC', 'DESC']);
const SUPPORTED_DEPLOY_STATUS = new Set(['pending', 'deploying', 'running', 'stopped', 'error', 'rollback']);
const SUPPORTED_RUNTIME_MODE = new Set(['DEV', 'RELEASE']);
const RUNTIME_STATUS_PRIORITY = {
  RUNNING: 50,
  DEPLOYING: 40,
  ERROR: 30,
  STOPPED: 20,
  NOT_DEPLOYED: 10,
  UNKNOWN: 0,
};

const trimText = (value) => String(value ?? '').trim();

const normalizeArrayQuery = (value) => {
  if (Array.isArray(value)) {
    return value
      .flatMap((item) => String(item ?? '').split(','))
      .map((item) => item.trim())
      .filter(Boolean);
  }
  return String(value ?? '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
};

const isUuid = (value) => UUID_PATTERN.test(trimText(value));
const normalizeUuid = (value) => (isUuid(value) ? trimText(value).toLowerCase() : '');

const toSafeDateValue = (value) => {
  if (!value) return null;
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return null;
  return parsed;
};

const resolvePaginationNumber = (value, fallback, min = 1, max = Number.MAX_SAFE_INTEGER) => {
  const parsed = Number.parseInt(value, 10);
  if (!Number.isInteger(parsed)) return fallback;
  if (parsed < min) return min;
  if (parsed > max) return max;
  return parsed;
};

const pickUserName = (user) => {
  if (!user || typeof user !== 'object') return null;
  const fullName = trimText(user.fullName);
  if (fullName) return fullName;
  const username = trimText(user.username);
  return username || null;
};

const resolveSortField = (query = {}) => {
  const candidate = trimText(query.sortBy || query.sortField || DEFAULT_SORT_BY);
  return SUPPORTED_SORT_FIELDS.has(candidate) ? candidate : DEFAULT_SORT_BY;
};

const resolveSortOrder = (query = {}) => {
  const candidate = trimText(query.sortOrder || query.order || DEFAULT_SORT_ORDER).toUpperCase();
  return SUPPORTED_SORT_DIRECTIONS.has(candidate) ? candidate : DEFAULT_SORT_ORDER;
};

const resolveDeployStatuses = (query = {}) => {
  return normalizeArrayQuery(query.deployStatus).map((item) => item.toLowerCase()).filter((item) => SUPPORTED_DEPLOY_STATUS.has(item));
};

const resolveRuntimeModes = (query = {}) => {
  return normalizeArrayQuery(query.runtimeMode).map((item) => item.toUpperCase()).filter((item) => SUPPORTED_RUNTIME_MODE.has(item));
};

const resolveNormalizedQuery = (query = {}) => {
  const page = resolvePaginationNumber(
    query.page,
    appConfig.pagination.defaultPage,
    1,
  );
  const limit = resolvePaginationNumber(
    query.limit,
    appConfig.pagination.defaultLimit,
    1,
    appConfig.pagination.maxLimit,
  );

  return {
    page,
    limit,
    name: trimText(query.name),
    groupQuery: trimText(query.group || query.groupId),
    tagQueries: normalizeArrayQuery(query.tag || query.tagId || query.tags),
    runtimeModes: resolveRuntimeModes(query),
    deployStatuses: resolveDeployStatuses(query),
    createdByQuery: trimText(query.createdBy || query.createdByName),
    sortBy: resolveSortField(query),
    sortOrder: resolveSortOrder(query),
  };
};

/**
 * 汇总工程的部署记录，统一生成列表侧边栏可复用的运行态摘要。
 * 这里会显式输出 runtimeStatus 与 lastDeployedAt，供筛选/排序与卡片展示共用。
 */
const buildRuntimeSummary = (deployments = []) => {
  const normalized = Array.isArray(deployments) ? deployments : [];
  const statusCounts = {
    pending: 0,
    deploying: 0,
    running: 0,
    stopped: 0,
    error: 0,
    rollback: 0,
  };
  const modeCounts = { DEV: 0, RELEASE: 0 };
  let latest = null;

  normalized.forEach((deployment) => {
    const status = trimText(deployment?.status).toLowerCase();
    const mode = trimText(deployment?.mode).toUpperCase();
    const timeline = toSafeDateValue(deployment?.deployedAt || deployment?.updatedAt || deployment?.createdAt);

    if (Object.prototype.hasOwnProperty.call(statusCounts, status)) {
      statusCounts[status] += 1;
    }
    if (Object.prototype.hasOwnProperty.call(modeCounts, mode)) {
      modeCounts[mode] += 1;
    }
    if (timeline && (!latest || timeline > latest)) {
      latest = timeline;
    }
  });

  let runtimeStatus = 'NOT_DEPLOYED';
  const deploymentCount = normalized.length;
  if (!deploymentCount) {
    runtimeStatus = 'NOT_DEPLOYED';
  } else if (statusCounts.running > 0) {
    runtimeStatus = 'RUNNING';
  } else if (statusCounts.deploying > 0 || statusCounts.pending > 0) {
    runtimeStatus = 'DEPLOYING';
  } else if (statusCounts.error > 0) {
    runtimeStatus = 'ERROR';
  } else if (statusCounts.stopped > 0 || statusCounts.rollback > 0) {
    runtimeStatus = 'STOPPED';
  } else {
    runtimeStatus = 'UNKNOWN';
  }

  return {
    runtimeStatus,
    deploymentCount,
    runningCount: statusCounts.running,
    statusCounts,
    modeCounts,
    lastDeployedAt: latest ? latest.toISOString() : null,
  };
};

/**
 * 统一把 Project 模型序列化为工程总览结构。
 * 注意：必须调用 toOverviewPayload()，避免再手工拼装一套 tags/group 字段。
 */
const toProjectOverview = (project) => {
  if (!project || typeof project.toOverviewPayload !== 'function') {
    throw new TypeError('Project.toOverviewPayload() 不可用，无法生成工程总览');
  }

  const payload = project.toOverviewPayload();
  const runtimeSummary = buildRuntimeSummary(payload.nodeDeployments);
  const createdByName = pickUserName(payload.creator);
  const updatedByName = pickUserName(payload.updater);
  const { nodeDeployments: _ignoredNodeDeployments, ...rest } = payload;

  return {
    ...rest,
    runtimeSummary,
    lastDeployedAt: runtimeSummary.lastDeployedAt,
    createdByName,
    updatedByName,
  };
};

const isMatchByKeyword = (value, keyword) => {
  const normalizedValue = trimText(value).toLowerCase();
  const normalizedKeyword = trimText(keyword).toLowerCase();
  if (!normalizedKeyword) return true;
  if (!normalizedValue) return false;
  return normalizedValue.includes(normalizedKeyword);
};

const matchGroup = (project, groupQuery) => {
  if (!groupQuery) return true;
  const group = project.group;
  if (!group) return false;
  if (isUuid(groupQuery)) {
    return normalizeUuid(group.id) === normalizeUuid(groupQuery);
  }
  return isMatchByKeyword(group.name, groupQuery);
};

const matchTags = (project, tagQueries = []) => {
  if (!tagQueries.length) return true;
  const tags = Array.isArray(project.tags) ? project.tags : [];
  if (!tags.length) return false;

  return tagQueries.some((query) => {
    if (isUuid(query)) {
      const normalizedQueryId = normalizeUuid(query);
      return tags.some((tag) => normalizeUuid(tag?.id) === normalizedQueryId);
    }
    return tags.some((tag) => isMatchByKeyword(tag?.name, query));
  });
};

const matchRuntimeMode = (project, runtimeModes = []) => {
  if (!runtimeModes.length) return true;
  const modeCounts = project.runtimeSummary?.modeCounts || {};
  return runtimeModes.some((mode) => Number(modeCounts[mode] || 0) > 0);
};

const matchDeployStatus = (project, deployStatuses = []) => {
  if (!deployStatuses.length) return true;
  const statusCounts = project.runtimeSummary?.statusCounts || {};
  return deployStatuses.some((status) => Number(statusCounts[status] || 0) > 0);
};

const matchCreator = (project, createdByQuery) => {
  if (!createdByQuery) return true;
  if (isUuid(createdByQuery)) {
    return normalizeUuid(project.createdBy) === normalizeUuid(createdByQuery);
  }
  return (
    isMatchByKeyword(project.createdByName, createdByQuery)
    || isMatchByKeyword(project.creator?.username, createdByQuery)
  );
};

const compareDateField = (left, right, field, sortOrder) => {
  const leftDate = toSafeDateValue(left[field]);
  const rightDate = toSafeDateValue(right[field]);
  const direction = sortOrder === 'ASC' ? 1 : -1;

  if (!leftDate && !rightDate) return 0;
  if (!leftDate) return 1;
  if (!rightDate) return -1;
  if (leftDate.getTime() === rightDate.getTime()) return 0;

  return leftDate.getTime() > rightDate.getTime() ? direction : -direction;
};

const compareRuntimeStatus = (left, right, sortOrder) => {
  const leftStatus = trimText(left.runtimeSummary?.runtimeStatus).toUpperCase() || 'UNKNOWN';
  const rightStatus = trimText(right.runtimeSummary?.runtimeStatus).toUpperCase() || 'UNKNOWN';
  const leftPriority = RUNTIME_STATUS_PRIORITY[leftStatus] ?? RUNTIME_STATUS_PRIORITY.UNKNOWN;
  const rightPriority = RUNTIME_STATUS_PRIORITY[rightStatus] ?? RUNTIME_STATUS_PRIORITY.UNKNOWN;

  if (leftPriority === rightPriority) {
    return 0;
  }

  if (sortOrder === 'ASC') {
    return leftPriority - rightPriority;
  }
  return rightPriority - leftPriority;
};

const sortProjectOverviews = (items = [], sortBy = DEFAULT_SORT_BY, sortOrder = DEFAULT_SORT_ORDER) => {
  const list = Array.isArray(items) ? [...items] : [];
  return list.sort((left, right) => {
    let result = 0;

    if (sortBy === 'runtimeStatus') {
      result = compareRuntimeStatus(left, right, sortOrder);
    } else if (sortBy === 'lastDeployedAt') {
      result = compareDateField(left, right, 'lastDeployedAt', sortOrder);
    } else {
      result = compareDateField(left, right, sortBy, sortOrder);
    }

    if (result !== 0) return result;

    // 主排序值相同后，使用更新时间兜底，确保列表在分页下结果稳定。
    const fallbackResult = compareDateField(left, right, 'updatedAt', 'DESC');
    if (fallbackResult !== 0) return fallbackResult;

    return String(left.id || '').localeCompare(String(right.id || ''));
  });
};

const buildProjectInclude = () => ([
  { model: Tenant, as: 'tenant', required: false },
  { model: User, as: 'creator', required: false, attributes: ['id', 'username', 'fullName'] },
  { model: User, as: 'updater', required: false, attributes: ['id', 'username', 'fullName'] },
  {
    model: ProjectTagBinding,
    as: 'tagBindings',
    required: false,
    attributes: ['id', 'tagId', 'projectId', 'tenantId', 'createdAt'],
    include: [
      {
        model: ProjectTag,
        as: 'tag',
        required: false,
        attributes: ['id', 'name', 'description', 'sortOrder'],
      },
    ],
  },
  {
    model: ProjectGroupMember,
    as: 'groupMember',
    required: false,
    attributes: ['id', 'groupId', 'projectId', 'tenantId', 'createdAt'],
    include: [
      {
        model: ProjectGroup,
        as: 'group',
        required: false,
        attributes: ['id', 'name', 'description', 'sortOrder'],
      },
    ],
  },
  {
    model: NodeDeployment,
    as: 'nodeDeployments',
    required: false,
    attributes: ['id', 'projectId', 'status', 'mode', 'deployedAt', 'updatedAt', 'createdAt'],
  },
]);

const applyOverviewFilters = (projects = [], normalizedQuery = {}) => {
  return (Array.isArray(projects) ? projects : []).filter((project) => {
    return (
      matchGroup(project, normalizedQuery.groupQuery)
      && matchTags(project, normalizedQuery.tagQueries)
      && matchRuntimeMode(project, normalizedQuery.runtimeModes)
      && matchDeployStatus(project, normalizedQuery.deployStatuses)
      && matchCreator(project, normalizedQuery.createdByQuery)
    );
  });
};

async function listProjectOverviews({ req, query = {} } = {}) {
  const normalizedQuery = resolveNormalizedQuery(query);
  const where = buildTenantWhere({}, req || {});

  if (normalizedQuery.name) {
    where.name = { [Op.like]: `%${normalizedQuery.name}%` };
  }
  if (normalizedQuery.createdByQuery && isUuid(normalizedQuery.createdByQuery)) {
    where.createdBy = normalizeUuid(normalizedQuery.createdByQuery);
  }

  const projects = await Project.findAll({
    where,
    include: buildProjectInclude(),
  });

  const overviews = projects.map((project) => toProjectOverview(project));
  const filtered = applyOverviewFilters(overviews, normalizedQuery);
  const sorted = sortProjectOverviews(filtered, normalizedQuery.sortBy, normalizedQuery.sortOrder);

  const total = sorted.length;
  const totalPages = total === 0 ? 0 : Math.ceil(total / normalizedQuery.limit);
  const startIndex = (normalizedQuery.page - 1) * normalizedQuery.limit;
  const pagedProjects = sorted.slice(startIndex, startIndex + normalizedQuery.limit);

  return {
    projects: pagedProjects,
    total,
    page: normalizedQuery.page,
    limit: normalizedQuery.limit,
    totalPages,
  };
}

module.exports = {
  buildRuntimeSummary,
  sortProjectOverviews,
  listProjectOverviews,
};
