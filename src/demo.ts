import type { Alert } from './panels/AlertmanagerPanel.js';
import type { CalendarEvent } from './panels/AppleCalendarPanel.js';
import type { MailMessage } from './panels/AppleMailPanel.js';
import type { SearchItem } from './panels/github.js';
import type { Notification } from './panels/GithubNotificationsPanel.js';
import type { Config } from './types.js';

// Demo mode (--demo): the app runs with the built-in config below and every
// fetch function returns generic fake data instead of talking to the real
// tools. Used for screenshots; keymaps stay wired and may hit real tools.

let demoMode = false;

export function enableDemoMode(): void {
  demoMode = true;
}

export function isDemoMode(): boolean {
  return demoMode;
}

export const demoConfig: Config = {
  editor: process.env.EDITOR || 'vim',
  dashboards: [
    {
      name: 'Demo',
      layout: {
        direction: 'row',
        children: [
          {
            direction: 'column',
            weight: 2,
            children: [
              { panel: 'alertmanager', params: { filter: 'all' } },
              {
                panel: 'kubectl-issues',
                params: {
                  command: 'pods',
                  contexts: ['prod-eu1', 'stage-eu1', 'dev-eu1'],
                  args: ['-A'],
                },
              },
            ],
          },
          {
            direction: 'column',
            weight: 2,
            children: [
              {
                direction: 'column',
                children: [
                  {
                    panel: 'jira',
                    params: { jql: 'assignee = currentUser()' },
                  },
                  { panel: 'github-notifications', weight: 2 },
                ],
              },

              {
                direction: 'column',
                children: [
                  {
                    panel: 'github-prs',
                    params: { query: 'author:@me is:open' },
                  },
                  {
                    panel: 'github-issues',
                    params: { query: 'assignee:@me is:open' },
                  },
                ],
              },
            ],
          },
          {
            direction: 'column',
            children: [
              { panel: 'apple-calendar' },
              {
                panel: 'ricoberger-notes',
                weight: 2,
                params: { dir: '~/notes' },
              },
              { panel: 'apple-mail' },
            ],
          },
        ],
      },
    },
  ],
};

function minutesAgo(minutes: number): string {
  return new Date(Date.now() - minutes * 60000).toISOString();
}

// Times are relative to the given day (and to now for the running event) so
// the panel always shows a lively, current-looking schedule.
export function demoCalendarEvents(days: Date[]): CalendarEvent[] {
  const at = (day: Date, hour: number, minute: number, duration: number) => {
    const start = new Date(day);
    start.setHours(hour, minute, 0, 0);
    const end = new Date(start.getTime() + duration * 60000);
    return { start: start.toISOString(), end: end.toISOString() };
  };
  return days.flatMap((day) => [
    {
      title: 'On-call',
      calendar: 'Work',
      ...at(day, 0, 0, 24 * 60),
      isAllDay: true,
      location: null,
    },
    {
      title: 'Daily Standup',
      calendar: 'Work',
      ...at(day, 9, 30, 15),
      isAllDay: false,
      location: null,
    },
    {
      title: 'Incident Review INC-217',
      calendar: 'Work',
      ...at(day, 10, 0, 30),
      isAllDay: false,
      location: null,
    },
    {
      title: 'Focus Time',
      calendar: 'Work',
      start: minutesAgo(30),
      end: minutesAgo(-60),
      isAllDay: false,
      location: null,
    },
    {
      title: 'Lunch',
      calendar: 'Private',
      ...at(day, 12, 30, 45),
      isAllDay: false,
      location: null,
    },
    {
      title: '1:1 with Dana',
      calendar: 'Work',
      ...at(day, 14, 0, 30),
      isAllDay: false,
      location: null,
    },
    {
      title: 'Platform Sync',
      calendar: 'Work',
      ...at(day, 15, 0, 60),
      isAllDay: false,
      location: 'Room 4.2',
    },
    {
      title: 'Postgres 17 upgrade planning',
      calendar: 'Work',
      ...at(day, 16, 30, 45),
      isAllDay: false,
      location: null,
    },
    {
      title: 'Climbing',
      calendar: 'Private',
      ...at(day, 18, 30, 90),
      isAllDay: false,
      location: null,
    },
  ]);
}

