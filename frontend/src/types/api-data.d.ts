/**
 * JSON-compatible data returned by legacy backend endpoints that do not yet
 * expose a stable response contract. Prefer a concrete response interface for
 * endpoints with a defined payload.
 */
type ApiResponseData = ReturnType<typeof JSON.parse>;
