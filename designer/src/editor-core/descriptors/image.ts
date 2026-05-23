/**
 * Image 图片组件 Descriptor
 */

import type { ComponentDescriptor } from './registry'

export const descriptor: ComponentDescriptor = {
  renderTag: 'el-image',
  isContainer: false,
  acceptChildren: false,
  isMovable: true,
  defaultSize: { width: 240, height: 160 },
}

export default descriptor
