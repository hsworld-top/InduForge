const bcrypt = require("bcryptjs");
const { Op } = require("sequelize");
const AppError = require("../utils/AppError");
const ErrorCodes = require("../constants/errorCodes");
const {
  Project,
  ProjectRole,
  ProjectRuntimeUser,
  ProjectUserRoleBinding,
  ProjectRoleGrant,
} = require("../models");

const DEFAULT_RUNTIME_ADMIN_ROLE_CODE = "PROJECT_RUNTIME_ADMIN";
const DEFAULT_RUNTIME_ADMIN_USERNAME = "runtime_admin";
const DEFAULT_RUNTIME_ADMIN_NAME = "运行态管理员";
const BCRYPT_SALT_ROUNDS = 12;

const normalizeGrantCollection = (grants = []) => (Array.isArray(grants) ? grants : []);
const normalizeText = (value) => String(value ?? "").trim();
const isUniqueConstraintError = (error) => error?.name === "SequelizeUniqueConstraintError";

const buildMapEntry = () => ({
  localAllowRoles: new Set(),
  localDenyRoles: new Set(),
});

const isSameValue = (left, right) => normalizeText(left) === normalizeText(right);

const extractActorId = (context = {}) =>
  normalizeText(
    context.creator?.id || context.createdBy || context.project?.createdBy || context.project?.creator?.id,
  );

const buildBootstrapUsername = (creator = {}, context = {}) =>
  normalizeText(creator.username || context.username) || DEFAULT_RUNTIME_ADMIN_USERNAME;

const buildBootstrapDisplayName = (creator = {}, context = {}) => {
  return (
    normalizeText(context.displayName) ||
    normalizeText(creator.fullName) ||
    normalizeText(creator.username) ||
    DEFAULT_RUNTIME_ADMIN_NAME
  );
};

const assertBootstrapOwnerOrThrow = (existingUser, actorId, username) => {
  if (!existingUser) {
    return;
  }

  const sameOwner = actorId && isSameValue(existingUser.createdBy, actorId);

  if (!sameOwner) {
    throw new Error(`运行态账号 ${username} 已存在且不属于当前工程创建者，拒绝自动提权`);
  }
};

const findExistingBootstrapRuntimeUser = async ({
  projectId,
  actorId,
  username,
  transaction,
}) => {
  if (actorId) {
    const ownerRuntimeUser = await ProjectRuntimeUser.findOne({
      where: {
        projectId,
        createdBy: actorId,
      },
      transaction,
    });

    if (ownerRuntimeUser) {
      return ownerRuntimeUser;
    }
  }

  return ProjectRuntimeUser.findOne({
    where: {
      projectId,
      username,
    },
    transaction,
  });
};

const sanitizeRuntimeUser = (runtimeUser) => {
  if (!runtimeUser) {
    return null;
  }

  const plainUser = typeof runtimeUser.toJSON === "function" ? runtimeUser.toJSON() : { ...runtimeUser };
  delete plainUser.passwordHash;
  return plainUser;
};

const normalizeRoleIds = (roleIds = []) =>
  [...new Set((Array.isArray(roleIds) ? roleIds : []).map((item) => normalizeText(item)).filter(Boolean))];

const normalizeRuntimeUserStatusForOutput = (status) => {
  const normalizedStatus = normalizeText(status);
  return normalizedStatus === "active" ? "active" : "disabled";
};

const normalizeRuntimeUserStatusForStorage = (status) => {
  const normalizedStatus = normalizeText(status);
  if (normalizedStatus === "active") {
    return "active";
  }
  if (normalizedStatus === "disabled") {
    return "inactive";
  }
  throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, { message: "状态值不合法" });
};

const normalizeRuntimeRoleStatusForOutput = (status) => {
  const normalizedStatus = normalizeText(status);
  return normalizedStatus === "active" ? "active" : "disabled";
};

const normalizeRuntimeRoleStatusForStorage = (status) => {
  const normalizedStatus = normalizeText(status);
  if (!normalizedStatus) {
    return "";
  }
  if (normalizedStatus === "active") {
    return "active";
  }
  if (normalizedStatus === "disabled") {
    return "inactive";
  }
  throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, { message: "状态值不合法" });
};

const buildRuntimeUserInclude = () => ([
  {
    model: ProjectUserRoleBinding,
    as: "roleBindings",
    required: false,
    include: [
      {
        model: ProjectRole,
        as: "role",
        required: false,
      },
    ],
  },
]);

