/**
 * MockDataProvider - 设计态 Mock 数据提供者
 * 在设计态为组件提供模拟数据
 */

import { EventEmitter } from "../editor-core/utils/EventEmitter.ts";

/**
 * @typedef {import('./types.js').DatapointStatusInfo} DatapointStatusInfo
 * @typedef {import('./types.js').DatapointValue} DatapointValue
 */

/**
 * Mock 配置
 * @typedef {Object} MockConfig
 * @property {string} path - 数据点路径
 * @property {*} value - Mock 值
 * @property {string} [dataType] - 数据类型
 * @property {boolean} [autoAnimate] - 是否自动动画
 * @property {number} [animateMin] - 动画最小值
 * @property {number} [animateMax] - 动画最大值
 * @property {number} [animateInterval] - 动画间隔（毫秒）
 */

// ==================== 默认值推断 ====================

/**
 * 根据数据类型推断默认 Mock 值
 * @param {string} dataType - 数据类型
 * @returns {*}
 */
function inferDefaultValue(dataType) {
  switch (dataType?.toLowerCase()) {
    case "number":
    case "int":
    case "float":
    case "double":
    case "integer":
      return 0;
    case "boolean":
    case "bool":
      return false;
    case "string":
    case "text":
      return "";
    case "array":
    case "list":
      return [];
    case "object":
    case "json":
      return {};
    case "date":
    case "datetime":
    case "timestamp":
      return new Date().toISOString();
    default:
      return null;
  }
}

/**
 * 根据路径名推断数据类型
 * @param {string} path - 数据点路径
 * @returns {string}
 */
function inferDataType(path) {
  const lowerPath = path.toLowerCase();

  // 数值类型推断
  if (
    lowerPath.includes("temp") ||
    lowerPath.includes("pressure") ||
    lowerPath.includes("speed") ||
    lowerPath.includes("level") ||
    lowerPath.includes("flow") ||
    lowerPath.includes("value") ||
    lowerPath.includes("count") ||
    lowerPath.includes("voltage") ||
    lowerPath.includes("current") ||
    lowerPath.includes("power")
  ) {
    return "number";
  }

  // 布尔类型推断
  if (
    lowerPath.includes("status") ||
    lowerPath.includes("state") ||
    lowerPath.includes("running") ||
    lowerPath.includes("enabled") ||
    lowerPath.includes("active") ||
    lowerPath.includes("alarm") ||
    lowerPath.includes("fault")
  ) {
    return "boolean";
  }

  // 时间类型推断
  if (
    lowerPath.includes("time") ||
    lowerPath.includes("date") ||
    lowerPath.includes("timestamp")
  ) {
    return "datetime";
  }

  // 默认为字符串
  return "string";
}

// ==================== MockDataProvider 类 ====================

/**
 * Mock 数据提供者
 */
export class MockDataProvider extends EventEmitter {
  constructor() {
    super();

    /** @type {Map<string, MockConfig>} Mock 配置 */
    this._configs = new Map();

    /** @type {Map<string, *>} 当前值缓存 */
    this._values = new Map();

    /** @type {Map<string, number>} 动画定时器 */
    this._animationTimers = new Map();

    /** @type {Set<string>} 订阅的路径 */
    this._subscriptions = new Set();
  }

  // ==================== Mock 配置 ====================

  /**
   * 设置 Mock 配置
   * @param {string} path - 数据点路径
   * @param {MockConfig | *} config - Mock 配置或直接值
   */
  setMock(path, config) {
    // 支持直接传值
    const mockConfig =
      config && typeof config === "object" && "path" in config
        ? config
        : { path, value: config };

    this._configs.set(path, mockConfig);
    this._values.set(path, mockConfig.value);

    // 如果启用了动画，启动动画
    if (mockConfig.autoAnimate) {
      this._startAnimation(path, mockConfig);
    }

    // 通知订阅者
    if (this._subscriptions.has(path)) {
      this._notifyChange(path, mockConfig.value);
    }
  }

  /**
   * 批量设置 Mock 配置
   * @param {Record<string, MockConfig | *>} configs - 配置映射
   */
  setMocks(configs) {
    for (const [path, config] of Object.entries(configs)) {
      this.setMock(path, config);
    }
  }

  /**
   * 移除 Mock 配置
   * @param {string} path - 数据点路径
   */
  removeMock(path) {
    this._stopAnimation(path);
    this._configs.delete(path);
    this._values.delete(path);
  }

  /**
   * 清除所有 Mock 配置
   */
  clearMocks() {
    // 停止所有动画
    for (const path of this._animationTimers.keys()) {
      this._stopAnimation(path);
    }
    this._configs.clear();
    this._values.clear();
  }

  /**
   * 获取 Mock 配置
   * @param {string} path - 数据点路径
   * @returns {MockConfig | undefined}
   */
  getMockConfig(path) {
    return this._configs.get(path);
  }

  // ==================== 值获取 ====================

  /**
   * 获取 Mock 值
   * @param {string} path - 数据点路径
   * @param {Object} [binding] - 绑定配置
   * @returns {*}
   */
  getValue(path, binding) {
    // 优先使用 binding 中的 designMock
    if (binding?.designMock !== undefined) {
      return binding.designMock;
    }

    // 使用配置的 Mock 值
    if (this._values.has(path)) {
      return this._values.get(path);
    }

    // 根据配置推断
    const config = this._configs.get(path);
    if (config?.value !== undefined) {
      return config.value;
    }

    // 根据数据类型推断默认值
    const dataType = config?.dataType || inferDataType(path);
    return inferDefaultValue(dataType);
  }

