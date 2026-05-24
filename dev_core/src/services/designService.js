/**
 * Design Service - 设计中心服务层
 * 处理设计页面的 CRUD 操作
 * Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6
 */
const { DesignPage, Project, ProjectRole } = require('../models')
const { sequelize } = require('../config/database')
const { validatePageSchema } = require('../dsl/validators')
const AppError = require('../utils/AppError')
const ErrorCodes = require('../constants/errorCodes')
const { literal } = require('sequelize')
const { getProjectSettingsRow, upsertProjectSettings } = require('./projectSettingsStore')

const LOCK_TIMEOUT_MS = 30 * 60 * 1000
const ENTRY_PAGE_KEYS = ['homePageId', 'loginPageId', 'logoutPageId']
const asArray = (value) => (Array.isArray(value) ? value : [])
const normalizeId = (value) => String(value ?? '').trim()

const resolveSchemaPageForRuntimeAccess = (schema, pageId) => {
  if (!schema || typeof schema !== 'object') return {}
  if (schema.page && typeof schema.page === 'object') return schema.page
  if (schema.pagesById && typeof schema.pagesById === 'object') {
    return schema.pagesById[pageId] || Object.values(schema.pagesById)[0] || {}
  }
  return {}
}

const collectSchemaNodesForRuntimeAccess = (schema) => {
  if (!schema || typeof schema !== 'object') return []
  if (schema.nodesById && typeof schema.nodesById === 'object') {
    return Object.values(schema.nodesById).filter(Boolean)
  }
  return []
}

async function validateRuntimeAccessSchema(projectId, pageId, schema) {
  const schemaPage = resolveSchemaPageForRuntimeAccess(schema, pageId)
  const access = schemaPage?.config?.runtimeAccess || {}
  const usedRoleIds = new Set()
  const schemeIds = new Set()
  const errors = []

  asArray(access.allowedRoles).forEach((roleRef) => {
    const roleId = normalizeId(roleRef?.roleId || roleRef?.id)
    if (roleId) usedRoleIds.add(roleId)
  })
  asArray(access.schemes).forEach((scheme) => {
    const schemeId = normalizeId(scheme?.id)
    if (!schemeId) return
    schemeIds.add(schemeId)
    asArray(scheme.roleRefs).forEach((roleRef) => {
      const roleId = normalizeId(roleRef?.roleId || roleRef?.id)
      if (roleId) usedRoleIds.add(roleId)
    })
  })
  collectSchemaNodesForRuntimeAccess(schema).forEach((node) => {
    const runtimeAccess = node?.permissions?.runtimeAccess
    if (!runtimeAccess || typeof runtimeAccess !== 'object') return
    ;[runtimeAccess.visibleSchemeId, runtimeAccess.operableSchemeId]
      .map(normalizeId)
      .filter(Boolean)
      .forEach((schemeId) => {
        if (!schemeIds.has(schemeId)) {
          errors.push(`组件 ${node.id} 引用了不存在的权限方案 ${schemeId}`)
        }
      })
  })

  if (usedRoleIds.size > 0) {
    const roles = await ProjectRole.findAll({
      where: { projectId, id: Array.from(usedRoleIds) },
      attributes: ['id'],
      raw: true,
    })
    const existing = new Set(roles.map((role) => role.id))
    Array.from(usedRoleIds).forEach((roleId) => {
      if (!existing.has(roleId)) {
        errors.push(`引用了不存在的运行态角色 ${roleId}`)
      }
    })
  }

  if (errors.length > 0) {
    throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
      message: `运行态权限配置校验失败: ${errors.join('; ')}`,
      errors,
    })
  }
}

const DEFAULT_GLOBAL_SCRIPTS = {
  system: {
    startup: { code: '' },
    shutdown: { code: '' },
  },
  timers: { groups: [], items: [] },
  variableChanges: { groups: [], items: [] },
  custom: { groups: [], items: [] },
}

const parseJsonField = (value) => {
  if (!value) return null
  if (typeof value === 'object') return value
  try {
    return JSON.parse(value)
  } catch (error) {
    return null
  }
}

const normalizeGlobalVariables = (raw, fallbackDefinitions = {}) => {
  if (!raw || typeof raw !== 'object') {
    return { definitions: fallbackDefinitions, groups: [] }
  }
  if (raw.definitions || raw.groups) {
    return {
      definitions:
        raw.definitions && typeof raw.definitions === 'object'
          ? raw.definitions
          : fallbackDefinitions,
      groups: Array.isArray(raw.groups) ? raw.groups : [],
    }
  }
  return { definitions: raw, groups: [] }
}

