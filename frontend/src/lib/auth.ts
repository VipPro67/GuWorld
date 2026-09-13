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

// Track latest request timestamp per URL to reject stale responses.
const pendingTimestamps = new Map<string, number>();

const getAbortSignal = (key: string, timeoutMs = 30000): AbortSignal => {
  const controller = new AbortController();
  const id = Date.now();
  pendingTimestamps.set(key, id);
  setTimeout(() => controller.abort(), timeoutMs);
  const originalAddEventListener = controller.signal.addEventListener.bind(
    controller.signal,
  );
  controller.signal.addEventListener = (
    type: string,
    listener: EventListenerOrEventListenerObject,
    options?: AddEventListenerOptions | boolean,
  ) => {
    if (type === "abort") {
      originalAddEventListener("abort", () => {
        if (pendingTimestamps.get(key) === id) pendingTimestamps.delete(key);
      });
    }
    originalAddEventListener(type, listener, options);
  };
  return controller.signal;
};

/**
 * Fetch wrapper that injects Bearer token, CSRF header, abort-signal timeout,
 * and `credentials: "include"`. Drop-in replacement for `fetch()`.
 */
export const authFetch = (
  input: RequestInfo | URL,
  init: RequestInit & { timeoutMs?: number } = {},
) => {
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
  const key = typeof input === "string" ? input : input.toString();
  const signal = getAbortSignal(key, init.timeoutMs ?? 30000);
  return fetch(input, {
    ...init,
    headers,
    credentials: "include",
    signal,
  });
};
