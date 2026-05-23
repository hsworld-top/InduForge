/**
 * placement-utils 单元测试
 *
 * @module editor-core/utils/placement-utils.test
 */

import { describe, expect, it } from 'vitest'
import {
  eventToCanvasPosition,
  clampPositionInContainer,
  type CanvasPoint,
  type PlacementSize,
} from './placement-utils'

describe('eventToCanvasPosition', () => {
  /**
   * Helper: 创建 mock container element with getBoundingClientRect and scroll
   */
  function createMockContainer(opts: {
    rect?: { left: number; top: number; width: number; height: number }
    scrollLeft?: number
    scrollTop?: number
  }): HTMLElement {
    return {
      getBoundingClientRect: () => ({
        left: opts.rect?.left ?? 0,
        top: opts.rect?.top ?? 0,
        width: opts.rect?.width ?? 800,
        height: opts.rect?.height ?? 600,
      }),
      scrollLeft: opts.scrollLeft ?? 0,
      scrollTop: opts.scrollTop ?? 0,
    } as unknown as HTMLElement
  }

  /**
   * Helper: 创建 mock mouse/drag event
   */
  function createMockEvent(clientX: number, clientY: number): MouseEvent {
    return {
      clientX,
      clientY,
    } as unknown as MouseEvent
  }

  // ── D-12: scroll 补偿 ────────────────────────────────────────────────────

  it('scroll 补偿: 当容器可滚动且有 scroll 偏移时，坐标应正确补偿 scrollLeft/scrollTop', () => {
    // container getBoundingClientRect 返回 rect.left=100, rect.top=50
    // 容器内部 scrollLeft=100, scrollTop=50
    // event.clientX=250, event.clientY=150
    // 正确计算: (250 - 100 + 100) / zoom = 250, (150 - 50 + 50) / zoom = 150
    const container = createMockContainer({
      rect: { left: 100, top: 50, width: 800, height: 600 },
      scrollLeft: 100,
      scrollTop: 50,
    })
    const event = createMockEvent(250, 150)

    const result = eventToCanvasPosition(event, container, 1)

    expect(result.x).toBe(250)
    expect(result.y).toBe(150)
  })

  it('scroll 补偿: scrollX=0, scrollY=0 时结果与之前一致（向后兼容）', () => {
    // 无 scroll 偏移时: (clientX - rect.left) / zoom
    const container = createMockContainer({
      rect: { left: 100, top: 50, width: 800, height: 600 },
      scrollLeft: 0,
      scrollTop: 0,
    })
    const event = createMockEvent(250, 150)

    const result = eventToCanvasPosition(event, container, 1)

    // (250 - 100) / 1 = 150, (150 - 50) / 1 = 100
    expect(result.x).toBe(150)
    expect(result.y).toBe(100)
  })

  // ── zoom 除法只发生一次 ─────────────────────────────────────────────────

  it('zoom 除法只发生一次: zoom=0.5 时坐标值应放大 2 倍', () => {
    const container = createMockContainer({
      rect: { left: 0, top: 0, width: 800, height: 600 },
      scrollLeft: 0,
      scrollTop: 0,
    })
    const event = createMockEvent(100, 75)

    const result = eventToCanvasPosition(event, container, 0.5)

    // 100 / 0.5 = 200, 75 / 0.5 = 150
    expect(result.x).toBe(200)
    expect(result.y).toBe(150)
  })

  it('zoom 除法只发生一次: zoom=2 时坐标值应缩小一半', () => {
    const container = createMockContainer({
      rect: { left: 0, top: 0, width: 800, height: 600 },
      scrollLeft: 0,
      scrollTop: 0,
    })
    const event = createMockEvent(200, 100)

    const result = eventToCanvasPosition(event, container, 2)

    // 200 / 2 = 100, 100 / 2 = 50
    expect(result.x).toBe(100)
    expect(result.y).toBe(50)
  })

  // ── 边界保护 ───────────────────────────────────────────────────────────

  it('containerElement 为 null 时返回 {x:0, y:0}', () => {
    const event = createMockEvent(100, 100)
    const result = eventToCanvasPosition(event, null as unknown as HTMLElement, 1)
    expect(result).toEqual({ x: 0, y: 0 })
  })

  it('event.clientX 非数字时返回 {x:0, y:0}', () => {
    const container = createMockContainer({})
    const badEvent = { clientX: 'bad' } as unknown as MouseEvent
    const result = eventToCanvasPosition(badEvent, container, 1)
    expect(result).toEqual({ x: 0, y: 0 })
  })
})

describe('clampPositionInContainer', () => {
  function createMockContainer(opts: { rect?: { width: number; height: number } }): HTMLElement {
    return {
      getBoundingClientRect: () => ({
        left: 0,
        top: 0,
        width: opts.rect?.width ?? 800,
        height: opts.rect?.height ?? 600,
      }),
    } as unknown as HTMLElement
  }

  // ── zoom 处理一致性 ─────────────────────────────────────────────────────

  it('zoom=0.5 时正确约束到容器边界（边界值应除 zoom）', () => {
    // container 800x600, item 100x50
    // maxX = 800/0.5 - 100 = 1600 - 100 = 1500
    // maxY = 600/0.5 - 50 = 1200 - 50 = 1150
    const container = createMockContainer({ rect: { width: 800, height: 600 } })
    const position: CanvasPoint = { x: 2000, y: 1500 }
    const size: PlacementSize = { width: 100, height: 50 }

    const result = clampPositionInContainer(position, container, size, 0.5)

    expect(result.x).toBe(1500)
    expect(result.y).toBe(1150)
  })

  it('返回值坐标在 [0, maxX] x [0, maxY] 范围内', () => {
    const container = createMockContainer({ rect: { width: 800, height: 600 } })
    const position: CanvasPoint = { x: -100, y: -200 }
    const size: PlacementSize = { width: 100, height: 50 }

    const result = clampPositionInContainer(position, container, size, 1)

    expect(result.x).toBe(0)
    expect(result.y).toBe(0)
  })

  it('zoom=1 时结果与 size.width/size.height 直接相减一致', () => {
    const container = createMockContainer({ rect: { width: 800, height: 600 } })
    const position: CanvasPoint = { x: 500, y: 400 }
    const size: PlacementSize = { width: 100, height: 50 }

    const result = clampPositionInContainer(position, container, size, 1)

    // maxX = 800 - 100 = 700, x=500 < 700 保留
    // maxY = 600 - 50 = 550, y=400 < 550 保留
    expect(result.x).toBe(500)
    expect(result.y).toBe(400)
  })

  it('containerElement 为 null 时原样返回 position', () => {
    const position: CanvasPoint = { x: 100, y: 200 }
    const result = clampPositionInContainer(
      position,
      null as unknown as HTMLElement,
      { width: 50, height: 50 },
      1,
    )
    expect(result).toEqual(position)
  })
})
