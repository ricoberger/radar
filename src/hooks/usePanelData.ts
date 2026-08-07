import { useCallback, useEffect, useRef, useState } from 'react';

import { getCache, refreshRegistry, setCache } from '../store.js';

export interface PanelData<T> {
  data: T | undefined;
  error: string | undefined;
  loading: boolean;
  lastUpdated: number | undefined;
  refresh: () => void;
}

// Fetches on an interval, keeps stale data on errors and survives remounts of
// the Ink tree through the module-level cache. Registers the refresh callback
// so the app can route the r/R keys.
export function usePanelData<T>(
  id: string,
  fetcher: () => Promise<T>,
  intervalSeconds: number,
): PanelData<T> {
  const cached = getCache<T>(id);
  const [data, setData] = useState<T | undefined>(cached.data);
  const [error, setError] = useState<string | undefined>(cached.error);
  const [lastUpdated, setLastUpdated] = useState<number | undefined>(
    cached.lastUpdated,
  );
  const [loading, setLoading] = useState(false);
  const inFlight = useRef(false);
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;

  const load = useCallback(async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    setLoading(true);
    try {
      const result = await fetcherRef.current();
      const entry = { data: result, error: undefined, lastUpdated: Date.now() };
      setCache(id, entry);
      setData(result);
      setError(undefined);
      setLastUpdated(entry.lastUpdated);
    } catch (err) {
      const message = (err as Error).message;
      const previous = getCache<T>(id);
      setCache(id, { ...previous, error: message });
      setError(message);
    } finally {
      inFlight.current = false;
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    const intervalMs = intervalSeconds * 1000;
    const age = Date.now() - (getCache<T>(id).lastUpdated ?? 0);
    const initialDelay = Math.max(0, intervalMs - age);

    let interval: ReturnType<typeof setInterval> | undefined;
    const timeout = setTimeout(() => {
      void load();
      interval = setInterval(() => void load(), intervalMs);
    }, initialDelay);

    refreshRegistry.set(id, () => void load());
    return () => {
      clearTimeout(timeout);
      if (interval) clearInterval(interval);
      refreshRegistry.delete(id);
    };
  }, [id, intervalSeconds, load]);

  return { data, error, loading, lastUpdated, refresh: () => void load() };
}
