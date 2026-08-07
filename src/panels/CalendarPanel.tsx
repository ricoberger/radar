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

const helperPath = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '..',
  '..',
  'bin',
  'calendar-helper',
);

export async function fetchCalendarEvents(): Promise<CalendarEvent[]> {
  const stdout = await run(helperPath, [], 30000);
  return JSON.parse(stdout) as CalendarEvent[];
}

function coversFullDay(event: CalendarEvent): boolean {
  if (event.isAllDay) return true;
  const dayStart = new Date();
  dayStart.setHours(0, 0, 0, 0);
  const dayEnd = new Date(dayStart);
  dayEnd.setDate(dayEnd.getDate() + 1);
  return (
    new Date(event.start).getTime() <= dayStart.getTime() &&
    new Date(event.end).getTime() >= dayEnd.getTime()
  );
}

function eventColor(event: CalendarEvent): {
  color?: string;
  dimColor?: boolean;
} {
  if (coversFullDay(event)) return {};
  const now = Date.now();
  const start = new Date(event.start).getTime();
  const end = new Date(event.end).getTime();
  if (end < now) return { dimColor: true };
  if (start <= now && now <= end) return { color: 'green' };
  return {};
}

export function CalendarPanel({
  id,
  index,
  title,
  focused,
  interval,
}: PanelProps) {
  const { data, error, loading, lastUpdated } = usePanelData(
    id,
    fetchCalendarEvents,
    interval,
  );
  const events = data ?? [];
  const selected = useListNavigation(id, events.length, () =>
    openExternal('ical://'),
  );

  return (
    <PanelFrame
      index={index}
      title={title}
      focused={focused}
      error={error}
      loading={loading}
      lastUpdated={lastUpdated}
    >
      {events.length === 0 && data ? (
        <Text dimColor>No events today</Text>
      ) : (
        <SelectList
          items={events}
          selected={focused ? selected : -1}
          renderItem={(event, isSelected) => (
            <Text wrap="truncate" bold={isSelected} {...eventColor(event)}>
              {isSelected ? '❯ ' : '  '}
              {coversFullDay(event)
                ? 'all-day       '
                : `${formatTime(new Date(event.start))} –${formatTime(new Date(event.end))} `}
              {event.title}
              <Text dimColor> · {event.calendar}</Text>
            </Text>
          )}
        />
      )}
    </PanelFrame>
  );
}
