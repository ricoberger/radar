import { Box, Text } from 'ink';
import fs from 'node:fs';
import path from 'node:path';
import { useRef, type RefObject } from 'react';

import { MarkdownLine } from '../components/MarkdownLine.js';
import { PanelFrame, useContentHeight } from '../components/PanelFrame.js';
import { useAppContext } from '../context.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { usePanelKeys } from '../hooks/usePanelKeys.js';
import { KeyInfo, useSessionState } from '../store.js';
import { PanelProps } from '../types.js';

export interface DailyNote {
  path: string;
  content: string | null;
}

function pad(value: number): string {
  return String(value).padStart(2, '0');
}

export function todayNotePath(
  dailyNotesDir: string,
  date = new Date(),
): string {
  const year = String(date.getFullYear());
  const month = pad(date.getMonth() + 1);
  const day = pad(date.getDate());
  return path.join(dailyNotesDir, year, month, `${year}-${month}-${day}.md`);
}

export async function fetchDailyNote(
  dailyNotesDir: string,
): Promise<DailyNote> {
  const notePath = todayNotePath(dailyNotesDir);
  if (!fs.existsSync(notePath)) {
    return { path: notePath, content: null };
  }
  return { path: notePath, content: fs.readFileSync(notePath, 'utf8') };
}

// Creates today's note from template.md (if present) and returns its path.
export function ensureTodayNote(dailyNotesDir: string): string {
  const notePath = todayNotePath(dailyNotesDir);
  if (fs.existsSync(notePath)) {
    return notePath;
  }

  const today = todayNotePath(dailyNotesDir).slice(-13, -3);
  const templatePath = path.join(dailyNotesDir, 'template.md');
  const template = fs.existsSync(templatePath)
    ? fs.readFileSync(templatePath, 'utf8').replace(/yyyy-mm-dd/g, today)
    : `# ${today}\n`;

  fs.mkdirSync(path.dirname(notePath), { recursive: true });
  fs.writeFileSync(notePath, template);
  return notePath;
}

type ScrollHandler = (input: string, key: KeyInfo) => void;

function NoteContent({
  id,
  lines,
  scrollRef,
}: {
  id: string;
  lines: string[];
  scrollRef: RefObject<ScrollHandler | null>;
}) {
  const height = useContentHeight();
  const [offset, setOffset] = useSessionState(`${id}:offset`, 0);
  const maxOffset = Math.max(0, lines.length - Math.max(1, height));
  const clamped = Math.min(offset, maxOffset);
  const visible = lines.slice(clamped, clamped + Math.max(1, height));

  scrollRef.current = (input, key) => {
    const move = (delta: number) =>
      setOffset((prev) => Math.max(0, Math.min(prev + delta, maxOffset)));
    if (input === 'j' || key.downArrow) move(1);
    else if (input === 'k' || key.upArrow) move(-1);
    else if (input === 'g') setOffset(0);
    else if (input === 'G') setOffset(maxOffset);
  };

  return (
    <Box flexDirection="column">
      {visible.map((line, i) => (
        <Box key={clamped + i} flexShrink={0}>
          <MarkdownLine line={line} />
        </Box>
      ))}
    </Box>
  );
}

export function NotePanel({ id, index, title, focused, interval }: PanelProps) {
  const { config, runExternal } = useAppContext();
  const scrollRef = useRef<ScrollHandler | null>(null);
  const { data, error, loading, lastUpdated, refresh } = usePanelData(
    id,
    () => fetchDailyNote(config.dailyNotesDir),
    interval,
  );

  const openEditor = () => {
    const notePath = ensureTodayNote(config.dailyNotesDir);
    const [command, ...args] = config.editor.split(/\s+/);
    runExternal(command, [...args, notePath]);
    refresh();
  };

  usePanelKeys(id, (input, key) => {
    if (key.return) {
      openEditor();
      return;
    }
    scrollRef.current?.(input, key);
  });

  return (
    <PanelFrame
      index={index}
      title={title}
      focused={focused}
      error={error}
      loading={loading}
      lastUpdated={lastUpdated}
    >
      {data && data.content === null ? (
        <Text dimColor>No note for today — press enter to create it</Text>
      ) : data?.content ? (
        <NoteContent
          id={id}
          lines={data.content.replace(/\n$/, '').split('\n')}
          scrollRef={scrollRef}
        />
      ) : null}
    </PanelFrame>
  );
}
