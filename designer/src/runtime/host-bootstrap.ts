import type { EditorLocale, EditorTheme, EditorUiStore } from '@/stores/editor-ui-store'
import { STORAGE_KEYS } from '@/constants'
import { Storage } from '@/utils/storage'
import { shouldUseDebugMode as resolveDebugModeForPath } from './debug-route'

const DESIGNER_APP_TYPE = 'designer'
const HANDOFF_STORAGE_PREFIX = 'embedded_app_handoff:'
const APP_BOOTSTRAP_REQUEST = 'APP_BOOTSTRAP_REQUEST'
const APP_BOOTSTRAP_RESPONSE = 'APP_BOOTSTRAP_RESPONSE'
const AUTH_REFRESHED = 'AUTH_REFRESHED'

type RuntimeMessageRecord = Record<string, unknown>

type HostMessageTarget = {
  source: Window | null
  origin: string | null
}

export type TrustedMessageSource = MessageEventSource | null

type BootstrapGate = {
  promise: Promise<boolean>
  settle: (value: boolean) => void
}

const HOST_BOOTSTRAP_TIMEOUT_MS = 3000

export type DesignerEntrypointPlan = {
  isDebugRoute: boolean
  handoffId: string | null
  shouldRedirectToIde: boolean
  ideRedirectUrl: string | null
  shouldWaitForBootstrap: boolean
  trustedHostOrigin: string | null
}

export type AppBootstrapRequestMessage = {
  type: typeof APP_BOOTSTRAP_REQUEST
  payload: {
    appType: typeof DESIGNER_APP_TYPE
    handoffId: string | null
    pathname: string
    requestId: string
    search: string
  }
}

export type AppBootstrapResponsePayload = {
  accessToken?: unknown
  appType?: unknown
  handoffId?: unknown
  locale?: unknown
  pid?: unknown
  projectId?: unknown
  refreshToken?: unknown
  tenantId?: unknown
  theme?: unknown
  token?: unknown
}

export type AuthRefreshPayload = {
  accessToken?: unknown
  refreshToken?: unknown
  token?: unknown
}

type DesignerHostBootstrapSession = {
  gate: BootstrapGate
  plan: DesignerEntrypointPlan
  requestMessage: AppBootstrapRequestMessage | null
  target: HostMessageTarget
}

type ResolveEntrypointPlanOptions = {
  ideOrigin?: string
  hasProjectId?: boolean
  isTopLevelWindow?: boolean
  hasToken?: boolean
  referrer?: string
}

type ResolveHostMessageTargetOptions = {
  parentWindow?: Window | null
  referrer?: string
  selfWindow?: Window | null
}

type ResolveDesignerIdeOriginRuntimeOptions = {
  configuredIdeOrigin?: string | null
  currentUrl?: string
  devHost?: string | null
  idePort?: number | string | null
  isDev?: boolean
  referrer?: string
}

type InitializeDesignerHostBootstrapOptions = ResolveEntrypointPlanOptions &
  ResolveHostMessageTargetOptions & {
    currentUrl?: string
  }

type ApplyBootstrapResponseDependencies = {
  allowUiSync?: boolean
  editorUi: EditorUiStore
}

type LocalHandoffProjectContext = {
  projectId: string
  tenantId: string | null
}

const isEditorTheme = (value: unknown): value is EditorTheme =>
  value === 'light' || value === 'dark'

const isEditorLocale = (value: unknown): value is EditorLocale => value === 'zh' || value === 'en'

const asNonEmptyString = (value: unknown): string | null =>
  typeof value === 'string' && value.trim().length > 0 ? value : null

const resolveMessagePayload = (value: unknown): RuntimeMessageRecord => {
  if (value === null || typeof value !== 'object') {
    return {}
  }

  const record = value as RuntimeMessageRecord
  if (record.payload && typeof record.payload === 'object') {
    return record.payload as RuntimeMessageRecord
  }

  return record
}

