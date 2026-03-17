/**
 * Design Service - 设计中心服务层
 * 处理设计页面的 CRUD 操作
 * Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6
 */
const { DesignPage, Project } = require("../models");
const { validatePageSchema } = require("../dsl/validators");
const AppError = require("../utils/AppError");
const ErrorCodes = require("../constants/errorCodes");
const { literal,QueryTypes } = require("sequelize");

const LOCK_TIMEOUT_MS = 30 * 60 * 1000;

const DEFAULT_GLOBAL_SCRIPTS = {
  system: {
    startup: { code: "" },
    shutdown: { code: "" },
  },
  timers: { groups: [], items: [] },
  variableChanges: { groups: [], items: [] },
  custom: { groups: [], items: [] },
};

const parseJsonField = (value) => {
  if (!value) return null;
  if (typeof value === "object") return value;
  try {
    return JSON.parse(value);
  } catch (error) {
    return null;
  }
};

const normalizeGlobalVariables = (raw, fallbackDefinitions = {}) => {
  if (!raw || typeof raw !== "object") {
    return { definitions: fallbackDefinitions, groups: [] };
  }
  if (raw.definitions || raw.groups) {
    return {
      definitions:
        raw.definitions && typeof raw.definitions === "object"
          ? raw.definitions
          : fallbackDefinitions,
      groups: Array.isArray(raw.groups) ? raw.groups : [],
    };
  }
  return { definitions: raw, groups: [] };
};

const normalizeGlobalScripts = (raw) => {
  const source = raw && typeof raw === "object" ? raw : DEFAULT_GLOBAL_SCRIPTS;
  const system = source.system || {};
  const timers = source.timers || {};
  const variableChanges = source.variableChanges || {};
  const custom = source.custom || {};

  return {
    system: {
      startup: { code: system.startup?.code || "" },
      shutdown: { code: system.shutdown?.code || "" },
    },
    timers: {
      groups: Array.isArray(timers.groups) ? timers.groups : [],
      items: Array.isArray(timers.items) ? timers.items : [],
    },
    variableChanges: {
      groups: Array.isArray(variableChanges.groups)
        ? variableChanges.groups
        : [],
      items: Array.isArray(variableChanges.items)
        ? variableChanges.items
        : [],
    },
    custom: {
      groups: Array.isArray(custom.groups) ? custom.groups : [],
      items: Array.isArray(custom.items) ? custom.items : [],
    },
  };
};
/**
 * 设计服务类
 * 提供页面管理的业务逻辑
 */
class DesignService {
  /**
   * 获取工程级别设置（全局变量/脚本）
   * @param {string} projectId - 项目ID
   * @returns {Promise<Object>}
   */
  async getProjectSettings(projectId) {
    const project = await Project.findByPk(projectId);
    if (!project) {
      throw new AppError(ErrorCodes.PROJECT_NOT_FOUND, 404, {
        resource: "Project",
        id: projectId,
      });
    }

    let row = null;
    try {
      const rows = await Project.sequelize.query(
        "SELECT globalVariables, globalScripts FROM design_project_settings WHERE projectId = ? LIMIT 1",
        {
          replacements: [projectId],
          type: QueryTypes.SELECT,
        }
      );
      row = rows && rows.length ? rows[0] : null;
    } catch (error) {
      row = null;
    }

    const parsedVariables = parseJsonField(row?.globalVariables);
    const parsedScripts = parseJsonField(row?.globalScripts);
    const normalizedVariables = normalizeGlobalVariables(
      parsedVariables,
      project.projectVariables || {}
    );

    return {
      globalVariables: normalizedVariables,
      globalScripts: normalizeGlobalScripts(parsedScripts),
    };
  }

