/**
 * SelectionManager - 选择管理器
 * 
 * 管理组件的选择、多选、变换等操作
 */
import Konva from 'konva'

export class SelectionManager {
  constructor(renderer) {
    this.renderer = renderer
    this.stage = renderer.stage
    this.selectionLayer = renderer.selectionLayer
    
    // 当前选中的节点
    this.selectedNodes = []
    
    // Transformer（用于变换操作）
    this.transformer = new Konva.Transformer({
      rotateEnabled: true,
      borderStroke: '#409EFF',
      borderStrokeWidth: 2,
      anchorStroke: '#409EFF',
      anchorFill: '#ffffff',
      anchorSize: 8,
      anchorCornerRadius: 4,
      keepRatio: false,
      enabledAnchors: [
        'top-left',
        'top-center',
        'top-right',
        'middle-right',
        'middle-left',
        'bottom-left',
        'bottom-center',
        'bottom-right'
      ]
    })
    
    this.selectionLayer.add(this.transformer)
    
    // 选择框
    this.selectionRect = null
    this.isSelecting = false
    this.selectionStart = null
    
    // 绑定事件
    this.bindEvents()
  }
  
  /**
   * 绑定事件
   */
  bindEvents() {
    // 点击空白处取消选择
    this.stage.on('click', (e) => {
      // 如果点击的是 Stage 本身（空白处）
      if (e.target === this.stage) {
        this.clearSelection()
      }
    })
    
    // 框选开始
    this.stage.on('mousedown touchstart', (e) => {
      // 只在点击空白处时启动框选
      if (e.target !== this.stage) return
      
      // 按住 Shift 键可以多选
      const isMultiSelect = e.evt.shiftKey
      if (!isMultiSelect) {
        this.clearSelection()
      }
      
      this.isSelecting = true
      this.selectionStart = this.stage.getPointerPosition()
      
      // 创建选择框
      this.selectionRect = new Konva.Rect({
        x: this.selectionStart.x,
        y: this.selectionStart.y,
        width: 0,
        height: 0,
        fill: 'rgba(64, 158, 255, 0.1)',
        stroke: '#409EFF',
        strokeWidth: 1,
        dash: [4, 4]
      })
      
      this.selectionLayer.add(this.selectionRect)
      this.selectionLayer.batchDraw()
    })
    
    // 框选移动
    this.stage.on('mousemove touchmove', (e) => {
      if (!this.isSelecting) return
      
      const pos = this.stage.getPointerPosition()
      const x = Math.min(this.selectionStart.x, pos.x)
      const y = Math.min(this.selectionStart.y, pos.y)
      const width = Math.abs(pos.x - this.selectionStart.x)
      const height = Math.abs(pos.y - this.selectionStart.y)
      
      this.selectionRect.setAttrs({ x, y, width, height })
      this.selectionLayer.batchDraw()
    })
    
    // 框选结束
    this.stage.on('mouseup touchend', (e) => {
      if (!this.isSelecting) return
      
      this.isSelecting = false
      
      // 获取选择框内的节点
      const box = this.selectionRect.getClientRect()
      const selected = []
      
      this.renderer.componentNodes.forEach((node, id) => {
        const nodeBox = node.getClientRect()
        
        // 检查是否相交
        if (this.boxesIntersect(box, nodeBox)) {
          selected.push(node)
        }
      })
      
      // 选中节点
      if (selected.length > 0) {
        this.selectNodes(selected, e.evt.shiftKey)
      }
      
      // 移除选择框
      this.selectionRect.destroy()
      this.selectionRect = null
      this.selectionLayer.batchDraw()
    })
  }
  
  /**
   * 检查两个矩形是否相交
   */
  boxesIntersect(box1, box2) {
    return !(
      box1.x + box1.width < box2.x ||
      box2.x + box2.width < box1.x ||
      box1.y + box1.height < box2.y ||
      box2.y + box2.height < box1.y
    )
  }
  
  /**
   * 选中节点
   * @param {Konva.Node|Konva.Node[]} nodes - 节点或节点数组
   * @param {boolean} addToSelection - 是否添加到现有选择
   */
  selectNodes(nodes, addToSelection = false) {
    const nodeArray = Array.isArray(nodes) ? nodes : [nodes]
    
    if (!addToSelection) {
      this.selectedNodes = []
    }
    
    nodeArray.forEach(node => {
      if (!this.selectedNodes.includes(node)) {
        this.selectedNodes.push(node)
      }
    })
    
    // 更新 Transformer
    this.transformer.nodes(this.selectedNodes)
    this.selectionLayer.batchDraw()
    
    // 触发选择事件
    this.emitSelectionChange()
  }
  
  /**
   * 通过 ID 选中节点
   * @param {string|string[]} ids - 组件ID或ID数组
   * @param {boolean} addToSelection - 是否添加到现有选择
   */
  selectById(ids, addToSelection = false) {
    const idArray = Array.isArray(ids) ? ids : [ids]
    const nodes = idArray
      .map(id => this.renderer.getNode(id))
      .filter(node => node !== undefined)
    
    if (nodes.length > 0) {
      this.selectNodes(nodes, addToSelection)
    }
  }
  
  /**
   * 取消选择
   */
  clearSelection() {
    this.selectedNodes = []
    this.transformer.nodes([])
    this.selectionLayer.batchDraw()
    this.emitSelectionChange()
  }
  
