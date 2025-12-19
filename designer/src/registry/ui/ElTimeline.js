import ElTimelineComponent from './ElTimeline.vue';

export default {
  type: 'ElTimeline',
  name: '时间线',
  category: 'Element 组件',
  icon: 'time',
  thumbnail: null,
  tags: ['timeline'],
  description: 'Element Plus 时间线',
  component: ElTimelineComponent,

  defaultProps: {
    items: [
      { timestamp: '2025-01-01', type: 'primary', content: '节点 1' },
      { timestamp: '2025-01-02', type: 'success', content: '节点 2' },
      { timestamp: '2025-01-03', type: 'warning', content: '节点 3' },
    ],
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 240,
    height: 180,
    zIndex: 1,
  },

  propsSchema: {
    items: {
      type: 'string',
      label: '时间线数据 (JSON)',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(
        [
          { timestamp: '2025-01-01', type: 'primary', content: '节点 1' },
          { timestamp: '2025-01-02', type: 'success', content: '节点 2' },
          { timestamp: '2025-01-03', type: 'warning', content: '节点 3' },
        ],
        null,
        2,
      ),
    },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
