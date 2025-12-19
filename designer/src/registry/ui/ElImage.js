import ElImageComponent from './ElImage.vue';

export default {
  type: 'ElImage',
  name: '图片',
  category: 'Element 组件',
  icon: 'image',
  thumbnail: null,
  tags: ['image'],
  description: 'Element Plus 图片',
  component: ElImageComponent,

  defaultProps: {
    src: 'https://via.placeholder.com/240x120',
    fit: 'cover',
    lazy: false,
    previewSrcList: [],
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 240,
    height: 140,
    zIndex: 1,
  },

  propsSchema: {
    src: { type: 'string', label: '图片地址', group: '组件属性', default: 'https://via.placeholder.com/240x120' },
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
    lazy: { type: 'boolean', label: '懒加载', group: '组件属性', default: false },
    previewSrcList: {
      type: 'string',
      label: '预览列表 (JSON)',
      group: '组件属性',
      multiline: true,
      format: 'json',
      default: '[]',
    },
  },

  eventsSchema: {},
  container: false,
  version: '1.0.0',
};
