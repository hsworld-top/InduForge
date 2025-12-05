/**
 * Design Service - 设计中心服务层
 * 处理设计页面的 CRUD 操作
 * Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6
 */
const { DesignPage, Project, User } = require('../models');
const { validatePageSchema } = require('../dsl/validators');
const AppError = require('../utils/AppError');
const ErrorCodes = require('../constants/errorCodes');

/**
 * 创建默认的 Page Schema
 * @param {string} pageId - 页面ID
 * @param {string} name - 页面名称
 * @returns {Object} 默认 Page Schema
 */
function createDefaultPageSchema(pageId, name) {
  return {
    version: '2.0.0',
    meta: {
      id: pageId,
      name: name,
      screenshot: null,
      lockedBy: null,
      lockedAt: null,
    },
    config: {
      width: 1920,
      height: 1080,
      scaleMode: 'fit',
      backgroundColor: '#ffffff',
      backgroundImage: null,
      gridSize: 8,
      snapToGrid: true,
      theme: 'light',
    },
    variables: {},
    dataSources: [],
    components: [],
    permissions: {
      roles: [],
      componentAcl: [],
    },
  };
}

class DesignService {
  /**
   * 获取项目的页面列表
   * Requirements: 7.1
   * @param {string} projectId - 工程ID
   * @returns {Promise<Array>} 页面列表 (id, name, type, parentId, sortOrder)
   */
  async getPages(projectId) {
    const pages = await DesignPage.findAll({
      where: { projectId },
      attributes: ['id', 'name', 'type', 'parentId', 'sortOrder', 'lockedBy', 'lockedAt', 'createdAt', 'updatedAt'],
      order: [
        ['parentId', 'ASC NULLS FIRST'],
        ['sortOrder', 'ASC'],
        ['createdAt', 'ASC'],
      ],
      include: [
        {
          model: User,
          as: 'locker',
          attributes: ['id', 'username', 'fullName'],
          required: false,
        },
      ],
    });

    return pages;
  }


