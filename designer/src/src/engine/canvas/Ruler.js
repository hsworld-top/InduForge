/**
 * Ruler.js - Canvas 标尺
 * Task 7.3: 实现标尺（Canvas Layer）
 *
 * Features:
 * - 水平标尺（顶部）
 * - 垂直标尺（左侧）
 * - 刻度显示（每 10px 小刻度，每 50px 大刻度）
 * - 同步缩放和滚动
 * - 鼠标位置指示线
 */
import Konva from 'konva';

export default class Ruler {
  /**
   * @param {Konva.Layer} layer - Konva Layer
   * @param {Object} options - 配置选项
   */
  constructor(layer, options = {}) {
    this.layer = layer;
    this.options = {
      thickness: 20, // 标尺厚度
      bgColor: '#f5f5f5', // 背景色
      lineColor: '#999', // 刻度线颜色
      textColor: '#666', // 文字颜色
      fontSize: 10, // 字体大小
      smallStep: 10, // 小刻度间距（px）
      largeStep: 50, // 大刻度间距（px）
      ...options,
    };

    this.zoom = 1;
    this.scrollX = 0;
    this.scrollY = 0;
    this.mouseX = null;
    this.mouseY = null;

    this.init();
  }

  /**
   * 初始化标尺
   */
  init() {
    // 创建标尺组
    this.group = new Konva.Group({
      name: 'ruler',
    });

    // 水平标尺背景
    this.horizontalRulerBg = new Konva.Rect({
      x: 0,
      y: 0,
      width: this.layer.width(),
      height: this.options.thickness,
      fill: this.options.bgColor,
    });

    // 垂直标尺背景
    this.verticalRulerBg = new Konva.Rect({
      x: 0,
      y: 0,
      width: this.options.thickness,
      height: this.layer.height(),
      fill: this.options.bgColor,
    });

    // 角落方块
    this.cornerRect = new Konva.Rect({
      x: 0,
      y: 0,
      width: this.options.thickness,
      height: this.options.thickness,
      fill: this.options.bgColor,
    });

    // 水平刻度组
    this.horizontalTicks = new Konva.Group();

    // 垂直刻度组
    this.verticalTicks = new Konva.Group();

    // 鼠标位置指示线
    this.mouseIndicatorH = new Konva.Line({
      points: [],
      stroke: '#5e7ce0',
      strokeWidth: 1,
      visible: false,
    });

    this.mouseIndicatorV = new Konva.Line({
      points: [],
      stroke: '#5e7ce0',
      strokeWidth: 1,
      visible: false,
    });

    // 添加到组
    this.group.add(this.horizontalRulerBg);
    this.group.add(this.verticalRulerBg);
    this.group.add(this.cornerRect);
    this.group.add(this.horizontalTicks);
    this.group.add(this.verticalTicks);
    this.group.add(this.mouseIndicatorH);
    this.group.add(this.mouseIndicatorV);

    // 添加到图层
    this.layer.add(this.group);
  }

  /**
   * 更新标尺（缩放、滚动变化时调用）
   * @param {Object} params - 参数
   * @param {number} params.zoom - 缩放比例
   * @param {number} params.scrollX - 水平滚动位置
   * @param {number} params.scrollY - 垂直滚动位置
   */
  update({ zoom, scrollX, scrollY }) {
    this.zoom = zoom || this.zoom;
    this.scrollX = scrollX !== undefined ? scrollX : this.scrollX;
    this.scrollY = scrollY !== undefined ? scrollY : this.scrollY;

    this.drawHorizontalRuler();
    this.drawVerticalRuler();
    this.layer.batchDraw();
  }

