import { Text } from 'ink';

import { PanelFrame } from '../components/PanelFrame.js';
import { SelectList } from '../components/SelectList.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { useListNavigation } from '../hooks/usePanelKeys.js';
import { PanelProps } from '../types.js';
import { formatAge, openExternal, run } from '../utils.js';

export interface Notification {
  id: string;
  reason: string;
  title: string;
  type: string;
  repository: string;
  updatedAt: string;
  url: string;
}

interface ApiNotification {
  id: string;
  reason: string;
  updated_at: string;
  subject: { title: string; type: string; url: string | null };
  repository: { full_name: string; html_url: string };
}

function notificationUrl(notification: ApiNotification): string {
  const { subject, repository } = notification;
  if (subject.type === 'CheckSuite') return `${repository.html_url}/actions`;
  if (subject.type === 'Release') return `${repository.html_url}/releases`;
  if (subject.type === 'Discussion')
    return `${repository.html_url}/discussions`;
  if (!subject.url) return repository.html_url;
  return subject.url
    .replace('https://api.github.com/repos/', 'https://github.com/')
    .replace('/pulls/', '/pull/')
    .replace('/commits/', '/commit/');
}

export async function fetchNotifications(
  limit: number,
): Promise<Notification[]> {
  const stdout = await run(
    'gh',
    ['api', `notifications?per_page=${limit}`],
    30000,
  );
  return (JSON.parse(stdout) as ApiNotification[]).map((notification) => ({
    id: notification.id,
    reason: notification.reason,
    title: notification.subject.title,
    type: notification.subject.type,
    repository: notification.repository.full_name,
    updatedAt: notification.updated_at,
    url: notificationUrl(notification),
  }));
}

const typeIcons: Record<string, string> = {
  PullRequest: '⇄',
  Issue: '◉',
  Release: '⏷',
  Discussion: '☰',
  CheckSuite: '✓',
  Commit: '⌥',
};

export function GithubNotificationsPanel({
  id,
  index,
  title,
  focused,
  interval,
  params,
}: PanelProps) {
  const limit = Number(params.limit ?? 50);
  const { data, error, loading, lastUpdated } = usePanelData(
    id,
    () => fetchNotifications(limit),
    interval,
  );
  const notifications = data ?? [];
  const selected = useListNavigation(id, notifications.length, (i) =>
    openExternal(notifications[i].url),
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
          renderItem={(notification, isSelected) => (
            <Text wrap="truncate" bold={isSelected}>
              {isSelected ? '❯ ' : '  '}
              <Text dimColor>
                {formatAge(new Date(notification.updatedAt)).padStart(3)}{' '}
              </Text>
              <Text color="blue">{typeIcons[notification.type] ?? '·'} </Text>
              <Text color="magenta">{notification.repository}</Text>
              <Text dimColor> · </Text>
              {notification.title}
            </Text>
          )}
        />
      )}
    </PanelFrame>
  );
}
