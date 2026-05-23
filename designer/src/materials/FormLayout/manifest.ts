/**
 * FormLayout 表单布局组件 Manifest
 */

import type { ComponentManifest } from '@/materials/manifests/manifest-registry'
import { registerManifest } from '@/materials/manifests/manifest-registry'

export const manifest: ComponentManifest = {
  type: 'FormLayout',
  name: '表单布局',
  category: '布局',
  isContainer: true,
  defaultSize: { width: 360, height: 280 },
  props: [
    {
      name: 'columns',
      type: 'number',
      label: '列数',
      group: '布局',
      defaultValue: 2,
      min: 1,
      max: 6,
    },
    {
      name: 'columnGap',
      type: 'number',
      label: '横向间距',
      group: '布局',
      defaultValue: 0,
      min: 0,
      max: 100,
    },
    {
      name: 'rowGap',
      type: 'number',
      label: '纵向间距',
      group: '布局',
      defaultValue: 0,
      min: 0,
      max: 100,
    },
    {
      name: 'showBorder',
      type: 'boolean',
      label: '显示边框',
      group: '样式',
      defaultValue: false,
    },
  ],
}

registerManifest(manifest)

export default manifest
