/**
 * AlignmentGuides - 对齐辅助线
 * 
 * Task 5.5: 实现对齐辅助线（Canvas Layer）
 * 
 * 职责：
 * - 检测组件边缘对齐（左、右、上、下）
 * - 检测组件中心对齐（水平、垂直）
 * - 绘制红色虚线
 * - 实现对齐阈值（5px）
 * - Task 5.6: 提供自动吸附位置建议
 */

import Konva from 'konva';

/**
 * 对齐类型
 */
export const ALIGNMENT_TYPES = {
    LEFT: 'left',
    RIGHT: 'right',
    TOP: 'top',
    BOTTOM: 'bottom',
    CENTER_HORIZONTAL: 'center-horizontal',
    CENTER_VERTICAL: 'center-vertical',
};

/**
 * AlignmentGuides 类
 * 管理对齐辅助线的显示和计算
 */
export class AlignmentGuides {
    /**
     * @param {Konva.Layer} layer - Konva 图层
     * @param {Object} options - 配置选项
     */
    constructor(layer, options = {}) {
        this.layer = layer;
        this.options = {
            lineColor: options.lineColor || '#f56c6c',
            lineWidth: options.lineWidth || 1,
            dashLength: options.dashLength || 4,
            threshold: options.threshold || 5, // 对齐阈值（像素）
            opacity: options.opacity || 0.8,
            ...options,
        };

        // 辅助线组
        this.group = new Konva.Group({
            visible: false,
            listening: false,
        });

        // 创建辅助线（复用）
        this.verticalLine = new Konva.Line({
            points: [0, 0, 0, 0],
            stroke: this.options.lineColor,
            strokeWidth: this.options.lineWidth,
            dash: [this.options.dashLength, this.options.dashLength],
            opacity: this.options.opacity,
        });

        this.horizontalLine = new Konva.Line({
            points: [0, 0, 0, 0],
            stroke: this.options.lineColor,
            strokeWidth: this.options.lineWidth,
            dash: [this.options.dashLength, this.options.dashLength],
            opacity: this.options.opacity,
        });

        this.group.add(this.verticalLine);
        this.group.add(this.horizontalLine);
        this.layer.add(this.group);

        // 当前对齐信息
        this.currentAlignment = null;
    }

    /**
     * 检测对齐并显示辅助线
     * 
     * @param {Object} movingRect - 正在移动的组件矩形 { x, y, width, height }
     * @param {Array} otherRects - 其他组件的矩形数组
     * @param {Object} canvasBounds - 画布边界 { x, y, width, height }
     * @returns {Object|null} 对齐信息 { type, position, snapPosition }
     */
    checkAlignment(movingRect, otherRects = [], canvasBounds = null) {
        if (!movingRect) {
            this.hide();
            return null;
        }

        const movingEdges = this.getEdges(movingRect);
        const alignments = [];

        // 检测与其他组件的对齐
        otherRects.forEach((rect) => {
            const edges = this.getEdges(rect);

            // 检测垂直对齐
            this.checkVerticalAlignment(movingEdges, edges, alignments, rect);

            // 检测水平对齐
            this.checkHorizontalAlignment(movingEdges, edges, alignments, rect);
        });

        // 检测与画布边缘的对齐
        if (canvasBounds) {
            this.checkCanvasAlignment(movingEdges, canvasBounds, alignments);
        }

        // 找到最近的对齐
        if (alignments.length > 0) {
            alignments.sort((a, b) => a.distance - b.distance);
            const bestAlignment = alignments[0];

            if (bestAlignment.distance <= this.options.threshold) {
                this.showGuide(bestAlignment);
                this.currentAlignment = bestAlignment;
                return bestAlignment;
            }
        }

        this.hide();
        this.currentAlignment = null;
        return null;
    }

    /**
     * 获取矩形的所有边缘和中心点
     */
    getEdges(rect) {
        return {
            left: rect.x,
            right: rect.x + rect.width,
            top: rect.y,
            bottom: rect.y + rect.height,
            centerX: rect.x + rect.width / 2,
            centerY: rect.y + rect.height / 2,
        };
    }

