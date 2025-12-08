/**
 * DesignCanvas Coordinate Sync Property-Based Tests
 *
 * Property-based tests for coordinate system consistency
 * Task 1.6: Property-based 测试：坐标一致性
 *
 * Requirements:
 * - Requirement 5: 坐标系统同步
 * - Acceptance Criteria 5.2: 坐标转换精度误差 < 1px
 *
 * Property: 对于任意组件位置和缩放比例，DOM 坐标和 Canvas 坐标应该保持一致
 */
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { nextTick } from 'vue';
import DesignCanvas from '../DesignCanvas.vue';
import { useDesignStore } from '@/store/design';

/**
 * Generate random test cases for property-based testing
 */
function generateTestCases(count = 20) {
    const cases = [];
    for (let i = 0; i < count; i++) {
        cases.push({
            x: Math.floor(Math.random() * 2000),
            y: Math.floor(Math.random() * 2000),
            width: Math.floor(Math.random() * 500) + 50,
            height: Math.floor(Math.random() * 500) + 50,
            zoom: Math.random() * 4.9 + 0.1, // 0.1 to 5.0
            scrollX: Math.floor(Math.random() * 1000),
            scrollY: Math.floor(Math.random() * 1000),
        });
    }
    return cases;
}

describe('DesignCanvas Coordinate Sync - Property-Based Tests', () => {
    let wrapper;
    let pinia;
    let designStore;

    beforeEach(() => {
        pinia = createPinia();
        setActivePinia(pinia);
        designStore = useDesignStore();
    });

    afterEach(() => {
        if (wrapper) {
            wrapper.unmount();
        }
    });

    it('Property: DOM coordinates should match Canvas coordinates for any zoom level', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const testCases = generateTestCases(20);

        for (const testCase of testCases) {
            // Set zoom
            wrapper.vm.setZoom(testCase.zoom);
            await nextTick();

            // Verify zoom is applied correctly
            expect(wrapper.vm.zoom).toBeCloseTo(testCase.zoom, 2);

            // Verify DOM Layer has correct transform
            const domLayer = wrapper.find('.dom-layer');
            const style = domLayer.attributes('style');
            expect(style).toContain(`scale(${testCase.zoom})`);
            expect(style).toContain('transform-origin: top left');

            // Verify CanvasAuxiliary receives correct zoom
            const canvasAuxiliary = wrapper.findComponent({ name: 'CanvasAuxiliary' });
            expect(canvasAuxiliary.props('zoom')).toBeCloseTo(testCase.zoom, 2);
        }
    });

    it('Property: Scroll position should sync correctly for any scroll values', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const testCases = generateTestCases(20);
        const viewport = wrapper.find('.design-canvas-viewport');
        const viewportElement = viewport.element;

        for (const testCase of testCases) {
            // Set scroll position
            viewportElement.scrollLeft = testCase.scrollX;
            viewportElement.scrollTop = testCase.scrollY;
            viewportElement.dispatchEvent(new Event('scroll'));
            await nextTick();

            // Verify scroll values are synced
            expect(wrapper.vm.scrollX).toBe(testCase.scrollX);
            expect(wrapper.vm.scrollY).toBe(testCase.scrollY);

            // Verify CanvasAuxiliary receives correct scroll values
            const canvasAuxiliary = wrapper.findComponent({ name: 'CanvasAuxiliary' });
            expect(canvasAuxiliary.props('scrollX')).toBe(testCase.scrollX);
            expect(canvasAuxiliary.props('scrollY')).toBe(testCase.scrollY);
        }
    });

    it('Property: Zoom and scroll should work together correctly', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const testCases = generateTestCases(20);
        const viewport = wrapper.find('.design-canvas-viewport');
        const viewportElement = viewport.element;

        for (const testCase of testCases) {
            // Set zoom
            wrapper.vm.setZoom(testCase.zoom);
            await nextTick();

            // Set scroll
            viewportElement.scrollLeft = testCase.scrollX;
            viewportElement.scrollTop = testCase.scrollY;
            viewportElement.dispatchEvent(new Event('scroll'));
            await nextTick();

            // Verify both zoom and scroll are applied
            expect(wrapper.vm.zoom).toBeCloseTo(testCase.zoom, 2);
            expect(wrapper.vm.scrollX).toBe(testCase.scrollX);
            expect(wrapper.vm.scrollY).toBe(testCase.scrollY);

            // Verify CanvasAuxiliary receives both
            const canvasAuxiliary = wrapper.findComponent({ name: 'CanvasAuxiliary' });
            expect(canvasAuxiliary.props('zoom')).toBeCloseTo(testCase.zoom, 2);
            expect(canvasAuxiliary.props('scrollX')).toBe(testCase.scrollX);
            expect(canvasAuxiliary.props('scrollY')).toBe(testCase.scrollY);
        }
    });

    it('Property: Coordinate sync should maintain consistency across component updates', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        // Create a page first
        designStore.createPage({
            id: 'test-page',
            name: 'Test Page',
            components: [],
        });
        designStore.setCurrentPage('test-page');

        const testCases = generateTestCases(10);

        for (let i = 0; i < testCases.length; i++) {
            const testCase = testCases[i];

            // Add component
            designStore.addComponent({
                id: `comp-${i}`,
                type: 'Text',
                props: { text: `Component ${i}` },
                style: {
                    x: testCase.x,
                    y: testCase.y,
                    width: testCase.width,
                    height: testCase.height,
                },
            });

            await nextTick();
            await nextTick();

            // Verify component is added
            expect(designStore.components.length).toBe(i + 1);

            // Verify coordinate sync is still working
            expect(wrapper.vm.coordinateSync).toBeDefined();
        }
    });

    it('Property: Zoom center point preservation should work for any center point', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const viewport = wrapper.find('.design-canvas-viewport');
        const viewportElement = viewport.element;

        // Set initial scroll position
        viewportElement.scrollLeft = 500;
        viewportElement.scrollTop = 500;
        viewportElement.dispatchEvent(new Event('scroll'));
        await nextTick();

        const testCases = [
            { zoom: 1.5, centerX: 100, centerY: 100 },
            { zoom: 2.0, centerX: 200, centerY: 200 },
            { zoom: 0.5, centerX: 300, centerY: 300 },
            { zoom: 3.0, centerX: 400, centerY: 400 },
            { zoom: 0.8, centerX: 150, centerY: 250 },
        ];

        for (const testCase of testCases) {
            const oldZoom = wrapper.vm.zoom;
            const oldScrollX = wrapper.vm.scrollX;
            const oldScrollY = wrapper.vm.scrollY;

            // Calculate expected canvas position at center
            const expectedCanvasX = (oldScrollX + testCase.centerX) / oldZoom;
            const expectedCanvasY = (oldScrollY + testCase.centerY) / oldZoom;

            // Set zoom with custom center point
            wrapper.vm.setZoom(testCase.zoom, testCase.centerX, testCase.centerY);
            await nextTick();

            // Verify zoom is applied
            expect(wrapper.vm.zoom).toBeCloseTo(testCase.zoom, 2);

            // Calculate actual canvas position at center after zoom
            const actualCanvasX = (wrapper.vm.scrollX + testCase.centerX) / testCase.zoom;
            const actualCanvasY = (wrapper.vm.scrollY + testCase.centerY) / testCase.zoom;

            // Verify canvas position at center is preserved (within 1px tolerance)
            expect(Math.abs(actualCanvasX - expectedCanvasX)).toBeLessThan(1);
            expect(Math.abs(actualCanvasY - expectedCanvasY)).toBeLessThan(1);
        }
    });

    it('Property: Zoom limits should be enforced for any input', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const testCases = [
            { input: 0.05, expected: 0.1 }, // Below minimum
            { input: 0.1, expected: 0.1 }, // At minimum
            { input: 1.0, expected: 1.0 }, // Normal
            { input: 5.0, expected: 5.0 }, // At maximum
            { input: 10.0, expected: 5.0 }, // Above maximum
            { input: -1.0, expected: 0.1 }, // Negative
            { input: 0, expected: 0.1 }, // Zero
        ];

        for (const testCase of testCases) {
            wrapper.vm.setZoom(testCase.input);
            await nextTick();

            expect(wrapper.vm.zoom).toBeCloseTo(testCase.expected, 2);
        }
    });

    it('Property: Multiple zoom operations should maintain coordinate consistency', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        const zoomSequence = [1.0, 1.5, 2.0, 1.2, 0.8, 1.0, 2.5, 0.5, 1.0];

        for (const zoom of zoomSequence) {
            wrapper.vm.setZoom(zoom);
            await nextTick();

            // Verify zoom is applied
            expect(wrapper.vm.zoom).toBeCloseTo(zoom, 2);

            // Verify DOM Layer transform
            const domLayer = wrapper.find('.dom-layer');
            expect(domLayer.attributes('style')).toContain(`scale(${zoom})`);

            // Verify CanvasAuxiliary receives correct zoom
            const canvasAuxiliary = wrapper.findComponent({ name: 'CanvasAuxiliary' });
            expect(canvasAuxiliary.props('zoom')).toBeCloseTo(zoom, 2);
        }
    });

    it('Property: Coordinate sync should work with rapid component additions', async () => {
        wrapper = mount(DesignCanvas, {
            global: {
                plugins: [pinia],
            },
            attachTo: document.body,
        });

        // Create a page first
        designStore.createPage({
            id: 'test-page-rapid',
            name: 'Test Page Rapid',
            components: [],
        });
        designStore.setCurrentPage('test-page-rapid');

        // Rapidly add 50 components
        const promises = [];
        for (let i = 0; i < 50; i++) {
            designStore.addComponent({
                id: `rapid-comp-${i}`,
                type: 'Text',
                props: { text: `Rapid ${i}` },
                style: {
                    x: Math.random() * 1000,
                    y: Math.random() * 1000,
                    width: 100,
                    height: 50,
                },
            });
            promises.push(nextTick());
        }

        await Promise.all(promises);
        await nextTick();

        // Verify all components are added
        expect(designStore.components.length).toBe(50);

        // Verify coordinate sync is still working
        expect(wrapper.vm.coordinateSync).toBeDefined();
        expect(() => wrapper.vm.updateCoordinateSync()).not.toThrow();
    });
});
