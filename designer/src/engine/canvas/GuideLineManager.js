/**
 * GuideLineManager - 对齐辅助线管理器
 * 
 * 在拖拽组件时显示对齐辅助线
 */
import Konva from 'konva'

export class GuideLineManager {
  constructor(renderer) {
    this.renderer = renderer
    this.stage = renderer.stage
    this.selectionLayer = renderer.selectionLayer
    
    // 辅助线
    this.verticalLine = new Konva.Line({
      stroke: '#FF0000',
      strokeWidth: 1,
      dash: [4, 6],
      visible: false
    })
    
    this.horizontalLine = new Konva.Line({
      stroke: '#FF0000',
      strokeWidth: 1,
      dash: [4, 6],
      visible: false
    })
    
    this.selectionLayer.add(this.verticalLine)
    this.selectionLayer.add(this.horizontalLine)
    
    // 吸附阈值（像素）
    this.snapThreshold = 5
    
    // 是否启用吸附
    this.snapEnabled = true
  }
  
  /**
   * 检查并显示对齐辅助线
   * @param {Konva.Node} node - 正在拖拽的节点
   * @param {Konva.Node[]} otherNodes - 其他节点
   */
  checkAlignment(node, otherNodes) {
    if (!this.snapEnabled) {
      this.hideLines()
      return null
    }
    
    const nodeBox = node.getClientRect()
    const nodeCenterX = nodeBox.x + nodeBox.width / 2
    const nodeCenterY = nodeBox.y + nodeBox.height / 2
    
    let verticalSnap = null
    let horizontalSnap = null
    let minVerticalDist = this.snapThreshold
    let minHorizontalDist = this.snapThreshold
    
    // 检查与其他节点的对齐
    otherNodes.forEach(other => {
      if (other === node) return
      
      const otherBox = other.getClientRect()
      const otherCenterX = otherBox.x + otherBox.width / 2
      const otherCenterY = otherBox.y + otherBox.height / 2
      
      // 垂直对齐检查
      // 左边对齐
      const leftDist = Math.abs(nodeBox.x - otherBox.x)
      if (leftDist < minVerticalDist) {
        minVerticalDist = leftDist
        verticalSnap = {
          type: 'left',
          x: otherBox.x,
          snapX: otherBox.x,
          y1: Math.min(nodeBox.y, otherBox.y),
          y2: Math.max(nodeBox.y + nodeBox.height, otherBox.y + otherBox.height)
        }
      }
      
      // 右边对齐
      const rightDist = Math.abs(nodeBox.x + nodeBox.width - (otherBox.x + otherBox.width))
      if (rightDist < minVerticalDist) {
        minVerticalDist = rightDist
        verticalSnap = {
          type: 'right',
          x: otherBox.x + otherBox.width,
          snapX: otherBox.x + otherBox.width - nodeBox.width,
          y1: Math.min(nodeBox.y, otherBox.y),
          y2: Math.max(nodeBox.y + nodeBox.height, otherBox.y + otherBox.height)
        }
      }
      
      // 中心对齐
      const centerXDist = Math.abs(nodeCenterX - otherCenterX)
      if (centerXDist < minVerticalDist) {
        minVerticalDist = centerXDist
        verticalSnap = {
          type: 'center',
          x: otherCenterX,
          snapX: otherCenterX - nodeBox.width / 2,
          y1: Math.min(nodeBox.y, otherBox.y),
          y2: Math.max(nodeBox.y + nodeBox.height, otherBox.y + otherBox.height)
        }
      }
      
      // 水平对齐检查
      // 顶部对齐
      const topDist = Math.abs(nodeBox.y - otherBox.y)
      if (topDist < minHorizontalDist) {
        minHorizontalDist = topDist
        horizontalSnap = {
          type: 'top',
          y: otherBox.y,
          snapY: otherBox.y,
          x1: Math.min(nodeBox.x, otherBox.x),
          x2: Math.max(nodeBox.x + nodeBox.width, otherBox.x + otherBox.width)
        }
      }
      
      // 底部对齐
      const bottomDist = Math.abs(nodeBox.y + nodeBox.height - (otherBox.y + otherBox.height))
      if (bottomDist < minHorizontalDist) {
        minHorizontalDist = bottomDist
        horizontalSnap = {
          type: 'bottom',
          y: otherBox.y + otherBox.height,
          snapY: otherBox.y + otherBox.height - nodeBox.height,
          x1: Math.min(nodeBox.x, otherBox.x),
          x2: Math.max(nodeBox.x + nodeBox.width, otherBox.x + otherBox.width)
        }
      }
      
      // 中心对齐
      const centerYDist = Math.abs(nodeCenterY - otherCenterY)
      if (centerYDist < minHorizontalDist) {
        minHorizontalDist = centerYDist
        horizontalSnap = {
          type: 'center',
          y: otherCenterY,
          snapY: otherCenterY - nodeBox.height / 2,
          x1: Math.min(nodeBox.x, otherBox.x),
          x2: Math.max(nodeBox.x + nodeBox.width, otherBox.x + otherBox.width)
        }
      }
    })
    
    // 显示辅助线
    if (verticalSnap) {
      this.showVerticalLine(verticalSnap.x, verticalSnap.y1, verticalSnap.y2)
    } else {
      this.verticalLine.visible(false)
    }
    
    if (horizontalSnap) {
      this.showHorizontalLine(horizontalSnap.y, horizontalSnap.x1, horizontalSnap.x2)
    } else {
      this.horizontalLine.visible(false)
    }
    
    this.selectionLayer.batchDraw()
    
    // 返回吸附位置
    return {
      x: verticalSnap ? verticalSnap.snapX : null,
      y: horizontalSnap ? horizontalSnap.snapY : null
    }
  }
  
  /**
   * 显示垂直辅助线
   */
  showVerticalLine(x, y1, y2) {
    this.verticalLine.points([x, y1, x, y2])
    this.verticalLine.visible(true)
  }
  
  /**
   * 显示水平辅助线
   */
  showHorizontalLine(y, x1, x2) {
    this.horizontalLine.points([x1, y, x2, y])
    this.horizontalLine.visible(true)
  }
  
  /**
   * 隐藏所有辅助线
   */
  hideLines() {
    this.verticalLine.visible(false)
    this.horizontalLine.visible(false)
    this.selectionLayer.batchDraw()
  }
  
  /**
   * 启用/禁用吸附
   * @param {boolean} enabled
   */
  setSnapEnabled(enabled) {
    this.snapEnabled = enabled
    if (!enabled) {
      this.hideLines()
    }
  }
  
  /**
   * 设置吸附阈值
   * @param {number} threshold - 阈值（像素）
   */
  setSnapThreshold(threshold) {
    this.snapThreshold = threshold
  }
  
  /**
   * 销毁辅助线管理器
   */
  destroy() {
    this.verticalLine.destroy()
    this.horizontalLine.destroy()
  }
}

export default GuideLineManager