    /**
     * 检测垂直方向对齐
     */
    checkVerticalAlignment(movingEdges, targetEdges, alignments, targetRect) {
        const threshold = this.options.threshold;

        // 左边缘对齐
        const leftDist = Math.abs(movingEdges.left - targetEdges.left);
        if (leftDist <= threshold) {
            alignments.push({
                type: ALIGNMENT_TYPES.LEFT,
                distance: leftDist,
                position: targetEdges.left,
                snapPosition: {
                    x: targetEdges.left,
                },
                lineStart: { x: targetEdges.left, y: Math.min(movingEdges.top, targetEdges.top) },
                lineEnd: { x: targetEdges.left, y: Math.max(movingEdges.bottom, targetEdges.bottom) },
                targetRect,
            });
        }

        // 右边缘对齐
        const rightDist = Math.abs(movingEdges.right - targetEdges.right);
        if (rightDist <= threshold) {
            alignments.push({
                type: ALIGNMENT_TYPES.RIGHT,
                distance: rightDist,
                position: targetEdges.right,
                snapPosition: {
                    x: targetEdges.right - (movingEdges.right - movingEdges.left),
                },
                lineStart: { x: targetEdges.right, y: Math.min(movingEdges.top, targetEdges.top) },
                lineEnd: { x: targetEdges.right, y: Math.max(movingEdges.bottom, targetEdges.bottom) },
                targetRect,
            });
        }

        // 中心垂直对齐
        const centerXDist = Math.abs(movingEdges.centerX - targetEdges.centerX);
        if (centerXDist <= threshold) {
            alignments.push({
                type: ALIGNMENT_TYPES.CENTER_VERTICAL,
                distance: centerXDist,
                position: targetEdges.centerX,
                snapPosition: {
                    x: targetEdges.centerX - (movingEdges.right - movingEdges.left) / 2,
                },
                lineStart: { x: targetEdges.centerX, y: Math.min(movingEdges.top, targetEdges.top) },
                lineEnd: { x: targetEdges.centerX, y: Math.max(movingEdges.bottom, targetEdges.bottom) },
                targetRect,
            });
        }
    }

    /**
     * 检测水平方向对齐
     */
    checkHorizontalAlignment(movingEdges, targetEdges, alignments, targetRect) {
        const threshold = this.options.threshold;

        // 上边缘对齐
        const topDist = Math.abs(movingEdges.top - targetEdges.top);
        if (topDist <= threshold) {
            alignments.push({
                type: ALIGNMENT_TYPES.TOP,
                distance: topDist,
                position: targetEdges.top,
                snapPosition: {
                    y: targetEdges.top,
                },
                lineStart: { x: Math.min(movingEdges.left, targetEdges.left), y: targetEdges.top },
                lineEnd: { x: Math.max(movingEdges.right, targetEdges.right), y: targetEdges.top },
                targetRect,
            });
        }

        // 下边缘对齐
        const bottomDist = Math.abs(movingEdges.bottom - targetEdges.bottom);
        if (bottomDist <= threshold) {
            alignments.push({
                type: ALIGNMENT_TYPES.BOTTOM,
                distance: bottomDist,
                position: targetEdges.bottom,
                snapPosition: {
                    y: targetEdges.bottom - (movingEdges.bottom - movingEdges.top),
                },
                lineStart: { x: Math.min(movingEdges.left, targetEdges.left), y: targetEdges.bottom },
                lineEnd: { x: Math.max(movingEdges.right, targetEdges.right), y: targetEdges.bottom },
                targetRect,
            });
        }

        // 中心水平对齐
        const centerYDist = Math.abs(movingEdges.centerY - targetEdges.centerY);
        if (centerYDist <= threshold) {
            alignments.push({
                type: ALIGNMENT_TYPES.CENTER_HORIZONTAL,
                distance: centerYDist,
                position: targetEdges.centerY,
                snapPosition: {
                    y: targetEdges.centerY - (movingEdges.bottom - movingEdges.top) / 2,
                },
                lineStart: { x: Math.min(movingEdges.left, targetEdges.left), y: targetEdges.centerY },
                lineEnd: { x: Math.max(movingEdges.right, targetEdges.right), y: targetEdges.centerY },
                targetRect,
            });
        }
    }

