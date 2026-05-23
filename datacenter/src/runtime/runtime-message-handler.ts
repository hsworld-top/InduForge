// @ts-nocheck
import { applyThemeToDocument } from './host-bootstrap'
import { setDatacenterLocale } from '../i18n/runtime'
import { Storage } from '../utils/storage'

export function createRuntimeMessageHandler({
  getTrustedOriginSet,
  getTrustedSources,
  isTrustedHostMessage,
  handleBootstrapResponseMessage,
  handleAuthRefreshedMessage,
} = {}) {
  return (event) => {
    const data = event.data
    if (!data || typeof data !== 'object') {
      return
    }

    if (
      !['APP_BOOTSTRAP_RESPONSE', 'AUTH_REFRESHED', 'THEME_UPDATE', 'LOCALE_UPDATE'].includes(
        data.type,
      )
    ) {
      return
    }

    if (
      !isTrustedHostMessage(event, {
        trustedOrigins: getTrustedOriginSet(),
        trustedSources: getTrustedSources(),
      })
    ) {
      return
    }

    if (handleBootstrapResponseMessage(data)) {
      return
    }

    if (handleAuthRefreshedMessage(data)) {
      return
    }

    if (data.type === 'THEME_UPDATE' && ['light', 'dark'].includes(data.theme)) {
      Storage.setTheme(data.theme)
      applyThemeToDocument(data.theme)
      return
    }

    if (data.type === 'LOCALE_UPDATE' && typeof data.locale === 'string') {
      const locale = data.locale.trim()
      if (locale) {
        setDatacenterLocale(locale)
      }
    }
  }
}