const buildRuntimeRoleInclude = () => ([
  {
    model: ProjectUserRoleBinding,
    as: "userBindings",
    required: false,
  },
  {
    model: ProjectRoleGrant,
    as: "grants",
    required: false,
  },
]);

const buildRuntimeUserSummary = (runtimeUser) => {
  const plainUser = sanitizeRuntimeUser(runtimeUser);
  if (!plainUser) {
    return null;
  }

  const roleBindings = Array.isArray(plainUser.roleBindings) ? plainUser.roleBindings : [];
  const roles = roleBindings
    .map((binding) => binding.role || null)
    .filter(Boolean)
    .map((role) => ({
      id: role.id,
      code: role.code,
      name: role.name,
      description: role.description || null,
      isSystem: Boolean(role.isSystem),
      status: normalizeRuntimeRoleStatusForOutput(role.status),
    }));

  plainUser.status = normalizeRuntimeUserStatusForOutput(plainUser.status);
  plainUser.roleIds = roleBindings.map((binding) => binding.roleId).filter(Boolean);
  plainUser.roles = roles;
  delete plainUser.roleBindings;
  return plainUser;
};

const buildRuntimeRoleSummary = (role) => {
  if (!role) {
    return null;
  }

  const plainRole = typeof role.toJSON === "function" ? role.toJSON() : { ...role };
  const bindings = Array.isArray(plainRole.userBindings) ? plainRole.userBindings : [];
  const grants = Array.isArray(plainRole.grants) ? plainRole.grants : [];
  const bindingCount = new Set(bindings.map((binding) => binding.runtimeUserId).filter(Boolean)).size;

  delete plainRole.userBindings;
  delete plainRole.grants;

  return {
    ...plainRole,
    status: normalizeRuntimeRoleStatusForOutput(plainRole.status),
    bindingCount,
    grantCount: grants.length,
  };
};

const loadRuntimeUserSummary = async ({
  projectId,
  runtimeUserId,
  transaction = null,
} = {}) => {
  const runtimeUser = await ProjectRuntimeUser.findOne({
    where: { projectId, id: runtimeUserId },
    include: buildRuntimeUserInclude(),
    transaction,
  });
  return buildRuntimeUserSummary(runtimeUser);
};

const loadRuntimeRoleSummary = async ({
  projectId,
  roleId,
  transaction = null,
} = {}) => {
  const role = await ProjectRole.findOne({
    where: { projectId, id: roleId },
    include: buildRuntimeRoleInclude(),
    transaction,
  });
  return buildRuntimeRoleSummary(role);
};

const ensureRoleSetBelongsToProject = async (projectId, roleIds, transaction = null) => {
  const uniqueRoleIds = normalizeRoleIds(roleIds);
  if (!uniqueRoleIds.length) {
    return [];
  }

  const roles = await ProjectRole.findAll({
    where: {
      projectId,
      id: { [Op.in]: uniqueRoleIds },
    },
    attributes: ["id"],
    transaction,
  });

  if (roles.length !== uniqueRoleIds.length) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
      message: "角色不属于当前工程或不存在",
    });
  }

  return uniqueRoleIds;
};

const translateUniqueConstraintError = (error, message) => {
  if (isUniqueConstraintError(error)) {
    throw new AppError(ErrorCodes.RESOURCE_ALREADY_EXISTS, 409, { message });
  }
  throw error;
};

const assertAnotherActiveRuntimeAdminExists = async ({
  projectId,
  runtimeUserId,
  transaction = null,
} = {}) => {
  const runtimeAdminRole = await ProjectRole.findOne({
    where: {
      projectId,
      code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
    },
    transaction,
  });
  if (!runtimeAdminRole) {
    return;
  }

  const currentBindings = await ProjectUserRoleBinding.findAll({
    where: {
      projectId,
      runtimeUserId,
    },
    attributes: ["roleId"],
    transaction,
  });
  const hasRuntimeAdminRole = currentBindings.some((binding) => binding.roleId === runtimeAdminRole.id);
  if (!hasRuntimeAdminRole) {
    return;
  }

  const adminBindings = await ProjectUserRoleBinding.findAll({
    where: {
      projectId,
      roleId: runtimeAdminRole.id,
    },
    attributes: ["runtimeUserId"],
    transaction,
  });
  const otherAdminIds = [
    ...new Set(
      adminBindings
        .map((binding) => normalizeText(binding.runtimeUserId))
        .filter((bindingRuntimeUserId) => bindingRuntimeUserId && bindingRuntimeUserId !== runtimeUserId),
    ),
  ];
  if (!otherAdminIds.length) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
      message: "至少保留一个启用中的运行态管理员",
    });
  }

  const activeAdmins = await ProjectRuntimeUser.findAll({
    where: {
      projectId,
      id: { [Op.in]: otherAdminIds },
      status: "active",
    },
    attributes: ["id", "status"],
    transaction,
  });
  if (!activeAdmins.length) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
      message: "至少保留一个启用中的运行态管理员",
    });
  }
};

