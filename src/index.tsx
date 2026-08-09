#!/usr/bin/env node
import { render } from 'ink';

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

const instance = render(<App config={config} />, {
  exitOnCtrlC: false,
  alternateScreen: true,
});
void instance.waitUntilExit().then(() => process.exit(0));
