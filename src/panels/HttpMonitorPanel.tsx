import { Box, Text } from 'ink';
import http from 'node:http';
import https from 'node:https';

import { PanelFrame } from '../components/PanelFrame.js';
import { SelectionMarker, SelectList } from '../components/SelectList.js';
import { demoHttpMonitor, isDemoMode } from '../demo.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { useListNavigation } from '../hooks/usePanelKeys.js';
import { PanelProps } from '../types.js';
import { openExternal } from '../utils.js';

const DEFAULT_TIMEOUT_SECONDS = 2;

interface HttpTarget {
  name: string;
  url: string;
  method: string;
  body?: string;
  username?: string;
  password?: string;
  token?: string;
  insecure: boolean;
  timeoutMs: number;
}

// Phase durations are in milliseconds; a phase that did not happen (TLS on
// plain HTTP, or anything after the point of failure) stays undefined.
export interface CheckResult {
  name: string;
  url: string;
  status: number;
  dnsLookup?: number;
  tcpConnection?: number;
  tlsHandshake?: number;
  serverProcessing?: number;
  contentTransfer?: number;
  total?: number;
}

function readTarget(raw: Record<string, unknown>): HttpTarget {
  return {
    name: String(raw.name),
    url: String(raw.url),
    method:
      typeof raw.method === 'string' && raw.method !== ''
        ? raw.method.toUpperCase()
        : 'GET',
    body: typeof raw.body === 'string' ? raw.body : undefined,
    username: typeof raw.username === 'string' ? raw.username : undefined,
    password: typeof raw.password === 'string' ? raw.password : undefined,
    token: typeof raw.token === 'string' ? raw.token : undefined,
    insecure: raw.insecure === true,
    timeoutMs:
      (typeof raw.timeout === 'number' && raw.timeout > 0
        ? raw.timeout
        : DEFAULT_TIMEOUT_SECONDS) * 1000,
  };
}

function readParams(params: Record<string, unknown>): HttpTarget[] {
  const targets = Array.isArray(params.targets) ? params.targets : [];
  return targets.map((target) => readTarget(target as Record<string, unknown>));
}

export function validateHttpMonitorParams(
  params: Record<string, unknown>,
  trail: string,
): void {
  if (!Array.isArray(params.targets) || params.targets.length === 0) {
    throw new Error(
      `${trail}: "params.targets" is required for httpmonitor panels and must be a non-empty list`,
    );
  }
  params.targets.forEach((target, i) => {
    const at = `${trail}: "params.targets[${i}]`;
    if (typeof target !== 'object' || target === null) {
      throw new Error(`${at}" must be an object`);
    }
    const t = target as Record<string, unknown>;
    if (typeof t.name !== 'string' || t.name.trim() === '') {
      throw new Error(`${at}.name" is required`);
    }
    if (typeof t.url !== 'string' || !/^https?:\/\//.test(t.url)) {
      throw new Error(`${at}.url" is required and must be a http(s) URL`);
    }
    for (const field of ['method', 'body', 'username', 'password', 'token']) {
      if (t[field] !== undefined && typeof t[field] !== 'string') {
        throw new Error(`${at}.${field}" must be a string`);
      }
    }
    if (t.insecure !== undefined && typeof t.insecure !== 'boolean') {
      throw new Error(`${at}.insecure" must be a boolean`);
    }
    if (
      t.timeout !== undefined &&
      (typeof t.timeout !== 'number' || t.timeout <= 0)
    ) {
      throw new Error(`${at}.timeout" must be a positive number of seconds`);
    }
  });
}

function authorization(target: HttpTarget): Record<string, string> {
  if (target.username && target.password) {
    const credentials = Buffer.from(
      `${target.username}:${target.password}`,
    ).toString('base64');
    return { Authorization: `Basic ${credentials}` };
  }
  if (target.token) {
    return { Authorization: `Bearer ${target.token}` };
  }
  return {};
}

