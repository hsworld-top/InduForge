import { STORAGE_KEYS } from "../constants/index.js";
import { Storage } from "../utils/storage.js";

const APP_NAME = "datacenter";
const APP_BOOTSTRAP_REQUEST = "APP_BOOTSTRAP_REQUEST";
const APP_BOOTSTRAP_RESPONSE = "APP_BOOTSTRAP_RESPONSE";
const AUTH_REFRESHED = "AUTH_REFRESHED";
const AUTH_EXPIRED = "AUTH_EXPIRED";

const DEFAULT_BOOTSTRAP_TIMEOUT_MS = 3000;

let currentSession = null;
let bootstrapListenerAttached = false;

function getRuntimeWindow() {
  return typeof window === "undefined" ? null : window;
}

function getRuntimeDocumentReferrer() {
  return typeof document === "undefined" ? "" : document.referrer || "";
}

function getCurrentLocationHref() {
  const runtimeWindow = getRuntimeWindow();
  return runtimeWindow?.location?.href || "http://localhost/";
}

function asNonEmptyString(value) {
  return typeof value === "string" && value.trim().length > 0 ? value : null;
}

function toAbsoluteUrl(value, base = getCurrentLocationHref()) {
  try {
    return new URL(value, base);
  } catch {
    return null;
  }
}

function resolveTrustedOriginFromReferrer(referrer) {
  const trustedReferrer = asNonEmptyString(referrer);
  if (!trustedReferrer) {
    return null;
  }

  try {
    return new URL(trustedReferrer).origin;
  } catch {
    return null;
  }
}

function resolveTrustedSourceFromWindow(parentWindow, selfWindow) {
  if (!parentWindow || parentWindow === selfWindow) {
    return null;
  }

  return parentWindow;
}

function normalizeTimeoutMs(timeoutMs) {
  return Number.isFinite(timeoutMs) && timeoutMs > 0 ? timeoutMs : DEFAULT_BOOTSTRAP_TIMEOUT_MS;
}

function resolveHandoffValue(handoff) {
  if (typeof handoff === "string") {
    return asNonEmptyString(handoff);
  }

  if (!handoff || typeof handoff !== "object") {
    return null;
  }

  return asNonEmptyString(handoff.handoffId);
}

function resolveHandoffFromUrl(url) {
  const handoffId = asNonEmptyString(url.searchParams.get("handoffId"));
  return handoffId ? { handoffId } : null;
}

function clearAuthStorage() {
  Storage.remove(STORAGE_KEYS.TOKEN);
  Storage.remove(STORAGE_KEYS.REFRESH_TOKEN);
  Storage.remove(STORAGE_KEYS.USER_INFO);
  Storage.remove(STORAGE_KEYS.TENANT_ID);
}

function clearProjectStorage() {
  Storage.remove(STORAGE_KEYS.PROJECT_ID);
}

function ensureBootstrapListener() {
  const runtimeWindow = getRuntimeWindow();
  if (!runtimeWindow || bootstrapListenerAttached) {
    return;
  }

  runtimeWindow.addEventListener("message", handleWindowMessage);
  bootstrapListenerAttached = true;
}

function settleCurrentSession(value) {
  if (!currentSession) {
    return;
  }

  currentSession.bootstrapGate.settle(value);
}

function resolveMessagePayload(data) {
  if (!data || typeof data !== "object") {
    return {};
  }

  const record = data;
  if (record.payload && typeof record.payload === "object") {
    return record.payload;
  }

  return record;
}

function handleWindowMessage(event) {
  if (!currentSession || !isTrustedHostMessage(event)) {
    return;
  }

  const data = event.data;
  if (!data || typeof data !== "object" || data.type !== APP_BOOTSTRAP_RESPONSE) {
    return;
  }

  const responsePayload = resolveMessagePayload(data);
  const responseRequestId = asNonEmptyString(data.requestId) || asNonEmptyString(responsePayload.requestId);
  const expectedRequestId = currentSession.requestMessage?.requestId || null;

  if (expectedRequestId && responseRequestId && responseRequestId !== expectedRequestId) {
    return;
  }

  applyBootstrapPayload(responsePayload);
  settleCurrentSession(true);
}

function createBootstrapGate(timeoutMs) {
  const runtimeWindow = getRuntimeWindow();
  const timerApi = runtimeWindow || globalThis;
  let settled = false;
  let resolvePromise = () => {};

  const promise = new Promise((resolve) => {
    resolvePromise = resolve;
  });

  const settle = (value) => {
    if (settled) {
      return;
    }

    settled = true;
    if (timeoutId !== null) {
      timerApi.clearTimeout(timeoutId);
    }
    resolvePromise(value);
  };

  const timeoutId = timerApi.setTimeout(() => {
    settle(false);
  }, timeoutMs);

  return { promise, settle };
}

function postMessageToTarget(target, origin, message) {
  if (!target?.postMessage || !origin) {
    return false;
  }

  target.postMessage(message, origin);
  return true;
}

