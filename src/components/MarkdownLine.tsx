import { Text } from 'ink';
import type { ReactNode } from 'react';

// Minimal line-based markdown styling for the daily note.

function renderInline(text: string, dim = false): ReactNode {
  const parts = text.split(/\*\*([^*]+)\*\*/g);
  return parts.map((part, i) =>
    i % 2 === 1 ? (
      <Text key={i} bold dimColor={dim}>
        {part}
      </Text>
    ) : (
      <Text key={i} dimColor={dim}>
        {part}
      </Text>
    ),
  );
}

export function MarkdownLine({ line }: { line: string }) {
  if (line === '---' || /^tags:/.test(line)) {
    return (
      <Text wrap="truncate" dimColor>
        {line}
      </Text>
    );
  }
  if (line.startsWith('# ')) {
    return (
      <Text wrap="truncate" bold color="cyan">
        {line.slice(2)}
      </Text>
    );
  }
  if (line.startsWith('## ')) {
    return (
      <Text wrap="truncate" bold color="yellow">
        {line.slice(3)}
      </Text>
    );
  }
  if (line.startsWith('### ')) {
    return (
      <Text wrap="truncate" bold>
        {line.slice(4)}
      </Text>
    );
  }
  const done = line.match(/^(\s*)- \[[xX]\] (.*)$/);
  if (done) {
    return (
      <Text wrap="truncate">
        {done[1]}
        <Text color="green">☑ </Text>
        {renderInline(done[2], true)}
      </Text>
    );
  }
  const todo = line.match(/^(\s*)- \[ \] (.*)$/);
  if (todo) {
    return (
      <Text wrap="truncate">
        {todo[1]}
        <Text color="yellow">☐ </Text>
        {renderInline(todo[2])}
      </Text>
    );
  }
  const bullet = line.match(/^(\s*)- (.*)$/);
  if (bullet) {
    return (
      <Text wrap="truncate">
        {bullet[1]}
        <Text dimColor>• </Text>
        {renderInline(bullet[2])}
      </Text>
    );
  }
  return <Text wrap="truncate">{renderInline(line)}</Text>;
}
