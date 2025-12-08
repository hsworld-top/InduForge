/**
 * DragPreview - 拖拽预览类
 *
 * 职责：
 * - 在 Canvas Layer 上绘制半透明的组件轮廓
 * - 实现预览跟随鼠标移动
 * - 实现预览显示/隐藏
 *
 * Requirements:
 * - Requirement 6: 拖拽功能
 * - Acceptance Criteria 6.5: 显示拖拽预览（半透明轮廓）
 *
 * Task 4.7: 实现拖拽预览（Canvas Layer）
 */
import Konva from 'konva';

export class DragPreview {
    /**
     * @param {Konva.Layer} layer - Konva 图层（通常是 selectionLayer）
     */
    constructor(layer) {
        this.layer = layer;
        this.previewGroup = null;
        this.isVisible = false;
    }

    /**
     * 显示拖拽预览
     * @param {Object} component - 组件实例
     * @param {number} x - 鼠标 X 坐标（画布坐标系）
     * @param {number} y - 鼠标 Y 坐标（画布坐标系）
     */
    show(component, x, y) {
        // 如果已有预览，先隐藏
        if (this.previewGroup) {
            this.hide();
        }

        // 创建预览组
        this.previewGroup = new Konva.Group({
            x,
            y,
            opacity: 0.5,
        });

        // 获取组件尺寸
        const width = component.style?.width || 100;
        const height = component.style?.height || 50;

        // 绘制半透明矩形轮廓
        const rect = new Konva.Rect({
            x: 0,
            y: 0,
            width,
            height,
            fill: '#5e7ce0',
            stroke: '#5e7ce0',
            strokeWidth: 2,
            dash: [5, 5],
            opacity: 0.3,
        });

        // 添加组件类型文本
        const text = new Konva.Text({
            x: 5,
            y: 5,
            text: component.name || component.type,
            fontSize: 12,
            fill: '#ffffff',
            fontStyle: 'bold',
        });

        this.previewGroup.add(rect);
        this.previewGroup.add(text);
        this.layer.add(this.previewGroup);
        this.layer.batchDraw();

        this.isVisible = true;
    }

    /**
     * 更新拖拽预览位置
     * @param {number} x - 鼠标 X 坐标（画布坐标系）
     * @param {number} y - 鼠标 Y 坐标（画布坐标系）
     */
    updatePosition(x, y) {
        if (!this.previewGroup || !this.isVisible) {
            return;
        }

        this.previewGroup.position({ x, y });
        this.layer.batchDraw();
    }

    /**
     * 隐藏拖拽预览
     */
    hide() {
        if (this.previewGroup) {
            this.previewGroup.destroy();
            this.previewGroup = null;
            this.layer.batchDraw();
        }

        this.isVisible = false;
    }

    /**
     * 销毁拖拽预览
     */
    destroy() {
        this.hide();
        this.layer = null;
    }
}
