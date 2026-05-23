/**
 * Video 视频组件 Descriptor
 */

import type { ComponentDescriptor } from './registry'

export const descriptor: ComponentDescriptor = {
  renderTag: 'video',
  isContainer: false,
  acceptChildren: false,
  isMovable: true,
  defaultSize: { width: 320, height: 180 },
}

export default descriptor
