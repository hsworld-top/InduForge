/**
 * Input Component Definition
 * 
 * A text input component with validation support.
 * Requirements: 8.7
 */

export default {
  type: 'Input',
  name: '输入框',
  category: 'form',
  icon: 'edit-pen',
  
  defaultProps: {
    value: '',
    placeholder: '请输入',
    type: 'text',
    maxlength: null,
    minlength: null,
    showWordLimit: false,
    clearable: false,
    disabled: false,
    readonly: false,
    size: 'default',
    prefixIcon: '',
    suffixIcon: '',
    rows: 2,
    autosize: false,
    validateEvent: true,
    required: false,
    pattern: '',
    errorMessage: ''
  },
  
  defaultStyle: {
    position: 'absolute',
    left: 0,
    top: 0,
    width: 200,
    height: 32,
    zIndex: 1
  },
  
  propsSchema: {
    value: {
      type: 'string',
      label: '默认值',
      default: ''
    },
    placeholder: {
      type: 'string',
      label: '占位文本',
      default: '请输入'
    },
    type: {
      type: 'enum',
      label: '类型',
      options: [
        { value: 'text', label: '文本' },
        { value: 'password', label: '密码' },
        { value: 'textarea', label: '多行文本' },
        { value: 'number', label: '数字' },
        { value: 'email', label: '邮箱' },
        { value: 'tel', label: '电话' },
        { value: 'url', label: '网址' }
      ],
      default: 'text'
    },
    maxlength: {
      type: 'number',
      label: '最大长度',
      default: null,
      min: 0
    },
    minlength: {
      type: 'number',
      label: '最小长度',
      default: null,
      min: 0
    },
    showWordLimit: {
      type: 'boolean',
      label: '显示字数统计',
      default: false
    },
    clearable: {
      type: 'boolean',
      label: '可清空',
      default: false
    },
    disabled: {
      type: 'boolean',
      label: '禁用',
      default: false
    },
    readonly: {
      type: 'boolean',
      label: '只读',
      default: false
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
    prefixIcon: {
      type: 'string',
      label: '前缀图标',
      default: ''
    },
    suffixIcon: {
      type: 'string',
      label: '后缀图标',
      default: ''
    },
    rows: {
      type: 'number',
      label: '行数',
      default: 2,
      min: 1,
      max: 20,
      visible: (props) => props.type === 'textarea'
    },
    autosize: {
      type: 'boolean',
      label: '自适应高度',
      default: false,
      visible: (props) => props.type === 'textarea'
    },
    required: {
      type: 'boolean',
      label: '必填',
      default: false
    },
    pattern: {
      type: 'string',
      label: '验证正则',
      default: '',
      description: '用于验证输入的正则表达式'
    },
    errorMessage: {
      type: 'string',
      label: '错误提示',
      default: '',
      description: '验证失败时显示的错误信息'
    }
  },
  
  // Events that this component can emit
  events: {
    input: {
      label: '输入',
      description: '输入值变化时触发'
    },
    change: {
      label: '变更',
      description: '值变更且失去焦点时触发'
    },
    focus: {
      label: '获得焦点',
      description: '输入框获得焦点时触发'
    },
    blur: {
      label: '失去焦点',
      description: '输入框失去焦点时触发'
    },
    clear: {
      label: '清空',
      description: '点击清空按钮时触发'
    }
  }
}
