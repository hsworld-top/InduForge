/**
 * Design Controller - 设计中心控制器
 * 处理设计页面的 HTTP 请求
 * Requirements: 7.1, 7.2, 7.3, 7.4, 7.5
 */
const designService = require("../services/designService");
const { Project } = require("../models");
const ApiResponse = require("../utils/response");
const AppError = require("../utils/AppError");
const ErrorCodes = require("../constants/errorCodes");

/**
 * 检查用户是否有权限访问指定工程
 * @param {Object} req - Express 请求对象
 * @param {string} projectId - 工程ID
 */
async function checkProjectAccess(req, projectId) {
  // 系统管理员可以访问所有工程
  if (req.user.role === "SYSTEM_ADMIN") {
    return;
  }

  // 检查工程是否属于用户的租户
  const project = await Project.findOne({
    where: { id: projectId, tenantId: req.user.tenantId },
  });

  if (!project) {
    throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
      message: "无权访问此工程",
    });
  }
}

/**
 * 获取项目的页面列表
 * GET /api/v1/design/projects/:projectId/pages
 * Requirements: 7.1
 */
async function getPages(req, res, next) {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const pages = await designService.getPages(projectId);

    return ApiResponse.success(res, pages);
  } catch (error) {
    return next(error);
  }
}

/**
 * 获取单个页面的完整 Schema
 * GET /api/v1/design/projects/:projectId/pages/:pageId
 * Requirements: 7.2
 */
async function getPage(req, res, next) {
  try {
    const { projectId, pageId } = req.params;
    await checkProjectAccess(req, projectId);

    // 先获取页面详情验证归属
    const pageDetail = await designService.getPageDetail(pageId);
    if (pageDetail.projectId !== projectId) {
      throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
        message: "页面不属于此工程",
      });
    }

    // 返回 schema 内容
    return ApiResponse.success(res, pageDetail.schemaContent);
  } catch (error) {
    return next(error);
  }
}

/**
 * 创建新页面
 * POST /api/v1/design/projects/:projectId/pages
 * Requirements: 7.3
 */
async function createPage(req, res, next) {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const { name, type, parentId, path, schemaContent } = req.body;

    if (!name || typeof name !== "string" || name.trim().length === 0) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "页面名称不能为空",
      });
    }

    const page = await designService.createPage(
      projectId,
      { name: name.trim(), type, parentId, path, schemaContent },
      req.user.id
    );

    return ApiResponse.success(res, page, null, {}, 201);
  } catch (error) {
    return next(error);
  }
}

/**
 * 更新页面 Schema
 * PUT /api/v1/design/projects/:projectId/pages/:pageId
 * Requirements: 7.4
 */
async function updatePage(req, res, next) {
  try {
    const { projectId, pageId } = req.params;
    await checkProjectAccess(req, projectId);

    const { schema } = req.body;

    if (!schema || typeof schema !== "object") {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "Schema 不能为空",
      });
    }

    // 先验证页面属于该工程
    const existingPage = await designService.getPageDetail(pageId);
    if (existingPage.projectId !== projectId) {
      throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
        message: "页面不属于此工程",
      });
    }

    await designService.updatePage(pageId, schema, req.user.id);

    return ApiResponse.success(res, null);
  } catch (error) {
    return next(error);
  }
}

/**
 * 删除页面
 * DELETE /api/v1/design/projects/:projectId/pages/:pageId
 * Requirements: 7.5
 */
async function deletePage(req, res, next) {
  try {
    const { projectId, pageId } = req.params;
    const { mode } = req.query;
    await checkProjectAccess(req, projectId);

    // 先验证页面属于该工程
    const existingPage = await designService.getPageDetail(pageId);
    if (existingPage.projectId !== projectId) {
      throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
        message: "页面不属于此工程",
      });
    }

    await designService.deletePage(pageId, mode);

    return ApiResponse.success(res, null);
  } catch (error) {
    return next(error);
  }
}

/**
 * 重命名页面
 * PATCH /api/v1/design/projects/:projectId/pages/:pageId/rename
 */
