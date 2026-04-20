import { Storage } from "../utils/storage.js";

export const DATACENTER_APP = "datacenter";
export const APP_BOOTSTRAP_REQUEST = "APP_BOOTSTRAP_REQUEST";
export const APP_BOOTSTRAP_RESPONSE = "APP_BOOTSTRAP_RESPONSE";
export const AUTH_REFRESHED = "AUTH_REFRESHED";
export const AUTH_EXPIRED = "AUTH_EXPIRED";
export const APP_BOOTSTRAP_TIMEOUT_MS = 3000;

const DEBUG_BASE_PATH = "/datacenter/debug";

let currentSession = null;

const asNonEmptyString = (value) =>
  typeof value === "string" && value.trim().length > 0 ? value.trim() : null;

const getTimerApi = () => globalThis.window ?? globalThis;

const resolveWindowLike = () => globalThis.window ?? null;

const resolveDocumentLike = () => globalThis.document ?? null;

const resolveCurrentUrl = (url) => {
  const candidate =
    asNonEmptyString(url) ?? resolveWindowLike()?.location?.href;
  if (!candidate) {
    throw new Error("缺少当前 URL，无法初始化宿主 bootstrap");
  }
  return new URL(candidate);
};

const resolveHandoffValue = (handoff) => {
  if (typeof handoff === "string") {
    return asNonEmptyString(handoff);
  }

  if (!handoff || typeof handoff !== "object") {
    return null;
  }

  return asNonEmptyString(handoff.handoff);
};

const resolveMessagePayload = (value) => {
  if (!value || typeof value !== "object") {
    return {};
  }

  if (value.payload && typeof value.payload === "object") {
    return value.payload;
  }

  return value;
};

function createBootstrapGate(shouldWaitForBootstrap) {
  if (!shouldWaitForBootstrap) {
    return {
      promise: Promise.resolve(true),
      settle: () => {},
    };
  }

  let settled = false;
  let resolvePromise = (_value) => {};
  const timerApi = getTimerApi();
  const promise = new Promise((resolve) => {
    resolvePromise = resolve;
  });

  /**
   * iframe 正式入口只允许等待有限时间。
   * 宿主未响应、origin/source 不匹配或 requestId 对不上时，最终都必须回落，
   * 避免路由守卫永久卡住。
   */
  const settle = (value) => {
    if (settled) {
      return;
    }

    settled = true;
    timerApi.clearTimeout(timeoutId);
    resolvePromise(value);
  };

  const timeoutId = timerApi.setTimeout(() => {
    settle(false);
  }, APP_BOOTSTRAP_TIMEOUT_MS);

  return {
    promise,
    settle,
  };
}

function createBootstrapRequestId() {
  return `datacenter_bootstrap_${Date.now().toString(36)}_${Math.random()
    .toString(36)
    .slice(2, 10)}`;
}

function resolveTrustedHostOrigin(referrer = "") {
  const referrerValue = asNonEmptyString(referrer);
  if (!referrerValue) {
    return null;
  }

  try {
    return new URL(referrerValue).origin;
  } catch {
    return null;
  }
}

function resolveOriginCandidate(value, baseUrl) {
  const candidate = asNonEmptyString(value);
  if (!candidate) {
    return null;
  }

  try {
    return new URL(candidate, baseUrl).origin;
  } catch {
    return null;
  }
}

export function shouldUseDebugMode(pathname) {
  const normalizedPath = pathname || "";
  return (
    normalizedPath === DEBUG_BASE_PATH ||
    normalizedPath.startsWith(`${DEBUG_BASE_PATH}/`)
  );
}

export function shouldRedirectTopLevelToIde(pathname, isTopLevel) {
  return Boolean(isTopLevel) && !shouldUseDebugMode(pathname);
}

export function buildIdeRestoreUrl(handoff, ideOrigin) {
  const targetUrl = new URL("/", ideOrigin);
  const handoffValue = resolveHandoffValue(handoff);
  if (handoffValue) {
    targetUrl.searchParams.set("handoff", handoffValue);
  }
  return targetUrl.toString();
}

export function buildIdeLoginUrl(currentUrl, ideOrigin) {
  const targetUrl = new URL("/login", ideOrigin);
  targetUrl.searchParams.set("redirect", currentUrl);
  return targetUrl.toString();
}

