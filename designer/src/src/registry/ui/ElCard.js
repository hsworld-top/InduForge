import ElCardComponent from './ElCard.vue';

export default {
  type: 'ElCard',
  name: '卡片',
  category: 'Element 组件',
  icon: 'board',
  thumbnail: null,
  tags: ['card'],
  description: 'Element Plus 卡片',
  component: ElCardComponent,

  defaultProps: {
    header: '卡片标题',
    content: '卡片内容',
    shadow: 'always',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 260,
    height: 160,
    zIndex: 1,
  },

  propsSchema: {
    header: { type: 'string', label: '标题', group: '组件属性', default: '卡片标题' },
    content: { type: 'string', label: '内容', group: '组件属性', default: '卡片内容', multiline: true },
    shadow: {
      type: 'enum',
      label: '阴影',
      group: '组件属性',
      options: [
        { value: 'always', label: '总是' },
        { value: 'hover', label: '悬停' },
        { value: 'never', label: '无' },
      ],
      default: 'always',
    },
  },

  eventsSchema: {},
  container: true,
  version: '1.0.0',
};
