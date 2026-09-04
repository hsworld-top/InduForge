import { describe, expect, it } from 'vitest'
import {
  readFrontendLinuxProxyTarget,
  validateFrontendLinuxProxyTarget,
} from '../../../scripts/dev/frontend-linux-env.mjs'

describe('frontend-linux 开发配置', () => {
  it('读取带引号的中心 origin，并接受 HTTP(S) 纯 origin', () => {
    expect(
      readFrontendLinuxProxyTarget('IF_FRONTEND_PROXY_TARGET="https://center.induforge.local:18080"'),
    ).toBe('https://center.induforge.local:18080')
    expect(validateFrontendLinuxProxyTarget('https://center.induforge.local:18080')).toEqual({
      valid: true,
      target: 'https://center.induforge.local:18080',
    })
  })

  it.each([
    '',
    'http://your-center-host:18080',
    'ftp://center.induforge.local',
    'https://user:password@center.induforge.local',
    'https://center.induforge.local/api/v1',
    'https://center.induforge.local?tenant=demo',
    'https://center.induforge.local#section',
  ])('拒绝无效或不安全的中心目标：%s', (target) => {
    expect(validateFrontendLinuxProxyTarget(target).valid).toBe(false)
  })
})