/**
 * 创建或补齐工程运行态的默认管理员角色、用户和绑定关系。
 * 设计上保持幂等：已有记录则直接复用，不重复创建。
 */
async function ensureRuntimeAdminBootstrap(input = {}, legacyOptions = {}) {
  const context =
    typeof input === "string"
      ? { project: { id: input }, ...(legacyOptions || {}) }
      : input || {};
  const project = context.project || {};
  const creator = context.creator || project.creator || {};
  const transaction = context.transaction;
  const resolvedProjectId = normalizeText(
    project.id || context.projectId || context.id,
  );
  if (!resolvedProjectId) {
    throw new Error("projectId 不能为空");
  }

  const initialPassword = normalizeText(context.initialPassword);
  if (!initialPassword) {
    throw new Error("initialPassword 不能为空");
  }

  const roleCode = normalizeText(context.roleCode) || DEFAULT_RUNTIME_ADMIN_ROLE_CODE;
  const username = buildBootstrapUsername(creator, context);
  const roleName = normalizeText(context.roleName) || (
    normalizeText(project.name)
      ? `${normalizeText(project.name)}运行态管理员`
      : DEFAULT_RUNTIME_ADMIN_NAME
  );
  const roleDescription =
    normalizeText(context.roleDescription) ||
    "工程运行态默认管理员角色，用于初始化首个可管理账号";
  const displayName = buildBootstrapDisplayName(creator, context);
  const actorId = extractActorId(context);
  const roleAttributes = {
    projectId: resolvedProjectId,
    code: roleCode,
    name: roleName,
    description: roleDescription,
    isSystem: true,
    status: "active",
    createdBy: actorId || null,
    updatedBy: actorId || null,
  };
  const loadOrCreateWithRetry = async (model, where, defaults) => {
    const existingRecord = await model.findOne({
      where,
      transaction,
    });

    if (existingRecord) {
      return existingRecord;
    }

    try {
      const [createdRecord] = await model.findOrCreate({
        where,
        defaults,
        transaction,
      });
      return createdRecord;
    } catch (error) {
      if (error?.name !== "SequelizeUniqueConstraintError") {
        throw error;
      }

      const reloadedRecord = await model.findOne({
        where,
        transaction,
      });
      if (!reloadedRecord) {
        throw error;
      }
      return reloadedRecord;
    }
  };

  const role = await loadOrCreateWithRetry(
    ProjectRole,
    {
      projectId: resolvedProjectId,
      code: roleCode,
    },
    roleAttributes,
  );

  const usernameWhere = {
    projectId: resolvedProjectId,
    username,
  };
  let runtimeUser = await findExistingBootstrapRuntimeUser({
    projectId: resolvedProjectId,
    actorId,
    username,
    transaction,
  });

  if (runtimeUser) {
    assertBootstrapOwnerOrThrow(runtimeUser, actorId, username);
  } else {
    const passwordHash = await bcrypt.hash(initialPassword, BCRYPT_SALT_ROUNDS);
    const runtimeUserAttributes = {
      projectId: resolvedProjectId,
      createdBy: actorId || null,
      updatedBy: actorId || null,
      username,
      passwordHash,
      displayName,
      status: "active",
    };

    runtimeUser = await loadOrCreateWithRetry(
      ProjectRuntimeUser,
      usernameWhere,
      runtimeUserAttributes,
    );
    assertBootstrapOwnerOrThrow(runtimeUser, actorId, username);
  }

  const bindingAttributes = {
    projectId: resolvedProjectId,
    createdBy: actorId || null,
    runtimeUserId: runtimeUser.id,
    roleId: role.id,
    assignedAt: new Date(),
  };
  const binding = await loadOrCreateWithRetry(
    ProjectUserRoleBinding,
    {
      projectId: resolvedProjectId,
      runtimeUserId: runtimeUser.id,
      roleId: role.id,
    },
    bindingAttributes,
  );

  return {
    role,
    runtimeUser: sanitizeRuntimeUser(runtimeUser),
    binding,
  };
}

