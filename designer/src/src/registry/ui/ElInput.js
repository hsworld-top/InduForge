import ElInputComponent from './ElInput.vue';

export default {
  type: 'ElInput',
  name: '输入框',
  category: 'Element 组件',
  icon: 'edit',
  thumbnail: null,
  tags: ['输入', 'input', '表单'],
  description: 'Element Plus 输入框',
  component: ElInputComponent,

  defaultProps: {
    modelValue: '',
    placeholder: '请输入',
    type: 'text',
    clearable: true,
    disabled: false,
    showPassword: false,
    maxlength: null,
    showWordLimit: false,
    prefixIcon: '',
    suffixIcon: '',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 240,
    height: 40,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'string', label: '输入值', group: '组件属性', default: '' },
    placeholder: { type: 'string', label: '占位', group: '组件属性', default: '请输入' },
    type: {
      type: 'enum',
      label: '类型',
      group: '组件属性',
      options: [
        { value: 'text', label: '文本' },
        { value: 'textarea', label: '文本域' },
        { value: 'password', label: '密码' },
        { value: 'number', label: '数字' },
      ],
      default: 'text',
    },
    clearable: { type: 'boolean', label: '可清空', group: '组件属性', default: true },
    disabled: { type: 'boolean', label: '禁用', group: '组件属性', default: false },
    showPassword: { type: 'boolean', label: '显示密码', group: '组件属性', default: false, visible: (p) => p.type === 'password' },
    maxlength: { type: 'number', label: '最大长度', group: '组件属性', default: null },
    showWordLimit: { type: 'boolean', label: '显示字数', group: '组件属性', default: false },
    prefixIcon: { type: 'string', label: '前缀图标', group: '组件属性', default: '' },
    suffixIcon: { type: 'string', label: '后缀图标', group: '组件属性', default: '' },
  },

  eventsSchema: {
    input: { label: '输入' },
    change: { label: '变更' },
    focus: { label: '获得焦点' },
    blur: { label: '失去焦点' },
    clear: { label: '清空' },
  },

  container: false,
  version: '1.0.0',
};
