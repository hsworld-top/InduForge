/**
 * DragPreview 单元测试
 *
 * Task 4.7: 测试拖拽预览功能
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { DragPreview } from '../DragPreview';

// Mock Konva
vi.mock('konva', () => {
    const mockAdd = vi.fn();
    const mockDestroy = vi.fn();
    const mockPosition = vi.fn();

    return {
        default: {
            Group: vi.fn(() => ({
                add: mockAdd,
                destroy: mockDestroy,
                position: mockPosition,
            })),
            Rect: vi.fn(() => ({})),
            Text: vi.fn(() => ({})),
        },
    };
});

describe('DragPreview', () => {
    let mockLayer;
    let dragPreview;

    beforeEach(() => {
        mockLayer = {
            add: vi.fn(),
            batchDraw: vi.fn(),
        };
        dragPreview = new DragPreview(mockLayer);
    });

    it('should create a DragPreview instance', () => {
        expect(dragPreview).toBeDefined();
        expect(dragPreview.layer).toBe(mockLayer);
        expect(dragPreview.isVisible).toBe(false);
    });

    it('should show drag preview', () => {
        const component = {
            type: 'Button',
            name: '按钮',
            style: { width: 100, height: 50 },
        };

        dragPreview.show(component, 100, 200);

        expect(dragPreview.isVisible).toBe(true);
        expect(mockLayer.add).toHaveBeenCalled();
        expect(mockLayer.batchDraw).toHaveBeenCalled();
    });

    it('should hide drag preview', () => {
        const component = {
            type: 'Button',
            name: '按钮',
            style: { width: 100, height: 50 },
        };

        dragPreview.show(component, 100, 200);
        dragPreview.hide();

        expect(dragPreview.isVisible).toBe(false);
        expect(dragPreview.previewGroup).toBeNull();
    });

    it('should update position when visible', () => {
        const component = {
            type: 'Button',
            name: '按钮',
            style: { width: 100, height: 50 },
        };

        dragPreview.show(component, 100, 200);
        const initialBatchDrawCalls = mockLayer.batchDraw.mock.calls.length;

        dragPreview.updatePosition(150, 250);

        expect(mockLayer.batchDraw.mock.calls.length).toBeGreaterThan(initialBatchDrawCalls);
    });

    it('should not update position when not visible', () => {
        const initialBatchDrawCalls = mockLayer.batchDraw.mock.calls.length;

        dragPreview.updatePosition(150, 250);

        expect(mockLayer.batchDraw.mock.calls.length).toBe(initialBatchDrawCalls);
    });

    it('should use default dimensions if not provided', () => {
        const component = {
            type: 'Button',
            name: '按钮',
        };

        dragPreview.show(component, 100, 200);

        expect(dragPreview.isVisible).toBe(true);
    });

    it('should destroy properly', () => {
        const component = {
            type: 'Button',
            name: '按钮',
            style: { width: 100, height: 50 },
        };

        dragPreview.show(component, 100, 200);
        dragPreview.destroy();

        expect(dragPreview.isVisible).toBe(false);
        expect(dragPreview.layer).toBeNull();
    });
});