export function createBootstrapRequest({ handoff, requestId, url } = {}) {
  const resolvedUrl = resolveCurrentUrl(url);
  const handoffValue =
    resolveHandoffValue(handoff) ??
    asNonEmptyString(resolvedUrl.searchParams.get("handoff"));
  const requestIdValue =
    asNonEmptyString(requestId) ?? createBootstrapRequestId();

  return {
    type: APP_BOOTSTRAP_REQUEST,
    app: DATACENTER_APP,
    requestId: requestIdValue,
    handoff: handoffValue,
    requestedPath: resolvedUrl.pathname,
    requestedQuery: resolvedUrl.search,
  };
}

export function applyThemeToDocument(theme) {
  if (!["light", "dark"].includes(theme)) {
    return;
  }

  resolveDocumentLike()?.documentElement?.classList?.toggle(
    "dark",
    theme === "dark",
  );
}

export function applyBootstrapPayload(payload = {}) {
  const nextToken =
    asNonEmptyString(payload.token) ?? asNonEmptyString(payload.accessToken);
  const nextRefreshToken = asNonEmptyString(payload.refreshToken);
  const nextTenantId = asNonEmptyString(payload.tenantId);
  const nextProjectId =
    asNonEmptyString(payload.projectId) ?? asNonEmptyString(payload.pid);
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

  if (nextTheme) {
    Storage.setTheme(nextTheme);
    applyThemeToDocument(nextTheme);
  }

  if (nextLocale) {
    Storage.setLanguage(nextLocale);
  }
}

export function applyAuthRefreshedPayload(payload = {}) {
  const nextToken =
    asNonEmptyString(payload.token) ?? asNonEmptyString(payload.accessToken);
  const nextRefreshToken = asNonEmptyString(payload.refreshToken);

  if (!nextToken) {
    return;
  }

  Storage.setToken(nextToken);
  if (nextRefreshToken) {
    Storage.setRefreshToken(nextRefreshToken);
  }
}

export function resolveIdeOriginFromRuntime(options = {}) {
  const currentUrl = options.currentUrl ?? resolveWindowLike()?.location?.href;
  const configuredIdeOrigin = resolveOriginCandidate(
    options.configuredIdeOrigin ?? import.meta.env?.VITE_IDE_ORIGIN,
    currentUrl,
  );
  if (configuredIdeOrigin) {
    return configuredIdeOrigin;
  }

  const referrerOrigin = resolveTrustedHostOrigin(
    options.referrer ?? resolveDocumentLike()?.referrer ?? "",
  );
  if (referrerOrigin) {
    return referrerOrigin;
  }

  const currentLocation = currentUrl
    ? new URL(currentUrl)
    : resolveWindowLike()?.location;
  const currentProtocol =
    currentLocation?.protocol === "https:" ? "https:" : "http:";
  const currentHost = currentLocation?.hostname || "localhost";
  const currentOrigin = currentLocation?.origin || "";

  if (options.isDev ?? import.meta.env?.DEV) {
    const devHost =
      asNonEmptyString(options.devHost) ??
      asNonEmptyString(import.meta.env?.VITE_DEV_HOST) ??
      currentHost;
    const idePort = String(
      options.idePort ?? import.meta.env?.VITE_IDE_PORT ?? "18601",
    ).trim();

    if (devHost && idePort) {
      return `${currentProtocol}//${devHost}:${idePort}`;
    }
  }

  return currentOrigin;
}

export function resolveHostMessageTarget({
  parentWindow,
  referrer,
  selfWindow,
} = {}) {
  const currentWindow = selfWindow ?? resolveWindowLike();
  const currentParentWindow = parentWindow ?? currentWindow?.parent ?? null;

  if (!currentParentWindow || currentParentWindow === currentWindow) {
    return {
      origin: null,
      source: null,
    };
  }

  return {
    origin: resolveTrustedHostOrigin(
      referrer ?? resolveDocumentLike()?.referrer ?? "",
    ),
    source: currentParentWindow,
  };
}

