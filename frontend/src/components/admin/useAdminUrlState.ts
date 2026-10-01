"use client";

import { useCallback, useEffect, useState } from "react";

type Values = Record<string, string>;

/** Keeps controlled admin table state bookmarkable without coupling dictionaries to server queries. */
export function useAdminUrlState(defaults: Values, keys: readonly string[]) {
  const read = useCallback(() => {
    if (typeof window === "undefined") return defaults;
    const params = new URLSearchParams(window.location.search);
    return Object.fromEntries(keys.map((key) => [key, params.get(key) ?? defaults[key] ?? ""]));
  }, [defaults, keys]);
  const [values, setValues] = useState<Values>(defaults);

  useEffect(() => {
    const sync = () => setValues(read());
    sync();
    window.addEventListener("popstate", sync);
    return () => window.removeEventListener("popstate", sync);
  }, [read]);

  const update = useCallback((patch: Values, replace = false) => {
    const params = new URLSearchParams(window.location.search);
    Object.entries(patch).forEach(([key, value]) => {
      if (!value || value === defaults[key]) params.delete(key);
      else params.set(key, value);
    });
    const url = `${window.location.pathname}${params.size ? `?${params}` : ""}${window.location.hash}`;
    window.history[replace ? "replaceState" : "pushState"](null, "", url);
    setValues((current) => ({ ...current, ...patch }));
  }, [defaults]);

  return [values, update] as const;
}
