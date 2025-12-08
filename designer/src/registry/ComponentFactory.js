/**
 * ComponentFactory - 组件工厂类
 *
 * 负责组件的注册、获取、实例化和搜索
 * Task 2.1: 创建 ComponentFactory 类
 *
 * Requirements:
 * - Requirement 2: 组件注册机制
 * - Acceptance Criteria 2.1: 实现组件注册和获取功能
 * - Acceptance Criteria 2.2: 支持按分类获取组件
 * - Acceptance Criteria 2.3: 支持组件搜索功能
 */

class ComponentFactory {
    constructor() {
        // 存储所有注册的组件定义
        // key: component type, value: component definition
        this.components = new Map();

        // 按分类存储组件类型
        // key: category, value: Set of component types
        this.categories = new Map();
    }

    /**
     * 注册组件定义
     * @param {Object} definition - 组件定义对象
     * @param {string} definition.type - 组件类型（唯一标识）
     * @param {string} definition.name - 组件显示名称
     * @param {string} definition.category - 组件分类
     * @param {string} definition.icon - 组件图标
     * @param {Object} definition.defaultProps - 默认属性
     * @param {Object} definition.defaultStyle - 默认样式
     * @param {Object} definition.propsSchema - 属性schema定义
     * @param {Function|Object} definition.component - Vue组件
     */
    register(definition) {
        if (!definition || !definition.type) {
            throw new Error('Component definition must have a type');
        }

        if (this.components.has(definition.type)) {
            console.warn(`Component type "${definition.type}" is already registered. Overwriting...`);
        }

        // 存储组件定义
        this.components.set(definition.type, definition);

        // 按分类存储
        const category = definition.category || 'uncategorized';
        if (!this.categories.has(category)) {
            this.categories.set(category, new Set());
        }
        this.categories.get(category).add(definition.type);

        console.log(`✅ Registered component: ${definition.type} (${category})`);
    }

    /**
     * 获取组件定义
     * @param {string} type - 组件类型
     * @returns {Object|null} 组件定义对象，如果不存在返回null
     */
    get(type) {
        return this.components.get(type) || null;
    }

    /**
     * 创建组件实例
     * @param {string} type - 组件类型
     * @param {Object} overrides - 覆盖的属性和样式
     * @param {Object} overrides.props - 覆盖的属性
     * @param {Object} overrides.style - 覆盖的样式
     * @returns {Object} 组件实例对象
     */
    createInstance(type, overrides = {}) {
        const definition = this.get(type);

        if (!definition) {
            throw new Error(`Component type "${type}" is not registered`);
        }

        // 生成唯一ID
        const id = `${type}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

        // 合并默认属性和覆盖属性
        const props = {
            ...definition.defaultProps,
            ...overrides.props,
        };

        // 合并默认样式和覆盖样式
        const style = {
            ...definition.defaultStyle,
            ...overrides.style,
        };

        return {
            id,
            type,
            props,
            style,
            children: overrides.children || [],
        };
    }

    /**
     * 获取所有分类及其组件
     * @returns {Object} 分类对象，key为分类名，value为组件定义数组
     */
    getAllByCategory() {
        const result = {};

        this.categories.forEach((types, category) => {
            result[category] = Array.from(types).map((type) => this.get(type));
        });

        return result;
    }

    /**
     * 搜索组件
     * @param {string} keyword - 搜索关键词
     * @returns {Array} 匹配的组件定义数组
     */
    search(keyword) {
        if (!keyword || keyword.trim() === '') {
            return [];
        }

        const lowerKeyword = keyword.toLowerCase().trim();
        const results = [];

        this.components.forEach((definition) => {
            // 搜索组件名称、类型、分类
            const searchableText = [definition.name, definition.type, definition.category, definition.description || ''].join(' ').toLowerCase();

            if (searchableText.includes(lowerKeyword)) {
                results.push(definition);
            }
        });

        return results;
    }

    /**
     * 获取所有已注册的组件类型
     * @returns {Array} 组件类型数组
     */
    getAllTypes() {
        return Array.from(this.components.keys());
    }

    /**
     * 获取所有分类名称
     * @returns {Array} 分类名称数组
     */
    getAllCategories() {
        return Array.from(this.categories.keys());
    }

    /**
     * 检查组件类型是否已注册
     * @param {string} type - 组件类型
     * @returns {boolean} 是否已注册
     */
    has(type) {
        return this.components.has(type);
    }

    /**
     * 取消注册组件
     * @param {string} type - 组件类型
     * @returns {boolean} 是否成功取消注册
     */
    unregister(type) {
        const definition = this.get(type);

        if (!definition) {
            return false;
        }

        // 从分类中移除
        const category = definition.category || 'uncategorized';
        if (this.categories.has(category)) {
            this.categories.get(category).delete(type);

            // 如果分类为空，删除分类
            if (this.categories.get(category).size === 0) {
                this.categories.delete(category);
            }
        }

        // 从组件映射中移除
        this.components.delete(type);

        console.log(`🗑️ Unregistered component: ${type}`);
        return true;
    }

    /**
     * 清空所有注册的组件
     */
    clear() {
        this.components.clear();
        this.categories.clear();
        console.log('🗑️ Cleared all registered components');
    }

    /**
     * 获取组件数量
     * @returns {number} 已注册的组件数量
     */
    getCount() {
        return this.components.size;
    }
}

// 创建单例实例
const componentFactory = new ComponentFactory();

export default componentFactory;
export { ComponentFactory };