  /**
   * 更新工程级别设置（全局变量/脚本）
   * @param {string} projectId - 项目ID
   * @param {Object} settings - 设置数据
   * @param {string} userId - 更新者用户ID
   * @returns {Promise<Object>}
   */
  async updateProjectSettings(projectId, settings, userId) {
    const project = await Project.findByPk(projectId);
    if (!project) {
      throw new AppError(ErrorCodes.PROJECT_NOT_FOUND, 404, {
        resource: "Project",
        id: projectId,
      });
    }

    const normalizedVariables = normalizeGlobalVariables(
      settings?.globalVariables,
      project.projectVariables || {}
    );
    const normalizedScripts = normalizeGlobalScripts(settings?.globalScripts);

    await project.update({
      projectVariables: normalizedVariables.definitions,
      updatedBy: userId,
    });

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
          projectId,
          "1.0.0",
          JSON.stringify(normalizedVariables),
          JSON.stringify(normalizedScripts),
          userId,
          new Date(),
        ],
      }
    );

    return {
      globalVariables: normalizedVariables,
      globalScripts: normalizedScripts,
    };
  }
  /**
   * 获取项目的页面列表
   * Requirements: 7.1
   * @param {string} projectId - 项目ID
   * @returns {Promise<Array>} 页面列表，包含 id, name, type, parentId
   */
  async getPages(projectId) {
    // 验证项目是否存在
    const project = await Project.findByPk(projectId);
    if (!project) {
      throw new AppError(ErrorCodes.PROJECT_NOT_FOUND, 404, {
        resource: "Project",
        id: projectId,
      });
    }

    const pages = await DesignPage.findAll({
      where: { projectId },
      attributes: [
        "id",
        "name",
        "type",
        "parentId",
        "sortOrder",
        "lockedBy",
        "lockedAt",
        "createdAt",
        "updatedAt",
        "schemaContent",
      ],
      order: [
        [literal("parentId IS NOT NULL"), "ASC"], // NULL 优先
        ["parentId", "ASC"],
        ["sortOrder", "ASC"],
        ["createdAt", "ASC"],
      ],
    });

    // 获取项目的入口配置
    const entryConfig = project.entryConfig || {};

    const pageList = pages.map((page) => {
      const payload = page.get({ plain: true });
      const { schemaContent, ...rest } = payload;
      const path = schemaContent?.page?.path || null;
      return { ...rest, path };
    });

    return {
      pages: pageList,
      entryConfig,
    };
  }

  /**
   * 获取单个页面的完整 Schema
   * Requirements: 7.2
   * @param {string} pageId - 页面ID
   * @returns {Promise<Object>} 完整的 Page Schema
   */
  async getPage(pageId) {
    const page = await DesignPage.findByPk(pageId);

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: "DesignPage",
        id: pageId,
      });
    }

    return page.schemaContent;
  }

  /**
   * 更新项目入口配置
   * @param {string} projectId - 项目ID
   * @param {Object} entryConfig - 入口配置 { homePageId, loginPageId, logoutPageId }
   * @returns {Promise<Object>} 更新后的入口配置
   */
  async updateEntryConfig(projectId, entryConfig) {
    const project = await Project.findByPk(projectId);
    if (!project) {
      throw new AppError(ErrorCodes.PROJECT_NOT_FOUND, 404, {
        resource: "Project",
        id: projectId,
      });
    }

    // 合并现有配置
    const newEntryConfig = {
      ...(project.entryConfig || {}),
      ...entryConfig,
    };

    await project.update({ entryConfig: newEntryConfig });

    return newEntryConfig;
  }

  /**
   * 创建新页面
   * Requirements: 7.3
   * @param {string} projectId - 项目ID
   * @param {Object} data - 页面数据 { name, type, parentId, schemaContent }
   * @param {string} userId - 创建者用户ID
   * @returns {Promise<Object>} 创建的页面数据
   */
  async createPage(projectId, data, userId) {
    const { name, type = "page", parentId = null, schemaContent = null } = data;

    // 验证项目是否存在
    const project = await Project.findByPk(projectId);
    if (!project) {
      throw new AppError(ErrorCodes.PROJECT_NOT_FOUND, 404, {
        resource: "Project",
        id: projectId,
      });
    }

    // 如果指定了父页面，验证父页面是否存在且为文件夹
    if (parentId) {
      const parentPage = await DesignPage.findOne({
        where: { id: parentId, projectId },
      });

      if (!parentPage) {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: "父页面不存在",
          parentId,
        });
      }

      if (parentPage.type !== "folder") {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: "父页面必须是文件夹类型",
          parentId,
        });
      }
    }

    // 计算排序顺序（放在同级最后）
    const maxSortOrder = await DesignPage.max("sortOrder", {
      where: { projectId, parentId: parentId || null },
    });
    const sortOrder = (maxSortOrder || 0) + 1;

    // 如果提供了 schemaContent，验证其格式
    // 注意：新版 schema 格式与旧版不同，暂时跳过验证
    // TODO: 更新验证器以支持新版 schema 格式
    // if (schemaContent && type === "page") {
    //   const validation = validatePageSchema(schemaContent);
    //   if (!validation.valid) {
    //     throw new AppError(ErrorCodes.DESIGN_SCHEMA_VALIDATION_FAILED, 400, {
    //       message: "Schema 验证失败",
    //       errors: validation.errors,
    //     });
    //   }
    // }

    // 创建页面记录
    const page = await DesignPage.create({
      projectId,
      parentId,
      name,
      type,
      schemaContent: type === "folder" ? null : schemaContent,
      sortOrder,
      createdBy: userId,
      updatedBy: userId,
    });

    return {
      id: page.id,
      projectId: page.projectId,
      parentId: page.parentId,
      name: page.name,
      type: page.type,
      sortOrder: page.sortOrder,
      createdAt: page.createdAt,
      updatedAt: page.updatedAt,
    };
  }

  /**
   * 更新页面 Schema
   * Requirements: 7.4
   * @param {string} pageId - 页面ID
   * @param {Object} schema - 更新的 Page Schema
   * @param {string} userId - 更新者用户ID
   * @returns {Promise<void>}
   */
  async updatePage(pageId, schema, userId) {
    const page = await DesignPage.findByPk(pageId);

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: "DesignPage",
        id: pageId,
      });
    }

    // 文件夹类型不能更新 schema
    if (page.type === "folder") {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: "文件夹类型不支持 Schema 更新",
      });
    }

    // 验证 Schema 格式
    // const validation = validatePageSchema(schema);
    // if (!validation.valid) {
    //   throw new AppError(ErrorCodes.DESIGN_SCHEMA_VALIDATION_FAILED, 400, {
    //     message: "Schema 验证失败",
    //     errors: validation.errors,
    //   });
    // }

    // 更新页面
    await page.update({
      schemaContent: schema,
      updatedBy: userId,
    });
  }

  /**
   * 删除页面
   * Requirements: 7.5, 7.6
   * @param {string} pageId - 页面ID
   * @returns {Promise<void>}
   */
  async deletePage(pageId) {
    const page = await DesignPage.findByPk(pageId);

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: "DesignPage",
        id: pageId,
      });
    }

    // 如果是文件夹，检查是否有子页面
    // Requirements: 7.6 - 文件夹删除保护
    if (page.type === "folder") {
      const childCount = await DesignPage.count({
        where: { parentId: pageId },
      });

      if (childCount > 0) {
        throw new AppError(ErrorCodes.DESIGN_FOLDER_NOT_EMPTY, 400, {
          message: "文件夹不为空，请先删除或移动子页面",
          childCount,
        });
      }
    }

    // 删除页面
    await page.destroy();
  }

  /**
   * 重命名页面
   * @param {string} pageId - 页面ID
   * @param {string} name - 新名称
   * @param {string} userId - 更新者用户ID
   * @param {string} [path] - 页面路径
   * @returns {Promise<void>}
   */
  async renamePage(pageId, name, userId, path) {
    const page = await DesignPage.findByPk(pageId);

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: "DesignPage",
        id: pageId,
      });
    }

    const updatePayload = {
      name,
      updatedBy: userId,
    };

    if (page.schemaContent && typeof page.schemaContent === "object") {
      const updatedSchema = {
        ...page.schemaContent,
        meta: {
          ...(page.schemaContent.meta || {}),
          name,
        },
      };
      if (path !== undefined && path !== null) {
        updatedSchema.page = {
          ...(page.schemaContent.page || {}),
          path,
        };
      }
      updatePayload.schemaContent = updatedSchema;
    }

    await page.update(updatePayload);
  }

  /**
   * 获取页面详情（包含完整信息）
   * @param {string} pageId - 页面ID
   * @returns {Promise<Object>} 页面完整信息
   */
  async getPageDetail(pageId) {
    const page = await DesignPage.findByPk(pageId, {
      include: [
        { association: "creator", attributes: ["id", "username"] },
        { association: "updater", attributes: ["id", "username"] },
        { association: "locker", attributes: ["id", "username"] },
      ],
    });

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: "DesignPage",
        id: pageId,
      });
    }

    return page;
  }

  /**
   * 移动页面
   * @param {string} pageId - 页面ID
   * @param {Object} data - 移动数据 { parentId, sortOrder, path }
   * @param {string} userId - 更新者用户ID
   * @returns {Promise<void>}
   */
  async movePage(pageId, data, userId) {
    const { parentId, sortOrder, path } = data;

    const page = await DesignPage.findByPk(pageId);

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: "DesignPage",
        id: pageId,
      });
    }

    // 如果指定了新的父页面，验证父页面是否存在且为文件夹
    if (parentId !== undefined && parentId !== null) {
      const parentPage = await DesignPage.findOne({
        where: { id: parentId, projectId: page.projectId },
      });

      if (!parentPage) {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: "父页面不存在",
          parentId,
        });
      }

      if (parentPage.type !== "folder") {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: "父页面必须是文件夹类型",
          parentId,
        });
      }

      // 防止循环引用：不能将页面移动到自己或自己的子页面下
      if (parentId === pageId) {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: "不能将页面移动到自身",
        });
      }
    }

    const updateData = { updatedBy: userId };

    if (parentId !== undefined) {
      updateData.parentId = parentId;
    }

    if (sortOrder !== undefined) {
      updateData.sortOrder = sortOrder;
    }

    if (path !== undefined && page.schemaContent && typeof page.schemaContent === "object") {
      updateData.schemaContent = {
        ...page.schemaContent,
        page: {
          ...(page.schemaContent.page || {}),
          path,
        },
      };
    }

    await page.update(updateData);
  }

  /**
   * 获取页面锁状态
   * @param {string} pageId - 页面 ID
   * @returns {Promise<Object>}
   */
  async getPageLockStatus(pageId) {
    const page = await DesignPage.findByPk(pageId, {
      include: [{ association: "locker", attributes: ["id", "username"] }],
    });

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: "DesignPage",
        id: pageId,
      });
    }

    const expired = await this._clearExpiredLock(page);
    if (expired) {
      return {
        locked: false,
        lockedBy: null,
        lockedByName: null,
        lockedAt: null,
      };
    }

    return {
      locked: Boolean(page.lockedBy),
      lockedBy: page.lockedBy,
      lockedByName: page.locker?.username || null,
      lockedAt: page.lockedAt,
    };
  }

  /**
   * 获取页面锁
   * @param {string} pageId - 页面 ID
   * @param {string} userId - 用户 ID
   * @param {string} userName - 用户名
   * @returns {Promise<{success: boolean, data: Object}>}
   */
  async acquirePageLock(pageId, userId, userName) {
    const page = await DesignPage.findByPk(pageId, {
      include: [{ association: "locker", attributes: ["id", "username"] }],
    });

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: "DesignPage",
        id: pageId,
      });
    }

    await this._clearExpiredLock(page);

    if (page.lockedBy && page.lockedBy !== userId) {
      return {
        success: false,
        data: {
          locked: true,
          lockedBy: page.lockedBy,
          lockedByName: page.locker?.username || null,
          lockedAt: page.lockedAt,
        },
      };
    }

    const lockedAt = new Date();
    await page.update({ lockedBy: userId, lockedAt });

    return {
      success: true,
      data: {
        locked: true,
        lockedBy: userId,
        lockedByName: userName || null,
        lockedAt,
      },
    };
  }

  /**
   * 释放页面锁
   * @param {string} pageId - 页面 ID
   * @param {string} userId - 用户 ID
   * @returns {Promise<Object>}
   */
  async releasePageLock(pageId, userId) {
    const page = await DesignPage.findByPk(pageId);

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: "DesignPage",
        id: pageId,
      });
    }

    if (page.lockedBy && page.lockedBy !== userId) {
      throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
        message: "非锁定者无法释放页面锁",
      });
    }

    await page.update({ lockedBy: null, lockedAt: null });
    return { locked: false };
  }

  /**
   * 心跳续锁
   * @param {string} pageId - 页面 ID
   * @param {string} userId - 用户 ID
   * @returns {Promise<Object>}
   */
  async heartbeatPageLock(pageId, userId) {
    const page = await DesignPage.findByPk(pageId);

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: "DesignPage",
        id: pageId,
      });
    }

    if (page.lockedBy && page.lockedBy !== userId) {
      throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
        message: "非锁定者无法续锁",
      });
    }

    const lockedAt = new Date();
    await page.update({ lockedBy: userId, lockedAt });
    return { locked: true, lockedBy: userId, lockedAt };
  }

  /**
   * 强制释放页面锁
   * @param {string} pageId - 页面 ID
   * @returns {Promise<Object>}
   */
  async forceReleasePageLock(pageId) {
    const page = await DesignPage.findByPk(pageId);

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: "DesignPage",
        id: pageId,
      });
    }

    await page.update({ lockedBy: null, lockedAt: null });
    return { locked: false };
  }

  /**
   * 判断锁是否过期
   * @param {Date | null} lockedAt - 锁定时间
   * @returns {boolean}
   * @private
   */
  _isLockExpired(lockedAt) {
    if (!lockedAt) return false;
    const lockedTime = new Date(lockedAt).getTime();
    return Date.now() - lockedTime > LOCK_TIMEOUT_MS;
  }

  /**
   * 清理过期锁
   * @param {import('../models').DesignPage} page - 页面记录
   * @returns {Promise<boolean>} 是否清理
   * @private
   */
  async _clearExpiredLock(page) {
    if (!page.lockedBy || !page.lockedAt) return false;
    if (!this._isLockExpired(page.lockedAt)) return false;

    await page.update({ lockedBy: null, lockedAt: null });
    return true;
  }
}

module.exports = new DesignService();
