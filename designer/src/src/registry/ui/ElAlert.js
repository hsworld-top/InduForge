import ElAlertComponent from './ElAlert.vue';

export default {
  type: 'ElAlert',
  name: '提示',
  category: 'Element 组件',
  icon: 'info',
  thumbnail: null,
  tags: ['alert', 'feedback'],
  description: 'Element Plus 提示',
  component: ElAlertComponent,

  defaultProps: {
    title: '成功提示',
    type: 'success',
    description: '这是一条提示',
    closable: false,
    showIcon: true,
    effect: 'light',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 320,
    height: 64,
    zIndex: 1,
  },

  propsSchema: {
    title: { type: 'string', label: '标题', group: '组件属性', default: '成功提示' },
    type: {
      type: 'enum',
      label: '类型',
      group: '组件属性',
      options: [
        { value: 'success', label: '成功' },
        { value: 'info', label: '信息' },
        { value: 'warning', label: '警告' },
        { value: 'error', label: '错误' },
      ],
      default: 'success',
    },
    description: { type: 'string', label: '描述', group: '组件属性', default: '这是一条提示', multiline: true },
    closable: { type: 'boolean', label: '可关闭', group: '组件属性', default: false },
    showIcon: { type: 'boolean', label: '显示图标', group: '组件属性', default: true },
    effect: {
      type: 'enum',
      label: '主题',
      group: '组件属性',
      options: [
        { value: 'light', label: '浅色' },
        { value: 'dark', label: '深色' },
      ],
      default: 'light',
    },
  },

  eventsSchema: {
    close: { label: '关闭' },
  },

  container: false,
  version: '1.0.0',
};
