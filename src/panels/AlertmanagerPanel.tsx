import { Text } from 'ink';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

import { PanelFrame } from '../components/PanelFrame.js';
import { SelectionMarker, SelectList } from '../components/SelectList.js';
import { useAppContext } from '../context.js';
import { demoAlerts, isDemoMode } from '../demo.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { useListNavigation } from '../hooks/usePanelKeys.js';
import { PanelProps } from '../types.js';
import { formatAge, openExternal, runWithInput } from '../utils.js';

const DEFAULT_URL = 'http://127.0.0.1:9093';
const ICON_ANALYZE = '\uf0d0';

interface AlertmanagerParams {
  url: string;
  filter?: string;
  alertmanager?: string;
}

export interface Alert {
  fingerprint: string;
  amId: string;
  name: string;
  severity: string;
  state: string;
  summary: string;
  alertmanager: string;
  startsAt: string;
  markdown: string;
  actions: Record<string, string>;
  analysisAvailable: boolean;
  analysisExists: boolean;
  analysisRunning: boolean;
}

interface ApiSource {
  id: string;
  name: string;
}

interface ApiAlert {
  fingerprint: string;
  alertName: string;
  severity?: string;
  state?: string;
  summary?: string;
  markdown?: string;
  alertmanager?: { id?: string; name?: string };
  actions?: Record<string, string | null>;
  analysis?: { available?: boolean; exists?: boolean; running?: boolean };
  raw?: { startsAt?: string };
}

function readParams(params: Record<string, unknown>): AlertmanagerParams {
  return {
    url: String(params.url ?? DEFAULT_URL).replace(/\/$/, ''),
    filter: typeof params.filter === 'string' ? params.filter : undefined,
    alertmanager:
      typeof params.alertmanager === 'string' ? params.alertmanager : undefined,
  };
}

export function validateAlertmanagerParams(
  params: Record<string, unknown>,
  trail: string,
): void {
  if (params.url !== undefined && typeof params.url !== 'string') {
    throw new Error(`${trail}: "params.url" must be a string`);
  }
  const hasFilter =
    typeof params.filter === 'string' && params.filter.trim() !== '';
  const hasAlertmanager =
    typeof params.alertmanager === 'string' &&
    params.alertmanager.trim() !== '';
  if (hasFilter === hasAlertmanager) {
    throw new Error(
      `${trail}: exactly one of "params.filter" or "params.alertmanager" is required for alertmanager panels`,
    );
  }
}

async function getJson<T>(url: string): Promise<T> {
  const response = await fetch(url, { signal: AbortSignal.timeout(5000) });
  if (!response.ok) {
    throw new Error(`${response.status} ${response.statusText}`);
  }
  return (await response.json()) as T;
}

// Resolves the configured filter / alertmanager name to its id and returns
// the path serving its alerts.
async function alertsPath(params: AlertmanagerParams): Promise<string> {
  const kind = params.filter !== undefined ? 'filters' : 'alertmanagers';
  const name = params.filter ?? params.alertmanager ?? '';
  const sources = await getJson<ApiSource[]>(`${params.url}/api/${kind}`);
  const source = sources.find((s) => s.name === name);
  if (!source) {
    throw new Error(
      `${kind === 'filters' ? 'filter' : 'alertmanager'} "${name}" not found (available: ${sources.map((s) => s.name).join(', ')})`,
    );
  }
  return `/api/${kind}/${source.id}/alerts`;
}

