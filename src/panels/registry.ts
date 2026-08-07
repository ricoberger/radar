import { ComponentType } from 'react';

import { PanelProps } from '../types.js';
import { AlertsPanel } from './AlertsPanel.js';
import {
  AppleCalendarPanel,
  appleCalendarTitle,
  validateAppleCalendarParams,
} from './AppleCalendarPanel.js';
import { GithubNotificationsPanel } from './GithubNotificationsPanel.js';
import { GithubPrsPanel } from './GithubPrsPanel.js';
import { MailPanel } from './MailPanel.js';
import { NotePanel } from './NotePanel.js';

export interface PanelTypeDefaults {
  title: string;
  interval: number;
  deriveTitle?: (params: Record<string, unknown>) => string;
  validateParams?: (params: Record<string, unknown>, trail: string) => void;
}

export const panelDefaults: Record<string, PanelTypeDefaults> = {
  'apple-calendar': {
    title: 'Calendar',
    interval: 300,
    deriveTitle: appleCalendarTitle,
    validateParams: validateAppleCalendarParams,
  },
  note: { title: 'Daily Note', interval: 5 },
  mail: { title: 'Mail', interval: 60 },
  alerts: { title: 'Alerts', interval: 60 },
  'github-prs': { title: 'Pull Requests', interval: 120 },
  'github-notifications': { title: 'GitHub Notifications', interval: 120 },
};

export const panelComponents: Record<string, ComponentType<PanelProps>> = {
  'apple-calendar': AppleCalendarPanel,
  note: NotePanel,
  mail: MailPanel,
  alerts: AlertsPanel,
  'github-prs': GithubPrsPanel,
  'github-notifications': GithubNotificationsPanel,
};
