/**
 * Container Component Definition
 * 
 * A layout container that supports flex/grid layout for children.
 * Requirements: 8.3
 */

export default {
  type: 'Container',
  name: '容器',
  category: 'Layout',
  icon: 'folder',
  
  defaultProps: {
    layout: 'flex',
    flexDirection: 'row',
    justifyContent: 'flex-start',
    alignItems: 'stretch',
    flexWrap: 'nowrap',
    gap: 0,
    padding: 0,
    overflow: 'visible'
  },
  
  defaultStyle: {
    position: 'absolute',
    left: 0,
    top: 0,
    width: 200,
    height: 150,
    zIndex: 1,
    backgroundColor: 'transparent',
    borderWidth: 0,
    borderStyle: 'solid',
    borderColor: '#e0e0e0',
    borderRadius: 0
  },
  
  propsSchema: {
    layout: {
      type: 'enum',
      label: '布局模式',
      options: [
        { value: 'flex', label: 'Flex' },
        { value: 'grid', label: 'Grid' },
        { value: 'block', label: 'Block' }
      ],
      default: 'flex'
    },
    flexDirection: {
      type: 'enum',
      label: '排列方向',
      options: [
        { value: 'row', label: '水平' },
        { value: 'row-reverse', label: '水平反向' },
        { value: 'column', label: '垂直' },
        { value: 'column-reverse', label: '垂直反向' }
      ],
      default: 'row',
      visible: (props) => props.layout === 'flex'
    },
    justifyContent: {
      type: 'enum',
      label: '主轴对齐',
      options: [
        { value: 'flex-start', label: '起始' },
        { value: 'flex-end', label: '结束' },
        { value: 'center', label: '居中' },
        { value: 'space-between', label: '两端对齐' },
        { value: 'space-around', label: '均匀分布' },
        { value: 'space-evenly', label: '等间距' }
      ],
      default: 'flex-start',
      visible: (props) => props.layout === 'flex'
    },
    alignItems: {
      type: 'enum',
      label: '交叉轴对齐',
      options: [
        { value: 'flex-start', label: '起始' },
        { value: 'flex-end', label: '结束' },
        { value: 'center', label: '居中' },
        { value: 'stretch', label: '拉伸' },
        { value: 'baseline', label: '基线' }
      ],
      default: 'stretch',
      visible: (props) => props.layout === 'flex'
    },
    flexWrap: {
      type: 'enum',
      label: '换行',
      options: [
        { value: 'nowrap', label: '不换行' },
        { value: 'wrap', label: '换行' },
        { value: 'wrap-reverse', label: '反向换行' }
      ],
      default: 'nowrap',
      visible: (props) => props.layout === 'flex'
    },
    gap: {
      type: 'number',
      label: '间距',
      default: 0,
      min: 0,
      max: 100
    },
    padding: {
      type: 'number',
      label: '内边距',
      default: 0,
      min: 0,
      max: 100
    },
    overflow: {
      type: 'enum',
      label: '溢出处理',
      options: [
        { value: 'visible', label: '可见' },
        { value: 'hidden', label: '隐藏' },
        { value: 'scroll', label: '滚动' },
        { value: 'auto', label: '自动' }
      ],
      default: 'visible'
    }
  }
}
