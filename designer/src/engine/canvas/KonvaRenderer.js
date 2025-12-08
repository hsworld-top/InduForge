/**
 * KonvaRenderer - Konva.js 渲染引擎
 * 
 * 负责将组件 Schema 渲染为 Konva 节点
 */
import Konva from 'konva'

export class KonvaRenderer {
  constructor(container, config = {}) {
    this.container = container
    this.config = {
      width: config.width || 1920,
      height: config.height || 1080,
      ...config
    }
    
    // 创建 Stage
    this.stage = new Konva.Stage({
      container: this.container,
      width: this.config.width,
      height: this.config.height,
    })
    
    // 创建主图层
    this.mainLayer = new Konva.Layer()
    this.stage.add(this.mainLayer)
    
    // 创建选择层（用于显示选择框、辅助线等）
    this.selectionLayer = new Konva.Layer()
    this.stage.add(this.selectionLayer)
    
    // 组件节点映射 { componentId: KonvaNode }
    this.componentNodes = new Map()
    
    // 选择框
    this.selectionBox = null
    
    // 辅助线
    this.guideLines = {
      vertical: [],
      horizontal: []
    }
    
    // 事件处理器
    this.eventHandlers = new Map()
  }
  
  /**
   * 渲染组件
   * @param {Object} componentSchema - 组件 Schema
   * @returns {Konva.Node} Konva 节点
   */
  renderComponent(componentSchema) {
    const { id, type, style, props } = componentSchema
    
    // 如果已存在，先移除
    if (this.componentNodes.has(id)) {
      this.removeComponent(id)
    }
    
    let node = null
    
    // 根据组件类型创建对应的 Konva 节点
    switch (type) {
      case 'Rectangle':
        node = this.createRectangle(componentSchema)
        break
      case 'Circle':
        node = this.createCircle(componentSchema)
        break
      case 'Text':
        node = this.createText(componentSchema)
        break
      case 'Image':
        node = this.createImage(componentSchema)
        break
      case 'Line':
        node = this.createLine(componentSchema)
        break
      case 'Ellipse':
        node = this.createEllipse(componentSchema)
        break
      default:
        console.warn(`Unknown component type: ${type}`)
        return null
    }
    
    if (node) {
      // 设置通用属性
      node.id(id)
      node.draggable(true)
      
      // 应用样式
      this.applyStyle(node, style)
      
      // 绑定事件
      this.bindEvents(node, componentSchema)
      
      // 添加到图层
      this.mainLayer.add(node)
      
      // 保存引用
      this.componentNodes.set(id, node)
      
      // 重绘
      this.mainLayer.batchDraw()
    }
    
    return node
  }
  
  /**
   * 创建矩形
   */
  createRectangle(schema) {
    const { style, props } = schema
    
    return new Konva.Rect({
      x: style.left || 0,
      y: style.top || 0,
      width: style.width || 100,
      height: style.height || 100,
      fill: props.fill || '#409EFF',
      stroke: props.stroke || '#303133',
      strokeWidth: props.strokeWidth || 1,
      cornerRadius: props.cornerRadius || 0,
      opacity: props.opacity !== undefined ? props.opacity : 1,
    })
  }
  
  /**
   * 创建圆形
   */
  createCircle(schema) {
    const { style, props } = schema
    const radius = Math.min(style.width || 100, style.height || 100) / 2
    
    return new Konva.Circle({
      x: (style.left || 0) + radius,
      y: (style.top || 0) + radius,
      radius: radius,
      fill: props.fill || '#67C23A',
      stroke: props.stroke || '#303133',
      strokeWidth: props.strokeWidth || 1,
      opacity: props.opacity !== undefined ? props.opacity : 1,
    })
  }
  
  /**
   * 创建文本
   */
  createText(schema) {
    const { style, props } = schema
    
    return new Konva.Text({
      x: style.left || 0,
      y: style.top || 0,
      width: style.width || 120,
      height: style.height || 30,
      text: props.content || '文本内容',
      fontSize: props.fontSize || 14,
      fontFamily: props.fontFamily || 'Arial',
      fontStyle: props.fontWeight === 'bold' ? 'bold' : 'normal',
      fill: props.color || '#303133',
      align: props.textAlign || 'left',
      verticalAlign: 'middle',
      opacity: props.opacity !== undefined ? props.opacity : 1,
    })
  }
  
  /**
   * 创建图片
   */
  createImage(schema) {
    const { style, props } = schema
    
    const imageObj = new Image()
    const konvaImage = new Konva.Image({
      x: style.left || 0,
      y: style.top || 0,
      width: style.width || 150,
      height: style.height || 150,
      opacity: props.opacity !== undefined ? props.opacity : 1,
    })
    
    imageObj.onload = () => {
      konvaImage.image(imageObj)
      this.mainLayer.batchDraw()
    }
    
    imageObj.src = props.src || 'https://via.placeholder.com/150'
    
    return konvaImage
  }
  
  /**
   * 创建线条
   */
  createLine(schema) {
    const { style, props } = schema
    
    return new Konva.Line({
      points: props.points || [0, 0, 100, 100],
      stroke: props.stroke || '#303133',
      strokeWidth: props.strokeWidth || 2,
      lineCap: 'round',
      lineJoin: 'round',
      opacity: props.opacity !== undefined ? props.opacity : 1,
    })
  }
  
