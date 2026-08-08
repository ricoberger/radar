import { Text } from 'ink';

import { PanelFrame } from '../components/PanelFrame.js';
import { SelectionMarker, SelectList } from '../components/SelectList.js';
import { usePanelData } from '../hooks/usePanelData.js';
import { useListNavigation } from '../hooks/usePanelKeys.js';
import { PanelProps } from '../types.js';
import { formatAge, openExternal, run } from '../utils.js';

export interface MailMessage {
  subject: string;
  sender: string;
  date: string;
  id: string;
  account: string;
}

interface AppleMailParams {
  messages: 'unread' | 'all';
  limit: number;
  include: string[];
  exclude: string[];
}

function readParams(params: Record<string, unknown>): AppleMailParams {
  return {
    messages: (params.messages as AppleMailParams['messages']) ?? 'unread',
    limit: typeof params.limit === 'number' ? params.limit : 10,
    include: Array.isArray(params.include) ? params.include.map(String) : [],
    exclude: Array.isArray(params.exclude) ? params.exclude.map(String) : [],
  };
}

export function validateAppleMailParams(
  params: Record<string, unknown>,
  trail: string,
): void {
  if (
    params.messages !== undefined &&
    params.messages !== 'unread' &&
    params.messages !== 'all'
  ) {
    throw new Error(`${trail}: "messages" must be "unread" or "all"`);
  }
  if (
    params.limit !== undefined &&
    (!Number.isInteger(params.limit) || (params.limit as number) < 1)
  ) {
    throw new Error(`${trail}: "limit" must be a positive integer`);
  }
  for (const key of ['include', 'exclude'] as const) {
    if (params[key] !== undefined && !Array.isArray(params[key])) {
      throw new Error(`${trail}: "${key}" must be a list of account names`);
    }
  }
}

// Reads the INBOX of every (matching) account with bulk property fetches -
// one Apple Event per property instead of five per message. If include and
// exclude are both set the filters are ignored and everything is fetched.
function buildScript(params: AppleMailParams): string {
  return `
function run() {
  const params = ${JSON.stringify(params)};
  const Mail = Application("Mail");
  if (!Mail.running()) {
    return "NOT_RUNNING";
  }
  const useInclude = params.include.length > 0 && params.exclude.length === 0;
  const useExclude = params.exclude.length > 0 && params.include.length === 0;
  const out = [];
  for (const acct of Mail.accounts()) {
    const account = acct.name();
    if (useInclude && params.include.indexOf(account) === -1) continue;
    if (useExclude && params.exclude.indexOf(account) !== -1) continue;
    let msgs;
    try {
      msgs = acct.mailboxes.byName("INBOX").messages;
      if (params.messages === "unread") {
        msgs = msgs.whose({ readStatus: false });
      }
      const subjects = msgs.subject();
      const senders = msgs.sender();
      const dates = msgs.dateReceived();
      const ids = msgs.messageId();
      for (let i = 0; i < subjects.length; i++) {
        out.push({
          subject: subjects[i],
          sender: senders[i],
          date: dates[i],
          id: ids[i],
          account: account,
        });
      }
    } catch (error) {
      continue;
    }
  }
  out.sort((a, b) => b.date - a.date);
  return JSON.stringify(out.slice(0, params.limit));
}
`;
}

export async function fetchMail(
  params: AppleMailParams,
): Promise<MailMessage[]> {
  const stdout = await run(
    'osascript',
    ['-l', 'JavaScript', '-e', buildScript(params)],
    30000,
  );
  const trimmed = stdout.trim();
  if (trimmed === 'NOT_RUNNING') {
    throw new Error('Mail.app is not running');
  }
  return JSON.parse(trimmed) as MailMessage[];
}

export function mailUrl(message: MailMessage): string {
  return `message://%3C${encodeURIComponent(message.id)}%3E`;
}

export function senderName(sender: string): string {
  const match = sender.match(/^"?([^"<]+)"?\s*</);
  return (match ? match[1] : sender).trim();
}

export function AppleMailPanel({
  id,
  index,
  title,
  focused,
  interval,
  params,
}: PanelProps) {
  const mailParams = readParams(params);
  const { data, error, loading, lastUpdated } = usePanelData(
    id,
    () => fetchMail(mailParams),
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
        mailParams.messages === 'unread' ? (
          <Text color="green" dimColor>
            Inbox zero ✓
          </Text>
        ) : (
          <Text dimColor>No messages</Text>
        )
      ) : (
        <SelectList
          items={messages}
          selected={focused ? selected : -1}
          renderItem={(message, isSelected) => (
            <Text wrap="truncate" bold={isSelected}>
              <SelectionMarker selected={isSelected} />
              <Text color="yellow">{senderName(message.sender)}</Text>
              <Text dimColor> · </Text>
              {message.subject}
              <Text dimColor>
                {' '}
                · {formatAge(new Date(message.date))} · {message.account}
              </Text>
            </Text>
          )}
        />
      )}
    </PanelFrame>
  );
}