// Runs one check against a target, timing the connection phases through the
// socket events (like httpmonitor does with Go's httptrace). Every check uses
// a fresh connection (agent: false), redirects are not followed and the
// response body is consumed so the content transfer time is measured. Never
// rejects: failures resolve with status 0 and the phases reached so far.
function check(target: HttpTarget): Promise<CheckResult> {
  return new Promise((resolve) => {
    const result: CheckResult = {
      name: target.name,
      url: target.url,
      status: 0,
    };
    const start = performance.now();
    let lookupAt: number | undefined;
    let connectAt: number | undefined;
    let requestSentAt: number | undefined;
    let responseAt: number | undefined;
    let done = false;

    const finish = () => {
      if (done) return;
      done = true;
      clearTimeout(deadline);
      result.total = performance.now() - start;
      resolve(result);
    };

    const request = (
      target.url.startsWith('https:')
        ? https.request
        : (http.request as typeof https.request)
    )(
      target.url,
      {
        method: target.method,
        agent: false,
        rejectUnauthorized: !target.insecure,
        headers: authorization(target),
      },
      (response) => {
        responseAt = performance.now();
        result.status = response.statusCode ?? 0;
        if (requestSentAt !== undefined) {
          result.serverProcessing = responseAt - requestSentAt;
        }
        response.on('data', () => {});
        response.on('end', () => {
          result.contentTransfer = performance.now() - (responseAt ?? start);
          finish();
        });
        response.on('error', finish);
      },
    );

    // The timeout covers the whole check like httpmonitor's context timeout,
    // not just socket inactivity.
    const deadline = setTimeout(() => {
      result.status = 0;
      request.destroy(new Error('timeout'));
    }, target.timeoutMs);

    request.on('socket', (socket) => {
      socket.once('lookup', () => {
        lookupAt = performance.now();
        result.dnsLookup = lookupAt - start;
      });
      socket.once('connect', () => {
        connectAt = performance.now();
        result.tcpConnection = connectAt - (lookupAt ?? start);
      });
      socket.once('secureConnect', () => {
        result.tlsHandshake = performance.now() - (connectAt ?? start);
      });
    });
    request.on('finish', () => {
      requestSentAt = performance.now();
    });
    request.on('error', finish);
    request.on('close', finish);
    request.end(target.body);
  });
}

async function fetchChecks(targets: HttpTarget[]): Promise<CheckResult[]> {
  if (isDemoMode()) return demoHttpMonitor();
  return Promise.all(targets.map((target) => check(target)));
}

function statusColor(status: number): string {
  if (status === 0 || status >= 500) return 'red';
  if (status >= 400) return 'yellow';
  return 'green';
}

function formatDuration(ms?: number): string {
  if (ms === undefined) return '-';
  if (ms < 1000) return `${Math.round(ms)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

const COLUMNS: Array<{
  header: string;
  cell: (result: CheckResult) => string;
}> = [
  { header: 'Name', cell: (result) => result.name },
  { header: 'Status', cell: (result) => String(result.status) },
  { header: 'Total', cell: (result) => formatDuration(result.total) },
  { header: 'DNS Lookup', cell: (result) => formatDuration(result.dnsLookup) },
  {
    header: 'TCP Connection',
    cell: (result) => formatDuration(result.tcpConnection),
  },
  {
    header: 'TLS Handshake',
    cell: (result) => formatDuration(result.tlsHandshake),
  },
  {
    header: 'Server Processing',
    cell: (result) => formatDuration(result.serverProcessing),
  },
  {
    header: 'Content Transfer',
    cell: (result) => formatDuration(result.contentTransfer),
  },
];

function columnWidths(rows: string[][]): number[] {
  return COLUMNS.map((column, i) =>
    Math.max(column.header.length, ...rows.map((cells) => cells[i].length)),
  );
}

function pad(cell: string, width: number, last: boolean): string {
  return last ? cell : cell.padEnd(width);
}

export function HttpMonitorPanel({
  id,
  index,
  title,
  focused,
  interval,
  params,
}: PanelProps) {
  const targets = readParams(params);
  const { data, error, loading, lastUpdated } = usePanelData(
    id,
    () => fetchChecks(targets),
    interval,
  );
  const results = data ?? [];
  const rows = results.map((result) =>
    COLUMNS.map((column) => column.cell(result)),
  );
  const widths = columnWidths(rows);

  const selected = useListNavigation(id, results.length, (i) =>
    openExternal(results[i].url),
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
      <Box flexDirection="column">
        <Text dimColor wrap="truncate">
          {'  '}
          {COLUMNS.map((column, i) =>
            pad(column.header, widths[i], i === COLUMNS.length - 1),
          ).join('   ')}
        </Text>
        <SelectList
          items={results}
          selected={focused ? selected : -1}
          reserve={1}
          renderItem={(result, isSelected) => {
            const cells = COLUMNS.map((column) => column.cell(result));
            return (
              <Text wrap="truncate" bold={isSelected}>
                <SelectionMarker selected={isSelected} />
                {pad(cells[0], widths[0], false)}
                {'   '}
                <Text color={statusColor(result.status)}>
                  {pad(cells[1], widths[1], false)}
                </Text>
                {'   '}
                {cells
                  .slice(2)
                  .map((cell, i) =>
                    pad(cell, widths[i + 2], i + 2 === COLUMNS.length - 1),
                  )
                  .join('   ')}
              </Text>
            );
          }}
        />
      </Box>
    </PanelFrame>
  );
}
