import { useState } from 'react';

// Module-level state that survives remounts of the Ink tree (the app is
// unmounted and remounted around external editor sessions).

const store = new Map<string, unknown>();

export function useSessionState<T>(
  key: string,
  initial: T,
): [T, (value: T | ((prev: T) => T)) => void] {
  const [value, setValue] = useState<T>(() =>
    store.has(key) ? (store.get(key) as T) : initial,
  );
  const set = (next: T | ((prev: T) => T)) => {
    const prev = store.has(key) ? (store.get(key) as T) : initial;
    const resolved =
      typeof next === 'function' ? (next as (prev: T) => T)(prev) : next;
    store.set(key, resolved);
    setValue(resolved);
  };
  return [value, set];
}

export interface CacheEntry<T> {
  data?: T;
  error?: string;
  lastUpdated?: number;
}

const dataCache = new Map<string, CacheEntry<unknown>>();

export function getCache<T>(id: string): CacheEntry<T> {
  return (dataCache.get(id) as CacheEntry<T>) ?? {};
}

export function setCache<T>(id: string, entry: CacheEntry<T>): void {
  dataCache.set(id, entry);
}

// Registries used to route global keys (r/R, j/k/enter) to the focused panel.

export const refreshRegistry = new Map<string, () => void>();
export const keyRegistry = new Map<
  string,
  (input: string, key: KeyInfo) => void
>();

export interface KeyInfo {
  upArrow: boolean;
  downArrow: boolean;
  return: boolean;
  ctrl: boolean;
  shift: boolean;
  tab: boolean;
}