async function listRuntimeUsers({ projectId } = {}) {
  const users = await ProjectRuntimeUser.findAll({
    where: { projectId },
    include: buildRuntimeUserInclude(),
    order: [["createdAt", "ASC"]],
  });

  return users.map((user) => buildRuntimeUserSummary(user));
}

async function createRuntimeUser({
  projectId,
  actorId = null,
  username,
  displayName,
  initialPassword,
  roleIds = [],
} = {}) {
  const normalizedUsername = normalizeText(username);
  if (!projectId) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, { message: "projectId 不能为空" });
  }
  if (!normalizedUsername) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, { message: "用户名不能为空" });
  }
  if (!normalizeText(initialPassword)) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, { message: "初始密码不能为空" });
  }

  return Project.sequelize.transaction(async (transaction) => {
    const uniqueRoleIds = await ensureRoleSetBelongsToProject(projectId, roleIds, transaction);
    const existingUser = await ProjectRuntimeUser.findOne({
      where: {
        projectId,
        username: normalizedUsername,
      },
      transaction,
    });

    if (existingUser) {
      throw new AppError(ErrorCodes.RESOURCE_ALREADY_EXISTS, 409, {
        message: "运行态用户名已存在",
      });
    }

    const passwordHash = await bcrypt.hash(initialPassword, BCRYPT_SALT_ROUNDS);
    let runtimeUser;
    try {
      runtimeUser = await ProjectRuntimeUser.create({
        projectId,
        createdBy: actorId || null,
        updatedBy: actorId || null,
        username: normalizedUsername,
        passwordHash,
        displayName: normalizeText(displayName) || normalizedUsername,
        status: "active",
      }, {
        transaction,
      });
    } catch (error) {
      translateUniqueConstraintError(error, "运行态用户名已存在");
    }

    for (const roleId of uniqueRoleIds) {
      await ProjectUserRoleBinding.create({
        projectId,
        createdBy: actorId || null,
        runtimeUserId: runtimeUser.id,
        roleId,
        assignedBy: actorId || null,
        assignedAt: new Date(),
      }, {
        transaction,
      });
    }

    return loadRuntimeUserSummary({
      projectId,
      runtimeUserId: runtimeUser.id,
      transaction,
    });
  });
}

async function updateRuntimeUserStatus({
  projectId,
  runtimeUserId,
  status,
  actorId = null,
} = {}) {
  const normalizedStatus = normalizeRuntimeUserStatusForStorage(status);
  if (!projectId || !runtimeUserId) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, { message: "参数不完整" });
  }

  const runtimeUser = await ProjectRuntimeUser.findOne({
    where: { projectId, id: runtimeUserId },
  });

  if (!runtimeUser) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: "运行态用户不存在",
    });
  }

  if (normalizedStatus === "inactive" && normalizeText(runtimeUser.status) === "active") {
    await assertAnotherActiveRuntimeAdminExists({ projectId, runtimeUserId });
  }

  await runtimeUser.update({
    status: normalizedStatus,
    updatedBy: actorId || null,
  });

  return loadRuntimeUserSummary({ projectId, runtimeUserId });
}

async function resetRuntimeUserPassword({
  projectId,
  runtimeUserId,
  newPassword,
  actorId = null,
} = {}) {
  if (!normalizeText(newPassword)) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, { message: "新密码不能为空" });
  }

  const runtimeUser = await ProjectRuntimeUser.findOne({
    where: { projectId, id: runtimeUserId },
  });
  if (!runtimeUser) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: "运行态用户不存在",
    });
  }

  await runtimeUser.update({
    passwordHash: await bcrypt.hash(newPassword, BCRYPT_SALT_ROUNDS),
    updatedBy: actorId || null,
  });

  return loadRuntimeUserSummary({ projectId, runtimeUserId });
}

