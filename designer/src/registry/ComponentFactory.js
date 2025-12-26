/**
 * ComponentFactory - 缁勪欢宸ュ巶绫? *
 * 璐熻矗缁勪欢鐨勬敞鍐屻€佽幏鍙栥€佸疄渚嬪寲鍜屾悳绱? * Task 2.1: 鍒涘缓 ComponentFactory 绫? *
 * Requirements:
 * - Requirement 2: 缁勪欢娉ㄥ唽鏈哄埗
 * - Acceptance Criteria 2.1: 瀹炵幇缁勪欢娉ㄥ唽鍜岃幏鍙栧姛鑳? * - Acceptance Criteria 2.2: 鏀寔鎸夊垎绫昏幏鍙栫粍浠? * - Acceptance Criteria 2.3: 鏀寔缁勪欢鎼滅储鍔熻兘
 */

class ComponentFactory {
    constructor() {
        // 瀛樺偍鎵€鏈夋敞鍐岀殑缁勪欢瀹氫箟
        // key: component type, value: component definition
        this.components = new Map();

        // 鎸夊垎绫诲瓨鍌ㄧ粍浠剁被鍨?        // key: category, value: Set of component types
        this.categories = new Map();
    }

    /**
     * 娉ㄥ唽缁勪欢瀹氫箟
     * @param {Object} definition - 缁勪欢瀹氫箟瀵硅薄
     * @param {string} definition.type - 缁勪欢绫诲瀷锛堝敮涓€鏍囪瘑锛?     * @param {string} definition.name - 缁勪欢鏄剧ず鍚嶇О
     * @param {string} definition.category - 缁勪欢鍒嗙被
     * @param {string} definition.icon - 缁勪欢鍥炬爣
     * @param {Object} definition.defaultProps - 榛樿灞炴€?     * @param {Object} definition.defaultStyle - 榛樿鏍峰紡
     * @param {Object} definition.propsSchema - 灞炴€chema瀹氫箟
     * @param {Function|Object} definition.component - Vue缁勪欢
     */
    register(definition) {
        if (!definition || !definition.type) {
            throw new Error('Component definition must have a type');
        }

        if (this.components.has(definition.type)) {
            console.warn(`Component type "${definition.type}" is already registered. Overwriting...`);
        }

        // 褰掍竴鍖栵細琛ュ厖 styleConfig 灞炴€т互绗﹀悎鎸夐挳缁勪欢瑙勮寖
        const normalized = {
            ...definition,
            defaultProps: { ...(definition.defaultProps || {}) },
            propsSchema: { ...(definition.propsSchema || {}) },
        };
        if (normalized.defaultProps.styleConfig === undefined) {
            normalized.defaultProps.styleConfig = '';
        }
        if (!normalized.propsSchema.styleConfig) {
            normalized.propsSchema.styleConfig = {
                type: 'string',
                label: '鏍峰紡閰嶇疆',
                group: '澶栬',
                default: '',
            };
        }

        // 瀛樺偍缁勪欢瀹氫箟
        this.components.set(normalized.type, normalized);

        // 鎸夊垎绫诲瓨鍌?
        const category = normalized.category || 'uncategorized';
        if (!this.categories.has(category)) {
            this.categories.set(category, new Set());
        }
        this.categories.get(category).add(normalized.type);

        console.log(`鉁?Registered component: ${normalized.type} (${category})`);
    }

    /**
     * 鑾峰彇缁勪欢瀹氫箟
     * @param {string} type - 缁勪欢绫诲瀷
     * @returns {Object|null} 缁勪欢瀹氫箟瀵硅薄锛屽鏋滀笉瀛樺湪杩斿洖null
     */
    get(type) {
        return this.components.get(type) || null;
    }

