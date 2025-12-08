/**
 * useCoordinateSync 测试
 *
 * Task: 1.4 - 实现坐标系统同步
 *
 * 测试内容：
 * - DOM 元素位置获取
 * - DOM 到 Canvas 坐标转换
 * - Canvas 到 DOM 坐标转换
 * - 组件边界获取
 * - ResizeObserver 监听
 * - MutationObserver 监听
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { ref } from 'vue';
import { useCoordinateSync } from '../useCoordinateSync';

describe('useCoordinateSync', () => {
    let mockElement;
    let mockContainer;

    beforeEach(() => {
        // 创建模拟的 DOM 元素
        mockElement = {
            id: 'test-component',
            getBoundingClientRect: vi.fn(() => ({
                x: 100,
                y: 200,
                width: 300,
                height: 400,
                top: 200,
                right: 400,
                bottom: 600,
                left: 100,
            })),
        };

        mockContainer = {
            getBoundingClientRect: vi.fn(() => ({
                x: 0,
                y: 0,
                width: 1920,
                height: 1080,
                top: 0,
                right: 1920,
                bottom: 1080,
                left: 0,
            })),
        };

        // Mock document.getElementById
        global.document.getElementById = vi.fn((id) => {
            if (id === 'test-component') return mockElement;
            return null;
        });
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe('getElementBounds', () => {
        it('should get element bounds correctly', () => {
            const canvasContainerRef = ref(mockContainer);
            const { getElementBounds } = useCoordinateSync({ canvasContainerRef });

            const bounds = getElementBounds(mockElement);

            expect(bounds).toEqual({
                x: 100,
                y: 200,
                width: 300,
                height: 400,
                top: 200,
                right: 400,
                bottom: 600,
                left: 100,
            });
        });

        it('should return null for invalid element', () => {
            const canvasContainerRef = ref(mockContainer);
            const { getElementBounds } = useCoordinateSync({ canvasContainerRef });

            const bounds = getElementBounds(null);

            expect(bounds).toBeNull();
        });
    });

    describe('domToCanvasCoords', () => {
        it('should convert DOM coords to Canvas coords without zoom', () => {
            const canvasContainerRef = ref(mockContainer);
            const zoom = ref(1);
            const scrollX = ref(0);
            const scrollY = ref(0);

            const { domToCanvasCoords } = useCoordinateSync({
                canvasContainerRef,
                zoom,
                scrollX,
                scrollY,
            });

            const result = domToCanvasCoords(100, 200);

            // Canvas X = (100 - 0 + 0) / 1 = 100
            // Canvas Y = (200 - 0 + 0) / 1 = 200
            expect(result).toEqual({ x: 100, y: 200 });
        });

        it('should convert DOM coords to Canvas coords with zoom', () => {
            const canvasContainerRef = ref(mockContainer);
            const zoom = ref(2);
            const scrollX = ref(0);
            const scrollY = ref(0);

            const { domToCanvasCoords } = useCoordinateSync({
                canvasContainerRef,
                zoom,
                scrollX,
                scrollY,
            });

            const result = domToCanvasCoords(100, 200);

            // Canvas X = (100 - 0 + 0) / 2 = 50
            // Canvas Y = (200 - 0 + 0) / 2 = 100
            expect(result).toEqual({ x: 50, y: 100 });
        });

        it('should convert DOM coords to Canvas coords with scroll', () => {
            const canvasContainerRef = ref(mockContainer);
            const zoom = ref(1);
            const scrollX = ref(50);
            const scrollY = ref(100);

            const { domToCanvasCoords } = useCoordinateSync({
                canvasContainerRef,
                zoom,
                scrollX,
                scrollY,
            });

            const result = domToCanvasCoords(100, 200);

            // Canvas X = (100 - 0 + 50) / 1 = 150
            // Canvas Y = (200 - 0 + 100) / 1 = 300
            expect(result).toEqual({ x: 150, y: 300 });
        });
    });

    describe('canvasToDomCoords', () => {
        it('should convert Canvas coords to DOM coords without zoom', () => {
            const canvasContainerRef = ref(mockContainer);
            const zoom = ref(1);
            const scrollX = ref(0);
            const scrollY = ref(0);

            const { canvasToDomCoords } = useCoordinateSync({
                canvasContainerRef,
                zoom,
                scrollX,
                scrollY,
            });

            const result = canvasToDomCoords(100, 200);

            // DOM X = 100 * 1 + 0 - 0 = 100
            // DOM Y = 200 * 1 + 0 - 0 = 200
            expect(result).toEqual({ x: 100, y: 200 });
        });

        it('should convert Canvas coords to DOM coords with zoom', () => {
            const canvasContainerRef = ref(mockContainer);
            const zoom = ref(2);
            const scrollX = ref(0);
            const scrollY = ref(0);

            const { canvasToDomCoords } = useCoordinateSync({
                canvasContainerRef,
                zoom,
                scrollX,
                scrollY,
            });

            const result = canvasToDomCoords(50, 100);

            // DOM X = 50 * 2 + 0 - 0 = 100
            // DOM Y = 100 * 2 + 0 - 0 = 200
            expect(result).toEqual({ x: 100, y: 200 });
        });
    });

    describe('getComponentCanvasBounds', () => {
        it('should get component bounds in canvas coordinates', () => {
            const canvasContainerRef = ref(mockContainer);
            const zoom = ref(1);
            const scrollX = ref(0);
            const scrollY = ref(0);

            const { getComponentCanvasBounds } = useCoordinateSync({
                canvasContainerRef,
                zoom,
                scrollX,
                scrollY,
            });

            const bounds = getComponentCanvasBounds('test-component');

            expect(bounds).toEqual({
                x: 100,
                y: 200,
                width: 300,
                height: 400,
            });
        });

        it('should return null for non-existent component', () => {
            const canvasContainerRef = ref(mockContainer);
            const { getComponentCanvasBounds } = useCoordinateSync({ canvasContainerRef });

            const bounds = getComponentCanvasBounds('non-existent');

            expect(bounds).toBeNull();
        });
    });

    describe('updateComponentBounds', () => {
        it('should update component bounds cache', () => {
            const canvasContainerRef = ref(mockContainer);
            const zoom = ref(1);
            const scrollX = ref(0);
            const scrollY = ref(0);

            const { updateComponentBounds, componentBounds } = useCoordinateSync({
                canvasContainerRef,
                zoom,
                scrollX,
                scrollY,
            });

            updateComponentBounds('test-component');

            const cached = componentBounds.value.get('test-component');
            expect(cached).toEqual({
                x: 100,
                y: 200,
                width: 300,
                height: 400,
            });
        });
    });

    describe('updateAllComponentBounds', () => {
        it('should update all component bounds', () => {
            const canvasContainerRef = ref(mockContainer);
            const zoom = ref(1);
            const scrollX = ref(0);
            const scrollY = ref(0);

            const { updateAllComponentBounds, componentBounds } = useCoordinateSync({
                canvasContainerRef,
                zoom,
                scrollX,
                scrollY,
            });

            updateAllComponentBounds(['test-component']);

            expect(componentBounds.value.size).toBe(1);
            expect(componentBounds.value.has('test-component')).toBe(true);
        });
    });

    describe('cleanup', () => {
        it('should clear component bounds on cleanup', () => {
            const canvasContainerRef = ref(mockContainer);
            const { updateComponentBounds, componentBounds, cleanup } = useCoordinateSync({
                canvasContainerRef,
            });

            updateComponentBounds('test-component');
            expect(componentBounds.value.size).toBe(1);

            cleanup();
            expect(componentBounds.value.size).toBe(0);
        });
    });
});
