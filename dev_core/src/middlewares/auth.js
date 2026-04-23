const { User, Tenant, Project } = require('../models');
const ApiResponse = require('../utils/response');
const ErrorCodes = require('../constants/errorCodes');
const AppError = require('../utils/AppError');
const TokenManager = require('../utils/token');

const ROLE_CAPABILITIES = {
  SUPER_ADMIN: ['tenant:manage'],
  SYSTEM_ADMIN: ['*'],
  PROJECT_ADMIN: ['project:read', 'project:write', 'release:publish', 'deploy:execute', 'runtime:operate', 'node:read'],
  OPS_ADMIN: ['project:read', 'release:publish', 'deploy:execute', 'runtime:operate', 'node:read', 'node:approve'],
  USER_ADMIN: ['user:write'],
  USER: ['project:read'],
};

// 验证JWT token中间件
const authenticateToken = async (req, res, next) => {
  try {
    // 优先从 Authorization header 获取 token，如果没有则从 cookie 获取
    let token = null;
    const authHeader = req.headers.authorization;
    if (authHeader) {
      token = authHeader.split(' ')[1]; // Bearer TOKEN
    }
    
    // 如果 header 中没有 token，尝试从 cookie 获取
    if (!token && req.cookies && req.cookies.accessToken) {
      token = req.cookies.accessToken;
    }
    
    if(process.env.NODE_ENV === 'development' && !token){
      req.user = {
        id: 86,
        username: 'admin',
        role: 'SYSTEM_ADMIN',
        tenantId: '550e8400-e29b-41d4-a716-446655440000',
        tenant: {
          id: '550e8400-e29b-41d4-a716-446655440000',
          name: '默认租户',
          code: 'default',
          description: '系统默认租户',
          status: 'active',
          contactEmail: 'admin@example.com',
          maxUsers: 100,
          maxProjects: 50,
        },
      };
      return next();
    }
    if (!token) {
      res.locals.language = req.language || 'zh-CN';
      res.locals.requestId = req.requestId;
      return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_REQUIRED, {}, 200);
    }

    // 检查是否在黑名单中
    const isBlacklisted = await TokenManager.isAccessTokenBlacklisted(token);
    if (isBlacklisted) {
      res.locals.language = req.language || 'zh-CN';
      res.locals.requestId = req.requestId;
      return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_INVALID, {}, 200);
    }

    // 验证 Access Token
    const decoded = TokenManager.verifyAccessToken(token);
    if (!decoded) {
      res.locals.language = req.language || 'zh-CN';
      res.locals.requestId = req.requestId;
      return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_INVALID, {}, 200);
    }

    // 获取用户信息
    const user = await User.findByPk(decoded.userId, {
      include: [{ model: Tenant, as: 'tenant' }]
    });

    if (!user || user.status !== 'active') {
      res.locals.language = req.language || 'zh-CN';
      res.locals.requestId = req.requestId;
      return ApiResponse.error(res, ErrorCodes.AUTH_USER_NOT_FOUND, {}, 200);
    }

    // 检查租户状态
    if (user.tenant && user.tenant.status !== 'active') {
      res.locals.language = req.language || 'zh-CN';
      res.locals.requestId = req.requestId;
      return ApiResponse.error(res, ErrorCodes.AUTH_TENANT_INACTIVE, {}, 200);
    }

    req.user = {
      id: user.id,
      username: user.username,
      role: user.role,
      tenantId: user.tenantId,
      tenant: user.tenant,
    };

    next();
  } catch (error) {
    res.locals.language = req.language || 'zh-CN';
    res.locals.requestId = req.requestId;
    
    if (error.name === 'JsonWebTokenError') {
      return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_INVALID, {}, 200);
    }
    if (error.name === 'TokenExpiredError') {
      return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_EXPIRED, {}, 200);
    }
    console.error('Auth middleware error:', error);
    return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
  }
};

// 角色权限检查中间件
const requireRole = (...allowedRoles) => {
  return (req, res, next) => {
    if (!req.user) {
      res.locals.language = req.language || 'zh-CN';
      res.locals.requestId = req.requestId;
      return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_REQUIRED, {}, 200);
    }

    if (!allowedRoles.includes(req.user.role)) {
      res.locals.language = req.language || 'zh-CN';
      res.locals.requestId = req.requestId;
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
    }

    next();
  };
};

const hasCapability = (userRole, capability) => {
  if (!userRole || !capability) return false;
  const caps = ROLE_CAPABILITIES[userRole] || [];
  return caps.includes('*') || caps.includes(capability);
};

const requireCapability = (...capabilities) => {
  return (req, res, next) => {
    if (!req.user) {
      res.locals.language = req.language || 'zh-CN';
      res.locals.requestId = req.requestId;
      return ApiResponse.error(res, ErrorCodes.AUTH_TOKEN_REQUIRED, {}, 200);
    }
    const canPass = capabilities.some((cap) => hasCapability(req.user.role, cap));
    if (!canPass) {
      res.locals.language = req.language || 'zh-CN';
      res.locals.requestId = req.requestId;
      return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
    }
    next();
  };
};

