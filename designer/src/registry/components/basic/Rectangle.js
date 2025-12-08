/**
 * Rectangle - 矩形组件
 * 
 * 基础图形组件，支持填充、边框、圆角等属性
 */

export default {
  type: 'Rectangle',
  name: '矩形',
  category: '基础图形',
  icon: 'crop-square',
  thumbnail: null,
  tags: ['图形', '矩形', '方形', 'shape'],
  description: '矩形图形，支持填充色、边框、圆角等属性',
  
  defaultProps: {
    fill: '#409EFF',
    stroke: '#303133',
    strokeWidth: 1,
    cornerRadius: 0,
    opacity: 1
  },
  
  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 120,
    height: 80,
    zIndex: 1
  },
  
  propsSchema: {
    fill: {
      type: 'color',
      label: '填充颜色',
      group: '外观',
      default: '#409EFF'
    },
    stroke: {
      type: 'color',
      label: '边框颜色',
      group: '外观',
      default: '#303133'
    },
    strokeWidth: {
      type: 'number',
      label: '边框宽度',
      group: '外观',
      min: 0,
      max: 20,
      step: 1,
      default: 1
    },
    cornerRadius: {
      type: 'number',
      label: '圆角',
      group: '外观',
      min: 0,
      max: 100,
      step: 1,
      default: 0
    },
    opacity: {
      type: 'number',
      label: '不透明度',
      group: '外观',
      min: 0,
      max: 1,
      step: 0.1,
      default: 1
    }
  },
  
  eventsSchema: {
    click: { label: '点击', description: '鼠标点击时触发' },
    dblclick: { label: '双击', description: '鼠标双击时触发' },
    mouseenter: { label: '鼠标进入', description: '鼠标进入组件时触发' },
    mouseleave: { label: '鼠标离开', description: '鼠标离开组件时触发' }
  },
  
  container: false,
  version: '1.0.0'
}
