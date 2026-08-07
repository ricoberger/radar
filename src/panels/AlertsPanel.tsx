import { Text } from 'ink';

import { PanelFrame } from '../components/PanelFrame.js';
import { SelectList } from '../components/SelectList.js';
import { useAppContext } from '../context.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { useListNavigation } from '../hooks/usePanelKeys.js';
import { PanelProps } from '../types.js';
import { formatAge, openExternal } from '../utils.js';

export interface Alert {
  name: string;
  severity: string;
  state: string;
  summary: string;
  alertmanager: string;
  startsAt: string;
  url: string | null;
}

interface ApiFilter {
  id: string;
  name: string;
}

interface ApiAlert {
  alertName: string;
  severity?: string;
  state?: string;
  summary?: string;
  alertmanager?: { name?: string };
  actions?: { source?: string };
  raw?: { startsAt?: string; generatorURL?: string };
}

const severityRank: Record<string, number> = {
  critical: 0,
  error: 1,
  warning: 2,
  info: 3,
};

async function getJson<T>(url: string): Promise<T> {
  const response = await fetch(url, { signal: AbortSignal.timeout(5000) });
  if (!response.ok) {
    throw new Error(`${response.status} ${response.statusText}`);
  }
  return (await response.json()) as T;
}

export async function fetchAlerts(
  baseUrl: string,
  filterName: string,
): Promise<Alert[]> {
  const filters = await getJson<ApiFilter[]>(`${baseUrl}/api/filters`);
  const filter = filters.find((f) => f.name === filterName);
  if (!filter) {
    throw new Error(
      `filter "${filterName}" not found (available: ${filters.map((f) => f.name).join(', ')})`,
    );
  }

  const alerts = await getJson<ApiAlert[]>(
    `${baseUrl}/api/filters/${filter.id}/alerts`,
  );
  return alerts
    .map((alert) => ({
      name: alert.alertName,
      severity: alert.severity ?? 'none',
      state: alert.state ?? 'active',
      summary: alert.summary ?? '',
      alertmanager: alert.alertmanager?.name ?? '',
      startsAt: alert.raw?.startsAt ?? '',
      url: alert.actions?.source ?? alert.raw?.generatorURL ?? null,
    }))
    .sort((a, b) => {
      const rank =
        (severityRank[a.severity] ?? 9) - (severityRank[b.severity] ?? 9);
      if (rank !== 0) return rank;
      return b.startsAt.localeCompare(a.startsAt);
    });
}

function severityColor(alert: Alert): string {
  if (alert.state === 'suppressed') return 'gray';
  switch (alert.severity) {
    case 'critical':
    case 'error':
      return 'red';
    case 'warning':
      return 'yellow';
    case 'info':
      return 'blue';
    default:
      return 'gray';
  }
}

export function AlertsPanel({
  id,
  index,
  title,
  focused,
  interval,
  params,
}: PanelProps) {
  const { config } = useAppContext();
  const filter = String(params.filter ?? 'all');
  const { data, error, loading, lastUpdated } = usePanelData(
    id,
    () => fetchAlerts(config.alertmanagerUrl, filter),
    interval,
  );
  const alerts = data ?? [];
  const selected = useListNavigation(id, alerts.length, (i) => {
    if (alerts[i].url) openExternal(alerts[i].url!);
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
      {alerts.length === 0 && data ? (
        <Text color="green" dimColor>
          No alerts ✓
        </Text>
      ) : (
        <SelectList
          items={alerts}
          selected={focused ? selected : -1}
          renderItem={(alert, isSelected) => (
            <Text
              wrap="truncate"
              bold={isSelected}
              dimColor={alert.state === 'suppressed'}
            >
              {isSelected ? '❯ ' : '  '}
              <Text dimColor>
                {alert.startsAt
                  ? formatAge(new Date(alert.startsAt)).padStart(3)
                  : '  ?'}{' '}
              </Text>
              <Text color={severityColor(alert)}>● </Text>
              {alert.name}
              <Text dimColor> · {alert.alertmanager}</Text>
            </Text>
          )}
        />
      )}
    </PanelFrame>
  );
}