  /**
   * 创建椭圆
   */
  createEllipse(schema) {
    const { style, props } = schema
    
    return new Konva.Ellipse({
      x: (style.left || 0) + (style.width || 100) / 2,
      y: (style.top || 0) + (style.height || 100) / 2,
      radiusX: (style.width || 100) / 2,
      radiusY: (style.height || 100) / 2,
      fill: props.fill || '#E6A23C',
      stroke: props.stroke || '#303133',
      strokeWidth: props.strokeWidth || 1,
      opacity: props.opacity !== undefined ? props.opacity : 1,
    })
  }
  
  /**
   * 应用样式
   */
  applyStyle(node, style) {
    if (style.zIndex !== undefined) {
      node.zIndex(style.zIndex)
    }
    
    if (style.rotation !== undefined) {
      node.rotation(style.rotation)
    }
    
    if (style.opacity !== undefined) {
      node.opacity(style.opacity)
    }
  }
  
  /**
   * 绑定事件
   */
  bindEvents(node, schema) {
    const { id, events = {} } = schema
    
    // 点击事件
    if (events.click) {
      node.on('click', (e) => {
        this.emitEvent('component:click', { id, event: e })
      })
    }
    
    // 双击事件
    if (events.dblclick) {
      node.on('dblclick', (e) => {
        this.emitEvent('component:dblclick', { id, event: e })
      })
    }
    
    // 拖拽事件
    node.on('dragstart', (e) => {
      this.emitEvent('component:dragstart', { id, event: e })
    })
    
    node.on('dragmove', (e) => {
      this.emitEvent('component:dragmove', { id, event: e, position: node.position() })
    })
    
    node.on('dragend', (e) => {
      this.emitEvent('component:dragend', { id, event: e, position: node.position() })
    })
    
    // 变换事件
    node.on('transformstart', (e) => {
      this.emitEvent('component:transformstart', { id, event: e })
    })
    
    node.on('transform', (e) => {
      this.emitEvent('component:transform', { id, event: e })
    })
    
    node.on('transformend', (e) => {
      this.emitEvent('component:transformend', { id, event: e })
    })
  }
  
  /**
   * 更新组件
   * @param {string} id - 组件ID
   * @param {Object} updates - 更新内容
   */
  updateComponent(id, updates) {
    const node = this.componentNodes.get(id)
    if (!node) return
    
    // 更新属性
    if (updates.props) {
      Object.entries(updates.props).forEach(([key, value]) => {
        if (key === 'content' && node instanceof Konva.Text) {
          node.text(value)
        } else if (key === 'src' && node instanceof Konva.Image) {
          const imageObj = new Image()
          imageObj.onload = () => {
            node.image(imageObj)
            this.mainLayer.batchDraw()
          }
          imageObj.src = value
        } else {
          node.setAttr(key, value)
        }
      })
    }
    
    // 更新样式
    if (updates.style) {
      if (updates.style.left !== undefined) {
        node.x(updates.style.left)
      }
      if (updates.style.top !== undefined) {
        node.y(updates.style.top)
      }
      if (updates.style.width !== undefined) {
        node.width(updates.style.width)
      }
      if (updates.style.height !== undefined) {
        node.height(updates.style.height)
      }
      if (updates.style.rotation !== undefined) {
        node.rotation(updates.style.rotation)
      }
      if (updates.style.opacity !== undefined) {
        node.opacity(updates.style.opacity)
      }
    }
    
    this.mainLayer.batchDraw()
  }
  
  /**
   * 移除组件
   * @param {string} id - 组件ID
   */
  removeComponent(id) {
    const node = this.componentNodes.get(id)
    if (node) {
      node.destroy()
      this.componentNodes.delete(id)
      this.mainLayer.batchDraw()
    }
  }
  
  /**
   * 清空画布
   */
  clear() {
    this.mainLayer.destroyChildren()
    this.componentNodes.clear()
    this.mainLayer.batchDraw()
  }
  
  /**
   * 获取组件节点
   * @param {string} id - 组件ID
   * @returns {Konva.Node}
   */
  getNode(id) {
    return this.componentNodes.get(id)
  }
  
  /**
   * 设置缩放
   * @param {number} scale - 缩放比例
   */
  setScale(scale) {
    this.stage.scale({ x: scale, y: scale })
    this.stage.batchDraw()
  }
  
  /**
   * 获取缩放
   * @returns {number}
   */
  getScale() {
    return this.stage.scaleX()
  }
  
  /**
   * 调整画布大小
   * @param {number} width - 宽度
   * @param {number} height - 高度
   */
  resize(width, height) {
    this.stage.width(width)
    this.stage.height(height)
    this.stage.batchDraw()
  }
  
  /**
   * 注册事件处理器
   * @param {string} event - 事件名
   * @param {Function} handler - 处理函数
   */
  on(event, handler) {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, [])
    }
    this.eventHandlers.get(event).push(handler)
  }
  
  /**
   * 移除事件处理器
   * @param {string} event - 事件名
   * @param {Function} handler - 处理函数
   */
  off(event, handler) {
    if (!this.eventHandlers.has(event)) return
    const handlers = this.eventHandlers.get(event)
    const index = handlers.indexOf(handler)
    if (index > -1) {
      handlers.splice(index, 1)
    }
  }
  
  /**
   * 触发事件
   * @param {string} event - 事件名
   * @param {any} data - 事件数据
   */
  emitEvent(event, data) {
    if (!this.eventHandlers.has(event)) return
    const handlers = this.eventHandlers.get(event)
    handlers.forEach(handler => handler(data))
  }
  
  /**
   * 销毁渲染器
   */
  destroy() {
    this.stage.destroy()
    this.componentNodes.clear()
    this.eventHandlers.clear()
  }
}

export default KonvaRenderer