  /**
   * 获取单个页面的完整 Schema
   * Requirements: 7.2
   * @param {string} pageId - 页面ID
   * @returns {Promise<Object>} 页面 Schema
   */
  async getPage(pageId) {
    const page = await DesignPage.findByPk(pageId, {
      include: [
        {
          model: User,
          as: 'creator',
          attributes: ['id', 'username', 'fullName'],
        },
        {
          model: User,
          as: 'updater',
          attributes: ['id', 'username', 'fullName'],
          required: false,
        },
        {
          model: User,
          as: 'locker',
          attributes: ['id', 'username', 'fullName'],
          required: false,
        },
      ],
    });

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        message: '页面不存在',
        pageId,
      });
    }

    return page;
  }

  /**
   * 创建新页面
   * Requirements: 7.3
   * @param {string} projectId - 工程ID
   * @param {Object} data - 页面数据 { name, type, parentId }
   * @param {string} userId - 创建者ID
   * @returns {Promise<Object>} 创建的页面
   */
  async createPage(projectId, data, userId) {
    const { name, type = 'page', parentId = null } = data;

    // 验证父页面存在且为文件夹
    if (parentId) {
      const parent = await DesignPage.findOne({
        where: { id: parentId, projectId },
      });

      if (!parent) {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: '父页面不存在',
          parentId,
        });
      }

      if (parent.type !== 'folder') {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: '父页面必须是文件夹类型',
          parentId,
        });
      }
    }

    // 计算排序顺序
    const maxSortOrder = await DesignPage.max('sortOrder', {
      where: { projectId, parentId: parentId || null },
    });
    const sortOrder = (maxSortOrder || 0) + 1;

    // 创建页面记录
    const page = await DesignPage.create({
      projectId,
      parentId,
      name,
      type,
      sortOrder,
      createdBy: userId,
      // 文件夹不需要 schema，页面和对话框需要默认 schema
      schemaContent: type === 'folder' ? null : null, // schema 在创建后设置
    });

    // 为非文件夹类型创建默认 schema
    if (type !== 'folder') {
      const defaultSchema = createDefaultPageSchema(page.id, name);
      await page.update({ schemaContent: defaultSchema });
    }

    // 重新获取完整数据
    return this.getPage(page.id);
  }

  /**
   * 更新页面 Schema
   * Requirements: 7.4
   * @param {string} pageId - 页面ID
   * @param {Object} schema - 新的 Page Schema
   * @param {string} userId - 更新者ID
   * @returns {Promise<Object>} 更新后的页面
   */
  async updatePage(pageId, schema, userId) {
    const page = await DesignPage.findByPk(pageId);

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        message: '页面不存在',
        pageId,
      });
    }

    // 文件夹不能更新 schema
    if (page.type === 'folder') {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '文件夹不支持 Schema 更新',
      });
    }

    // 验证 Schema
    const validationResult = validatePageSchema(schema);
    if (!validationResult.valid) {
      throw new AppError(ErrorCodes.DESIGN_SCHEMA_VALIDATION_FAILED, 400, {
        message: 'Schema 验证失败',
        errors: validationResult.errors,
      });
    }

    // 更新页面
    await page.update({
      schemaContent: schema,
      updatedBy: userId,
    });

    return this.getPage(pageId);
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
        message: '页面不存在',
        pageId,
      });
    }

    // 检查是否有子页面（文件夹保护）
    if (page.type === 'folder') {
      const childCount = await DesignPage.count({
        where: { parentId: pageId },
      });

      if (childCount > 0) {
        throw new AppError(ErrorCodes.DESIGN_FOLDER_NOT_EMPTY, 400, {
          message: '文件夹不为空，请先删除或移动子页面',
          childCount,
        });
      }
    }

    await page.destroy();
  }

  /**
   * 重命名页面
   * @param {string} pageId - 页面ID
   * @param {string} name - 新名称
   * @param {string} userId - 更新者ID
   * @returns {Promise<Object>} 更新后的页面
   */
  async renamePage(pageId, name, userId) {
    const page = await DesignPage.findByPk(pageId);

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        message: '页面不存在',
        pageId,
      });
    }

    // 更新页面名称
    await page.update({
      name,
      updatedBy: userId,
    });

    // 如果有 schema，同步更新 meta.name
    if (page.schemaContent && page.type !== 'folder') {
      const updatedSchema = {
        ...page.schemaContent,
        meta: {
          ...page.schemaContent.meta,
          name,
        },
      };
      await page.update({ schemaContent: updatedSchema });
    }

    return this.getPage(pageId);
  }

  /**
   * 移动页面（更改父级或排序）
   * @param {string} pageId - 页面ID
   * @param {Object} data - { parentId, sortOrder }
   * @param {string} userId - 更新者ID
   * @returns {Promise<Object>} 更新后的页面
   */
  async movePage(pageId, data, userId) {
    const { parentId, sortOrder } = data;
    const page = await DesignPage.findByPk(pageId);

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        message: '页面不存在',
        pageId,
      });
    }

    // 验证新父级
    if (parentId !== undefined && parentId !== null) {
      const parent = await DesignPage.findOne({
        where: { id: parentId, projectId: page.projectId },
      });

      if (!parent) {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: '目标父页面不存在',
          parentId,
        });
      }

      if (parent.type !== 'folder') {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: '目标父页面必须是文件夹类型',
          parentId,
        });
      }

      // 防止循环引用
      if (parentId === pageId) {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: '不能将页面移动到自身',
        });
      }
    }

    const updates = { updatedBy: userId };
    if (parentId !== undefined) updates.parentId = parentId;
    if (sortOrder !== undefined) updates.sortOrder = sortOrder;

    await page.update(updates);

    return this.getPage(pageId);
  }
}

module.exports = new DesignService();
