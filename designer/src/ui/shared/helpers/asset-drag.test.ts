import { describe, expect, it } from 'vitest'
import {
  buildAssetDragPayload,
  buildAssetNodeProps,
  parseAssetDragPayload,
  resolveAssetComponentType,
  serializeAssetDragPayload,
} from './asset-drag'

describe('asset-drag utils', () => {
  it('应构建并解析资源拖拽 payload', () => {
    const payload = buildAssetDragPayload({
      id: 'asset-1',
      name: 'abc.png',
      url: '/api/file/asset-1',
      type: 'image',
      mimeType: 'image/png',
      size: 1234,
    })

    expect(payload).not.toBeNull()
    const serialized = serializeAssetDragPayload(payload!)
    const parsed = parseAssetDragPayload(serialized)
    expect(parsed?.id).toBe('asset-1')
    expect(parsed?.type).toBe('image')
  })

  it('图片和视频资源应映射为对应组件', () => {
    const imageType = resolveAssetComponentType({
      id: 'asset-img',
      type: 'image',
      mimeType: 'image/jpeg',
    })
    const videoType = resolveAssetComponentType({
      id: 'asset-video',
      type: 'video',
      mimeType: 'video/mp4',
    })
    const fileType = resolveAssetComponentType({
      id: 'asset-zip',
      type: 'other',
      mimeType: 'application/zip',
    })
    expect(imageType).toBe('Image')
    expect(videoType).toBe('Video')
    expect(fileType).toBe('DownloadLink')
  })

  it('应生成 Image 组件属性', () => {
    const props = buildAssetNodeProps(
      {
        id: 'asset-2',
        displayName: '封面图',
        url: '/api/v1/design/projects/p1/assets/a2/file',
      },
      'Image',
    )

    expect(props).toMatchObject({
      src: '/api/v1/design/projects/p1/assets/a2/file',
      alt: '封面图',
      fit: 'contain',
    })
  })

  it('应生成 Video 组件属性', () => {
    const props = buildAssetNodeProps(
      {
        id: 'asset-4',
        displayName: '演示视频.mp4',
        url: '/api/v1/design/projects/p1/assets/a4/file',
      },
      'Video',
    )

    expect(props).toMatchObject({
      src: '/api/v1/design/projects/p1/assets/a4/file',
      controls: true,
      autoplay: false,
      loop: false,
      muted: false,
    })
  })

  it('应生成 DownloadLink 组件属性', () => {
    const props = buildAssetNodeProps(
      {
        id: 'asset-3',
        displayName: '说明文档.zip',
        url: '/api/v1/design/projects/p1/assets/a3/file',
      },
      'DownloadLink',
    )

    expect(props).toMatchObject({
      text: '说明文档.zip',
      href: '/api/v1/design/projects/p1/assets/a3/file',
      displayMode: 'button',
      actionMode: 'download',
      triggerMode: 'double',
      downloadFileName: '说明文档.zip',
      target: '_blank',
    })
  })
})
