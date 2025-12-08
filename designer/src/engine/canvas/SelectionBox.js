/**
 * SelectionBox - 选择框绘制器
 * 
 * Task 5.2: 实现选择框（Canvas Layer）
 * 
 * 职责：
 * - 在 Canvas Layer 绘制选择框
 * - 绘制 8 个控制点（四角 + 四边中点）
 * - 根据 DOM 组件位置更新选择框
 * - 支持拖拽缩放和旋转
 */

import Konva from 'konva';

/**
 * 控制点类型
 */
export const HANDLE_TYPES = {
    TOP_LEFT: 'nw',
    TOP_CENTER: 'n',
    TOP_RIGHT: 'ne',
    MIDDLE_RIGHT: 'e',
    BOTTOM_RIGHT: 'se',
    BOTTOM_CENTER: 's',
    BOTTOM_LEFT: 'sw',
    MIDDLE_LEFT: 'w',
    ROTATE: 'rotate',
};

/**
 * SelectionBox 类
 * 管理选择框的显示、隐藏和交互
 */
export class SelectionBox {
    /**
     * @param {Konva.Layer} layer - Konva 图层
     * @param {Object} options - 配置选项
     */
    constructor(layer, options = {}) {
        this.layer = layer;
        this.options = {
            borderColor: options.borderColor || '#409eff',
            borderWidth: options.borderWidth || 2,
            handleSize: options.handleSize || 8,
            handleFill: options.handleFill || '#ffffff',
            handleStroke: options.handleStroke || '#409eff',
            rotateHandleOffset: options.rotateHandleOffset || 30,
            ...options,
        };

        // 当前选中的组件信息
        this.selectedRect = null;

        // 创建选择框组
        this.group = new Konva.Group({
            visible: false,
            name: 'selection-box',
        });

        // 创建边框矩形
        this.border = new Konva.Rect({
            stroke: this.options.borderColor,
            strokeWidth: this.options.borderWidth,
            fill: 'transparent',
            dash: [4, 4],
            listening: false,
        });

        // 创建控制点
        this.handles = {};
        this.createHandles();

        // 创建旋转控制点
        this.createRotateHandle();

        // 添加到组
        this.group.add(this.border);
        Object.values(this.handles).forEach((handle) => this.group.add(handle));

        // 添加到图层
        this.layer.add(this.group);

        // 事件回调
        this.onResizeStart = null;
        this.onResize = null;
        this.onResizeEnd = null;
        this.onRotateStart = null;
        this.onRotate = null;
        this.onRotateEnd = null;
    }

    /**
     * 创建 8 个控制点
     */
    createHandles() {
        const handleTypes = [
            HANDLE_TYPES.TOP_LEFT,
            HANDLE_TYPES.TOP_CENTER,
            HANDLE_TYPES.TOP_RIGHT,
            HANDLE_TYPES.MIDDLE_RIGHT,
            HANDLE_TYPES.BOTTOM_RIGHT,
            HANDLE_TYPES.BOTTOM_CENTER,
            HANDLE_TYPES.BOTTOM_LEFT,
            HANDLE_TYPES.MIDDLE_LEFT,
        ];

        handleTypes.forEach((type) => {
            const handle = new Konva.Rect({
                width: this.options.handleSize,
                height: this.options.handleSize,
                fill: this.options.handleFill,
                stroke: this.options.handleStroke,
                strokeWidth: 1,
                draggable: true,
                name: `handle-${type}`,
                cursor: this.getCursor(type),
            });

            // 绑定拖拽事件
            handle.on('dragstart', (e) => this.handleResizeStart(e, type));
            handle.on('dragmove', (e) => this.handleResize(e, type));
            handle.on('dragend', (e) => this.handleResizeEnd(e, type));

            this.handles[type] = handle;
        });
    }

