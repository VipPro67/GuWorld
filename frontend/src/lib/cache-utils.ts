/**
 * Utility functions for clearing client-side persistent storage and caches.
 */
import { clearDictionaryCache } from "./db-cache";

export interface ClearCacheOptions {
  preserveAuth?: boolean;
  reloadAfterClear?: boolean;
}

const DB_NAME = "GuWorld_Cache_v1";

/**
 * Clears all client-side cached data (IndexedDB, Cache Storage, Service Workers, LocalStorage, SessionStorage).
 */
export async function clearLocalGameCache(options: ClearCacheOptions = { preserveAuth: true, reloadAfterClear: true }): Promise<void> {
  const { preserveAuth = true, reloadAfterClear = true } = options;

  // 1. Clear IndexedDB
  try {
    await clearDictionaryCache();
    if (typeof window !== "undefined" && window.indexedDB) {
      try {
        window.indexedDB.deleteDatabase(DB_NAME);
      } catch (err) {
        console.warn("[Cache] Error deleting IndexedDB database:", err);
      }
    }
  } catch (err) {
    console.warn("[Cache] Error clearing IndexedDB cache:", err);
  }

  // 2. Clear browser Cache Storage API (HTTP caches, service worker assets)
  try {
    if (typeof window !== "undefined" && "caches" in window) {
      const cacheNames = await window.caches.keys();
      await Promise.all(cacheNames.map((name) => window.caches.delete(name)));
    }
  } catch (err) {
    console.warn("[Cache] Error clearing window.caches:", err);
  }

  // 3. Unregister service workers if any
  try {
    if (typeof window !== "undefined" && "serviceWorker" in navigator) {
      const registrations = await navigator.serviceWorker.getRegistrations();
      for (const reg of registrations) {
        await reg.unregister();
      }
    }
  } catch (err) {
    console.warn("[Cache] Error unregistering service workers:", err);
  }

  // 4. Clear sessionStorage
  try {
    if (typeof window !== "undefined" && window.sessionStorage) {
      window.sessionStorage.clear();
    }
  } catch (err) {
    console.warn("[Cache] Error clearing sessionStorage:", err);
  }

  // 5. Clear localStorage
  try {
    if (typeof window !== "undefined" && window.localStorage) {
      if (preserveAuth) {
        const authToken = localStorage.getItem("gu_auth_token");
        const playerId = localStorage.getItem("gu_player_id");
        localStorage.clear();
        if (authToken) {
          localStorage.setItem("gu_auth_token", authToken);
        }
        if (playerId) {
          localStorage.setItem("gu_player_id", playerId);
        }
      } else {
        localStorage.clear();
      }
    }
  } catch (err) {
    console.warn("[Cache] Error clearing localStorage:", err);
  }

  // 6. Hard reload the page to refresh all in-memory runtime states
  if (reloadAfterClear && typeof window !== "undefined") {
    window.location.reload();
  }
}
