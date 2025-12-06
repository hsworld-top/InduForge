/**
 * Image Component Definition
 * 
 * A component for displaying images from assets or URLs.
 * Requirements: 8.6
 */

export default {
  type: 'Image',
  name: '图片',
  category: 'basic',
  icon: 'picture',
  
  defaultProps: {
    src: '',
    alt: '',
    fit: 'contain',
    lazy: false,
    previewDisabled: true,
    fallback: '',
    placeholder: ''
  },
  
  defaultStyle: {
    position: 'absolute',
    left: 0,
    top: 0,
    width: 150,
    height: 100,
    zIndex: 1,
    borderRadius: 0,
    objectFit: 'contain'
  },
  
  propsSchema: {
    src: {
      type: 'string',
      label: '图片地址',
      default: '',
      format: 'image'
    },
    alt: {
      type: 'string',
      label: '替代文本',
      default: '',
      description: '图片无法显示时的替代文本'
    },
    fit: {
      type: 'enum',
      label: '填充模式',
      options: [
        { value: 'fill', label: '填充' },
        { value: 'contain', label: '包含' },
        { value: 'cover', label: '覆盖' },
        { value: 'none', label: '无' },
        { value: 'scale-down', label: '缩小' }
      ],
      default: 'contain'
    },
    lazy: {
      type: 'boolean',
      label: '懒加载',
      default: false,
      description: '是否开启懒加载'
    },
    previewDisabled: {
      type: 'boolean',
      label: '禁用预览',
      default: true,
      description: '是否禁用点击预览'
    },
    fallback: {
      type: 'string',
      label: '加载失败图片',
      default: '',
      format: 'image',
      description: '加载失败时显示的图片'
    },
    placeholder: {
      type: 'string',
      label: '占位图片',
      default: '',
      format: 'image',
      description: '加载中显示的占位图片'
    }
  },
  
  // Events that this component can emit
  events: {
    load: {
      label: '加载完成',
      description: '图片加载完成时触发'
    },
    error: {
      label: '加载失败',
      description: '图片加载失败时触发'
    },
    click: {
      label: '点击',
      description: '图片被点击时触发'
    }
  }
}