    /**
     * 创建旋转控制点
     */
    createRotateHandle() {
        // 旋转控制点（圆形）
        this.rotateHandle = new Konva.Circle({
            radius: this.options.handleSize / 2,
            fill: this.options.handleFill,
            stroke: this.options.handleStroke,
            strokeWidth: 1,
            draggable: true,
            name: 'handle-rotate',
            cursor: 'crosshair',
        });

        // 连接线
        this.rotateLine = new Konva.Line({
            points: [0, 0, 0, 0],
            stroke: this.options.handleStroke,
            strokeWidth: 1,
            dash: [2, 2],
            listening: false,
        });

        // 绑定旋转事件
        this.rotateHandle.on('dragstart', (e) => this.handleRotateStart(e));
        this.rotateHandle.on('dragmove', (e) => this.handleRotateMove(e));
        this.rotateHandle.on('dragend', (e) => this.handleRotateEnd(e));

        this.handles[HANDLE_TYPES.ROTATE] = this.rotateHandle;
        this.group.add(this.rotateLine);
    }

    /**
     * 获取控制点的光标样式
     */
    getCursor(handleType) {
        const cursorMap = {
            [HANDLE_TYPES.TOP_LEFT]: 'nw-resize',
            [HANDLE_TYPES.TOP_CENTER]: 'n-resize',
            [HANDLE_TYPES.TOP_RIGHT]: 'ne-resize',
            [HANDLE_TYPES.MIDDLE_RIGHT]: 'e-resize',
            [HANDLE_TYPES.BOTTOM_RIGHT]: 'se-resize',
            [HANDLE_TYPES.BOTTOM_CENTER]: 's-resize',
            [HANDLE_TYPES.BOTTOM_LEFT]: 'sw-resize',
            [HANDLE_TYPES.MIDDLE_LEFT]: 'w-resize',
        };
        return cursorMap[handleType] || 'default';
    }

    /**
     * 显示选择框
     * 
     * @param {Object} rect - 组件的边界矩形 { x, y, width, height }
     */
    show(rect) {
        if (!rect) {
            this.hide();
            return;
        }

        this.selectedRect = rect;
        this.updatePosition(rect);
        this.group.visible(true);
        this.layer.batchDraw();
    }

    /**
     * 隐藏选择框
     */
    hide() {
        this.selectedRect = null;
        this.group.visible(false);
        this.layer.batchDraw();
    }

    /**
     * 更新选择框（根据 DOM 元素）
     * Task 5.2: 同步 DOM 坐标到 Canvas
     * 
     * 通过获取 DOM 元素的视口坐标，自动计算相对于 Canvas Layer 的位置，
     * 无需手动传入滚动偏移。
     * 
     * @param {HTMLElement[]} domElements - DOM 元素数组
     * @param {number} zoom - 缩放比例
     * @param {Object} scroll - (已弃用) 滚动偏移，不再使用
     */
    update(domElements, zoom = 1, scroll = { x: 0, y: 0 }) {
        if (!domElements || domElements.length === 0) {
            this.hide();
            return;
        }

        // 获取 Canvas Layer 容器的位置
        const stageContainer = this.layer.getStage().container();
        const stageRect = stageContainer.getBoundingClientRect();

        // 计算所有选中元素的包围盒（相对于 Canvas Layer）
        let minX = Infinity;
        let minY = Infinity;
        let maxX = -Infinity;
        let maxY = -Infinity;

        domElements.forEach(el => {
            const rect = el.getBoundingClientRect();
            
            // 转换为相对于 Canvas Layer 的坐标
            // getBoundingClientRect() 返回相对于视口的坐标，需要减去 stage 的位置
            const left = (rect.left - stageRect.left) / zoom;
            const top = (rect.top - stageRect.top) / zoom;
            const width = rect.width / zoom;
            const height = rect.height / zoom;
            
            const right = left + width;
            const bottom = top + height;

            minX = Math.min(minX, left);
            minY = Math.min(minY, top);
            maxX = Math.max(maxX, right);
            maxY = Math.max(maxY, bottom);
        });

        // 构造包围盒矩形（已经是 Canvas 坐标系）
        const boundingRect = {
            x: minX,
            y: minY,
            width: maxX - minX,
            height: maxY - minY,
        };

        // 显示并更新位置
        this.show(boundingRect);
    }

