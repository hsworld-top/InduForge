import ElUploadComponent from './ElUpload.vue';

export default {
  type: 'ElUpload',
  name: '上传',
  category: 'Element 组件',
  icon: 'upload',
  thumbnail: null,
  tags: ['upload', '表单'],
  description: 'Element Plus 上传',
  component: ElUploadComponent,

  defaultProps: {
    action: 'https://jsonplaceholder.typicode.com/posts/',
    multiple: true,
    limit: 3,
    drag: true,
    listType: 'text',
    headers: {},
    data: {},
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 320,
    height: 180,
    zIndex: 1,
  },

  propsSchema: {
    action: { type: 'string', label: '上传地址', group: '组件属性', default: 'https://jsonplaceholder.typicode.com/posts/' },
    multiple: { type: 'boolean', label: '多选', group: '组件属性', default: true },
    limit: { type: 'number', label: '数量限制', group: '组件属性', default: 3 },
    drag: { type: 'boolean', label: '拖拽上传', group: '组件属性', default: true },
    listType: {
      type: 'enum',
      label: '列表样式',
      group: '组件属性',
      options: [
        { value: 'text', label: '文本' },
        { value: 'picture', label: '图片' },
        { value: 'picture-card', label: '图片卡片' },
      ],
      default: 'text',
    },
    headers: { type: 'string', label: '请求头', group: '组件属性', multiline: true, format: 'json', default: '{}' },
    data: { type: 'string', label: '额外参数', group: '组件属性', multiline: true, format: 'json', default: '{}' },
  },

  eventsSchema: {
    change: { label: '文件列表变更' },
    success: { label: '上传成功' },
    error: { label: '上传失败' },
    remove: { label: '移除文件' },
    preview: { label: '预览' },
  },

  container: false,
  version: '1.0.0',
};
