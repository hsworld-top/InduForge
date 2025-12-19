import ElCalendarComponent from './ElCalendar.vue';

export default {
  type: 'ElCalendar',
  name: '日历',
  category: 'Element 组件',
  icon: 'calendar',
  thumbnail: null,
  tags: ['calendar'],
  description: 'Element Plus 日历',
  component: ElCalendarComponent,

  defaultProps: {
    modelValue: '2025-01-01',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 360,
    height: 260,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'string', label: '日期', group: '组件属性', default: '2025-01-01' },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