async function bindRuntimeUserRoles({
  projectId,
  runtimeUserId,
  roleIds = [],
  actorId = null,
} = {}) {
  return Project.sequelize.transaction(async (transaction) => {
    const uniqueRoleIds = await ensureRoleSetBelongsToProject(projectId, roleIds, transaction);
    const runtimeAdminRole = await ProjectRole.findOne({
      where: {
        projectId,
        code: DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
      },
      transaction,
    });
    const runtimeUser = await ProjectRuntimeUser.findOne({
      where: { projectId, id: runtimeUserId },
      transaction,
    });

    if (!runtimeUser) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: "运行态用户不存在",
      });
    }

    if (
      runtimeAdminRole
      && normalizeText(runtimeUser.status) === "active"
      && !uniqueRoleIds.includes(runtimeAdminRole.id)
    ) {
      await assertAnotherActiveRuntimeAdminExists({
        projectId,
        runtimeUserId,
        transaction,
      });
    }

    await ProjectUserRoleBinding.destroy({
      where: {
        projectId,
        runtimeUserId,
      },
      transaction,
    });

    for (const roleId of uniqueRoleIds) {
      await ProjectUserRoleBinding.create({
        projectId,
        createdBy: actorId || null,
        runtimeUserId,
        roleId,
        assignedBy: actorId || null,
        assignedAt: new Date(),
      }, {
        transaction,
      });
    }

    return loadRuntimeUserSummary({
      projectId,
      runtimeUserId,
      transaction,
    });
  });
}

async function listRuntimeRoles({ projectId } = {}) {
  const roles = await ProjectRole.findAll({
    where: { projectId },
    include: buildRuntimeRoleInclude(),
    order: [["createdAt", "ASC"]],
  });

  return roles.map((role) => buildRuntimeRoleSummary(role));
}

async function createRuntimeRole({
  projectId,
  actorId = null,
  code,
  name,
  description,
} = {}) {
  const normalizedCode = normalizeText(code);
  const normalizedName = normalizeText(name);
  if (!projectId) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, { message: "projectId 不能为空" });
  }
  if (!normalizedCode || !normalizedName) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, { message: "角色编码和名称不能为空" });
  }

  const existingRole = await ProjectRole.findOne({
    where: {
      projectId,
      code: normalizedCode,
    },
  });
  if (existingRole) {
    throw new AppError(ErrorCodes.RESOURCE_ALREADY_EXISTS, 409, {
      message: "角色编码已存在",
    });
  }

  let role;
  try {
    role = await ProjectRole.create({
      projectId,
      createdBy: actorId || null,
      updatedBy: actorId || null,
      code: normalizedCode,
      name: normalizedName,
      description: normalizeText(description) || null,
      isSystem: false,
      status: "active",
    });
  } catch (error) {
    translateUniqueConstraintError(error, "角色编码已存在");
  }

  return buildRuntimeRoleSummary(role);
}

async function updateRuntimeRole({
  projectId,
  roleId,
  actorId = null,
  code,
  name,
  description,
  status,
} = {}) {
  const role = await ProjectRole.findOne({
    where: { projectId, id: roleId },
  });

  if (!role) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: "运行态角色不存在",
    });
  }

  if (role.isSystem || role.code === DEFAULT_RUNTIME_ADMIN_ROLE_CODE) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
      message: "系统内置角色不允许编辑",
    });
  }

  const nextCode = normalizeText(code);
  const nextName = normalizeText(name);
  const nextStatus = normalizeRuntimeRoleStatusForStorage(status);

  if (nextCode && nextCode !== role.code) {
    const duplicate = await ProjectRole.findOne({
      where: {
        projectId,
        code: nextCode,
        id: { [Op.ne]: roleId },
      },
    });
    if (duplicate) {
      throw new AppError(ErrorCodes.RESOURCE_ALREADY_EXISTS, 409, {
        message: "角色编码已存在",
      });
    }
  }

  await role.update({
    ...(nextCode ? { code: nextCode } : {}),
    ...(nextName ? { name: nextName } : {}),
    ...(typeof description !== "undefined" ? { description: normalizeText(description) || null } : {}),
    ...(nextStatus ? { status: nextStatus } : {}),
    updatedBy: actorId || null,
  });

  return loadRuntimeRoleSummary({ projectId, roleId });
}

async function deleteRuntimeRole({
  projectId,
  roleId,
  actorId = null,
} = {}) {
  const role = await ProjectRole.findOne({
    where: { projectId, id: roleId },
    include: buildRuntimeRoleInclude(),
  });

  if (!role) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: "运行态角色不存在",
    });
  }

  if (role.isSystem || role.code === DEFAULT_RUNTIME_ADMIN_ROLE_CODE) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
      message: "系统内置角色不能删除",
    });
  }

  const bindingCount = Array.isArray(role.userBindings) ? role.userBindings.length : 0;
  const grantCount = Array.isArray(role.grants) ? role.grants.length : 0;
  if (bindingCount > 0 || grantCount > 0) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
      message: "角色存在绑定或授权记录，无法删除",
      bindingCount,
      grantCount,
    });
  }

  await role.destroy();

  return {
    deleted: true,
    roleId,
    projectId,
    deletedBy: actorId || null,
  };
}

