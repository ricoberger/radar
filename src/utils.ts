import { execFile, spawn } from 'node:child_process';

export function run(
  command: string,
  args: string[],
  timeoutMs = 20000,
): Promise<string> {
  return new Promise((resolve, reject) => {
    execFile(
      command,
      args,
      { timeout: timeoutMs, maxBuffer: 16 * 1024 * 1024 },
      (error, stdout, stderr) => {
        if (error) {
          const detail = stderr.trim().split('\n')[0] || error.message;
          reject(new Error(detail));
          return;
        }
        resolve(stdout);
      },
    );
  });
}

export function openExternal(url: string): void {
  spawn('open', [url], { detached: true, stdio: 'ignore' }).unref();
}

export function formatAge(date: Date): string {
  const seconds = Math.max(0, Math.floor((Date.now() - date.getTime()) / 1000));
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
  return `${Math.floor(seconds / 86400)}d`;
}

export function formatTime(date: Date): string {
  return date.toLocaleTimeString('de-DE', {
    hour: '2-digit',
    minute: '2-digit',
  });
}
