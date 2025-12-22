/**
 * DesignCanvas Integration Tests
 *
 * Tests for the integration between DesignCanvas and CanvasAuxiliary
 * Validates that the hybrid rendering architecture works correctly
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import DesignCanvas from '../DesignCanvas.vue';

describe('DesignCanvas Integration', () => {
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

    it('should render both DOM Layer and Canvas Layer', () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        // Check DOM Layer exists
        expect(wrapper.find('.dom-layer').exists()).toBe(true);

        // Check Canvas Layer exists
        expect(wrapper.find('.canvas-layer').exists()).toBe(true);
    });

    it('should initialize CanvasAuxiliary component', () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        // Check CanvasAuxiliary component is rendered
        const canvasAuxiliary = wrapper.findComponent({ name: 'CanvasAuxiliary' });
        expect(canvasAuxiliary.exists()).toBe(true);
    });

    it('should pass correct props to CanvasAuxiliary', () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        const canvasAuxiliary = wrapper.findComponent({ name: 'CanvasAuxiliary' });

        // Check props are passed correctly
        expect(canvasAuxiliary.props('width')).toBeDefined();
        expect(canvasAuxiliary.props('height')).toBeDefined();
        expect(canvasAuxiliary.props('zoom')).toBe(1);
        expect(canvasAuxiliary.props('scrollX')).toBe(0);
        expect(canvasAuxiliary.props('scrollY')).toBe(0);
    });

    it('should handle canvas-ready event', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        const canvasAuxiliary = wrapper.findComponent({ name: 'CanvasAuxiliary' });

        // Emit canvas-ready event
        await canvasAuxiliary.vm.$emit('canvas-ready', {
            stage: {},
            mainLayer: {},
            rulerLayer: {},
            guideLayer: {},
            selectionLayer: {},
        });

        // Wait for event to be processed
        await wrapper.vm.$nextTick();

        // The component should handle the event without errors
        expect(wrapper.exists()).toBe(true);
    });

    it('should handle canvas-click event', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        const canvasAuxiliary = wrapper.findComponent({ name: 'CanvasAuxiliary' });

        // Emit canvas-click event
        await canvasAuxiliary.vm.$emit('canvas-click', {
            x: 100,
            y: 200,
        });

        // Wait for event to be processed
        await wrapper.vm.$nextTick();

        // The component should handle the event without errors
        expect(wrapper.exists()).toBe(true);
    });

    it('should expose canvasAuxiliaryRef', () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        // Check exposed properties
        expect(wrapper.vm.canvasAuxiliaryRef).toBeDefined();
        expect(wrapper.vm.zoom).toBeDefined();
        expect(wrapper.vm.scrollX).toBeDefined();
        expect(wrapper.vm.scrollY).toBeDefined();
        expect(wrapper.vm.setZoom).toBeDefined();
    });

    it('should update zoom value', () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        // Initial zoom should be 1
        expect(wrapper.vm.zoom).toBe(1);

        // Update zoom
        wrapper.vm.setZoom(1.5);

        // Zoom should be updated
        expect(wrapper.vm.zoom).toBe(1.5);
    });

    it('should apply CSS transform scale to DOM Layer', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        // Get DOM Layer element
        const domLayer = wrapper.find('.dom-layer');
        expect(domLayer.exists()).toBe(true);

        // Initial scale should be 1
        expect(domLayer.attributes('style')).toContain('transform: scale(1)');
        expect(domLayer.attributes('style')).toContain('transform-origin: top left');

        // Update zoom
        wrapper.vm.setZoom(1.5);
        await wrapper.vm.$nextTick();

        // Scale should be updated
        expect(domLayer.attributes('style')).toContain('transform: scale(1.5)');
        expect(domLayer.attributes('style')).toContain('transform-origin: top left');
    });

    it('should apply CSS transform scale with different zoom values', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
        });

        const domLayer = wrapper.find('.dom-layer');

        // Test zoom 0.5
        wrapper.vm.setZoom(0.5);
        await wrapper.vm.$nextTick();
        expect(domLayer.attributes('style')).toContain('transform: scale(0.5)');

        // Test zoom 2
        wrapper.vm.setZoom(2);
        await wrapper.vm.$nextTick();
        expect(domLayer.attributes('style')).toContain('transform: scale(2)');

        // Test zoom 0.1
        wrapper.vm.setZoom(0.1);
        await wrapper.vm.$nextTick();
        expect(domLayer.attributes('style')).toContain('transform: scale(0.1)');
    });
});
