/**
 * useCoordinateSync - 坐标系统同步 Composable
 *
 * 职责：
 * - 获取 DOM 元素的位置和尺寸（getBoundingClientRect）
 * - 实现 DOM 到 Canvas 坐标转换
 * - 监听组件尺寸变化（ResizeObserver）
 * - 监听组件位置变化（MutationObserver）
 * - 实现坐标同步更新机制
 *
 * Requirements:
 * - Requirement 1: 混合渲染架构
 * - Acceptance Criteria 1.4: 确保两层坐标系统同步且事件不冲突
 *
 * Task: 1.4 - 实现坐标系统同步
 * Sub-tasks:
 * - [x] 创建 composables/useCoordinateSync.js
 * - [x] 实现 DOM 元素位置获取（getBoundingClientRect）
 * - [x] 实现 DOM 到 Canvas 坐标转换
 * - [x] 使用 ResizeObserver 监听组件尺寸变化
 * - [x] 使用 MutationObserver 监听组件位置变化
 * - [x] 实现坐标同步更新机制
 */
import { ref, onUnmounted } from 'vue';

/**
 * 坐标系统同步 Composable
 *
 * @param {Object} options - 配置选项
 * @param {Ref} options.canvasContainerRef - Canvas 容器引用
 * @param {Ref} options.zoom - 缩放比例
 * @param {Ref} options.scrollX - 水平滚动位置
 * @param {Ref} options.scrollY - 垂直滚动位置
 * @returns {Object} 坐标同步方法和状态
 */