/**
 * 把多个角色的授权记录合并成易于权限判定的映射。
 * 规则很简单：
 * - 同一资源/动作下，allowRoles 记录允许的角色
 * - denyRoles 记录拒绝的角色
 * - 同一角色同时出现在 allow/deny 时，以 deny 为准，允许列表会被移除
 */
function buildEffectiveRoleGrantMap(roleGrants = []) {
  const grantMap = new Map();

  for (const grant of normalizeGrantCollection(roleGrants)) {
    const resourceType = normalizeText(grant?.resourceType || grant?.resource);
    const resourceId = normalizeText(
      grant?.resourceId || grant?.scopeConfig?.resourceId || grant?.scopeConfig?.id,
    ) || "*";
    const action = normalizeText(grant?.action);
    const roleCode = normalizeText(grant?.roleCode || grant?.role?.code);
    const effect = normalizeText(grant?.effect).toLowerCase() || "allow";

    if (!resourceType || !action || !roleCode) {
      continue;
    }

    if (!grantMap.has(resourceType)) {
      grantMap.set(resourceType, new Map());
    }

    const resourceBucket = grantMap.get(resourceType);
    if (!resourceBucket.has(resourceId)) {
      resourceBucket.set(resourceId, new Map());
    }

    const resourceInstanceBucket = resourceBucket.get(resourceId);
    if (!resourceInstanceBucket.has(action)) {
      resourceInstanceBucket.set(action, buildMapEntry());
    }

    const actionBucket = resourceInstanceBucket.get(action);
    if (effect === "deny") {
      actionBucket.localDenyRoles.add(roleCode);
      actionBucket.localAllowRoles.delete(roleCode);
      continue;
    }

    if (!actionBucket.localDenyRoles.has(roleCode)) {
      actionBucket.localAllowRoles.add(roleCode);
    }
  }

  const finalizeGrantBucket = (localBucket, inheritedBucket = null) => {
    const effectiveAllowRoles = new Set(
      inheritedBucket ? inheritedBucket.allowRoles : [],
    );
    const effectiveDenyRoles = new Set(
      inheritedBucket ? inheritedBucket.denyRoles : [],
    );

    for (const roleCode of localBucket.localAllowRoles) {
      if (!effectiveDenyRoles.has(roleCode)) {
        effectiveAllowRoles.add(roleCode);
      }
    }

    for (const roleCode of localBucket.localDenyRoles) {
      effectiveDenyRoles.add(roleCode);
      effectiveAllowRoles.delete(roleCode);
    }

    return {
      allowRoles: [...effectiveAllowRoles].sort(),
      denyRoles: [...effectiveDenyRoles].sort(),
      localAllowRoles: [...localBucket.localAllowRoles].sort(),
      localDenyRoles: [...localBucket.localDenyRoles].sort(),
    };
  };

  return Object.fromEntries(
    [...grantMap.entries()].map(([resourceType, resourceBuckets]) => {
      const wildcardActionBuckets = resourceBuckets.get("*") || new Map();
      return [
        resourceType,
        Object.fromEntries(
          [...resourceBuckets.entries()].map(([resourceId, actionBuckets]) => {
            const mergedActionBuckets = new Map();
            const actionNames = new Set([
              ...wildcardActionBuckets.keys(),
              ...actionBuckets.keys(),
            ]);

            for (const action of actionNames) {
              const wildcardBucket = wildcardActionBuckets.get(action) || buildMapEntry();
              const specificBucket = actionBuckets.get(action) || buildMapEntry();
              if (resourceId === "*") {
                mergedActionBuckets.set(action, finalizeGrantBucket(wildcardBucket));
                continue;
              }

              mergedActionBuckets.set(
                action,
                finalizeGrantBucket(specificBucket, finalizeGrantBucket(wildcardBucket)),
              );
            }

            return [
              resourceId,
              Object.fromEntries(mergedActionBuckets.entries()),
            ];
          }),
        ),
      ];
    }),
  );
}

module.exports = {
  DEFAULT_RUNTIME_ADMIN_ROLE_CODE,
  DEFAULT_RUNTIME_ADMIN_USERNAME,
  ensureRuntimeAdminBootstrap,
  buildEffectiveRoleGrantMap,
  listRuntimeUsers,
  createRuntimeUser,
  updateRuntimeUserStatus,
  resetRuntimeUserPassword,
  bindRuntimeUserRoles,
  listRuntimeRoles,
  createRuntimeRole,
  updateRuntimeRole,
  deleteRuntimeRole,
};
