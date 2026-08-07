import { Text } from 'ink';

import { PanelFrame } from '../components/PanelFrame.js';
import { SelectList } from '../components/SelectList.js';
import { useAppContext } from '../context.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { useListNavigation } from '../hooks/usePanelKeys.js';
import { PanelProps } from '../types.js';
import { run } from '../utils.js';
import {
  openGithubItem,
  readOpenParam,
  reltime,
  validateOpenParam,
} from './github.js';

const ICON_UNREAD = '\uea71';
const ICON_READ = '\uebb5';

export interface Notification {
  id: string;
  isUnread: boolean;
  reason: string;
  title: string;
  type: string;
  repository: string;
  updatedAt: string;
  url: string;
  prState: string;
  issueState: string;
  isDraft: boolean;
  conclusion: string;
}

interface ApiNotification {
  id: string;
  isUnread?: boolean;
  reason?: string;
  title?: string;
  lastUpdatedAt?: string;
  url?: string;
  subject?: {
    __typename?: string;
    pullRequestState?: string;
    issueState?: string;
    isDraft?: boolean;
    conclusion?: string;
  };
}

// Fetches the inbox through the gh-notifications helper (GraphQL), which
// provides the state fields behind fzfgh's colored type icons.
export async function fetchNotifications(
  limit: number,
): Promise<Notification[]> {
  const stdout = await run('gh-notifications', ['list'], 30000);
  return (JSON.parse(stdout) as ApiNotification[])
    .slice(0, limit)
    .map((notification) => ({
      id: notification.id,
      isUnread: notification.isUnread ?? false,
      reason: (notification.reason ?? '').toLowerCase().replace(/_/g, ' '),
      title: notification.title ?? '-',
      type: notification.subject?.__typename ?? '',
      repository:
        notification.url?.match(/github\.com\/([^/]+\/[^/]+)/)?.[1] ?? '-',
      updatedAt: notification.lastUpdatedAt ?? '',
      url: notification.url ?? '',
      prState: notification.subject?.pullRequestState ?? '',
      issueState: notification.subject?.issueState ?? '',
      isDraft: notification.subject?.isDraft ?? false,
      conclusion: notification.subject?.conclusion ?? '',
    }));
}

export function validateGithubNotificationsParams(
  params: Record<string, unknown>,
  trail: string,
): void {
  validateOpenParam(params, trail);
}

// fzfgh's type icon + color mapping.
function typeIcon(notification: Notification): {
  icon: string;
  color: string;
} {
  switch (notification.type) {
    case 'PullRequest':
      return {
        icon: '\uf407',
        color: notification.isDraft
          ? 'gray'
          : notification.prState === 'OPEN'
            ? 'green'
            : notification.prState === 'MERGED'
              ? 'magenta'
              : notification.prState === 'CLOSED'
                ? 'red'
                : 'gray',
      };
    case 'Issue':
      return {
        icon: '\uf41b',
        color:
          notification.issueState === 'OPEN'
            ? 'green'
            : notification.issueState === 'CLOSED'
              ? 'magenta'
              : 'gray',
      };
    case 'Release':
      return { icon: '\uf412', color: 'gray' };
    case 'CheckSuite':
      return {
        icon: '\uf52e',
        color: notification.conclusion === 'SUCCESS' ? 'green' : 'red',
      };
    case 'WorkflowRun':
      return { icon: '\uf52e', color: 'red' };
    case 'Commit':
      return { icon: '\uf417', color: 'gray' };
    case 'Gist':
      return { icon: '\uf480', color: 'gray' };
    case 'Discussion':
    case 'TeamDiscussion':
      return { icon: '\uf442', color: 'gray' };
    default:
      return { icon: '\uf128', color: 'gray' };
  }
}

export function GithubNotificationsPanel({
  id,
  index,
  title,
  focused,
  interval,
  params,
}: PanelProps) {
  const { runExternal } = useAppContext();
  const limit = Number(params.limit ?? 50);
  const open = readOpenParam(params);
  const { data, error, loading, lastUpdated } = usePanelData(
    id,
    () => fetchNotifications(limit),
    interval,
  );
  const notifications = data ?? [];
  const selected = useListNavigation(id, notifications.length, (i) =>
    openGithubItem(notifications[i].url, open, runExternal),
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
      {notifications.length === 0 && data ? (
        <Text color="green" dimColor>
          No notifications ✓
        </Text>
      ) : (
        <SelectList
          items={notifications}
          selected={focused ? selected : -1}
          renderItem={(notification, isSelected) => {
            const { icon, color } = typeIcon(notification);
            return (
              <Text wrap="truncate" bold={isSelected}>
                {isSelected ? '❯ ' : '  '}
                <Text color={notification.isUnread ? 'yellow' : 'gray'}>
                  {notification.isUnread ? ICON_UNREAD : ICON_READ}
                </Text>{' '}
                <Text color={color}>{icon}</Text> [{notification.type}]{' '}
                {notification.repository}: {notification.title} (
                {notification.reason} - {reltime(notification.updatedAt)})
              </Text>
            );
          }}
        />
      )}
    </PanelFrame>
  );
}
