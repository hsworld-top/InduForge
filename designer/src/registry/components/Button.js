/**
 * Button Component Definition
 * 
 * A button component with click event support.
 * Requirements: 8.5
 */

export default {
  type: 'Button',
  name: '按钮',
  category: 'basic',
  icon: 'pointer',
  
  defaultProps: {
    text: '按钮',
    type: 'primary',
    size: 'default',
    disabled: false,
    loading: false,
    plain: false,
    round: false,
    circle: false,
    icon: ''
  },
  
  defaultStyle: {
    position: 'absolute',
    left: 0,
    top: 0,
    width: 80,
    height: 32,
    zIndex: 1
  },
  
  propsSchema: {
    text: {
      type: 'string',
      label: '按钮文字',
      default: '按钮'
    },
    type: {
      type: 'enum',
      label: '类型',
      options: [
        { value: 'primary', label: '主要' },
        { value: 'success', label: '成功' },
        { value: 'warning', label: '警告' },
        { value: 'danger', label: '危险' },
        { value: 'info', label: '信息' },
        { value: 'default', label: '默认' }
      ],
      default: 'primary'
    },
    size: {
      type: 'enum',
      label: '尺寸',
      options: [
        { value: 'large', label: '大' },
        { value: 'default', label: '默认' },
        { value: 'small', label: '小' }
      ],
      default: 'default'
    },
    disabled: {
      type: 'boolean',
      label: '禁用',
      default: false
    },
    loading: {
      type: 'boolean',
      label: '加载中',
      default: false
    },
    plain: {
      type: 'boolean',
      label: '朴素按钮',
      default: false
    },
    round: {
      type: 'boolean',
      label: '圆角按钮',
      default: false
    },
    circle: {
      type: 'boolean',
      label: '圆形按钮',
      default: false
    },
    icon: {
      type: 'string',
      label: '图标',
      default: ''
    }
  },
  
  // Events that this component can emit
  events: {
    click: {
      label: '点击',
      description: '按钮被点击时触发'
    }
  }
}
