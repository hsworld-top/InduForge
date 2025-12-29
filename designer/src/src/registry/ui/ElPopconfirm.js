import ElPopconfirmComponent from './ElPopconfirm.vue';

export default {
  type: 'ElPopconfirm',
  name: '气泡确认',
  category: 'Element 组件',
  icon: 'warning',
  thumbnail: null,
  tags: ['popconfirm', 'feedback'],
  description: 'Element Plus 气泡确认',
  component: ElPopconfirmComponent,

  defaultProps: {
    title: '确认删除？',
    width: 200,
    confirmButtonText: '确认',
    cancelButtonText: '取消',
    icon: 'QuestionFilled',
    iconColor: '#f56c6c',
    hideAfter: 0,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 200,
    height: 56,
    zIndex: 1,
  },

  propsSchema: {
    title: { type: 'string', label: '标题', group: '组件属性', default: '确认删除？', multiline: true },
    width: { type: 'number', label: '宽度', group: '组件属性', default: 200 },
    confirmButtonText: { type: 'string', label: '确认文案', group: '组件属性', default: '确认' },
    cancelButtonText: { type: 'string', label: '取消文案', group: '组件属性', default: '取消' },
    icon: { type: 'string', label: '图标', group: '组件属性', default: 'QuestionFilled' },
    iconColor: { type: 'string', label: '图标颜色', group: '组件属性', default: '#f56c6c' },
    hideAfter: { type: 'number', label: '隐藏延迟(ms)', group: '组件属性', default: 0 },
  },

  eventsSchema: {
    confirm: { label: '确认' },
    cancel: { label: '取消' },
  },

  container: false,
  version: '1.0.0',
};
