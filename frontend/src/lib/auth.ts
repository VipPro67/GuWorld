/**
 * Shared authentication helpers for browser-side fetch.
 * Used by useGameStore, useWorldStore, and any future stores.
 */

/** Read a cookie value by name (browser-only). */
export const getCookie = (name: string): string | null => {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
};

/** Build Authorization + CSRF headers from localStorage / cookies. */
export const authHeaders = (method?: string): Record<string, string> => {
  const headers: Record<string, string> = {};
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("gu_auth_token");
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }
  const csrfToken = getCookie("gu_csrf_token");
  if (csrfToken && method && method !== "GET" && method !== "HEAD") {
    headers["X-CSRF-Token"] = csrfToken;
  }
  return headers;
};

/**
 * Fetch wrapper that injects Bearer token, CSRF header, abort-signal timeout,
 * and `credentials: "include"`. Drop-in replacement for `fetch()`.
 */
export const authFetch = async (
  input: RequestInfo | URL,
  init: RequestInit & { timeoutMs?: number; onForbiddenRollback?: () => void } = {},
): Promise<Response> => {
  const csrfToken = getCookie("gu_csrf_token");
  const headers: Record<string, string> = {};
  if (init.headers) {
    if (Array.isArray(init.headers)) {
      for (const [k, v] of init.headers) headers[k] = v;
    } else if (init.headers instanceof Headers) {
      init.headers.forEach((v, k) => {
        headers[k] = v;
      });
    } else {
      Object.assign(headers, init.headers);
    }
  }
  if (csrfToken && init.method && init.method !== "GET" && init.method !== "HEAD") {
    headers["X-CSRF-Token"] = csrfToken;
  }
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("gu_auth_token");
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }
  const { timeoutMs = 30000, signal: callerSignal, onForbiddenRollback, ...fetchInit } = init;
  const timeoutSignal = AbortSignal.timeout(timeoutMs);
  const signal = callerSignal
    ? AbortSignal.any([callerSignal, timeoutSignal])
    : timeoutSignal;

  const response = await fetch(input, {
    ...fetchInit,
    headers,
    credentials: "include",
    signal,
  });
  const requestURL = String(input);
  if (response.status === 403 && typeof window !== "undefined" && requestURL.includes("/api/admin/") && !requestURL.includes("/api/admin/me")) {
    const denied = await response.clone().json().catch(() => null) as { permission?: unknown } | null;
    if (typeof denied?.permission === "string") {
      // Rollback is deliberately synchronous and precedes the access refresh event.
      onForbiddenRollback?.();
      window.dispatchEvent(new CustomEvent("admin:permission-denied", { detail: { url: requestURL, permission: denied.permission } }));
    }
  }
  return response;
};