const normalizeGlobalScripts = (raw) => {
  const source = raw && typeof raw === 'object' ? raw : DEFAULT_GLOBAL_SCRIPTS
  const system = source.system || {}
  const timers = source.timers || {}
  const variableChanges = source.variableChanges || {}
  const custom = source.custom || {}

  return {
    system: {
      startup: { code: system.startup?.code || '' },
      shutdown: { code: system.shutdown?.code || '' },
    },
    timers: {
      groups: Array.isArray(timers.groups) ? timers.groups : [],
      items: Array.isArray(timers.items) ? timers.items : [],
    },
    variableChanges: {
      groups: Array.isArray(variableChanges.groups) ? variableChanges.groups : [],
      items: Array.isArray(variableChanges.items) ? variableChanges.items : [],
    },
    custom: {
      groups: Array.isArray(custom.groups) ? custom.groups : [],
      items: Array.isArray(custom.items) ? custom.items : [],
    },
  }
}

const normalizeProjectI18n = (raw) => {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return {}
  }
  return raw
}

const toPathSegment = (value) => {
  const normalized = String(value || '')
    .trim()
    .replace(/\s+/g, '-')
  const sanitized = normalized.replace(/[/?#\\%]+/g, '-')
  return sanitized || 'page'
}

/**
 * 从页面 Schema 中提取路径。
 * @param {Object|null|undefined} schemaContent - 页面 Schema
 * @returns {string|null|undefined}
 */
const resolvePathFromSchemaContent = (schemaContent) => {
  if (!schemaContent || typeof schemaContent !== 'object') {
    return undefined
  }

  const schemaPath = schemaContent?.page?.path
  if (typeof schemaPath !== 'string') {
    return undefined
  }

  const normalizedPath = schemaPath.trim()
  return normalizedPath || null
}

/**
 * 统一解析需持久化的页面路径。
 * @param {string|null|undefined} explicitPath - 显式传入路径
 * @param {Object|null|undefined} schemaContent - 页面 Schema
 * @param {string} type - 页面类型
 * @returns {string|null}
 */
const resolvePersistedPagePath = (explicitPath, schemaContent, type = 'page') => {
  if (type !== 'page' && type !== 'dialog') {
    return null
  }

  if (typeof explicitPath === 'string') {
    const normalizedPath = explicitPath.trim()
    return normalizedPath || null
  }

  return resolvePathFromSchemaContent(schemaContent) ?? null
}
/**
 * 设计服务类
 * 提供页面管理的业务逻辑
 */
class DesignService {
  /**
   * 清理入口配置中指向已删除页面的绑定，避免残留脏引用。
   * @param {string} projectId - 项目ID
   * @param {string[]} removedPageIds - 已删除页面ID列表
   * @param {import("sequelize").Transaction} [transaction] - 可选事务
   * @returns {Promise<void>}
   * @private
   */
  async _cleanupProjectEntryConfig(projectId, removedPageIds, transaction) {
    const normalizedRemovedIds = (Array.isArray(removedPageIds) ? removedPageIds : [])
      .filter((id) => typeof id === 'string' && id.trim())
      .map((id) => id.trim())

    if (normalizedRemovedIds.length === 0) {
      return
    }

    const removedIdSet = new Set(normalizedRemovedIds)
    const project = await Project.findByPk(projectId, transaction ? { transaction } : undefined)
    if (!project) {
      return
    }

    const currentEntryConfig =
      project.entryConfig && typeof project.entryConfig === 'object' ? project.entryConfig : {}
    const nextEntryConfig = { ...currentEntryConfig }
    let changed = false

    ENTRY_PAGE_KEYS.forEach((entryKey) => {
      const boundPageId = nextEntryConfig[entryKey]
      if (typeof boundPageId === 'string' && removedIdSet.has(boundPageId)) {
        nextEntryConfig[entryKey] = null
        changed = true
      }
    })

    if (!changed) {
      return
    }

    await project.update(
      { entryConfig: nextEntryConfig },
      transaction ? { transaction } : undefined,
    )
  }

  /**
   * 获取工程级别设置（全局变量/脚本/国际化）
   * @param {string} projectId - 项目ID
   * @returns {Promise<Object>}
   */
  async getProjectSettings(projectId) {
    const project = await Project.findByPk(projectId)
    if (!project) {
      throw new AppError(ErrorCodes.PROJECT_NOT_FOUND, 404, {
        resource: 'Project',
        id: projectId,
      })
    }

    let row = null
    try {
      row = await getProjectSettingsRow(Project.sequelize, projectId)
    } catch (error) {
      row = null
    }

    const parsedVariables = parseJsonField(row?.globalVariables)
    const parsedScripts = parseJsonField(row?.globalScripts)
    const parsedI18n = parseJsonField(row?.i18n)
    const normalizedVariables = normalizeGlobalVariables(
      parsedVariables,
      project.projectVariables || {},
    )

    return {
      globalVariables: normalizedVariables,
      globalScripts: normalizeGlobalScripts(parsedScripts),
      i18n: normalizeProjectI18n(parsedI18n),
    }
  }

  /**
   * 更新工程级别设置（全局变量/脚本/国际化）
   * @param {string} projectId - 项目ID
   * @param {Object} settings - 设置数据
   * @param {string} userId - 更新者用户ID
   * @returns {Promise<Object>}
   */
  async updateProjectSettings(projectId, settings, userId) {
    const project = await Project.findByPk(projectId)
    if (!project) {
      throw new AppError(ErrorCodes.PROJECT_NOT_FOUND, 404, {
        resource: 'Project',
        id: projectId,
      })
    }

    const normalizedVariables = normalizeGlobalVariables(
      settings?.globalVariables,
      project.projectVariables || {},
    )
    const normalizedScripts = normalizeGlobalScripts(settings?.globalScripts)
    const normalizedI18n = normalizeProjectI18n(settings?.i18n)

    await project.update({
      projectVariables: normalizedVariables.definitions,
      updatedBy: userId,
    })

    await upsertProjectSettings(Project.sequelize, {
      projectId,
      schemaVersion: '1.0.0',
      globalVariables: normalizedVariables,
      globalScripts: normalizedScripts,
      i18n: normalizedI18n,
      updatedBy: userId,
      updatedAt: new Date(),
    })

    return {
      globalVariables: normalizedVariables,
      globalScripts: normalizedScripts,
      i18n: normalizedI18n,
    }
  }
  /**
   * 获取项目的页面列表
   * Requirements: 7.1
   * @param {string} projectId - 项目ID
   * @returns {Promise<Array>} 页面列表，包含 id, name, type, parentId
   */
  async getPages(projectId) {
    // 验证项目是否存在
    const project = await Project.findByPk(projectId)
    if (!project) {
      throw new AppError(ErrorCodes.PROJECT_NOT_FOUND, 404, {
        resource: 'Project',
        id: projectId,
      })
    }

    const pages = await DesignPage.findAll({
      where: { projectId },
      attributes: [
        'id',
        'name',
        'path',
        'type',
        'parentId',
        'sortOrder',
        'lockedBy',
        'lockedAt',
        'createdAt',
        'updatedAt',
        'schemaContent',
      ],
      order: [
        [literal('"parentId" IS NOT NULL'), 'ASC'], // PostgreSQL 需要显式保留 camelCase 列名
        ['parentId', 'ASC'],
        ['sortOrder', 'ASC'],
        ['createdAt', 'ASC'],
      ],
    })

    // 获取项目的入口配置
    const entryConfig = project.entryConfig || {}

    const pageList = pages.map((page) => {
      const payload = page.get({ plain: true })
      const { schemaContent, ...rest } = payload
      const path = rest.path || resolvePathFromSchemaContent(schemaContent) || null
      return { ...rest, path }
    })

    return {
      pages: pageList,
      entryConfig,
    }
  }

  /**
   * 获取单个页面的完整 Schema
   * Requirements: 7.2
   * @param {string} pageId - 页面ID
   * @returns {Promise<Object>} 完整的 Page Schema
   */
  async getPage(pageId) {
    const page = await DesignPage.findByPk(pageId)

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: 'DesignPage',
        id: pageId,
      })
    }

    return page.schemaContent
  }

  /**
   * 更新项目入口配置
   * @param {string} projectId - 项目ID
   * @param {Object} entryConfig - 入口配置 { homePageId, loginPageId, logoutPageId }
   * @returns {Promise<Object>} 更新后的入口配置
   */
  async updateEntryConfig(projectId, entryConfig) {
    const project = await Project.findByPk(projectId)
    if (!project) {
      throw new AppError(ErrorCodes.PROJECT_NOT_FOUND, 404, {
        resource: 'Project',
        id: projectId,
      })
    }

    // 合并现有配置
    const newEntryConfig = {
      ...(project.entryConfig || {}),
      ...entryConfig,
    }

    await project.update({ entryConfig: newEntryConfig })

    return newEntryConfig
  }

  /**
   * 创建新页面
   * Requirements: 7.3
   * @param {string} projectId - 项目ID
   * @param {Object} data - 页面数据 { name, type, parentId, path, schemaContent }
   * @param {string} userId - 创建者用户ID
   * @returns {Promise<Object>} 创建的页面数据
   */
  async createPage(projectId, data, userId) {
    const { name, type = 'page', parentId = null, path, schemaContent = null } = data

    // 验证项目是否存在
    const project = await Project.findByPk(projectId)
    if (!project) {
      throw new AppError(ErrorCodes.PROJECT_NOT_FOUND, 404, {
        resource: 'Project',
        id: projectId,
      })
    }

    // 如果指定了父页面，验证父页面是否存在且为文件夹
    if (parentId) {
      const parentPage = await DesignPage.findOne({
        where: { id: parentId, projectId },
      })

      if (!parentPage) {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: '父页面不存在',
          parentId,
        })
      }

      if (parentPage.type !== 'folder') {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: '父页面必须是文件夹类型',
          parentId,
        })
      }
    }

    // 计算排序顺序（放在同级最后）
    const maxSortOrder = await DesignPage.max('sortOrder', {
      where: { projectId, parentId: parentId || null },
    })
    const sortOrder = (maxSortOrder || 0) + 1
    const persistedPath = resolvePersistedPagePath(path, schemaContent, type)

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
      path: persistedPath,
      type,
      schemaContent: type === 'folder' ? null : schemaContent,
      sortOrder,
      createdBy: userId,
      updatedBy: userId,
    })

    return {
      id: page.id,
      projectId: page.projectId,
      parentId: page.parentId,
      name: page.name,
      path: page.path,
      type: page.type,
      sortOrder: page.sortOrder,
      createdAt: page.createdAt,
      updatedAt: page.updatedAt,
    }
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
    const page = await DesignPage.findByPk(pageId)

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: 'DesignPage',
        id: pageId,
      })
    }

    // 文件夹类型不能更新 schema
    if (page.type === 'folder') {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '文件夹类型不支持 Schema 更新',
      })
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
    await validateRuntimeAccessSchema(page.projectId, page.id, schema)

    const nextPath = resolvePathFromSchemaContent(schema)
    const updatePayload = {
      schemaContent: schema,
      updatedBy: userId,
    }

    if (nextPath !== undefined) {
      updatePayload.path = nextPath
    }

    await page.update(updatePayload)
  }

  /**
   * 删除页面
   * Requirements: 7.5, 7.6
   * @param {string} pageId - 页面ID
   * @param {"single" | "folder-only" | "cascade"} [mode='single'] - 删除模式
   * @returns {Promise<void>}
   */
  async deletePage(pageId, mode = 'single') {
    const page = await DesignPage.findByPk(pageId)

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: 'DesignPage',
        id: pageId,
      })
    }

    if (page.type !== 'folder') {
      await sequelize.transaction(async (transaction) => {
        await page.destroy({ transaction })
        await this._cleanupProjectEntryConfig(page.projectId, [page.id], transaction)
      })
      return
    }

    if (!['single', 'folder-only', 'cascade'].includes(mode)) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '删除模式无效',
        mode,
      })
    }

    const descendants = await DesignPage.findAll({
      where: { projectId: page.projectId },
      order: [['createdAt', 'ASC']],
    })
    const childMap = new Map()
    descendants.forEach((item) => {
      const parentKey = item.parentId || '__root__'
      if (!childMap.has(parentKey)) {
        childMap.set(parentKey, [])
      }
      childMap.get(parentKey).push(item)
    })
    const collectDescendants = (folderId) => {
      const pages = []
      const folders = []
      const stack = [folderId]
      while (stack.length) {
        const currentFolderId = stack.pop()
        const children = childMap.get(currentFolderId) || []
        children.forEach((child) => {
          if (child.type === 'folder') {
            folders.push(child)
            stack.push(child.id)
            return
          }
          pages.push(child)
        })
      }
      return { pages, folders }
    }

    const { pages: descendantPages, folders: descendantFolders } = collectDescendants(pageId)

    if (mode === 'single') {
      if (descendantPages.length > 0 || descendantFolders.length > 0) {
        throw new AppError(ErrorCodes.DESIGN_FOLDER_NOT_EMPTY, 400, {
          message: '文件夹不为空，请先删除或移动子页面',
          childCount: descendantPages.length + descendantFolders.length,
        })
      }
      await sequelize.transaction(async (transaction) => {
        await page.destroy({ transaction })
        await this._cleanupProjectEntryConfig(page.projectId, [page.id], transaction)
      })
      return
    }

    await sequelize.transaction(async (transaction) => {
      if (mode === 'folder-only') {
        for (const childPage of descendantPages) {
          const schemaContent =
            childPage.schemaContent && typeof childPage.schemaContent === 'object'
              ? {
                  ...childPage.schemaContent,
                  page: {
                    ...(childPage.schemaContent.page || {}),
                    path: `/${toPathSegment(childPage.name)}`,
                  },
                }
              : childPage.schemaContent
          await childPage.update(
            {
              parentId: null,
              path: `/${toPathSegment(childPage.name)}`,
              schemaContent,
            },
            { transaction },
          )
        }

        for (const folder of [...descendantFolders].reverse()) {
          await folder.destroy({ transaction })
        }
        await page.destroy({ transaction })
        await this._cleanupProjectEntryConfig(
          page.projectId,
          [page.id, ...descendantFolders.map((folder) => folder.id)],
          transaction,
        )
        return
      }

      for (const childPage of descendantPages) {
        await childPage.destroy({ transaction })
      }
      for (const folder of [...descendantFolders].reverse()) {
        await folder.destroy({ transaction })
      }
      await page.destroy({ transaction })
      await this._cleanupProjectEntryConfig(
        page.projectId,
        [
          page.id,
          ...descendantPages.map((childPage) => childPage.id),
          ...descendantFolders.map((folder) => folder.id),
        ],
        transaction,
      )
    })
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
    const page = await DesignPage.findByPk(pageId)

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: 'DesignPage',
        id: pageId,
      })
    }

    const updatePayload = {
      name,
      updatedBy: userId,
    }

    if (path !== undefined) {
      updatePayload.path = typeof path === 'string' ? path.trim() || null : path
    }

    if (page.schemaContent && typeof page.schemaContent === 'object') {
      const updatedSchema = {
        ...page.schemaContent,
        meta: {
          ...(page.schemaContent.meta || {}),
          name,
        },
      }
      if (path !== undefined && path !== null) {
        updatedSchema.page = {
          ...(page.schemaContent.page || {}),
          path,
        }
      }
      updatePayload.schemaContent = updatedSchema
    }

    await page.update(updatePayload)
  }

  /**
   * 获取页面详情（包含完整信息）
   * @param {string} pageId - 页面ID
   * @returns {Promise<Object>} 页面完整信息
   */
  async getPageDetail(pageId) {
    const page = await DesignPage.findByPk(pageId, {
      include: [
        { association: 'creator', attributes: ['id', 'username'] },
        { association: 'updater', attributes: ['id', 'username'] },
        { association: 'locker', attributes: ['id', 'username'] },
      ],
    })

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: 'DesignPage',
        id: pageId,
      })
    }

    return page
  }

  /**
   * 移动页面
   * @param {string} pageId - 页面ID
   * @param {Object} data - 移动数据 { parentId, sortOrder, path }
   * @param {string} userId - 更新者用户ID
   * @returns {Promise<void>}
   */
  async movePage(pageId, data, userId) {
    const { parentId, sortOrder, path } = data

    const page = await DesignPage.findByPk(pageId)

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: 'DesignPage',
        id: pageId,
      })
    }

    // 如果指定了新的父页面，验证父页面是否存在且为文件夹
    if (parentId !== undefined && parentId !== null) {
      const parentPage = await DesignPage.findOne({
        where: { id: parentId, projectId: page.projectId },
      })

      if (!parentPage) {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: '父页面不存在',
          parentId,
        })
      }

      if (parentPage.type !== 'folder') {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: '父页面必须是文件夹类型',
          parentId,
        })
      }

      // 防止循环引用：不能将页面移动到自己或自己的子页面下
      if (parentId === pageId) {
        throw new AppError(ErrorCodes.DESIGN_INVALID_PARENT, 400, {
          message: '不能将页面移动到自身',
        })
      }
    }

    const updateData = { updatedBy: userId }

    if (parentId !== undefined) {
      updateData.parentId = parentId
    }

    if (sortOrder !== undefined) {
      updateData.sortOrder = sortOrder
    }

    if (path !== undefined && page.schemaContent && typeof page.schemaContent === 'object') {
      updateData.schemaContent = {
        ...page.schemaContent,
        page: {
          ...(page.schemaContent.page || {}),
          path,
        },
      }
    }

    if (path !== undefined) {
      updateData.path = typeof path === 'string' ? path.trim() || null : path
    }

    await page.update(updateData)
  }

  /**
   * 获取页面锁状态
   * @param {string} pageId - 页面 ID
   * @returns {Promise<Object>}
   */
  async getPageLockStatus(pageId) {
    const page = await DesignPage.findByPk(pageId, {
      include: [{ association: 'locker', attributes: ['id', 'username'] }],
    })

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: 'DesignPage',
        id: pageId,
      })
    }

    const expired = await this._clearExpiredLock(page)
    if (expired) {
      return {
        locked: false,
        lockedBy: null,
        lockedByName: null,
        lockedAt: null,
      }
    }

    return {
      locked: Boolean(page.lockedBy),
      lockedBy: page.lockedBy,
      lockedByName: page.locker?.username || null,
      lockedAt: page.lockedAt,
    }
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
      include: [{ association: 'locker', attributes: ['id', 'username'] }],
    })

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: 'DesignPage',
        id: pageId,
      })
    }

    await this._clearExpiredLock(page)

    if (page.lockedBy && page.lockedBy !== userId) {
      return {
        success: false,
        data: {
          locked: true,
          lockedBy: page.lockedBy,
          lockedByName: page.locker?.username || null,
          lockedAt: page.lockedAt,
        },
      }
    }

    const lockedAt = new Date()
    await page.update({ lockedBy: userId, lockedAt })

    return {
      success: true,
      data: {
        locked: true,
        lockedBy: userId,
        lockedByName: userName || null,
        lockedAt,
      },
    }
  }

  /**
   * 释放页面锁
   * @param {string} pageId - 页面 ID
   * @param {string} userId - 用户 ID
   * @returns {Promise<Object>}
   */
  async releasePageLock(pageId, userId) {
    const page = await DesignPage.findByPk(pageId)

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: 'DesignPage',
        id: pageId,
      })
    }

    if (page.lockedBy && page.lockedBy !== userId) {
      throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
        message: '非锁定者无法释放页面锁',
      })
    }

    await page.update({ lockedBy: null, lockedAt: null })
    return { locked: false }
  }

  /**
   * 心跳续锁
   * @param {string} pageId - 页面 ID
   * @param {string} userId - 用户 ID
   * @returns {Promise<Object>}
   */
  async heartbeatPageLock(pageId, userId) {
    const page = await DesignPage.findByPk(pageId)

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: 'DesignPage',
        id: pageId,
      })
    }

    if (page.lockedBy && page.lockedBy !== userId) {
      throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
        message: '非锁定者无法续锁',
      })
    }

    const lockedAt = new Date()
    await page.update({ lockedBy: userId, lockedAt })
    return { locked: true, lockedBy: userId, lockedAt }
  }

  /**
   * 强制释放页面锁
   * @param {string} pageId - 页面 ID
   * @returns {Promise<Object>}
   */
  async forceReleasePageLock(pageId) {
    const page = await DesignPage.findByPk(pageId)

    if (!page) {
      throw new AppError(ErrorCodes.DESIGN_PAGE_NOT_FOUND, 404, {
        resource: 'DesignPage',
        id: pageId,
      })
    }

    await page.update({ lockedBy: null, lockedAt: null })
    return { locked: false }
  }

  /**
   * 判断锁是否过期
   * @param {Date | null} lockedAt - 锁定时间
   * @returns {boolean}
   * @private
   */
  _isLockExpired(lockedAt) {
    if (!lockedAt) return false
    const lockedTime = new Date(lockedAt).getTime()
    return Date.now() - lockedTime > LOCK_TIMEOUT_MS
  }

  /**
   * 清理过期锁
   * @param {import('../models').DesignPage} page - 页面记录
   * @returns {Promise<boolean>} 是否清理
   * @private
   */
  async _clearExpiredLock(page) {
    if (!page.lockedBy || !page.lockedAt) return false
    if (!this._isLockExpired(page.lockedAt)) return false

    await page.update({ lockedBy: null, lockedAt: null })
    return true
  }
}

module.exports = new DesignService()
