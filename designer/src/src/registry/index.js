/**
 * Component Registry - 组件注册中心
 *
 * 职责：
 * - 管理所有组件的注册和获取
 * - 提供组件定义查询接口
 * - 支持组件搜索和分类
 *
 * Requirements:
 * - Requirement 12: 组件注册机制
 *
 * Task 2.3: 重构现有组件注册使用 ComponentFactory
 */

import componentFactory from './ComponentFactory.js';

/**
 * 注册组件
 *
 * @param {Object} definition - 组件定义
 * @param {string} definition.type - 组件类型
 * @param {string} definition.name - 组件名称
 * @param {string} definition.category - 组件分类
 * @param {Component} definition.component - Vue 组件
 * @param {boolean} definition.container - 是否为容器组件
 * @param {Object} definition.defaultProps - 默认属性
 * @param {Object} definition.defaultStyle - 默认样式
 *
 * @example
 * register({
 *   type: 'Button',
 *   name: '按钮',
 *   category: 'basic',
 *   component: ButtonComponent,
 *   container: false,
 *   defaultProps: { text: '按钮' },
 *   defaultStyle: { width: 80, height: 32 }
 * })
 */
export function register(definition) {
    componentFactory.register(definition);
}

/**
 * 获取组件定义
 *
 * @param {string} type - 组件类型
 * @returns {Object|null} 组件定义或 null
 *
 * @example
 * const definition = getComponent('Button')
 * console.log(definition.name) // '按钮'
 */
export function getComponent(type) {
    return componentFactory.get(type);
}

/**
 * 获取所有组件
 *
 * @returns {Array} 所有组件定义
 */
export function getAllComponents() {
    return componentFactory.getAllTypes().map((type) => componentFactory.get(type));
}

/**
 * 获取所有组件（按分类）
 *
 * @returns {Object} 按分类分组的组件
 *
 * @example
 * const categories = getAllComponentsByCategory()
 * console.log(categories.layout) // [Container, Row, Col, ...]
 */
export function getAllComponentsByCategory() {
    return componentFactory.getAllByCategory();
}

/**
 * 搜索组件
 *
 * @param {string} keyword - 搜索关键词
 * @returns {Array} 匹配的组件列表
 *
 * @example
 * const results = searchComponents('按钮')
 * console.log(results) // [{ type: 'Button', name: '按钮', ... }]
 */
export function searchComponents(keyword) {
    if (!keyword) {
        return getAllComponents();
    }

    return componentFactory.search(keyword);
}

/**
 * 创建组件实例
 *
 * @param {string} type - 组件类型
 * @param {Object} overrides - 覆盖的属性
 * @returns {Object|null} 组件实例或 null
 *
 * @example
 * const instance = createComponentInstance('Button', {
 *   props: { text: '确定' },
 *   style: { left: 100, top: 100 }
 * })
 */
export function createComponentInstance(type, overrides = {}) {
    try {
        const instance = componentFactory.createInstance(type, overrides);

        // 添加额外的字段以保持向后兼容
        return {
            ...instance,
            events: overrides.events || {},
            bindings: overrides.bindings || {},
        };
    } catch (error) {
        console.warn(`[Registry] ${error.message}`);
        return null;
    }
}

/**
 * 注册所有组件
 * 这个函数将在应用初始化时调用
 *
 * Task 2.3: 重构现有组件注册
 */
export function registerAllComponents() {
    console.log('[Registry] Registering all components...');

    // 导入布局组件
    import('./layout/index.js').then((module) => {
        const layoutComponents = module.default;
        layoutComponents.forEach((component) => {
            register(component);
        });
        console.log(`[Registry] Registered ${layoutComponents.length} layout components`);
    });

    // 导入基础组件
    import('./basic/index.js').then((module) => {
        const basicComponents = module.default;
        basicComponents.forEach((component) => {
            register(component);
        });
        console.log(`[Registry] Registered ${basicComponents.length} basic components`);
    });

    // 导入 UI 组件
    import('./ui/index.js').then((module) => {
        const uiComponents = module.default;
        uiComponents.forEach((component) => {
            register(component);
        });
        console.log(`[Registry] Registered ${uiComponents.length} UI components`);
    });

    // 导入图表组件
    import('./charts/index.js').then((module) => {
        const chartComponents = module.default;
        chartComponents.forEach((component) => {
            register(component);
        });
        console.log(`[Registry] Registered ${chartComponents.length} chart components`);
    });

    console.log(`[Registry] Total registered: ${componentFactory.getCount()} components`);
}

/**
 * 同步注册所有组件（用于测试和开发）
 *
 * @param {Array} layoutComponents - 布局组件数组
 * @param {Array} basicComponents - 基础组件数组
 * @param {Array} uiComponents - UI组件数组
 * @param {Array} chartComponents - 图表组件数组
 */
export function registerAllComponentsSync(layoutComponents = [], basicComponents = [], uiComponents = [], chartComponents = []) {
    console.log('[Registry] Registering all components (sync)...');

    // 注册布局组件
    layoutComponents.forEach((component) => {
        register(component);
    });

    // 注册基础组件
    basicComponents.forEach((component) => {
        register(component);
    });

    // 注册 UI 组件
    uiComponents.forEach((component) => {
        register(component);
    });

    // 注册图表组件
    chartComponents.forEach((component) => {
        register(component);
    });

    console.log(`[Registry] Total registered: ${componentFactory.getCount()} components`);
    console.log(`[Registry] - Layout: ${layoutComponents.length}`);
    console.log(`[Registry] - Basic: ${basicComponents.length}`);
    console.log(`[Registry] - UI: ${uiComponents.length}`);
    console.log(`[Registry] - Chart: ${chartComponents.length}`);
}

/**
 * 获取所有分类
 *
 * @returns {Array} 分类名称数组
 */
export function getAllCategories() {
    return componentFactory.getAllCategories();
}

/**
 * 获取所有组件（按分类）- 别名函数
 *
 * @returns {Object} 按分类分组的组件
 */
export function getComponentsByCategory() {
    return getAllComponentsByCategory();
}

/**
 * 注册组件 - 别名函数
 *
 * @param {Object} definition - 组件定义
 */
export function registerComponent(definition) {
    return register(definition);
}

/**
 * 清空注册表
 */
export function clearRegistry() {
    componentFactory.clear();
}

// 导出 ComponentFactory 实例（用于调试和高级用法）
export { componentFactory };
