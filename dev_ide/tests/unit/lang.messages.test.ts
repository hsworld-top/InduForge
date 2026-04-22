import { describe, expect, it } from 'vitest'
import i18n from '@/lang'

describe('i18n messages', () => {
  it('中英文核心文案存在', () => {
    const zhMessages = i18n.global.getLocaleMessage('zh') as Record<string, unknown>
    const enMessages = i18n.global.getLocaleMessage('en') as Record<string, unknown>
    const zhDashboard = zhMessages.dashboard as Record<string, unknown>
    const enDashboard = enMessages.dashboard as Record<string, unknown>
    const zhProfile = zhMessages.profile as Record<string, unknown>
    const enProfile = enMessages.profile as Record<string, unknown>
    const zhAuth = zhMessages.auth as Record<string, unknown>
    const enAuth = enMessages.auth as Record<string, unknown>

    expect(zhDashboard.title).toBe('仪表盘')
    expect(enDashboard.title).toBe('Dashboard')
    expect(zhProfile.uploadAvatar).toBe('上传头像')
    expect(enProfile.uploadAvatar).toBe('Upload Avatar')
    expect(zhAuth.tenantCode).toBe('租户代码')
    expect(enAuth.tenantCode).toBe('Tenant Code')
  })
})