// 租户权限检查中间件（确保用户只能访问自己租户的数据）
const requireTenantAccess = (req, res, next) => {
  const { tenantId } = req.params;
  const userTenantId = req.user.tenantId;

  // 超级管理员可以访问所有租户
  if (req.user.role === 'SUPER_ADMIN') {
    return next();
  }

  if (tenantId && tenantId !== userTenantId) {
    res.locals.language = req.language || 'zh-CN';
    res.locals.requestId = req.requestId;
    return ApiResponse.error(res, ErrorCodes.PERMISSION_TENANT_MISMATCH, {}, 200);
  }

  next();
};

// 资源所有权检查中间件
const requireResourceOwnership = (resourceType) => {
  return async (req, res, next) => {
    try {
      const { id } = req.params;

      if (!id) {
        return next();
      }

      let resource;
      switch (resourceType) {
        case 'project':
          const { Project } = require('../models');
          resource = await Project.findByPk(id);
          break;
        case 'user':
          resource = await User.findByPk(id);
          break;
        default:
          res.locals.language = req.language || 'zh-CN';
          res.locals.requestId = req.requestId;
          return ApiResponse.error(res, ErrorCodes.RESOURCE_INVALID_TYPE, {}, 200);
      }

      if (!resource) {
        res.locals.language = req.language || 'zh-CN';
        res.locals.requestId = req.requestId;
        return ApiResponse.error(res, ErrorCodes.RESOURCE_NOT_FOUND, {}, 200);
      }

      // 检查资源是否属于用户的租户
      if (resource.tenantId !== req.user.tenantId) {
        res.locals.language = req.language || 'zh-CN';
        res.locals.requestId = req.requestId;
        return ApiResponse.error(res, ErrorCodes.PERMISSION_RESOURCE_OWNERSHIP, {}, 200);
      }

      // 检查用户是否有权限操作此资源
      const hasPermission = checkResourcePermission(req.user.role, resourceType, req.method);
      if (!hasPermission) {
        res.locals.language = req.language || 'zh-CN';
        res.locals.requestId = req.requestId;
        return ApiResponse.error(res, ErrorCodes.PERMISSION_INSUFFICIENT, {}, 200);
      }

      req.resource = resource;
      next();
    } catch (error) {
      console.error('Resource ownership check error:', error);
      res.locals.language = req.language || 'zh-CN';
      res.locals.requestId = req.requestId;
      return ApiResponse.error(res, ErrorCodes.INTERNAL_SERVER_ERROR, {}, 500);
    }
  };
};

// 检查用户对特定资源的权限
function checkResourcePermission(userRole, resourceType, method) {
  const permissions = {
    SYSTEM_ADMIN: {
      project: ['GET', 'POST', 'PUT', 'DELETE'],
      user: ['GET', 'POST', 'PUT', 'DELETE'],
    },
    PROJECT_ADMIN: {
      project: ['GET', 'POST', 'PUT', 'DELETE'],
      user: ['GET'],
    },
    OPS_ADMIN: {
      project: ['GET', 'PUT'], // 可以查看和更新工程状态（运维）
      user: ['GET'],
    },
    USER_ADMIN: {
      project: ['GET'],
      user: ['GET', 'POST', 'PUT', 'DELETE'],
    },
  };

  const rolePermissions = permissions[userRole];
  if (!rolePermissions) {
    return false;
  }

  const resourcePermissions = rolePermissions[resourceType];
  if (!resourcePermissions) {
    return false;
  }

  return resourcePermissions.includes(method);
}

/**
 * 检查用户是否有权限访问指定工程
 * @param {Object} req - Express request
 * @param {string} projectId - 工程ID
 * @throws {AppError} 如果无权访问
 */
async function checkProjectAccess(req, projectId) {
  if (!req.user) {
    throw new AppError(ErrorCodes.AUTH_TOKEN_REQUIRED, 401);
  }

  // 系统管理员拥有所有权限
  if (req.user.role === "SYSTEM_ADMIN") {
    return true;
  }

  // 其他角色：基于工程租户归属校验（兼容当前模型未定义 user.getProjects 的情况）。
  const project = await Project.findByPk(projectId, {
    attributes: ['id', 'tenantId'],
  });
  if (!project) {
    throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
      message: "工程不存在",
    });
  }

  if (project.tenantId === req.user.tenantId) {
    return true;
  }

  // 兼容旧模型：若存在用户-工程关联方法，则作为补充判定。
  const user = await User.findByPk(req.user.id);
  if (user && typeof user.getProjects === 'function') {
    const userProjects = await user.getProjects();
    const hasAccess = userProjects.some((p) => p.id === projectId);
    if (hasAccess) {
      return true;
    }
  }

  throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
    message: "无权访问此工程",
  });
}

module.exports = {
  authenticateToken,
  authenticate: authenticateToken, // 别名，用于部分旧路由
  requireRole,
  requireCapability,
  hasCapability,
  requireTenantAccess,
  requireResourceOwnership,
  checkProjectAccess,
};
