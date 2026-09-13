/**
 * Deduplicate concurrent fetches by key so multiple callers share one in-flight request.
 */
const pendingFetches = new Map<string, Promise<void>>();

export const fetchWithDedupe = async (
  key: string,
  fetcher: () => Promise<void>,
): Promise<void> => {
  const existing = pendingFetches.get(key);
  if (existing) return existing;
  const promise = fetcher().finally(() => pendingFetches.delete(key));
  pendingFetches.set(key, promise);
  return promise;
};