export const demoNote = `# ${new Date().toISOString().slice(0, 10)}

## Todos

- [x] Review checkout-service deployment
- [x] Rotate the registry pull credentials
- [ ] Finish the incident report for INC-217
- [ ] Prepare the platform sync agenda
- [ ] Review DEMO-433 (rate limiting for the payments API)
- [ ] Schedule the Postgres 17 upgrade for stage-eu1

## Notes

The new ingress setup is live on **prod-eu1**. Latency dropped by ~15%,
error rate unchanged. Rollout to \`prod-us1\` is planned for tomorrow.

Standup: search-indexer image pulls are failing on prod-us1 since the
registry migration — hubot is on it, workaround is a manual pull.

## Links

- [Ingress rollout plan](https://example.com/ingress-rollout)
- [INC-217 timeline](https://example.com/inc-217)
`;

export function demoMail(): MailMessage[] {
  const messages = [
    ['Grafana', 'Alert: HighMemoryUsage resolved', 12, 'Work'],
    ['Dana Miller', 'Re: Platform sync agenda', 48, 'Work'],
    ['GitHub', 'acme/checkout-service: v2.4.0 released', 95, 'Work'],
    ['Statuspage', 'Scheduled maintenance on Saturday', 130, 'Private'],
    ['Jira', 'DEMO-421 was moved to In Progress', 180, 'Work'],
    ['Alex Chen', 'Lunch next week?', 240, 'Private'],
    ['PagerDuty', 'On-call handoff: you are next', 320, 'Work'],
    ['Confluence', 'Dana Miller edited "Ingress rollout plan"', 410, 'Work'],
    ['Newsletter', 'This week in Kubernetes', 520, 'Private'],
  ] as const;
  return messages.map(([sender, subject, minutes, account], i) => ({
    id: `demo-${i}`,
    sender,
    subject,
    date: minutesAgo(minutes),
    account,
  }));
}

export function demoAlerts(): Alert[] {
  const alerts = [
    [
      'KubePodCrashLooping',
      'critical',
      'active',
      'Pod checkout/worker-6d9f is crash looping',
      35,
      true,
    ],
    [
      'BlackboxProbeFailed',
      'critical',
      'active',
      'probe for https://status.acme.dev failed',
      8,
      false,
    ],
    [
      'HighMemoryUsage',
      'error',
      'active',
      'payments-api memory above 90% for 15m',
      140,
      false,
    ],
    [
      'KafkaConsumerLag',
      'error',
      'active',
      'checkout-events consumer lag above 100k messages',
      75,
      false,
    ],
    [
      'KubeNodeNotReady',
      'error',
      'active',
      'node prod-us1-worker-3 is NotReady for 10m',
      48,
      false,
    ],
    [
      'CertificateExpiresSoon',
      'warning',
      'active',
      'cert for api.acme.dev expires in 13 days',
      2000,
      false,
    ],
    [
      'TargetDown',
      'warning',
      'active',
      '25% of search-indexer targets are down',
      55,
      false,
    ],
    [
      'KubeHpaMaxedOut',
      'warning',
      'active',
      'HPA payments/payments-api at max replicas for 30m',
      95,
      false,
    ],
    [
      'HighErrorRate',
      'warning',
      'active',
      '5xx rate on checkout-service above 1% for 10m',
      22,
      false,
    ],
    [
      'PersistentVolumeFillingUp',
      'warning',
      'active',
      'PV postgres-data-2 will be full in 4 days',
      430,
      false,
    ],
    [
      'KubeJobFailed',
      'warning',
      'active',
      'job search/reindex-29ln4 failed to complete',
      190,
      false,
    ],
    [
      'PostgresReplicationLag',
      'info',
      'active',
      'replica lag on stage-eu1 is 42s',
      260,
      false,
    ],
    [
      'CronJobSuspended',
      'info',
      'active',
      'cronjob checkout/cleanup has been suspended for 7d',
      1600,
      false,
    ],
    [
      'DeploymentReplicasMismatch',
      'info',
      'suppressed',
      'search-indexer has 2/3 ready replicas',
      600,
      false,
    ],
    [
      'NodeDiskPressure',
      'info',
      'suppressed',
      'node prod-eu1-worker-8 is under disk pressure',
      780,
      false,
    ],
    [
      'WatchdogMissing',
      'info',
      'suppressed',
      'Watchdog alert from stage-eu1 was not received',
      1100,
      false,
    ],
  ] as const;
  return alerts.map(([name, severity, state, summary, minutes, exists], i) => ({
    fingerprint: `demo-${i}`,
    amId: 'demo',
    name,
    severity,
    state,
    summary,
    alertmanager: 'prod-eu1',
    startsAt: minutesAgo(minutes),
    markdown: `# ${name}\n\n${summary}\n`,
    actions: { source: 'https://example.com' },
    analysisAvailable: true,
    analysisExists: exists,
    analysisRunning: false,
  }));
}

