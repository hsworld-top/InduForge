/**
 * Design Store - 设计中心状态管理
 * 管理页面、组件选择和编辑状态
 * Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 6.1, 6.2, 6.5
 */
import { defineStore } from 'pinia'
import { designAPI } from '@/api/design.api'

/**
 * 生成默认的 Page Schema
 * @param {string} name - 页面名称
 * @returns {Object} 默认 Page Schema
 */
function createDefaultPageSchema(name) {
  return {
    version: '2.0.0',
    meta: {
      id: crypto.randomUUID(),
      name,
      description: '',
    },
    config: {
      width: 1920,
      height: 1080,
      scaleMode: 'fit',
      backgroundColor: '#ffffff',
      gridSize: 10,
      snapToGrid: true,
      theme: 'light',
    },
    variables: {},
    dataSources: [],
    components: [],
    permissions: {},
  }
}

/**
 * 递归查找组件
 * @param {Array} components - 组件数组
 * @param {string} componentId - 组件ID
 * @returns {Object|null} 找到的组件或 null
 */
function findComponentById(components, componentId) {
  for (const component of components) {
    if (component.id === componentId) {
      return component
    }
    if (component.children && component.children.length > 0) {
      const found = findComponentById(component.children, componentId)
      if (found) return found
    }
  }
  return null
}

/**
 * 递归查找组件的父组件
 * @param {Array} components - 组件数组
 * @param {string} componentId - 组件ID
 * @param {Object|null} parent - 当前父组件
 * @returns {Object|null} 父组件或 null（表示在根级别）
 */
function findParentComponent(components, componentId, parent = null) {
  for (const component of components) {
    if (component.id === componentId) {
      return parent
    }
    if (component.children && component.children.length > 0) {
      const found = findParentComponent(component.children, componentId, component)
      if (found !== undefined) return found
    }
  }
  return undefined
}

/**
 * 递归删除组件
 * @param {Array} components - 组件数组
 * @param {string} componentId - 要删除的组件ID
 * @returns {boolean} 是否成功删除
 */
function removeComponentById(components, componentId) {
  const index = components.findIndex((c) => c.id === componentId)
  if (index !== -1) {
    components.splice(index, 1)
    return true
  }
  for (const component of components) {
    if (component.children && component.children.length > 0) {
      if (removeComponentById(component.children, componentId)) {
        return true
      }
    }
  }
  return false
}