  /**
   * 获取选中的节点ID
   * @returns {string[]}
   */
  getSelectedIds() {
    return this.selectedNodes.map(node => node.id())
  }
  
  /**
   * 获取选中的节点
   * @returns {Konva.Node[]}
   */
  getSelectedNodes() {
    return this.selectedNodes
  }
  
  /**
   * 是否有选中的节点
   * @returns {boolean}
   */
  hasSelection() {
    return this.selectedNodes.length > 0
  }
  
  /**
   * 删除选中的节点
   */
  deleteSelected() {
    const ids = this.getSelectedIds()
    ids.forEach(id => {
      this.renderer.removeComponent(id)
    })
    this.clearSelection()
    return ids
  }
  
  /**
   * 复制选中的节点
   * @returns {Object[]} 组件 Schema 数组
   */
  copySelected() {
    // 这里需要从 Store 获取组件 Schema
    // 暂时返回 ID 列表
    return this.getSelectedIds()
  }
  
  /**
   * 对齐选中的节点
   * @param {string} type - 对齐类型: left, right, top, bottom, center-h, center-v
   */
  alignSelected(type) {
    if (this.selectedNodes.length < 2) return
    
    const boxes = this.selectedNodes.map(node => node.getClientRect())
    
    switch (type) {
      case 'left': {
        const minX = Math.min(...boxes.map(box => box.x))
        this.selectedNodes.forEach(node => {
          node.x(minX)
        })
        break
      }
      case 'right': {
        const maxX = Math.max(...boxes.map(box => box.x + box.width))
        this.selectedNodes.forEach((node, i) => {
          node.x(maxX - boxes[i].width)
        })
        break
      }
      case 'top': {
        const minY = Math.min(...boxes.map(box => box.y))
        this.selectedNodes.forEach(node => {
          node.y(minY)
        })
        break
      }
      case 'bottom': {
        const maxY = Math.max(...boxes.map(box => box.y + box.height))
        this.selectedNodes.forEach((node, i) => {
          node.y(maxY - boxes[i].height)
        })
        break
      }
      case 'center-h': {
        const avgX = boxes.reduce((sum, box) => sum + box.x + box.width / 2, 0) / boxes.length
        this.selectedNodes.forEach((node, i) => {
          node.x(avgX - boxes[i].width / 2)
        })
        break
      }
      case 'center-v': {
        const avgY = boxes.reduce((sum, box) => sum + box.y + box.height / 2, 0) / boxes.length
        this.selectedNodes.forEach((node, i) => {
          node.y(avgY - boxes[i].height / 2)
        })
        break
      }
    }
    
    this.renderer.mainLayer.batchDraw()
    this.emitTransformEnd()
  }
  
  /**
   * 分布选中的节点
   * @param {string} type - 分布类型: horizontal, vertical
   */
  distributeSelected(type) {
    if (this.selectedNodes.length < 3) return
    
    const boxes = this.selectedNodes.map((node, i) => ({
      node,
      box: node.getClientRect(),
      index: i
    }))
    
    if (type === 'horizontal') {
      // 按 X 坐标排序
      boxes.sort((a, b) => a.box.x - b.box.x)
      
      const first = boxes[0].box
      const last = boxes[boxes.length - 1].box
      const totalWidth = boxes.reduce((sum, item) => sum + item.box.width, 0)
      const gap = (last.x + last.width - first.x - totalWidth) / (boxes.length - 1)
      
      let currentX = first.x
      boxes.forEach(item => {
        item.node.x(currentX)
        currentX += item.box.width + gap
      })
    } else if (type === 'vertical') {
      // 按 Y 坐标排序
      boxes.sort((a, b) => a.box.y - b.box.y)
      
      const first = boxes[0].box
      const last = boxes[boxes.length - 1].box
      const totalHeight = boxes.reduce((sum, item) => sum + item.box.height, 0)
      const gap = (last.y + last.height - first.y - totalHeight) / (boxes.length - 1)
      
      let currentY = first.y
      boxes.forEach(item => {
        item.node.y(currentY)
        currentY += item.box.height + gap
      })
    }
    
    this.renderer.mainLayer.batchDraw()
    this.emitTransformEnd()
  }
  
  /**
   * 触发选择变化事件
   */
  emitSelectionChange() {
    this.renderer.emitEvent('selection:change', {
      ids: this.getSelectedIds(),
      nodes: this.selectedNodes
    })
  }
  
  /**
   * 触发变换结束事件
   */
  emitTransformEnd() {
    const updates = this.selectedNodes.map(node => ({
      id: node.id(),
      position: node.position(),
      size: { width: node.width(), height: node.height() },
      rotation: node.rotation(),
      scale: node.scale()
    }))
    
    this.renderer.emitEvent('transform:end', { updates })
  }
  
  /**
   * 启用/禁用变换
   * @param {boolean} enabled
   */
  setTransformEnabled(enabled) {
    this.transformer.visible(enabled)
    this.transformer.listening(enabled)
    this.selectionLayer.batchDraw()
  }
  
  /**
   * 销毁选择管理器
   */
  destroy() {
    this.transformer.destroy()
    if (this.selectionRect) {
      this.selectionRect.destroy()
    }
    this.selectedNodes = []
  }
}

export default SelectionManager
