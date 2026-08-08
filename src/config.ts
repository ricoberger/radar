import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { parse } from 'yaml';

import { panelDefaults } from './panels/registry.js';
import { Config, FlatPanel, isPanelNode, LayoutNode } from './types.js';

const DEFAULT_CONFIG_PATH = path.join(
  os.homedir(),
  '.config',
  'radar',
  'config.yaml',
);

const defaults: Omit<Config, 'dashboards'> = {
  editor: process.env.EDITOR || 'vim',
};

function validateLayout(node: LayoutNode, trail: string): void {
  if (isPanelNode(node)) {
    if (!(node.panel in panelDefaults)) {
      throw new Error(
        `${trail}: unknown panel type "${node.panel}" (available: ${Object.keys(panelDefaults).join(', ')})`,
      );
    }
    panelDefaults[node.panel].validateParams?.(node.params ?? {}, trail);
    return;
  }
  if (!Array.isArray(node.children) || node.children.length === 0) {
    throw new Error(
      `${trail}: a layout node needs either "panel" or a non-empty "children" list`,
    );
  }
  if (
    node.direction !== undefined &&
    node.direction !== 'row' &&
    node.direction !== 'column'
  ) {
    throw new Error(`${trail}: "direction" must be "row" or "column"`);
  }
  node.children.forEach((child, i) =>
    validateLayout(child, `${trail}.children[${i}]`),
  );
}

export function loadConfig(argPath?: string): Config {
  const configPath = argPath ?? process.env.RADAR_CONFIG ?? DEFAULT_CONFIG_PATH;
  if (!fs.existsSync(configPath)) {
    throw new Error(
      `No config file found at ${configPath}.\nCreate it or pass a path: radar --config <config.yaml>`,
    );
  }

  let fileConfig: Partial<Config>;
  try {
    fileConfig = (parse(fs.readFileSync(configPath, 'utf8')) ??
      {}) as Partial<Config>;
  } catch (error) {
    throw new Error(
      `Failed to parse ${configPath}: ${(error as Error).message}`,
    );
  }

  const config: Config = { ...defaults, dashboards: [], ...fileConfig };
  if (!Array.isArray(config.dashboards) || config.dashboards.length === 0) {
    throw new Error(`${configPath}: "dashboards" must be a non-empty list`);
  }
  config.dashboards = config.dashboards.map((dashboard, i) => {
    if (!dashboard || typeof dashboard !== 'object' || !dashboard.layout) {
      throw new Error(`dashboards[${i}]: missing "layout"`);
    }
    validateLayout(dashboard.layout, `dashboards[${i}].layout`);
    return {
      name: dashboard.name ?? `Dashboard ${i + 1}`,
      layout: dashboard.layout,
    };
  });
  return config;
}

export function flattenPanels(
  node: LayoutNode,
  dashboardIndex: number,
  panels: FlatPanel[] = [],
): FlatPanel[] {
  if (isPanelNode(node)) {
    const index = panels.length + 1;
    const defaultsForType = panelDefaults[node.panel];
    panels.push({
      id: `d${dashboardIndex}:p${index}:${node.panel}`,
      type: node.panel,
      index,
      title:
        node.title ??
        defaultsForType.deriveTitle?.(node.params ?? {}) ??
        defaultsForType.title,
      interval: node.interval ?? defaultsForType.interval,
      params: node.params ?? {},
    });
    return panels;
  }
  for (const child of node.children)
    flattenPanels(child, dashboardIndex, panels);
  return panels;
}
