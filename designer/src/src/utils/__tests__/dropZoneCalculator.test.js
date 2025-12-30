/**
 * Drop Zone Calculator Tests
 * 
 * 测试插入位置计算逻辑
 */

import { describe, it, expect } from 'vitest';
import {
    calculateFlexInsertPosition,
    calculateGridInsertPosition,
    calculateBlockInsertPosition,
    calculateRowColInsertPosition,
    isMouseOverContainer,
    calculateInsertPosition,
} from '../dropZoneCalculator';

describe('Drop Zone Calculator', () => {
    describe('isMouseOverContainer', () => {
        it('should detect mouse inside container', () => {
            const containerRect = {
                left: 100,
                top: 100,
                right: 300,
                bottom: 300,
                width: 200,
                height: 200,
            };

            expect(isMouseOverContainer(containerRect, { x: 150, y: 150 })).toBe(true);
            expect(isMouseOverContainer(containerRect, { x: 200, y: 200 })).toBe(true);
            expect(isMouseOverContainer(containerRect, { x: 100, y: 100 })).toBe(true);
            expect(isMouseOverContainer(containerRect, { x: 300, y: 300 })).toBe(true);
        });

        it('should detect mouse outside container', () => {
            const containerRect = {
                left: 100,
                top: 100,
                right: 300,
                bottom: 300,
                width: 200,
                height: 200,
            };

            expect(isMouseOverContainer(containerRect, { x: 50, y: 150 })).toBe(false);
            expect(isMouseOverContainer(containerRect, { x: 350, y: 150 })).toBe(false);
            expect(isMouseOverContainer(containerRect, { x: 150, y: 50 })).toBe(false);
            expect(isMouseOverContainer(containerRect, { x: 150, y: 350 })).toBe(false);
        });
    });

    describe('calculateFlexInsertPosition', () => {
        it('should calculate insert position in empty flex container', () => {
            const containerRect = {
                left: 0,
                top: 0,
                right: 400,
                bottom: 300,
                width: 400,
                height: 300,
            };

            const result = calculateFlexInsertPosition(containerRect, [], { x: 200, y: 150 }, 'row');

            expect(result.index).toBe(0);
            expect(result.insertLine).toBeDefined();
            expect(result.insertLine.x).toBe(0);
        });

        it('should calculate insert position before first child in row layout', () => {
            const containerRect = {
                left: 0,
                top: 0,
                right: 400,
                bottom: 300,
                width: 400,
                height: 300,
            };

            const childrenRects = [
                { left: 100, top: 0, right: 200, bottom: 100, width: 100, height: 100 },
                { left: 200, top: 0, right: 300, bottom: 100, width: 100, height: 100 },
            ];

            const result = calculateFlexInsertPosition(containerRect, childrenRects, { x: 50, y: 50 }, 'row');

            expect(result.index).toBe(0);
            expect(result.insertLine).toBeDefined();
        });

        it('should calculate insert position in column layout', () => {
            const containerRect = {
                left: 0,
                top: 0,
                right: 300,
                bottom: 400,
                width: 300,
                height: 400,
            };

            const childrenRects = [
                { left: 0, top: 0, right: 300, bottom: 100, width: 300, height: 100 },
                { left: 0, top: 100, right: 300, bottom: 200, width: 300, height: 100 },
            ];

            // Y=50 is at the center of first child (0-100), so it should insert after
            const result = calculateFlexInsertPosition(containerRect, childrenRects, { x: 150, y: 50 }, 'column');

            expect(result.index).toBe(1);
            expect(result.insertLine).toBeDefined();
        });
    });

    describe('calculateBlockInsertPosition', () => {
        it('should calculate insert position in empty block container', () => {
            const containerRect = {
                left: 0,
                top: 0,
                right: 400,
                bottom: 300,
                width: 400,
                height: 300,
            };

            const result = calculateBlockInsertPosition(containerRect, [], { x: 200, y: 150 });

            expect(result.index).toBe(0);
            expect(result.insertLine).toBeDefined();
            expect(result.insertLine.y).toBe(0);
        });

        it('should calculate insert position between children', () => {
            const containerRect = {
                left: 0,
                top: 0,
                right: 400,
                bottom: 400,
                width: 400,
                height: 400,
            };

            const childrenRects = [
                { left: 0, top: 0, right: 400, bottom: 100, width: 400, height: 100 },
                { left: 0, top: 100, right: 400, bottom: 200, width: 400, height: 100 },
            ];

            const result = calculateBlockInsertPosition(containerRect, childrenRects, { x: 200, y: 120 });

            expect(result.index).toBe(1);
            expect(result.insertLine.y).toBeGreaterThan(0);
        });

        it('should insert at the end when mouse is below all children', () => {
            const containerRect = {
                left: 0,
                top: 0,
                right: 400,
                bottom: 400,
                width: 400,
                height: 400,
            };

            const childrenRects = [
                { left: 0, top: 0, right: 400, bottom: 100, width: 400, height: 100 },
                { left: 0, top: 100, right: 400, bottom: 200, width: 400, height: 100 },
            ];

            const result = calculateBlockInsertPosition(containerRect, childrenRects, { x: 200, y: 300 });

            expect(result.index).toBe(2);
        });
    });

    describe('calculateGridInsertPosition', () => {
        it('should calculate insert position in empty grid', () => {
            const containerRect = {
                left: 0,
                top: 0,
                right: 400,
                bottom: 300,
                width: 400,
                height: 300,
            };

            const result = calculateGridInsertPosition(containerRect, [], { x: 200, y: 150 });

            expect(result.index).toBe(0);
            expect(result.gridPosition).toEqual({ row: 1, column: 1 });
        });

        it('should calculate insert position near existing grid item', () => {
            const containerRect = {
                left: 0,
                top: 0,
                right: 400,
                bottom: 300,
                width: 400,
                height: 300,
            };

            const childrenRects = [
                { left: 0, top: 0, right: 133, bottom: 100, width: 133, height: 100 },
                { left: 133, top: 0, right: 266, bottom: 100, width: 133, height: 100 },
            ];

            const result = calculateGridInsertPosition(containerRect, childrenRects, { x: 150, y: 50 });

            expect(result.index).toBeDefined();
            expect(result.insertLine).toBeDefined();
        });
    });

    describe('calculateInsertPosition (通用)', () => {
        it('should use flex calculator for flex layout', () => {
            const container = {
                rect: {
                    left: 0,
                    top: 0,
                    right: 400,
                    bottom: 300,
                    width: 400,
                    height: 300,
                },
                layoutMode: 'flex',
                props: { flexDirection: 'row' },
            };

            const result = calculateInsertPosition(container, [], { x: 200, y: 150 });

            expect(result.index).toBe(0);
            expect(result.insertLine).toBeDefined();
        });

        it('should use grid calculator for grid layout', () => {
            const container = {
                rect: {
                    left: 0,
                    top: 0,
                    right: 400,
                    bottom: 300,
                    width: 400,
                    height: 300,
                },
                layoutMode: 'grid',
                props: { gridTemplateColumns: 'repeat(3, 1fr)' },
            };

            const result = calculateInsertPosition(container, [], { x: 200, y: 150 });

            expect(result.index).toBe(0);
        });

        it('should use block calculator for block layout', () => {
            const container = {
                rect: {
                    left: 0,
                    top: 0,
                    right: 400,
                    bottom: 300,
                    width: 400,
                    height: 300,
                },
                layoutMode: 'block',
            };

            const result = calculateInsertPosition(container, [], { x: 200, y: 150 });

            expect(result.index).toBe(0);
            expect(result.insertLine).toBeDefined();
        });

        it('should default to flex row for unknown layout', () => {
            const container = {
                rect: {
                    left: 0,
                    top: 0,
                    right: 400,
                    bottom: 300,
                    width: 400,
                    height: 300,
                },
                layoutMode: 'unknown',
            };

            const result = calculateInsertPosition(container, [], { x: 200, y: 150 });

            expect(result.index).toBe(0);
        });
    });
});

