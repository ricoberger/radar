import { Box, DOMElement, measureElement, Text } from 'ink';
import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react';

import { formatAge } from '../utils.js';

const ContentHeightContext = createContext(0);

export function useContentHeight(): number {
  return useContext(ContentHeightContext);
}

interface PanelFrameProps {
  index: number;
  title: string;
  focused: boolean;
  error?: string;
  loading?: boolean;
  lastUpdated?: number;
  children: ReactNode;
}

// Panel chrome: border, header line with number + title + status, and a
// content area whose measured height is provided to children via context.
export function PanelFrame({
  index,
  title,
  focused,
  error,
  loading,
  lastUpdated,
  children,
}: PanelFrameProps) {
  const contentRef = useRef<DOMElement>(null);
  const [height, setHeight] = useState(0);

  useEffect(() => {
    if (contentRef.current) {
      const measured = measureElement(contentRef.current).height;
      if (measured !== height) setHeight(measured);
    }
  });

  return (
    <Box
      flexDirection="column"
      flexGrow={1}
      borderStyle="round"
      borderColor={focused ? 'cyan' : 'gray'}
      paddingX={1}
      overflow="hidden"
    >
      <Box flexShrink={0} gap={1}>
        <Text bold color={focused ? 'cyan' : undefined} wrap="truncate">
          {index} {title}
        </Text>
        <Box flexGrow={1} />
        {loading && !lastUpdated ? <Text dimColor>…</Text> : null}
        {error ? (
          <Text color="red" wrap="truncate">
            ! {error}
          </Text>
        ) : lastUpdated ? (
          <Text dimColor>{formatAge(new Date(lastUpdated))}</Text>
        ) : null}
      </Box>
      <Box
        ref={contentRef}
        flexDirection="column"
        flexGrow={1}
        overflow="hidden"
      >
        <ContentHeightContext.Provider value={height}>
          {children}
        </ContentHeightContext.Provider>
      </Box>
    </Box>
  );
}
