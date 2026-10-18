import { BACKEND_URL } from "@/lib/config";

/**
 * Shared authentication helpers for browser-side fetch.
 * Used by useGameStore, useWorldStore, and any future stores.
 */

/** Auth endpoints that must never trigger the session-expired handler. */
const AUTH_ENDPOINTS = new Set([
  "/api/auth/send-otp",
  "/api/auth/register",
  "/api/auth/login",
  "/api/auth/logout",
  "/api/auth/claim-tab",
  "/api/auth/google/login",
  "/api/auth/google/callback",
]);

let unauthorizedHandler: (() => void) | null = null;

/**
 * Register the callback invoked when an authenticated request returns 401.
 * Wired once by the game store. Kept as a callback instead of importing the
 * store here, which would create an import cycle.
 */
export const setUnauthorizedHandler = (fn: (() => void) | null): void => {
  unauthorizedHandler = fn;
};

/** Read a cookie value by name (browser-only). */
export const getCookie = (name: string): string | null => {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
};

const gameplayTabId = (): string => {
  if (typeof window === "undefined") return "";
  const key = "gu_gameplay_tab_id";
  let id = sessionStorage.getItem(key);
  if (!id) {
    id = crypto.randomUUID();
    sessionStorage.setItem(key, id);
  }
  if (!sessionStorage.getItem("gu_tab_replace_listener")) {
    sessionStorage.setItem("gu_tab_replace_listener", "1");
    window.addEventListener("storage", (event) => {
      if (event.key !== "gu_active_gameplay_tab_id" || !event.newValue || event.newValue === id) return;
      window.dispatchEvent(new CustomEvent("game:tab-replaced"));
      if (!sessionStorage.getItem("gu_tab_replaced_notice")) {
        sessionStorage.setItem("gu_tab_replaced_notice", "1");
        window.alert("Phiên chơi này đã được thay thế bởi tab mới. Hãy đóng tab này.");
      }
    });
  }
  return id;
};

let claimedTabSession = "";
const isAdminRequest = (input: RequestInfo | URL): boolean => {
  try {
    return new URL(String(input), window.location.origin).pathname.startsWith("/api/admin/");
  } catch {
    return false;
  }
};

// Pathname of a request, or "" when the input/base is not a parseable URL.
// Safari throws "The string did not match the expected pattern" from new URL
// when origin is opaque (WebView/iframe), so this must never be unguarded.
const safePathname = (input: RequestInfo | URL): string => {
  try {
    return new URL(String(input), window.location.origin).pathname;
  } catch {
    return "";
  }
};

const ensureGameplayTabClaim = async (token: string, tabId: string): Promise<void> => {
  const claimKey = `${token}:${tabId}`;
  if (claimedTabSession === claimKey) return;
  const response = await fetch(`${BACKEND_URL}/api/auth/claim-tab`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}`, "X-Player-Tab-ID": tabId },
    credentials: "include",
  });
  if (!response.ok) {
    const payload = await response.json().catch(() => null) as { error?: string } | null;
    throw new Error(payload?.error || "Không thể mở phiên chơi ở tab này.");
  }
  claimedTabSession = claimKey;
  localStorage.setItem("gu_active_gameplay_tab_id", tabId);
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
      const urlPath = safePathname(input);
      if (!isAdminRequest(input) && !AUTH_ENDPOINTS.has(urlPath)) {
        const tabId = gameplayTabId();
        await ensureGameplayTabClaim(token, tabId);
        headers["X-Player-Tab-ID"] = tabId;
      }
    }
	if (init.method === "DELETE" && String(input).includes("/api/admin/")) {
	  const target = decodeURIComponent(new URL(String(input), window.location.origin).pathname);
	  const reason = window.prompt(`Lý do xóa target “${target}”:`)?.trim();
	  if (!reason) throw new Error("Lý do xóa dữ liệu là bắt buộc");
	  headers["X-Admin-Reason"] = reason;
	  headers["X-Admin-Confirmation-Target"] = target;
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
  if (response.status === 409 && typeof window !== "undefined") {
    const payload = await response.clone().json().catch(() => null) as { error?: string } | null;
    if (payload?.error?.includes("bị thay thế bởi tab mới")) {
      if (!sessionStorage.getItem("gu_tab_replaced_notice")) {
        sessionStorage.setItem("gu_tab_replaced_notice", "1");
        window.alert("Phiên chơi này đã được thay thế bởi tab mới. Hãy đóng tab này.");
      }
    }
  }
  const requestURL = String(input);
  if (response.status === 403 && typeof window !== "undefined" && requestURL.includes("/api/admin/") && !requestURL.includes("/api/admin/me")) {
    const denied = await response.clone().json().catch(() => null) as { permission?: unknown } | null;
    if (typeof denied?.permission === "string") {
      // Rollback is deliberately synchronous and precedes the access refresh event.
      onForbiddenRollback?.();
      window.dispatchEvent(new CustomEvent("admin:permission-denied", { detail: { url: requestURL, permission: denied.permission } }));
    }
  }
  // A 401 means the session is gone. Handling it here covers every caller,
  // including the ones that never checked for 401 themselves — and stops short
  // of the auth endpoints, so logout() logging out cannot recurse.
  if (response.status === 401 && typeof window !== "undefined" && unauthorizedHandler) {
    let isAuthEndpoint = false;
    try {
      isAuthEndpoint = AUTH_ENDPOINTS.has(new URL(requestURL, window.location.origin).pathname);
    } catch {
      // Unparseable URL: treat it as a normal endpoint.
    }
    if (!isAuthEndpoint) unauthorizedHandler();
  }
  return response;
};
