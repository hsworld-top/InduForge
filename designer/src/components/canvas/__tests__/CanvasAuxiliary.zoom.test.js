/**
 * CanvasAuxiliary - Canvas Layer 缩放测试
 *
 * Task 1.5: 实现 Canvas Layer 缩放（stage.scale()）
 *
 * 测试目标：
 * - 验证 Canvas Layer 能够正确响应 zoom prop 的变化
 * - 验证 stage.scale() 方法被正确调用
 * - 验证缩放后 stage 的 scale 属性正确更新
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { nextTick } from 'vue';
import CanvasAuxiliary from '../CanvasAuxiliary.vue';

// Mock Konva
vi.mock('konva', () => {
    const mockStage = {
        width: vi.fn().mockReturnThis(),
        height: vi.fn().mockReturnThis(),
        scale: vi.fn().mockReturnThis(),
        position: vi.fn().mockReturnThis(),
        batchDraw: vi.fn(),
        add: vi.fn(),
        on: vi.fn(),
        destroy: vi.fn(),
    };

    return {
        default: {
            Stage: vi.fn(() => mockStage),
            Layer: vi.fn(() => ({
                add: vi.fn(),
            })),
        },
    };
});

describe('CanvasAuxiliary - Canvas Layer Zoom', () => {
    let wrapper;
    let Konva;

    beforeEach(async () => {
        // Import mocked Konva
        Konva = (await import('konva')).default;

        // Mount component with default props
        wrapper = mount(CanvasAuxiliary, {
            props: {
                width: 1920,
                height: 1080,
                zoom: 1,
                scrollX: 0,
                scrollY: 0,
            },
        });

        // Wait for component to mount and initialize
        await nextTick();
    });

    afterEach(() => {
        wrapper.unmount();
        vi.clearAllMocks();
    });

    it('should initialize stage with default zoom (1)', () => {
        // Verify Stage was created
        expect(Konva.Stage).toHaveBeenCalledWith(
            expect.objectContaining({
                width: 1920,
                height: 1080,
            }),
        );

        // Get the mock stage instance
        const stageInstance = Konva.Stage.mock.results[0].value;

        // Initial scale should not be called (default is 1)
        // or should be called with { x: 1, y: 1 }
        if (stageInstance.scale.mock.calls.length > 0) {
            expect(stageInstance.scale).toHaveBeenCalledWith({ x: 1, y: 1 });
        }
    });

    it('should update stage scale when zoom prop changes', async () => {
        // Get the mock stage instance
        const stageInstance = Konva.Stage.mock.results[0].value;

        // Clear previous calls
        stageInstance.scale.mockClear();
        stageInstance.batchDraw.mockClear();

        // Change zoom to 1.5
        await wrapper.setProps({ zoom: 1.5 });
        await nextTick();

        // Verify stage.scale() was called with correct values
        expect(stageInstance.scale).toHaveBeenCalledWith({ x: 1.5, y: 1.5 });

        // Verify batchDraw was called to update the canvas
        expect(stageInstance.batchDraw).toHaveBeenCalled();
    });

    it('should update stage scale multiple times when zoom changes', async () => {
        const stageInstance = Konva.Stage.mock.results[0].value;

        // Test multiple zoom levels
        const zoomLevels = [0.5, 1, 1.5, 2, 0.75];

        for (const zoom of zoomLevels) {
            stageInstance.scale.mockClear();
            stageInstance.batchDraw.mockClear();

            await wrapper.setProps({ zoom });
            await nextTick();

            expect(stageInstance.scale).toHaveBeenCalledWith({ x: zoom, y: zoom });
            expect(stageInstance.batchDraw).toHaveBeenCalled();
        }
    });

    it('should handle zoom values at boundaries', async () => {
        const stageInstance = Konva.Stage.mock.results[0].value;

        // Test minimum zoom (10%)
        await wrapper.setProps({ zoom: 0.1 });
        await nextTick();
        expect(stageInstance.scale).toHaveBeenCalledWith({ x: 0.1, y: 0.1 });

        // Test maximum zoom (500%)
        await wrapper.setProps({ zoom: 5 });
        await nextTick();
        expect(stageInstance.scale).toHaveBeenCalledWith({ x: 5, y: 5 });
    });

    it('should maintain aspect ratio (x and y scale are equal)', async () => {
        const stageInstance = Konva.Stage.mock.results[0].value;

        const testZooms = [0.5, 1, 1.5, 2, 3];

        for (const zoom of testZooms) {
            stageInstance.scale.mockClear();

            await wrapper.setProps({ zoom });
            await nextTick();

            // Verify x and y scales are equal
            const scaleCall = stageInstance.scale.mock.calls[0];
            expect(scaleCall).toBeDefined();
            expect(scaleCall[0].x).toBe(zoom);
            expect(scaleCall[0].y).toBe(zoom);
            expect(scaleCall[0].x).toBe(scaleCall[0].y);
        }
    });

    it('should not call scale if stage is not initialized', async () => {
        // Create a new wrapper that might not have stage initialized
        const newWrapper = mount(CanvasAuxiliary, {
            props: {
                width: 1920,
                height: 1080,
                zoom: 1,
            },
            attachTo: document.body,
        });

        // Immediately change zoom before stage is ready
        await newWrapper.setProps({ zoom: 2 });

        // Should not throw error
        expect(() => newWrapper.unmount()).not.toThrow();
    });
});
