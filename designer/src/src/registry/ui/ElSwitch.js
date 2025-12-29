import ElSwitchComponent from './ElSwitch.vue';

export default {
  type: 'ElSwitch',
  name: '开关',
  category: 'Element 组件',
  icon: 'switch',
  thumbnail: null,
  tags: ['switch', '表单'],
  description: 'Element Plus 开关',
  component: ElSwitchComponent,

  defaultProps: {
    modelValue: true,
    activeText: '开',
    inactiveText: '关',
    activeColor: '#409EFF',
    inactiveColor: '#dcdfe6',
    inlinePrompt: false,
    disabled: false,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 120,
    height: 40,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'boolean', label: '开关值', group: '组件属性', default: true },
    activeText: { type: 'string', label: '开启文案', group: '组件属性', default: '开' },
    inactiveText: { type: 'string', label: '关闭文案', group: '组件属性', default: '关' },
    activeColor: { type: 'string', label: '开启颜色', group: '组件属性', default: '#409EFF' },
    inactiveColor: { type: 'string', label: '关闭颜色', group: '组件属性', default: '#dcdfe6' },
    inlinePrompt: { type: 'boolean', label: '行内提示', group: '组件属性', default: false },
    disabled: { type: 'boolean', label: '禁用', group: '组件属性', default: false },
  },

  eventsSchema: {
    change: { label: '变更' },
  },

  container: false,
  version: '1.0.0',
};
