/**
 * DataBinder - 数据绑定管理器
 *
 * 管理组件属性与数据源之间的绑定关系
 */
import { watch } from "vue";
import expressionEngine from "./ExpressionEngine";

export class DataBinder {
  constructor(store) {
    this.store = store;
    this.watchers = new Map();
    this.bindings = new Map();
  }

  /**
   * 绑定组件属性到表达式
   * @param {string} componentId - 组件ID
   * @param {Object} bindings - 绑定配置 { 'props.value': '{{ data.ds_temp.value }}' }
   */
  bind(componentId, bindings) {
    if (!bindings || Object.keys(bindings).length === 0) {
      return;
    }

    // 保存绑定配置
    this.bindings.set(componentId, bindings);

    // 为每个绑定创建 watcher
    Object.entries(bindings).forEach(([path, expression]) => {
      const watcherKey = `${componentId}:${path}`;

      // 如果已存在，先取消
      if (this.watchers.has(watcherKey)) {
        this.watchers.get(watcherKey)();
        this.watchers.delete(watcherKey);
      }

      // 创建新的 watcher
      const watcher = watch(
        () => this.evaluateExpression(expression),
        (newValue) => {
          this.updateComponentProperty(componentId, path, newValue);
        },
        {
          immediate: true,
          deep: false,
        }
      );

      this.watchers.set(watcherKey, watcher);
    });
  }

  /**
   * 解绑组件的所有绑定
   * @param {string} componentId - 组件ID
   */
  unbind(componentId) {
    // 取消所有相关的 watcher
    for (const [key, watcher] of this.watchers.entries()) {
      if (key.startsWith(componentId + ":")) {
        watcher();
        this.watchers.delete(key);
      }
    }

    // 删除绑定配置
    this.bindings.delete(componentId);
  }

  /**
   * 重新绑定组件（用于更新绑定配置）
   * @param {string} componentId - 组件ID
   * @param {Object} bindings - 新的绑定配置
   */
  rebind(componentId, bindings) {
    this.unbind(componentId);
    this.bind(componentId, bindings);
  }

  /**
   * 计算表达式
   * @param {string} expression - 表达式字符串
   * @returns {any} 计算结果
   */
  evaluateExpression(expression) {
    // 构建上下文
    const context = this.buildContext();

    // 设置上下文并计算
    expressionEngine.setContext(context);
    return expressionEngine.evaluate(expression);
  }

  /**
   * 构建表达式上下文
   * @returns {Object} 上下文对象
   */
  buildContext() {
    const currentPage = this.store.currentPage;

    return {
      // 页面变量
      vars: currentPage?.variables || {},

      // 数据源数据
      data: this.store.dataSources || {},

      // 当前用户信息
      $user: this.store.user || {
        id: null,
        name: "Guest",
        role: "viewer",
        permissions: [],
      },

      // 路由信息
      $route: {
        params: {},
        query: {},
      },

      // 环境变量
      $env: {
        API_BASE: import.meta.env.VITE_API_BASE || "",
        MODE: import.meta.env.MODE,
      },

      // 全局变量（预留）
      $global: {},
    };
  }

  /**
   * 更新组件属性
   * @param {string} componentId - 组件ID
   * @param {string} path - 属性路径，如 'props.value' 或 'style.color'
   * @param {any} value - 新值
   */
  updateComponentProperty(componentId, path, value) {
    const [section, ...keys] = path.split(".");
    const key = keys.join(".");

    // 构建更新对象
    const updates = {};
    if (keys.length === 1) {
      updates[section] = { [key]: value };
    } else {
      // 处理嵌套属性
      updates[section] = this.setNestedProperty({}, keys, value);
    }

    // 更新组件
    this.store.updateComponent(componentId, updates);
  }

  /**
   * 设置嵌套属性
   * @param {Object} obj - 目标对象
   * @param {Array} keys - 键路径数组
   * @param {any} value - 值
   * @returns {Object} 更新后的对象
   */
  setNestedProperty(obj, keys, value) {
    if (keys.length === 1) {
      obj[keys[0]] = value;
      return obj;
    }

    const [first, ...rest] = keys;
    obj[first] = this.setNestedProperty(obj[first] || {}, rest, value);
    return obj;
  }

  /**
   * 清理所有绑定
   */
  clear() {
    // 取消所有 watcher
    for (const watcher of this.watchers.values()) {
      watcher();
    }
    this.watchers.clear();
    this.bindings.clear();
  }

  /**
   * 获取组件的绑定配置
   * @param {string} componentId - 组件ID
   * @returns {Object|null} 绑定配置
   */
  getBindings(componentId) {
    return this.bindings.get(componentId) || null;
  }
}

export default DataBinder;