    /**
     * 更新选择框位置
     * 
     * @param {Object} rect - 组件的边界矩形
     */
    updatePosition(rect) {
        const { x, y, width, height } = rect;
        const halfHandle = this.options.handleSize / 2;

        // 更新边框
        this.border.position({ x, y });
        this.border.size({ width, height });

        // 更新控制点位置
        this.handles[HANDLE_TYPES.TOP_LEFT].position({
            x: x - halfHandle,
            y: y - halfHandle,
        });
        this.handles[HANDLE_TYPES.TOP_CENTER].position({
            x: x + width / 2 - halfHandle,
            y: y - halfHandle,
        });
        this.handles[HANDLE_TYPES.TOP_RIGHT].position({
            x: x + width - halfHandle,
            y: y - halfHandle,
        });
        this.handles[HANDLE_TYPES.MIDDLE_RIGHT].position({
            x: x + width - halfHandle,
            y: y + height / 2 - halfHandle,
        });
        this.handles[HANDLE_TYPES.BOTTOM_RIGHT].position({
            x: x + width - halfHandle,
            y: y + height - halfHandle,
        });
        this.handles[HANDLE_TYPES.BOTTOM_CENTER].position({
            x: x + width / 2 - halfHandle,
            y: y + height - halfHandle,
        });
        this.handles[HANDLE_TYPES.BOTTOM_LEFT].position({
            x: x - halfHandle,
            y: y + height - halfHandle,
        });
        this.handles[HANDLE_TYPES.MIDDLE_LEFT].position({
            x: x - halfHandle,
            y: y + height / 2 - halfHandle,
        });

        // 更新旋转控制点位置
        const rotateY = y - this.options.rotateHandleOffset;
        this.rotateHandle.position({
            x: x + width / 2,
            y: rotateY,
        });

        // 更新旋转连接线
        this.rotateLine.points([x + width / 2, y, x + width / 2, rotateY]);
    }

    /**
     * 处理缩放开始
     */
    handleResizeStart(event, handleType) {
        if (this.onResizeStart) {
            this.onResizeStart({
                handleType,
                rect: this.selectedRect,
                event,
            });
        }
    }

    /**
     * 处理缩放中
     */
    handleResize(event, handleType) {
        if (!this.selectedRect || !this.onResize) return;

        const pos = event.target.position();
        this.onResize({
            handleType,
            position: pos,
            rect: this.selectedRect,
            event,
        });
    }

    /**
     * 处理缩放结束
     */
    handleResizeEnd(event, handleType) {
        if (this.onResizeEnd) {
            this.onResizeEnd({
                handleType,
                rect: this.selectedRect,
                event,
            });
        }
    }

    /**
     * 处理旋转开始
     */
    handleRotateStart(event) {
        if (this.onRotateStart) {
            this.onRotateStart({
                rect: this.selectedRect,
                event,
            });
        }
    }

    /**
     * 处理旋转中
     */
    handleRotateMove(event) {
        if (!this.selectedRect || !this.onRotate) return;

        const pos = event.target.position();
        const centerX = this.selectedRect.x + this.selectedRect.width / 2;
        const centerY = this.selectedRect.y + this.selectedRect.height / 2;

        // 计算旋转角度
        const angle = Math.atan2(pos.y - centerY, pos.x - centerX);
        const rotation = (angle * 180) / Math.PI + 90; // 转换为度数，调整基准

        this.onRotate({
            rotation,
            center: { x: centerX, y: centerY },
            rect: this.selectedRect,
            event,
        });
    }

    /**
     * 处理旋转结束
     */
    handleRotateEnd(event) {
        if (this.onRotateEnd) {
            this.onRotateEnd({
                rect: this.selectedRect,
                event,
            });
        }
    }

    /**
     * 设置边框颜色
     */
    setBorderColor(color) {
        this.options.borderColor = color;
        this.border.stroke(color);
        Object.values(this.handles).forEach((handle) => {
            handle.stroke(color);
        });
        this.rotateLine.stroke(color);
        this.layer.batchDraw();
    }

    /**
     * 销毁选择框
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

export default SelectionBox;

