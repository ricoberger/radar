import { ComponentType } from 'react';

import { PanelProps } from '../types.js';
import { AlertsPanel } from './AlertsPanel.js';
import { CalendarPanel } from './CalendarPanel.js';
import { GithubNotificationsPanel } from './GithubNotificationsPanel.js';
import { GithubPrsPanel } from './GithubPrsPanel.js';
import { MailPanel } from './MailPanel.js';
import { NotePanel } from './NotePanel.js';

export const panelDefaults: Record<
  string,
  { title: string; interval: number }
> = {
  calendar: { title: 'Calendar', interval: 300 },
  note: { title: 'Daily Note', interval: 5 },
  mail: { title: 'Mail', interval: 60 },
  alerts: { title: 'Alerts', interval: 60 },
  'github-prs': { title: 'Pull Requests', interval: 120 },
  'github-notifications': { title: 'GitHub Notifications', interval: 120 },
};

export const panelComponents: Record<string, ComponentType<PanelProps>> = {
  calendar: CalendarPanel,
  note: NotePanel,
  mail: MailPanel,
  alerts: AlertsPanel,
  'github-prs': GithubPrsPanel,
  'github-notifications': GithubNotificationsPanel,
};
