import ElResultComponent from './ElResult.vue';

export default {
  type: 'ElResult',
  name: '结果',
  category: 'Element 组件',
  icon: 'info',
  thumbnail: null,
  tags: ['result'],
  description: 'Element Plus 结果',
  component: ElResultComponent,

  defaultProps: {
    icon: 'success',
    title: '提交成功',
    subTitle: '您的请求已完成',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 320,
    height: 220,
    zIndex: 1,
  },

  propsSchema: {
    icon: { type: 'string', label: '图标', group: '组件属性', default: 'success' },
    title: { type: 'string', label: '标题', group: '组件属性', default: '提交成功' },
    subTitle: { type: 'string', label: '副标题', group: '组件属性', default: '您的请求已完成', multiline: true },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
