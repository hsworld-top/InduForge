/**
 * Circle - 圆形组件
 */

export default {
  type: 'Circle',
  name: '圆形',
  category: '基础图形',
  icon: 'radio-button-off',
  tags: ['图形', '圆形', '圆', 'circle'],
  description: '圆形图形，支持填充色、边框等属性',
  
  defaultProps: {
    fill: '#67C23A',
    stroke: '#303133',
    strokeWidth: 1,
    opacity: 1
  },
  
  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 100,
    height: 100,
    zIndex: 1
  },
  
  propsSchema: {
    fill: {
      type: 'color',
      label: '填充颜色',
      group: '外观',
      default: '#67C23A'
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
    click: { label: '点击' },
    dblclick: { label: '双击' },
    mouseenter: { label: '鼠标进入' },
    mouseleave: { label: '鼠标离开' }
  },
  
  container: false,
  version: '1.0.0'
}