export async function fetchAlerts(
  params: AlertmanagerParams,
): Promise<Alert[]> {
  if (isDemoMode()) return demoAlerts();
  const alerts = await getJson<ApiAlert[]>(
    `${params.url}${await alertsPath(params)}`,
  );
  // Server order is preserved (like fzfalertmanager).
  return alerts.map((alert) => ({
    fingerprint: alert.fingerprint,
    amId: alert.alertmanager?.id ?? '',
    name: alert.alertName,
    severity: alert.severity ?? 'none',
    state: alert.state ?? 'active',
    summary: (alert.summary ?? '').replace(/\s+/g, ' ').trim(),
    alertmanager: alert.alertmanager?.name ?? '',
    startsAt: alert.raw?.startsAt ?? '',
    markdown: alert.markdown ?? '',
    actions: Object.fromEntries(
      Object.entries(alert.actions ?? {}).filter(
        (entry): entry is [string, string] => !!entry[1],
      ),
    ),
    analysisAvailable: alert.analysis?.available ?? false,
    analysisExists: alert.analysis?.exists ?? false,
    analysisRunning: alert.analysis?.running ?? false,
  }));
}

function severityColor(severity: string): string {
  switch (severity) {
    case 'critical':
      return 'magenta';
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

function analysisColor(alert: Alert): string {
  if (alert.analysisExists) return 'green';
  if (alert.analysisRunning) return 'yellow';
  return 'gray';
}

// Writes the alert markdown to a temp file and opens it in the editor,
// mirroring fzfalertmanager's view action.
function viewMarkdown(
  alert: Alert,
  editor: string,
  runExternal: (command: string, args: string[]) => void,
): void {
  const file = path.join(os.tmpdir(), `radar-alert-${alert.fingerprint}.md`);
  fs.writeFileSync(file, alert.markdown);
  const [command, ...args] = editor.split(/\s+/);
  runExternal(command, [...args, file]);
}

export function AlertmanagerPanel({
  id,
  index,
  title,
  focused,
  interval,
  params,
}: PanelProps) {
  const { config, runExternal } = useAppContext();
  const resolved = readParams(params);
  const { data, error, loading, lastUpdated, refresh } = usePanelData(
    id,
    () => fetchAlerts(resolved),
    interval,
  );
  const alerts = data ?? [];

  // Opens the finished analysis in the editor, or starts a run when none
  // exists yet; a running analysis is left alone (icon already shows it).
  const analyze = (alert: Alert) => {
    const base = `${resolved.url}/api/alertmanagers/${alert.amId}/alerts/${alert.fingerprint}`;
    if (alert.analysisExists) {
      void getJson<{ filePath?: string }>(`${base}/analysis`)
        .then(({ filePath }) => {
          if (filePath && fs.existsSync(filePath)) {
            const [command, ...args] = config.editor.split(/\s+/);
            runExternal(command, [...args, filePath]);
          }
        })
        .catch(() => {});
      return;
    }
    if (alert.analysisRunning || !alert.analysisAvailable) return;
    void fetch(`${base}/analyze`, {
      method: 'POST',
      signal: AbortSignal.timeout(20000),
    })
      .catch(() => {})
      .finally(() => refresh());
  };

  const selected = useListNavigation(
    id,
    alerts.length,
    (i) => viewMarkdown(alerts[i], config.editor, runExternal),
    (input, _key, i) => {
      if (i < 0) return;
      const alert = alerts[i];
      const links: Record<string, string> = {
        o: 'source',
        s: 'silence',
        b: 'runbook',
        d: 'dashboard',
        p: 'panel',
      };
      if (input in links) {
        const url = alert.actions[links[input]];
        if (url) openExternal(url);
      } else if (input === 'a') {
        analyze(alert);
      } else if (input === 'y') {
        void runWithInput(
          'pbcopy',
          [],
          alert.actions.source ?? alert.fingerprint,
        ).catch(() => {});
      }
    },
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
              <SelectionMarker selected={isSelected} />
              <Text color={analysisColor(alert)}>{ICON_ANALYZE} </Text>
              <Text color={severityColor(alert.severity)}>
                {alert.severity.padEnd(9)}
              </Text>{' '}
              {alert.name}
              {alert.summary ? <Text dimColor> {alert.summary}</Text> : null}
              <Text dimColor>
                {'  '}
                {alert.startsAt ? formatAge(new Date(alert.startsAt)) : '?'}
                {'  '}
                {alert.alertmanager}
              </Text>
            </Text>
          )}
        />
      )}
    </PanelFrame>
  );
}
