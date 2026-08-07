import { ComponentType } from 'react';

import { PanelProps } from '../types.js';
import {
  AlertmanagerPanel,
  validateAlertmanagerParams,
} from './AlertmanagerPanel.js';
import {
  AppleCalendarPanel,
  appleCalendarTitle,
  validateAppleCalendarParams,
} from './AppleCalendarPanel.js';
import { AppleMailPanel, validateAppleMailParams } from './AppleMailPanel.js';
import { validateGithubSearchParams } from './github.js';
import { GithubIssuesPanel } from './GithubIssuesPanel.js';
import {
  GithubNotificationsPanel,
  validateGithubNotificationsParams,
} from './GithubNotificationsPanel.js';
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
  alertmanager: {
    title: 'Alerts',
    interval: 60,
    validateParams: validateAlertmanagerParams,
  },
  'github-prs': {
    title: 'Pull Requests',
    interval: 300,
    validateParams: validateGithubSearchParams('github-prs'),
  },
  'github-issues': {
    title: 'Issues',
    interval: 300,
    validateParams: validateGithubSearchParams('github-issues'),
  },
  'github-notifications': {
    title: 'GitHub Notifications',
    interval: 300,
    validateParams: validateGithubNotificationsParams,
  },
};

export const panelComponents: Record<string, ComponentType<PanelProps>> = {
  'apple-calendar': AppleCalendarPanel,
  'ricoberger-notes': RicobergerNotesPanel,
  'apple-mail': AppleMailPanel,
  alertmanager: AlertmanagerPanel,
  'github-prs': GithubPrsPanel,
  'github-issues': GithubIssuesPanel,
  'github-notifications': GithubNotificationsPanel,
};
