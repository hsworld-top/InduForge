/**
 * Design Store - 设计中心状态管理
 * 管理页面、组件选择和编辑状态
 * Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 6.1, 6.2, 6.5
 */
import { defineStore } from 'pinia';
import { designAPI } from '@/api/design.api';
import { useHistory } from '@/composables/useHistory';
import { DataBinder } from '@/engine/binding/DataBinder';
import { AnimationManager } from '@/engine/animation/AnimationManager';
import { DataSourceManager } from '@/engine/datasource/DataSourceManager';

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
        events: {},
        permissions: {
            roles: [],
            componentAcl: [],
        },
    };
}

function normalizeComponentSchema(component) {
    if (!component || typeof component !== 'object') return;
    if (!component.id || typeof component.id !== 'string') {
        component.id = crypto.randomUUID();
    }
    if (!component.type || typeof component.type !== 'string') {
        component.type = 'Unknown';
    }
    if (!component.label || typeof component.label !== 'string') {
        component.label = component.name || component.type;
    }
    if (typeof component.locked !== 'boolean') {
        component.locked = false;
    }
    if (typeof component.visible !== 'boolean') {
        component.visible = true;
    }
    if (!component.style || typeof component.style !== 'object') {
        component.style = {};
    }
    if (!component.props || typeof component.props !== 'object') {
        component.props = {};
    }
    if (!component.bindings || typeof component.bindings !== 'object') {
        component.bindings = {};
    }
    if (!component.events || typeof component.events !== 'object') {
        component.events = {};
    }
    if (!Array.isArray(component.animations)) {
        component.animations = [];
    }
    if (!Array.isArray(component.children)) {
        component.children = [];
    }
    component.children.forEach(normalizeComponentSchema);
}

function normalizePageSchema(page) {
    if (!page || typeof page !== 'object') return;
    if (typeof page.version !== 'string') {
        page.version = '2.0.0';
    }
    if (!page.meta || typeof page.meta !== 'object') {
        page.meta = {};
    }
    if (!page.meta.id || typeof page.meta.id !== 'string') {
        page.meta.id = crypto.randomUUID();
    }
    if (!page.meta.name || typeof page.meta.name !== 'string') {
        page.meta.name = '页面';
    }
    if (typeof page.meta.description !== 'string') {
        page.meta.description = '';
    }
    if (!page.config || typeof page.config !== 'object') {
        page.config = {};
    }
    if (typeof page.config.width !== 'number') {
        page.config.width = 1920;
    }
    if (typeof page.config.height !== 'number') {
        page.config.height = 1080;
    }
    if (typeof page.config.gridSize !== 'number') {
        page.config.gridSize = 10;
    }
    if (typeof page.config.snapToGrid !== 'boolean') {
        page.config.snapToGrid = true;
    }
    if (!page.config.backgroundColor || typeof page.config.backgroundColor !== 'string') {
        page.config.backgroundColor = '#ffffff';
    }
    if (!['fit', 'fill', 'fixed'].includes(page.config.scaleMode)) {
        page.config.scaleMode = 'fit';
    }
    if (!['light', 'dark'].includes(page.config.theme)) {
        page.config.theme = 'light';
    }
    if (!page.variables || typeof page.variables !== 'object' || Array.isArray(page.variables)) {
        page.variables = {};
    }
    if (!Array.isArray(page.dataSources)) {
        page.dataSources = [];
    }
    if (!Array.isArray(page.components)) {
        page.components = [];
    }
    page.components.forEach(normalizeComponentSchema);
    if (!page.permissions || typeof page.permissions !== 'object') {
        page.permissions = {};
    }
    if (!Array.isArray(page.permissions.roles)) {
        page.permissions.roles = [];
    }
    if (!Array.isArray(page.permissions.componentAcl)) {
        page.permissions.componentAcl = [];
    }
    if (!page.events || typeof page.events !== 'object') {
        page.events = {};
    }
    if (page.variableMeta && typeof page.variableMeta === 'object') {
        Object.entries(page.variableMeta).forEach(([name, meta]) => {
            if (!name || Object.prototype.hasOwnProperty.call(page.variables, name)) return;
            if (meta && Object.prototype.hasOwnProperty.call(meta, 'defaultValue')) {
                page.variables[name] = meta.defaultValue;
            }
        });
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
            return component;
        }
        if (component.children && component.children.length > 0) {
            const found = findComponentById(component.children, componentId);
            if (found) return found;
        }
    }
    return null;
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
            return parent;
        }
        if (component.children && component.children.length > 0) {
            const found = findParentComponent(component.children, componentId, component);
            if (found !== undefined) return found;
        }
    }
    return undefined;
}