export function demoJira() {
  const items = [
    [
      'DEMO-421',
      'In Progress',
      'yellow',
      'Migrate checkout-service to the new ingress',
    ],
    [
      'DEMO-433',
      'In Review',
      'yellow',
      'Add rate limiting to the payments API',
    ],
    [
      'DEMO-429',
      'In Progress',
      'yellow',
      'Fix stale search results after reindexing',
    ],
    ['DEMO-407', 'Open', 'blue', 'Evaluate OpenTelemetry sampling strategies'],
    ['DEMO-402', 'Open', 'blue', 'Add SLO dashboards for the checkout flow'],
    ['DEMO-395', 'On Hold', 'blue', 'Upgrade PostgreSQL clusters to 17'],
    ['DEMO-390', 'Done', 'green', 'Rotate the registry pull credentials'],
    ['DEMO-388', 'Done', 'green', 'Document the on-call escalation policy'],
  ] as const;
  return items.map(([key, status, statusColor, summary]) => ({
    key,
    status,
    statusColor,
    summary,
  }));
}

export function demoKubectlIssues(): Array<Record<string, string>> {
  const rows = [
    [
      'prod-eu1',
      'checkout',
      'worker-6d9f7b5c4-x2x9p',
      '0/1',
      'CrashLoopBackOff',
      '212 (2m ago)',
      '3d',
    ],
    [
      'prod-eu1',
      'checkout',
      'worker-6d9f7b5c4-p8wn4',
      '0/1',
      'CrashLoopBackOff',
      '198 (4m ago)',
      '3d',
    ],
    [
      'prod-eu1',
      'payments',
      'payments-api-79c9b6-kx2lm',
      '1/1',
      'Running',
      '14 (35m ago)',
      '12d',
    ],
    [
      'prod-eu1',
      'payments',
      'payments-api-79c9b6-r7d2s',
      '1/1',
      'Running',
      '11 (52m ago)',
      '12d',
    ],
    [
      'prod-eu1',
      'ingress',
      'ingress-nginx-c8b5d-t7pqm',
      '1/1',
      'Running',
      '3 (2h ago)',
      '9d',
    ],
    [
      'prod-eu1',
      'kafka',
      'kafka-broker-2',
      '1/2',
      'Running',
      '6 (1h ago)',
      '45d',
    ],
    [
      'prod-us1',
      'search',
      'search-indexer-1',
      '0/1',
      'ImagePullBackOff',
      '0',
      '4h',
    ],
    [
      'prod-us1',
      'search',
      'search-indexer-2',
      '0/1',
      'ImagePullBackOff',
      '0',
      '4h',
    ],
    [
      'prod-us1',
      'monitoring',
      'node-exporter-w4kd9',
      '0/1',
      'Error',
      '7 (18m ago)',
      '31d',
    ],
    [
      'prod-us1',
      'monitoring',
      'prometheus-1',
      '1/2',
      'OOMKilled',
      '4 (12m ago)',
      '31d',
    ],
    [
      'prod-us1',
      'checkout',
      'checkout-api-5f7dd9-m3znq',
      '1/1',
      'Running',
      '2 (3h ago)',
      '6d',
    ],
    [
      'prod-us1',
      'kube-system',
      'coredns-787d4b-b5msj',
      '1/1',
      'Running',
      '1 (6h ago)',
      '31d',
    ],
    ['stage-eu1', 'default', 'load-test-jx8j2', '0/1', 'Pending', '0', '25m'],
    [
      'stage-eu1',
      'checkout',
      'checkout-migrate-29ln4',
      '0/1',
      'Completed',
      '0',
      '2h',
    ],
    ['stage-eu1', 'search', 'reindex-29ln4-x8kfx', '0/1', 'Error', '0', '3h'],
    [
      'stage-eu1',
      'postgres',
      'postgres-2',
      '1/1',
      'Running',
      '5 (30m ago)',
      '18d',
    ],
    [
      'dev-eu1',
      'checkout',
      'worker-8b4f2c1d9-q6wtr',
      '0/1',
      'ContainerCreating',
      '0',
      '2m',
    ],
    ['dev-eu1', 'sandbox', 'debug-shell-mona', '1/1', 'Running', '0', '5h'],
  ] as const;
  return rows.map(
    ([context, namespace, name, ready, status, restarts, age]) => ({
      CONTEXT: context,
      NAMESPACE: namespace,
      NAME: name,
      READY: ready,
      STATUS: status,
      RESTARTS: restarts,
      AGE: age,
    }),
  );
}

