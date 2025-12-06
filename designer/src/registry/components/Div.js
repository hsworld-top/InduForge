/**
 * Div Component - 基础容器
 * 参考 OpenTiny 的 Div 容器
 */
export default {
  type: 'Div',
  name: 'Div',
  category: 'container',
  icon: 'grid',
  defaultProps: {},
  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 300,
    height: 200,
    padding: '16px',
    backgroundColor: '#ffffff',
    border: '1px solid #dcdfe6',
    borderRadius: '4px',
  },
  propsSchema: {},
}
