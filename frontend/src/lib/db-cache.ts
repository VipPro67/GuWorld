/**
 * Client-side persistent cache using IndexedDB for Game Dictionaries.
 * Provides fast 0ms startup hydration and low memory overhead.
 */

const DB_NAME = "GuWorld_Cache_v1";
const DB_VERSION = 1;
const STORE_NAME = "dictionaries";

interface CacheEntry<T> {
  key: string;
  data: T;
  etag?: string;
  timestamp: number;
}

function openDB(): Promise<IDBDatabase | null> {
  if (typeof window === "undefined" || !window.indexedDB) {
    return Promise.resolve(null);
  }

  return new Promise((resolve) => {
    try {
      const request = window.indexedDB.open(DB_NAME, DB_VERSION);

      request.onupgradeneeded = (event) => {
        const db = (event.target as IDBOpenDBRequest).result;
        if (!db.objectStoreNames.contains(STORE_NAME)) {
          db.createObjectStore(STORE_NAME, { keyPath: "key" });
        }
      };

      request.onsuccess = () => {
        resolve(request.result);
      };

      request.onerror = (err) => {
        console.warn("[IndexedDB] Failed to open database:", err);
        resolve(null);
      };
    } catch (e) {
      console.warn("[IndexedDB] Exception opening database:", e);
      resolve(null);
    }
  });
}

export async function getDictionaryCache<T>(
  key: string
): Promise<{ data: T; etag?: string; timestamp: number } | null> {
  const db = await openDB();
  if (!db) return null;

  return new Promise((resolve) => {
    try {
      const transaction = db.transaction(STORE_NAME, "readonly");
      const store = transaction.objectStore(STORE_NAME);
      const request = store.get(key);

      request.onsuccess = () => {
        const result = request.result as CacheEntry<T> | undefined;
        if (result && result.data) {
          resolve({
            data: result.data,
            etag: result.etag,
            timestamp: result.timestamp,
          });
        } else {
          resolve(null);
        }
      };

      request.onerror = () => {
        resolve(null);
      };
    } catch (err) {
      console.warn(`[IndexedDB] Failed to read ${key}:`, err);
      resolve(null);
    }
  });
}

export async function setDictionaryCache<T>(
  key: string,
  data: T,
  etag?: string
): Promise<void> {
  const db = await openDB();
  if (!db) return;

  return new Promise((resolve) => {
    try {
      const transaction = db.transaction(STORE_NAME, "readwrite");
      const store = transaction.objectStore(STORE_NAME);
      const entry: CacheEntry<T> = {
        key,
        data,
        etag,
        timestamp: Date.now(),
      };
      store.put(entry);

      transaction.oncomplete = () => {
        resolve();
      };

      transaction.onerror = (err) => {
        console.warn(`[IndexedDB] Failed to write ${key}:`, err);
        resolve();
      };
    } catch (err) {
      console.warn(`[IndexedDB] Exception writing ${key}:`, err);
      resolve();
    }
  });
}

export async function clearDictionaryCache(key?: string): Promise<void> {
  const db = await openDB();
  if (!db) return;

  return new Promise((resolve) => {
    try {
      const transaction = db.transaction(STORE_NAME, "readwrite");
      const store = transaction.objectStore(STORE_NAME);
      if (key) {
        store.delete(key);
      } else {
        store.clear();
      }

      transaction.oncomplete = () => {
        resolve();
      };

      transaction.onerror = () => {
        resolve();
      };
    } catch (err) {
      console.warn("[IndexedDB] Exception clearing cache:", err);
      resolve();
    }
  });
}
