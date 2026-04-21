const bcrypt = require("bcryptjs");
const {
  ProjectRole,
  ProjectRuntimeUser,
  ProjectUserRoleBinding,
} = require("../models");

const DEFAULT_RUNTIME_ADMIN_ROLE_CODE = "PROJECT_RUNTIME_ADMIN";
const DEFAULT_RUNTIME_ADMIN_USERNAME = "runtime_admin";
const DEFAULT_RUNTIME_ADMIN_NAME = "运行态管理员";
const BCRYPT_SALT_ROUNDS = 12;

const normalizeGrantCollection = (grants = []) => (Array.isArray(grants) ? grants : []);
const normalizeText = (value) => String(value ?? "").trim();

const buildMapEntry = () => ({
  localAllowRoles: new Set(),
  localDenyRoles: new Set(),
});

const isSameValue = (left, right) => normalizeText(left) === normalizeText(right);

const extractActorId = (context = {}) =>
  normalizeText(
    context.creator?.id || context.createdBy || context.project?.createdBy || context.project?.creator?.id,
  );

const buildBootstrapUsername = (context = {}) =>
  normalizeText(context.creator?.username || context.username) || DEFAULT_RUNTIME_ADMIN_USERNAME;

const buildBootstrapDisplayName = (context = {}) => {
  const creator = context.creator || {};
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
  const username = buildBootstrapUsername(context);
  const roleName = normalizeText(context.roleName) || (
    normalizeText(project.name)
      ? `${normalizeText(project.name)}运行态管理员`
      : DEFAULT_RUNTIME_ADMIN_NAME
  );
  const roleDescription =
    normalizeText(context.roleDescription) ||
    "工程运行态默认管理员角色，用于初始化首个可管理账号";
  const displayName = buildBootstrapDisplayName(context);
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
};
