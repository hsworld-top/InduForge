/**
 * FormLayout 表单布局组件 Descriptor
 */

import type { ComponentDescriptor, DescriptorNode } from './registry'

export const descriptor: ComponentDescriptor = {
  renderTag: 'div',
  isContainer: true,
  childLayout: 'flex',
  childPositioning: 'flow',
  childFlowLayout: { grow: 0, shrink: 0, basis: 'auto', alignSelf: 'start' },
  defaultSize: { width: 360, height: 280 },
  containerStyle: (node: DescriptorNode) => {
    const props = node.props ?? {}
    const legacyGap = Number(props.itemGap)
    const rawColumnGap = Number(props.columnGap ?? props.itemGap)
    const rawRowGap = Number(props.rowGap ?? props.itemGap)
    const columnGap = Number.isFinite(rawColumnGap)
      ? Math.max(0, rawColumnGap)
      : Number.isFinite(legacyGap)
        ? Math.max(0, legacyGap)
        : 0
    const rowGap = Number.isFinite(rawRowGap)
      ? Math.max(0, rawRowGap)
      : Number.isFinite(legacyGap)
        ? Math.max(0, legacyGap)
        : 0
    const rawColumns = Number(props.columns)
    const columns = Number.isFinite(rawColumns)
      ? Math.min(6, Math.max(1, Math.floor(rawColumns)))
      : 2
    const style: Record<string, string | number> = {
      display: 'grid',
      gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
      gridAutoRows: 'max-content',
      alignItems: 'start',
      alignContent: 'start',
      position: 'relative',
      boxSizing: 'border-box',
      width: '100%',
      minHeight: '120px',
      gridAutoFlow: 'row',
      gridAutoColumns: 'minmax(0, 1fr)',
      columnGap: `${columnGap}px`,
      rowGap: `${rowGap}px`,
    }
    if (props.showBorder === true) {
      style.border = '1px solid #dcdfe6'
      style.borderRadius = '4px'
    }
    return style
  },
  childStyle: () => ({
    minWidth: '0',
    width: '100%',
    height: 'auto',
    alignSelf: 'start',
  }),
}

export default descriptor