function resolveBootstrapTarget(options = {}) {
  const runtimeWindow = options.selfWindow ?? getRuntimeWindow();
  const parentWindow = options.parentWindow ?? runtimeWindow?.parent ?? null;
  const source = resolveTrustedSourceFromWindow(parentWindow, runtimeWindow);
  const origin =
    asNonEmptyString(options.trustedOrigin) ||
    resolveTrustedOriginFromReferrer(options.referrer ?? getRuntimeDocumentReferrer());

  return {
    origin,
    source,
  };
}

function resolveIdeOriginFromRuntime(options = {}) {
  const currentUrl = options.currentUrl ?? getCurrentLocationHref();
  const configuredIdeOrigin = asNonEmptyString(options.configuredIdeOrigin ?? import.meta.env?.VITE_IDE_ORIGIN);
  if (configuredIdeOrigin) {
    return configuredIdeOrigin;
  }

  const referrerOrigin = resolveTrustedOriginFromReferrer(options.referrer ?? getRuntimeDocumentReferrer());
  if (referrerOrigin) {
    return referrerOrigin;
  }

  const currentLocation = toAbsoluteUrl(currentUrl);
  const currentOrigin = currentLocation?.origin || getRuntimeWindow()?.location?.origin || "http://localhost";

  if (options.isDev ?? import.meta.env?.DEV) {
    const devHost = asNonEmptyString(options.devHost ?? import.meta.env?.VITE_DEV_HOST) || "localhost";
    const idePort = asNonEmptyString(String(options.idePort ?? import.meta.env?.VITE_IDE_PORT ?? "18601"));
    const protocol = currentLocation?.protocol === "https:" ? "https:" : "http:";

    if (devHost && idePort) {
      return `${protocol}//${devHost}:${idePort}`;
    }
  }

  return currentOrigin;
}

function resolveTrustedTargetSet() {
  if (!currentSession?.target?.origin || !currentSession?.target?.source) {
    return {
      origins: new Set(),
      sources: new Set(),
    };
  }

  return {
    origins: new Set([currentSession.target.origin]),
    sources: new Set([currentSession.target.source]),
  };
}

export function shouldUseDebugMode(pathname) {
  return pathname === "/datacenter/debug" || pathname.startsWith("/datacenter/debug/");
}

export function shouldRedirectTopLevelToIde(pathname, isTopLevel) {
  return Boolean(isTopLevel) && !shouldUseDebugMode(pathname);
}

export function buildIdeRestoreUrl(handoff, ideOrigin) {
  const targetOrigin = asNonEmptyString(ideOrigin) || resolveIdeOriginFromRuntime();
  const targetUrl = new URL("/", targetOrigin);
  const handoffId = resolveHandoffValue(handoff);

  if (handoffId) {
    targetUrl.searchParams.set("handoffId", handoffId);
  }

  return targetUrl.toString();
}

export function createBootstrapRequest({
  handoff = null,
  requestId = createBootstrapRequestId(),
  url = getCurrentLocationHref(),
} = {}) {
  const resolvedUrl = toAbsoluteUrl(url) || toAbsoluteUrl(getCurrentLocationHref());
  const handoffPayload = resolveHandoffValue(handoff);

  return {
    type: APP_BOOTSTRAP_REQUEST,
    app: APP_NAME,
    requestId,
    handoff: handoffPayload ? { handoffId: handoffPayload } : null,
    requestedPath: resolvedUrl?.pathname || "/",
    requestedQuery: resolvedUrl?.search || "",
  };
}

export function createAuthRefreshedMessage(token, refreshToken = null) {
  return {
    type: AUTH_REFRESHED,
    app: APP_NAME,
    token,
    refreshToken: refreshToken ?? null,
  };
}

export function createAuthExpiredMessage() {
  return {
    type: AUTH_EXPIRED,
    app: APP_NAME,
  };
}

export function applyBootstrapPayload(payload = {}) {
  const nextToken = asNonEmptyString(payload.token) || asNonEmptyString(payload.accessToken);
  const nextRefreshToken = asNonEmptyString(payload.refreshToken);
  const nextTenantId = asNonEmptyString(payload.tenantId);
  const nextProjectId = asNonEmptyString(payload.projectId) || asNonEmptyString(payload.pid);
  const nextTheme = asNonEmptyString(payload.theme);
  const nextLocale = asNonEmptyString(payload.locale);

  if (nextToken) {
    Storage.setToken(nextToken);
  }

  if (nextRefreshToken) {
    Storage.setRefreshToken(nextRefreshToken);
  }

  if (nextTenantId) {
    Storage.setTenantId(nextTenantId);
  }

  if (nextProjectId) {
    Storage.setProjectId(nextProjectId);
  }

  if (nextTheme === "light" || nextTheme === "dark") {
    Storage.set(STORAGE_KEYS.THEME, nextTheme);
  }

  if (nextLocale) {
    Storage.set(STORAGE_KEYS.LANGUAGE, nextLocale);
  }

  return {
    token: nextToken,
    refreshToken: nextRefreshToken,
    tenantId: nextTenantId,
    projectId: nextProjectId,
    theme: nextTheme,
    locale: nextLocale,
  };
}

