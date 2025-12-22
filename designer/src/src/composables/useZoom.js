/**
 * useZoom - 画布缩放管理 Composable
 * Task 7.1: 实现画布缩放
 *
 * 功能：
 * - 缩放范围：10% ~ 500%
 * - 鼠标滚轮缩放（Ctrl + 滚轮）
 * - 快捷键缩放（Ctrl + Plus/Minus）
 * - 缩放工具栏（+、-、100%、适应画布）
 * - 缩放中心点保持
 *
 * 使用方式：
 * ```js
 * const { zoom, zoomIn, zoomOut, resetZoom, fitToScreen, setZoom } = useZoom();
 * ```
 */

import { ref, readonly, computed } from 'vue';

/**
 * 创建缩放管理器
 * @param {Object} options - 配置选项
 * @returns {Object} 缩放管理器实例
 */
export function useZoom(options = {}) {
    const {
        initialZoom = 1,
        minZoom = 0.1,
        maxZoom = 5,
        zoomStep = 0.1,
        wheelZoomStep = 0.05,
    } = options;

    // 当前缩放比例
    const zoom = ref(initialZoom);

    // 缩放百分比（用于显示）
    const zoomPercentage = computed(() => Math.round(zoom.value * 100));

    /**
     * 设置缩放比例
     * @param {number} newZoom - 新的缩放比例
     * @param {boolean} clamp - 是否限制在范围内
     */
    function setZoom(newZoom, clamp = true) {
        if (clamp) {
            zoom.value = Math.max(minZoom, Math.min(maxZoom, newZoom));
        } else {
            zoom.value = newZoom;
        }

        console.log(`🔍 Zoom set to ${zoomPercentage.value}%`);
    }

    /**
     * 放大
     */
    function zoomIn() {
        const newZoom = zoom.value + zoomStep;
        setZoom(newZoom);
    }

    /**
     * 缩小
     */
    function zoomOut() {
        const newZoom = zoom.value - zoomStep;
        setZoom(newZoom);
    }

    /**
     * 重置缩放（100%）
     */
    function resetZoom() {
        setZoom(1);
    }

    /**
     * 适应画布
     * @param {Object} canvasSize - 画布尺寸 { width, height }
     * @param {Object} viewportSize - 视口尺寸 { width, height }
     * @param {number} padding - 内边距（px）
     */
    function fitToScreen(canvasSize, viewportSize, padding = 40) {
        if (!canvasSize || !viewportSize) {
            console.warn('⚠️ Cannot fit to screen: missing size information');
            return;
        }

        const scaleX = (viewportSize.width - padding * 2) / canvasSize.width;
        const scaleY = (viewportSize.height - padding * 2) / canvasSize.height;

        // 使用较小的缩放比例以确保完整显示
        const newZoom = Math.min(scaleX, scaleY);
        setZoom(newZoom);

        console.log(`📐 Fit to screen: ${zoomPercentage.value}%`);
    }

    /**
     * 处理鼠标滚轮缩放
     * @param {WheelEvent} event - 滚轮事件
     * @param {Object} zoomCenter - 缩放中心点 { x, y }
     */
    function handleWheel(event, zoomCenter = null) {
        // 只在按下 Ctrl 或 Cmd 时缩放
        if (!event.ctrlKey && !event.metaKey) {
            return false; // 返回 false 表示不处理
        }

        event.preventDefault();

        // 计算缩放增量
        const delta = -event.deltaY * wheelZoomStep * 0.01;
        const newZoom = zoom.value * (1 + delta);

        // 如果提供了缩放中心点，可以在这里处理中心点保持逻辑
        // （需要结合滚动位置调整）
        if (zoomCenter) {
            // TODO: 实现缩放中心点保持
            console.log('Zoom center:', zoomCenter);
        }

        setZoom(newZoom);
        return true; // 返回 true 表示已处理
    }

    /**
     * 处理键盘缩放
     * @param {KeyboardEvent} event - 键盘事件
     */
    function handleKeyboard(event) {
        // Ctrl + Plus：放大
        if ((event.ctrlKey || event.metaKey) && (event.key === '+' || event.key === '=')) {
            event.preventDefault();
            zoomIn();
            return true;
        }

        // Ctrl + Minus：缩小
        if ((event.ctrlKey || event.metaKey) && event.key === '-') {
            event.preventDefault();
            zoomOut();
            return true;
        }

        // Ctrl + 0：重置缩放
        if ((event.ctrlKey || event.metaKey) && event.key === '0') {
            event.preventDefault();
            resetZoom();
            return true;
        }

        return false;
    }

    /**
     * 获取预设缩放比例列表
     */
    const presetZooms = [
        { label: '10%', value: 0.1 },
        { label: '25%', value: 0.25 },
        { label: '50%', value: 0.5 },
        { label: '75%', value: 0.75 },
        { label: '100%', value: 1 },
        { label: '125%', value: 1.25 },
        { label: '150%', value: 1.5 },
        { label: '200%', value: 2 },
        { label: '300%', value: 3 },
        { label: '400%', value: 4 },
        { label: '500%', value: 5 },
    ];

    /**
     * 检查是否可以继续放大
     */
    const canZoomIn = computed(() => zoom.value < maxZoom);

    /**
     * 检查是否可以继续缩小
     */
    const canZoomOut = computed(() => zoom.value > minZoom);

    return {
        // 状态
        zoom: readonly(zoom),
        zoomPercentage: readonly(zoomPercentage),
        canZoomIn: readonly(canZoomIn),
        canZoomOut: readonly(canZoomOut),
        minZoom,
        maxZoom,
        presetZooms,

        // 方法
        setZoom,
        zoomIn,
        zoomOut,
        resetZoom,
        fitToScreen,
        handleWheel,
        handleKeyboard,
    };
}

export default useZoom;

