/**
 * multi-selection.integration.test.js - 多选功能集成测试
 * Task 5.9*: 测试选中与辅助线系统（Store部分）
 */
import { describe, it, expect, beforeEach } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useDesignStore } from '@/store/design';

describe('Multi-Selection Integration Tests', () => {
  let store;

  beforeEach(() => {
    setActivePinia(createPinia());
    store = useDesignStore();
    store.createPage('Test Page');
    store.selectPage(store.pages[0].id);

    // Add test components
    for (let i = 1; i <= 5; i++) {
      store.addComponent({
        id: `comp-${i}`,
        type: 'Text',
        props: { text: `Component ${i}` },
        style: { left: i * 100, top: i * 100, width: 100, height: 100 },
      });
    }
  });

  describe('Selection Operations', () => {
    it('should select single component', () => {
      store.selectComponent('comp-1');

      expect(store.selectedComponentId).toBe('comp-1');
      expect(store.selectedComponentIds).toEqual(['comp-1']);
      expect(store.selectedComponents).toHaveLength(1);
    });

    it('should select multiple components', () => {
      store.selectMultipleComponents(['comp-1', 'comp-2', 'comp-3']);

      expect(store.selectedComponentIds).toEqual(['comp-1', 'comp-2', 'comp-3']);
      expect(store.selectedComponents).toHaveLength(3);
    });

    it('should toggle component selection', () => {
      store.selectComponent('comp-1');
      store.toggleComponentSelection('comp-2');

      expect(store.selectedComponentIds).toEqual(['comp-1', 'comp-2']);

      store.toggleComponentSelection('comp-1');
      expect(store.selectedComponentIds).toEqual(['comp-2']);
    });

    it('should select all components', () => {
      store.selectAllComponents();

      expect(store.selectedComponentIds).toHaveLength(5);
      expect(store.selectedComponentIds).toContain('comp-1');
      expect(store.selectedComponentIds).toContain('comp-5');
    });

    it('should clear selection', () => {
      store.selectMultipleComponents(['comp-1', 'comp-2']);
      store.clearSelection();

      expect(store.selectedComponentIds).toEqual([]);
      expect(store.selectedComponentId).toBeNull();
    });
  });

  describe('Batch Operations', () => {
    it('should batch update component properties', () => {
      store.selectMultipleComponents(['comp-1', 'comp-2', 'comp-3']);
      
      store.batchUpdateComponents(store.selectedComponentIds, {
        props: { text: 'Updated' },
      });

      const comp1 = store.currentPage.components.find(c => c.id === 'comp-1');
      const comp2 = store.currentPage.components.find(c => c.id === 'comp-2');
      const comp3 = store.currentPage.components.find(c => c.id === 'comp-3');

      expect(comp1.props.text).toBe('Updated');
      expect(comp2.props.text).toBe('Updated');
      expect(comp3.props.text).toBe('Updated');
    });

    it('should batch move components', () => {
      store.selectMultipleComponents(['comp-1', 'comp-2']);

      const comp1Before = store.currentPage.components.find(c => c.id === 'comp-1');
      const comp2Before = store.currentPage.components.find(c => c.id === 'comp-2');

      const originalComp1Left = comp1Before.style.left;
      const originalComp2Left = comp2Before.style.left;

      store.batchMoveComponents(store.selectedComponentIds, { x: 50, y: 30 });

      const comp1After = store.currentPage.components.find(c => c.id === 'comp-1');
      const comp2After = store.currentPage.components.find(c => c.id === 'comp-2');

      expect(comp1After.style.left).toBe(originalComp1Left + 50);
      expect(comp1After.style.top).toBe(comp1Before.style.top + 30);
      expect(comp2After.style.left).toBe(originalComp2Left + 50);
      expect(comp2After.style.top).toBe(comp2Before.style.top + 30);
    });

    it('should batch delete components', () => {
      store.selectMultipleComponents(['comp-1', 'comp-2', 'comp-3']);
      
      store.batchDeleteComponents(store.selectedComponentIds);

      expect(store.currentPage.components).toHaveLength(2);
      expect(store.currentPage.components.find(c => c.id === 'comp-1')).toBeUndefined();
      expect(store.currentPage.components.find(c => c.id === 'comp-4')).toBeDefined();
    });

    it('should batch copy components', () => {
      store.selectMultipleComponents(['comp-1', 'comp-2']);

      const newIds = store.batchCopyComponents(store.selectedComponentIds);

      expect(newIds).toHaveLength(2);
      expect(store.currentPage.components).toHaveLength(7); // 5 + 2 copies

      // New components should have offset positions
      const newComp1 = store.currentPage.components.find(c => c.id === newIds[0]);
      expect(newComp1.style.left).toBe(120); // 100 + 20
      expect(newComp1.style.top).toBe(120);
    });
  });

  describe('Layer Management', () => {
    it('should bring component to front', () => {
      store.bringToFront('comp-1');

      const comp1 = store.currentPage.components.find(c => c.id === 'comp-1');
      const otherComps = store.currentPage.components.filter(c => c.id !== 'comp-1');

      const maxOtherZIndex = Math.max(...otherComps.map(c => c.style.zIndex || 0));
      expect(comp1.style.zIndex).toBeGreaterThan(maxOtherZIndex);
    });

    it('should send component to back', () => {
      store.sendToBack('comp-5');

      const comp5 = store.currentPage.components.find(c => c.id === 'comp-5');
      const otherComps = store.currentPage.components.filter(c => c.id !== 'comp-5');

      const minOtherZIndex = Math.min(...otherComps.map(c => c.style.zIndex || 0));
      expect(comp5.style.zIndex).toBeLessThan(minOtherZIndex);
    });

    it('should move component up one layer', () => {
      const beforeZIndex = store.currentPage.components.find(c => c.id === 'comp-2').style.zIndex || 0;
      
      store.moveUp('comp-2');

      const afterZIndex = store.currentPage.components.find(c => c.id === 'comp-2').style.zIndex || 0;
      expect(afterZIndex).toBeGreaterThan(beforeZIndex);
    });

    it('should move component down one layer', () => {
      const beforeZIndex = store.currentPage.components.find(c => c.id === 'comp-3').style.zIndex || 0;
      
      store.moveDown('comp-3');

      const afterZIndex = store.currentPage.components.find(c => c.id === 'comp-3').style.zIndex || 0;
      expect(afterZIndex).toBeLessThan(beforeZIndex);
    });
  });
});