function resolveLocalHandoffProjectContext(
  handoffId: string | null,
  appType: string,
): LocalHandoffProjectContext | null {
  const normalizedHandoffId = asNonEmptyString(handoffId)
  if (!normalizedHandoffId) {
    return null
  }

  try {
    const rawRecord = localStorage.getItem(`${HANDOFF_STORAGE_PREFIX}${normalizedHandoffId}`)
    if (!rawRecord) {
      return null
    }

    const parsedRecord = JSON.parse(rawRecord) as RuntimeMessageRecord
    if (asNonEmptyString(parsedRecord.handoffId) !== normalizedHandoffId) {
      return null
    }

    if (asNonEmptyString(parsedRecord.appType) !== appType) {
      return null
    }

    const expiresAt =
      typeof parsedRecord.expiresAt === 'number' ? parsedRecord.expiresAt : Number.NaN
    if (Number.isFinite(expiresAt) && expiresAt <= Date.now()) {
      localStorage.removeItem(`${HANDOFF_STORAGE_PREFIX}${normalizedHandoffId}`)
      return null
    }

    const projectId = asNonEmptyString(parsedRecord.projectId)
    if (!projectId) {
      return null
    }

    return {
      projectId,
      tenantId: asNonEmptyString(parsedRecord.tenantId),
    }
  } catch (error) {
    console.warn(`Resolve handoff project context failed for "${normalizedHandoffId}":`, error)
    return null
  }
}

function createBootstrapGate(shouldWaitForBootstrap: boolean): BootstrapGate {
  if (!shouldWaitForBootstrap) {
    return {
      promise: Promise.resolve(true),
      settle: () => {},
    }
  }

  let settled = false
  let resolvePromise = (_value: boolean) => {}
  const promise = new Promise<boolean>((resolve) => {
    resolvePromise = resolve
  })
  const timeoutId = window.setTimeout(() => {
    resolveOnce(false)
  }, HOST_BOOTSTRAP_TIMEOUT_MS)

  /**
   * iframe 正式入口缺 token 时只允许等待有限时间。
   * 宿主不响应、响应 requestId 不匹配，或消息链路异常时，都必须收敛到 false，
   * 让路由守卫继续走现有登录/IDE 回跳兜底，而不是永久挂起。
   */
  const resolveOnce = (value: boolean) => {
    if (settled) {
      return
    }

    settled = true
    window.clearTimeout(timeoutId)
    resolvePromise(value)
  }

  return {
    promise,
    settle: resolveOnce,
  }
}

function createBootstrapRequestId(): string {
  return `designer_bootstrap_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`
}

function resolveHandoffId(url: URL): string | null {
  return asNonEmptyString(url.searchParams.get('handoffId'))
}

function resolveHandoffValue(
  handoff: string | { handoffId?: unknown } | null | undefined,
): string | null {
  if (typeof handoff === 'string') {
    return asNonEmptyString(handoff)
  }

  if (!handoff || typeof handoff !== 'object') {
    return null
  }

  return asNonEmptyString(handoff.handoffId)
}

/**
 * 正式入口的 debug 例外只由 pathname 决定，避免不同入口重复维护判断逻辑。
 */
export function shouldUseDebugMode(pathname: string, debugRouteEnabled?: boolean): boolean {
  return resolveDebugModeForPath(pathname, debugRouteEnabled)
}

/**
 * 顶层窗口缺少可复用的工程会话时才回到 IDE，由 IDE 决定 handoff 恢复与 iframe 挂载。
 * 已经从 IDE 拿到 token 与 projectId 的独立标签页应允许继续运行，避免被硬性拉回宿主。
 */
export function hasReusableEntrypointSession(
  options: {
    hasProjectId?: boolean
    hasToken?: boolean
    handoffId?: string | null
  } = {},
): boolean {
  const hasToken = options.hasToken ?? Boolean(Storage.getToken())
  const handoffId = asNonEmptyString(options.handoffId)
  /**
   * handoffId 代表宿主明确要求恢复某一次新的打开动作。
   * 不能继续把“本地旧 projectId”当成可复用会话；但如果同源 localStorage
   * 里已经存在这次 handoff 对应的 projectId/tenantId，就允许顶层新标签页
   * 直接按 handoff 恢复，避免被无谓拉回 IDE。
   */
  if (handoffId) {
    return Boolean(
      hasToken && resolveLocalHandoffProjectContext(handoffId, DESIGNER_APP_TYPE)?.projectId,
    )
  }

  const hasProjectId = options.hasProjectId ?? Boolean(Storage.getProjectId())
  return hasToken && hasProjectId
}

/**
 * 顶层独立标签页携带 handoffId 时，直接从同源 handoff 记录恢复工程上下文。
 * 这样可以在新标签页进入纯设计中心，同时避免继续复用旧 projectId。
 * @param handoffId - handoff 标识
 * @returns 是否恢复成功
 */
export function restoreEntrypointSessionFromHandoff(handoffId: string | null): boolean {
  const projectContext = resolveLocalHandoffProjectContext(handoffId, DESIGNER_APP_TYPE)
  if (!projectContext) {
    return false
  }

  Storage.setProjectId(projectContext.projectId)
  if (projectContext.tenantId) {
    Storage.setTenantId(projectContext.tenantId)
  } else {
    Storage.remove(STORAGE_KEYS.TENANT_ID)
  }

  return true
}

