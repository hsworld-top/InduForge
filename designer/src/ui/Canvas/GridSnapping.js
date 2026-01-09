/**
 * 网格吸附工具
 * 提供坐标和尺寸的网格吸附功能
 */

/**
 * 网格吸附类
 */
export class GridSnapping {
  /**
   * 创建网格吸附实例
   * @param {number} [gridSize=10] - 网格大小
   */
  constructor(gridSize = 10) {
    /** @type {number} */
    this.gridSize = gridSize;
    /** @type {boolean} */
    this.enabled = true;
  }

  /**
   * 吸附单个数值到网格
   * @param {number} value - 原始值
   * @returns {number} 吸附后的值
   */
  snap(value) {
    if (!this.enabled) return value;
    return Math.round(value / this.gridSize) * this.gridSize;
  }

  /**
   * 吸附位置坐标到网格
   * @param {number} x - X 坐标
   * @param {number} y - Y 坐标
   * @returns {{ x: number, y: number }} 吸附后的坐标
   */
  snapPosition(x, y) {
    if (!this.enabled) {
      return { x, y };
    }

    return {
      x: this.snap(x),
      y: this.snap(y),
    };
  }

  /**
   * 吸附尺寸到网格
   * @param {number} width - 宽度
   * @param {number} height - 高度
   * @returns {{ width: number, height: number }} 吸附后的尺寸
   */
  snapSize(width, height) {
    if (!this.enabled) {
      return { width, height };
    }

    return {
      width: Math.max(this.gridSize, this.snap(width)),
      height: Math.max(this.gridSize, this.snap(height)),
    };
  }

  /**
   * 吸附矩形区域到网格
   * @param {{ x: number, y: number, width: number, height: number }} rect - 矩形区域
   * @returns {{ x: number, y: number, width: number, height: number }} 吸附后的矩形
   */
  snapRect(rect) {
    if (!this.enabled) {
      return rect;
    }

    const position = this.snapPosition(rect.x, rect.y);
    const size = this.snapSize(rect.width, rect.height);

    return {
      x: position.x,
      y: position.y,
      width: size.width,
      height: size.height,
    };
  }

  /**
   * 设置网格大小
   * @param {number} size - 新的网格大小
   */
  setGridSize(size) {
    if (size > 0) {
      this.gridSize = size;
    }
  }

  /**
   * 启用网格吸附
   */
  enable() {
    this.enabled = true;
  }

  /**
   * 禁用网格吸附
   */
  disable() {
    this.enabled = false;
  }

  /**
   * 切换启用状态
   */
  toggle() {
    this.enabled = !this.enabled;
  }
}

/**
 * 创建默认网格吸附实例
 * @param {number} [gridSize=10] - 网格大小
 * @returns {GridSnapping}
 */
export function createGridSnapping(gridSize = 10) {
  return new GridSnapping(gridSize);
}

export default GridSnapping;
