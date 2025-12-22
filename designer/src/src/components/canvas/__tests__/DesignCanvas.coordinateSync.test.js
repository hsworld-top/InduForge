/**
 * DesignCanvas Coordinate Sync Tests
 *
 * Tests for coordinate system synchronization between DOM Layer and Canvas Layer
 * Task 1.6: 测试坐标系统同步正确性
 *
 * Requirements:
 * - Requirement 5: 坐标系统同步
 * - Acceptance Criteria 5.1: DOM 元素位置变化时，Canvas Layer 的辅助图形同步更新
 * - Acceptance Criteria 5.2: 坐标转换精度误差 < 1px
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';
import DesignCanvas from '../DesignCanvas.vue';
import { useDesignStore } from '@/store/design';

describe('DesignCanvas Coordinate Sync', () => {
    let wrapper;
    let pinia;
    let designStore;

    beforeEach(() => {
        // Create a fresh Pinia instance for each test
        pinia = createPinia();
        setActivePinia(pinia);
        designStore = useDesignStore();
    });

    afterEach(() => {
        if (wrapper) {
            wrapper.unmount();
        }
        vi.clearAllMocks();
    });

    it('should initialize coordinate sync on canvas ready', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        // Get CanvasAuxiliary component
        const canvasAuxiliary = wrapper.findComponent({ name: 'CanvasAuxiliary' });

        // Emit canvas-ready event
        await canvasAuxiliary.vm.$emit('canvas-ready', {
            stage: {},
            mainLayer: {},
            rulerLayer: {},
            guideLayer: {},
            selectionLayer: {},
        });

        await nextTick();

        // Coordinate sync should be initialized
        expect(wrapper.vm.coordinateSync).toBeDefined();
        expect(wrapper.vm.coordinateSync.updateAllComponentBounds).toBeDefined();
    });

    it('should expose coordinateSync through defineExpose', () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        // Check exposed coordinateSync
        expect(wrapper.vm.coordinateSync).toBeDefined();
        expect(typeof wrapper.vm.coordinateSync).toBe('object');
    });

    it('should expose updateCoordinateSync method', () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        // Check exposed method
        expect(wrapper.vm.updateCoordinateSync).toBeDefined();
        expect(typeof wrapper.vm.updateCoordinateSync).toBe('function');
    });

    it('should update coordinate sync when components change', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        // Spy on updateCoordinateSync
        const updateSpy = vi.spyOn(wrapper.vm, 'updateCoordinateSync');

        // Add a component to the store
        designStore.addComponent({
            id: 'comp-1',
            type: 'Text',
            props: { text: 'Hello' },
            style: { x: 100, y: 100, width: 200, height: 50 },
        });

        await nextTick();
        await nextTick(); // Wait for watch to trigger

        // updateCoordinateSync should be called
        expect(updateSpy).toHaveBeenCalled();
    });

    it('should handle coordinate sync with zoom', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        // Set zoom
        wrapper.vm.setZoom(1.5);
        await nextTick();

        // Coordinate sync should still work with zoom
        expect(wrapper.vm.coordinateSync).toBeDefined();
        expect(wrapper.vm.zoom).toBe(1.5);
    });

    it('should handle coordinate sync with scroll', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const viewport = wrapper.find('.design-canvas-viewport');
        const viewportElement = viewport.element;

        // Scroll
        viewportElement.scrollLeft = 100;
        viewportElement.scrollTop = 200;
        viewportElement.dispatchEvent(new Event('scroll'));
        await nextTick();

        // Coordinate sync should still work with scroll
        expect(wrapper.vm.coordinateSync).toBeDefined();
        expect(wrapper.vm.scrollX).toBe(100);
        expect(wrapper.vm.scrollY).toBe(200);
    });

    it('should handle coordinate sync with both zoom and scroll', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const viewport = wrapper.find('.design-canvas-viewport');
        const viewportElement = viewport.element;

        // Set zoom
        wrapper.vm.setZoom(2);
        await nextTick();

        // Scroll
        viewportElement.scrollLeft = 150;
        viewportElement.scrollTop = 250;
        viewportElement.dispatchEvent(new Event('scroll'));
        await nextTick();

        // Both zoom and scroll should be applied
        expect(wrapper.vm.zoom).toBe(2);
        expect(wrapper.vm.scrollX).toBe(150);
        expect(wrapper.vm.scrollY).toBe(250);
    });

    it('should not throw when updating coordinate sync with no DOM layer', () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        // Temporarily set domLayerRef to null
        const originalDomLayerRef = wrapper.vm.domLayerRef;
        wrapper.vm.domLayerRef = null;

        // This should not throw
        expect(() => {
            wrapper.vm.updateCoordinateSync();
        }).not.toThrow();

        // Restore domLayerRef
        wrapper.vm.domLayerRef = originalDomLayerRef;
    });

    it('should handle empty component list', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        // Ensure no components
        expect(designStore.components).toEqual([]);

        // This should not throw
        expect(() => {
            wrapper.vm.updateCoordinateSync();
        }).not.toThrow();
    });

    it('should update coordinate sync for multiple components', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        // Add multiple components
        designStore.addComponent({
            id: 'comp-1',
            type: 'Text',
            props: { text: 'Component 1' },
            style: { x: 100, y: 100, width: 200, height: 50 },
        });

        designStore.addComponent({
            id: 'comp-2',
            type: 'Text',
            props: { text: 'Component 2' },
            style: { x: 200, y: 200, width: 200, height: 50 },
        });

        designStore.addComponent({
            id: 'comp-3',
            type: 'Text',
            props: { text: 'Component 3' },
            style: { x: 300, y: 300, width: 200, height: 50 },
        });

        await nextTick();
        await nextTick();

        // All components should be tracked
        expect(designStore.components.length).toBe(3);
    });
});