    /**
     * 检测与画布边缘的对齐
     */
    checkCanvasAlignment(movingEdges, canvasBounds, alignments) {
        const threshold = this.options.threshold;
        const canvasEdges = this.getEdges(canvasBounds);

        // 左边缘
        const leftDist = Math.abs(movingEdges.left - canvasEdges.left);
        if (leftDist <= threshold) {
            alignments.push({
                type: ALIGNMENT_TYPES.LEFT,
                distance: leftDist,
                position: canvasEdges.left,
                snapPosition: { x: canvasEdges.left },
                lineStart: { x: canvasEdges.left, y: canvasEdges.top },
                lineEnd: { x: canvasEdges.left, y: canvasEdges.bottom },
                isCanvasEdge: true,
            });
        }

        // 右边缘
        const rightDist = Math.abs(movingEdges.right - canvasEdges.right);
        if (rightDist <= threshold) {
            alignments.push({
                type: ALIGNMENT_TYPES.RIGHT,
                distance: rightDist,
                position: canvasEdges.right,
                snapPosition: { x: canvasEdges.right - (movingEdges.right - movingEdges.left) },
                lineStart: { x: canvasEdges.right, y: canvasEdges.top },
                lineEnd: { x: canvasEdges.right, y: canvasEdges.bottom },
                isCanvasEdge: true,
            });
        }

        // 上边缘
        const topDist = Math.abs(movingEdges.top - canvasEdges.top);
        if (topDist <= threshold) {
            alignments.push({
                type: ALIGNMENT_TYPES.TOP,
                distance: topDist,
                position: canvasEdges.top,
                snapPosition: { y: canvasEdges.top },
                lineStart: { x: canvasEdges.left, y: canvasEdges.top },
                lineEnd: { x: canvasEdges.right, y: canvasEdges.top },
                isCanvasEdge: true,
            });
        }

        // 下边缘
        const bottomDist = Math.abs(movingEdges.bottom - canvasEdges.bottom);
        if (bottomDist <= threshold) {
            alignments.push({
                type: ALIGNMENT_TYPES.BOTTOM,
                distance: bottomDist,
                position: canvasEdges.bottom,
                snapPosition: { y: canvasEdges.bottom - (movingEdges.bottom - movingEdges.top) },
                lineStart: { x: canvasEdges.left, y: canvasEdges.bottom },
                lineEnd: { x: canvasEdges.right, y: canvasEdges.bottom },
                isCanvasEdge: true,
            });
        }
    }

    /**
     * 显示辅助线
     */
    showGuide(alignment) {
        const { lineStart, lineEnd, type } = alignment;

        // 判断是垂直线还是水平线
        const isVertical = type === ALIGNMENT_TYPES.LEFT || 
                          type === ALIGNMENT_TYPES.RIGHT || 
                          type === ALIGNMENT_TYPES.CENTER_VERTICAL;

        if (isVertical) {
            this.verticalLine.points([lineStart.x, lineStart.y, lineEnd.x, lineEnd.y]);
            this.verticalLine.visible(true);
            this.horizontalLine.visible(false);
        } else {
            this.horizontalLine.points([lineStart.x, lineStart.y, lineEnd.x, lineEnd.y]);
            this.horizontalLine.visible(true);
            this.verticalLine.visible(false);
        }

        this.group.visible(true);
        this.layer.batchDraw();
    }

    /**
     * 隐藏辅助线
     */
    hide() {
        this.group.visible(false);
        this.verticalLine.visible(false);
        this.horizontalLine.visible(false);
        this.layer.batchDraw();
    }

    /**
     * 获取吸附位置建议
     * Task 5.6: 实现自动吸附
     * 
     * @param {Object} currentPosition - 当前位置 { x, y }
     * @returns {Object|null} 吸附后的位置 { x, y }
     */
    getSnapPosition(currentPosition) {
        if (!this.currentAlignment) return null;

        const snapPosition = { ...currentPosition };

        if (this.currentAlignment.snapPosition.x !== undefined) {
            snapPosition.x = this.currentAlignment.snapPosition.x;
        }
        if (this.currentAlignment.snapPosition.y !== undefined) {
            snapPosition.y = this.currentAlignment.snapPosition.y;
        }

        return snapPosition;
    }

    /**
     * 设置辅助线颜色
     */
    setColor(color) {
        this.options.lineColor = color;
        this.verticalLine.stroke(color);
        this.horizontalLine.stroke(color);
        this.layer.batchDraw();
    }

    /**
     * 设置对齐阈值
     */
    setThreshold(threshold) {
        this.options.threshold = threshold;
    }

    /**
     * 销毁辅助线
     */
    destroy() {
        this.group.destroy();
        this.layer.batchDraw();
    }

    /**
     * 获取当前是否可见
     */
    isVisible() {
        return this.group.visible();
    }
}

export default AlignmentGuides;