async function renamePage(req, res, next) {
  try {
    const { projectId, pageId } = req.params;
    await checkProjectAccess(req, projectId);

    const { name, path } = req.body;

    if (!name || typeof name !== "string" || name.trim().length === 0) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "页面名称不能为空",
      });
    }

    // 先验证页面属于该工程
    const existingPage = await designService.getPageDetail(pageId);
    if (existingPage.projectId !== projectId) {
      throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
        message: "页面不属于此工程",
      });
    }

    await designService.renamePage(pageId, name.trim(), req.user.id, path);

    return ApiResponse.success(res, null);
  } catch (error) {
    return next(error);
  }
}

/**
 * 移动页面
 * PATCH /api/v1/design/projects/:projectId/pages/:pageId/move
 */
async function movePage(req, res, next) {
  try {
    const { projectId, pageId } = req.params;
    await checkProjectAccess(req, projectId);

    const { parentId, sortOrder, path } = req.body;

    // 先验证页面属于该工程
    const existingPage = await designService.getPageDetail(pageId);
    if (existingPage.projectId !== projectId) {
      throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
        message: "页面不属于此工程",
      });
    }

    await designService.movePage(pageId, { parentId, sortOrder, path }, req.user.id);

    return ApiResponse.success(res, null);
  } catch (error) {
    return next(error);
  }
}

/**
 * 获取工程级别全局变量
 * GET /api/v1/design/projects/:projectId/variables
 */
async function getProjectVariables(req, res, next) {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const project = await Project.findByPk(projectId);
    if (!project) {
      throw new AppError(ErrorCodes.PROJECT_NOT_FOUND, 404, {
        resource: "Project",
        id: projectId,
      });
    }

    return ApiResponse.success(res, project.projectVariables || {});
  } catch (error) {
    return next(error);
  }
}

/**
 * 更新工程级别全局变量
 * PUT /api/v1/design/projects/:projectId/variables
 */
async function updateProjectVariables(req, res, next) {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const { variables } = req.body;
    if (
      !variables ||
      typeof variables !== "object" ||
      Array.isArray(variables)
    ) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "variables 必须是对象",
      });
    }

    const project = await Project.findByPk(projectId);
    if (!project) {
      throw new AppError(ErrorCodes.PROJECT_NOT_FOUND, 404, {
        resource: "Project",
        id: projectId,
      });
    }

    await project.update({
      projectVariables: variables,
      updatedBy: req.user.id,
    });

    return ApiResponse.success(res, project.projectVariables || {});
  } catch (error) {
    return next(error);
  }
}

/**
 * 更新项目入口配置
 * PUT /api/v1/projects/:projectId/entry
 */
async function updateEntryConfig(req, res, next) {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const entryConfig = req.body;
    if (
      !entryConfig ||
      typeof entryConfig !== "object" ||
      Array.isArray(entryConfig)
    ) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "入口配置必须是对象",
      });
    }

    const result = await designService.updateEntryConfig(
      projectId,
      entryConfig
    );

    return ApiResponse.success(res, result);
  } catch (error) {
    return next(error);
  }
}



/**
 * 获取工程级设置（全局变量/脚本）
 * GET /api/v1/design/projects/:projectId/settings
 */
async function getProjectSettings(req, res, next) {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const settings = await designService.getProjectSettings(projectId);

    return ApiResponse.success(res, settings);
  } catch (error) {
    return next(error);
  }
}

/**
 * 更新工程级设置（全局变量/脚本）
 * PUT /api/v1/design/projects/:projectId/settings
 */
async function updateProjectSettings(req, res, next) {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const settings = req.body;
    if (!settings || typeof settings !== "object" || Array.isArray(settings)) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "settings 必须是对象",
      });
    }

    const result = await designService.updateProjectSettings(
      projectId,
      settings,
      req.user.id
    );

    return ApiResponse.success(res, result);
  } catch (error) {
    return next(error);
  }
}

module.exports = {
  getPages,
  getPage,
  createPage,
  updatePage,
  deletePage,
  renamePage,
  movePage,
  getProjectVariables,
  updateProjectVariables,
  getProjectSettings,
  updateProjectSettings,
  updateEntryConfig,
};
