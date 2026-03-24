/**
 * 组件注册表
 * 管理可用组件类型的注册和查询
 */

/**
 * @typedef {Object} EventDefinition
 * 事件定义
 * @property {string} name - 事件名称
 * @property {string} label - 事件显示名称
 * @property {string} [description] - 事件说明
 */

/**
 * @typedef {Object} ComponentManifest
 * 组件清单
 * @property {string} type - 组件类型标识
 * @property {string} name - 组件显示名称
 * @property {string} category - 组件分类
 * @property {string} [icon] - 组件图标
 * @property {string} [description] - 组件描述
 * @property {Object} defaultProps - 默认属性
 * @property {Object} defaultStyle - 默认样式
 * @property {{ width?: number, height?: number }} [defaultSize] - 默认尺寸
 * @property {Object} propsSchema - 属性 Schema（用于属性面板）
 * @property {Object} [styleSchema] - 样式 Schema
 * @property {Array<string | EventDefinition>} [events] - 支持的事件列表
 * @property {boolean} [isContainer] - 是否是容器组件
 * @property {string[]} [allowedChildren] - 允许的子组件类型
 * @property {Object} [slots] - 插槽定义
 */

/**
 * 组件分类
 * @enum {string}
 */
export const ComponentCategory = {
  /** 基础组件 */
  BASIC: "basic",
  /** 容器组件 */
  CONTAINER: "container",
  /** 表单组件 */
  FORM: "form",
  /** 数据展示 */
  DATA: "data",
  /** 图表组件 */
  CHART: "chart",
  /** 导航组件 */
  NAVIGATION: "navigation",
  /** 布局组件 */
  LAYOUT: "layout",
  /** 媒体组件 */
  MEDIA: "media",
  /** 自定义组件 */
  CUSTOM: "custom",
};

/**
 * 组件注册表类
 */
export class ComponentRegistry {
  constructor() {
    /** @type {Map<string, ComponentManifest>} */
    this._manifests = new Map();

    /** @type {Map<string, ComponentManifest[]>} */
    this._byCategory = new Map();
  }

  /**
   * 注册组件
   * @param {ComponentManifest} manifest - 组件清单
   */
  register(manifest) {
    if (!manifest.type) {
      throw new Error("组件清单缺少 type 字段");
    }

    if (this._manifests.has(manifest.type)) {
      console.warn(`组件 ${manifest.type} 已存在，将被覆盖`);
    }

    this._manifests.set(manifest.type, manifest);

    // 更新分类索引
    const category = manifest.category || ComponentCategory.CUSTOM;
    if (!this._byCategory.has(category)) {
      this._byCategory.set(category, []);
    }
    this._byCategory.get(category).push(manifest);
  }

  /**
   * 批量注册组件
   * @param {ComponentManifest[]} manifests - 组件清单列表
   */
  registerAll(manifests) {
    for (const manifest of manifests) {
      this.register(manifest);
    }
  }

  /**
   * 获取组件清单
   * @param {string} type - 组件类型
   * @returns {ComponentManifest | undefined}
   */
  get(type) {
    return this._manifests.get(type);
  }

  /**
   * 检查组件是否已注册
   * @param {string} type - 组件类型
   * @returns {boolean}
   */
  has(type) {
    return this._manifests.has(type);
  }

  /**
   * 获取所有组件清单
   * @returns {ComponentManifest[]}
   */
  getAll() {
    return Array.from(this._manifests.values());
  }

  /**
   * 按分类获取组件
   * @param {string} category - 分类
   * @returns {ComponentManifest[]}
   */
  getByCategory(category) {
    return this._byCategory.get(category) || [];
  }

  /**
   * 获取所有分类
   * @returns {string[]}
   */
  getCategories() {
    return Array.from(this._byCategory.keys());
  }

  /**
   * 搜索组件
   * @param {string} keyword - 关键词
   * @returns {ComponentManifest[]}
   */
  search(keyword) {
    const lowerKeyword = keyword.toLowerCase();
    return this.getAll().filter(
      (m) =>
        m.type.toLowerCase().includes(lowerKeyword) ||
        m.name.toLowerCase().includes(lowerKeyword) ||
        m.description?.toLowerCase().includes(lowerKeyword),
    );
  }

  /**
   * 获取组件的默认节点配置
   * @param {string} type - 组件类型
   * @returns {Object | null}
   */
  getDefaultNode(type) {
    const manifest = this._manifests.get(type);
    if (!manifest) return null;

    return {
      type,
      label: manifest.name,
      props: { ...manifest.defaultProps },
      style: { ...manifest.defaultStyle },
      children: [],
    };
  }

  /**
   * 注销组件
   * @param {string} type - 组件类型
   */
  unregister(type) {
    const manifest = this._manifests.get(type);
    if (manifest) {
      this._manifests.delete(type);

      // 从分类索引中移除
      const category = manifest.category || ComponentCategory.CUSTOM;
      const categoryList = this._byCategory.get(category);
      if (categoryList) {
        const index = categoryList.indexOf(manifest);
        if (index > -1) {
          categoryList.splice(index, 1);
        }
      }
    }
  }

  /**
   * 清空注册表
   */
  clear() {
    this._manifests.clear();
    this._byCategory.clear();
  }
}

// 导出单例
export const componentRegistry = new ComponentRegistry();

export default ComponentRegistry;
