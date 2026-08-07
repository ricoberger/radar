import { Box, Text } from 'ink';
import fs from 'node:fs';
import path from 'node:path';
import { useEffect, useRef, useState, type RefObject } from 'react';

import {
  PanelFrame,
  useContentHeight,
  useContentWidth,
} from '../components/PanelFrame.js';
import { useAppContext } from '../context.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { usePanelKeys } from '../hooks/usePanelKeys.js';
import { KeyInfo, useSessionState } from '../store.js';
import { PanelProps } from '../types.js';
import { expandHome, runWithInput } from '../utils.js';

type Day = 'today' | 'yesterday';

interface NotesParams {
  dir: string;
  day: Day;
}

export interface DailyNote {
  path: string;
  content: string | null;
}

function readParams(params: Record<string, unknown>): NotesParams {
  return {
    dir: expandHome(String(params.dir ?? '')),
    day: params.day === 'yesterday' ? 'yesterday' : 'today',
  };
}

export function validateRicobergerNotesParams(
  params: Record<string, unknown>,
  trail: string,
): void {
  if (typeof params.dir !== 'string' || params.dir.trim() === '') {
    throw new Error(
      `${trail}: "params.dir" is required for ricoberger-notes panels`,
    );
  }
  if (
    params.day !== undefined &&
    params.day !== 'today' &&
    params.day !== 'yesterday'
  ) {
    throw new Error(`${trail}: "params.day" must be "today" or "yesterday"`);
  }
}

export function ricobergerNotesTitle(params: Record<string, unknown>): string {
  return params.day === 'yesterday' ? 'Daily Note · Yesterday' : 'Daily Note';
}

function pad(value: number): string {
  return String(value).padStart(2, '0');
}

function noteDate(day: Day): Date {
  const date = new Date();
  if (day === 'yesterday') date.setDate(date.getDate() - 1);
  return date;
}

export function notePath(dir: string, date: Date): string {
  const year = String(date.getFullYear());
  const month = pad(date.getMonth() + 1);
  const day = pad(date.getDate());
  return path.join(dir, year, month, `${year}-${month}-${day}.md`);
}

function stripFrontmatter(content: string): string {
  return content.replace(/^---\n[\s\S]*?\n---\n+/, '');
}

export async function fetchDailyNote(params: NotesParams): Promise<DailyNote> {
  const p = notePath(params.dir, noteDate(params.day));
  if (!fs.existsSync(p)) {
    return { path: p, content: null };
  }
  return { path: p, content: stripFrontmatter(fs.readFileSync(p, 'utf8')) };
}

// Creates today's note from template.md (if present) and returns its path.
export function ensureTodayNote(dir: string): string {
  const target = notePath(dir, new Date());
  if (fs.existsSync(target)) {
    return target;
  }

  const today = path.basename(target, '.md');
  const templatePath = path.join(dir, 'template.md');
  const template = fs.existsSync(templatePath)
    ? fs.readFileSync(templatePath, 'utf8').replace(/yyyy-mm-dd/g, today)
    : `# ${today}\n`;

  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, template);
  return target;
}

const ANSI_RE = /\x1b\[[0-9;]*m/g;

function trimBlankEdges(lines: string[]): string[] {
  const isBlank = (line: string) => line.replace(ANSI_RE, '').trim() === '';
  let start = 0;
  let end = lines.length;
  while (start < end && isBlank(lines[start])) start++;
  while (end > start && isBlank(lines[end - 1])) end--;
  return lines.slice(start, end);
}

type ScrollHandler = (input: string, key: KeyInfo) => void;

function NoteContent({
  id,
  content,
  scrollRef,
}: {
  id: string;
  content: string;
  scrollRef: RefObject<ScrollHandler | null>;
}) {
  const height = useContentHeight();
  const width = useContentWidth();
  const [lines, setLines] = useState<string[] | null>(null);
  const [offset, setOffset] = useSessionState(`${id}:offset`, 0);

  // Render through the `md` binary (glamour); fall back to plain text when
  // it is not installed or fails.
  useEffect(() => {
    if (width <= 0) return;
    let cancelled = false;
    runWithInput('md', [], content, { MD_WORD_WRAP: String(width) })
      .then((out) => {
        if (!cancelled) setLines(trimBlankEdges(out.split('\n')));
      })
      .catch(() => {
        if (!cancelled) setLines(content.replace(/\n$/, '').split('\n'));
      });
    return () => {
      cancelled = true;
    };
  }, [content, width]);

  const total = lines?.length ?? 0;
  const maxOffset = Math.max(0, total - Math.max(1, height));
  const clamped = Math.min(offset, maxOffset);
  const visible = (lines ?? []).slice(clamped, clamped + Math.max(1, height));

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
          <Text wrap="truncate">{line}</Text>
        </Box>
      ))}
    </Box>
  );
}

export function RicobergerNotesPanel({
  id,
  index,
  title,
  focused,
  interval,
  params,
}: PanelProps) {
  const { config, runExternal } = useAppContext();
  const { dir, day } = readParams(params);
  const scrollRef = useRef<ScrollHandler | null>(null);
  const { data, error, loading, lastUpdated, refresh } = usePanelData(
    id,
    () => fetchDailyNote({ dir, day }),
    interval,
  );

  // Enter opens the note in the editor; today's note is created on demand,
  // but a missing note for yesterday is never created retroactively.
  const openEditor = () => {
    let target: string | null = null;
    if (day === 'today') target = ensureTodayNote(dir);
    else if (data?.content != null) target = data.path;
    if (!target) return;
    const [command, ...args] = config.editor.split(/\s+/);
    runExternal(command, [...args, target]);
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
        <Text dimColor>
          {day === 'today'
            ? 'No note for today — press enter to create it'
            : 'No note for yesterday'}
        </Text>
      ) : data?.content ? (
        <NoteContent id={id} content={data.content} scrollRef={scrollRef} />
      ) : null}
    </PanelFrame>
  );
}
