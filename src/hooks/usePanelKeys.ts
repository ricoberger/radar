import { useEffect } from 'react';

import { clampSelection } from '../components/SelectList.js';
import { KeyInfo, keyRegistry, useSessionState } from '../store.js';

export function usePanelKeys(
  id: string,
  handler: (input: string, key: KeyInfo) => void,
): void {
  useEffect(() => {
    keyRegistry.set(id, handler);
    return () => {
      keyRegistry.delete(id);
    };
  });
}

// Standard j/k/g/G/enter navigation for list panels. Selection survives
// remounts via the session store. Unhandled keys fall through to onKey so
// panels can define extra panel-specific keymaps.
export function useListNavigation(
  id: string,
  length: number,
  onEnter: (index: number) => void,
  onKey?: (input: string, key: KeyInfo, selected: number) => void,
): number {
  const [selected, setSelected] = useSessionState(`${id}:selected`, 0);
  const clamped = clampSelection(selected, length);

  usePanelKeys(id, (input, key) => {
    if (length === 0) {
      onKey?.(input, key, -1);
      return;
    }
    const move = (delta: number) =>
      setSelected((prev) =>
        Math.max(0, Math.min(clampSelection(prev, length) + delta, length - 1)),
      );
    if (input === 'j' || key.downArrow) move(1);
    else if (input === 'k' || key.upArrow) move(-1);
    else if (input === 'g') setSelected(0);
    else if (input === 'G') setSelected(length - 1);
    else if (key.return) onEnter(clamped);
    else onKey?.(input, key, clamped);
  });

  return clamped;
}