export function demoGithubSearch(kind: 'prs' | 'issues'): SearchItem[] {
  const prs = [
    [
      512,
      'feat(api): add idempotency keys to checkout',
      'acme/checkout-service',
      'mona',
      'open',
      false,
      90,
    ],
    [
      508,
      'fix(worker): retry failed webhook deliveries',
      'acme/checkout-service',
      'mona',
      'open',
      true,
      400,
    ],
    [
      122,
      'chore(deps): bump opentelemetry to 1.32',
      'acme/payments-api',
      'renovate',
      'open',
      false,
      1500,
    ],
    [
      119,
      'feat(metrics): expose queue depth per shard',
      'acme/payments-api',
      'mona',
      'open',
      false,
      1900,
    ],
    [
      87,
      'docs: document the reindexing runbook',
      'acme/search-indexer',
      'mona',
      'open',
      true,
      2400,
    ],
    [
      45,
      'feat(ui): add dark mode toggle',
      'acme/statuspage',
      'mona',
      'open',
      false,
      3100,
    ],
  ] as const;
  const issues = [
    [
      307,
      'Search results are stale after reindexing',
      'acme/search-indexer',
      'hubot',
      'open',
      false,
      200,
    ],
    [
      301,
      'Add dark mode to the status dashboard',
      'acme/statuspage',
      'mona',
      'open',
      false,
      900,
    ],
    [
      298,
      'Image pulls fail on prod-us1 since registry migration',
      'acme/search-indexer',
      'octocat',
      'open',
      false,
      1400,
    ],
    [
      291,
      'Webhook deliveries are not retried on 5xx',
      'acme/checkout-service',
      'hubot',
      'open',
      false,
      2000,
    ],
    [
      284,
      'Flaky test: TestCheckoutTimeout',
      'acme/checkout-service',
      'octocat',
      'open',
      false,
      2600,
    ],
    [
      278,
      'Expose replication lag as a metric',
      'acme/payments-api',
      'mona',
      'open',
      false,
      3300,
    ],
  ] as const;
  return (kind === 'prs' ? prs : issues).map(
    ([number, title, repository, author, state, isDraft, minutes]) => ({
      number,
      title,
      repository,
      author,
      state,
      isDraft,
      createdAt: minutesAgo(minutes),
      url: `https://github.com/${repository}/${kind === 'prs' ? 'pull' : 'issues'}/${number}`,
    }),
  );
}

export function demoNotifications(): Notification[] {
  const items = [
    [
      true,
      'review requested',
      'feat(api): add idempotency keys to checkout',
      'PullRequest',
      'acme/checkout-service',
      'OPEN',
      '',
      false,
      '',
      25,
    ],
    [
      true,
      'mention',
      'Search results are stale after reindexing',
      'Issue',
      'acme/search-indexer',
      '',
      'OPEN',
      false,
      '',
      130,
    ],
    [
      false,
      'subscribed',
      'fix(worker): retry failed webhook deliveries',
      'PullRequest',
      'acme/checkout-service',
      'MERGED',
      '',
      false,
      '',
      300,
    ],
    [
      false,
      'subscribed',
      'v2.4.0',
      'Release',
      'acme/checkout-service',
      '',
      '',
      false,
      '',
      700,
    ],
    [
      true,
      'assign',
      'Image pulls fail on prod-us1 since registry migration',
      'Issue',
      'acme/search-indexer',
      '',
      'OPEN',
      false,
      '',
      850,
    ],
    [
      false,
      'author',
      'docs: document the reindexing runbook',
      'PullRequest',
      'acme/search-indexer',
      'OPEN',
      '',
      true,
      '',
      1100,
    ],
    [
      false,
      'comment',
      'Expose replication lag as a metric',
      'Issue',
      'acme/payments-api',
      '',
      'CLOSED',
      false,
      '',
      1300,
    ],
    [
      false,
      'ci activity',
      'Nightly build',
      'CheckSuite',
      'acme/payments-api',
      '',
      '',
      false,
      'FAILURE',
      1500,
    ],
    [
      false,
      'state change',
      'feat(ui): add dark mode toggle',
      'PullRequest',
      'acme/statuspage',
      'CLOSED',
      '',
      false,
      '',
      1800,
    ],
  ] as const;
  return items.map(
    (
      [
        isUnread,
        reason,
        title,
        type,
        repository,
        prState,
        issueState,
        isDraft,
        conclusion,
        minutes,
      ],
      i,
    ) => ({
      id: `demo-${i}`,
      isUnread,
      reason,
      title,
      type,
      repository,
      updatedAt: minutesAgo(minutes),
      url: `https://github.com/${repository}`,
      prState,
      issueState,
      isDraft,
      conclusion,
    }),
  );
}
