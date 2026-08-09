import { Box, DOMElement, measureElement, Text } from 'ink';
import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react';

import { MAUVE } from '../theme.js';
import { formatAge } from '../utils.js';

const ContentHeightContext = createContext(0);
const ContentWidthContext = createContext(0);

// Re-renders on a 1s tick so the age keeps counting between fetches; setting
// an unchanged string bails out of the re-render, so once the age is in the
// minutes range the tick becomes effectively free.
function Age({ lastUpdated }: { lastUpdated: number }) {
  const [age, setAge] = useState(() => formatAge(new Date(lastUpdated)));
  useEffect(() => {
    const update = () => setAge(formatAge(new Date(lastUpdated)));
    update();
    const timer = setInterval(update, 1000);
    return () => clearInterval(timer);
  }, [lastUpdated]);
  return <Text dimColor>{age}</Text>;
}

export function useContentHeight(): number {
  return useContext(ContentHeightContext);
}

export function useContentWidth(): number {
  return useContext(ContentWidthContext);
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
  const [size, setSize] = useState({ width: 0, height: 0 });

  useEffect(() => {
    if (contentRef.current) {
      const measured = measureElement(contentRef.current);
      if (measured.width !== size.width || measured.height !== size.height) {
        setSize({ width: measured.width, height: measured.height });
      }
    }
  });

  return (
    <Box
      flexDirection="column"
      flexGrow={1}
      borderStyle="round"
      borderColor={focused ? MAUVE : 'gray'}
      paddingX={1}
      overflow="hidden"
    >
      <Box
        flexShrink={0}
        gap={1}
        borderStyle="single"
        borderTop={false}
        borderLeft={false}
        borderRight={false}
        borderColor={focused ? MAUVE : 'gray'}
      >
        <Text bold color={focused ? MAUVE : undefined} wrap="truncate">
          {index} {title}
        </Text>
        <Box flexGrow={1} />
        {loading && !lastUpdated ? <Text dimColor>…</Text> : null}
        {error ? (
          <Text color="red" wrap="truncate">
            ! {error}
          </Text>
        ) : lastUpdated ? (
          <Age lastUpdated={lastUpdated} />
        ) : null}
      </Box>
      <Box
        ref={contentRef}
        flexDirection="column"
        flexGrow={1}
        overflow="hidden"
      >
        <ContentHeightContext.Provider value={size.height}>
          <ContentWidthContext.Provider value={size.width}>
            {children}
          </ContentWidthContext.Provider>
        </ContentHeightContext.Provider>
      </Box>
    </Box>
  );
}
