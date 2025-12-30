/**
 * 选择和多选功能测试
 * Task 5.9: 测试选中与辅助线系统
 *
 * 测试内容：
 * - 单选组件
 * - 多选组件（Ctrl+点击）
 * - 全选组件
 * - 批量操作（移动、删除、复制）
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useDesignStore } from '../design';

describe('Selection and Multi-Select System', () => {
    let store;

    beforeEach(() => {
        // 创建一个新的 Pinia 实例
        setActivePinia(createPinia());
        store = useDesignStore();

        // 初始化测试页面
        store.currentPage = {
            id: 'test-page',
            name: '测试页面',
            components: [
                {
                    id: 'comp-1',
                    type: 'Text',
                    props: { text: 'Component 1' },
                    style: { left: 0, top: 0, width: 100, height: 50 },
                },
                {
                    id: 'comp-2',
                    type: 'Text',
                    props: { text: 'Component 2' },
                    style: { left: 150, top: 0, width: 100, height: 50 },
                },
                {
                    id: 'comp-3',
                    type: 'Text',
                    props: { text: 'Component 3' },
                    style: { left: 0, top: 100, width: 100, height: 50 },
                },
            ],
        };
    });

    describe('Task 5.1: Component Selection', () => {
        it('should select a single component', () => {
            store.selectComponent('comp-1');

            expect(store.selectedComponentId).toBe('comp-1');
            expect(store.selectedComponentIds).toEqual(['comp-1']);
        });

        it('should deselect when selecting null', () => {
            store.selectComponent('comp-1');
            store.selectComponent(null);

            expect(store.selectedComponentId).toBeNull();
            expect(store.selectedComponentIds).toEqual([]);
        });

        it('should toggle component selection', () => {
            // 第一次点击：选中
            store.toggleComponentSelection('comp-1');
            expect(store.selectedComponentIds).toEqual(['comp-1']);
            expect(store.selectedComponentId).toBe('comp-1');

            // 第二次点击：取消选中
            store.toggleComponentSelection('comp-1');
            expect(store.selectedComponentIds).toEqual([]);
            expect(store.selectedComponentId).toBeNull();
        });

        it('should support multi-select with toggle', () => {
            store.toggleComponentSelection('comp-1');
            store.toggleComponentSelection('comp-2');
            store.toggleComponentSelection('comp-3');

            expect(store.selectedComponentIds).toEqual(['comp-1', 'comp-2', 'comp-3']);
            expect(store.selectedComponentId).toBeNull(); // 多选时为 null
        });

        it('should select all components', () => {
            store.selectAllComponents();

            expect(store.selectedComponentIds).toEqual(['comp-1', 'comp-2', 'comp-3']);
            expect(store.selectedComponentId).toBeNull();
        });

        it('should clear selection', () => {
            store.selectMultipleComponents(['comp-1', 'comp-2']);
            store.clearSelection();

            expect(store.selectedComponentIds).toEqual([]);
            expect(store.selectedComponentId).toBeNull();
        });

        it('should select multiple components', () => {
            store.selectMultipleComponents(['comp-1', 'comp-3']);

            expect(store.selectedComponentIds).toEqual(['comp-1', 'comp-3']);
            expect(store.selectedComponentId).toBeNull();
        });

        it('should set selectedComponentId when only one component is selected', () => {
            store.selectMultipleComponents(['comp-2']);

            expect(store.selectedComponentIds).toEqual(['comp-2']);
            expect(store.selectedComponentId).toBe('comp-2');
        });
    });

    describe('Task 5.8: Batch Operations', () => {
        beforeEach(() => {
            // 选中多个组件
            store.selectMultipleComponents(['comp-1', 'comp-2']);
        });

        it('should batch update component properties', () => {
            store.batchUpdateComponents(['comp-1', 'comp-2'], {
                style: { backgroundColor: 'red' },
            });

            const comp1 = store.currentPage.components.find((c) => c.id === 'comp-1');
            const comp2 = store.currentPage.components.find((c) => c.id === 'comp-2');

            expect(comp1.style.backgroundColor).toBe('red');
            expect(comp2.style.backgroundColor).toBe('red');
        });

        it('should batch move components', () => {
            const delta = { x: 50, y: 30 };
            store.batchMoveComponents(['comp-1', 'comp-2'], delta);

            const comp1 = store.currentPage.components.find((c) => c.id === 'comp-1');
            const comp2 = store.currentPage.components.find((c) => c.id === 'comp-2');

            expect(comp1.style.left).toBe(50); // 0 + 50
            expect(comp1.style.top).toBe(30); // 0 + 30
            expect(comp2.style.left).toBe(200); // 150 + 50
            expect(comp2.style.top).toBe(30); // 0 + 30
        });

        it('should batch delete components', () => {
            store.batchDeleteComponents(['comp-1', 'comp-2']);

            expect(store.currentPage.components).toHaveLength(1);
            expect(store.currentPage.components[0].id).toBe('comp-3');
            expect(store.selectedComponentIds).toEqual([]);
        });

        it('should batch copy components', () => {
            const newIds = store.batchCopyComponents(['comp-1', 'comp-2']);

            expect(newIds).toHaveLength(2);
            expect(store.currentPage.components).toHaveLength(5); // 原3个 + 新2个

            // 检查新组件的位置偏移
            const newComp1 = store.currentPage.components.find((c) => c.id === newIds[0]);
            expect(newComp1.style.left).toBe(20); // 0 + 20
            expect(newComp1.style.top).toBe(20); // 0 + 20

            // 检查选中状态
            expect(store.selectedComponentIds).toEqual(newIds);
        });
    });

    describe('Edge Cases', () => {
        it('should handle empty component list for batch operations', () => {
            expect(() => {
                store.batchUpdateComponents([], { style: { color: 'blue' } });
            }).not.toThrow();

            expect(() => {
                store.batchMoveComponents([], { x: 10, y: 10 });
            }).not.toThrow();

            expect(() => {
                store.batchDeleteComponents([]);
            }).not.toThrow();

            const newIds = store.batchCopyComponents([]);
            expect(newIds).toEqual([]);
        });

        it('should handle selecting non-existent component', () => {
            store.toggleComponentSelection('non-existent');
            expect(store.selectedComponentIds).toEqual(['non-existent']);
            // 注意：实际使用中可能需要验证组件是否存在
        });

        it('should maintain selection order when toggling', () => {
            store.toggleComponentSelection('comp-3');
            store.toggleComponentSelection('comp-1');
            store.toggleComponentSelection('comp-2');

            expect(store.selectedComponentIds).toEqual(['comp-3', 'comp-1', 'comp-2']);
        });
    });

    describe('Integration', () => {
        it('should work with single select after multi-select', () => {
            store.selectMultipleComponents(['comp-1', 'comp-2', 'comp-3']);
            store.selectComponent('comp-2'); // 切换回单选

            expect(store.selectedComponentId).toBe('comp-2');
            expect(store.selectedComponentIds).toEqual(['comp-2']);
        });

        it('should clear multi-select when single selecting', () => {
            store.toggleComponentSelection('comp-1');
            store.toggleComponentSelection('comp-2');

            expect(store.selectedComponentIds).toHaveLength(2);

            store.selectComponent('comp-3'); // 单选

            expect(store.selectedComponentIds).toEqual(['comp-3']);
            expect(store.selectedComponentId).toBe('comp-3');
        });
    });
});

