import { ComponentType } from 'react';

import { PanelProps } from '../types.js';
import { AlertsPanel } from './AlertsPanel.js';
import {
  AppleCalendarPanel,
  appleCalendarTitle,
  validateAppleCalendarParams,
} from './AppleCalendarPanel.js';
import { AppleMailPanel, validateAppleMailParams } from './AppleMailPanel.js';
import { GithubNotificationsPanel } from './GithubNotificationsPanel.js';
import { GithubPrsPanel } from './GithubPrsPanel.js';
import {
  RicobergerNotesPanel,
  ricobergerNotesTitle,
  validateRicobergerNotesParams,
} from './RicobergerNotesPanel.js';

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
  'ricoberger-notes': {
    title: 'Daily Note',
    interval: 300,
    deriveTitle: ricobergerNotesTitle,
    validateParams: validateRicobergerNotesParams,
  },
  'apple-mail': {
    title: 'Mail',
    interval: 300,
    validateParams: validateAppleMailParams,
  },
  alerts: { title: 'Alerts', interval: 60 },
  'github-prs': { title: 'Pull Requests', interval: 120 },
  'github-notifications': { title: 'GitHub Notifications', interval: 120 },
};

export const panelComponents: Record<string, ComponentType<PanelProps>> = {
  'apple-calendar': AppleCalendarPanel,
  'ricoberger-notes': RicobergerNotesPanel,
  'apple-mail': AppleMailPanel,
  alerts: AlertsPanel,
  'github-prs': GithubPrsPanel,
  'github-notifications': GithubNotificationsPanel,
};
