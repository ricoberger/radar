import { Text } from 'ink';

import { PanelFrame } from '../components/PanelFrame.js';
import { SelectList } from '../components/SelectList.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { useListNavigation } from '../hooks/usePanelKeys.js';
import { PanelProps } from '../types.js';
import { formatAge, openExternal, run } from '../utils.js';

export interface PullRequest {
  number: number;
  title: string;
  repository: string;
  url: string;
  isDraft: boolean;
  updatedAt: string;
}

interface ApiPullRequest {
  number: number;
  title: string;
  repository: { nameWithOwner: string };
  url: string;
  isDraft: boolean;
  updatedAt: string;
}

export async function fetchPullRequests(
  query: string,
  limit: number,
): Promise<PullRequest[]> {
  const args = [
    'search',
    'prs',
    '--json',
    'number,title,repository,url,isDraft,updatedAt',
    '--limit',
    String(limit),
    ...query.split(/\s+/).filter(Boolean),
  ];
  const stdout = await run('gh', args, 30000);
  return (JSON.parse(stdout) as ApiPullRequest[]).map((pr) => ({
    number: pr.number,
    title: pr.title,
    repository: pr.repository.nameWithOwner,
    url: pr.url,
    isDraft: pr.isDraft,
    updatedAt: pr.updatedAt,
  }));
}

export function GithubPrsPanel({
  id,
  index,
  title,
  focused,
  interval,
  params,
}: PanelProps) {
  const query = typeof params.query === 'string' ? params.query : '';
  const limit = Number(params.limit ?? 20);
  const { data, error, loading, lastUpdated } = usePanelData(
    id,
    () => {
      if (!query) throw new Error('missing "query" param');
      return fetchPullRequests(query, limit);
    },
    interval,
  );
  const prs = data ?? [];
  const selected = useListNavigation(id, prs.length, (i) =>
    openExternal(prs[i].url),
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
      {prs.length === 0 && data ? (
        <Text dimColor>No pull requests</Text>
      ) : (
        <SelectList
          items={prs}
          selected={focused ? selected : -1}
          renderItem={(pr, isSelected) => (
            <Text wrap="truncate" bold={isSelected}>
              {isSelected ? '❯ ' : '  '}
              <Text dimColor>
                {formatAge(new Date(pr.updatedAt)).padStart(3)}{' '}
              </Text>
              {pr.isDraft ? <Text dimColor>✎ </Text> : null}
              <Text color="magenta">
                {pr.repository}#{pr.number}
              </Text>
              <Text dimColor> · </Text>
              {pr.title}
            </Text>
          )}
        />
      )}
    </PanelFrame>
  );
}
