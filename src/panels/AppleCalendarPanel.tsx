import { Text } from 'ink';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { PanelFrame } from '../components/PanelFrame.js';
import { SelectList } from '../components/SelectList.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { useListNavigation } from '../hooks/usePanelKeys.js';
import { PanelProps } from '../types.js';
import { formatTime, openExternal, run } from '../utils.js';

export interface CalendarEvent {
  title: string;
  calendar: string;
  start: string;
  end: string;
  isAllDay: boolean;
  location: string | null;
}

interface CalendarParams {
  day: 'yesterday' | 'today' | 'tomorrow';
  view: 'day' | 'week';
}

const dayOffsets = { yesterday: -1, today: 0, tomorrow: 1 } as const;

function readParams(params: Record<string, unknown>): CalendarParams {
  return {
    day: (params.day as CalendarParams['day']) ?? 'today',
    view: (params.view as CalendarParams['view']) ?? 'day',
  };
}

export function validateAppleCalendarParams(
  params: Record<string, unknown>,
  trail: string,
): void {
  if (params.day !== undefined && !(String(params.day) in dayOffsets)) {
    throw new Error(
      `${trail}: "day" must be one of ${Object.keys(dayOffsets).join(', ')}`,
    );
  }
  if (
    params.view !== undefined &&
    params.view !== 'day' &&
    params.view !== 'week'
  ) {
    throw new Error(`${trail}: "view" must be "day" or "week"`);
  }
}

export function appleCalendarTitle(params: Record<string, unknown>): string {
  const { day, view } = readParams(params);
  if (view === 'week') return 'Calendar · Week';
  if (day === 'yesterday') return 'Calendar · Yesterday';
  if (day === 'tomorrow') return 'Calendar · Tomorrow';
  return 'Calendar · Today';
}

const helperPath = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '..',
  '..',
  'bin',
  'apple-calendar-helper',
);

function startOfDay(date: Date): Date {
  const result = new Date(date);
  result.setHours(0, 0, 0, 0);
  return result;
}

function addDays(date: Date, days: number): Date {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}

function localDate(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

// Days shown for the configured view, computed at call time so a dashboard
// running past midnight stays current. Week = Monday-Sunday containing today.
function viewDays(params: CalendarParams): Date[] {
  const today = startOfDay(new Date());
  if (params.view === 'week') {
    const monday = addDays(today, -((today.getDay() + 6) % 7));
    return Array.from({ length: 7 }, (_, i) => addDays(monday, i));
  }
  return [addDays(today, dayOffsets[params.day])];
}

export async function fetchCalendarEvents(
  days: Date[],
): Promise<CalendarEvent[]> {
  const start = days[0];
  const end = addDays(days[days.length - 1], 1);
  const stdout = await run(
    helperPath,
    [localDate(start), localDate(end)],
    30000,
  );
  return JSON.parse(stdout) as CalendarEvent[];
}

type CalendarRow =
  | { kind: 'header'; day: Date }
  | { kind: 'spacer' }
  | { kind: 'event'; event: CalendarEvent; day: Date };

function overlapsDay(event: CalendarEvent, day: Date): boolean {
  const dayEnd = addDays(day, 1);
  return (
    new Date(event.start).getTime() < dayEnd.getTime() &&
    new Date(event.end).getTime() > day.getTime()
  );
}

// Multi-day events repeat under every day they cover; the label reflects the
// part of the event that falls on that day.
function buildRows(events: CalendarEvent[], days: Date[]): CalendarRow[] {
  if (days.length === 1) {
    return events
      .filter((event) => overlapsDay(event, days[0]))
      .map((event) => ({ kind: 'event', event, day: days[0] }));
  }
  return days.flatMap((day, i): CalendarRow[] => [
    ...(i > 0 ? [{ kind: 'spacer' } as const] : []),
    { kind: 'header', day },
    ...events
      .filter((event) => overlapsDay(event, day))
      .map((event) => ({ kind: 'event', event, day }) as const),
  ]);
}

function timeLabel(event: CalendarEvent, day: Date): string {
  const dayEnd = addDays(day, 1);
  const start = new Date(event.start);
  const end = new Date(event.end);
  if (
    event.isAllDay ||
    (start.getTime() <= day.getTime() && end.getTime() >= dayEnd.getTime())
  ) {
    return 'all-day';
  }
  const startsToday = start.getTime() >= day.getTime();
  const endsToday = end.getTime() <= dayEnd.getTime();
  if (startsToday && endsToday)
    return `${formatTime(start)} – ${formatTime(end)}`;
  if (startsToday) return formatTime(start);
  return `– ${formatTime(end)}`;
}

function eventColor(event: CalendarEvent, day: Date): { color?: string } {
  if (timeLabel(event, day) === 'all-day') return {};
  const now = Date.now();
  const start = new Date(event.start).getTime();
  const end = new Date(event.end).getTime();
  if (start <= now && now <= end) return { color: 'green' };
  return {};
}

function sameDay(a: Date, b: Date): boolean {
  return startOfDay(a).getTime() === startOfDay(b).getTime();
}

function dayHeading(day: Date): string {
  const weekday = day.toLocaleDateString('de-DE', { weekday: 'long' });
  const date = day.toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
  });
  return `${weekday} ${date}`;
}

export function AppleCalendarPanel({
  id,
  index,
  title,
  focused,
  interval,
  params,
}: PanelProps) {
  const calendarParams = readParams(params);
  const { data, error, loading, lastUpdated } = usePanelData(
    id,
    () => fetchCalendarEvents(viewDays(calendarParams)),
    interval,
  );
  const rows = buildRows(data ?? [], viewDays(calendarParams));
  const eventRowIndexes = rows.flatMap((row, i) =>
    row.kind === 'event' ? [i] : [],
  );
  const selected = useListNavigation(id, eventRowIndexes.length, () =>
    openExternal('ical://'),
  );
  const selectedRow =
    focused && eventRowIndexes.length > 0 ? eventRowIndexes[selected] : -1;

  return (
    <PanelFrame
      index={index}
      title={title}
      focused={focused}
      error={error}
      loading={loading}
      lastUpdated={lastUpdated}
    >
      {rows.length === 0 && data ? (
        <Text dimColor>No events {calendarParams.day}</Text>
      ) : (
        <SelectList
          items={rows}
          selected={selectedRow}
          renderItem={(row, isSelected) => {
            if (row.kind === 'spacer') return <Text> </Text>;
            if (row.kind === 'header') {
              return (
                <Text
                  bold
                  color={sameDay(row.day, new Date()) ? 'cyan' : undefined}
                  wrap="truncate"
                >
                  {dayHeading(row.day)}
                </Text>
              );
            }
            return (
              <Text
                wrap="truncate"
                bold={isSelected}
                {...eventColor(row.event, row.day)}
              >
                {isSelected ? '❯ ' : '  '}
                {timeLabel(row.event, row.day).padEnd(14)}
                {row.event.title}
                <Text dimColor> · {row.event.calendar}</Text>
              </Text>
            );
          }}
        />
      )}
    </PanelFrame>
  );
}
