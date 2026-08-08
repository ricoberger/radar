import { Box, Text } from 'ink';

import { PanelFrame } from '../components/PanelFrame.js';
import { SelectionMarker, SelectList } from '../components/SelectList.js';
import { demoKubectlIssues, isDemoMode } from '../demo.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { useListNavigation } from '../hooks/usePanelKeys.js';
import { PanelProps } from '../types.js';
import { run } from '../utils.js';

interface KubectlIssuesParams {
  command: string;
  contexts: string[];
  args: string[];
}

type Row = Record<string, string>;

function readParams(params: Record<string, unknown>): KubectlIssuesParams {
  return {
    command: typeof params.command === 'string' ? params.command : '',
    contexts: Array.isArray(params.contexts) ? params.contexts.map(String) : [],
    args: Array.isArray(params.args) ? params.args.map(String) : [],
  };
}

function validateStringList(value: unknown, name: string, trail: string): void {
  if (
    value !== undefined &&
    (!Array.isArray(value) || value.some((item) => typeof item !== 'string'))
  ) {
    throw new Error(`${trail}: "params.${name}" must be a list of strings`);
  }
}

export function validateKubectlIssuesParams(
  params: Record<string, unknown>,
  trail: string,
): void {
  if (typeof params.command !== 'string' || params.command.trim() === '') {
    throw new Error(
      `${trail}: "params.command" is required for kubectl-issues panels`,
    );
  }
  validateStringList(params.contexts, 'contexts', trail);
  validateStringList(params.args, 'args', trail);
  if (
    Array.isArray(params.args) &&
    params.args.some((arg) => arg === '-o' || arg === '--output')
  ) {
    throw new Error(
      `${trail}: "params.args" must not contain "-o" / "--output", the panel always uses JSON`,
    );
  }
}

async function fetchIssues(params: KubectlIssuesParams): Promise<Row[]> {
  if (isDemoMode()) return demoKubectlIssues();
  const stdout = await run(
    'kubectl',
    [
      'issues',
      params.command,
      ...params.contexts.flatMap((context) => ['--context', context]),
      ...params.args,
      '-o',
      'json',
    ],
    60000,
  );
  return (
    (stdout.trim() === '' ? [] : (JSON.parse(stdout) as Row[] | null)) ?? []
  );
}

// Column names in table order, taken from the first row (the plugin's JSON
// keys match the CLI table headers).
function columns(rows: Row[]): string[] {
  return rows.length > 0 ? Object.keys(rows[0]) : [];
}

function columnWidths(rows: Row[], names: string[]): number[] {
  return names.map((name, i) =>
    Math.max(
      name.length,
      ...rows.map((row) => (row[name] ?? '').length),
      i === names.length - 1 ? 0 : 1,
    ),
  );
}

function formatRow(cells: string[], widths: number[]): string {
  return cells
    .map((cell, i) => (i === widths.length - 1 ? cell : cell.padEnd(widths[i])))
    .join('   ');
}

export function KubectlIssuesPanel({
  id,
  index,
  title,
  focused,
  interval,
  params,
}: PanelProps) {
  const resolved = readParams(params);
  const { data, error, loading, lastUpdated } = usePanelData(
    id,
    () => fetchIssues(resolved),
    interval,
  );
  const rows = data ?? [];
  const names = columns(rows);
  const widths = columnWidths(rows, names);

  const selected = useListNavigation(id, rows.length, () => {});

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
        <Text color="green" dimColor>
          No issues ✓
        </Text>
      ) : (
        <Box flexDirection="column">
          <Text dimColor wrap="truncate">
            {'  '}
            {formatRow(names, widths)}
          </Text>
          <SelectList
            items={rows}
            selected={focused ? selected : -1}
            reserve={1}
            renderItem={(row, isSelected) => (
              <Text wrap="truncate" bold={isSelected}>
                <SelectionMarker selected={isSelected} />
                {formatRow(
                  names.map((name) => row[name] ?? ''),
                  widths,
                )}
              </Text>
            )}
          />
        </Box>
      )}
    </PanelFrame>
  );
}
