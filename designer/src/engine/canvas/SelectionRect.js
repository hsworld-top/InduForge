/**
 * SelectionRect - 框选矩形
 * 
 * Task 5.7: 实现框选多选
 * 
 * 职责：
 * - 在 Canvas Layer 绘制框选矩形
 * - 显示蓝色半透明矩形
 * - 实时更新矩形大小
 * - 计算框选范围内的组件
 */

import Konva from 'konva';

/**
 * SelectionRect 类
 * 管理框选矩形的绘制和交互
 */
export class SelectionRect {
    /**
     * @param {Konva.Layer} layer - Konva 图层
     * @param {Object} options - 配置选项
     */
    constructor(layer, options = {}) {
        this.layer = layer;
        this.options = {
            fillColor: options.fillColor || 'rgba(64, 158, 255, 0.1)',
            strokeColor: options.strokeColor || '#409eff',
            strokeWidth: options.strokeWidth || 1,
            ...options,
        };

        // 框选起始点
        this.startPoint = null;

        // 创建框选矩形
        this.rect = new Konva.Rect({
            fill: this.options.fillColor,
            stroke: this.options.strokeColor,
            strokeWidth: this.options.strokeWidth,
            visible: false,
            listening: false,
        });

        this.layer.add(this.rect);
    }

    /**
     * 开始框选
     * 
     * @param {Object} point - 起始点 { x, y }
     */
    start(point) {
        this.startPoint = { ...point };
        this.rect.position(point);
        this.rect.size({ width: 0, height: 0 });
        this.rect.visible(true);
        this.layer.batchDraw();
    }

    /**
     * 更新框选矩形
     * 
     * @param {Object} point - 当前鼠标位置 { x, y }
     */
    update(point) {
        if (!this.startPoint) return;

        const x = Math.min(this.startPoint.x, point.x);
        const y = Math.min(this.startPoint.y, point.y);
        const width = Math.abs(point.x - this.startPoint.x);
        const height = Math.abs(point.y - this.startPoint.y);

        this.rect.position({ x, y });
        this.rect.size({ width, height });
        this.layer.batchDraw();
    }

    /**
     * 结束框选
     * 
     * @returns {Object} 框选矩形 { x, y, width, height }
     */
    end() {
        const result = {
            x: this.rect.x(),
            y: this.rect.y(),
            width: this.rect.width(),
            height: this.rect.height(),
        };

        this.hide();
        return result;
    }

    /**
     * 隐藏框选矩形
     */
    hide() {
        this.startPoint = null;
        this.rect.visible(false);
        this.layer.batchDraw();
    }

    /**
     * 获取当前框选矩形
     * 
     * @returns {Object|null} 框选矩形或 null
     */
    getRect() {
        if (!this.rect.visible()) return null;

        return {
            x: this.rect.x(),
            y: this.rect.y(),
            width: this.rect.width(),
            height: this.rect.height(),
        };
    }

    /**
     * 检查组件是否在框选范围内
     * 
     * @param {Object} componentRect - 组件矩形 { x, y, width, height }
     * @returns {boolean} 是否在框选范围内
     */
    isComponentInSelection(componentRect) {
        const selectionRect = this.getRect();
        if (!selectionRect) return false;

        const selLeft = selectionRect.x;
        const selRight = selectionRect.x + selectionRect.width;
        const selTop = selectionRect.y;
        const selBottom = selectionRect.y + selectionRect.height;

        const compLeft = componentRect.x;
        const compRight = componentRect.x + componentRect.width;
        const compTop = componentRect.y;
        const compBottom = componentRect.y + componentRect.height;

        // 检查是否有重叠（任何部分在框选范围内）
        return !(
            compRight < selLeft ||
            compLeft > selRight ||
            compBottom < selTop ||
            compTop > selBottom
        );
    }

    /**
     * 从组件列表中筛选出框选范围内的组件
     * 
     * @param {Array} components - 组件列表
     * @returns {Array} 框选范围内的组件ID列表
     */
    getSelectedComponents(components) {
        const selectionRect = this.getRect();
        if (!selectionRect) return [];

        const selectedIds = [];

        components.forEach((component) => {
            const componentRect = {
                x: component.style.left || 0,
                y: component.style.top || 0,
                width: component.style.width || 0,
                height: component.style.height || 0,
            };

            if (this.isComponentInSelection(componentRect)) {
                selectedIds.push(component.id);
            }
        });

        return selectedIds;
    }

    /**
     * 销毁框选矩形
     */
    destroy() {
        this.rect.destroy();
        this.layer.batchDraw();
    }

    /**
     * 获取当前是否可见
     */
    isVisible() {
        return this.rect.visible();
    }
}

export default SelectionRect;

