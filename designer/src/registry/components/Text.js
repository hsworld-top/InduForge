/**
 * Text Component Definition
 * 
 * A component for displaying static or bound text content.
 * Requirements: 8.4
 */

export default {
  type: 'Text',
  name: '文本',
  category: 'Basic',
  icon: 'document',
  
  defaultProps: {
    content: '文本内容',
    fontSize: 14,
    fontWeight: 'normal',
    fontFamily: 'inherit',
    color: '#333333',
    textAlign: 'left',
    lineHeight: 1.5,
    letterSpacing: 0,
    textDecoration: 'none',
    whiteSpace: 'normal',
    overflow: 'visible',
    textOverflow: 'clip'
  },
  
  defaultStyle: {
    position: 'absolute',
    left: 0,
    top: 0,
    width: 100,
    height: 24,
    zIndex: 1
  },
  
  propsSchema: {
    content: {
      type: 'string',
      label: '文本内容',
      default: '文本内容',
      multiline: true
    },
    fontSize: {
      type: 'number',
      label: '字体大小',
      default: 14,
      min: 8,
      max: 200,
      unit: 'px'
    },
    fontWeight: {
      type: 'enum',
      label: '字体粗细',
      options: [
        { value: 'normal', label: '正常' },
        { value: 'bold', label: '粗体' },
        { value: '100', label: '100' },
        { value: '200', label: '200' },
        { value: '300', label: '300' },
        { value: '400', label: '400' },
        { value: '500', label: '500' },
        { value: '600', label: '600' },
        { value: '700', label: '700' },
        { value: '800', label: '800' },
        { value: '900', label: '900' }
      ],
      default: 'normal'
    },
    fontFamily: {
      type: 'string',
      label: '字体',
      default: 'inherit'
    },
    color: {
      type: 'string',
      label: '文字颜色',
      default: '#333333',
      format: 'color'
    },
    textAlign: {
      type: 'enum',
      label: '对齐方式',
      options: [
        { value: 'left', label: '左对齐' },
        { value: 'center', label: '居中' },
        { value: 'right', label: '右对齐' },
        { value: 'justify', label: '两端对齐' }
      ],
      default: 'left'
    },
    lineHeight: {
      type: 'number',
      label: '行高',
      default: 1.5,
      min: 0.5,
      max: 5,
      step: 0.1
    },
    letterSpacing: {
      type: 'number',
      label: '字间距',
      default: 0,
      min: -10,
      max: 50,
      unit: 'px'
    },
    textDecoration: {
      type: 'enum',
      label: '文字装饰',
      options: [
        { value: 'none', label: '无' },
        { value: 'underline', label: '下划线' },
        { value: 'line-through', label: '删除线' },
        { value: 'overline', label: '上划线' }
      ],
      default: 'none'
    },
    whiteSpace: {
      type: 'enum',
      label: '空白处理',
      options: [
        { value: 'normal', label: '正常' },
        { value: 'nowrap', label: '不换行' },
        { value: 'pre', label: '保留空白' },
        { value: 'pre-wrap', label: '保留并换行' }
      ],
      default: 'normal'
    },
    overflow: {
      type: 'enum',
      label: '溢出处理',
      options: [
        { value: 'visible', label: '可见' },
        { value: 'hidden', label: '隐藏' }
      ],
      default: 'visible'
    },
    textOverflow: {
      type: 'enum',
      label: '文本溢出',
      options: [
        { value: 'clip', label: '裁剪' },
        { value: 'ellipsis', label: '省略号' }
      ],
      default: 'clip',
      visible: (props) => props.overflow === 'hidden'
    }
  }
}
