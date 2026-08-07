import { Box } from 'ink';
import type { ReactNode } from 'react';

import { useContentHeight } from './PanelFrame.js';

interface SelectListProps<T> {
  items: T[];
  selected: number;
  renderItem: (item: T, selected: boolean) => ReactNode;
}

// Renders a window of items around the selection, sized by the measured
// panel content height.
export function SelectList<T>({
  items,
  selected,
  renderItem,
}: SelectListProps<T>) {
  const height = useContentHeight();
  const visible = Math.max(1, height);

  let start = 0;
  if (items.length > visible) {
    start = Math.min(
      Math.max(0, selected - Math.floor(visible / 2)),
      items.length - visible,
    );
  }
  const window = items.slice(start, start + visible);

  return (
    <Box flexDirection="column">
      {window.map((item, i) => (
        <Box key={start + i} flexShrink={0}>
          {renderItem(item, start + i === selected)}
        </Box>
      ))}
    </Box>
  );
}

export function clampSelection(selected: number, length: number): number {
  return Math.max(0, Math.min(selected, length - 1));
}
