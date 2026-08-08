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

const ICON_PR = '\uf407';

export function GithubPrsPanel({
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
    () => searchGithub('prs', query, limit),
    interval,
  );
  const prs = data ?? [];
  const selected = useListNavigation(id, prs.length, (i) =>
    openGithubItem(prs[i].url, open, runExternal),
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
              <SelectionMarker selected={isSelected} />
              <Text color={searchItemColor(pr)}>{ICON_PR}</Text> [#{pr.number}]{' '}
              {pr.repository}: {pr.title}
              {'  '}({pr.author} · {reltime(pr.createdAt)})
            </Text>
          )}
        />
      )}
    </PanelFrame>
  );
}
