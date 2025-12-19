import ElBadgeComponent from './ElBadge.vue';

export default {
  type: 'ElBadge',
  name: '徽章',
  category: 'Element 组件',
  icon: 'notification',
  thumbnail: null,
  tags: ['badge'],
  description: 'Element Plus 徽章',
  component: ElBadgeComponent,

  defaultProps: {
    value: 12,
    max: 99,
    isDot: false,
    type: 'primary',
    hidden: false,
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 120,
    height: 32,
    zIndex: 1,
  },

  propsSchema: {
    value: { type: 'string', label: '数值', group: '组件属性', default: '12' },
    max: { type: 'number', label: '封顶值', group: '组件属性', default: 99 },
    isDot: { type: 'boolean', label: '显示小圆点', group: '组件属性', default: false },
    type: {
      type: 'enum',
      label: '类型',
      group: '组件属性',
      options: [
        { value: 'primary', label: '主要' },
        { value: 'success', label: '成功' },
        { value: 'warning', label: '警告' },
        { value: 'danger', label: '危险' },
        { value: 'info', label: '信息' },
      ],
      default: 'primary',
    },
    hidden: { type: 'boolean', label: '隐藏', group: '组件属性', default: false },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
