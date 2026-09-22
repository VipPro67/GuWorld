export type RecipeSyncState<T> = {
  recipes?: T[];
  recipesLoaded?: boolean;
  recipesLoading?: boolean;
  recipesError?: string | null;
  recipesFetchedAt?: number | null;
};

type CachedRecipes<T> = { data: T[]; etag?: string; timestamp: number } | null;

export async function syncRecipes<T>({
  cacheKey,
  initialLoaded,
  readCache,
  writeCache,
  request,
  setState,
  now = Date.now,
}: {
  cacheKey: string;
  initialLoaded: boolean;
  readCache: (key: string) => Promise<CachedRecipes<T>>;
  writeCache: (key: string, data: T[], etag?: string) => Promise<void>;
  request: (etag?: string) => Promise<Response>;
  setState: (state: RecipeSyncState<T>) => void;
  now?: () => number;
}): Promise<void> {
  let cachedEtag: string | undefined;
  let hasValidRecipes = initialLoaded;

  try {
    const cached = await readCache(cacheKey);
    if (cached && Array.isArray(cached.data)) {
      cachedEtag = cached.etag;
      if (!hasValidRecipes) {
        hasValidRecipes = true;
        setState({ recipes: cached.data, recipesLoaded: true, recipesError: null, recipesFetchedAt: cached.timestamp });
      }
    }
  } catch (err) {
    console.warn("[Recipes] Failed to read from IndexedDB:", err);
  }

  setState({ recipesLoading: true, recipesError: null });
  try {
    const response = await request(cachedEtag);
    if (response.status === 304) {
      if (!hasValidRecipes) throw new Error("Recipe cache is unavailable for a 304 response");
      setState({ recipesLoaded: true, recipesLoading: false, recipesError: null, recipesFetchedAt: now() });
      return;
    }
    if (!response.ok) throw new Error(`Failed to load recipes (${response.status})`);

    const data: unknown = await response.json();
    if (!Array.isArray(data)) throw new Error("Invalid recipes response");
    const recipes = data as T[];
    const etag = response.headers.get("ETag") || undefined;
    const fetchedAt = now();
    setState({ recipes, recipesLoaded: true, recipesLoading: false, recipesError: null, recipesFetchedAt: fetchedAt });
    void writeCache(cacheKey, recipes, etag);
  } catch (caught: unknown) {
    const err = caught instanceof Error ? caught : new Error(String(caught));
    setState({ recipesLoaded: hasValidRecipes, recipesLoading: false, recipesError: err.message });
  }
}
