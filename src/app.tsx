import { Box, Text, useApp, useInput } from 'ink';
import { spawnSync } from 'node:child_process';
import { useCallback, useMemo, type ReactNode } from 'react';

import { flattenPanels } from './config.js';
import { AppContext } from './context.js';
import { useScreenSize } from './hooks/useScreenSize.js';
import { panelComponents } from './panels/registry.js';
import { keyRegistry, refreshRegistry, useSessionState } from './store.js';
import { MAUVE } from './theme.js';
import { Config, FlatPanel, isPanelNode, LayoutNode } from './types.js';

interface AppProps {
  config: Config;
}

function PanelHost({ panel, focused }: { panel: FlatPanel; focused: boolean }) {
  const Component = panelComponents[panel.type];
  return (
    <Component
      id={panel.id}
      index={panel.index}
      title={panel.title}
      focused={focused}
      interval={panel.interval}
      params={panel.params}
    />
  );
}

function renderNode(
  node: LayoutNode,
  panels: FlatPanel[],
  counter: { i: number },
  focusedIndex: number,
  key: string,
): ReactNode {
  if (isPanelNode(node)) {
    const panel = panels[counter.i++];
    return (
      <Box key={key} flexGrow={node.weight ?? 1} flexBasis={0} flexShrink={1}>
        <PanelHost panel={panel} focused={panel.index === focusedIndex} />
      </Box>
    );
  }
  return (
    <Box
      key={key}
      flexDirection={node.direction === 'column' ? 'column' : 'row'}
      flexGrow={node.weight ?? 1}
      flexBasis={0}
      flexShrink={1}
    >
      {node.children.map((child, i) =>
        renderNode(child, panels, counter, focusedIndex, `${key}.${i}`),
      )}
    </Box>
  );
}

export function App({ config }: AppProps) {
  const { exit, suspendTerminal } = useApp();
  const { columns, rows } = useScreenSize();

  // Hands the terminal to an external command (editor, fzfgh) and blocks
  // until it exits; ink restores its terminal state and redraws afterwards.
  const runExternal = useCallback(
    (command: string, args: string[]) => {
      void suspendTerminal(() => {
        spawnSync(command, args, { stdio: 'inherit' });
      });
    },
    [suspendTerminal],
  );

  const dashboards = config.dashboards;
  const [active, setActive] = useSessionState('app:dashboard', 0);
  const dashboard = dashboards[active] ?? dashboards[0];
  const panels = useMemo(
    () => flattenPanels(dashboard.layout, dashboards.indexOf(dashboard)),
    [dashboard, dashboards],
  );
  const [focusMap, setFocusMap] = useSessionState<Record<number, number>>(
    'app:focus',
    {},
  );
  const [zoomed, setZoomed] = useSessionState('app:zoomed', false);

  const focusedIndex = Math.min(focusMap[active] ?? 1, panels.length);
  const focusedPanel = panels[focusedIndex - 1] ?? panels[0];

  const setFocusedIndex = (next: number | ((prev: number) => number)) =>
    setFocusMap((prev) => {
      const current = Math.min(prev[active] ?? 1, panels.length);
      return {
        ...prev,
        [active]: typeof next === 'function' ? next(current) : next,
      };
    });

  const switchDashboard = (delta: number) => {
    setActive((prev) => (prev + delta + dashboards.length) % dashboards.length);
    setZoomed(false);
  };

  useInput((input, key) => {
    const handleKey = (char: string) => {
      if (char === 'q' || (key.ctrl && char === 'c')) {
        exit();
        return;
      }
      if (/^[1-9]$/.test(char)) {
        const index = Number(char);
        if (index <= panels.length) setFocusedIndex(index);
        return;
      }
      if (char === '[' || char === ']') {
        if (dashboards.length > 1) switchDashboard(char === '[' ? -1 : 1);
        return;
      }
      if (key.tab) {
        setFocusedIndex((prev) =>
          key.shift
            ? ((prev - 2 + panels.length) % panels.length) + 1
            : (prev % panels.length) + 1,
        );
        return;
      }
      if (char === 'z') {
        setZoomed((prev) => !prev);
        return;
      }
      if (char === 'r') {
        refreshRegistry.get(focusedPanel.id)?.();
        return;
      }
      if (char === 'R') {
        for (const refresh of refreshRegistry.values()) refresh();
        return;
      }
      keyRegistry.get(focusedPanel.id)?.(char, {
        upArrow: key.upArrow,
        downArrow: key.downArrow,
        return: key.return,
        ctrl: key.ctrl,
        shift: key.shift,
        tab: key.tab,
      });
    };

    // Fast typing can deliver several printable keys in one chunk.
    const chars =
      input.length > 1 &&
      !key.upArrow &&
      !key.downArrow &&
      !key.return &&
      !key.tab
        ? [...input]
        : [input];
    for (const char of chars) handleKey(char);
  });

  const context = useMemo(
    () => ({ config, runExternal }),
    [config, runExternal],
  );

  return (
    <AppContext.Provider value={context}>
      <Box flexDirection="column" width={columns} height={rows}>
        {dashboards.length > 1 ? (
          <Box flexShrink={0} paddingX={1} gap={2}>
            {dashboards.map((d, i) => (
              <Text
                key={`${i}:${d.name}`}
                bold={i === active}
                color={i === active ? MAUVE : undefined}
                dimColor={i !== active}
              >
                {d.name}
              </Text>
            ))}
          </Box>
        ) : null}
        {zoomed ? (
          <Box flexGrow={1}>
            <PanelHost panel={focusedPanel} focused />
          </Box>
        ) : (
          renderNode(
            dashboard.layout,
            panels,
            { i: 0 },
            focusedIndex,
            `d${active}`,
          )
        )}
        <Box flexShrink={0} paddingX={1}>
          <Text dimColor wrap="truncate">
            {dashboards.length > 1 ? '[/] dashboard · ' : ''}1-
            {Math.min(panels.length, 9)} focus · tab cycle · j/k select · enter
            open · z zoom · r/R refresh · q quit
          </Text>
        </Box>
      </Box>
    </AppContext.Provider>
  );
}
