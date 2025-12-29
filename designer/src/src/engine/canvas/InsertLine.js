/**
 * InsertLine - 插入线绘制器
 * 
 * Task 4.8: 实现插入线（Canvas Layer）
 * 
 * 职责：
 * - 在 Canvas Layer 绘制插入线
 * - 指示拖拽到容器时的插入位置
 * - 支持水平和垂直方向
 */

import Konva from 'konva';

/**
 * InsertLine 类
 * 管理插入线的显示和隐藏
 */
export class InsertLine {
    /**
     * @param {Konva.Layer} layer - Konva 图层
     * @param {Object} options - 配置选项
     */
    constructor(layer, options = {}) {
        this.layer = layer;
        this.options = {
            color: options.color || '#f56c6c', // 红色
            width: options.width || 2,
            dashLength: options.dashLength || 8,
            opacity: options.opacity || 0.8,
            ...options,
        };

        // 创建插入线组
        this.group = new Konva.Group({
            visible: false,
        });

        // 创建主线条
        this.line = new Konva.Line({
            points: [0, 0, 0, 0],
            stroke: this.options.color,
            strokeWidth: this.options.width,
            dash: [this.options.dashLength, this.options.dashLength],
            lineCap: 'round',
            lineJoin: 'round',
            opacity: this.options.opacity,
        });

        // 创建端点圆圈（两个）
        this.startCircle = new Konva.Circle({
            x: 0,
            y: 0,
            radius: 4,
            fill: this.options.color,
            opacity: this.options.opacity,
        });

        this.endCircle = new Konva.Circle({
            x: 0,
            y: 0,
            radius: 4,
            fill: this.options.color,
            opacity: this.options.opacity,
        });

        // 添加到组
        this.group.add(this.line);
        this.group.add(this.startCircle);
        this.group.add(this.endCircle);

        // 添加到图层
        this.layer.add(this.group);
    }

    /**
     * 显示插入线
     * 
     * @param {Object} insertLineInfo - 插入线信息 { x, y, width, height }
     */
    show(insertLineInfo) {
        if (!insertLineInfo) {
            this.hide();
            return;
        }

        const { x, y, width, height } = insertLineInfo;

        // 判断是水平还是垂直线
        const isVertical = height > width;

        if (isVertical) {
            // 垂直线
            this.line.points([x, y, x, y + height]);
            this.startCircle.position({ x, y });
            this.endCircle.position({ x, y: y + height });
        } else {
            // 水平线
            this.line.points([x, y, x + width, y]);
            this.startCircle.position({ x, y });
            this.endCircle.position({ x: x + width, y });
        }

        this.group.visible(true);
        this.layer.batchDraw();
    }

    /**
     * 隐藏插入线
     */
    hide() {
        this.group.visible(false);
        this.layer.batchDraw();
    }

    /**
     * 更新插入线位置
     * 
     * @param {Object} insertLineInfo - 插入线信息
     */
    update(insertLineInfo) {
        this.show(insertLineInfo);
    }

    /**
     * 设置插入线颜色
     * 
     * @param {string} color - 颜色值
     */
    setColor(color) {
        this.options.color = color;
        this.line.stroke(color);
        this.startCircle.fill(color);
        this.endCircle.fill(color);
        this.layer.batchDraw();
    }

    /**
     * 设置插入线透明度
     * 
     * @param {number} opacity - 透明度 (0-1)
     */
    setOpacity(opacity) {
        this.options.opacity = opacity;
        this.line.opacity(opacity);
        this.startCircle.opacity(opacity);
        this.endCircle.opacity(opacity);
        this.layer.batchDraw();
    }

    /**
     * 销毁插入线
     */
    destroy() {
        this.group.destroy();
        this.layer.batchDraw();
    }

    /**
     * 获取当前是否可见
     * 
     * @returns {boolean} 是否可见
     */
    isVisible() {
        return this.group.visible();
    }
}

export default InsertLine;

