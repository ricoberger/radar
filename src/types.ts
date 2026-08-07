export interface PanelNode {
  panel: string;
  title?: string;
  weight?: number;
  interval?: number;
  params?: Record<string, unknown>;
}

export interface SplitNode {
  direction?: 'row' | 'column';
  weight?: number;
  children: LayoutNode[];
}

export type LayoutNode = PanelNode | SplitNode;

export function isPanelNode(node: LayoutNode): node is PanelNode {
  return typeof (node as PanelNode).panel === 'string';
}

export interface Dashboard {
  name: string;
  layout: LayoutNode;
}

export interface Config {
  editor: string;
  dashboards: Dashboard[];
}

export interface FlatPanel {
  id: string;
  type: string;
  index: number;
  title: string;
  interval: number;
  params: Record<string, unknown>;
}

export interface PanelProps {
  id: string;
  index: number;
  title: string;
  focused: boolean;
  interval: number;
  params: Record<string, unknown>;
}
