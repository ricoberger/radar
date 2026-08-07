import { Text } from 'ink';

import { PanelFrame } from '../components/PanelFrame.js';
import { SelectList } from '../components/SelectList.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { useListNavigation } from '../hooks/usePanelKeys.js';
import { PanelProps } from '../types.js';
import { formatAge, openExternal, run } from '../utils.js';

export interface MailMessage {
  subject: string;
  sender: string;
  date: string;
  id: string;
}

const script = `
function run() {
  const Mail = Application("Mail");
  if (!Mail.running()) {
    return "NOT_RUNNING";
  }
  const messages = Mail.inbox.messages.whose({ readStatus: false });
  const subjects = messages.subject();
  const senders = messages.sender();
  const dates = messages.dateReceived();
  const ids = messages.messageId();
  return JSON.stringify(
    subjects.map((subject, i) => ({
      subject: subject,
      sender: senders[i],
      date: dates[i],
      id: ids[i],
    })),
  );
}
`;

export async function fetchUnreadMail(): Promise<MailMessage[]> {
  const stdout = await run(
    'osascript',
    ['-l', 'JavaScript', '-e', script],
    30000,
  );
  const trimmed = stdout.trim();
  if (trimmed === 'NOT_RUNNING') {
    throw new Error('Mail.app is not running');
  }
  const messages = JSON.parse(trimmed) as MailMessage[];
  return messages.sort((a, b) => b.date.localeCompare(a.date));
}

export function mailUrl(message: MailMessage): string {
  return `message://%3C${encodeURIComponent(message.id)}%3E`;
}

export function senderName(sender: string): string {
  const match = sender.match(/^"?([^"<]+)"?\s*</);
  return (match ? match[1] : sender).trim();
}

export function MailPanel({ id, index, title, focused, interval }: PanelProps) {
  const { data, error, loading, lastUpdated } = usePanelData(
    id,
    fetchUnreadMail,
    interval,
  );
  const messages = data ?? [];
  const selected = useListNavigation(id, messages.length, (i) =>
    openExternal(mailUrl(messages[i])),
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
      {messages.length === 0 && data ? (
        <Text color="green" dimColor>
          Inbox zero ✓
        </Text>
      ) : (
        <SelectList
          items={messages}
          selected={focused ? selected : -1}
          renderItem={(message, isSelected) => (
            <Text wrap="truncate" bold={isSelected}>
              {isSelected ? '❯ ' : '  '}
              <Text dimColor>
                {formatAge(new Date(message.date)).padStart(3)}{' '}
              </Text>
              <Text color="yellow">{senderName(message.sender)}</Text>
              <Text dimColor> · </Text>
              {message.subject}
            </Text>
          )}
        />
      )}
    </PanelFrame>
  );
}
