/**
 * CanvasAuxiliary Component Tests
 *
 * Tests for the Canvas auxiliary functionality component
 * Validates Konva Stage initialization, layer creation, and event handling
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import CanvasAuxiliary from '../CanvasAuxiliary.vue';

describe('CanvasAuxiliary', () => {
    let wrapper;

    beforeEach(() => {
        // Mock Konva Stage to avoid canvas rendering issues in tests
        vi.mock('konva', () => ({
            default: {
                Stage: vi.fn().mockImplementation(() => ({
                    width: vi.fn(),
                    height: vi.fn(),
                    scale: vi.fn(),
                    position: vi.fn(),
                    batchDraw: vi.fn(),
                    add: vi.fn(),
                    on: vi.fn(),
                    destroy: vi.fn(),
                })),
                Layer: vi.fn().mockImplementation(() => ({})),
            },
        }));
    });

    afterEach(() => {
        if (wrapper) {
            wrapper.unmount();
        }
        vi.clearAllMocks();
    });

    it('should render the canvas auxiliary container', () => {
        wrapper = mount(CanvasAuxiliary, {
            props: {
                width: 1920,
                height: 1080,
            },
        });

        expect(wrapper.find('.canvas-auxiliary').exists()).toBe(true);
    });

    it('should accept required props', () => {
        wrapper = mount(CanvasAuxiliary, {
            props: {
                width: 1920,
                height: 1080,
                zoom: 1.5,
                scrollX: 100,
                scrollY: 200,
            },
        });

        expect(wrapper.props('width')).toBe(1920);
        expect(wrapper.props('height')).toBe(1080);
        expect(wrapper.props('zoom')).toBe(1.5);
        expect(wrapper.props('scrollX')).toBe(100);
        expect(wrapper.props('scrollY')).toBe(200);
    });

    it('should use default values for optional props', () => {
        wrapper = mount(CanvasAuxiliary, {
            props: {
                width: 1920,
                height: 1080,
            },
        });

        expect(wrapper.props('zoom')).toBe(1);
        expect(wrapper.props('scrollX')).toBe(0);
        expect(wrapper.props('scrollY')).toBe(0);
    });

    it('should emit canvas-ready event on mount', async () => {
        wrapper = mount(CanvasAuxiliary, {
            props: {
                width: 1920,
                height: 1080,
            },
        });

        // Wait for component to mount and emit event
        await wrapper.vm.$nextTick();

        // Check if canvas-ready event was emitted
        expect(wrapper.emitted('canvas-ready')).toBeTruthy();
    });

    it('should expose stage and layer getters', () => {
        wrapper = mount(CanvasAuxiliary, {
            props: {
                width: 1920,
                height: 1080,
            },
        });

        // Check exposed methods
        expect(wrapper.vm.getStage).toBeDefined();
        expect(wrapper.vm.getMainLayer).toBeDefined();
        expect(wrapper.vm.getRulerLayer).toBeDefined();
        expect(wrapper.vm.getGuideLayer).toBeDefined();
        expect(wrapper.vm.getSelectionLayer).toBeDefined();
    });
});
