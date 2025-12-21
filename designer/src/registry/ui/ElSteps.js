import ElStepsComponent from './ElSteps.vue';

export default {
  type: 'ElSteps',
  name: '步骤条',
  category: 'Element 组件',
  icon: 'flag',
  thumbnail: null,
  tags: ['steps', 'progress'],
  description: 'Element Plus 步骤条',
  component: ElStepsComponent,

  defaultProps: {
    active: 1,
    direction: 'horizontal',
    alignCenter: true,
    finishStatus: 'success',
    processStatus: 'process',
    steps: [
      { title: '步骤一', description: '描述 1' },
      { title: '步骤二', description: '描述 2' },
      { title: '步骤三', description: '描述 3' },
    ],
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 300,
    height: 120,
    zIndex: 1,
  },

  propsSchema: {
    active: { type: 'number', label: '当前步骤', group: '组件属性', default: 1 },
    direction: {
      type: 'enum',
      label: '方向',
      group: '组件属性',
      options: [
        { value: 'horizontal', label: '横向' },
        { value: 'vertical', label: '纵向' },
      ],
      default: 'horizontal',
    },
    alignCenter: { type: 'boolean', label: '居中', group: '组件属性', default: true },
    finishStatus: { type: 'string', label: '完成状态', group: '组件属性', default: 'success' },
    processStatus: { type: 'string', label: '进行状态', group: '组件属性', default: 'process' },
    steps: {
      type: 'string',
      label: '步骤列表',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: JSON.stringify(
        [
          { title: '步骤一', description: '描述 1' },
          { title: '步骤二', description: '描述 2' },
          { title: '步骤三', description: '描述 3' },
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
