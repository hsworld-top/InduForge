import defaultLogoUrl from '@/assets/images/default-logo.svg'
import { STORAGE_KEYS } from '@/constants'
import { Storage } from '@/utils/storage'

type TenantBrand = {
  name?: unknown
  logoUrl?: unknown
}

const DEFAULT_TENANT_NAME = 'InduFrame'

const normalizeText = (value: unknown): string => {
  return typeof value === 'string' ? value.trim() : ''
}

const normalizeTenantBrand = (tenant?: TenantBrand | null): TenantBrand | null => {
  const name = normalizeText(tenant?.name)
  const logoUrl = normalizeText(tenant?.logoUrl)
  return name || logoUrl ? { name, logoUrl } : null
}

export const readStoredTenantBrand = (): TenantBrand | null => {
  const storedBrand = normalizeTenantBrand(Storage.get<TenantBrand>(STORAGE_KEYS.TENANT_BRAND))
  if (storedBrand) return storedBrand

  const userInfo = Storage.getUserInfo()
  const userTenant =
    userInfo?.tenant && typeof userInfo.tenant === 'object'
      ? normalizeTenantBrand(userInfo.tenant as TenantBrand)
      : null
  return userTenant
}

export const storeTenantBrand = (tenant?: TenantBrand | null): void => {
  const brand = normalizeTenantBrand(tenant)
  if (!brand) return
  Storage.set(STORAGE_KEYS.TENANT_BRAND, brand)
}

export const clearTenantBrand = (): void => {
  Storage.remove(STORAGE_KEYS.TENANT_BRAND)
}

export const resolveTenantBrandName = (tenant?: TenantBrand | null): string => {
  return normalizeText(tenant?.name) || DEFAULT_TENANT_NAME
}

export const resolveTenantBrandLogo = (tenant?: TenantBrand | null): string => {
  return normalizeText(tenant?.logoUrl) || defaultLogoUrl
}

/**
 * 固定浏览器标签品牌，不随内部工作台标签切换而变化。
 * @param tenant 当前租户品牌信息
 */
export const applyTenantBrowserBrand = (tenant?: TenantBrand | null): void => {
  if (typeof document === 'undefined') return

  const brand = normalizeTenantBrand(tenant) || readStoredTenantBrand()
  if (tenant) {
    storeTenantBrand(brand)
  }

  document.title = resolveTenantBrandName(brand)

  const iconHref = resolveTenantBrandLogo(brand)
  let iconLink = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (!iconLink) {
    iconLink = document.createElement('link')
    iconLink.rel = 'icon'
    document.head.appendChild(iconLink)
  }
  iconLink.type = iconHref.endsWith('.svg') ? 'image/svg+xml' : 'image/png'
  iconLink.href = iconHref
}
