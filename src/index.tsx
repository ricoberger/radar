#!/usr/bin/env node
import { render } from 'ink';
import { spawnSync } from 'node:child_process';

import { App } from './app.js';
import { loadConfig } from './config.js';
import { demoConfig, enableDemoMode } from './demo.js';
import { Config } from './types.js';

const args = process.argv.slice(2);

let config: Config;
try {
  let configPath: string | undefined;
  let demo = false;
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--demo') {
      demo = true;
    } else if (args[i] === '--config') {
      configPath = args[++i];
      if (!configPath) throw new Error('Missing value for --config.');
    } else {
      throw new Error(
        `Unknown argument: ${args[i]}.\nUsage: radar [--config <config.yaml>] [--demo]`,
      );
    }
  }
  if (demo) {
    enableDemoMode();
    config = demoConfig;
  } else {
    config = loadConfig(configPath);
  }
} catch (error) {
  console.error((error as Error).message);
  process.exit(1);
}

const enterAltScreen = () => process.stdout.write('\u001B[?1049h');
const leaveAltScreen = () => process.stdout.write('\u001B[?1049l');

let instance: ReturnType<typeof render>;
let shouldExit = false;
let suspending = false;

function onQuit(): void {
  shouldExit = true;
  instance.unmount();
}

// Suspends the TUI (unmount + leave the alternate screen), runs an external
// command like the editor with full terminal control, then remounts. Panel
// data and UI state survive through the module-level caches in store.ts.
function runExternal(command: string, args: string[]): void {
  suspending = true;
  instance.unmount();
  leaveAltScreen();
  spawnSync(command, args, { stdio: 'inherit' });
  enterAltScreen();
  mount();
}

function mount(): void {
  suspending = false;
  instance = render(
    <App config={config} runExternal={runExternal} onQuit={onQuit} />,
    {
      exitOnCtrlC: false,
    },
  );
  void instance.waitUntilExit().then(() => {
    if (shouldExit && !suspending) {
      process.exit(0);
    }
  });
}

process.on('exit', () => leaveAltScreen());
enterAltScreen();
mount();