export const useDesignStore = defineStore('design', {
  state: () => ({
    // 当前项目 ID
    projectId: null,
    // 页面列表
    pages: [],
    // 当前页面 ID
    currentPageId: null,
    // 当前页面 Schema
    currentPage: null,
    // 选中组件 ID
    selectedComponentId: null,
    // 是否有未保存更改
    // Requirements: 6.1
    isDirty: false,
    // 剪贴板
    clipboard: null,
    // 加载状态
    loading: false,
    // 保存状态
    saving: false,
    // 错误信息
    error: null,
  }),

  getters: {
    /**
     * 获取选中的组件
     */
    selectedComponent: (state) => {
      if (!state.currentPage || !state.selectedComponentId) {
        return null
      }
      return findComponentById(state.currentPage.components, state.selectedComponentId)
    },

    /**
     * 获取页面树结构（带层级）
     */
    pageTree: (state) => {
      const buildTree = (parentId = null) => {
        return state.pages
          .filter((page) => page.parentId === parentId)
          .sort((a, b) => (a.sortOrder || 0) - (b.sortOrder || 0))
          .map((page) => ({
            ...page,
            children: page.type === 'folder' ? buildTree(page.id) : [],
          }))
      }
      return buildTree()
    },

    /**
     * 获取当前页面的组件列表
     */
    components: (state) => {
      return state.currentPage?.components || []
    },

    /**
     * 获取当前页面配置
     */
    pageConfig: (state) => {
      return state.currentPage?.config || null
    },
  },

  actions: {
    /**
     * 加载项目页面列表
     * Requirements: 1.1
     * @param {string} projectId - 项目ID
     */
    async loadProject(projectId) {
      this.loading = true
      this.error = null
      try {
        this.projectId = projectId
        const response = await designAPI.getPages(projectId)
        this.pages = response.data || response
        // 重置当前页面状态
        this.currentPageId = null
        this.currentPage = null
        this.selectedComponentId = null
        this.isDirty = false
      } catch (error) {
        this.error = error.message || '加载项目失败'
        throw error
      } finally {
        this.loading = false
      }
    },

    /**
     * 加载页面 Schema
     * Requirements: 1.3
     * @param {string} pageId - 页面ID
     */
    async loadPage(pageId) {
      // 如果有未保存的更改，提示用户
      if (this.isDirty) {
        // 这里可以触发确认对话框，暂时直接加载
        console.warn('有未保存的更改')
      }

      this.loading = true
      this.error = null
      try {
        const response = await designAPI.getPage(pageId)
        this.currentPageId = pageId
        this.currentPage = response.data || response
        this.selectedComponentId = null
        this.isDirty = false
      } catch (error) {
        this.error = error.message || '加载页面失败'
        throw error
      } finally {
        this.loading = false
      }
    },

    /**
     * 保存当前页面
     * Requirements: 6.2
     */
    async savePage() {
      if (!this.currentPageId || !this.currentPage) {
        throw new Error('没有可保存的页面')
      }

      this.saving = true
      this.error = null
      try {
        await designAPI.updatePage(this.currentPageId, this.currentPage)
        // Requirements: 6.5 - 保存成功后清除未保存标记
        this.isDirty = false
      } catch (error) {
        this.error = error.message || '保存页面失败'
        throw error
      } finally {
        this.saving = false
      }
    },

    /**
     * 创建新页面
     * Requirements: 1.2
     * @param {string} name - 页面名称
     * @param {string} parentId - 父页面ID（可选）
     * @param {string} type - 页面类型 ('page' | 'folder')
     * @returns {Promise<Object>} 创建的页面
     */
    async createPage(name, parentId = null, type = 'page') {
      if (!this.projectId) {
        throw new Error('未选择项目')
      }

      this.loading = true
      this.error = null
      try {
        const data = {
          name,
          type,
          parentId,
          schemaContent: type === 'page' ? createDefaultPageSchema(name) : null,
        }
        const response = await designAPI.createPage(this.projectId, data)
        const newPage = response.data || response
        this.pages.push(newPage)
        return newPage
      } catch (error) {
        this.error = error.message || '创建页面失败'
        throw error
      } finally {
        this.loading = false
      }
    },

    /**
     * 删除页面
     * Requirements: 1.5
     * @param {string} pageId - 页面ID
     */
    async deletePage(pageId) {
      this.loading = true
      this.error = null
      try {
        await designAPI.deletePage(pageId)
        // 从列表中移除
        this.pages = this.pages.filter((p) => p.id !== pageId)
        // 如果删除的是当前页面，清除当前页面状态
        if (this.currentPageId === pageId) {
          this.currentPageId = null
          this.currentPage = null
          this.selectedComponentId = null
          this.isDirty = false
        }
      } catch (error) {
        this.error = error.message || '删除页面失败'
        throw error
      } finally {
        this.loading = false
      }
    },

    /**
     * 重命名页面
     * Requirements: 1.4
     * @param {string} pageId - 页面ID
     * @param {string} name - 新名称
     */
    async renamePage(pageId, name) {
      this.error = null
      try {
        await designAPI.renamePage(pageId, name)
        // 更新本地状态
        const page = this.pages.find((p) => p.id === pageId)
        if (page) {
          page.name = name
        }
        // 如果是当前页面，同步更新 schema 中的 meta.name
        if (this.currentPageId === pageId && this.currentPage?.meta) {
          this.currentPage.meta.name = name
          this.isDirty = true
        }
      } catch (error) {
        this.error = error.message || '重命名页面失败'
        throw error
      }
    },

    // ==================== 组件操作 ====================

    /**
     * 选择组件
     * @param {string|null} componentId - 组件ID，null 表示取消选择
     */
    selectComponent(componentId) {
      this.selectedComponentId = componentId
    },

    /**
     * 更新组件属性
     * @param {string} componentId - 组件ID
     * @param {Object} updates - 更新的属性 { props?, style?, ... }
     */
    updateComponent(componentId, updates) {
      if (!this.currentPage) return

      const component = findComponentById(this.currentPage.components, componentId)
      if (!component) return

      // 合并更新
      Object.keys(updates).forEach((key) => {
        if (key === 'props' || key === 'style') {
          component[key] = { ...component[key], ...updates[key] }
        } else {
          component[key] = updates[key]
        }
      })

      // Requirements: 6.1 - 标记为有未保存更改
      this.isDirty = true
    },

    /**
     * 添加组件
     * @param {Object} component - 组件 Schema
     * @param {string} parentId - 父组件ID（可选，null 表示添加到根级别）
     */
    addComponent(component, parentId = null) {
      if (!this.currentPage) return

      // 确保组件有 ID
      if (!component.id) {
        component.id = crypto.randomUUID()
      }

      if (parentId) {
        const parent = findComponentById(this.currentPage.components, parentId)
        if (parent) {
          if (!parent.children) {
            parent.children = []
          }
          parent.children.push(component)
        }
      } else {
        this.currentPage.components.push(component)
      }

      // Requirements: 6.1 - 标记为有未保存更改
      this.isDirty = true
    },

    /**
     * 删除组件
     * @param {string} componentId - 组件ID
     */
    removeComponent(componentId) {
      if (!this.currentPage) return

      const removed = removeComponentById(this.currentPage.components, componentId)
      if (removed) {
        // 如果删除的是选中的组件，取消选择
        if (this.selectedComponentId === componentId) {
          this.selectedComponentId = null
        }
        // Requirements: 6.1 - 标记为有未保存更改
        this.isDirty = true
      }
    },

    /**
     * 移动组件
     * @param {string} componentId - 组件ID
     * @param {string|null} targetParentId - 目标父组件ID（null 表示移动到根级别）
     * @param {number} index - 目标位置索引
     */
    moveComponent(componentId, targetParentId, index) {
      if (!this.currentPage) return

      // 找到组件
      const component = findComponentById(this.currentPage.components, componentId)
      if (!component) return

      // 从原位置删除
      removeComponentById(this.currentPage.components, componentId)

      // 添加到新位置
      let targetArray
      if (targetParentId) {
        const targetParent = findComponentById(this.currentPage.components, targetParentId)
        if (!targetParent) return
        if (!targetParent.children) {
          targetParent.children = []
        }
        targetArray = targetParent.children
      } else {
        targetArray = this.currentPage.components
      }

      // 插入到指定位置
      targetArray.splice(index, 0, component)

      // Requirements: 6.1 - 标记为有未保存更改
      this.isDirty = true
    },

    /**
     * 清除错误状态
     */
    clearError() {
      this.error = null
    },

    /**
     * 重置 store 状态
     */
    reset() {
      this.projectId = null
      this.pages = []
      this.currentPageId = null
      this.currentPage = null
      this.selectedComponentId = null
      this.isDirty = false
      this.clipboard = null
      this.loading = false
      this.saving = false
      this.error = null
    },
  },
})

export default useDesignStore
