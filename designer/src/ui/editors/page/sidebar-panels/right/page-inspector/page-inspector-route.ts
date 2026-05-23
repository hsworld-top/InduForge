import type { PageRole, RouteMode } from './page-inspector-types'

const PATH_SEGMENT_SPACE_RE = /\s+/g
const PATH_SEGMENT_FORBIDDEN_RE = /[/?#\\]+/g

export interface ResolveRouteByRoleInput {
  role: PageRole
  name: string
  routeMode?: RouteMode
  routePath?: string
  routeSlug?: string
  parentRoutePath?: string | null
}

function sanitizeSegment(value: string): string {
  const normalized = String(value || '')
    .trim()
    .replace(PATH_SEGMENT_SPACE_RE, '-')
  const sanitized = normalized.replace(PATH_SEGMENT_FORBIDDEN_RE, '-')
  return sanitized || 'page'
}

function normalizeParentRoutePath(value: string | null | undefined): string {
  const text = String(value || '').trim()
  if (!text || text === '/') {
    return ''
  }
  return text.startsWith('/') ? text.replace(/\/+$/, '') : `/${text.replace(/\/+$/, '')}`
}

function normalizeManualPath(value: string): string {
  const text = String(value || '').trim()
  if (!text) {
    return '/'
  }
  return text.startsWith('/') ? text : `/${text}`
}

function extractSlug(path: string): string {
  const segments = normalizeManualPath(path).split('/').filter(Boolean)
  return segments.at(-1) || ''
}

export function buildBusinessPagePath(slug: string, parentRoutePath?: string | null): string {
  const normalizedSlug = sanitizeSegment(slug)
  const parent = normalizeParentRoutePath(parentRoutePath)
  return parent ? `${parent}/${normalizedSlug}` : `/${normalizedSlug}`
}

export function resolveRouteByRole(input: ResolveRouteByRoleInput): {
  routeMode: RouteMode
  routePath: string
  routeSlug: string
} {
  if (input.role === 'home') {
    return { routeMode: 'auto', routePath: '/', routeSlug: '' }
  }
  if (input.role === 'login') {
    return { routeMode: 'auto', routePath: '/login', routeSlug: 'login' }
  }
  if (input.role === 'logout') {
    return { routeMode: 'auto', routePath: '/logout', routeSlug: 'logout' }
  }
  if (input.routeMode === 'manual' && input.routePath) {
    const routePath = normalizeManualPath(input.routePath)
    return {
      routeMode: 'manual',
      routePath,
      routeSlug: sanitizeSegment(input.routeSlug || extractSlug(routePath)),
    }
  }

  const routeSlug = sanitizeSegment(input.routeSlug || input.name)
  return {
    routeMode: 'auto',
    routePath: buildBusinessPagePath(routeSlug, input.parentRoutePath),
    routeSlug,
  }
}
