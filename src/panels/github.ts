import { demoGithubSearch, isDemoMode } from '../demo.js';
import { openExternal, run } from '../utils.js';

// Shared helpers for the github-prs, github-issues and github-notifications
// panels, mirroring fzfgh's list rendering and view action.

export type OpenMode = 'browser' | 'fzfgh';

export function readOpenParam(params: Record<string, unknown>): OpenMode {
  return params.open === 'fzfgh' ? 'fzfgh' : 'browser';
}

export function validateOpenParam(
  params: Record<string, unknown>,
  trail: string,
): void {
  if (
    params.open !== undefined &&
    params.open !== 'browser' &&
    params.open !== 'fzfgh'
  ) {
    throw new Error(`${trail}: "params.open" must be "browser" or "fzfgh"`);
  }
}

// Opens a GitHub URL either in the browser or inside fzfgh. fzfgh's view
// command only supports pull request and issue URLs; anything else falls
// back to the browser.
export function openGithubItem(
  url: string,
  mode: OpenMode,
  runExternal: (command: string, args: string[]) => void,
): void {
  if (
    mode === 'fzfgh' &&
    /github\.com\/[^/]+\/[^/]+\/(pull|issues)\/\d+/.test(url)
  ) {
    runExternal('fzfgh', ['view', url]);
    return;
  }
  openExternal(url);
}

// fzfgh's relative time: "just now", "5m ago", ..., "2w ago", then "Mar 05".
export function reltime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 60) return 'just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;
  if (seconds < 2592000) return `${Math.floor(seconds / 604800)}w ago`;
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: '2-digit',
    timeZone: 'UTC',
  });
}

export interface SearchItem {
  number: number;
  title: string;
  repository: string;
  author: string;
  state: string;
  isDraft: boolean;
  createdAt: string;
  url: string;
}

interface ApiSearchItem {
  number: number;
  title: string;
  repository: { nameWithOwner: string };
  author?: { login?: string };
  state?: string;
  isDraft?: boolean;
  createdAt: string;
  url: string;
}

export async function searchGithub(
  kind: 'prs' | 'issues',
  query: string,
  limit: number,
): Promise<SearchItem[]> {
  if (isDemoMode()) return demoGithubSearch(kind);
  const fields = [
    'number',
    'title',
    'repository',
    'author',
    'state',
    'createdAt',
    'url',
  ];
  if (kind === 'prs') fields.push('isDraft');
  const stdout = await run(
    'gh',
    [
      'search',
      kind,
      '--json',
      fields.join(','),
      '--limit',
      String(limit),
      ...query.split(/\s+/).filter(Boolean),
    ],
    30000,
  );
  return (JSON.parse(stdout) as ApiSearchItem[]).map((item) => ({
    number: item.number,
    title: item.title,
    repository: item.repository.nameWithOwner,
    author: item.author?.login ?? '-',
    state: (item.state ?? '').toLowerCase(),
    isDraft: item.isDraft ?? false,
    createdAt: item.createdAt,
    url: item.url,
  }));
}

// fzfgh's icon colors: draft = gray, open = green, closed = mauve.
export function searchItemColor(item: SearchItem): string {
  if (item.isDraft) return 'gray';
  if (item.state === 'closed') return 'magenta';
  return 'green';
}

export function validateGithubSearchParams(type: string) {
  return (params: Record<string, unknown>, trail: string): void => {
    if (typeof params.query !== 'string' || params.query.trim() === '') {
      throw new Error(
        `${trail}: "params.query" is required for ${type} panels`,
      );
    }
    validateOpenParam(params, trail);
  };
}
