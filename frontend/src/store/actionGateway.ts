import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface GameActionOptions<TResponse> {
  endpoint: string;
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  headers?: Record<string, string>;
  actionLock?: string;
  successToast?: string | ((data: TResponse) => string | null);
  errorMessage?: string;
  onSuccess?: (data: TResponse) => void | Promise<void>;
  optimisticUpdate?: () => void;
  rollback?: () => void;
}

export interface GameActionResult<TResponse> {
  ok: boolean;
  data?: TResponse;
  error?: string;
}

export interface ActionGatewayStore {
  token?: string | null;
  beginAction?: (key: string) => boolean;
  endAction?: (key: string) => void;
  addToast?: (message: string, type?: "success" | "error" | "info" | "warning") => void;
  error?: string | null;
  [key: string]: unknown;
}

/**
 * Executes a player action through a unified gateway:
 * - Checks active authentication token.
 * - Handles action concurrency locking (beginAction / endAction).
 * - Executes optimistic updates with automatic rollback on error.
 * - Centralizes HTTP call dispatch and JSON error message parsing.
 * - Emits toast notifications for failures and success messages.
 * - Eliminates redundant 401 checks in slices (handled centrally in authFetch).
 */
export async function executeGameAction<TResponse = unknown>(
  get: () => ActionGatewayStore,
  set: (state: Partial<ActionGatewayStore>) => void,
  options: GameActionOptions<TResponse>
): Promise<GameActionResult<TResponse>> {
  const { token, beginAction, endAction, addToast } = get();
  if (!token) {
    return { ok: false, error: "Chưa đăng nhập" };
  }

  if (options.actionLock && beginAction) {
    if (!beginAction(options.actionLock)) {
      return { ok: false, error: "Hành động đang được thực hiện" };
    }
  }

  set({ error: null });
  options.optimisticUpdate?.();

  try {
    const url = options.endpoint.startsWith("http")
      ? options.endpoint
      : `${BACKEND_URL}${options.endpoint.startsWith("/") ? "" : "/"}${options.endpoint}`;

    const headers: Record<string, string> = {
      ...(options.body ? { "Content-Type": "application/json" } : {}),
      ...options.headers,
    };

    const res = await authFetch(url, {
      method: options.method ?? (options.body ? "POST" : "GET"),
      headers,
      body: options.body ? JSON.stringify(options.body) : undefined,
    });

    if (!res.ok) {
      let errMsg = options.errorMessage || `Yêu cầu thất bại (${res.status})`;
      try {
        const errJson = (await res.json()) as { error?: string };
        if (errJson && typeof errJson === "object" && typeof errJson.error === "string") {
          errMsg = errJson.error;
        }
      } catch {
        // use default error message
      }

      options.rollback?.();
      set({ error: errMsg });
      addToast?.(errMsg, "error");
      return { ok: false, error: errMsg };
    }

    let data: TResponse | undefined;
    const contentType = res.headers.get("content-type") || "";
    if (contentType.includes("application/json")) {
      data = (await res.json()) as TResponse;
    }

    if (data !== undefined && options.onSuccess) {
      await options.onSuccess(data);
    }

    if (options.successToast && addToast) {
      const msg = typeof options.successToast === "function" ? options.successToast(data as TResponse) : options.successToast;
      if (msg) {
        addToast(msg, "success");
      }
    }

    return { ok: true, data };
  } catch (caught: unknown) {
    const err = caught instanceof Error ? caught : new Error(String(caught));
    options.rollback?.();
    const errMsg = err.message || options.errorMessage || "Lỗi kết nối máy chủ";
    set({ error: errMsg });
    addToast?.(errMsg, "error");
    return { ok: false, error: errMsg };
  } finally {
    if (options.actionLock && endAction) {
      endAction(options.actionLock);
    }
  }
}
