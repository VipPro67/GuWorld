interface CacheEntry<T = unknown> {
  data: T;
  timestamp: number;
}

const responseCache = new Map<string, CacheEntry<unknown>>();
const pendingFetches = new Map<string, Promise<unknown>>();

/**
 * Deduplicate concurrent fetches by key so multiple callers share one in-flight request.
 */
export const fetchWithDedupe = async (
  key: string,
  fetcher: () => Promise<void>,
): Promise<void> => {
  const existing = pendingFetches.get(key);
  if (existing) return existing as Promise<void>;
  const promise = fetcher().finally(() => pendingFetches.delete(key));
  pendingFetches.set(key, promise);
  return promise as Promise<void>;
};

/**
 * Fetch with TTL cache and in-flight deduplication.
 * If cached data is still fresh (< ttlMs), returns cached data without calling fetcher.
 * If forceRefresh is true, bypasses the cache and executes fetcher.
 */
export const fetchWithCache = async <T>(
  key: string,
  fetcher: () => Promise<T>,
  ttlMs: number = 10000,
  forceRefresh: boolean = false,
): Promise<T> => {
  const now = Date.now();
  if (!forceRefresh) {
    const cached = responseCache.get(key);
    if (cached && now - cached.timestamp < ttlMs) {
      return cached.data as T;
    }
  }

  const existing = pendingFetches.get(key);
  if (existing) return existing as Promise<T>;

  const promise = fetcher()
    .then((data) => {
      responseCache.set(key, { data, timestamp: Date.now() });
      return data;
    })
    .finally(() => pendingFetches.delete(key));

  pendingFetches.set(key, promise);
  return promise;
};

/**
 * Invalidate cache entries whose keys match or start with prefix.
 */
export const invalidateCache = (prefix?: string): void => {
  if (!prefix) {
    responseCache.clear();
    return;
  }
  for (const key of responseCache.keys()) {
    if (key.startsWith(prefix)) {
      responseCache.delete(key);
    }
  }
};

