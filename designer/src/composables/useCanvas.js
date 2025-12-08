/**
 * useCanvas Composable - 画布逻辑
 * 处理画布缩放、网格吸附和坐标转换
 * Requirements: 2.3, 2.4, 3.6
 */
import { ref, computed, reactive } from 'vue';

/**
 * 计算缩放比例
 * Property 2: Canvas Scale Calculation
 * For scaleMode "fit", scale = min(viewportWidth/canvasWidth, viewportHeight/canvasHeight)
 *
 * @param {Object} viewportSize - 视口尺寸 { width, height }
 * @param {Object} canvasSize - 画布尺寸 { width, height }
 * @param {string} scaleMode - 缩放模式 ('fit' | 'fill' | 'fixed')
 * @returns {number} 缩放比例
 */
export function calculateScale(viewportSize, canvasSize, scaleMode) {
    // 验证输入
    if (!viewportSize || !canvasSize) {
        return 1;
    }

    const { width: vw, height: vh } = viewportSize;
    const { width: cw, height: ch } = canvasSize;

    // 防止除以零
    if (cw <= 0 || ch <= 0 || vw <= 0 || vh <= 0) {
        return 1;
    }

    switch (scaleMode) {
        case 'fit':
            // Requirements: 2.4 - 保持宽高比，适应视口
            return Math.min(vw / cw, vh / ch);
        case 'fill':
            // 填充模式：取较大的缩放比例
            return Math.max(vw / cw, vh / ch);
        case 'fixed':
        default:
            // 固定模式：不缩放
            return 1;
    }
}

/**
 * 网格吸附计算
 * Property 3: Grid Snapping Calculation
 * For any position (x, y) and grid size g > 0, snapped = (round(x/g)*g, round(y/g)*g)
 *
 * @param {Object} position - 位置 { x, y }
 * @param {number} gridSize - 网格大小
 * @returns {Object} 吸附后的位置 { x, y }
 */
export function snapToGrid(position, gridSize) {
    // 验证输入
    if (!position || typeof position.x !== 'number' || typeof position.y !== 'number') {
        return { x: 0, y: 0 };
    }

    // 如果网格大小无效，返回原位置
    if (!gridSize || gridSize <= 0) {
        return { x: position.x, y: position.y };
    }

    return {
        x: Math.round(position.x / gridSize) * gridSize,
        y: Math.round(position.y / gridSize) * gridSize,
    };
}

/**
 * useCanvas Composable
 * 提供画布状态和操作方法
 */
export function useCanvas() {
    // 画布状态
    const canvasState = reactive({
        scale: 1,
        offset: { x: 0, y: 0 },
        gridSize: 10,
        snapToGrid: true,
    });

    // 视口尺寸
    const viewportSize = ref({ width: 0, height: 0 });

    // 画布尺寸（来自页面配置）
    const canvasSize = ref({ width: 1920, height: 1080 });

    // 缩放模式
    const scaleMode = ref('fit');

    /**
     * 更新视口尺寸并重新计算缩放
     * @param {Object} size - 视口尺寸 { width, height }
     */
    function updateViewportSize(size) {
        viewportSize.value = size;
        canvasState.scale = calculateScale(viewportSize.value, canvasSize.value, scaleMode.value);
    }

    /**
     * 更新画布尺寸并重新计算缩放
     * @param {Object} size - 画布尺寸 { width, height }
     */
    function updateCanvasSize(size) {
        canvasSize.value = size;
        canvasState.scale = calculateScale(viewportSize.value, canvasSize.value, scaleMode.value);
    }

    /**
     * 更新缩放模式
     * @param {string} mode - 缩放模式
     */
    function updateScaleMode(mode) {
        scaleMode.value = mode;
        canvasState.scale = calculateScale(viewportSize.value, canvasSize.value, scaleMode.value);
    }

    /**
     * 设置网格大小
     * @param {number} size - 网格大小
     */
    function setGridSize(size) {
        if (size > 0) {
            canvasState.gridSize = size;
        }
    }

    /**
     * 切换网格吸附
     * @param {boolean} enabled - 是否启用
     */
    function setSnapToGrid(enabled) {
        canvasState.snapToGrid = enabled;
    }

    /**
     * 屏幕坐标转画布坐标
     * @param {Object} screenPos - 屏幕坐标 { x, y }
     * @returns {Object} 画布坐标 { x, y }
     */
    function screenToCanvas(screenPos) {
        if (!screenPos) {
            return { x: 0, y: 0 };
        }

        const scale = canvasState.scale || 1;
        const offset = canvasState.offset || { x: 0, y: 0 };

        return {
            x: (screenPos.x - offset.x) / scale,
            y: (screenPos.y - offset.y) / scale,
        };
    }

    /**
     * 画布坐标转屏幕坐标
     * @param {Object} canvasPos - 画布坐标 { x, y }
     * @returns {Object} 屏幕坐标 { x, y }
     */
    function canvasToScreen(canvasPos) {
        if (!canvasPos) {
            return { x: 0, y: 0 };
        }

        const scale = canvasState.scale || 1;
        const offset = canvasState.offset || { x: 0, y: 0 };

        return {
            x: canvasPos.x * scale + offset.x,
            y: canvasPos.y * scale + offset.y,
        };
    }

    /**
     * 设置画布偏移
     * @param {Object} offset - 偏移量 { x, y }
     */
    function setOffset(offset) {
        canvasState.offset = offset;
    }

    /**
     * 手动设置缩放比例
     * @param {number} scale - 缩放比例
     */
    function setScale(scale) {
        if (scale > 0) {
            canvasState.scale = scale;
        }
    }

    /**
     * 对位置应用网格吸附（如果启用）
     * @param {Object} position - 位置 { x, y }
     * @returns {Object} 处理后的位置
     */
    function applySnapToGrid(position) {
        if (canvasState.snapToGrid) {
            return snapToGrid(position, canvasState.gridSize);
        }
        return position;
    }

    return {
        // 状态
        canvasState,
        viewportSize,
        canvasSize,
        scaleMode,

        // 方法
        calculateScale,
        snapToGrid,
        screenToCanvas,
        canvasToScreen,
        updateViewportSize,
        updateCanvasSize,
        updateScaleMode,
        setGridSize,
        setSnapToGrid,
        setOffset,
        setScale,
        applySnapToGrid,
    };
}

export default useCanvas;
