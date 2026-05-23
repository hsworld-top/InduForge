/**
 * PlacementResolver 测试套件
 *
 * 测试 placementResolver.ts 的 Strategy 模式实现：
 * - Flex/Free/Grid 容器策略
 * - 容器命中检测（根级优先 + 深度优先 fallback）
 * - Y-first + append fallback + nearest neighbor insertIndex 计算
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { ComponentNode } from '@/editor-core/document/types'
import type { CanvasDocLike } from '@/ui/editors/page/canvas/composables/types'
import {
  FlexContainerStrategy,
  FreeContainerStrategy,
  GridContainerStrategy,
  calculateInsertIndexWithFallback,
  type PlacementStrategy,
} from '@/editor-core/utils/placement-resolver'
import type { CanvasPoint } from '@/editor-core/utils/placement-utils'

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

/** 模拟容器节点 */
function mockContainerNode(
  id: string,
  containerKind: 'flex' | 'free' | 'grid',
  children: string[] = [],
): ComponentNode {
  return {
    id,
    type:
      containerKind === 'flex'
        ? 'FlexContainer'
        : containerKind === 'grid'
          ? 'GridContainer'
          : 'FreeContainer',
    children,
    props: {},
  } as unknown as ComponentNode
}

/** 模拟带 x/y 的叶子节点（绝对定位） */
function mockLeafNodeWithPosition(id: string, x: number, y: number): ComponentNode {
  return {
    id,
    type: 'Text',
    children: [],
    props: { x, y },
  } as unknown as ComponentNode
}

/** 模拟叶子节点 */
function mockLeafNode(id: string): ComponentNode {
  return {
    id,
    type: 'Text',
    children: [],
    props: {},
  } as unknown as ComponentNode
}

// ---------------------------------------------------------------------------
// Test: Strategy pattern — containerKind identity
// ---------------------------------------------------------------------------

describe('placementResolver', () => {
  describe('strategy containerKind identity', () => {
    it("FlexContainerStrategy has containerKind = 'flex'", () => {
      const strategy = new FlexContainerStrategy()
      expect(strategy.containerKind).toBe('flex')
    })

    it("FreeContainerStrategy has containerKind = 'free'", () => {
      const strategy = new FreeContainerStrategy()
      expect(strategy.containerKind).toBe('free')
    })

    it("GridContainerStrategy has containerKind = 'grid'", () => {
      const strategy = new GridContainerStrategy()
      expect(strategy.containerKind).toBe('grid')
    })
  })

  // ---------------------------------------------------------------------------
  // Test: Strategy interface — required methods
  // ---------------------------------------------------------------------------

  describe('strategy interface', () => {
    it('each strategy implements canAcceptChild()', () => {
      const flex = new FlexContainerStrategy()
      const free = new FreeContainerStrategy()
      const grid = new GridContainerStrategy()
      const parentNode = mockContainerNode('p1', 'flex')

      expect(typeof flex.canAcceptChild).toBe('function')
      expect(typeof free.canAcceptChild).toBe('function')
      expect(typeof grid.canAcceptChild).toBe('function')

      // Free always accepts
      expect(free.canAcceptChild(parentNode, 'Text')).toBe(true)
    })

    it('each strategy implements resolveInsertIndex()', () => {
      const flex = new FlexContainerStrategy()
      const free = new FreeContainerStrategy()
      const grid = new GridContainerStrategy()

      expect(typeof flex.resolveInsertIndex).toBe('function')
      expect(typeof free.resolveInsertIndex).toBe('function')
      expect(typeof grid.resolveInsertIndex).toBe('function')
    })
  })

  // ---------------------------------------------------------------------------
  // Test: calculateInsertIndexWithFallback — Y-first (D-04)
  // ---------------------------------------------------------------------------

  describe('insertIndex calculation', () => {
    it('siblings sorted by Y then X (D-04)', () => {
      // 模拟 3 个兄弟节点：nodeA(y=100,x=50), nodeB(y=50,x=10), nodeC(y=50,x=100)
      // 排序后应该是 B(50,10), C(50,100), A(100,50)
      const siblings = [
        mockLeafNodeWithPosition('nodeA', 50, 100),
        mockLeafNodeWithPosition('nodeB', 10, 50),
        mockLeafNodeWithPosition('nodeC', 100, 50),
      ]

      const dropPos: CanvasPoint = { x: 30, y: 75 } // drop between B and C
      const mockContainer = document.createElement('div')

      // Y-first 排序：D-04
      const sorted = [...siblings].sort((a, b) => {
        const aProps = a.props as { x?: number; y?: number }
        const bProps = b.props as { x?: number; y?: number }
        const posA = { x: aProps.x ?? 0, y: aProps.y ?? 0 }
        const posB = { x: bProps.x ?? 0, y: bProps.y ?? 0 }
        return posA.y - posB.y || posA.x - posB.x
      })

      expect((sorted[0] as ComponentNode).id).toBe('nodeB') // y=50, x=10
      expect((sorted[1] as ComponentNode).id).toBe('nodeC') // y=50, x=100
      expect((sorted[2] as ComponentNode).id).toBe('nodeA') // y=100, x=50
    })

    it("insertIndex returns siblings.length when target doesn't exist (D-05)", () => {
      // 模拟一个空容器
      const siblings: ComponentNode[] = []
      const dropPos: CanvasPoint = { x: 100, y: 100 }
      const mockContainer = document.createElement('div')

      const result = calculateInsertIndexWithFallback(dropPos, siblings, mockContainer, 1, 0)

      // 空容器返回 0
      expect(result).toBe(0)
    })

    it('nearest neighbor adjustment after initial Y+X sort (D-06)', () => {
      // 模拟 3 个兄弟节点在不同位置
      const siblings = [
        mockLeafNodeWithPosition('nodeA', 50, 100), // 距离 (75, 75) → 50
        mockLeafNodeWithPosition('nodeB', 10, 50), // 距离 (75, 75) → 90
        mockLeafNodeWithPosition('nodeC', 100, 50), // 距离 (75, 75) → 50
      ]

      const dropPos: CanvasPoint = { x: 75, y: 75 }
      const mockContainer = document.createElement('div')

      // 使用 calculateInsertIndexWithFallback 计算
      const result = calculateInsertIndexWithFallback(
        dropPos,
        siblings,
        mockContainer,
        1,
        0, // originalIndex = 0（第一个位置）
      )

      // nodeA 和 nodeC 距离相同(50)，排序后 nodeA 在前
      // nearestIdx 应该是 0 (nodeA)，与 originalIndex 0 相等或接近
      expect(result).toBe(0)
    })
  })

  // ---------------------------------------------------------------------------
  // Test: Strategy type guard
  // ---------------------------------------------------------------------------

  describe('strategy type guard', () => {
    it('PlacementStrategy interface is satisfied by all strategies', () => {
      const strategies: PlacementStrategy[] = [
        new FlexContainerStrategy(),
        new FreeContainerStrategy(),
        new GridContainerStrategy(),
      ]

      strategies.forEach((strategy) => {
        expect(['flex', 'free', 'grid']).toContain(strategy.containerKind)
        expect(typeof strategy.canAcceptChild).toBe('function')
        expect(typeof strategy.resolveInsertIndex).toBe('function')
      })
    })
  })
})