export function shouldRedirectTopLevelToIde(
  pathname: string,
  isTopLevel: boolean,
  hasReusableSession = hasReusableEntrypointSession(),
): boolean {
  return isTopLevel && !shouldUseDebugMode(pathname) && !hasReusableSession
}

export function resolveTrustedHostOrigin(referrer = ''): string | null {
  if (!referrer) {
    return null
  }

  try {
    return new URL(referrer).origin
  } catch {
    return null
  }
}

function resolveOriginCandidate(value: unknown, baseUrl?: string): string | null {
  const candidate = asNonEmptyString(value)
  if (!candidate) {
    return null
  }

  try {
    return new URL(candidate, baseUrl).origin
  } catch {
    return null
  }
}

/**
 * IDE origin 优先取显式配置，其次取父页面 referrer。
 * 开发态再回退到约定端口，生产态最后回退当前页 origin，
 * 让顶层直接访问、iframe 超时回登录、handoff 恢复三条链路共用同一判定。
 */
export function resolveDesignerIdeOriginFromRuntime(
  options: ResolveDesignerIdeOriginRuntimeOptions = {},
): string {
  const currentUrl = options.currentUrl ?? window.location.href
  const configuredIdeOrigin = resolveOriginCandidate(
    options.configuredIdeOrigin ?? import.meta.env.VITE_IDE_ORIGIN,
    currentUrl,
  )
  if (configuredIdeOrigin) {
    return configuredIdeOrigin
  }

  const referrerOrigin = resolveTrustedHostOrigin(options.referrer ?? document.referrer)
  if (referrerOrigin) {
    return referrerOrigin
  }

  let currentOrigin = window.location.origin
  let currentHost = window.location.hostname || 'localhost'
  let currentProtocol = window.location.protocol || 'http:'

  try {
    const currentLocation = new URL(currentUrl)
    currentOrigin = currentLocation.origin
    currentHost = currentLocation.hostname || currentHost
    currentProtocol = currentLocation.protocol || currentProtocol
  } catch {
    // currentUrl 异常时继续使用浏览器当前地址兜底，避免入口解析直接中断。
  }

  if (options.isDev ?? import.meta.env.DEV) {
    const devHost =
      asNonEmptyString(options.devHost) ??
      asNonEmptyString(import.meta.env.VITE_DEV_HOST) ??
      currentHost
    const idePortValue = options.idePort ?? import.meta.env.VITE_IDE_PORT ?? '18601'
    const idePort = String(idePortValue).trim()
    const protocol = currentProtocol === 'https:' ? 'https:' : 'http:'

    if (devHost && idePort) {
      return `${protocol}//${devHost}:${idePort}`
    }
  }

  return currentOrigin
}

export function resolveDesignerEntrypointPlan(
  currentUrl: string,
  options: ResolveEntrypointPlanOptions = {},
): DesignerEntrypointPlan {
  const url = new URL(currentUrl)
  const referrer = options.referrer ?? ''
  const isDebugRoute = shouldUseDebugMode(url.pathname)
  const handoffId = resolveHandoffId(url)
  const isTopLevelWindow = options.isTopLevelWindow ?? true
  const hasToken = options.hasToken ?? Boolean(Storage.getToken())
  const hasProjectId = options.hasProjectId ?? Boolean(Storage.getProjectId())
  const shouldForceBootstrapByHandoff = Boolean(handoffId) && !isTopLevelWindow
  const shouldRedirectToIde = shouldRedirectTopLevelToIde(
    url.pathname,
    isTopLevelWindow,
    hasReusableEntrypointSession({
      handoffId,
      hasProjectId,
      hasToken,
    }),
  )
  const ideOrigin =
    options.ideOrigin ??
    resolveDesignerIdeOriginFromRuntime({
      currentUrl,
      referrer,
    })
  const ideRedirectUrl = shouldRedirectToIde ? buildIdeRestoreUrl(handoffId, ideOrigin) : null

  return {
    isDebugRoute,
    handoffId,
    shouldRedirectToIde,
    ideRedirectUrl,
    /**
     * iframe 正式入口一旦带 handoffId，就表示宿主显式指定了新的工程上下文，
     * 必须等待 bootstrap 覆盖本地旧 token/projectId；否则会把上一工程错误复用到新入口。
     * 只有“无 handoffId 的独立续用场景”才允许依赖本地 token 直接进入。
     */
    shouldWaitForBootstrap:
      !isDebugRoute && !shouldRedirectToIde && (shouldForceBootstrapByHandoff || !hasToken),
    trustedHostOrigin: resolveTrustedHostOrigin(referrer),
  }
}

