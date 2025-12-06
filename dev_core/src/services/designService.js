/**
 * Design Service - 设计中心服务层
 * 处理设计页面的 CRUD 操作
 * Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6
 */
const { DesignPage, Project } = require("../models");
const { validatePageSchema } = require("../dsl/validators");
const AppError = require("../utils/AppError");
const ErrorCodes = require("../constants/errorCodes");
const { literal } = require("sequelize");
/**
 * 设计服务类
 * 提供页面管理的业务逻辑
 */
class DesignService {
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
      ],
      order: [
        [literal("parentId IS NOT NULL"), "ASC"], // NULL 优先
        ["parentId", "ASC"],
        ["sortOrder", "ASC"],
        ["createdAt", "ASC"],
      ],
    });

    return pages;
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
    if (schemaContent && type === "page") {
      const validation = validatePageSchema(schemaContent);
      if (!validation.valid) {
        throw new AppError(ErrorCodes.DESIGN_SCHEMA_VALIDATION_FAILED, 400, {
          message: "Schema 验证失败",
          errors: validation.errors,
        });
      }
    }

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
    const validation = validatePageSchema(schema);
    if (!validation.valid) {
      throw new AppError(ErrorCodes.DESIGN_SCHEMA_VALIDATION_FAILED, 400, {
        message: "Schema 验证失败",
        errors: validation.errors,
      });
    }

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
   * @returns {Promise<void>}
   */
  async renamePage(pageId, name, userId) {
    const page = await DesignPage.findByPk(pageId);

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: "DesignPage",
        id: pageId,
      });
    }

    await page.update({
      name,
      updatedBy: userId,
    });

    // 如果页面有 schemaContent，同步更新 meta.name
    if (page.schemaContent && page.schemaContent.meta) {
      const updatedSchema = {
        ...page.schemaContent,
        meta: {
          ...page.schemaContent.meta,
          name,
        },
      };
      await page.update({ schemaContent: updatedSchema });
    }
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
   * @param {Object} data - 移动数据 { parentId, sortOrder }
   * @param {string} userId - 更新者用户ID
   * @returns {Promise<void>}
   */
  async movePage(pageId, data, userId) {
    const { parentId, sortOrder } = data;

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

    await page.update(updateData);
  }
}

module.exports = new DesignService();