  /**
   * 绘制水平标尺
   */
  drawHorizontalRuler() {
    const width = this.layer.width();
    const thickness = this.options.thickness;
    const { smallStep, largeStep } = this.options;

    // 清除旧刻度
    this.horizontalTicks.destroyChildren();

    // 计算起始位置（考虑滚动和缩放）
    const startPx = Math.floor(this.scrollX / this.zoom / smallStep) * smallStep;
    const endPx = Math.ceil((this.scrollX + width) / this.zoom / smallStep) * smallStep;

    // 绘制刻度
    for (let px = startPx; px <= endPx; px += smallStep) {
      const screenX = (px * this.zoom) - this.scrollX;

      // 跳过标尺外的刻度
      if (screenX < this.options.thickness || screenX > width) continue;

      const isLarge = px % largeStep === 0;
      const tickHeight = isLarge ? 12 : 6;

      // 刻度线
      const tick = new Konva.Line({
        points: [screenX, thickness, screenX, thickness - tickHeight],
        stroke: this.options.lineColor,
        strokeWidth: 1,
      });
      this.horizontalTicks.add(tick);

      // 大刻度显示数字
      if (isLarge) {
        const text = new Konva.Text({
          x: screenX - 15,
          y: 2,
          text: String(px),
          fontSize: this.options.fontSize,
          fill: this.options.textColor,
          width: 30,
          align: 'center',
        });
        this.horizontalTicks.add(text);
      }
    }
  }

  /**
   * 绘制垂直标尺
   */
  drawVerticalRuler() {
    const height = this.layer.height();
    const thickness = this.options.thickness;
    const { smallStep, largeStep } = this.options;

    // 清除旧刻度
    this.verticalTicks.destroyChildren();

    // 计算起始位置（考虑滚动和缩放）
    const startPx = Math.floor(this.scrollY / this.zoom / smallStep) * smallStep;
    const endPx = Math.ceil((this.scrollY + height) / this.zoom / smallStep) * smallStep;

    // 绘制刻度
    for (let px = startPx; px <= endPx; px += smallStep) {
      const screenY = (px * this.zoom) - this.scrollY;

      // 跳过标尺外的刻度
      if (screenY < this.options.thickness || screenY > height) continue;

      const isLarge = px % largeStep === 0;
      const tickWidth = isLarge ? 12 : 6;

      // 刻度线
      const tick = new Konva.Line({
        points: [thickness, screenY, thickness - tickWidth, screenY],
        stroke: this.options.lineColor,
        strokeWidth: 1,
      });
      this.verticalTicks.add(tick);

      // 大刻度显示数字
      if (isLarge) {
        const text = new Konva.Text({
          x: 2,
          y: screenY - 6,
          text: String(px),
          fontSize: this.options.fontSize,
          fill: this.options.textColor,
        });
        this.verticalTicks.add(text);
      }
    }
  }

  /**
   * 更新鼠标位置指示
   * @param {number} x - 鼠标X坐标（Canvas坐标）
   * @param {number} y - 鼠标Y坐标（Canvas坐标）
   */
  updateMousePosition(x, y) {
    if (x === null || y === null) {
      this.hideMouseIndicator();
      return;
    }

    this.mouseX = x;
    this.mouseY = y;

    const thickness = this.options.thickness;

    // 更新水平指示线
    this.mouseIndicatorH.points([this.mouseX, 0, this.mouseX, thickness]);
    this.mouseIndicatorH.visible(true);

    // 更新垂直指示线
    this.mouseIndicatorV.points([0, this.mouseY, thickness, this.mouseY]);
    this.mouseIndicatorV.visible(true);

    this.layer.batchDraw();
  }

  /**
   * 隐藏鼠标位置指示
   */
  hideMouseIndicator() {
    this.mouseIndicatorH.visible(false);
    this.mouseIndicatorV.visible(false);
    this.layer.batchDraw();
  }

  /**
   * 调整标尺尺寸
   * @param {number} width - 画布宽度
   * @param {number} height - 画布高度
   */
  resize(width, height) {
    this.horizontalRulerBg.width(width);
    this.verticalRulerBg.height(height);
    this.update({});
  }

  /**
   * 显示标尺
   */
  show() {
    this.group.visible(true);
    this.layer.batchDraw();
  }

  /**
   * 隐藏标尺
   */
  hide() {
    this.group.visible(false);
    this.layer.batchDraw();
  }

  /**
   * 销毁标尺
   */
  destroy() {
    this.group.destroy();
    this.layer.batchDraw();
  }
}

