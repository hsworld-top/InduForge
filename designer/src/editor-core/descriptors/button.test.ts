import { describe, expect, it } from 'vitest'
import { descriptor } from './button'

describe('button descriptor', () => {
  it('maps designer-only shape before rendering and strips non-button helpers', () => {
    expect(
      descriptor.propsFilter?.({
        text: '提交',
        type: 'primary',
        shape: 'circle',
        round: true,
      }),
    ).toEqual({
      type: 'primary',
      round: false,
      circle: true,
    })
  })

  it('preserves legacy round and circle props when shape is not set', () => {
    expect(
      descriptor.propsFilter?.({
        text: '提交',
        type: 'primary',
        round: true,
      }),
    ).toEqual({
      type: 'primary',
      round: true,
    })
  })

  it('maps known icon names to renderable components and preserves custom icon strings', () => {
    const withPresetIcon = descriptor.propsFilter?.({
      text: '下载',
      icon: 'Download',
    })
    expect(typeof withPresetIcon?.icon).toBe('object')
    expect((withPresetIcon?.icon as { name?: string }).name).toBe('IconEpDownload')

    const withCustomIcon = descriptor.propsFilter?.({
      text: '自定义',
      icon: 'CustomIcon',
    })
    expect(withCustomIcon?.icon).toBe('CustomIcon')
  })
})
