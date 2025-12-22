import ElCarouselComponent from './ElCarousel.vue';

export default {
  type: 'ElCarousel',
  name: '走马灯',
  category: 'Element 组件',
  icon: 'image',
  thumbnail: null,
  tags: ['carousel'],
  description: 'Element Plus 走马灯',
  component: ElCarouselComponent,

  defaultProps: {
    height: '160px',
    type: 'card',
    autoplay: false,
    items: ['轮播 1', '轮播 2', '轮播 3'],
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 320,
    height: 200,
    zIndex: 1,
  },

  propsSchema: {
    height: { type: 'string', label: '高度', group: '组件属性', default: '160px' },
    type: {
      type: 'enum',
      label: '样式',
      group: '组件属性',
      options: [
        { value: '', label: '默认' },
        { value: 'card', label: '卡片' },
      ],
      default: 'card',
    },
    autoplay: { type: 'boolean', label: '自动播放', group: '组件属性', default: false },
    items: {
      type: 'string',
      label: '轮播项',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(['轮播 1', '轮播 2', '轮播 3'], null, 2),
    },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
