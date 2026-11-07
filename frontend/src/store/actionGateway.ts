import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

export interface GameActionOptions<TResponse> {
  endpoint: string;
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  headers?: Record<string, string>;
  actionLock?: string;
  successToast?: string | boolean | ((data: TResponse) => string | null);
  errorMessage?: string;
  logMessage?: string | boolean | ((data: TResponse) => string | null);
  skipReconciliation?: boolean;
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
  addLog?: (message: string) => void;
  applyInventoryFromResponse?: (data: unknown) => void;
  setPlayerFromServer?: (player: unknown) => boolean;
  error?: string | null;
  [key: string]: unknown;
}

/**
 * Executes a player action through a unified, deep action gateway:
 * - Checks active authentication token.
 * - Handles action concurrency locking (beginAction / endAction).
 * - Executes optimistic updates with automatic rollback on error.
 * - Centralizes HTTP call dispatch and JSON error message parsing.
 * - Automatically reconciles server player/inventory responses with local state.
 * - Emits toast notifications and activity logs for success and failures.
 * - Eliminates redundant 401 checks in slices (handled centrally in authFetch).
 */
export async function executeGameAction<TResponse = unknown>(
  get: () => ActionGatewayStore,
  set: (state: Partial<ActionGatewayStore>) => void,
  options: GameActionOptions<TResponse>
): Promise<GameActionResult<TResponse>> {
  const { token, beginAction, endAction, addToast, addLog, applyInventoryFromResponse, setPlayerFromServer } = get();
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

    if (data !== undefined) {
      const dataObj = typeof data === "object" && data !== null ? (data as Record<string, unknown>) : null;

      // 1. Automatic state reconciliation
      if (!options.skipReconciliation && dataObj) {
        if (
          typeof applyInventoryFromResponse === "function" &&
          (dataObj.player ||
            dataObj.gu_inventory ||
            dataObj.materials_inventory ||
            dataObj.current_mana !== undefined ||
            dataObj.primeval_stones !== undefined ||
            dataObj.wall_progress !== undefined)
        ) {
          applyInventoryFromResponse(data);
        } else if (typeof setPlayerFromServer === "function" && dataObj.player) {
          setPlayerFromServer(dataObj.player);
        }
      }

      // 2. Caller-specified custom onSuccess callback
      if (options.onSuccess) {
        await options.onSuccess(data);
      }

      const defaultMsg = typeof dataObj?.message === "string" ? dataObj.message : null;

      // 3. Activity log emission
      if (options.logMessage !== false && addLog) {
        const log = typeof options.logMessage === "function"
          ? options.logMessage(data)
          : typeof options.logMessage === "string"
            ? options.logMessage
            : defaultMsg;
        if (log) {
          addLog(log);
        }
      }

      // 4. Toast notification emission
      if (options.successToast !== false && addToast) {
        const toast = typeof options.successToast === "function"
          ? options.successToast(data)
          : typeof options.successToast === "string"
            ? options.successToast
            : defaultMsg;
        if (toast) {
          addToast(toast, "success");
        }
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