    /**
     * 鍒涘缓缁勪欢瀹炰緥
     * @param {string} type - 缁勪欢绫诲瀷
     * @param {Object} overrides - 瑕嗙洊鐨勫睘鎬у拰鏍峰紡
     * @param {Object} overrides.props - 瑕嗙洊鐨勫睘鎬?     * @param {Object} overrides.style - 瑕嗙洊鐨勬牱寮?     * @returns {Object} 缁勪欢瀹炰緥瀵硅薄
     */
    createInstance(type, overrides = {}) {
        const definition = this.get(type);

        if (!definition) {
            throw new Error(`Component type "${type}" is not registered`);
        }

        // 鐢熸垚鍞竴ID
        const id = crypto.randomUUID();

        // 鍚堝苟榛樿灞炴€у拰瑕嗙洊灞炴€?
        const props = {
            ...definition.defaultProps,
            ...overrides.props,
        };

        // 鍚堝苟榛樿鏍峰紡鍜岃鐩栨牱寮?
        const style = {
            ...definition.defaultStyle,
            ...overrides.style,
        };

        return {
            id,
            type,
            label: overrides.label || definition.name || type,
            locked: typeof overrides.locked === 'boolean' ? overrides.locked : false,
            visible: typeof overrides.visible === 'boolean' ? overrides.visible : true,
            props,
            style,
            bindings: overrides.bindings || {},
            events: overrides.events || {},
            animations: Array.isArray(overrides.animations) ? overrides.animations : [],
            children: Array.isArray(overrides.children) ? overrides.children : [],
        };
    }

    /**
     * 鑾峰彇鎵€鏈夊垎绫诲強鍏剁粍浠?     * @returns {Object} 鍒嗙被瀵硅薄锛宬ey涓哄垎绫诲悕锛寁alue涓虹粍浠跺畾涔夋暟缁?     */
    getAllByCategory() {
        const result = {};

        this.categories.forEach((types, category) => {
            result[category] = Array.from(types).map((type) => this.get(type));
        });

        return result;
    }

    /**
     * 鎼滅储缁勪欢
     * @param {string} keyword - 鎼滅储鍏抽敭璇?     * @returns {Array} 鍖归厤鐨勭粍浠跺畾涔夋暟缁?     */
    search(keyword) {
        if (!keyword || keyword.trim() === '') {
            return [];
        }

        const lowerKeyword = keyword.toLowerCase().trim();
        const results = [];

        this.components.forEach((definition) => {
            // 鎼滅储缁勪欢鍚嶇О銆佺被鍨嬨€佸垎绫?
            const searchableText = [definition.name, definition.type, definition.category, definition.description || ''].join(' ').toLowerCase();

            if (searchableText.includes(lowerKeyword)) {
                results.push(definition);
            }
        });

        return results;
    }

    /**
     * 鑾峰彇鎵€鏈夊凡娉ㄥ唽鐨勭粍浠剁被鍨?     * @returns {Array} 缁勪欢绫诲瀷鏁扮粍
     */
    getAllTypes() {
        return Array.from(this.components.keys());
    }

    /**
     * 鑾峰彇鎵€鏈夊垎绫诲悕绉?     * @returns {Array} 鍒嗙被鍚嶇О鏁扮粍
     */
    getAllCategories() {
        return Array.from(this.categories.keys());
    }

    /**
     * 妫€鏌ョ粍浠剁被鍨嬫槸鍚﹀凡娉ㄥ唽
     * @param {string} type - 缁勪欢绫诲瀷
     * @returns {boolean} 鏄惁宸叉敞鍐?     */
    has(type) {
        return this.components.has(type);
    }

    /**
     * 鍙栨秷娉ㄥ唽缁勪欢
     * @param {string} type - 缁勪欢绫诲瀷
     * @returns {boolean} 鏄惁鎴愬姛鍙栨秷娉ㄥ唽
     */
    unregister(type) {
        const definition = this.get(type);

        if (!definition) {
            return false;
        }

        // 浠庡垎绫讳腑绉婚櫎
        const category = definition.category || 'uncategorized';
        if (this.categories.has(category)) {
            this.categories.get(category).delete(type);

            // 濡傛灉鍒嗙被涓虹┖锛屽垹闄ゅ垎绫?
            if (this.categories.get(category).size === 0) {
                this.categories.delete(category);
            }
        }

        // 浠庣粍浠舵槧灏勪腑绉婚櫎
        this.components.delete(type);

        console.log(`馃棏锔?Unregistered component: ${type}`);
        return true;
    }

    /**
     * 娓呯┖鎵€鏈夋敞鍐岀殑缁勪欢
     */
    clear() {
        this.components.clear();
        this.categories.clear();
        console.log('馃棏锔?Cleared all registered components');
    }

    /**
     * 鑾峰彇缁勪欢鏁伴噺
     * @returns {number} 宸叉敞鍐岀殑缁勪欢鏁伴噺
     */
    getCount() {
        return this.components.size;
    }
}

// 鍒涘缓鍗曚緥瀹炰緥
const componentFactory = new ComponentFactory();

export default componentFactory;
export { ComponentFactory };



