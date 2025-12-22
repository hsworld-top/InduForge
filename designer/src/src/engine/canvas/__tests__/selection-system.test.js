/**
 * Canvas 层选择系统测试
 * Task 5.9: 测试选择框和辅助线系统
 *
 * 测试内容：
 * - SelectionBox: 选择框和控制点
 * - SelectionRect: 框选矩形
 * - AlignmentGuides: 对齐辅助线
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import Konva from 'konva';
import SelectionBox from '../SelectionBox';
import SelectionRect from '../SelectionRect';
import AlignmentGuides from '../AlignmentGuides';

describe('Canvas Layer Selection System', () => {
    let stage;
    let layer;

    beforeEach(() => {
        // 创建一个测试用的 Konva Stage
        const container = document.createElement('div');
        document.body.appendChild(container);

        stage = new Konva.Stage({
            container: container,
            width: 800,
            height: 600,
        });

        layer = new Konva.Layer();
        stage.add(layer);
    });

    afterEach(() => {
        stage.destroy();
        document.body.innerHTML = '';
    });

    describe('Task 5.2: SelectionBox', () => {
        let selectionBox;

        beforeEach(() => {
            selectionBox = new SelectionBox(layer);
        });

        afterEach(() => {
            selectionBox.destroy();
        });

        it('should create a selection box instance', () => {
            expect(selectionBox).toBeDefined();
            expect(selectionBox.transformer).toBeDefined();
        });

        it('should initially be hidden', () => {
            expect(selectionBox.transformer.visible()).toBe(false);
        });

        it('should update selection box position and size', () => {
            const domElement = document.createElement('div');
            domElement.style.position = 'absolute';
            domElement.style.left = '100px';
            domElement.style.top = '100px';
            domElement.style.width = '200px';
            domElement.style.height = '150px';
            document.body.appendChild(domElement);

            // Mock getBoundingClientRect
            domElement.getBoundingClientRect = () => ({
                left: 100,
                top: 100,
                right: 300,
                bottom: 250,
                width: 200,
                height: 150,
            });

            selectionBox.update([domElement], 1, { x: 0, y: 0 });

            expect(selectionBox.transformer.visible()).toBe(true);
            expect(selectionBox.transformer.x()).toBe(100);
            expect(selectionBox.transformer.y()).toBe(100);
        });

        it('should hide when no elements provided', () => {
            selectionBox.update([], 1, { x: 0, y: 0 });
            expect(selectionBox.transformer.visible()).toBe(false);
        });

        it('should have resize event callbacks', () => {
            let resizeStartCalled = false;
            let resizeCalled = false;
            let resizeEndCalled = false;

            selectionBox = new SelectionBox(layer, {
                onResizeStart: () => {
                    resizeStartCalled = true;
                },
                onResize: () => {
                    resizeCalled = true;
                },
                onResizeEnd: () => {
                    resizeEndCalled = true;
                },
            });

            // 触发回调
            if (selectionBox.options.onResizeStart) selectionBox.options.onResizeStart();
            if (selectionBox.options.onResize) selectionBox.options.onResize();
            if (selectionBox.options.onResizeEnd) selectionBox.options.onResizeEnd();

            expect(resizeStartCalled).toBe(true);
            expect(resizeCalled).toBe(true);
            expect(resizeEndCalled).toBe(true);
        });

        it('should have rotate event callbacks', () => {
            let rotateStartCalled = false;
            let rotateCalled = false;
            let rotateEndCalled = false;

            selectionBox = new SelectionBox(layer, {
                onRotateStart: () => {
                    rotateStartCalled = true;
                },
                onRotate: () => {
                    rotateCalled = true;
                },
                onRotateEnd: () => {
                    rotateEndCalled = true;
                },
            });

            // 触发回调
            if (selectionBox.options.onRotateStart) selectionBox.options.onRotateStart();
            if (selectionBox.options.onRotate) selectionBox.options.onRotate();
            if (selectionBox.options.onRotateEnd) selectionBox.options.onRotateEnd();

            expect(rotateStartCalled).toBe(true);
            expect(rotateCalled).toBe(true);
            expect(rotateEndCalled).toBe(true);
        });
    });

    describe('Task 5.7: SelectionRect', () => {
        let selectionRect;

        beforeEach(() => {
            selectionRect = new SelectionRect(layer);
        });

        afterEach(() => {
            selectionRect.destroy();
        });

        it('should create a selection rect instance', () => {
            expect(selectionRect).toBeDefined();
            expect(selectionRect.rect).toBeDefined();
        });

        it('should initially be hidden', () => {
            expect(selectionRect.isVisible()).toBe(false);
        });

        it('should start selection at a point', () => {
            selectionRect.start({ x: 100, y: 150 });

            expect(selectionRect.isVisible()).toBe(true);
            expect(selectionRect.rect.x()).toBe(100);
            expect(selectionRect.rect.y()).toBe(150);
        });

        it('should update selection rect size', () => {
            selectionRect.start({ x: 100, y: 150 });
            selectionRect.update({ x: 300, y: 250 });

            const rect = selectionRect.getRect();
            expect(rect.x).toBe(100);
            expect(rect.y).toBe(150);
            expect(rect.width).toBe(200); // 300 - 100
            expect(rect.height).toBe(100); // 250 - 150
        });

        it('should handle reverse selection (dragging up-left)', () => {
            selectionRect.start({ x: 300, y: 250 });
            selectionRect.update({ x: 100, y: 150 });

            const rect = selectionRect.getRect();
            expect(rect.x).toBe(100); // min(300, 100)
            expect(rect.y).toBe(150); // min(250, 150)
            expect(rect.width).toBe(200);
            expect(rect.height).toBe(100);
        });

        it('should check if component is in selection', () => {
            selectionRect.start({ x: 0, y: 0 });
            selectionRect.update({ x: 200, y: 200 });

            const insideComponent = { x: 50, y: 50, width: 100, height: 100 };
            const outsideComponent = { x: 300, y: 300, width: 100, height: 100 };

            expect(selectionRect.isComponentInSelection(insideComponent)).toBe(true);
            expect(selectionRect.isComponentInSelection(outsideComponent)).toBe(false);
        });

        it('should get selected components from list', () => {
            selectionRect.start({ x: 0, y: 0 });
            selectionRect.update({ x: 200, y: 200 });

            const components = [
                { id: 'comp-1', style: { left: 50, top: 50, width: 100, height: 100 } },
                { id: 'comp-2', style: { left: 300, top: 300, width: 100, height: 100 } },
                { id: 'comp-3', style: { left: 150, top: 150, width: 100, height: 100 } },
            ];

            const selectedIds = selectionRect.getSelectedComponents(components);

            expect(selectedIds).toContain('comp-1');
            expect(selectedIds).toContain('comp-3'); // 部分重叠
            expect(selectedIds).not.toContain('comp-2');
        });

        it('should hide and clear selection', () => {
            selectionRect.start({ x: 100, y: 100 });
            selectionRect.update({ x: 200, y: 200 });

            expect(selectionRect.isVisible()).toBe(true);

            selectionRect.hide();

            expect(selectionRect.isVisible()).toBe(false);
            expect(selectionRect.getRect()).toBeNull();
        });

        it('should return selection rect on end', () => {
            selectionRect.start({ x: 100, y: 100 });
            selectionRect.update({ x: 300, y: 250 });

            const result = selectionRect.end();

            expect(result).toEqual({
                x: 100,
                y: 100,
                width: 200,
                height: 150,
            });

            expect(selectionRect.isVisible()).toBe(false);
        });
    });

    describe('Task 5.5 & 5.6: AlignmentGuides', () => {
        let alignmentGuides;

        beforeEach(() => {
            alignmentGuides = new AlignmentGuides(layer);
        });

        afterEach(() => {
            alignmentGuides.destroy();
        });

        it('should create an alignment guides instance', () => {
            expect(alignmentGuides).toBeDefined();
        });

        it('should initially have no guides', () => {
            expect(alignmentGuides.getActiveGuides()).toEqual([]);
        });

        it('should detect left edge alignment', () => {
            const movingRect = { x: 100, y: 100, width: 100, height: 100 };
            const otherRects = [{ x: 100, y: 200, width: 100, height: 100 }];

            const alignment = alignmentGuides.checkAlignment(movingRect, otherRects);

            expect(alignment).toBeDefined();
            expect(alignment.type).toContain('LEFT');
        });

        it('should detect horizontal center alignment', () => {
            const movingRect = { x: 100, y: 100, width: 100, height: 100 };
            const otherRects = [{ x: 100, y: 200, width: 100, height: 100 }];

            const alignment = alignmentGuides.checkAlignment(movingRect, otherRects);

            expect(alignment.type).toContain('CENTER_HORIZONTAL');
        });

        it('should detect vertical center alignment', () => {
            const movingRect = { x: 100, y: 100, width: 100, height: 100 };
            const otherRects = [{ x: 200, y: 100, width: 100, height: 100 }];

            const alignment = alignmentGuides.checkAlignment(movingRect, otherRects);

            expect(alignment.type).toContain('CENTER_VERTICAL');
        });

        it('should calculate snap position', () => {
            const movingRect = { x: 102, y: 100, width: 100, height: 100 };
            const otherRects = [{ x: 100, y: 200, width: 100, height: 100 }];

            const snapPosition = alignmentGuides.getSnapPosition(movingRect, otherRects);

            expect(snapPosition).toBeDefined();
            expect(snapPosition.x).toBe(100); // 应该吸附到 100
        });

        it('should not snap if distance is too large', () => {
            const movingRect = { x: 120, y: 100, width: 100, height: 100 };
            const otherRects = [{ x: 100, y: 200, width: 100, height: 100 }];

            const snapPosition = alignmentGuides.getSnapPosition(movingRect, otherRects);

            // 距离超过阈值，不应吸附
            expect(snapPosition).toBeNull();
        });

        it('should detect canvas edge alignment', () => {
            const movingRect = { x: 2, y: 100, width: 100, height: 100 };
            const canvasBounds = { x: 0, y: 0, width: 800, height: 600 };

            const alignment = alignmentGuides.checkAlignment(movingRect, [], canvasBounds);

            expect(alignment).toBeDefined();
            expect(alignment.type).toContain('LEFT');
        });

        it('should show alignment guides', () => {
            const alignment = {
                type: ['LEFT', 'TOP'],
                lines: [
                    { x1: 100, y1: 0, x2: 100, y2: 600 },
                    { x1: 0, y1: 100, x2: 800, y2: 100 },
                ],
            };

            alignmentGuides.showGuides(alignment);

            const activeGuides = alignmentGuides.getActiveGuides();
            expect(activeGuides.length).toBeGreaterThan(0);
        });

        it('should hide alignment guides', () => {
            const alignment = {
                type: ['LEFT'],
                lines: [{ x1: 100, y1: 0, x2: 100, y2: 600 }],
            };

            alignmentGuides.showGuides(alignment);
            expect(alignmentGuides.getActiveGuides().length).toBeGreaterThan(0);

            alignmentGuides.hideGuides();
            expect(alignmentGuides.getActiveGuides()).toEqual([]);
        });
    });
});