export function buildIdeRestoreUrl(
  handoff: string | { handoffId?: unknown } | null | undefined,
  ideOrigin: string,
): string {
  const targetUrl = new URL('/', ideOrigin)
  const handoffId = resolveHandoffValue(handoff)
  if (handoffId) {
    targetUrl.searchParams.set('handoffId', handoffId)
  }

  return targetUrl.toString()
}

export function buildIdeRedirectUrl({
  handoffId,
  ideOrigin,
}: {
  handoffId: string | null
  ideOrigin: string
}): string {
  return buildIdeRestoreUrl(handoffId, ideOrigin)
}

export function buildIdeLoginUrl({
  currentUrl,
  ideOrigin,
}: {
  currentUrl: string
  ideOrigin: string
}): string {
  const targetUrl = new URL('/login', ideOrigin)
  targetUrl.searchParams.set('redirect', currentUrl)
  return targetUrl.toString()
}

export function createBootstrapRequest(currentUrl: string): AppBootstrapRequestMessage {
  const url = new URL(currentUrl)

  return {
    type: APP_BOOTSTRAP_REQUEST,
    payload: {
      appType: DESIGNER_APP_TYPE,
      handoffId: resolveHandoffId(url),
      pathname: url.pathname,
      requestId: createBootstrapRequestId(),
      search: url.search,
    },
  }
}

export const createAppBootstrapRequest = createBootstrapRequest

export function resolveHostMessageTarget(
  options: ResolveHostMessageTargetOptions = {},
): HostMessageTarget {
  const selfWindow = options.selfWindow ?? window
  const parentWindow = options.parentWindow ?? window.parent

  if (!parentWindow || parentWindow === selfWindow) {
    return {
      source: null,
      origin: null,
    }
  }

  return {
    source: parentWindow,
    origin: resolveTrustedHostOrigin(options.referrer ?? document.referrer),
  }
}

let currentSession: DesignerHostBootstrapSession | null = null

export function initializeDesignerHostBootstrap(
  options: InitializeDesignerHostBootstrapOptions = {},
): DesignerHostBootstrapSession {
  const currentUrl = options.currentUrl ?? window.location.href
  const referrer = options.referrer ?? document.referrer
  const planOptions: ResolveEntrypointPlanOptions = {
    hasProjectId: Boolean(Storage.getProjectId()),
    hasToken: Boolean(Storage.getToken()),
    isTopLevelWindow: options.isTopLevelWindow ?? window.parent === window,
    referrer,
  }
  if (options.ideOrigin) {
    planOptions.ideOrigin = options.ideOrigin
  }

  const plan = resolveDesignerEntrypointPlan(currentUrl, planOptions)
  const requestMessage = plan.shouldWaitForBootstrap ? createBootstrapRequest(currentUrl) : null
  const targetOptions: ResolveHostMessageTargetOptions = {
    referrer,
  }
  if (options.parentWindow !== undefined) {
    targetOptions.parentWindow = options.parentWindow
  }
  if (options.selfWindow !== undefined) {
    targetOptions.selfWindow = options.selfWindow
  }

  const target = resolveHostMessageTarget(targetOptions)
  const gate = createBootstrapGate(plan.shouldWaitForBootstrap)

  currentSession = {
    gate,
    plan,
    requestMessage,
    target,
  }

  if (plan.shouldWaitForBootstrap && (!target.origin || !target.source || !requestMessage)) {
    gate.settle(false)
  }

  if (plan.isDebugRoute || plan.shouldRedirectToIde) {
    gate.settle(true)
  }

  return currentSession
}

export function waitForHostBootstrap(): Promise<boolean> {
  return currentSession?.gate.promise ?? Promise.resolve(true)
}

function getCurrentSessionTarget(): HostMessageTarget {
  return currentSession?.target ?? resolveHostMessageTarget()
}

export function getTrustedHostOriginSet(): Set<string> {
  const origin = getCurrentSessionTarget().origin
  return origin ? new Set([origin]) : new Set<string>()
}

export function getTrustedHostSources(): TrustedMessageSource[] {
  const source = getCurrentSessionTarget().source
  return source ? [source] : []
}

/**
 * origin 与 source 必须同时命中宿主登记信息，才能接受父窗口下发的运行时消息。
 */
export function isTrustedHostOrigin(origin: string, trustedOrigins: Iterable<string>): boolean {
  if (!origin) {
    return false
  }

  for (const trustedOrigin of trustedOrigins) {
    if (trustedOrigin === origin) {
      return true
    }
  }

  return false
}

