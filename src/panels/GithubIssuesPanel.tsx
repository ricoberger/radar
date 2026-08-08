import { Text } from 'ink';

import { PanelFrame } from '../components/PanelFrame.js';
import { SelectionMarker, SelectList } from '../components/SelectList.js';
import { useAppContext } from '../context.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { useListNavigation } from '../hooks/usePanelKeys.js';
import { PanelProps } from '../types.js';
import {
  openGithubItem,
  readOpenParam,
  reltime,
  searchGithub,
  searchItemColor,
} from './github.js';

const ICON_ISSUE = '\uf41b';

export function GithubIssuesPanel({
  id,
  index,
  title,
  focused,
  interval,
  params,
}: PanelProps) {
  const { runExternal } = useAppContext();
  const query = typeof params.query === 'string' ? params.query : '';
  const limit = Number(params.limit ?? 20);
  const open = readOpenParam(params);
  const { data, error, loading, lastUpdated } = usePanelData(
    id,
    () => searchGithub('issues', query, limit),
    interval,
  );
  const issues = data ?? [];
  const selected = useListNavigation(id, issues.length, (i) =>
    openGithubItem(issues[i].url, open, runExternal),
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
      {issues.length === 0 && data ? (
        <Text dimColor>No issues</Text>
      ) : (
        <SelectList
          items={issues}
          selected={focused ? selected : -1}
          renderItem={(issue, isSelected) => (
            <Text wrap="truncate" bold={isSelected}>
              <SelectionMarker selected={isSelected} />
              <Text color={searchItemColor(issue)}>{ICON_ISSUE}</Text> [#
              {issue.number}] {issue.repository}: {issue.title}
              {'  '}({issue.author} · {reltime(issue.createdAt)})
            </Text>
          )}
        />
      )}
    </PanelFrame>
  );
}
