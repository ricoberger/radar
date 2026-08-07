import { Text } from 'ink';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

import { PanelFrame } from '../components/PanelFrame.js';
import { SelectList } from '../components/SelectList.js';
import { useAppContext } from '../context.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { useListNavigation } from '../hooks/usePanelKeys.js';
import { PanelProps } from '../types.js';
import { run, runWithInput } from '../utils.js';

const DEFAULT_FLAG_FIELD = 'customfield_10002';

interface JiraParams {
  jql: string;
  limit: number;
  flagField: string;
}

interface WorkItem {
  key: string;
  status: string;
  statusColor: string;
  summary: string;
}

interface ApiNamed {
  name?: string;
}

interface ApiUser {
  displayName?: string;
}

interface ApiIssueRef {
  key?: string;
  fields?: {
    summary?: string;
    issuetype?: ApiNamed;
    status?: ApiNamed;
  };
}

interface ApiComment {
  author?: ApiUser;
  created?: string;
  body?: unknown;
}

interface ApiIssueLink {
  type?: { name?: string; inward?: string; outward?: string };
  outwardIssue?: ApiIssueRef;
  inwardIssue?: ApiIssueRef;
}

interface ApiWorkItem {
  key: string;
  fields?: {
    summary?: string;
    status?: {
      name?: string;
      statusCategory?: { colorName?: string };
    };
    issuetype?: ApiNamed;
    priority?: ApiNamed;
    assignee?: ApiUser;
    reporter?: ApiUser;
    labels?: string[];
    parent?: ApiIssueRef;
    created?: string;
    updated?: string;
    description?: unknown;
    comment?: { total?: number; comments?: ApiComment[] };
    subtasks?: ApiIssueRef[];
    issuelinks?: ApiIssueLink[];
    [field: string]: unknown;
  };
}

function readParams(params: Record<string, unknown>): JiraParams {
  return {
    jql: typeof params.jql === 'string' ? params.jql : '',
    limit: Number(params.limit ?? 50),
    flagField:
      typeof params.flagField === 'string'
        ? params.flagField
        : DEFAULT_FLAG_FIELD,
  };
}

export function validateJiraParams(
  params: Record<string, unknown>,
  trail: string,
): void {
  if (typeof params.jql !== 'string' || params.jql.trim() === '') {
    throw new Error(`${trail}: "params.jql" is required for jira panels`);
  }
  if (params.flagField !== undefined && typeof params.flagField !== 'string') {
    throw new Error(`${trail}: "params.flagField" must be a string`);
  }
}

// fzfjira's status colors, keyed by the status category color name.
function statusColor(colorName: string): string {
  switch (colorName) {
    case 'green':
      return 'green';
    case 'yellow':
    case 'brown':
      return 'yellow';
    case 'blue-gray':
      return 'blue';
    case 'warm-red':
      return 'red';
    default:
      return 'white';
  }
}

// fzfjira's pad(s; n): pad and truncate to exactly n characters.
function pad(s: string, n: number): string {
  return s.padEnd(n).slice(0, n);
}

function clean(s: string | undefined): string {
  return (s ?? '-').replace(/[\t\n\r]/g, ' ');
}

async function fetchWorkItems(params: JiraParams): Promise<WorkItem[]> {
  const stdout = await run(
    'acli',
    [
      'jira',
      'workitem',
      'search',
      '--jql',
      params.jql,
      '--limit',
      String(params.limit),
      '--json',
    ],
    30000,
  );
  const items =
    (stdout.trim() === ''
      ? []
      : (JSON.parse(stdout) as ApiWorkItem[] | null)) ?? [];
  // Server order is preserved (like fzfjira).
  return items.map((item) => ({
    key: item.key,
    status: item.fields?.status?.name ?? '-',
    statusColor: statusColor(
      item.fields?.status?.statusCategory?.colorName ?? 'medium-gray',
    ),
    summary: clean(item.fields?.summary),
  }));
}

// Converts a Jira ADF document to Markdown text via md's raw mode, falling
// back to a short note when md is not installed (like fzfjira).
async function adfToMd(adf: unknown): Promise<string> {
  try {
    const out = await runWithInput('md', [], JSON.stringify(adf), {
      MD_ADF: 'true',
      MD_RAW: 'true',
    });
    return out.replace(/\s+$/, '');
  } catch {
    return '_(install md to render this content)_';
  }
}

function fmtdate(s: string | undefined): string {
  return s ? s.slice(0, 16).replace('T', ' ') : '';
}

