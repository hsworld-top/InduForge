/**
 * 网格吸附工具
 */

export interface SnapRect {
  x: number
  y: number
  width: number
  height: number
}

export class GridSnapping {
  gridSize: number
  enabled: boolean

  constructor(gridSize = 10) {
    this.gridSize = gridSize
    this.enabled = true
  }

  snap(value: number): number {
    if (!this.enabled) return value
    return Math.round(value / this.gridSize) * this.gridSize
  }

  snapPosition(x: number, y: number): { x: number; y: number } {
    if (!this.enabled) {
      return { x, y }
    }

    return {
      x: this.snap(x),
      y: this.snap(y),
    }
  }

  snapSize(width: number, height: number): { width: number; height: number } {
    if (!this.enabled) {
      return { width, height }
    }

    return {
      width: Math.max(this.gridSize, this.snap(width)),
      height: Math.max(this.gridSize, this.snap(height)),
    }
  }

  snapRect(rect: SnapRect): SnapRect {
    if (!this.enabled) {
      return rect
    }

    const position = this.snapPosition(rect.x, rect.y)
    const size = this.snapSize(rect.width, rect.height)

    return {
      x: position.x,
      y: position.y,
      width: size.width,
      height: size.height,
    }
  }

  setGridSize(size: number): void {
    if (size > 0) {
      this.gridSize = size
    }
  }

  enable(): void {
    this.enabled = true
  }

  disable(): void {
    this.enabled = false
  }

  toggle(): void {
    this.enabled = !this.enabled
  }
}

export function createGridSnapping(gridSize = 10): GridSnapping {
  return new GridSnapping(gridSize)
}

export default GridSnapping
