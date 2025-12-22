/**
 * DesignCanvas Zoom Center Point Preservation Tests
 *
 * Task 1.5: 实现缩放中心点保持
 *
 * Requirements:
 * - Requirement 21: 缩放功能
 * - Acceptance Criteria 21.4: 缩放后保持画布中心点不变
 *
 * Test Strategy:
 * - Test that zoom preserves the visual center point
 * - Test that zoom works with different center points
 * - Test that zoom respects min/max bounds (10% ~ 500%)
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';
import DesignCanvas from '../DesignCanvas.vue';
import { useDesignStore } from '@/store/design';

describe('DesignCanvas - Zoom Center Point Preservation', () => {
    let wrapper;
    let store;

    beforeEach(() => {
        // Create a fresh Pinia instance for each test
        setActivePinia(createPinia());
        store = useDesignStore();

        // Initialize store with a simple page
        store.currentPage = {
            id: 'page_001',
            name: 'Test Page',
            width: 1920,
            height: 1080,
            backgroundColor: '#ffffff',
        };

        store.components = [];

        // Mount the component
        wrapper = mount(DesignCanvas, {
            props: {
                showGrid: true,
            },
            global: {
                stubs: {
                    DomRenderer: true,
                    CanvasAuxiliary: true,
                },
            },
        });
    });

    it('should preserve center point when zooming in', async () => {
        // Get the viewport element
        const viewport = wrapper.vm.viewportRef;

        // Mock viewport dimensions
        Object.defineProperty(viewport, 'clientWidth', { value: 800, writable: true });
        Object.defineProperty(viewport, 'clientHeight', { value: 600, writable: true });
        Object.defineProperty(viewport, 'scrollLeft', { value: 100, writable: true });
        Object.defineProperty(viewport, 'scrollTop', { value: 100, writable: true });

        // Initial zoom is 1.0
        expect(wrapper.vm.zoom).toBe(1);

        // Calculate the canvas position at the center of the viewport before zoom
        const centerX = viewport.clientWidth / 2; // 400
        const centerY = viewport.clientHeight / 2; // 300
        const canvasXBefore = (viewport.scrollLeft + centerX) / 1.0; // (100 + 400) / 1.0 = 500
        const canvasYBefore = (viewport.scrollTop + centerY) / 1.0; // (100 + 300) / 1.0 = 400

        // Zoom in to 2.0
        wrapper.vm.setZoom(2.0);
        await nextTick();

        // After zoom, the canvas position at the center should remain the same
        const canvasXAfter = (viewport.scrollLeft + centerX) / 2.0;
        const canvasYAfter = (viewport.scrollTop + centerY) / 2.0;

        // Verify that the canvas position is preserved (within 1px tolerance)
        expect(Math.abs(canvasXAfter - canvasXBefore)).toBeLessThan(1);
        expect(Math.abs(canvasYAfter - canvasYBefore)).toBeLessThan(1);

        // Verify zoom value updated
        expect(wrapper.vm.zoom).toBe(2.0);
    });

    it('should preserve center point when zooming out', async () => {
        // Get the viewport element
        const viewport = wrapper.vm.viewportRef;

        // Mock viewport dimensions
        Object.defineProperty(viewport, 'clientWidth', { value: 800, writable: true });
        Object.defineProperty(viewport, 'clientHeight', { value: 600, writable: true });
        Object.defineProperty(viewport, 'scrollLeft', { value: 400, writable: true });
        Object.defineProperty(viewport, 'scrollTop', { value: 300, writable: true });

        // Start with zoom 2.0
        wrapper.vm.zoom = 2.0;
        await nextTick();

        // Calculate the canvas position at the center before zoom
        const centerX = viewport.clientWidth / 2;
        const centerY = viewport.clientHeight / 2;
        const canvasXBefore = (viewport.scrollLeft + centerX) / 2.0;
        const canvasYBefore = (viewport.scrollTop + centerY) / 2.0;

        // Zoom out to 1.0
        wrapper.vm.setZoom(1.0);
        await nextTick();

        // After zoom, the canvas position at the center should remain the same
        const canvasXAfter = (viewport.scrollLeft + centerX) / 1.0;
        const canvasYAfter = (viewport.scrollTop + centerY) / 1.0;

        // Verify that the canvas position is preserved (within 1px tolerance)
        expect(Math.abs(canvasXAfter - canvasXBefore)).toBeLessThan(1);
        expect(Math.abs(canvasYAfter - canvasYBefore)).toBeLessThan(1);

        // Verify zoom value updated
        expect(wrapper.vm.zoom).toBe(1.0);
    });

    it('should respect minimum zoom limit (10%)', async () => {
        // Try to zoom below 10%
        wrapper.vm.setZoom(0.05);
        await nextTick();

        // Should be clamped to 0.1
        expect(wrapper.vm.zoom).toBe(0.1);
    });

    it('should respect maximum zoom limit (500%)', async () => {
        // Try to zoom above 500%
        wrapper.vm.setZoom(6.0);
        await nextTick();

        // Should be clamped to 5.0
        expect(wrapper.vm.zoom).toBe(5.0);
    });

    it('should handle custom zoom center point', async () => {
        // Get the viewport element
        const viewport = wrapper.vm.viewportRef;

        // Mock viewport dimensions
        Object.defineProperty(viewport, 'clientWidth', { value: 800, writable: true });
        Object.defineProperty(viewport, 'clientHeight', { value: 600, writable: true });
        Object.defineProperty(viewport, 'scrollLeft', { value: 100, writable: true });
        Object.defineProperty(viewport, 'scrollTop', { value: 100, writable: true });

        // Use a custom center point (top-left corner)
        const customCenterX = 0;
        const customCenterY = 0;

        // Calculate canvas position at custom center before zoom
        const canvasXBefore = (viewport.scrollLeft + customCenterX) / 1.0;
        const canvasYBefore = (viewport.scrollTop + customCenterY) / 1.0;

        // Zoom with custom center point
        wrapper.vm.setZoom(2.0, customCenterX, customCenterY);
        await nextTick();

        // After zoom, the canvas position at the custom center should remain the same
        const canvasXAfter = (viewport.scrollLeft + customCenterX) / 2.0;
        const canvasYAfter = (viewport.scrollTop + customCenterY) / 2.0;

        // Verify that the canvas position is preserved at the custom center
        expect(Math.abs(canvasXAfter - canvasXBefore)).toBeLessThan(1);
        expect(Math.abs(canvasYAfter - canvasYBefore)).toBeLessThan(1);
    });

    it('should not change zoom if new value equals current value', async () => {
        const viewport = wrapper.vm.viewportRef;
        Object.defineProperty(viewport, 'scrollLeft', { value: 100, writable: true });
        Object.defineProperty(viewport, 'scrollTop', { value: 100, writable: true });

        const initialScrollLeft = viewport.scrollLeft;
        const initialScrollTop = viewport.scrollTop;

        // Try to set the same zoom value
        wrapper.vm.setZoom(1.0);
        await nextTick();

        // Scroll position should not change
        expect(viewport.scrollLeft).toBe(initialScrollLeft);
        expect(viewport.scrollTop).toBe(initialScrollTop);
    });

    it('should update scroll state after zoom', async () => {
        const viewport = wrapper.vm.viewportRef;

        // Mock viewport dimensions
        Object.defineProperty(viewport, 'clientWidth', { value: 800, writable: true });
        Object.defineProperty(viewport, 'clientHeight', { value: 600, writable: true });
        Object.defineProperty(viewport, 'scrollLeft', { value: 100, writable: true });
        Object.defineProperty(viewport, 'scrollTop', { value: 100, writable: true });

        // Zoom in
        wrapper.vm.setZoom(2.0);
        await nextTick();

        // Verify that scrollX and scrollY state are updated
        expect(wrapper.vm.scrollX).toBe(viewport.scrollLeft);
        expect(wrapper.vm.scrollY).toBe(viewport.scrollTop);
    });
});