export function isTrustedHostMessage(event, options = {}) {
  const trustedOrigins =
    options.trustedOrigins ||
    (currentSession?.target?.origin ? new Set([currentSession.target.origin]) : resolveTrustedTargetSet().origins);
  const trustedSources =
    options.trustedSources ||
    (currentSession?.target?.source ? new Set([currentSession.target.source]) : resolveTrustedTargetSet().sources);

  if (!event?.origin || !event?.source) {
    return false;
  }

  return trustedOrigins.has(event.origin) && trustedSources.has(event.source);
}

export function handleBootstrapResponseMessage(event) {
  if (!isTrustedHostMessage(event)) {
    return false;
  }

  const data = event?.data;
  if (!data || typeof data !== "object" || data.type !== APP_BOOTSTRAP_RESPONSE) {
    return false;
  }

  const responsePayload = resolveMessagePayload(data);
  const responseRequestId = asNonEmptyString(data.requestId) || asNonEmptyString(responsePayload.requestId);
  const expectedRequestId = currentSession?.requestMessage?.requestId || null;

  if (expectedRequestId && responseRequestId && responseRequestId !== expectedRequestId) {
    return false;
  }

  applyBootstrapPayload(responsePayload);
  settleCurrentSession(true);
  return true;
}

export function initializeHostBootstrap(options = {}) {
  const currentUrl = options.currentUrl ?? getCurrentLocationHref();
  const currentLocation = toAbsoluteUrl(currentUrl) || toAbsoluteUrl(getCurrentLocationHref());
  const isTopLevelWindow =
    options.isTopLevelWindow ?? (getRuntimeWindow()?.parent === getRuntimeWindow());
  const ideOrigin = resolveIdeOriginFromRuntime({
    configuredIdeOrigin: options.ideOrigin,
    currentUrl,
    devHost: options.devHost,
    idePort: options.idePort,
    isDev: options.isDev,
    referrer: options.referrer,
  });
  const target = resolveBootstrapTarget({
    parentWindow: options.parentWindow,
    referrer: options.referrer,
    selfWindow: options.selfWindow,
    trustedOrigin: options.trustedOrigin || ideOrigin,
  });
  const handoff = options.handoff ?? resolveHandoffFromUrl(currentLocation || new URL(currentUrl));
  const shouldRedirectToIde = shouldRedirectTopLevelToIde(currentLocation?.pathname || "/", isTopLevelWindow);
  const shouldWaitForBootstrap =
    !shouldUseDebugMode(currentLocation?.pathname || "/") &&
    !shouldRedirectToIde &&
    !Storage.getToken();
  const bootstrapGate = createBootstrapGate(normalizeTimeoutMs(options.bootstrapTimeoutMs));
  const requestMessage = shouldWaitForBootstrap
    ? createBootstrapRequest({
        handoff,
        url: currentLocation?.toString() || currentUrl,
      })
    : null;

  currentSession = {
    currentUrl: currentLocation?.toString() || currentUrl,
    ideOrigin,
    target,
    handoff,
    requestMessage,
    bootstrapGate,
    shouldRedirectToIde,
    shouldWaitForBootstrap,
  };

  ensureBootstrapListener();

  if (currentSession.shouldRedirectToIde || !currentSession.shouldWaitForBootstrap) {
    bootstrapGate.settle(true);
  }

  if (currentSession.shouldWaitForBootstrap && (!target.origin || !target.source || !requestMessage)) {
    bootstrapGate.settle(false);
  }

  return currentSession;
}

export function waitForHostBootstrap() {
  return currentSession?.bootstrapGate.promise || Promise.resolve(true);
}

export function postAppBootstrapRequest() {
  if (!currentSession?.requestMessage) {
    return false;
  }

  return postMessageToTarget(currentSession.target.source, currentSession.target.origin, currentSession.requestMessage);
}

export function postMessageToHost(message) {
  if (!currentSession?.target?.source || !currentSession?.target?.origin) {
    return false;
  }

  return postMessageToTarget(currentSession.target.source, currentSession.target.origin, message);
}

export function getTrustedHostOriginSet() {
  return resolveTrustedTargetSet().origins;
}

export function getTrustedHostSources() {
  return resolveTrustedTargetSet().sources;
}

export function resetHostBootstrapStateForTests() {
  const runtimeWindow = getRuntimeWindow();
  if (runtimeWindow && bootstrapListenerAttached) {
    runtimeWindow.removeEventListener("message", handleWindowMessage);
  }

  currentSession = null;
  bootstrapListenerAttached = false;
}

function createBootstrapRequestId() {
  return `datacenter_bootstrap_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`;
}