export function initializeHostBootstrap({
  currentUrl,
  ideOrigin,
  isTopLevelWindow,
  parentWindow,
  referrer,
  selfWindow,
} = {}) {
  const resolvedUrl = resolveCurrentUrl(currentUrl);
  const topLevelWindow =
    isTopLevelWindow ??
    resolveWindowLike()?.parent === (selfWindow ?? resolveWindowLike());
  const trustedReferrer = referrer ?? resolveDocumentLike()?.referrer ?? "";
  const shouldRedirectToIde = shouldRedirectTopLevelToIde(
    resolvedUrl.pathname,
    topLevelWindow,
  );
  const shouldWaitForBootstrap =
    !shouldUseDebugMode(resolvedUrl.pathname) &&
    !shouldRedirectToIde &&
    !Storage.getToken();
  const trustedOrigin = resolveTrustedHostOrigin(trustedReferrer);
  const target = resolveHostMessageTarget({
    parentWindow,
    referrer: trustedReferrer,
    selfWindow,
  });
  const gate = createBootstrapGate(shouldWaitForBootstrap);
  const requestMessage = shouldWaitForBootstrap
    ? createBootstrapRequest({
        handoff: resolvedUrl.searchParams.get("handoff"),
        url: resolvedUrl.toString(),
      })
    : null;
  const restoreUrl = shouldRedirectToIde
    ? buildIdeRestoreUrl(
        resolvedUrl.searchParams.get("handoff"),
        ideOrigin ??
          resolveIdeOriginFromRuntime({
            currentUrl: resolvedUrl.toString(),
            referrer: trustedReferrer,
          }),
      )
    : null;

  currentSession = {
    gate,
    plan: {
      handoff: asNonEmptyString(resolvedUrl.searchParams.get("handoff")),
      ideRedirectUrl: restoreUrl,
      shouldRedirectToIde,
      shouldWaitForBootstrap,
      trustedHostOrigin: trustedOrigin,
    },
    requestMessage,
    target,
  };

  if (
    shouldWaitForBootstrap &&
    (!target.origin || !target.source || !requestMessage)
  ) {
    gate.settle(false);
  }

  if (shouldRedirectToIde || shouldUseDebugMode(resolvedUrl.pathname)) {
    gate.settle(true);
  }

  return currentSession.plan;
}

export function waitForHostBootstrap() {
  return currentSession?.gate.promise ?? Promise.resolve(true);
}

export function getTrustedHostOriginSet() {
  const origin = currentSession?.target?.origin ?? null;
  return origin ? new Set([origin]) : new Set();
}

export function getTrustedHostSources() {
  const source = currentSession?.target?.source ?? null;
  return source ? [source] : [];
}

export function isTrustedHostOrigin(
  origin,
  trustedOrigins = getTrustedHostOriginSet(),
) {
  if (!origin) {
    return false;
  }

  for (const trustedOrigin of trustedOrigins) {
    if (trustedOrigin === origin) {
      return true;
    }
  }

  return false;
}

export function isTrustedHostSource(
  source,
  trustedSources = getTrustedHostSources(),
) {
  if (!source) {
    return false;
  }

  for (const trustedSource of trustedSources) {
    if (trustedSource === source) {
      return true;
    }
  }

  return false;
}

export function isTrustedHostMessage(
  event,
  {
    trustedOrigins = getTrustedHostOriginSet(),
    trustedSources = getTrustedHostSources(),
  } = {},
) {
  return (
    isTrustedHostOrigin(event?.origin, trustedOrigins) &&
    isTrustedHostSource(event?.source, trustedSources)
  );
}

export function postMessageToHost(message) {
  const target = currentSession?.target ?? resolveHostMessageTarget();
  if (!target?.source?.postMessage || !target.origin) {
    return false;
  }

  target.source.postMessage(message, target.origin);
  return true;
}

export function postAppBootstrapRequest() {
  if (!currentSession?.requestMessage) {
    return false;
  }

  return postMessageToHost(currentSession.requestMessage);
}

export function handleBootstrapResponseMessage(data) {
  if (data?.type !== APP_BOOTSTRAP_RESPONSE) {
    return false;
  }

  const payload = resolveMessagePayload(data);
  const responseRequestId =
    asNonEmptyString(data.requestId) ?? asNonEmptyString(payload.requestId);
  const expectedRequestId = currentSession?.requestMessage?.requestId ?? null;

  if (
    responseRequestId &&
    expectedRequestId &&
    responseRequestId !== expectedRequestId
  ) {
    return false;
  }

  applyBootstrapPayload(payload);
  currentSession?.gate.settle(true);
  return true;
}

export function handleAuthRefreshedMessage(data) {
  if (data?.type !== AUTH_REFRESHED) {
    return false;
  }

  applyAuthRefreshedPayload(resolveMessagePayload(data));
  return true;
}

export function resetHostBootstrapSessionForTests() {
  currentSession = null;
}