export function isTrustedHostSource(
  source: TrustedMessageSource,
  trustedSources: Iterable<TrustedMessageSource>,
): boolean {
  if (!source) {
    return false
  }

  for (const trustedSource of trustedSources) {
    if (trustedSource === source) {
      return true
    }
  }

  return false
}

export function isTrustedHostMessage(
  event: Pick<MessageEvent, 'origin' | 'source'>,
  options: {
    trustedOrigins?: Iterable<string>
    trustedSources?: Iterable<TrustedMessageSource>
  } = {},
): boolean {
  const trustedOrigins = options.trustedOrigins ?? getTrustedHostOriginSet()
  const trustedSources = options.trustedSources ?? getTrustedHostSources()

  return (
    isTrustedHostOrigin(event.origin, trustedOrigins) &&
    isTrustedHostSource(event.source as TrustedMessageSource, trustedSources)
  )
}

export function postAppBootstrapRequest(): boolean {
  if (!currentSession?.requestMessage) {
    return false
  }

  return postMessageToHost(currentSession.requestMessage)
}

export function postMessageToHost(message: unknown): boolean {
  const target = getCurrentSessionTarget()
  if (!target.source?.postMessage || !target.origin) {
    return false
  }

  target.source.postMessage(message, target.origin)
  return true
}

function resolveToken(payload: AppBootstrapResponsePayload | AuthRefreshPayload): string | null {
  return asNonEmptyString(payload.token) ?? asNonEmptyString(payload.accessToken)
}

export function applyBootstrapResponsePayload(
  payload: AppBootstrapResponsePayload,
  { editorUi }: ApplyBootstrapResponseDependencies,
): void {
  applyBootstrapPayload(payload, { editorUi })
}

export function applyBootstrapPayload(
  payload: AppBootstrapResponsePayload,
  { allowUiSync = true, editorUi }: ApplyBootstrapResponseDependencies,
): void {
  const nextToken = resolveToken(payload)
  const nextRefreshToken = asNonEmptyString(payload.refreshToken)
  const nextProjectId = asNonEmptyString(payload.projectId) ?? asNonEmptyString(payload.pid)
  const nextTenantId = asNonEmptyString(payload.tenantId)
  const runtimeTheme = isEditorTheme(payload.theme) ? payload.theme : undefined
  const runtimeLocale = isEditorLocale(payload.locale) ? payload.locale : undefined

  if (nextToken) {
    Storage.setToken(nextToken)
  }

  if (nextRefreshToken) {
    Storage.setRefreshToken(nextRefreshToken)
  }

  if (nextProjectId) {
    Storage.setProjectId(nextProjectId)
  }

  if (nextTenantId) {
    Storage.setTenantId(nextTenantId)
  }

  if (allowUiSync) {
    editorUi.initFromRuntime({
      theme: runtimeTheme,
      locale: runtimeLocale,
    })
  }

  currentSession?.gate.settle(true)
}

export function applyAuthRefreshedPayload(payload: AuthRefreshPayload): void {
  const nextToken = resolveToken(payload)
  const nextRefreshToken = asNonEmptyString(payload.refreshToken)

  if (!nextToken) {
    return
  }

  Storage.setToken(nextToken)
  if (nextRefreshToken) {
    Storage.setRefreshToken(nextRefreshToken)
  }
}

export function handleBootstrapResponseMessage(
  data: unknown,
  dependencies: ApplyBootstrapResponseDependencies,
): boolean {
  const message = data as RuntimeMessageRecord
  if (message?.type !== APP_BOOTSTRAP_RESPONSE) {
    return false
  }

  const payload = resolveMessagePayload(data) as AppBootstrapResponsePayload
  const responseRequestId =
    asNonEmptyString(message.requestId) ??
    asNonEmptyString((payload as RuntimeMessageRecord).requestId)
  const expectedRequestId = currentSession?.requestMessage?.payload.requestId ?? null

  if (responseRequestId && expectedRequestId && responseRequestId !== expectedRequestId) {
    return false
  }

  applyBootstrapPayload(payload, dependencies)
  return true
}

export function handleAuthRefreshedMessage(data: unknown): boolean {
  const message = data as RuntimeMessageRecord
  if (message?.type !== AUTH_REFRESHED) {
    return false
  }

  applyAuthRefreshedPayload(resolveMessagePayload(data) as AuthRefreshPayload)
  return true
}

export function resetHostBootstrapSessionForTests(): void {
  currentSession = null
}
