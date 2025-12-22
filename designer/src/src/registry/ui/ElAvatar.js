import ElAvatarComponent from './ElAvatar.vue';

export default {
  type: 'ElAvatar',
  name: '头像',
  category: 'Element 组件',
  icon: 'user',
  thumbnail: null,
  tags: ['avatar'],
  description: 'Element Plus 头像',
  component: ElAvatarComponent,

  defaultProps: {
    size: 48,
    shape: 'circle',
    src: 'https://via.placeholder.com/80',
    fit: 'cover',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 60,
    height: 60,
    zIndex: 1,
  },

  propsSchema: {
    size: { type: 'number', label: '尺寸', group: '组件属性', default: 48 },
    shape: {
      type: 'enum',
      label: '形状',
      group: '组件属性',
      options: [
        { value: 'circle', label: '圆形' },
        { value: 'square', label: '方形' },
      ],
      default: 'circle',
    },
    src: { type: 'string', label: '图片地址', group: '组件属性', default: 'https://via.placeholder.com/80' },
    fit: {
      type: 'enum',
      label: '适应方式',
      group: '组件属性',
      options: [
        { value: 'fill', label: '填充' },
        { value: 'contain', label: '包含' },
        { value: 'cover', label: '覆盖' },
        { value: 'none', label: '不缩放' },
        { value: 'scale-down', label: '缩小适应' },
      ],
      default: 'cover',
    },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