export function useCoordinateSync(options = {}) {
    const { canvasContainerRef = ref(null), zoom = ref(1), scrollX = ref(0), scrollY = ref(0) } = options;

    // 组件位置和尺寸缓存
    const componentBounds = ref(new Map());

    // Observer 实例
    let resizeObserver = null;
    let mutationObserver = null;

    /**
     * 获取 DOM 元素的边界信息
     * Task 1.4 Sub-task: 实现 DOM 元素位置获取（getBoundingClientRect）
     *
     * @param {HTMLElement} element - DOM 元素
     * @returns {DOMRect|null} 边界信息
     */
    function getElementBounds(element) {
        if (!element) return null;

        try {
            const rect = element.getBoundingClientRect();
            return {
                x: rect.x,
                y: rect.y,
                width: rect.width,
                height: rect.height,
                top: rect.top,
                right: rect.right,
                bottom: rect.bottom,
                left: rect.left,
            };
        } catch (error) {
            console.error('[useCoordinateSync] Failed to get element bounds:', error);
            return null;
        }
    }

    /**
     * DOM 坐标转换为 Canvas 坐标
     * Task 1.4 Sub-task: 实现 DOM 到 Canvas 坐标转换
     *
     * 转换公式：
     * - Canvas X = (DOM X - Canvas Container X + Scroll X) / Zoom
     * - Canvas Y = (DOM Y - Canvas Container Y + Scroll Y) / Zoom
     *
     * @param {number} domX - DOM X 坐标
     * @param {number} domY - DOM Y 坐标
     * @returns {Object} Canvas 坐标 { x, y }
     */
    function domToCanvasCoords(domX, domY) {
        if (!canvasContainerRef.value) {
            return { x: domX, y: domY };
        }

        const containerBounds = getElementBounds(canvasContainerRef.value);
        if (!containerBounds) {
            return { x: domX, y: domY };
        }

        const canvasX = (domX - containerBounds.left + scrollX.value) / zoom.value;
        const canvasY = (domY - containerBounds.top + scrollY.value) / zoom.value;

        return { x: canvasX, y: canvasY };
    }

    /**
     * Canvas 坐标转换为 DOM 坐标
     *
     * 转换公式：
     * - DOM X = Canvas X * Zoom + Canvas Container X - Scroll X
     * - DOM Y = Canvas Y * Zoom + Canvas Container Y - Scroll Y
     *
     * @param {number} canvasX - Canvas X 坐标
     * @param {number} canvasY - Canvas Y 坐标
     * @returns {Object} DOM 坐标 { x, y }
     */
    function canvasToDomCoords(canvasX, canvasY) {
        if (!canvasContainerRef.value) {
            return { x: canvasX, y: canvasY };
        }

        const containerBounds = getElementBounds(canvasContainerRef.value);
        if (!containerBounds) {
            return { x: canvasX, y: canvasY };
        }

        const domX = canvasX * zoom.value + containerBounds.left - scrollX.value;
        const domY = canvasY * zoom.value + containerBounds.top - scrollY.value;

        return { x: domX, y: domY };
    }

    /**
     * 获取组件在 Canvas 中的位置和尺寸
     *
     * @param {string} componentId - 组件 ID
     * @returns {Object|null} Canvas 坐标和尺寸 { x, y, width, height }
     */
    function getComponentCanvasBounds(componentId) {
        const element = document.getElementById(componentId);
        if (!element) {
            console.warn(`[useCoordinateSync] Component element not found: ${componentId}`);
            return null;
        }

        const domBounds = getElementBounds(element);
        if (!domBounds) return null;

        // 转换左上角坐标
        const topLeft = domToCanvasCoords(domBounds.left, domBounds.top);

        // 尺寸需要除以缩放比例
        const width = domBounds.width / zoom.value;
        const height = domBounds.height / zoom.value;

        return {
            x: topLeft.x,
            y: topLeft.y,
            width,
            height,
        };
    }

    /**
     * 更新组件边界缓存
     *
     * @param {string} componentId - 组件 ID
     */
    function updateComponentBounds(componentId) {
        const bounds = getComponentCanvasBounds(componentId);
        if (bounds) {
            componentBounds.value.set(componentId, bounds);
        }
    }

    /**
     * 批量更新所有组件边界
     *
     * @param {Array<string>} componentIds - 组件 ID 列表
     */
    function updateAllComponentBounds(componentIds = []) {
        componentIds.forEach((id) => {
            updateComponentBounds(id);
        });
    }

    /**
     * 监听组件尺寸变化
     * Task 1.4 Sub-task: 使用 ResizeObserver 监听组件尺寸变化
     *
     * @param {Array<string>} componentIds - 要监听的组件 ID 列表
     * @param {Function} callback - 尺寸变化回调函数
     */
    function observeComponentResize(componentIds, callback) {
        // 清理旧的 observer
        if (resizeObserver) {
            resizeObserver.disconnect();
        }

        // 创建新的 ResizeObserver
        resizeObserver = new ResizeObserver((entries) => {
            entries.forEach((entry) => {
                const componentId = entry.target.id;
                if (componentId) {
                    // 更新缓存
                    updateComponentBounds(componentId);

                    // 调用回调
                    if (callback) {
                        const bounds = componentBounds.value.get(componentId);
                        callback(componentId, bounds, entry);
                    }
                }
            });
        });

        // 监听所有组件
        componentIds.forEach((id) => {
            const element = document.getElementById(id);
            if (element) {
                resizeObserver.observe(element);
            }
        });

        console.log(`[useCoordinateSync] Observing resize for ${componentIds.length} components`);
    }

    /**
     * 监听组件位置变化
     * Task 1.4 Sub-task: 使用 MutationObserver 监听组件位置变化
     *
     * @param {HTMLElement} containerElement - 容器元素
     * @param {Function} callback - 位置变化回调函数
     */
    function observeComponentPosition(containerElement, callback) {
        // 清理旧的 observer
        if (mutationObserver) {
            mutationObserver.disconnect();
        }

        // 创建新的 MutationObserver
        mutationObserver = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
                // 监听 style 属性变化（位置、尺寸）
                if (mutation.type === 'attributes' && mutation.attributeName === 'style') {
                    const element = mutation.target;
                    const componentId = element.id;

                    if (componentId) {
                        // 更新缓存
                        updateComponentBounds(componentId);

                        // 调用回调
                        if (callback) {
                            const bounds = componentBounds.value.get(componentId);
                            callback(componentId, bounds, mutation);
                        }
                    }
                }
            });
        });

        // 监听容器及其子元素
        if (containerElement) {
            mutationObserver.observe(containerElement, {
                attributes: true,
                attributeFilter: ['style'],
                subtree: true, // 监听所有子元素
            });

            console.log('[useCoordinateSync] Observing position changes');
        }
    }

    /**
     * 停止监听尺寸变化
     */
    function stopObservingResize() {
        if (resizeObserver) {
            resizeObserver.disconnect();
            resizeObserver = null;
            console.log('[useCoordinateSync] Stopped observing resize');
        }
    }

    /**
     * 停止监听位置变化
     */
    function stopObservingPosition() {
        if (mutationObserver) {
            mutationObserver.disconnect();
            mutationObserver = null;
            console.log('[useCoordinateSync] Stopped observing position');
        }
    }

    /**
     * 清理所有 observers
     */
    function cleanup() {
        stopObservingResize();
        stopObservingPosition();
        componentBounds.value.clear();
        console.log('[useCoordinateSync] Cleanup completed');
    }

    // 组件卸载时清理
    onUnmounted(() => {
        cleanup();
    });

    return {
        // 状态
        componentBounds,

        // 坐标转换方法
        getElementBounds,
        domToCanvasCoords,
        canvasToDomCoords,
        getComponentCanvasBounds,

        // 边界更新方法
        updateComponentBounds,
        updateAllComponentBounds,

        // Observer 方法
        observeComponentResize,
        observeComponentPosition,
        stopObservingResize,
        stopObservingPosition,

        // 清理方法
        cleanup,
    };
}