// Assembles the whole work item as a single Markdown document, mirroring
// fzfjira's emit_markdown: metadata, description, comments, sub-tasks, links.
async function emitMarkdown(
  wi: ApiWorkItem,
  flagField: string,
): Promise<string> {
  const f = wi.fields ?? {};
  const lines: string[] = [];

  lines.push(`# ${wi.key}  ${clean(f.summary)}`, '');
  lines.push(`- **Key:** \`${wi.key}\``);
  lines.push(`- **Status:** ${f.status?.name ?? '-'}`);
  if (f.issuetype?.name) lines.push(`- **Type:** ${f.issuetype.name}`);
  if (f.priority?.name) lines.push(`- **Priority:** ${f.priority.name}`);
  if (f.assignee?.displayName) {
    lines.push(`- **Assignee:** ${f.assignee.displayName}`);
  }
  if (f.reporter?.displayName) {
    lines.push(`- **Reporter:** ${f.reporter.displayName}`);
  }
  if (f.labels && f.labels.length > 0) {
    lines.push(
      `- **Labels:** ${f.labels.map((label) => `\`${label}\``).join(' ')}`,
    );
  }
  if (f.parent) {
    const kind =
      f.parent.fields?.issuetype?.name === 'Epic' ? 'Epic' : 'Parent';
    lines.push(
      `- **${kind}:** \`${f.parent.key ?? '-'}\` · ${clean(f.parent.fields?.summary)}`,
    );
  }
  const flags = (f[flagField] as Array<{ value?: string }> | undefined) ?? [];
  if (flags.length > 0) {
    lines.push(
      `- **Flagged:** ${flags.map((flag) => flag.value ?? '').join(', ')}`,
    );
  }
  if (f.created) lines.push(`- **Created:** ${fmtdate(f.created)}`);
  if (f.updated) lines.push(`- **Updated:** ${fmtdate(f.updated)}`);
  lines.push('', '---', '');

  if (typeof f.description === 'object' && f.description !== null) {
    lines.push(await adfToMd(f.description));
  } else {
    lines.push('_No description provided._');
  }
  lines.push('');

  const comments = f.comment?.comments ?? [];
  const total = f.comment?.total ?? comments.length;
  if (comments.length > 0) {
    lines.push(
      comments.length < total
        ? `# Comments (showing ${comments.length} of ${total})`
        : `# Comments (${total})`,
      '',
    );
    for (const comment of comments) {
      lines.push(
        `**${comment.author?.displayName ?? '?'}** · ${fmtdate(comment.created)}`,
        '',
      );
      lines.push(await adfToMd(comment.body), '');
    }
  }

  const subtasks = f.subtasks ?? [];
  if (subtasks.length > 0) {
    lines.push(`# Sub-tasks (${subtasks.length})`, '');
    for (const subtask of subtasks) {
      lines.push(
        `- \`${subtask.key ?? '-'}\` · ${subtask.fields?.status?.name ?? '-'} · ${clean(subtask.fields?.summary)}`,
      );
    }
    lines.push('');
  }

  const links = (f.issuelinks ?? []).filter(
    (link) => link.outwardIssue || link.inwardIssue,
  );
  if (links.length > 0) {
    lines.push(`# Links (${links.length})`, '');
    for (const link of links) {
      const dir = link.outwardIssue
        ? (link.type?.outward ?? link.type?.name)
        : (link.type?.inward ?? link.type?.name);
      const issue = link.outwardIssue ?? link.inwardIssue;
      lines.push(
        `- ${dir ?? '-'} · \`${issue?.key ?? '-'}\` · ${issue?.fields?.status?.name ?? '-'} · ${clean(issue?.fields?.summary)}`,
      );
    }
    lines.push('');
  }

  return lines.join('\n').replace(/\n+$/, '\n');
}

// Fetches the full work item, writes it as a Markdown document to a temp
// file and opens it in the editor, mirroring fzfjira's view action.
async function viewWorkItem(
  key: string,
  flagField: string,
  editor: string,
  runExternal: (command: string, args: string[]) => void,
): Promise<void> {
  const stdout = await run(
    'acli',
    ['jira', 'workitem', 'view', key, '--fields', '*all', '--json'],
    30000,
  );
  const doc = await emitMarkdown(JSON.parse(stdout) as ApiWorkItem, flagField);
  const file = path.join(os.tmpdir(), `radar-jira-${key}.md`);
  fs.writeFileSync(file, doc);
  const [command, ...args] = editor.split(/\s+/);
  runExternal(command, [...args, file]);
}

export function JiraPanel({
  id,
  index,
  title,
  focused,
  interval,
  params,
}: PanelProps) {
  const { config, runExternal } = useAppContext();
  const resolved = readParams(params);
  const { data, error, loading, lastUpdated } = usePanelData(
    id,
    () => fetchWorkItems(resolved),
    interval,
  );
  const items = data ?? [];

  const selected = useListNavigation(
    id,
    items.length,
    (i) => {
      void viewWorkItem(
        items[i].key,
        resolved.flagField,
        config.editor,
        runExternal,
      ).catch(() => {});
    },
    (input, _key, i) => {
      if (i < 0) return;
      const item = items[i];
      if (input === 'o') {
        void run('acli', ['jira', 'workitem', 'view', item.key, '--web']).catch(
          () => {},
        );
      } else if (input === 'y') {
        void runWithInput('pbcopy', [], item.key).catch(() => {});
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
      {items.length === 0 && data ? (
        <Text dimColor>No work items</Text>
      ) : (
        <SelectList
          items={items}
          selected={focused ? selected : -1}
          renderItem={(item, isSelected) => (
            <Text wrap="truncate" bold={isSelected}>
              {isSelected ? '❯ ' : '  '}
              {pad(item.key, 12)}{' '}
              <Text color={item.statusColor}>{pad(item.status, 14)}</Text>{' '}
              {item.summary}
            </Text>
          )}
        />
      )}
    </PanelFrame>
  );
}
