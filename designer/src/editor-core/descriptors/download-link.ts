/**
 * DownloadLink 下载组件 Descriptor（资产拖拽专用，默认不在物料区展示）
 */

import type { ComponentDescriptor } from './registry'
import DownloadLinkRenderer from '@/materials/DownloadLink/DownloadLinkRenderer.vue'

export const descriptor: ComponentDescriptor = {
  renderTag: 'div',
  propsFilter: () => ({}),
  customRenderer: DownloadLinkRenderer,
  isContainer: false,
  acceptChildren: false,
  isMovable: true,
  defaultSize: { width: 220, height: 32 },
}

export default descriptor