/**
 * 递归删除组件
 * @param {Array} components - 组件数组
 * @param {string} componentId - 要删除的组件ID
 * @returns {boolean} 是否成功删除
 */
function removeComponentById(components, componentId) {
    const index = components.findIndex((c) => c.id === componentId);
    if (index !== -1) {
        components.splice(index, 1);
        return true;
    }
    for (const component of components) {
        if (component.children && component.children.length > 0) {
            if (removeComponentById(component.children, componentId)) {
                return true;
            }
        }
    }
    return false;
}

// 创建历史记录管理器
const history = useHistory(50);

// 创建数据绑定管理器（延迟初始化）
let dataBinder = null;

// 创建动画管理器
const animationManager = new AnimationManager();

// 创建数据源管理器（延迟初始化）
let dataSourceManager = null;

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
        // 选中组件 ID（单选）
        selectedComponentId: null,
        // 选中组件 ID 列表（多选）- Task 5.1
        selectedComponentIds: [],
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
        // 数据源数据
        dataSources: {},
        // 数据源配置
        dataSourceConfigs: [],
        // 用户信息
        user: null,
        // 数据源管理器
        dataSourceManager: null,
    }),

    getters: {
        /**
         * 获取选中的组件
         */
        selectedComponent: (state) => {
            if (!state.currentPage || !state.selectedComponentId) {
                return null;
            }
            return findComponentById(state.currentPage.components, state.selectedComponentId);
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
                    }));
            };
            return buildTree();
        },

        /**
         * 获取当前页面的组件列表
         */
        components: (state) => {
            return state.currentPage?.components || [];
        },

        /**
         * 获取当前页面配置
         */
        pageConfig: (state) => {
            return state.currentPage?.config || null;
        },

        /**
         * 是否可以撤销
         */
        canUndo: () => {
            return history.canUndo.value;
        },

        /**
         * 是否可以重做
         */
        canRedo: () => {
            return history.canRedo.value;
        },
    },

    actions: {
        /**
         * 加载项目页面列表
         * Requirements: 1.1
         * @param {string} projectId - 项目ID
         */
        async loadProject(projectId) {
            this.loading = true;
            this.error = null;
            try {
                this.projectId = projectId;
                const response = await designAPI.getPages(projectId);
                this.pages = response.data || response;
                // 重置当前页面状态
                this.currentPageId = null;
                this.currentPage = null;
                this.selectedComponentId = null;
                this.isDirty = false;
            } catch (error) {
                this.error = error.message || '加载项目失败';
                throw error;
            } finally {
                this.loading = false;
            }
        },

        /**
         * 加载页面 Schema
         * Requirements: 1.3
         * @param {string} pageId - 页面ID
         */
        async loadPage(pageId) {
            if (!this.projectId) {
                throw new Error('未选择项目');
            }

            // 如果有未保存的更改，提示用户
            if (this.isDirty) {
                // 这里可以触发确认对话框，暂时直接加载
                console.warn('有未保存的更改');
            }

            this.loading = true;
            this.error = null;
            try {
                // 清理旧页面的数据源
                if (dataSourceManager) {
                    dataSourceManager.clear();
                }

                const response = await designAPI.getPage(this.projectId, pageId);
                this.currentPageId = pageId;
                this.currentPage = response.data || response;
                normalizePageSchema(this.currentPage);
                if (!this.currentPage.events) {
                    this.currentPage.events = {};
                }
                this.selectedComponentId = null;
                this.isDirty = false;

                // 初始化数据源管理器
                await this.initDataSourceManager();

                // 加载页面的数据源
                if (this.currentPage.dataSources && Array.isArray(this.currentPage.dataSources)) {
                    this.currentPage.dataSources.forEach((dsConfig) => {
                        dataSourceManager.register(dsConfig);
                    });
                }
            } catch (error) {
                this.error = error.message || '加载页面失败';
                throw error;
            } finally {
                this.loading = false;
            }
        },

        /**
         * 保存当前页面
         * Requirements: 6.2
         */
        async savePage() {
            if (!this.projectId || !this.currentPageId || !this.currentPage) {
                throw new Error('没有可保存的页面');
            }

            this.saving = true;
            this.error = null;
            try {
                normalizePageSchema(this.currentPage);
                await designAPI.updatePage(this.projectId, this.currentPageId, this.currentPage);
                // Requirements: 6.5 - 保存成功后清除未保存标记
                this.isDirty = false;
            } catch (error) {
                this.error = error.message || '保存页面失败';
                throw error;
            } finally {
                this.saving = false;
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
                throw new Error('未选择项目');
            }

            this.loading = true;
            this.error = null;
            try {
                const data = {
                    name,
                    type,
                    parentId,
                    schemaContent: type === 'page' ? createDefaultPageSchema(name) : null,
                };
                const response = await designAPI.createPage(this.projectId, data);
                const newPage = response.data || response;
                this.pages.push(newPage);
                return newPage;
            } catch (error) {
                this.error = error.message || '创建页面失败';
                throw error;
            } finally {
                this.loading = false;
            }
        },

        /**
         * 删除页面
         * Requirements: 1.5
         * @param {string} pageId - 页面ID
         */
        async deletePage(pageId) {
            if (!this.projectId) {
                throw new Error('未选择项目');
            }

            this.loading = true;
            this.error = null;
            try {
                await designAPI.deletePage(this.projectId, pageId);
                // 从列表中移除
                this.pages = this.pages.filter((p) => p.id !== pageId);
                // 如果删除的是当前页面，清除当前页面状态
                if (this.currentPageId === pageId) {
                    this.currentPageId = null;
                    this.currentPage = null;
                    this.selectedComponentId = null;
                    this.isDirty = false;
                }
            } catch (error) {
                this.error = error.message || '删除页面失败';
                throw error;
            } finally {
                this.loading = false;
            }
        },

        /**
         * 重命名页面
         * Requirements: 1.4
         * @param {string} pageId - 页面ID
         * @param {string} name - 新名称
         */
        async renamePage(pageId, name) {
            if (!this.projectId) {
                throw new Error('未选择项目');
            }

            this.error = null;
            try {
                await designAPI.renamePage(this.projectId, pageId, name);
                // 更新本地状态
                const page = this.pages.find((p) => p.id === pageId);
                if (page) {
                    page.name = name;
                }
                // 如果是当前页面，同步更新 schema 中的 meta.name
                if (this.currentPageId === pageId && this.currentPage?.meta) {
                    this.currentPage.meta.name = name;
                    this.isDirty = true;
                }
            } catch (error) {
                this.error = error.message || '重命名页面失败';
                throw error;
            }
        },

        /**
         * 移动页面到指定页面组
         * @param {string} pageId - 页面ID
         * @param {string|null} targetGroupId - 目标页面组ID，null 表示移动到根目录
         */
        async movePageToGroup(pageId, targetGroupId) {
            if (!this.projectId) {
                throw new Error('未选择项目');
            }

            this.error = null;
            try {
                await designAPI.movePageToGroup(this.projectId, pageId, targetGroupId);
                // 更新本地状态
                const page = this.pages.find((p) => p.id === pageId);
                if (page) {
                    page.parentId = targetGroupId;
                }
            } catch (error) {
                this.error = error.message || '移动页面失败';
                throw error;
            }
        },

        // ==================== 组件操作 ====================

        /**
         * 选择组件（单选）
         * @param {string|null} componentId - 组件ID，null 表示取消选择
         */
        selectComponent(componentId) {
            this.selectedComponentId = componentId;
            // 单选时清空多选列表
            this.selectedComponentIds = componentId ? [componentId] : [];
        },

        /**
         * 设置多个选中组件
         * Task 5.1 & 5.7: 多选支持
         * @param {string[]} componentIds - 组件ID列表
         */
        selectMultipleComponents(componentIds) {
            this.selectedComponentIds = [...componentIds];
            // 如果只有一个，也设置 selectedComponentId
            this.selectedComponentId = componentIds.length === 1 ? componentIds[0] : null;
        },

        /**
         * 切换组件选中状态（用于 Ctrl + 点击）
         * Task 5.1: 实现多选
         * @param {string} componentId - 组件ID
         */
        toggleComponentSelection(componentId) {
            const index = this.selectedComponentIds.indexOf(componentId);
            if (index > -1) {
                // 已选中，取消选中
                this.selectedComponentIds.splice(index, 1);
            } else {
                // 未选中，添加到选中列表
                this.selectedComponentIds.push(componentId);
            }

            // 更新 selectedComponentId
            this.selectedComponentId =
                this.selectedComponentIds.length === 1 ? this.selectedComponentIds[0] : null;
        },

        /**
         * 全选组件
         * Task 5.1: 实现全选
         */
        selectAllComponents() {
            if (!this.currentPage || !this.currentPage.components) return;

            // 获取所有根级别组件的 ID
            const allIds = this.currentPage.components.map((comp) => comp.id);
            this.selectMultipleComponents(allIds);
        },

        /**
         * 清除所有选择
         * Task 5.1: 清除选择
         */
        clearSelection() {
            this.selectedComponentId = null;
            this.selectedComponentIds = [];
        },

        /**
         * 更新组件属性
         * @param {string} componentId - 组件ID
         * @param {Object} updates - 更新的属性 { props?, style?, ... }
         */
        updateComponent(componentId, updates) {
            if (!this.currentPage) return;

            const component = findComponentById(this.currentPage.components, componentId);
            if (!component) return;

            // 合并更新
            Object.keys(updates).forEach((key) => {
                if (key === 'props' || key === 'style') {
                    component[key] = { ...component[key], ...updates[key] };
                } else {
                    component[key] = updates[key];
                }
            });

            // Requirements: 6.1 - 标记为有未保存更改
            this.isDirty = true;
        },

        /**
         * 更新页面事件
         * @param {string} eventName - 事件名称
         * @param {string} value - 事件脚本
         */
        updatePageEvent(eventName, value) {
            if (!this.currentPage) return;
            if (!this.currentPage.events) {
                this.currentPage.events = {};
            }
            this.currentPage.events = { ...this.currentPage.events, [eventName]: value };
            this.isDirty = true;
        },

        /**
         * 添加组件
         * @param {Object} component - 组件 Schema
         * @param {string} parentId - 父组件ID（可选，null 表示添加到根级别）
         * @param {number} index - 插入位置索引（可选）
         */
        addComponent(component, parentId = null, index = null) {
            if (!this.currentPage) {
                console.warn('No current page, cannot add component');
                return;
            }

            // 确保 components 数组存在
            if (!this.currentPage.components) {
                this.currentPage.components = [];
            }

            // 验证组件数据
            if (!component || typeof component !== 'object') {
                console.error('Invalid component data:', component);
                return;
            }

            // 确保组件有必需字段
            if (!component.id) {
                component.id = crypto.randomUUID();
            }
            if (!component.type) {
                console.error('Component missing type:', component);
                return;
            }
            if (!component.style) {
                component.style = {};
            }
            if (!component.props) {
                component.props = {};
            }

            // 深拷贝组件以避免引用问题
            const componentCopy = JSON.parse(JSON.stringify(component));

            if (parentId) {
                const parent = findComponentById(this.currentPage.components, parentId);
                if (parent) {
                    if (!parent.children) {
                        parent.children = [];
                    }
                    // Task 4.3: 支持插入到指定索引位置
                    if (index !== null && index >= 0 && index <= parent.children.length) {
                        parent.children.splice(index, 0, componentCopy);
                    } else {
                        parent.children.push(componentCopy);
                    }
                }
            } else {
                if (index !== null && index >= 0 && index <= this.currentPage.components.length) {
                    this.currentPage.components.splice(index, 0, componentCopy);
                } else {
                    this.currentPage.components.push(componentCopy);
                }
            }

            // Requirements: 6.1 - 标记为有未保存更改
            this.isDirty = true;

            console.log('✅ Component added:', componentCopy.type, componentCopy.id);
        },

        /**
         * 删除组件
         * @param {string} componentId - 组件ID
         */
        removeComponent(componentId) {
            if (!this.currentPage) return;

            const removed = removeComponentById(this.currentPage.components, componentId);
            if (removed) {
                // 如果删除的是选中的组件，取消选择
                if (this.selectedComponentId === componentId) {
                    this.selectedComponentId = null;
                }
                // Requirements: 6.1 - 标记为有未保存更改
                this.isDirty = true;
            }
        },

        /**
         * 移动组件
         * @param {string} componentId - 组件ID
         * @param {string|null} targetParentId - 目标父组件ID（null 表示移动到根级别）
         * @param {number} index - 目标位置索引
         */
        moveComponent(componentId, targetParentId, index) {
            if (!this.currentPage) return;

            // 找到组件
            const component = findComponentById(this.currentPage.components, componentId);
            if (!component) return;

            // 从原位置删除
            removeComponentById(this.currentPage.components, componentId);

            // 添加到新位置
            let targetArray;
            if (targetParentId) {
                const targetParent = findComponentById(this.currentPage.components, targetParentId);
                if (!targetParent) return;
                if (!targetParent.children) {
                    targetParent.children = [];
                }
                targetArray = targetParent.children;
            } else {
                targetArray = this.currentPage.components;
            }

            // 插入到指定位置
            targetArray.splice(index, 0, component);

            // Requirements: 6.1 - 标记为有未保存更改
            this.isDirty = true;
        },

        /**
         * 保存历史记录
         */
        saveHistory(description = '') {
            if (this.currentPage) {
                history.push(this.currentPage, description);
            }
        },

        /**
         * 撤销操作
         */
        undo() {
            const previousState = history.undo();
            if (previousState) {
                this.currentPage = JSON.parse(JSON.stringify(previousState));
                this.isDirty = true;
            }
        },

        /**
         * 重做操作
         */
        redo() {
            const nextState = history.redo();
            if (nextState) {
                this.currentPage = JSON.parse(JSON.stringify(nextState));
                this.isDirty = true;
            }
        },

        /**
         * 初始化数据绑定器
         */
        initDataBinder() {
            if (!dataBinder) {
                dataBinder = new DataBinder(this);
            }
            return dataBinder;
        },

        /**
         * 初始化数据源管理器
         * @param {Object} options - 配置选项
         * @param {string} options.mode - 模式：'api'（默认）或 'bridge'
         */
        async initDataSourceManager(options = {}) {
            if (!dataSourceManager) {
                // 默认使用API模式，不依赖iframe
                const mode = options.mode || 'api';
                dataSourceManager = new DataSourceManager(this, { mode });
                this.dataSourceManager = dataSourceManager;

                // 初始化
                const success = await dataSourceManager.init(options.iframeId);
                if (!success && mode === 'bridge') {
                    console.warn('DataSourceManager: Failed to initialize in bridge mode, falling back to API mode');
                    // 降级到API模式
                    dataSourceManager = new DataSourceManager(this, { mode: 'api' });
                    this.dataSourceManager = dataSourceManager;
                    await dataSourceManager.init();
                }
            }
            return dataSourceManager;
        },

        /**
         * 绑定组件数据
         */
        bindComponent(componentId, bindings) {
            const binder = this.initDataBinder();
            binder.bind(componentId, bindings);
        },

        /**
         * 解绑组件数据
         */
        unbindComponent(componentId) {
            if (dataBinder) {
                dataBinder.unbind(componentId);
            }
        },

        /**
         * 播放组件动画
         */
        playAnimation(componentId, animationConfig) {
            animationManager.play(componentId, animationConfig, (condition) => {
                if (dataBinder) {
                    return dataBinder.evaluateExpression(condition);
                }
                return true;
            });
        },

        /**
         * 停止组件动画
         */
        stopAnimation(componentId, animationId = null) {
            animationManager.stop(componentId, animationId);
        },

        /**
         * 清除错误状态
         */
        clearError() {
            this.error = null;
        },

        /**
         * 重置 store 状态
         */
        /**
         * 复制选中的组件
         */
        copySelectedComponent() {
            if (!this.selectedComponent) {
                console.warn('No component selected to copy');
                return;
            }

            // 深拷贝组件到剪贴板
            this.clipboard = JSON.parse(JSON.stringify(this.selectedComponent));
            console.log('📋 Component copied to clipboard:', this.clipboard.type);
        },

        /**
         * 粘贴组件
         */
        pasteComponent() {
            if (!this.clipboard) {
                console.warn('Clipboard is empty');
                return;
            }

            if (!this.currentPage) {
                console.warn('No current page');
                return;
            }

            // 创建新组件（生成新ID，偏移位置）
            const newComponent = JSON.parse(JSON.stringify(this.clipboard));
            newComponent.id = crypto.randomUUID();
            newComponent.style.left = (newComponent.style.left || 0) + 20;
            newComponent.style.top = (newComponent.style.top || 0) + 20;

            // 添加到页面
            this.addComponent(newComponent);

            // 选中新组件
            this.selectComponent(newComponent.id);

            console.log('📋 Component pasted:', newComponent.type);
        },

        /**
         * 复制并粘贴选中的组件
         */
        duplicateSelectedComponent() {
            this.copySelectedComponent();
            this.pasteComponent();
        },

        /**
         * 删除选中的组件
         */
        deleteSelectedComponent() {
            if (!this.selectedComponentId) {
                console.warn('No component selected to delete');
                return;
            }

            this.removeComponent(this.selectedComponentId);
            this.selectComponent(null);
            console.log('🗑️ Component deleted');
        },

        /**
         * 批量更新组件属性
         * Task 5.8: 实现批量操作
         * @param {string[]} componentIds - 组件ID列表
         * @param {Object} updates - 更新的属性
         */
        batchUpdateComponents(componentIds, updates) {
            if (!this.currentPage || !componentIds || componentIds.length === 0) return;

            componentIds.forEach((componentId) => {
                this.updateComponent(componentId, updates);
            });

            console.log(`📝 Batch updated ${componentIds.length} components`);
        },

        /**
         * 批量删除组件
         * Task 5.8: 实现批量删除
         * @param {string[]} componentIds - 组件ID列表
         */
        batchDeleteComponents(componentIds) {
            if (!this.currentPage || !componentIds || componentIds.length === 0) return;

            componentIds.forEach((componentId) => {
                this.removeComponent(componentId);
            });

            // 清除选择
            this.clearSelection();

            console.log(`🗑️ Batch deleted ${componentIds.length} components`);
        },

        /**
         * 批量移动组件
         * Task 5.8: 实现批量移动
         * @param {string[]} componentIds - 组件ID列表
         * @param {Object} delta - 移动的偏移量 { x, y }
         */
        batchMoveComponents(componentIds, delta) {
            if (!this.currentPage || !componentIds || componentIds.length === 0) return;

            componentIds.forEach((componentId) => {
                const component = findComponentById(this.currentPage.components, componentId);
                if (component) {
                    const currentLeft = component.style.left || 0;
                    const currentTop = component.style.top || 0;

                    this.updateComponent(componentId, {
                        style: {
                            left: currentLeft + delta.x,
                            top: currentTop + delta.y,
                        },
                    });
                }
            });

            console.log(`↔️ Batch moved ${componentIds.length} components`);
        },

        /**
         * 批量复制组件
         * Task 5.8: 实现批量复制
         * @param {string[]} componentIds - 组件ID列表
         * @returns {string[]} 新组件的ID列表
         */
        batchCopyComponents(componentIds) {
            if (!this.currentPage || !componentIds || componentIds.length === 0) return [];

            const newComponentIds = [];

            componentIds.forEach((componentId) => {
                const component = findComponentById(this.currentPage.components, componentId);
                if (component) {
                    // 深拷贝组件
                    const newComponent = JSON.parse(JSON.stringify(component));
                    newComponent.id = crypto.randomUUID();

                    // 偏移位置
                    newComponent.style.left = (newComponent.style.left || 0) + 20;
                    newComponent.style.top = (newComponent.style.top || 0) + 20;

                    // 添加到页面
                    this.addComponent(newComponent);
                    newComponentIds.push(newComponent.id);
                }
            });

            // 选中新复制的组件
            if (newComponentIds.length > 0) {
                this.selectMultipleComponents(newComponentIds);
            }

            console.log(`📋 Batch copied ${componentIds.length} components`);
            return newComponentIds;
        },

        /**
         * 置于顶层
         */
        bringToFront(componentId) {
            const component = findComponentById(this.currentPage.components, componentId);
            if (!component) return;

            // 获取同级组件
            const parent = findParentComponent(this.currentPage.components, componentId);
            const siblings = parent ? parent.children : this.currentPage.components;

            // 找到最大 zIndex
            const maxZIndex = Math.max(...siblings.map((c) => c.style.zIndex || 0), 0);

            // 更新组件 zIndex
            this.updateComponent(componentId, {
                style: { zIndex: maxZIndex + 1 },
            });

            console.log('⬆️ Component brought to front');
        },

        /**
         * 置于底层
         */
        sendToBack(componentId) {
            const component = findComponentById(this.currentPage.components, componentId);
            if (!component) return;

            // 更新组件 zIndex 为 0
            this.updateComponent(componentId, {
                style: { zIndex: 0 },
            });

            // 获取同级组件并调整其他组件的 zIndex
            const parent = findParentComponent(this.currentPage.components, componentId);
            const siblings = parent ? parent.children : this.currentPage.components;

            siblings.forEach((sibling) => {
                if (sibling.id !== componentId && (sibling.style.zIndex || 0) > 0) {
                    this.updateComponent(sibling.id, {
                        style: { zIndex: (sibling.style.zIndex || 0) + 1 },
                    });
                }
            });

            console.log('⬇️ Component sent to back');
        },

        /**
         * 上移一层
         */
        moveUp(componentId) {
            const component = findComponentById(this.currentPage.components, componentId);
            if (!component) return;

            const currentZIndex = component.style.zIndex || 0;
            this.updateComponent(componentId, {
                style: { zIndex: currentZIndex + 1 },
            });

            console.log('⬆️ Component moved up');
        },

        /**
         * 下移一层
         */
        moveDown(componentId) {
            const component = findComponentById(this.currentPage.components, componentId);
            if (!component) return;

            const currentZIndex = component.style.zIndex || 0;
            if (currentZIndex > 0) {
                this.updateComponent(componentId, {
                    style: { zIndex: currentZIndex - 1 },
                });
            }

            console.log('⬇️ Component moved down');
        },

        reset() {
            this.projectId = null;
            this.pages = [];
            this.currentPageId = null;
            this.currentPage = null;
            this.selectedComponentId = null;
            this.isDirty = false;
            this.clipboard = null;
            this.loading = false;
            this.saving = false;
            this.error = null;
            this.dataSources = {};
            this.dataSourceConfigs = [];

            // 清理历史记录
            history.clear();

            // 清理数据绑定
            if (dataBinder) {
                dataBinder.clear();
            }

            // 清理动画
            animationManager.clear();

            // 清理数据源
            if (dataSourceManager) {
                dataSourceManager.clear();
            }
        },
    },
});

export default useDesignStore;
