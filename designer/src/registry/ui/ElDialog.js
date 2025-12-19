import ElDialogComponent from './ElDialog.vue';

export default {
  type: 'ElDialog',
  name: '对话框',
  category: 'Element 组件',
  icon: 'dialog',
  thumbnail: null,
  tags: ['dialog', 'feedback'],
  description: 'Element Plus 对话框',
  component: ElDialogComponent,

  defaultProps: {
    modelValue: true,
    title: '对话框',
    width: '40%',
    modal: false,
    appendToBody: false,
    closeOnClickModal: false,
    showClose: false,
    draggable: true,
    content: '对话框内容',
    cancelText: '取消',
    confirmText: '确定',
  },

  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 360,
    height: 240,
    zIndex: 1,
  },

  propsSchema: {
    modelValue: { type: 'boolean', label: '默认可见', group: '组件属性', default: true },
    title: { type: 'string', label: '标题', group: '组件属性', default: '对话框' },
    width: { type: 'string', label: '宽度', group: '组件属性', default: '40%' },
    modal: { type: 'boolean', label: '遮罩', group: '组件属性', default: false },
    appendToBody: { type: 'boolean', label: '挂载到 body', group: '组件属性', default: false },
    closeOnClickModal: { type: 'boolean', label: '点击遮罩关闭', group: '组件属性', default: false },
    showClose: { type: 'boolean', label: '显示关闭', group: '组件属性', default: false },
    draggable: { type: 'boolean', label: '可拖拽', group: '组件属性', default: true },
    content: { type: 'string', label: '内容', group: '组件属性', default: '对话框内容', multiline: true },
    cancelText: { type: 'string', label: '取消文本', group: '组件属性', default: '取消' },
    confirmText: { type: 'string', label: '确认文本', group: '组件属性', default: '确定' },
  },

  eventsSchema: {
    open: { label: '打开' },
    close: { label: '关闭' },
  },

  container: true,
  version: '1.0.0',
};
