/**
 * DesignCanvas Scroll Tests
 *
 * Tests for scroll event monitoring and synchronization
 * Task 1.5: 实现滚动事件监听
 *
 * Requirements:
 * - Requirement 5: 坐标系统同步
 * - Acceptance Criteria 5.4: 画布滚动时同步更新 DOM Layer 和 Canvas Layer 的滚动位置
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';
import DesignCanvas from '../DesignCanvas.vue';

describe('DesignCanvas Scroll Event Monitoring', () => {
    let wrapper;
    let pinia;

    beforeEach(() => {
        // Create a fresh Pinia instance for each test
        pinia = createPinia();
        setActivePinia(pinia);
    });

    afterEach(() => {
        if (wrapper) {
            wrapper.unmount();
        }
        vi.clearAllMocks();
    });

    it('should initialize scroll values to 0', () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        // Initial scroll values should be 0
        expect(wrapper.vm.scrollX).toBe(0);
        expect(wrapper.vm.scrollY).toBe(0);
    });

    it('should pass scroll values to CanvasAuxiliary', () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        const canvasAuxiliary = wrapper.findComponent({ name: 'CanvasAuxiliary' });

        // Check scroll props are passed
        expect(canvasAuxiliary.props('scrollX')).toBe(0);
        expect(canvasAuxiliary.props('scrollY')).toBe(0);
    });

    it('should update scroll values when viewport scrolls', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body, // Attach to body to enable scroll events
        });

        // Get viewport element
        const viewport = wrapper.find('.design-canvas-viewport');
        expect(viewport.exists()).toBe(true);

        // Simulate scroll event
        const viewportElement = viewport.element;
        viewportElement.scrollLeft = 100;
        viewportElement.scrollTop = 200;

        // Trigger scroll event
        viewportElement.dispatchEvent(new Event('scroll'));
        await nextTick();

        // Scroll values should be updated
        expect(wrapper.vm.scrollX).toBe(100);
        expect(wrapper.vm.scrollY).toBe(200);
    });

    it('should update CanvasAuxiliary props when scroll changes', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const canvasAuxiliary = wrapper.findComponent({ name: 'CanvasAuxiliary' });
        const viewport = wrapper.find('.design-canvas-viewport');
        const viewportElement = viewport.element;

        // Simulate scroll
        viewportElement.scrollLeft = 150;
        viewportElement.scrollTop = 250;
        viewportElement.dispatchEvent(new Event('scroll'));
        await nextTick();

        // CanvasAuxiliary props should be updated
        expect(canvasAuxiliary.props('scrollX')).toBe(150);
        expect(canvasAuxiliary.props('scrollY')).toBe(250);
    });

    it('should handle multiple scroll events', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const viewport = wrapper.find('.design-canvas-viewport');
        const viewportElement = viewport.element;

        // First scroll
        viewportElement.scrollLeft = 50;
        viewportElement.scrollTop = 100;
        viewportElement.dispatchEvent(new Event('scroll'));
        await nextTick();

        expect(wrapper.vm.scrollX).toBe(50);
        expect(wrapper.vm.scrollY).toBe(100);

        // Second scroll
        viewportElement.scrollLeft = 150;
        viewportElement.scrollTop = 200;
        viewportElement.dispatchEvent(new Event('scroll'));
        await nextTick();

        expect(wrapper.vm.scrollX).toBe(150);
        expect(wrapper.vm.scrollY).toBe(200);

        // Third scroll
        viewportElement.scrollLeft = 0;
        viewportElement.scrollTop = 0;
        viewportElement.dispatchEvent(new Event('scroll'));
        await nextTick();

        expect(wrapper.vm.scrollX).toBe(0);
        expect(wrapper.vm.scrollY).toBe(0);
    });

    it('should handle scroll event when viewport is null', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        // Call handleScroll directly with null viewport
        // This should not throw an error
        const viewport = wrapper.find('.design-canvas-viewport');
        const viewportElement = viewport.element;

        // Temporarily set viewportRef to null
        const originalViewportRef = wrapper.vm.viewportRef;
        wrapper.vm.viewportRef = null;

        // This should not throw
        expect(() => {
            viewportElement.dispatchEvent(new Event('scroll'));
        }).not.toThrow();

        // Restore viewportRef
        wrapper.vm.viewportRef = originalViewportRef;
    });

    it('should clean up scroll event listener on unmount', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const viewport = wrapper.find('.design-canvas-viewport');
        const viewportElement = viewport.element;

        // Spy on removeEventListener
        const removeEventListenerSpy = vi.spyOn(viewportElement, 'removeEventListener');

        // Unmount component
        wrapper.unmount();

        // removeEventListener should be called for scroll event
        expect(removeEventListenerSpy).toHaveBeenCalledWith('scroll', expect.any(Function));
    });

    it('should synchronize scroll with zoom', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const canvasAuxiliary = wrapper.findComponent({ name: 'CanvasAuxiliary' });
        const viewport = wrapper.find('.design-canvas-viewport');
        const viewportElement = viewport.element;

        // Set zoom
        wrapper.vm.setZoom(1.5);
        await nextTick();

        // Scroll
        viewportElement.scrollLeft = 100;
        viewportElement.scrollTop = 200;
        viewportElement.dispatchEvent(new Event('scroll'));
        await nextTick();

        // Both zoom and scroll should be passed to CanvasAuxiliary
        expect(canvasAuxiliary.props('zoom')).toBe(1.5);
        expect(canvasAuxiliary.props('scrollX')).toBe(100);
        expect(canvasAuxiliary.props('scrollY')).toBe(200);
    });

    it('should handle horizontal scroll only', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const viewport = wrapper.find('.design-canvas-viewport');
        const viewportElement = viewport.element;

        // Horizontal scroll only
        viewportElement.scrollLeft = 300;
        viewportElement.scrollTop = 0;
        viewportElement.dispatchEvent(new Event('scroll'));
        await nextTick();

        expect(wrapper.vm.scrollX).toBe(300);
        expect(wrapper.vm.scrollY).toBe(0);
    });

    it('should handle vertical scroll only', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const viewport = wrapper.find('.design-canvas-viewport');
        const viewportElement = viewport.element;

        // Vertical scroll only
        viewportElement.scrollLeft = 0;
        viewportElement.scrollTop = 400;
        viewportElement.dispatchEvent(new Event('scroll'));
        await nextTick();

        expect(wrapper.vm.scrollX).toBe(0);
        expect(wrapper.vm.scrollY).toBe(400);
    });

    it('should expose scroll values through defineExpose', () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        // Check exposed scroll values
        expect(wrapper.vm.scrollX).toBeDefined();
        expect(wrapper.vm.scrollY).toBeDefined();
        expect(typeof wrapper.vm.scrollX).toBe('number');
        expect(typeof wrapper.vm.scrollY).toBe('number');
    });
});