  /**
   * 获取数据点状态（设计态始终返回 active）
   * @param {string} path - 数据点路径
   * @returns {DatapointStatusInfo}
   */
  getStatus(path) {
    const config = this._configs.get(path);
    return {
      path,
      status: "active",
      dataType: config?.dataType || inferDataType(path),
    };
  }

  /**
   * 手动设置值（用于调试）
   * @param {string} path - 数据点路径
   * @param {*} value - 值
   */
  setValue(path, value) {
    this._values.set(path, value);

    if (this._subscriptions.has(path)) {
      this._notifyChange(path, value);
    }
  }

  // ==================== 订阅管理 ====================

  /**
   * 订阅数据点
   * @param {string} path - 数据点路径
   * @param {function(DatapointValue): void} callback - 回调函数
   * @returns {function(): void} 取消订阅函数
   */
  subscribe(path, callback) {
    this._subscriptions.add(path);

    // 添加事件监听
    const handler = (data) => {
      if (data.path === path) {
        callback(data);
      }
    };

    this.on("change", handler);

    // 立即通知当前值
    const currentValue = this.getValue(path);
    callback({
      path,
      value: currentValue,
      timestamp: Date.now(),
    });

    // 返回取消订阅函数
    return () => {
      this.off("change", handler);
      this._subscriptions.delete(path);
    };
  }

  /**
   * 批量订阅
   * @param {string[]} paths - 数据点路径数组
   * @param {function(DatapointValue): void} callback - 回调函数
   * @returns {function(): void} 取消订阅函数
   */
  subscribeMany(paths, callback) {
    const unsubscribes = paths.map((path) => this.subscribe(path, callback));
    return () => {
      unsubscribes.forEach((unsub) => unsub());
    };
  }

  /**
   * 通知值变更
   * @param {string} path - 数据点路径
   * @param {*} value - 值
   * @private
   */
  _notifyChange(path, value) {
    this.emit("change", {
      path,
      value,
      timestamp: Date.now(),
    });
  }

  // ==================== 动画功能 ====================

  /**
   * 启动数值动画
   * @param {string} path - 数据点路径
   * @param {MockConfig} config - 配置
   * @private
   */
  _startAnimation(path, config) {
    // 停止现有动画
    this._stopAnimation(path);

    const { animateMin = 0, animateMax = 100, animateInterval = 1000 } = config;

    const timer = setInterval(() => {
      // 生成随机值
      const value = animateMin + Math.random() * (animateMax - animateMin);
      const roundedValue = Math.round(value * 100) / 100;

      this._values.set(path, roundedValue);

      if (this._subscriptions.has(path)) {
        this._notifyChange(path, roundedValue);
      }
    }, animateInterval);

    this._animationTimers.set(path, timer);
  }

  /**
   * 停止数值动画
   * @param {string} path - 数据点路径
   * @private
   */
  _stopAnimation(path) {
    const timer = this._animationTimers.get(path);
    if (timer) {
      clearInterval(timer);
      this._animationTimers.delete(path);
    }
  }

  // ==================== 预设场景 ====================

  /**
   * 加载预设场景数据
   * @param {string} sceneName - 场景名称
   */
  loadPreset(sceneName) {
    const presets = {
      // 工厂监控场景
      factory: {
        "factory/line1/temp": { value: 45.5, dataType: "number" },
        "factory/line1/pressure": { value: 1.2, dataType: "number" },
        "factory/line1/status": { value: true, dataType: "boolean" },
        "factory/line1/count": { value: 1250, dataType: "number" },
        "factory/line2/temp": { value: 52.3, dataType: "number" },
        "factory/line2/pressure": { value: 1.5, dataType: "number" },
        "factory/line2/status": { value: false, dataType: "boolean" },
        "factory/line2/count": { value: 980, dataType: "number" },
      },

      // 能源监控场景
      energy: {
        "energy/voltage": { value: 220, dataType: "number" },
        "energy/current": { value: 15.5, dataType: "number" },
        "energy/power": { value: 3410, dataType: "number" },
        "energy/consumption": { value: 12580, dataType: "number" },
        "energy/pf": { value: 0.92, dataType: "number" },
      },

      // 环境监控场景
      environment: {
        "env/temperature": { value: 24.5, dataType: "number" },
        "env/humidity": { value: 65, dataType: "number" },
        "env/pm25": { value: 35, dataType: "number" },
        "env/co2": { value: 420, dataType: "number" },
      },

      // 动画演示场景
      animated: {
        "demo/sine": {
          value: 50,
          dataType: "number",
          autoAnimate: true,
          animateMin: 0,
          animateMax: 100,
          animateInterval: 500,
        },
        "demo/random": {
          value: 0,
          dataType: "number",
          autoAnimate: true,
          animateMin: -50,
          animateMax: 50,
          animateInterval: 1000,
        },
      },
    };

    const preset = presets[sceneName];
    if (preset) {
      this.setMocks(preset);
    } else {
      console.warn(`Unknown preset: ${sceneName}`);
    }
  }

  // ==================== 销毁 ====================

  /**
   * 销毁 Provider
   */
  destroy() {
    // 停止所有动画
    for (const path of this._animationTimers.keys()) {
      this._stopAnimation(path);
    }

    // 清理状态
    this._configs.clear();
    this._values.clear();
    this._subscriptions.clear();

    // 移除所有事件监听
    this.removeAllListeners();
  }
}

/**
 * 创建 Mock 数据提供者实例
 * @returns {MockDataProvider}
 */
export function createMockDataProvider() {
  return new MockDataProvider();
}

export default MockDataProvider;
