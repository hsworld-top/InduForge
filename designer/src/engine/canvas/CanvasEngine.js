/**
 * CanvasEngine - Canvas 引擎
 * 
 * 整合 KonvaRenderer、SelectionManager、GuideLineManager
 * 提供统一的 Canvas 操作接口
 */
import { KonvaRenderer } from './KonvaRenderer'
import { SelectionManager } from './SelectionManager'
import { GuideLineManager } from './GuideLineManager'

export class CanvasEngine {
  constructor(container, config = {}) {
    // 创建渲染器
    this.renderer = new KonvaRenderer(container, config)
    
    // 创建选择管理器
    this.selectionManager = new SelectionManager(this.renderer)
    
    // 创建辅助线管理器
    this.guideLineManager = new GuideLineManager(this.renderer)
    
    // 绑定拖拽事件以显示辅助线
    this.bindDragEvents()
    
    // 绑定变换事件以更新组件位置
    this.bindTransformEvents()
  }
  
  /**
   * 绑定拖拽事件
   */
  bindDragEvents() {
    this.renderer.on('component:dragmove', ({ id, position }) => {
      const node = this.renderer.getNode(id)
      if (!node) return
      
      // 获取其他节点
      const otherNodes = Array.from(this.renderer.componentNodes.values())
        .filter(n => n !== node)
      
      // 检查对齐并获取吸附位置
      const snapPos = this.guideLineManager.checkAlignment(node, otherNodes)
      
      // 应用吸附
      if (snapPos.x !== null) {
        node.x(snapPos.x)
      }
      if (snapPos.y !== null) {
        node.y(snapPos.y)
      }
    })
    
    this.renderer.on('component:dragend', ({ id, position }) => {
      // 隐藏辅助线
      this.guideLineManager.hideLines()
      
      // 触发位置更新事件
      const node = this.renderer.getNode(id)
      if (node) {
        this.emitComponentUpdate(id, {
          style: {
            left: Math.round(node.x()),
            top: Math.round(node.y())
          }
        })
      }
    })
  }
  
  /**
   * 绑定变换事件
   */
  bindTransformEvents() {
    this.renderer.on('transform:end', ({ updates }) => {
      updates.forEach(update => {
        const { id, position, size, rotation, scale } = update
        
        this.emitComponentUpdate(id, {
          style: {
            left: Math.round(position.x),
            top: Math.round(position.y),
            width: Math.round(size.width * scale.x),
            height: Math.round(size.height * scale.y),
            rotation: Math.round(rotation)
          }
        })
      })
    })
  }
  
  /**
   * 渲染组件
   * @param {Object} componentSchema - 组件 Schema
   */
  renderComponent(componentSchema) {
    return this.renderer.renderComponent(componentSchema)
  }
  
  /**
   * 批量渲染组件
   * @param {Object[]} components - 组件 Schema 数组
   */
  renderComponents(components) {
    components.forEach(component => {
      this.renderComponent(component)
    })
  }
  
  /**
   * 更新组件
   * @param {string} id - 组件ID
   * @param {Object} updates - 更新内容
   */
  updateComponent(id, updates) {
    this.renderer.updateComponent(id, updates)
  }
  
  /**
   * 移除组件
   * @param {string} id - 组件ID
   */
  removeComponent(id) {
    this.renderer.removeComponent(id)
  }
  
  /**
   * 清空画布
   */
  clear() {
    this.renderer.clear()
    this.selectionManager.clearSelection()
  }
  
  /**
   * 选中组件
   * @param {string|string[]} ids - 组件ID或ID数组
   * @param {boolean} addToSelection - 是否添加到现有选择
   */
  selectComponents(ids, addToSelection = false) {
    this.selectionManager.selectById(ids, addToSelection)
  }
  
  /**
   * 取消选择
   */
  clearSelection() {
    this.selectionManager.clearSelection()
  }
  
  /**
   * 获取选中的组件ID
   * @returns {string[]}
   */
  getSelectedIds() {
    return this.selectionManager.getSelectedIds()
  }
  
  /**
   * 删除选中的组件
   * @returns {string[]} 被删除的组件ID
   */
  deleteSelected() {
    return this.selectionManager.deleteSelected()
  }
  
  /**
   * 对齐选中的组件
   * @param {string} type - 对齐类型
   */
  alignSelected(type) {
    this.selectionManager.alignSelected(type)
  }
  
  /**
   * 分布选中的组件
   * @param {string} type - 分布类型
   */
  distributeSelected(type) {
    this.selectionManager.distributeSelected(type)
  }
  
  /**
   * 设置缩放
   * @param {number} scale - 缩放比例
   */
  setScale(scale) {
    this.renderer.setScale(scale)
  }
  
  /**
   * 获取缩放
   * @returns {number}
   */
  getScale() {
    return this.renderer.getScale()
  }
  
  /**
   * 调整画布大小
   * @param {number} width - 宽度
   * @param {number} height - 高度
   */
  resize(width, height) {
    this.renderer.resize(width, height)
  }
  
  /**
   * 启用/禁用吸附
   * @param {boolean} enabled
   */
  setSnapEnabled(enabled) {
    this.guideLineManager.setSnapEnabled(enabled)
  }
  
  /**
   * 设置吸附阈值
   * @param {number} threshold - 阈值（像素）
   */
  setSnapThreshold(threshold) {
    this.guideLineManager.setSnapThreshold(threshold)
  }
  
  /**
   * 注册事件处理器
   * @param {string} event - 事件名
   * @param {Function} handler - 处理函数
   */
  on(event, handler) {
    this.renderer.on(event, handler)
  }
  
  /**
   * 移除事件处理器
   * @param {string} event - 事件名
   * @param {Function} handler - 处理函数
   */
  off(event, handler) {
    this.renderer.off(event, handler)
  }
  
  /**
   * 触发组件更新事件
   * @param {string} id - 组件ID
   * @param {Object} updates - 更新内容
   */
  emitComponentUpdate(id, updates) {
    this.renderer.emitEvent('component:update', { id, updates })
  }
  
  /**
   * 导出画布为图片
   * @param {Object} options - 导出选项
   * @returns {string} Data URL
   */
  toDataURL(options = {}) {
    return this.renderer.stage.toDataURL(options)
  }
  
  /**
   * 销毁引擎
   */
  destroy() {
    this.guideLineManager.destroy()
    this.selectionManager.destroy()
    this.renderer.destroy()
  }
}

export default CanvasEngine
