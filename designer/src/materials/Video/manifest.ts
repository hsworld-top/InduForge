/**
 * Video 视频组件 Manifest
 * 该组件保留给资产拖拽能力，不在默认物料区展示。
 */

import type { ComponentManifest } from '@/materials/manifests/manifest-registry'
import { registerManifest } from '@/materials/manifests/manifest-registry'

export const manifest: ComponentManifest = {
  type: 'Video',
  name: '视频',
  category: 'PC端组件',
  defaultSize: { width: 320, height: 180 },
  props: [
    {
      name: 'src',
      type: 'string',
      label: '视频地址',
      group: '基础',
      defaultValue: '',
      placeholder: '请输入视频 URL',
    },
    {
      name: 'controls',
      type: 'boolean',
      label: '显示控制条',
      group: '行为',
      defaultValue: true,
    },
    {
      name: 'autoplay',
      type: 'boolean',
      label: '自动播放',
      group: '行为',
      defaultValue: false,
    },
    {
      name: 'loop',
      type: 'boolean',
      label: '循环播放',
      group: '行为',
      defaultValue: false,
    },
    {
      name: 'muted',
      type: 'boolean',
      label: '静音',
      group: '行为',
      defaultValue: false,
    },
  ],
}

registerManifest(manifest)

export default manifest
