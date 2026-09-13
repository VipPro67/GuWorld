/** Backend base URL for client-side fetch calls. */
export const BACKEND_URL =
  process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

/** Backend base URL for SSR / server-side fetch (falls back to public). */
export const INTERNAL_BACKEND_URL =
  process.env.INTERNAL_BACKEND_URL || BACKEND_URL;
